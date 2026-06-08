package connections

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"agent-memory/internal/memory/types"
)

const connectionCategory = "connection"

type MemoryServiceStore interface {
	CreateMemoryWithOptions(ctx context.Context, mem *types.Memory, skip bool) (*types.Memory, error)
	GetMemory(ctx context.Context, id string) (*types.Memory, error)
	UpdateMemory(ctx context.Context, id string, content string, metadata map[string]interface{}) error
	DeleteMemory(ctx context.Context, id string) error
	GetMemoriesByUser(ctx context.Context, userID string) ([]*types.Memory, error)
	GetMemoriesByOrg(ctx context.Context, orgID string) ([]*types.Memory, error)
	GetAllMemories(ctx context.Context) ([]*types.Memory, error)
}

type MemoryStore struct {
	mem MemoryServiceStore
}

func NewMemoryStore(mem MemoryServiceStore) *MemoryStore {
	return &MemoryStore{mem: mem}
}

func (s *MemoryStore) Save(ctx context.Context, connection *Connection) error {
	content, err := marshalConnection(connection)
	if err != nil {
		return err
	}
	meta := map[string]interface{}{
		"connection_id": connection.ID,
		"provider":      connection.Provider,
		"status":        connection.Status,
	}
	if _, err := s.mem.GetMemory(ctx, connection.ID); err == nil {
		return s.mem.UpdateMemory(ctx, connection.ID, content, meta)
	}
	_, err = s.mem.CreateMemoryWithOptions(ctx, &types.Memory{
		ID:        connection.ID,
		TenantID:  connection.TenantID,
		UserID:    connection.UserID,
		OrgID:     connection.OrgID,
		Type:      types.MemoryTypeOrg,
		Content:   content,
		Category:  connectionCategory,
		Metadata:  meta,
		Status:    types.MemoryStatusActive,
		CreatedAt: connection.CreatedAt,
		UpdatedAt: connection.UpdatedAt,
	}, true)
	return err
}

func (s *MemoryStore) Get(ctx context.Context, id string) (*Connection, error) {
	mem, err := s.mem.GetMemory(ctx, id)
	if err != nil {
		return nil, err
	}
	return unmarshalConnection(mem)
}

func (s *MemoryStore) List(ctx context.Context, scope Scope) ([]*Connection, error) {
	var memories []*types.Memory
	var err error
	switch {
	case scope.UserID != "":
		memories, err = s.mem.GetMemoriesByUser(ctx, scope.UserID)
	case scope.OrgID != "":
		memories, err = s.mem.GetMemoriesByOrg(ctx, scope.OrgID)
	default:
		memories, err = s.mem.GetAllMemories(ctx)
	}
	if err != nil {
		return nil, err
	}
	conns := make([]*Connection, 0)
	for _, mem := range memories {
		if mem == nil || mem.Category != connectionCategory {
			continue
		}
		conn, parseErr := unmarshalConnection(mem)
		if parseErr != nil {
			continue
		}
		conns = append(conns, conn)
	}
	sort.Slice(conns, func(i, j int) bool {
		return conns[i].UpdatedAt.After(conns[j].UpdatedAt)
	})
	return conns, nil
}

func (s *MemoryStore) Delete(ctx context.Context, id string) error {
	return s.mem.DeleteMemory(ctx, id)
}

func marshalConnection(connection *Connection) (string, error) {
	data, err := json.Marshal(connection)
	if err != nil {
		return "", fmt.Errorf("marshal connection: %w", err)
	}
	return string(data), nil
}

func unmarshalConnection(mem *types.Memory) (*Connection, error) {
	if mem == nil {
		return nil, fmt.Errorf("connection not found")
	}
	var conn Connection
	if err := json.Unmarshal([]byte(mem.Content), &conn); err != nil {
		return nil, fmt.Errorf("decode connection: %w", err)
	}
	if conn.ID == "" {
		conn.ID = mem.ID
	}
	if conn.UserID == "" {
		conn.UserID = mem.UserID
	}
	if conn.OrgID == "" {
		conn.OrgID = mem.OrgID
	}
	if conn.TenantID == "" {
		conn.TenantID = mem.TenantID
	}
	if conn.CreatedAt.IsZero() {
		conn.CreatedAt = mem.CreatedAt
	}
	if conn.UpdatedAt.IsZero() {
		conn.UpdatedAt = mem.UpdatedAt
	}
	if conn.CreatedAt.IsZero() {
		conn.CreatedAt = time.Now().UTC()
	}
	if conn.UpdatedAt.IsZero() {
		conn.UpdatedAt = conn.CreatedAt
	}
	return &conn, nil
}
