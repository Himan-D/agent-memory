package connections

import (
	"context"
	"fmt"
	"testing"
	"time"

	"agent-memory/internal/memory/types"
)

type mockMemoryServiceStore struct {
	memories map[string]*types.Memory
}

func newMockMemoryServiceStore() *mockMemoryServiceStore {
	return &mockMemoryServiceStore{
		memories: make(map[string]*types.Memory),
	}
}

func (m *mockMemoryServiceStore) CreateMemoryWithOptions(ctx context.Context, mem *types.Memory, skip bool) (*types.Memory, error) {
	if mem.ID == "" {
		mem.ID = fmt.Sprintf("mem-%d", time.Now().UnixNano())
	}
	m.memories[mem.ID] = mem
	return mem, nil
}

func (m *mockMemoryServiceStore) GetMemory(ctx context.Context, id string) (*types.Memory, error) {
	if mem, ok := m.memories[id]; ok {
		return mem, nil
	}
	return nil, fmt.Errorf("memory not found")
}

func (m *mockMemoryServiceStore) UpdateMemory(ctx context.Context, id string, content string, metadata map[string]interface{}) error {
	if mem, ok := m.memories[id]; ok {
		mem.Content = content
		if mem.Metadata == nil {
			mem.Metadata = make(map[string]interface{})
		}
		for k, v := range metadata {
			mem.Metadata[k] = v
		}
		mem.UpdatedAt = time.Now().UTC()
		return nil
	}
	return fmt.Errorf("memory not found")
}

func (m *mockMemoryServiceStore) DeleteMemory(ctx context.Context, id string) error {
	if _, ok := m.memories[id]; ok {
		delete(m.memories, id)
		return nil
	}
	return fmt.Errorf("memory not found")
}

func (m *mockMemoryServiceStore) GetMemoriesByUser(ctx context.Context, userID string) ([]*types.Memory, error) {
	var result []*types.Memory
	for _, mem := range m.memories {
		if mem.UserID == userID {
			result = append(result, mem)
		}
	}
	return result, nil
}

func (m *mockMemoryServiceStore) GetMemoriesByOrg(ctx context.Context, orgID string) ([]*types.Memory, error) {
	var result []*types.Memory
	for _, mem := range m.memories {
		if mem.OrgID == orgID {
			result = append(result, mem)
		}
	}
	return result, nil
}

func (m *mockMemoryServiceStore) GetAllMemories(ctx context.Context) ([]*types.Memory, error) {
	var result []*types.Memory
	for _, mem := range m.memories {
		result = append(result, mem)
	}
	return result, nil
}

func TestMemoryStore_Save(t *testing.T) {
	mockStore := newMockMemoryServiceStore()
	store := NewMemoryStore(mockStore)

	ctx := context.Background()
	conn := &Connection{
		ID:        "conn-1",
		Provider:  ProviderNotion,
		Status:    StatusActive,
		UserID:    "user-1",
		OrgID:     "org-1",
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}

	// Test 1: Save new connection
	err := store.Save(ctx, conn)
	if err != nil {
		t.Fatalf("Failed to save new connection: %v", err)
	}

	if len(mockStore.memories) != 1 {
		t.Fatalf("Expected 1 memory, got %d", len(mockStore.memories))
	}

	mem := mockStore.memories["conn-1"]
	if mem.Category != connectionCategory {
		t.Errorf("Expected category %s, got %s", connectionCategory, mem.Category)
	}

	// Test 2: Save existing connection (Update)
	conn.Status = StatusError
	err = store.Save(ctx, conn)
	if err != nil {
		t.Fatalf("Failed to update connection: %v", err)
	}

	if len(mockStore.memories) != 1 {
		t.Fatalf("Expected 1 memory, got %d", len(mockStore.memories))
	}

	mem = mockStore.memories["conn-1"]
	if mem.Metadata["status"] != StatusError {
		t.Errorf("Expected status in metadata to be %s, got %v", StatusError, mem.Metadata["status"])
	}
}

