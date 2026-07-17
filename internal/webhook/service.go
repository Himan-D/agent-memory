package webhook

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"

	"agent-memory/internal/config"
	"agent-memory/internal/memory/types"
)

type DeliveryLog struct {
	ID         string    `json:"id"`
	WebhookID  string    `json:"webhook_id"`
	Event      string    `json:"event"`
	Attempt    int       `json:"attempt"`
	Success    bool      `json:"success"`
	Status     string    `json:"status"` // success | failed | pending — dashboard field
	StatusCode int       `json:"status_code,omitempty"`
	// response_code aliases StatusCode for the dashboard TypeScript client.
	ResponseCode int       `json:"response_code,omitempty"`
	Error        string    `json:"error,omitempty"`
	Timestamp    time.Time `json:"timestamp"`
	CreatedAt    time.Time `json:"created_at"`
	DurationMs   int64     `json:"duration_ms,omitempty"`
}

type DeadLetterEntry struct {
	ID        string             `json:"id"`
	WebhookID string             `json:"webhook_id"`
	Event     types.WebhookEvent `json:"event"`
	Payload   interface{}        `json:"payload,omitempty"`
	Error     string             `json:"error"`
	FailedAt  time.Time          `json:"failed_at"`
	CreatedAt time.Time          `json:"created_at"`
	Attempts  int                `json:"attempts"`
}

// DeliveryHook is invoked after each delivery attempt (success or failure).
type DeliveryHook func(webhookID string, success bool, event string, statusCode int)

type Service struct {
	webhooks        map[string]*types.Webhook
	clients         map[string]*http.Client
	mu              sync.RWMutex
	cfg             *config.Config
	store           *Neo4jStore
	deliveryLogs    []DeliveryLog
	deliveryMu      sync.Mutex
	deadLetterQueue []DeadLetterEntry
	persistPath     string // directory for delivery/DLQ JSON persistence
	onDelivery      DeliveryHook
}

func (s *Service) SetDeliveryHook(hook DeliveryHook) {
	s.onDelivery = hook
}

type persistedWebhookState struct {
	DeliveryLogs    []DeliveryLog     `json:"delivery_logs"`
	DeadLetterQueue []DeadLetterEntry  `json:"dead_letter_queue"`
}

func NewService(cfg *config.Config) *Service {
	return &Service{
		webhooks: make(map[string]*types.Webhook),
		clients:  make(map[string]*http.Client),
		cfg:      cfg,
	}
}

func (s *Service) SetStore(store *Neo4jStore) {
	s.store = store
}

// EnableFilePersistence stores delivery logs and DLQ under dataDir so they
// survive process restarts (in-memory alone loses them).
func (s *Service) EnableFilePersistence(dataDir string) {
	if dataDir == "" {
		dataDir = "data"
	}
	s.persistPath = filepath.Join(dataDir, "webhook_state.json")
	if err := os.MkdirAll(filepath.Dir(s.persistPath), 0o755); err != nil {
		log.Printf("webhook persistence: mkdir: %v", err)
		return
	}
	s.loadPersistedState()
}

func (s *Service) loadPersistedState() {
	if s.persistPath == "" {
		return
	}
	data, err := os.ReadFile(s.persistPath)
	if err != nil {
		return
	}
	var state persistedWebhookState
	if err := json.Unmarshal(data, &state); err != nil {
		log.Printf("webhook persistence: load: %v", err)
		return
	}
	s.deliveryMu.Lock()
	s.deliveryLogs = state.DeliveryLogs
	s.deadLetterQueue = state.DeadLetterQueue
	s.deliveryMu.Unlock()
	log.Printf("webhook persistence: loaded %d delivery logs, %d DLQ entries",
		len(state.DeliveryLogs), len(state.DeadLetterQueue))
}

func (s *Service) persistState() {
	if s.persistPath == "" {
		return
	}
	s.deliveryMu.Lock()
	state := persistedWebhookState{
		DeliveryLogs:    append([]DeliveryLog{}, s.deliveryLogs...),
		DeadLetterQueue: append([]DeadLetterEntry{}, s.deadLetterQueue...),
	}
	s.deliveryMu.Unlock()
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return
	}
	tmp := s.persistPath + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		log.Printf("webhook persistence: write: %v", err)
		return
	}
	_ = os.Rename(tmp, s.persistPath)
}

