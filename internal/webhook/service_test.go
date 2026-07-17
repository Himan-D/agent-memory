package webhook

import (
	"context"
	"testing"
	"time"

	"agent-memory/internal/config"
	"agent-memory/internal/memory/types"
)

func TestNewService(t *testing.T) {
	svc := NewService(&config.Config{})
	if svc == nil {
		t.Fatal("NewService returned nil")
	}
	if len(svc.webhooks) != 0 {
		t.Error("expected empty webhooks map")
	}
}

func TestCreateAndGetWebhook(t *testing.T) {
	svc := NewService(&config.Config{})
	wh, err := svc.CreateWebhook(context.Background(), &types.Webhook{URL: "https://example.com/hook", Events: []types.WebhookEvent{types.WebhookEventMemoryCreated}})
	if err != nil {
		t.Fatalf("CreateWebhook error: %v", err)
	}
	if wh.ID == "" {
		t.Error("expected non-empty ID")
	}
	if wh.Secret == "" {
		t.Error("expected auto-generated secret")
	}

	got, err := svc.GetWebhook(wh.ID)
	if err != nil {
		t.Fatalf("GetWebhook error: %v", err)
	}
	if got.URL != "https://example.com/hook" {
		t.Errorf("expected URL https://example.com/hook, got %s", got.URL)
	}
}

func TestGetWebhookNotFound(t *testing.T) {
	svc := NewService(&config.Config{})
	_, err := svc.GetWebhook("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent webhook")
	}
}

func TestDeleteWebhook(t *testing.T) {
	svc := NewService(&config.Config{})
	wh, _ := svc.CreateWebhook(context.Background(), &types.Webhook{URL: "https://example.com/hook", Events: []types.WebhookEvent{types.WebhookEventMemoryCreated}})
	err := svc.DeleteWebhook(wh.ID)
	if err != nil {
		t.Fatalf("DeleteWebhook error: %v", err)
	}
	_, err = svc.GetWebhook(wh.ID)
	if err == nil {
		t.Error("expected error after delete")
	}
}

func TestListWebhooks(t *testing.T) {
	svc := NewService(&config.Config{})
	svc.CreateWebhook(context.Background(), &types.Webhook{URL: "https://a.com", ProjectID: "p1", Events: []types.WebhookEvent{types.WebhookEventMemoryCreated}})
	svc.CreateWebhook(context.Background(), &types.Webhook{URL: "https://b.com", ProjectID: "p1", Events: []types.WebhookEvent{types.WebhookEventMemoryCreated}})
	svc.CreateWebhook(context.Background(), &types.Webhook{URL: "https://c.com", ProjectID: "p2", Events: []types.WebhookEvent{types.WebhookEventMemoryCreated}})

	all := svc.ListWebhooks("p1")
	if len(all) != 2 {
		t.Errorf("expected 2 webhooks for p1, got %d", len(all))
	}
}

func TestDeliveryLog(t *testing.T) {
	svc := NewService(&config.Config{})
	svc.recordDelivery(DeliveryLog{
		WebhookID: "wh1", Event: "memory.created",
		Attempt: 1, Success: true, Timestamp: time.Now(),
	})
	logs := svc.GetDeliveryLogs(10)
	if len(logs) != 1 {
		t.Fatalf("expected 1 log, got %d", len(logs))
	}
	if logs[0].WebhookID != "wh1" {
		t.Errorf("expected wh1, got %s", logs[0].WebhookID)
	}
}

func TestDeadLetterQueue(t *testing.T) {
	svc := NewService(&config.Config{})
	svc.deadLetterQueue = append(svc.deadLetterQueue, DeadLetterEntry{
		WebhookID: "wh1",
		Event:     types.WebhookEventMemoryCreated,
		Error:     "timeout",
		FailedAt:  time.Now(),
		Attempts:  4,
	})
	dlq := svc.GetDeadLetterQueue()
	if len(dlq) != 1 {
		t.Fatalf("expected 1 DLQ entry, got %d", len(dlq))
	}
	if dlq[0].Error != "timeout" {
		t.Errorf("expected timeout, got %s", dlq[0].Error)
	}
	if dlq[0].ID == "" {
		t.Error("expected normalized DLQ id")
	}
	if dlq[0].CreatedAt.IsZero() {
		t.Error("expected created_at filled from failed_at")
	}
}

func TestPatchWebhookActive(t *testing.T) {
	svc := NewService(&config.Config{})
	wh, err := svc.CreateWebhook(context.Background(), &types.Webhook{
		URL:    "https://example.com/hook",
		Events: []types.WebhookEvent{types.WebhookEventMemoryCreated},
		Active: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	updated, err := svc.PatchWebhook(context.Background(), wh.ID, map[string]interface{}{"active": false})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Active {
		t.Error("expected active=false after patch")
	}
	// URL-only patch must not re-enable
	updated, err = svc.PatchWebhook(context.Background(), wh.ID, map[string]interface{}{
		"url": "https://example.com/hook2",
	})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Active {
		t.Error("url-only patch should not change active")
	}
	if updated.URL != "https://example.com/hook2" {
		t.Errorf("url not updated: %s", updated.URL)
	}
}
