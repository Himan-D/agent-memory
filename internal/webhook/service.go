package webhook

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"

	"agent-memory/internal/config"
	"agent-memory/internal/memory/types"
)

type DeliveryLog struct {
	WebhookID  string    `json:"webhook_id"`
	Event      string    `json:"event"`
	Attempt    int       `json:"attempt"`
	Success    bool      `json:"success"`
	StatusCode int       `json:"status_code,omitempty"`
	Error      string    `json:"error,omitempty"`
	Timestamp  time.Time `json:"timestamp"`
}

type DeadLetterEntry struct {
	WebhookID string               `json:"webhook_id"`
	Event     types.WebhookEvent   `json:"event"`
	Payload   interface{}          `json:"payload"`
	Error     string               `json:"error"`
	FailedAt  time.Time            `json:"failed_at"`
	Attempts  int                  `json:"attempts"`
}

type Service struct {
	webhooks        map[string]*types.Webhook
	clients         map[string]*http.Client
	mu              sync.RWMutex
	cfg             *config.Config
	store           *Neo4jStore
	deliveryLogs    []DeliveryLog
	deliveryMu      sync.Mutex
	deadLetterQueue []DeadLetterEntry
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
	if wh.ID == "" {
		wh.ID = uuid.New().String()
	}
	wh.CreatedAt = time.Now()

	if wh.Secret == "" {
		b := make([]byte, 32)
		for i := range b {
			b[i] = byte(uuid.New().ID())
		}
		wh.Secret = hex.EncodeToString(b)
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
	s.mu.Lock()
	defer s.mu.Unlock()

	wh, ok := s.webhooks[id]
	if !ok {
		return nil, fmt.Errorf("webhook not found: %s", id)
	}

	if updates.URL != "" {
		wh.URL = updates.URL
	}
	if updates.Events != nil {
		wh.Events = updates.Events
	}
	if updates.Active != wh.Active {
		wh.Active = updates.Active
	}
	if updates.Metadata != nil {
		wh.Metadata = updates.Metadata
	}

	s.webhooks[id] = wh

	if s.store != nil {
		if err := s.store.Update(ctx, wh); err != nil {
			log.Printf("webhook persist update error: %v", err)
		}
	}

	return wh, nil
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
		statusCode, deliverErr := s.attemptDeliveryWithStatus(client, wh, body, payload)
		now := time.Now()
		if deliverErr != nil {
			log.Printf("webhook delivery attempt %d failed for %s: %v", attempt+1, wh.ID, deliverErr)
			s.recordDelivery(DeliveryLog{
				WebhookID: wh.ID, Event: string(payload.Event),
				Attempt: attempt + 1, Success: false, StatusCode: statusCode,
				Error: deliverErr.Error(), Timestamp: now,
			})
			s.updateWebhookStats(wh.ID, false, statusCode, now)
			if attempt == len(delays)-1 {
				log.Printf("webhook delivery exhausted retries for %s", wh.ID)
				s.deliveryMu.Lock()
				s.deadLetterQueue = append(s.deadLetterQueue, DeadLetterEntry{
					WebhookID: wh.ID,
					Event:     payload.Event,
					Payload:   payload,
					Error:     deliverErr.Error(),
					FailedAt:  now,
					Attempts:  attempt + 1,
				})
				s.deliveryMu.Unlock()
			}
			continue
		}
		s.recordDelivery(DeliveryLog{
			WebhookID: wh.ID, Event: string(payload.Event),
			Attempt: attempt + 1, Success: true, StatusCode: statusCode, Timestamp: now,
		})
		s.updateWebhookStats(wh.ID, true, statusCode, now)
		return
	}
}

func (s *Service) updateWebhookStats(webhookID string, success bool, statusCode int, at time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	wh, ok := s.webhooks[webhookID]
	if !ok {
		return
	}
	if success {
		wh.SuccessCount++
	} else {
		wh.FailureCount++
	}
	wh.LastDeliveryAt = &at
	wh.LastStatusCode = statusCode
}

func (s *Service) recordDelivery(log DeliveryLog) {
	s.deliveryMu.Lock()
	if len(s.deliveryLogs) > 1000 {
		s.deliveryLogs = s.deliveryLogs[500:]
	}
	s.deliveryLogs = append(s.deliveryLogs, log)
	s.deliveryMu.Unlock()
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
	return append([]DeadLetterEntry{}, s.deadLetterQueue...)
}

func (s *Service) GetDeliveryLogsByWebhook(webhookID string) []DeliveryLog {
	s.deliveryMu.Lock()
	defer s.deliveryMu.Unlock()
	var logs []DeliveryLog
	for _, l := range s.deliveryLogs {
		if l.WebhookID == webhookID {
			logs = append(logs, l)
		}
	}
	return logs
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

func (s *Service) TestWebhook(ctx context.Context, id string) error {
	wh, err := s.GetWebhook(id)
	if err != nil {
		return err
	}

	testPayload := types.WebhookPayload{
		Event:     types.WebhookEventMemoryCreated,
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"test":       true,
			"webhook_id": wh.ID,
		},
	}

	body, err := json.Marshal(testPayload)
	if err != nil {
		return fmt.Errorf("marshal test payload: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, wh.URL, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-AgentMemory-Event", string(testPayload.Event))
	req.Header.Set("X-AgentMemory-Timestamp", testPayload.Timestamp.Format(time.RFC3339))

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("deliver test webhook: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}

	return nil
}