func (s *Service) LoadFromStore(ctx context.Context) error {
	if s.store == nil {
		return nil
	}
	webhooks, err := s.store.List(ctx, "")
	if err != nil {
		return fmt.Errorf("load webhooks from neo4j: %w", err)
	}
	s.mu.Lock()
	for _, wh := range webhooks {
		s.webhooks[wh.ID] = wh
		s.clients[wh.ID] = &http.Client{Timeout: 10 * time.Second}
	}
	s.mu.Unlock()
	return nil
}

func (s *Service) CreateWebhook(ctx context.Context, wh *types.Webhook) (*types.Webhook, error) {
	if err := validateWebhookURL(wh.URL); err != nil {
		return nil, err
	}
	if len(wh.Events) == 0 {
		return nil, fmt.Errorf("webhook events are required")
	}
	if wh.ID == "" {
		wh.ID = uuid.New().String()
	}
	wh.CreatedAt = time.Now()

	if wh.Secret == "" {
		secret, err := generateWebhookSecret()
		if err != nil {
			return nil, err
		}
		wh.Secret = secret
	}

	s.mu.Lock()
	s.webhooks[wh.ID] = wh
	s.mu.Unlock()

	s.clients[wh.ID] = &http.Client{
		Timeout: 10 * time.Second,
	}

	if s.store != nil {
		if err := s.store.Store(ctx, wh); err != nil {
			log.Printf("webhook persist error: %v", err)
		}
	}

	return wh, nil
}

func (s *Service) GetWebhook(id string) (*types.Webhook, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if wh, ok := s.webhooks[id]; ok {
		return wh, nil
	}
	return nil, fmt.Errorf("webhook not found: %s", id)
}

func (s *Service) UpdateWebhook(ctx context.Context, id string, updates *types.Webhook) (*types.Webhook, error) {
	// Full replace-style update (legacy PUT body). Prefer PatchWebhook for partials.
	return s.PatchWebhook(ctx, id, map[string]interface{}{
		"url":        updates.URL,
		"tenant_id":  updates.TenantID,
		"project_id": updates.ProjectID,
		"events":     updates.Events,
		"fields":     updates.Fields,
		"active":     updates.Active,
		"metadata":   updates.Metadata,
		"_force_active": true, // always apply Active from typed update
	})
}

// PatchWebhook applies a partial update from a JSON object (PATCH-friendly).
// Only keys present in the map are applied. Use "active" bool to toggle.
func (s *Service) PatchWebhook(ctx context.Context, id string, patch map[string]interface{}) (*types.Webhook, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	wh, ok := s.webhooks[id]
	if !ok {
		return nil, fmt.Errorf("webhook not found: %s", id)
	}

	if v, ok := patch["url"].(string); ok && v != "" {
		if err := validateWebhookURL(v); err != nil {
			return nil, err
		}
		wh.URL = v
		wh.VerifiedAt = nil
	}
	if v, ok := patch["tenant_id"].(string); ok && v != "" {
		wh.TenantID = v
	}
	if v, ok := patch["project_id"].(string); ok && v != "" {
		wh.ProjectID = v
	}
	if v, ok := patch["events"]; ok && v != nil {
		wh.Events = toWebhookEvents(v)
	}
	if v, ok := patch["fields"]; ok && v != nil {
		wh.Fields = toStringSlice(v)
	}
	if v, ok := patch["active"].(bool); ok {
		wh.Active = v
	} else if _, force := patch["_force_active"]; force {
		// typed UpdateWebhook always sends Active (bool zero = false)
		if v, ok := patch["active"].(bool); ok {
			wh.Active = v
		}
	}
	if v, ok := patch["metadata"].(map[string]interface{}); ok && v != nil {
		wh.Metadata = v
	}

	s.webhooks[id] = wh

	if s.store != nil {
		if err := s.store.Update(ctx, wh); err != nil {
			log.Printf("webhook persist update error: %v", err)
		}
	}

	return wh, nil
}

func toWebhookEvents(v interface{}) []types.WebhookEvent {
	switch t := v.(type) {
	case []types.WebhookEvent:
		return t
	case []string:
		out := make([]types.WebhookEvent, 0, len(t))
		for _, e := range t {
			out = append(out, types.WebhookEvent(e))
		}
		return out
	case []interface{}:
		out := make([]types.WebhookEvent, 0, len(t))
		for _, e := range t {
			if s, ok := e.(string); ok {
				out = append(out, types.WebhookEvent(s))
			}
		}
		return out
	default:
		return nil
	}
}

