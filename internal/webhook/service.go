package webhook

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"

	"agent-memory/internal/config"
	"agent-memory/internal/memory/types"
)

type Service struct {
	webhooks map[string]*types.Webhook
	clients  map[string]*http.Client
	mu       sync.RWMutex
	cfg      *config.Config
}

func NewService(cfg *config.Config) *Service {
	return &Service{
		webhooks: make(map[string]*types.Webhook),
		clients:  make(map[string]*http.Client),
		cfg:      cfg,
	}
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
		fmt.Printf("webhook marshal error: %v\n", err)
		return
	}

	req, err := http.NewRequest(http.MethodPost, wh.URL, bytes.NewBuffer(body))
	if err != nil {
		fmt.Printf("webhook request error: %v\n", err)
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-AgentMemory-Event", string(payload.Event))
	req.Header.Set("X-AgentMemory-Timestamp", payload.Timestamp.Format(time.RFC3339))

	if wh.Secret != "" {
		signature := s.computeSignature(body, wh.Secret)
		req.Header.Set("X-AgentMemory-Signature", signature)
	}

	client, ok := s.clients[wh.ID]
	if !ok {
		client = &http.Client{Timeout: 10 * time.Second}
	}

	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("webhook delivery error for %s: %v\n", wh.ID, err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		fmt.Printf("webhook delivery failed for %s: status %d\n", wh.ID, resp.StatusCode)
	}
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
