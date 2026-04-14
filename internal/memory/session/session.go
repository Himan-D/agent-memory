package session

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"
	"time"

	"agent-memory/internal/memory/datapoint"
)

type Memory struct {
	ID        string                 `json:"id"`
	SessionID string                 `json:"session_id"`
	Content   string                 `json:"content"`
	Type      string                 `json:"type"`
	Role      string                 `json:"role"`
	Metadata  map[string]interface{} `json:"metadata"`
	CreatedAt time.Time              `json:"created_at"`
}

type Session struct {
	ID        string     `json:"id"`
	UserID    string     `json:"user_id"`
	Name      string     `json:"name"`
	Status    string     `json:"status"`
	Memories  []*Memory  `json:"memories"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

type Cache struct {
	mu       sync.RWMutex
	sessions map[string]*Session
	memories map[string][]*Memory
}

func NewCache() *Cache {
	return &Cache{
		sessions: make(map[string]*Session),
		memories: make(map[string][]*Memory),
	}
}

func (c *Cache) CreateSession(session *Session) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if session.ID == "" {
		return fmt.Errorf("session ID is required")
	}

	session.CreatedAt = time.Now()
	session.UpdatedAt = time.Now()
	c.sessions[session.ID] = session
	return nil
}

func (c *Cache) GetSession(sessionID string) (*Session, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	session, ok := c.sessions[sessionID]
	if !ok {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}

	if session.ExpiresAt != nil && time.Now().After(*session.ExpiresAt) {
		return nil, fmt.Errorf("session expired: %s", sessionID)
	}

	return session, nil
}

func (c *Cache) UpdateSession(sessionID string, updates map[string]interface{}) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	session, ok := c.sessions[sessionID]
	if !ok {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	if name, ok := updates["name"].(string); ok {
		session.Name = name
	}
	if status, ok := updates["status"].(string); ok {
		session.Status = status
	}
	if expiresAt, ok := updates["expires_at"].(time.Time); ok {
		session.ExpiresAt = &expiresAt
	}

	session.UpdatedAt = time.Now()
	return nil
}

func (c *Cache) DeleteSession(sessionID string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.sessions, sessionID)
	delete(c.memories, sessionID)
	return nil
}

func (c *Cache) AddMemory(sessionID string, memory *Memory) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	memory.CreatedAt = time.Now()
	c.memories[sessionID] = append(c.memories[sessionID], memory)

	if session, ok := c.sessions[sessionID]; ok {
		session.UpdatedAt = time.Now()
	}

	return nil
}

func (c *Cache) GetMemories(sessionID string, limit int) ([]*Memory, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	memories, ok := c.memories[sessionID]
	if !ok {
		return nil, nil
	}

	if limit <= 0 || limit > len(memories) {
		limit = len(memories)
	}

	return memories[:limit], nil
}

func (c *Cache) GetRecentMemories(sessionID string, count int) ([]*Memory, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	memories, ok := c.memories[sessionID]
	if !ok || len(memories) == 0 {
		return nil, nil
	}

	if count <= 0 || count > len(memories) {
		count = len(memories)
	}

	start := len(memories) - count
	return memories[start:], nil
}

func (c *Cache) SearchMemories(sessionID, query string) ([]*Memory, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var results []*Memory
	for _, m := range c.memories[sessionID] {
		if strings.Contains(strings.ToLower(m.Content), strings.ToLower(query)) {
			results = append(results, m)
		}
	}

	return results, nil
}

func (c *Cache) ClearMemories(sessionID string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.memories[sessionID] = make([]*Memory, 0)
	return nil
}

func (c *Cache) ListSessions() []*Session {
	c.mu.RLock()
	defer c.mu.RUnlock()

	sessions := make([]*Session, 0, len(c.sessions))
	for _, s := range c.sessions {
		sessions = append(sessions, s)
	}

	return sessions
}

type GraphSyncer interface {
	SyncToGraph(ctx context.Context, sessionID string, memories []*Memory) error
}

type Manager struct {
	cache        *Cache
	graphSync    GraphSyncer
	maxMemoryAge time.Duration
	cleanupTick  time.Duration
}

func NewManager() *Manager {
	return &Manager{
		cache:        NewCache(),
		maxMemoryAge: 24 * time.Hour,
		cleanupTick:  1 * time.Hour,
	}
}

func (m *Manager) SetGraphSyncer(syncer GraphSyncer) {
	m.graphSync = syncer
}

func (m *Manager) CreateSession(ctx context.Context, userID, name string) (*Session, error) {
	session := &Session{
		ID:     generateID(),
		UserID: userID,
		Name:   name,
		Status: "active",
	}

	if err := m.cache.CreateSession(session); err != nil {
		return nil, err
	}

	return session, nil
}

func (m *Manager) GetSession(ctx context.Context, sessionID string) (*Session, error) {
	return m.cache.GetSession(sessionID)
}

func (m *Manager) AddMemory(ctx context.Context, sessionID, content, role string) (*Memory, error) {
	memory := &Memory{
		ID:        generateID(),
		SessionID: sessionID,
		Content:   content,
		Role:      role,
		Type:      "message",
		Metadata:  make(map[string]interface{}),
	}

	if err := m.cache.AddMemory(sessionID, memory); err != nil {
		return nil, err
	}

	return memory, nil
}

func (m *Manager) GetContext(ctx context.Context, sessionID string, maxTokens int) ([]*datapoint.DataPoint, error) {
	memories, err := m.cache.GetRecentMemories(sessionID, 100)
	if err != nil {
		return nil, err
	}

	points := make([]*datapoint.DataPoint, 0, len(memories))
	currentTokens := 0

	for i := len(memories) - 1; i >= 0; i-- {
		m := memories[i]
		tokenEstimate := len(m.Content) / 4

		if currentTokens+tokenEstimate > maxTokens {
			break
		}

		dp := datapoint.New(m.Content, datapoint.DataPointTypeChunk)
		dp.SetSource("session", sessionID, "")
		dp.Metadata["role"] = m.Role
		dp.Metadata["memory_id"] = m.ID
		points = append(points, dp)
		currentTokens += tokenEstimate
	}

	return points, nil
}

func (m *Manager) Search(ctx context.Context, sessionID, query string) ([]*datapoint.DataPoint, error) {
	memories, err := m.cache.SearchMemories(sessionID, query)
	if err != nil {
		return nil, err
	}

	points := make([]*datapoint.DataPoint, 0, len(memories))
	for _, mem := range memories {
		dp := datapoint.New(mem.Content, datapoint.DataPointTypeChunk)
		dp.SetSource("session", sessionID, "")
		points = append(points, dp)
	}

	return points, nil
}

func (m *Manager) SyncToGraph(ctx context.Context, sessionID string) error {
	if m.graphSync == nil {
		return nil
	}

	memories, err := m.cache.GetMemories(sessionID, 1000)
	if err != nil {
		return err
	}

	return m.graphSync.SyncToGraph(ctx, sessionID, memories)
}

func (m *Manager) Compact(ctx context.Context, sessionID string, maxTokens int) error {
	memories, err := m.cache.GetMemories(sessionID, 1000)
	if err != nil {
		return err
	}

	if len(memories) <= 10 {
		return nil
	}

	newMemories := memories[len(memories)-10:]

	m.cache.ClearMemories(sessionID)

	for _, mem := range newMemories {
		m.cache.AddMemory(sessionID, mem)
	}

	return nil
}

type SessionManager interface {
	CreateSession(ctx context.Context, userID, name string) (*Session, error)
	GetSession(ctx context.Context, sessionID string) (*Session, error)
	AddMemory(ctx context.Context, sessionID, content, role string) (*Memory, error)
	GetContext(ctx context.Context, sessionID string, maxTokens int) ([]*datapoint.DataPoint, error)
	Search(ctx context.Context, sessionID, query string) ([]*datapoint.DataPoint, error)
	Compact(ctx context.Context, sessionID string, maxTokens int) error
}

func generateID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return hex.EncodeToString(b)
}