func toStringSlice(v interface{}) []string {
	switch t := v.(type) {
	case []string:
		return t
	case []interface{}:
		out := make([]string, 0, len(t))
		for _, e := range t {
			if s, ok := e.(string); ok {
				out = append(out, s)
			}
		}
		return out
	default:
		return nil
	}
}

func (s *Service) DeleteWebhook(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.webhooks[id]; !ok {
		return fmt.Errorf("webhook not found: %s", id)
	}

	delete(s.webhooks, id)
	delete(s.clients, id)

	if s.store != nil {
		if err := s.store.Delete(context.Background(), id); err != nil {
			log.Printf("webhook persist delete error: %v", err)
		}
	}

	return nil
}

func (s *Service) ListWebhooks(projectID string) []*types.Webhook {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*types.Webhook
	for _, wh := range s.webhooks {
		if projectID != "" && wh.ProjectID != projectID {
			continue
		}
		result = append(result, wh)
	}
	return result
}

func (s *Service) ListActiveWebhooks(projectID string) []*types.Webhook {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*types.Webhook
	for _, wh := range s.webhooks {
		if !wh.Active {
			continue
		}
		if projectID != "" && wh.ProjectID != projectID {
			continue
		}
		result = append(result, wh)
	}
	return result
}

func (s *Service) EmitEvent(ctx context.Context, event types.WebhookEvent, projectID string, data interface{}) {
	payload := types.WebhookPayload{
		Event:     event,
		Timestamp: time.Now(),
		Data:      data,
	}

	webhooks := s.ListActiveWebhooks(projectID)
	for _, wh := range webhooks {
		for _, e := range wh.Events {
			if e == event || e == "*" {
				go s.deliverWebhook(wh, payload)
				break
			}
		}
	}
}

func (s *Service) deliverWebhook(wh *types.Webhook, payload types.WebhookPayload) {
	payload = filterPayloadFields(wh, payload)
	body, err := json.Marshal(payload)
	if err != nil {
		log.Printf("webhook marshal error: %v", err)
		return
	}

	client, ok := s.clients[wh.ID]
	if !ok {
		client = &http.Client{Timeout: 10 * time.Second}
	}

	delays := []time.Duration{0, time.Second, 2 * time.Second, 4 * time.Second}
	for attempt, delay := range delays {
		if delay > 0 {
			time.Sleep(delay)
		}
		start := time.Now()
		statusCode, deliverErr := s.attemptDeliveryWithStatus(client, wh, body, payload)
		now := time.Now()
		durMs := now.Sub(start).Milliseconds()
		if deliverErr != nil {
			log.Printf("webhook delivery attempt %d failed for %s: %v", attempt+1, wh.ID, deliverErr)
			s.recordDelivery(newDeliveryLog(wh.ID, string(payload.Event), attempt+1, false, statusCode, deliverErr.Error(), now, durMs))
			s.updateWebhookStats(wh.ID, false, statusCode, now)
			if s.onDelivery != nil && attempt == len(delays)-1 {
				s.onDelivery(wh.ID, false, string(payload.Event), statusCode)
			}
			if attempt == len(delays)-1 {
				log.Printf("webhook delivery exhausted retries for %s", wh.ID)
				entry := DeadLetterEntry{
					ID:        uuid.New().String(),
					WebhookID: wh.ID,
					Event:     payload.Event,
					Payload:   payload,
					Error:     deliverErr.Error(),
					FailedAt:  now,
					CreatedAt: now,
					Attempts:  attempt + 1,
				}
				s.deliveryMu.Lock()
				s.deadLetterQueue = append(s.deadLetterQueue, entry)
				s.deliveryMu.Unlock()
				go s.persistState()
				if s.store != nil {
					go func(e DeadLetterEntry) {
						if err := s.store.StoreDeadLetter(context.Background(), e); err != nil {
							log.Printf("webhook neo4j DLQ store: %v", err)
						}
					}(entry)
				}
			}
			continue
		}
		s.recordDelivery(newDeliveryLog(wh.ID, string(payload.Event), attempt+1, true, statusCode, "", now, durMs))
		s.updateWebhookStats(wh.ID, true, statusCode, now)
		if s.onDelivery != nil {
			s.onDelivery(wh.ID, true, string(payload.Event), statusCode)
		}
		return
	}
}

