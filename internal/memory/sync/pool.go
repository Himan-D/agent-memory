package sync

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/google/uuid"

	"agent-memory/internal/memory/types"
)

type PoolConfig struct {
	RedisURL       string
	PoolKeyPrefix  string
	EventKeyPrefix string
	MaxPoolSize    int
	SyncIntervalMs int
	EnablePubSub   bool
}

type PoolSubscriber struct {
	ID        string
	GroupID   string
	AgentID   string
	Channel   chan *types.MemoryPoolEvent
	connected bool
	mu        sync.Mutex
}

type PoolEvent struct {
	Type      string                 `json:"type"`
	GroupID   string                 `json:"group_id"`
	AgentID   string                 `json:"agent_id"`
	MemoryID  string                 `json:"memory_id,omitempty"`
	SkillID   string                 `json:"skill_id,omitempty"`
	Data      map[string]interface{} `json:"data,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
}

type MemoryPool interface {
	PublishEvent(ctx context.Context, event *types.MemoryPoolEvent) error
	Subscribe(ctx context.Context, groupID, agentID string) (*PoolSubscriber, error)
	Unsubscribe(subscriberID string) error
	GetSharedMemories(ctx context.Context, groupID string, limit int) ([]*types.Memory, error)
	ShareMemory(ctx context.Context, memory *types.Memory, groupID, sharedBy string) error
	GetGroupAgents(ctx context.Context, groupID string) ([]string, error)
	SyncToGroup(ctx context.Context, groupID string, memory *types.Memory) error
	Ping(ctx context.Context) error
	Close() error
}

type LocalPool struct {
	mu          sync.RWMutex
	subscribers map[string]*PoolSubscriber
	eventQueue  chan *types.MemoryPoolEvent
	config      *PoolConfig
}

func NewPool(cfg *PoolConfig) (MemoryPool, error) {
	if cfg == nil {
		cfg = &PoolConfig{
			PoolKeyPrefix:  "agentmemory:pool:",
			EventKeyPrefix: "agentmemory:events:",
			MaxPoolSize:    1000,
			SyncIntervalMs: 1000,
			EnablePubSub:   true,
		}
	}

	return NewLocalPool(cfg), nil
}

func NewLocalPool(cfg *PoolConfig) *LocalPool {
	if cfg == nil {
		cfg = &PoolConfig{
			MaxPoolSize:    1000,
			SyncIntervalMs: 1000,
			EnablePubSub:   true,
		}
	}

	pool := &LocalPool{
		subscribers: make(map[string]*PoolSubscriber),
		eventQueue:  make(chan *types.MemoryPoolEvent, 1000),
		config:      cfg,
	}

	if cfg.EnablePubSub {
		go pool.processEvents()
	}

	return pool
}

func (p *LocalPool) processEvents() {
	for event := range p.eventQueue {
		p.mu.RLock()
		for _, sub := range p.subscribers {
			if sub.GroupID == event.GroupID && sub.connected {
				select {
				case sub.Channel <- event:
				default:
				}
			}
		}
		p.mu.RUnlock()
	}
}

func (p *LocalPool) PublishEvent(ctx context.Context, event *types.MemoryPoolEvent) error {
	if !p.config.EnablePubSub {
		return nil
	}

	event.Timestamp = time.Now()

	select {
	case p.eventQueue <- event:
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	return nil
}

func (p *LocalPool) Subscribe(ctx context.Context, groupID, agentID string) (*PoolSubscriber, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	subscriber := &PoolSubscriber{
		ID:      uuid.New().String(),
		GroupID: groupID,
		AgentID: agentID,
	}

	eventChan := make(chan *types.MemoryPoolEvent, 100)
	subscriber.Channel = eventChan
	subscriber.connected = true

	p.subscribers[subscriber.ID] = subscriber

	return subscriber, nil
}

func (p *LocalPool) Unsubscribe(subscriberID string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if sub, ok := p.subscribers[subscriberID]; ok {
		sub.mu.Lock()
		sub.connected = false
		sub.mu.Unlock()
		close(sub.Channel)
		delete(p.subscribers, subscriberID)
	}

	return nil
}

func (p *LocalPool) GetSharedMemories(ctx context.Context, groupID string, limit int) ([]*types.Memory, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if limit <= 0 {
		limit = 50
	}

	return nil, nil
}

func (p *LocalPool) ShareMemory(ctx context.Context, memory *types.Memory, groupID, sharedBy string) error {
	event := &types.MemoryPoolEvent{
		Type:      "memory.shared",
		GroupID:   groupID,
		AgentID:   sharedBy,
		MemoryID:  memory.ID,
		Timestamp: time.Now(),
	}

	return p.PublishEvent(ctx, event)
}

func (p *LocalPool) GetGroupAgents(ctx context.Context, groupID string) ([]string, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	var agents []string
	seen := make(map[string]bool)

	for _, sub := range p.subscribers {
		if sub.GroupID == groupID && !seen[sub.AgentID] {
			agents = append(agents, sub.AgentID)
			seen[sub.AgentID] = true
		}
	}

	return agents, nil
}

func (p *LocalPool) SyncToGroup(ctx context.Context, groupID string, memory *types.Memory) error {
	event := &types.MemoryPoolEvent{
		Type:      "memory.sync",
		GroupID:   groupID,
		MemoryID:  memory.ID,
		Data:      map[string]interface{}{"content": memory.Content},
		Timestamp: time.Now(),
	}

	return p.PublishEvent(ctx, event)
}

func (p *LocalPool) Ping(ctx context.Context) error {
	return nil
}

func (p *LocalPool) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	close(p.eventQueue)

	for id := range p.subscribers {
		delete(p.subscribers, id)
	}

	return nil
}

type PoolManager struct {
	pools  map[string]MemoryPool
	mu     sync.RWMutex
	config *PoolConfig
}

func NewPoolManager(cfg *PoolConfig) (*PoolManager, error) {
	if cfg == nil {
		cfg = &PoolConfig{
			PoolKeyPrefix:  "agentmemory:pool:",
			EventKeyPrefix: "agentmemory:events:",
			MaxPoolSize:    1000,
			SyncIntervalMs: 1000,
			EnablePubSub:   true,
		}
	}

	return &PoolManager{
		pools:  make(map[string]MemoryPool),
		config: cfg,
	}, nil
}

func (m *PoolManager) GetPool(tenantID string) (MemoryPool, error) {
	m.mu.RLock()
	if pool, ok := m.pools[tenantID]; ok {
		m.mu.RUnlock()
		return pool, nil
	}
	m.mu.RUnlock()

	m.mu.Lock()
	defer m.mu.Unlock()

	if pool, ok := m.pools[tenantID]; ok {
		return pool, nil
	}

	pool, err := NewPool(m.config)
	if err != nil {
		return nil, err
	}

	m.pools[tenantID] = pool
	return pool, nil
}

func (m *PoolManager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, pool := range m.pools {
		pool.Close()
	}

	return nil
}

type EventBroadcaster struct {
	tenantPools map[string]*LocalPool
	mu          sync.RWMutex
}

func NewEventBroadcaster() *EventBroadcaster {
	return &EventBroadcaster{
		tenantPools: make(map[string]*LocalPool),
	}
}

func (b *EventBroadcaster) GetPool(tenantID string) *LocalPool {
	b.mu.RLock()
	if pool, ok := b.tenantPools[tenantID]; ok {
		b.mu.RUnlock()
		return pool
	}
	b.mu.RUnlock()

	b.mu.Lock()
	defer b.mu.Unlock()

	if pool, ok := b.tenantPools[tenantID]; ok {
		return pool
	}

	pool := NewLocalPool(nil)
	b.tenantPools[tenantID] = pool
	return pool
}

func (b *EventBroadcaster) PublishToGroup(ctx context.Context, tenantID, groupID string, eventType string, data interface{}) error {
	pool := b.GetPool(tenantID)

	event := &types.MemoryPoolEvent{
		Type:      eventType,
		GroupID:   groupID,
		Data:      toMap(data),
		Timestamp: time.Now(),
	}

	return pool.PublishEvent(ctx, event)
}

func (b *EventBroadcaster) SubscribeToGroup(ctx context.Context, tenantID, groupID, agentID string) (*PoolSubscriber, error) {
	pool := b.GetPool(tenantID)
	return pool.Subscribe(ctx, groupID, agentID)
}

func (b *EventBroadcaster) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	for _, pool := range b.tenantPools {
		pool.Close()
	}

	return nil
}

func toMap(v interface{}) map[string]interface{} {
	if v == nil {
		return nil
	}

	switch val := v.(type) {
	case map[string]interface{}:
		return val
	case []byte:
		var result map[string]interface{}
		if err := json.Unmarshal(val, &result); err == nil {
			return result
		}
	default:
	}

	return nil
}
