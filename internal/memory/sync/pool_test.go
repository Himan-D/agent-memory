package sync

import (
	"context"
	"testing"
	"time"

	"agent-memory/internal/memory/types"
)

func TestLocalPool_PublishEvent(t *testing.T) {
	cfg := &PoolConfig{EnablePubSub: true}
	pool := NewLocalPool(cfg)
	defer pool.Close()

	event := &types.MemoryPoolEvent{
		Type:      "test.event",
		GroupID:   "group-1",
		AgentID:   "agent-1",
		Timestamp: time.Now(),
	}

	ctx := context.Background()
	if err := pool.PublishEvent(ctx, event); err != nil {
		t.Fatalf("PublishEvent failed: %v", err)
	}
}

func TestLocalPool_Subscribe(t *testing.T) {
	cfg := &PoolConfig{EnablePubSub: true}
	pool := NewLocalPool(cfg)
	defer pool.Close()

	ctx := context.Background()
	sub, err := pool.Subscribe(ctx, "group-1", "agent-1")
	if err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}

	if sub.GroupID != "group-1" {
		t.Errorf("expected group-1, got %s", sub.GroupID)
	}
	if sub.AgentID != "agent-1" {
		t.Errorf("expected agent-1, got %s", sub.AgentID)
	}
	if sub.ID == "" {
		t.Error("expected non-empty subscriber ID")
	}
}

func TestLocalPool_Unsubscribe(t *testing.T) {
	cfg := &PoolConfig{EnablePubSub: true}
	pool := NewLocalPool(cfg)
	defer pool.Close()

	ctx := context.Background()
	sub, err := pool.Subscribe(ctx, "group-1", "agent-1")
	if err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}

	if err := pool.Unsubscribe(sub.ID); err != nil {
		t.Fatalf("Unsubscribe failed: %v", err)
	}

	if pool.subscribers[sub.ID] != nil {
		t.Error("subscriber should be removed")
	}
}

func TestLocalPool_GetGroupAgents(t *testing.T) {
	cfg := &PoolConfig{EnablePubSub: true}
	pool := NewLocalPool(cfg)
	defer pool.Close()

	ctx := context.Background()

	pool.Subscribe(ctx, "group-1", "agent-1")
	pool.Subscribe(ctx, "group-1", "agent-2")
	pool.Subscribe(ctx, "group-2", "agent-3")

	agents, err := pool.GetGroupAgents(ctx, "group-1")
	if err != nil {
		t.Fatalf("GetGroupAgents failed: %v", err)
	}

	if len(agents) != 2 {
		t.Errorf("expected 2 agents, got %d", len(agents))
	}
}

func TestLocalPool_ShareMemory(t *testing.T) {
	cfg := &PoolConfig{EnablePubSub: true}
	pool := NewLocalPool(cfg)
	defer pool.Close()

	ctx := context.Background()
	memory := &types.Memory{
		ID:      "mem-1",
		Content: "test content",
	}

	if err := pool.ShareMemory(ctx, memory, "group-1", "agent-1"); err != nil {
		t.Fatalf("ShareMemory failed: %v", err)
	}
}

func TestLocalPool_SyncToGroup(t *testing.T) {
	cfg := &PoolConfig{EnablePubSub: true}
	pool := NewLocalPool(cfg)
	defer pool.Close()

	ctx := context.Background()
	memory := &types.Memory{
		ID:      "mem-1",
		Content: "sync content",
	}

	if err := pool.SyncToGroup(ctx, "group-1", memory); err != nil {
		t.Fatalf("SyncToGroup failed: %v", err)
	}
}

func TestLocalPool_Ping(t *testing.T) {
	cfg := &PoolConfig{EnablePubSub: true}
	pool := NewLocalPool(cfg)
	defer pool.Close()

	ctx := context.Background()
	if err := pool.Ping(ctx); err != nil {
		t.Fatalf("Ping failed: %v", err)
	}
}