func newDeliveryLog(webhookID, event string, attempt int, success bool, statusCode int, errMsg string, at time.Time, durMs int64) DeliveryLog {
	status := "failed"
	if success {
		status = "success"
	}
	return DeliveryLog{
		ID:           uuid.New().String(),
		WebhookID:    webhookID,
		Event:        event,
		Attempt:      attempt,
		Success:      success,
		Status:       status,
		StatusCode:   statusCode,
		ResponseCode: statusCode,
		Error:        errMsg,
		Timestamp:    at,
		CreatedAt:    at,
		DurationMs:   durMs,
	}
}

func (s *Service) updateWebhookStats(webhookID string, success bool, statusCode int, at time.Time) {
	s.mu.Lock()
	wh, ok := s.webhooks[webhookID]
	if !ok {
		s.mu.Unlock()
		return
	}
	if success {
		wh.SuccessCount++
	} else {
		wh.FailureCount++
	}
	wh.LastDeliveryAt = &at
	wh.LastTriggered = &at
	wh.LastStatusCode = statusCode
	// snapshot for async neo4j write
	snap := *wh
	s.mu.Unlock()

	if s.store != nil {
		go func() {
			if err := s.store.Update(context.Background(), &snap); err != nil {
				log.Printf("webhook stats persist: %v", err)
			}
		}()
	}
}

func (s *Service) recordDelivery(entry DeliveryLog) {
	s.deliveryMu.Lock()
	if len(s.deliveryLogs) > 1000 {
		s.deliveryLogs = s.deliveryLogs[500:]
	}
	s.deliveryLogs = append(s.deliveryLogs, entry)
	s.deliveryMu.Unlock()
	go s.persistState()
	if s.store != nil {
		go func(e DeliveryLog) {
			if err := s.store.StoreDelivery(context.Background(), e); err != nil {
				log.Printf("webhook neo4j delivery store: %v", err)
			}
		}(entry)
	}
}

func (s *Service) GetDeliveryLogs(limit int) []DeliveryLog {
	s.deliveryMu.Lock()
	defer s.deliveryMu.Unlock()
	if limit <= 0 || limit > len(s.deliveryLogs) {
		limit = len(s.deliveryLogs)
	}
	result := make([]DeliveryLog, limit)
	copy(result, s.deliveryLogs[len(s.deliveryLogs)-limit:])
	return result
}

func (s *Service) GetDeadLetterQueue() []DeadLetterEntry {
	s.deliveryMu.Lock()
	defer s.deliveryMu.Unlock()
	out := make([]DeadLetterEntry, 0, len(s.deadLetterQueue))
	for _, e := range s.deadLetterQueue {
		if e.ID == "" {
			e.ID = uuid.New().String()
		}
		if e.CreatedAt.IsZero() {
			e.CreatedAt = e.FailedAt
		}
		out = append(out, e)
	}
	return out
}

func (s *Service) GetDeliveryLogsByWebhook(webhookID string) []DeliveryLog {
	s.deliveryMu.Lock()
	defer s.deliveryMu.Unlock()
	var logs []DeliveryLog
	for _, l := range s.deliveryLogs {
		if l.WebhookID == webhookID {
			logs = append(logs, normalizeDeliveryLog(l))
		}
	}
	return logs
}

func normalizeDeliveryLog(l DeliveryLog) DeliveryLog {
	if l.ID == "" {
		l.ID = uuid.New().String()
	}
	if l.Status == "" {
		if l.Success {
			l.Status = "success"
		} else {
			l.Status = "failed"
		}
	}
	if l.ResponseCode == 0 {
		l.ResponseCode = l.StatusCode
	}
	if l.CreatedAt.IsZero() {
		l.CreatedAt = l.Timestamp
	}
	return l
}

func (s *Service) RetryDeadLetter(webhookID string, event string) error {
	s.deliveryMu.Lock()
	var found *DeadLetterEntry
	var foundIdx int
	for i, entry := range s.deadLetterQueue {
		if entry.WebhookID == webhookID && entry.Event == types.WebhookEvent(event) {
			found = &s.deadLetterQueue[i]
			foundIdx = i
			break
		}
	}
	if found == nil {
		s.deliveryMu.Unlock()
		return fmt.Errorf("dead letter entry not found")
	}
	replayPayload := found.Payload
	replayEvent := found.Event
	s.deadLetterQueue = append(s.deadLetterQueue[:foundIdx], s.deadLetterQueue[foundIdx+1:]...)
	s.deliveryMu.Unlock()
	go s.persistState()

	go s.EmitEvent(context.Background(), replayEvent, "", replayPayload)
	return nil
}

