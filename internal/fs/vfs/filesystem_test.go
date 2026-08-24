package vfs

import (
	"context"
	"testing"
	"time"

	"agent-memory/internal/memory/types"
	"agent-memory/internal/storage"
)

type mockSvc struct {
	mems map[string]*types.Memory
}

func newMock() *mockSvc {
	return &mockSvc{mems: map[string]*types.Memory{}}
}

func (m *mockSvc) SearchMemories(ctx context.Context, req *types.SearchRequest) ([]types.MemoryResult, error) {
	var out []types.MemoryResult
	for _, mem := range m.mems {
		out = append(out, types.MemoryResult{MemoryID: mem.ID, Text: mem.Content, Score: 1})
	}
	return out, nil
}
func (m *mockSvc) GetMemory(ctx context.Context, id string) (*types.Memory, error) {
	if mem, ok := m.mems[id]; ok {
		return mem, nil
	}
	return nil, context.Canceled
}
func (m *mockSvc) CreateMemory(ctx context.Context, mem *types.Memory) (*types.Memory, error) {
	if mem.ID == "" {
		mem.ID = "mem_test_1"
	}
	mem.CreatedAt = time.Now()
	mem.UpdatedAt = time.Now()
	m.mems[mem.ID] = mem
	return mem, nil
}
func (m *mockSvc) UpdateMemory(ctx context.Context, id, content string, meta map[string]interface{}) error {
	if mem, ok := m.mems[id]; ok {
		mem.Content = content
		return nil
	}
	return context.Canceled
}
func (m *mockSvc) DeleteMemory(ctx context.Context, id string) error {
	delete(m.mems, id)
	return nil
}
func (m *mockSvc) ListSkills(context.Context, string, string, int, int) ([]*types.Skill, error) {
	return []*types.Skill{{ID: "sk1", Name: "demo", Domain: "test", Trigger: "hi", Action: "hello"}}, nil
}
func (m *mockSvc) ListSessions(context.Context, string) ([]*types.Session, error) {
	return nil, nil
}
func (m *mockSvc) GetEntity(context.Context, string) (*types.Entity, error) {
	return nil, context.Canceled
}
func (m *mockSvc) GetMemoriesByTenant(ctx context.Context, tenantID string, limit int) ([]*types.Memory, error) {
	var out []*types.Memory
	for _, mem := range m.mems {
		out = append(out, mem)
	}
	return out, nil
}

func TestVirtualFSMemoryCRUD(t *testing.T) {
	svc := newMock()
	dir := t.TempDir()
	blob, _ := storage.NewLocalStore(dir)
	fs := NewVirtualFSWithOptions(svc, "/tmp/agent", "t1", blob, "pfx/")

	ctx := context.Background()
	entries, err := fs.ReadDir(ctx, "/")
	if err != nil || len(entries) < 5 {
		t.Fatalf("root: %v %v", entries, err)
	}

	// create via new.md
	err = fs.WriteFile(ctx, "/memories/new.md", []byte("---\nuser_id: u1\n---\n\nhello agentfs\n"))
	if err != nil {
		t.Fatal(err)
	}
	if len(svc.mems) != 1 {
		t.Fatalf("expected 1 memory, got %d", len(svc.mems))
	}

	mems, err := fs.ReadDir(ctx, "/memories")
	if err != nil {
		t.Fatal(err)
	}
	// new.md + created
	if len(mems) < 2 {
		t.Fatalf("memories dir: %+v", mems)
	}

	// archive write
	if err := fs.WriteFile(ctx, "/archive/note.txt", []byte("blob")); err != nil {
		t.Fatal(err)
	}
	data, err := fs.ReadFile(ctx, "/archive/note.txt")
	if err != nil || string(data) != "blob" {
		t.Fatalf("archive: %s %v", data, err)
	}

	// search path
	_ = fs.WriteFile(ctx, "/search/query.txt", []byte("hello"))
	res, err := fs.ReadFile(ctx, "/search/results.md")
	if err != nil || len(res) == 0 {
		t.Fatalf("search results: %v", err)
	}

	st := fs.Status()
	if st == "" {
		t.Fatal("empty status")
	}
}

func TestParseMarkdownMemory(t *testing.T) {
	body, meta := parseMarkdownMemory("---\nuser_id: alice\n---\n\ncontent here\n")
	if meta["user_id"] != "alice" || body != "content here" {
		t.Fatalf("%q %v", body, meta)
	}
}
