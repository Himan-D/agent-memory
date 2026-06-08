package sources

import (
	"bytes"
	"context"
	"fmt"
	"sort"
	"strings"
	"testing"

	"agent-memory/internal/memory/types"
)

type fakeMemoryService struct {
	memories map[string]*types.Memory
	deleted  []string
}

func newFakeMemoryService() *fakeMemoryService {
	return &fakeMemoryService{memories: map[string]*types.Memory{}}
}

func (f *fakeMemoryService) CreateMemory(ctx context.Context, mem *types.Memory) (*types.Memory, error) {
	cp := *mem
	f.memories[mem.ID] = &cp
	return &cp, nil
}

func (f *fakeMemoryService) GetMemory(ctx context.Context, id string) (*types.Memory, error) {
	mem, ok := f.memories[id]
	if !ok {
		return nil, fmt.Errorf("not found")
	}
	cp := *mem
	return &cp, nil
}

func (f *fakeMemoryService) DeleteMemory(ctx context.Context, id string) error {
	delete(f.memories, id)
	f.deleted = append(f.deleted, id)
	return nil
}

func (f *fakeMemoryService) GetMemoriesByUser(ctx context.Context, userID string) ([]*types.Memory, error) {
	return f.filtered(func(mem *types.Memory) bool { return mem.UserID == userID }), nil
}

func (f *fakeMemoryService) GetMemoriesByOrg(ctx context.Context, orgID string) ([]*types.Memory, error) {
	return f.filtered(func(mem *types.Memory) bool { return mem.OrgID == orgID }), nil
}

func (f *fakeMemoryService) SearchMemories(ctx context.Context, req *types.SearchRequest) ([]types.MemoryResult, error) {
	memories := f.filtered(func(mem *types.Memory) bool {
		return strings.Contains(strings.ToLower(mem.Content), strings.ToLower(req.Query))
	})
	results := make([]types.MemoryResult, 0, len(memories))
	for _, mem := range memories {
		results = append(results, types.MemoryResult{MemoryID: mem.ID, Text: mem.Content, Metadata: mem, Score: 1})
	}
	return results, nil
}

func (f *fakeMemoryService) filtered(match func(*types.Memory) bool) []*types.Memory {
	out := make([]*types.Memory, 0)
	for _, mem := range f.memories {
		if match(mem) {
			cp := *mem
			out = append(out, &cp)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

type fakeBlobStore struct {
	puts    map[string][]byte
	deletes []string
}

func (f *fakeBlobStore) Put(ctx context.Context, key string, data []byte, contentType string) error {
	if f.puts == nil {
		f.puts = map[string][]byte{}
	}
	f.puts[key] = append([]byte(nil), data...)
	return nil
}

func (f *fakeBlobStore) Delete(ctx context.Context, key string) error {
	f.deletes = append(f.deletes, key)
	return nil
}

func TestIngestTextCreatesSourceAndChunks(t *testing.T) {
	mem := newFakeMemoryService()
	svc := NewService(mem, nil, Config{ChunkMaxBytes: 30})

	result, err := svc.Ingest(context.Background(), IngestRequest{
		Type:    "text",
		Title:   "Runbook",
		Content: "Alpha beta gamma delta epsilon zeta eta theta iota kappa.",
		UserID:  "user-1",
		OrgID:   "org-1",
	})
	if err != nil {
		t.Fatalf("ingest: %v", err)
	}
	if result.SourceID == "" {
		t.Fatal("expected source id")
	}
	if result.ChunksCreated < 2 {
		t.Fatalf("expected chunking, got %d chunks", result.ChunksCreated)
	}
	if len(result.MemoryIDs) != result.ChunksCreated+1 {
		t.Fatalf("memory id count mismatch: %d", len(result.MemoryIDs))
	}
	sourceMem := mem.memories[result.SourceID]
	if sourceMem == nil || sourceMem.Category != CategorySource {
		t.Fatalf("missing source memory: %#v", sourceMem)
	}
	chunkIDs, ok := sourceMem.Metadata["chunk_memory_ids"].([]string)
	if !ok || len(chunkIDs) != result.ChunksCreated {
		t.Fatalf("source metadata missing chunk ids: %#v", sourceMem.Metadata["chunk_memory_ids"])
	}
	for _, id := range chunkIDs {
		chunk := mem.memories[id]
		if chunk == nil || chunk.Category != CategorySourceChunk {
			t.Fatalf("missing source chunk %s: %#v", id, chunk)
		}
		if chunk.Metadata["source_id"] != result.SourceID {
			t.Fatalf("chunk source id mismatch: %#v", chunk.Metadata)
		}
	}
}

func TestUploadStoresBlobAndCreatesSource(t *testing.T) {
	mem := newFakeMemoryService()
	blob := &fakeBlobStore{}
	svc := NewService(mem, blob, Config{ChunkMaxBytes: 1024})

	result, err := svc.Upload(context.Background(), UploadRequest{
		Filename:    "notes.txt",
		ContentType: "text/plain",
		Reader:      bytes.NewBufferString("Project notes for source ingestion."),
		UserID:      "user-1",
	})
	if err != nil {
		t.Fatalf("upload: %v", err)
	}
	if result.R2Key == "" {
		t.Fatal("expected upload key")
	}
	if len(blob.puts) != 1 {
		t.Fatalf("expected one blob write, got %d", len(blob.puts))
	}
	if mem.memories[result.SourceID].Metadata["filename"] != "notes.txt" {
		t.Fatalf("filename metadata missing: %#v", mem.memories[result.SourceID].Metadata)
	}
}

func TestDeleteRemovesSourceChunksAndBlob(t *testing.T) {
	mem := newFakeMemoryService()
	blob := &fakeBlobStore{}
	svc := NewService(mem, blob, Config{ChunkMaxBytes: 20})

	result, err := svc.Upload(context.Background(), UploadRequest{
		Filename:    "notes.txt",
		ContentType: "text/plain",
		Reader:      bytes.NewBufferString("Alpha beta gamma delta epsilon."),
		UserID:      "user-1",
	})
	if err != nil {
		t.Fatalf("upload: %v", err)
	}
	if err := svc.Delete(context.Background(), result.SourceID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, ok := mem.memories[result.SourceID]; ok {
		t.Fatal("source memory was not deleted")
	}
	for _, id := range result.Source.ChunkMemoryIDs {
		if _, ok := mem.memories[id]; ok {
			t.Fatalf("chunk memory %s was not deleted", id)
		}
	}
	if len(blob.deletes) != 1 || blob.deletes[0] != result.R2Key {
		t.Fatalf("blob delete mismatch: %#v", blob.deletes)
	}
}