func (s *Service) attemptDeliveryWithStatus(client *http.Client, wh *types.Webhook, body []byte, payload types.WebhookPayload) (int, error) {
	req, err := http.NewRequest(http.MethodPost, wh.URL, bytes.NewBuffer(body))
	if err != nil {
		return 0, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-AgentMemory-Event", string(payload.Event))
	req.Header.Set("X-AgentMemory-Timestamp", payload.Timestamp.Format(time.RFC3339))

	if wh.Secret != "" {
		signature := s.computeSignature(body, wh.Secret)
		req.Header.Set("X-AgentMemory-Signature", signature)
	}

	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return resp.StatusCode, fmt.Errorf("status %d", resp.StatusCode)
	}
	return resp.StatusCode, nil
}

func (s *Service) computeSignature(body []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

func (s *Service) TestWebhook(ctx context.Context, id string) (int, types.WebhookEvent, error) {
	wh, err := s.GetWebhook(id)
	if err != nil {
		return 0, "", err
	}

	event := types.WebhookEventMemoryCreated
	if len(wh.Events) > 0 && wh.Events[0] != "" && wh.Events[0] != "*" {
		event = wh.Events[0]
	}

	testPayload := types.WebhookPayload{
		Event:     event,
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"test":       true,
			"webhook_id": wh.ID,
			"event":      event,
			"id":         wh.ID,
			"name":       "Webhook test",
		},
	}

	client := &http.Client{Timeout: 10 * time.Second}
	body, err := json.Marshal(filterPayloadFields(wh, testPayload))
	if err != nil {
		return 0, event, fmt.Errorf("marshal test payload: %w", err)
	}
	statusCode, err := s.attemptDeliveryWithStatus(client, wh, body, testPayload)
	if err != nil {
		return statusCode, event, fmt.Errorf("deliver test webhook: %w", err)
	}

	now := time.Now()
	s.mu.Lock()
	if current, ok := s.webhooks[wh.ID]; ok {
		current.VerifiedAt = &now
		current.LastDeliveryAt = &now
		current.LastStatusCode = statusCode
		if s.store != nil {
			wh = current
		}
	}
	s.mu.Unlock()
	if s.store != nil {
		if err := s.store.Update(ctx, wh); err != nil {
			log.Printf("webhook persist verification error: %v", err)
		}
	}

	return statusCode, event, nil
}

func generateWebhookSecret() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("webhook secret: %w", err)
	}
	return hex.EncodeToString(b), nil
}

func validateWebhookURL(rawURL string) error {
	if rawURL == "" {
		return fmt.Errorf("webhook url is required")
	}
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid webhook url: %w", err)
	}
	if parsed.Scheme != "https" {
		return fmt.Errorf("webhook url must use https")
	}
	if parsed.User != nil {
		return fmt.Errorf("webhook url must not include credentials")
	}
	host := parsed.Hostname()
	if host == "" {
		return fmt.Errorf("webhook url host is required")
	}
	if ip := net.ParseIP(host); ip != nil {
		if ip.IsLoopback() || ip.IsPrivate() || ip.IsUnspecified() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
			return fmt.Errorf("webhook url must not target a private or local address")
		}
	}
	if host == "localhost" || host == "127.0.0.1" || host == "::1" {
		return fmt.Errorf("webhook url must not target localhost")
	}
	return nil
}

func filterPayloadFields(wh *types.Webhook, payload types.WebhookPayload) types.WebhookPayload {
	if len(wh.Fields) == 0 || payload.Data == nil {
		return payload
	}

	raw, err := json.Marshal(payload.Data)
	if err != nil {
		return payload
	}
	var data map[string]interface{}
	if err := json.Unmarshal(raw, &data); err != nil {
		return payload
	}

	allowed := make(map[string]bool, len(wh.Fields))
	for _, field := range wh.Fields {
		allowed[field] = true
	}

	filtered := make(map[string]interface{}, len(wh.Fields))
	for field := range allowed {
		if value, ok := data[field]; ok {
			filtered[field] = value
		}
	}
	payload.Data = filtered
	return payload
}