func TestLocalPool_PublishEvent_Disabled(t *testing.T) {
	cfg := &PoolConfig{EnablePubSub: false}
	pool := NewLocalPool(cfg)
	defer pool.Close()

	event := &types.MemoryPoolEvent{
		Type:      "test.event",
		GroupID:   "group-1",
		AgentID:   "agent-1",
		Timestamp: time.Now(),
	}

	ctx := context.Background()
	if err := pool.PublishEvent(ctx, event); err != nil {
		t.Fatalf("PublishEvent should not fail when disabled: %v", err)
	}
}

func TestPoolManager_GetPool(t *testing.T) {
	cfg := &PoolConfig{EnablePubSub: true}
	manager, err := NewPoolManager(cfg)
	if err != nil {
		t.Fatalf("NewPoolManager failed: %v", err)
	}
	defer manager.Close()

	pool1, err := manager.GetPool("tenant-1")
	if err != nil {
		t.Fatalf("GetPool failed: %v", err)
	}

	pool2, err := manager.GetPool("tenant-1")
	if err != nil {
		t.Fatalf("GetPool failed: %v", err)
	}

	if pool1 != pool2 {
		t.Error("same tenant should return same pool")
	}

	pool3, err := manager.GetPool("tenant-2")
	if err != nil {
		t.Fatalf("GetPool failed: %v", err)
	}

	if pool1 == pool3 {
		t.Error("different tenants should return different pools")
	}
}

func TestEventBroadcaster_PublishToGroup(t *testing.T) {
	broadcaster := NewEventBroadcaster()
	defer broadcaster.Close()

	ctx := context.Background()
	err := broadcaster.PublishToGroup(ctx, "tenant-1", "group-1", "test.event", map[string]interface{}{"key": "value"})
	if err != nil {
		t.Fatalf("PublishToGroup failed: %v", err)
	}
}

func TestEventBroadcaster_SubscribeToGroup(t *testing.T) {
	broadcaster := NewEventBroadcaster()
	defer broadcaster.Close()

	ctx := context.Background()
	sub, err := broadcaster.SubscribeToGroup(ctx, "tenant-1", "group-1", "agent-1")
	if err != nil {
		t.Fatalf("SubscribeToGroup failed: %v", err)
	}

	if sub.GroupID != "group-1" {
		t.Errorf("expected group-1, got %s", sub.GroupID)
	}
}

func TestEventBroadcaster_GetPool(t *testing.T) {
	broadcaster := NewEventBroadcaster()
	defer broadcaster.Close()

	pool1 := broadcaster.GetPool("tenant-1")
	pool2 := broadcaster.GetPool("tenant-1")

	if pool1 != pool2 {
		t.Error("same tenant should return same pool")
	}
}

func TestNewPool_WithDefaults(t *testing.T) {
	pool, err := NewPool(nil)
	if err != nil {
		t.Fatalf("NewPool failed: %v", err)
	}
	defer pool.Close()
}

func TestPoolSubscriber_Connected(t *testing.T) {
	cfg := &PoolConfig{EnablePubSub: true}
	pool := NewLocalPool(cfg)
	defer pool.Close()

	ctx := context.Background()
	sub, err := pool.Subscribe(ctx, "group-1", "agent-1")
	if err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}

	sub.mu.Lock()
	connected := sub.connected
	sub.mu.Unlock()

	if !connected {
		t.Error("subscriber should be connected")
	}
}

func TestPoolConfig_Defaults(t *testing.T) {
	pool := NewLocalPool(nil)

	if pool.config.MaxPoolSize != 1000 {
		t.Errorf("unexpected default max pool size: %d", pool.config.MaxPoolSize)
	}
	if pool.config.SyncIntervalMs != 1000 {
		t.Errorf("unexpected default sync interval: %d", pool.config.SyncIntervalMs)
	}
	if !pool.config.EnablePubSub {
		t.Error("unexpected default EnablePubSub: false")
	}
	pool.Close()
}
