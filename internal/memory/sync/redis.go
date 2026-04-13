package sync

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"agent-memory/internal/memory/types"
)

type RedisPool struct {
	client      *redis.Client
	subscribers map[string]*RedisSubscriber
	eventQueue  chan *types.MemoryPoolEvent
	config      *PoolConfig
	mu          sync.RWMutex
}

type RedisSubscriber struct {
	ID        string
	GroupID   string
	AgentID   string
	Channel   chan *types.MemoryPoolEvent
	connected bool
	mu        sync.RWMutex
}

func NewRedisPool(cfg *PoolConfig) (*RedisPool, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is required for RedisPool")
	}

	if cfg.RedisURL == "" {
		return nil, fmt.Errorf("redis URL is required")
	}

	client := redis.NewClient(&redis.Options{
		Addr:         cfg.RedisURL,
		PoolSize:     cfg.MaxPoolSize,
		MinIdleConns: 5,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	pool := &RedisPool{
		client:      client,
		subscribers: make(map[string]*RedisSubscriber),
		eventQueue:  make(chan *types.MemoryPoolEvent, 1000),
		config:      cfg,
	}

	if cfg.EnablePubSub {
		go pool.processEvents()
	}

	return pool, nil
}

func (p *RedisPool) processEvents() {
	for event := range p.eventQueue {
		channel := fmt.Sprintf("%s%s", p.config.EventKeyPrefix, event.GroupID)

		data, err := json.Marshal(event)
		if err != nil {
			continue
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		p.client.Publish(ctx, channel, data)
		cancel()
	}
}

func (p *RedisPool) PublishEvent(ctx context.Context, event *types.MemoryPoolEvent) error {
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

func (p *RedisPool) Subscribe(ctx context.Context, groupID, agentID string) (*PoolSubscriber, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	channel := make(chan *types.MemoryPoolEvent, 100)
	subscriber := &RedisSubscriber{
		ID:        uuid.New().String(),
		GroupID:   groupID,
		AgentID:   agentID,
		Channel:   channel,
		connected: true,
	}

	go p.listenToGroup(groupID, subscriber)

	p.subscribers[subscriber.ID] = subscriber

	return &PoolSubscriber{
		ID:      subscriber.ID,
		GroupID: groupID,
		AgentID: agentID,
		Channel: channel,
	}, nil
}

func (p *RedisPool) listenToGroup(groupID string, sub *RedisSubscriber) {
	channel := fmt.Sprintf("%s%s", p.config.EventKeyPrefix, groupID)

	ctx := context.Background()
	pubsub := p.client.Subscribe(ctx, channel)

	defer pubsub.Close()

	ch := pubsub.Channel()

	for msg := range ch {
		if !sub.connected {
			return
		}

		var event types.MemoryPoolEvent
		if err := json.Unmarshal([]byte(msg.Payload), &event); err != nil {
			continue
		}

		select {
		case sub.Channel <- &event:
		default:
		}
	}
}

func (p *RedisPool) Unsubscribe(subscriberID string) error {
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

func (p *RedisPool) GetSharedMemories(ctx context.Context, groupID string, limit int) ([]*types.Memory, error) {
	if limit <= 0 {
		limit = 50
	}

	key := fmt.Sprintf("%s%s:memories", p.config.PoolKeyPrefix, groupID)

	data, err := p.client.LRange(ctx, key, 0, int64(limit-1)).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get shared memories: %w", err)
	}

	memories := make([]*types.Memory, 0, len(data))
	for _, d := range data {
		var memory types.Memory
		if err := json.Unmarshal([]byte(d), &memory); err != nil {
			continue
		}
		memories = append(memories, &memory)
	}

	return memories, nil
}

func (p *RedisPool) ShareMemory(ctx context.Context, memory *types.Memory, groupID, sharedBy string) error {
	key := fmt.Sprintf("%s%s:memories", p.config.PoolKeyPrefix, groupID)

	data, err := json.Marshal(memory)
	if err != nil {
		return fmt.Errorf("failed to marshal memory: %w", err)
	}

	if err := p.client.RPush(ctx, key, data).Err(); err != nil {
		return fmt.Errorf("failed to share memory: %w", err)
	}

	event := &types.MemoryPoolEvent{
		Type:      "memory.shared",
		GroupID:   groupID,
		AgentID:   sharedBy,
		MemoryID:  memory.ID,
		Timestamp: time.Now(),
	}

	return p.PublishEvent(ctx, event)
}

func (p *RedisPool) GetGroupAgents(ctx context.Context, groupID string) ([]string, error) {
	key := fmt.Sprintf("%s%s:agents", p.config.PoolKeyPrefix, groupID)

	agents, err := p.client.SMembers(ctx, key).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get group agents: %w", err)
	}

	return agents, nil
}

func (p *RedisPool) SyncToGroup(ctx context.Context, groupID string, memory *types.Memory) error {
	event := &types.MemoryPoolEvent{
		Type:      "memory.sync",
		GroupID:   groupID,
		MemoryID:  memory.ID,
		Data:      map[string]interface{}{"content": memory.Content},
		Timestamp: time.Now(),
	}

	return p.PublishEvent(ctx, event)
}

func (p *RedisPool) Ping(ctx context.Context) error {
	return p.client.Ping(ctx).Err()
}

func (p *RedisPool) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	close(p.eventQueue)

	for id, sub := range p.subscribers {
		sub.mu.Lock()
		sub.connected = false
		sub.mu.Unlock()
		close(sub.Channel)
		delete(p.subscribers, id)
	}

	return p.client.Close()
}

type RedisPoolManager struct {
	pools  map[string]*RedisPool
	mu     sync.RWMutex
	config *PoolConfig
}

func NewRedisPoolManager(cfg *PoolConfig) (*RedisPoolManager, error) {
	if cfg == nil {
		cfg = &PoolConfig{
			PoolKeyPrefix:  "agentmemory:pool:",
			EventKeyPrefix: "agentmemory:events:",
			MaxPoolSize:    1000,
			SyncIntervalMs: 1000,
			EnablePubSub:   true,
		}
	}

	return &RedisPoolManager{
		pools:  make(map[string]*RedisPool),
		config: cfg,
	}, nil
}

func (m *RedisPoolManager) GetPool(tenantID string) (*RedisPool, error) {
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

	pool, err := NewRedisPool(m.config)
	if err != nil {
		return nil, err
	}

	m.pools[tenantID] = pool
	return pool, nil
}

func (m *RedisPoolManager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, pool := range m.pools {
		pool.Close()
	}

	return nil
}
