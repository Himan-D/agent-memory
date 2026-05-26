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
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/google/uuid"

	"agent-memory/internal/config"
	"agent-memory/internal/memory/types"
)

// privateIPNets are the IP ranges blocked for SSRF protection.
var privateIPNets []*net.IPNet

func init() {
	cidrs := []string{
		"127.0.0.0/8",
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
		"169.254.0.0/16",
		"::1/128",
	}
	for _, cidr := range cidrs {
		_, ipNet, err := net.ParseCIDR(cidr)
		if err == nil {
			privateIPNets = append(privateIPNets, ipNet)
		}
	}
}

// validateWebhookURL returns an error if the URL resolves to a private or
// loopback address (SSRF protection).
func validateWebhookURL(rawURL string) error {
	u, err := url.ParseRequestURI(rawURL)
	if err != nil {
		return fmt.Errorf("invalid webhook URL: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("webhook URL scheme must be http or https")
	}

	host := u.Hostname()
	addrs, err := net.LookupHost(host)
	if err != nil {
		return fmt.Errorf("webhook URL DNS lookup failed: %w", err)
	}

	for _, addr := range addrs {
		ip := net.ParseIP(addr)
		if ip == nil {
			continue
		}
		for _, blocked := range privateIPNets {
			if blocked.Contains(ip) {
				return fmt.Errorf("webhook URL resolves to a blocked private address: %s", addr)
			}
		}
	}
	return nil
}

type Service struct {
	webhooks map[string]*types.Webhook
	clients  map[string]*http.Client
	mu       sync.RWMutex
	cfg      *config.Config
	store    *Neo4jStore
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
	if err := validateWebhookURL(wh.URL); err != nil {
		return nil, err
	}

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
		if err := s.attemptDelivery(client, wh, body, payload); err != nil {
			log.Printf("webhook delivery attempt %d failed for %s: %v", attempt+1, wh.ID, err)
			if attempt == len(delays)-1 {
				log.Printf("webhook delivery exhausted retries for %s", wh.ID)
			}
			continue
		}
		return
	}
}

func (s *Service) attemptDelivery(client *http.Client, wh *types.Webhook, body []byte, payload types.WebhookPayload) error {
	if err := validateWebhookURL(wh.URL); err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, wh.URL, bytes.NewBuffer(body))
	if err != nil {
		return err
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
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("status %d", resp.StatusCode)
	}
	return nil
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

	if err := validateWebhookURL(wh.URL); err != nil {
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