func TestMemoryStore_Get(t *testing.T) {
	mockStore := newMockMemoryServiceStore()
	store := NewMemoryStore(mockStore)

	ctx := context.Background()
	conn := &Connection{
		ID:        "conn-2",
		Provider:  ProviderGitHub,
		Status:    StatusActive,
		UserID:    "user-2",
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}

	// Pre-populate mock
	store.Save(ctx, conn)

	retrievedConn, err := store.Get(ctx, "conn-2")
	if err != nil {
		t.Fatalf("Failed to get connection: %v", err)
	}

	if retrievedConn.ID != conn.ID {
		t.Errorf("Expected ID %s, got %s", conn.ID, retrievedConn.ID)
	}
	if retrievedConn.Provider != conn.Provider {
		t.Errorf("Expected Provider %s, got %s", conn.Provider, retrievedConn.Provider)
	}
}

func TestMemoryStore_List(t *testing.T) {
	mockStore := newMockMemoryServiceStore()
	store := NewMemoryStore(mockStore)
	ctx := context.Background()

	// Seed connections
	store.Save(ctx, &Connection{ID: "conn-u1", UserID: "user-1", UpdatedAt: time.Now().Add(-1 * time.Hour)})
	store.Save(ctx, &Connection{ID: "conn-u2", UserID: "user-2", UpdatedAt: time.Now().Add(-2 * time.Hour)})
	store.Save(ctx, &Connection{ID: "conn-u1-2", UserID: "user-1", OrgID: "org-1", UpdatedAt: time.Now()})
	store.Save(ctx, &Connection{ID: "conn-o1", OrgID: "org-1", UpdatedAt: time.Now().Add(-3 * time.Hour)})

	// Add a non-connection memory
	mockStore.memories["non-conn"] = &types.Memory{
		ID:       "non-conn",
		Category: "conversation",
		UserID:   "user-1",
	}

	tests := []struct {
		name     string
		scope    Scope
		expected int
	}{
		{"By User", Scope{UserID: "user-1"}, 2},
		{"By Org", Scope{OrgID: "org-1"}, 2},
		{"All", Scope{}, 4},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := store.List(ctx, tt.scope)
			if err != nil {
				t.Fatalf("Failed to list connections: %v", err)
			}
			if len(results) != tt.expected {
				t.Errorf("Expected %d results, got %d", tt.expected, len(results))
			}
			// Verify sorting (most recent first)
			if len(results) > 1 {
				if results[0].UpdatedAt.Before(results[1].UpdatedAt) {
					t.Errorf("Results not sorted by UpdatedAt descending")
				}
			}
		})
	}
}

func TestMemoryStore_Delete(t *testing.T) {
	mockStore := newMockMemoryServiceStore()
	store := NewMemoryStore(mockStore)
	ctx := context.Background()

	conn := &Connection{ID: "conn-del"}
	store.Save(ctx, conn)

	err := store.Delete(ctx, "conn-del")
	if err != nil {
		t.Fatalf("Failed to delete connection: %v", err)
	}

	if len(mockStore.memories) != 0 {
		t.Errorf("Expected 0 memories after delete, got %d", len(mockStore.memories))
	}
}

func TestMemoryStore_UnmarshalFallback(t *testing.T) {
	mem := &types.Memory{
		ID:        "mem-123",
		UserID:    "user-x",
		OrgID:     "org-x",
		TenantID:  "tenant-x",
		Content:   `{"provider": "notion"}`, // Missing ID, UserID, OrgID, TenantID in JSON
		CreatedAt: time.Now().Add(-1 * time.Hour),
		UpdatedAt: time.Now(),
	}

	conn, err := unmarshalConnection(mem)
	if err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if conn.ID != "mem-123" {
		t.Errorf("Expected ID fallback to 'mem-123', got '%s'", conn.ID)
	}
	if conn.UserID != "user-x" {
		t.Errorf("Expected UserID fallback to 'user-x', got '%s'", conn.UserID)
	}
	if conn.OrgID != "org-x" {
		t.Errorf("Expected OrgID fallback to 'org-x', got '%s'", conn.OrgID)
	}
	if conn.TenantID != "tenant-x" {
		t.Errorf("Expected TenantID fallback to 'tenant-x', got '%s'", conn.TenantID)
	}
	if !conn.CreatedAt.Equal(mem.CreatedAt) {
		t.Errorf("Expected CreatedAt fallback to memory CreatedAt")
	}
	if !conn.UpdatedAt.Equal(mem.UpdatedAt) {
		t.Errorf("Expected UpdatedAt fallback to memory UpdatedAt")
	}
}
