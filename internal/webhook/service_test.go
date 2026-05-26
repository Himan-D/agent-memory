package webhook

import (
	"context"
	"testing"
	"time"

	"agent-memory/internal/config"
	"agent-memory/internal/memory/types"
)

func TestNewService(t *testing.T) {
	svc := NewServiceForTest(&config.Config{})
	if svc == nil {
		t.Fatal("NewService returned nil")
	}
	if len(svc.webhooks) != 0 {
		t.Error("expected empty webhooks map")
	}
}

func TestCreateAndGetWebhook(t *testing.T) {
	svc := NewServiceForTest(&config.Config{})
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
	svc := NewServiceForTest(&config.Config{})
	_, err := svc.GetWebhook("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent webhook")
	}
}

func TestDeleteWebhook(t *testing.T) {
	svc := NewServiceForTest(&config.Config{})
	wh, _ := svc.CreateWebhook(context.Background(), &types.Webhook{URL: "https://example.com/hook"})
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
	svc := NewServiceForTest(&config.Config{})
	svc.CreateWebhook(context.Background(), &types.Webhook{URL: "https://a.com", ProjectID: "p1"})
	svc.CreateWebhook(context.Background(), &types.Webhook{URL: "https://b.com", ProjectID: "p1"})
	svc.CreateWebhook(context.Background(), &types.Webhook{URL: "https://c.com", ProjectID: "p2"})

	all := svc.ListWebhooks("p1")
	if len(all) != 2 {
		t.Errorf("expected 2 webhooks for p1, got %d", len(all))
	}
}

func TestDeliveryLog(t *testing.T) {
	svc := NewServiceForTest(&config.Config{})
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
	svc := NewServiceForTest(&config.Config{})
	svc.deadLetter = append(svc.deadLetter, DeliveryLog{
		WebhookID: "wh1", Event: "memory.created",
		Attempt: 4, Success: false, Error: "timeout", Timestamp: time.Now(),
	})
	dlq := svc.GetDeadLetterQueue()
	if len(dlq) != 1 {
		t.Fatalf("expected 1 DLQ entry, got %d", len(dlq))
	}
	if dlq[0].Error != "timeout" {
		t.Errorf("expected timeout, got %s", dlq[0].Error)
	}
}
