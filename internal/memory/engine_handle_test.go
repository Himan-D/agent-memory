package memory

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"

	"agent-memory/internal/memory/types"
)

// fakeVS satisfies the VectorStore interface defined in store.go so that the
// engine handle can type-assert factory results to VectorStore.
type fakeVS struct {
	id     int
	closed bool
}

func (f *fakeVS) StoreEmbedding(ctx context.Context, text, id string, embedding []float32, metadata map[string]interface{}) (string, error) {
	return id, nil
}
func (f *fakeVS) BatchStoreEmbeddings(ctx context.Context, items []types.BatchEmbeddingItem) error {
	return nil
}
func (f *fakeVS) Search(ctx context.Context, query []float32, limit int, threshold float32, filters map[string]interface{}) ([]types.MemoryResult, error) {
	return nil, nil
}
func (f *fakeVS) UpdateMemory(ctx context.Context, id, text string, metadata map[string]interface{}) error {
	return nil
}
func (f *fakeVS) DeleteMemory(ctx context.Context, id string) error {
	return nil
}
func (f *fakeVS) UpdateVector(ctx context.Context, id string, embedding []float32) error {
	return nil
}
func (f *fakeVS) Ping(ctx context.Context) error { return nil }
func (f *fakeVS) Close() error                  { f.closed = true; return nil }

// fakeGS is a minimal type used as the factory return value for
// GraphStoreHandle. It does not need to satisfy GraphStore because the handle
// only type-asserts to that interface, and our tests do not invoke any
// GraphStore methods on the result.
type fakeGS struct {
	id int
}

func TestVectorStoreHandle_ResolvesFreshEachCall(t *testing.T) {
	var calls int32
	h := NewVectorStoreHandle("test", func() (any, error) {
		n := atomic.AddInt32(&calls, 1)
		return &fakeVS{id: int(n)}, nil
	})

	a, err := h.Get()
	if err != nil {
		t.Fatal(err)
	}
	b, err := h.Get()
	if err != nil {
		t.Fatal(err)
	}
	va, _ := a.(*fakeVS)
	vb, _ := b.(*fakeVS)
	if va == nil || vb == nil {
		t.Fatal("factory returned non-fakeVS value")
	}
	if va.id == vb.id {
		t.Fatalf("expected distinct instances, got id=%d both times", va.id)
	}
	if calls != 2 {
		t.Fatalf("expected factory called twice, got %d", calls)
	}
}

func TestVectorStoreHandle_FactoryError(t *testing.T) {
	want := errors.New("factory down")
	h := NewVectorStoreHandle("bad", func() (any, error) { return nil, want })
	if _, err := h.Get(); !errors.Is(err, want) {
		t.Fatalf("expected wrapped factory error, got %v", err)
	}
}

func TestVectorStoreHandle_TypeMismatch(t *testing.T) {
	h := NewVectorStoreHandle("wrong", func() (any, error) { return "not a store", nil })
	if _, err := h.Get(); err == nil {
		t.Fatal("expected type-mismatch error")
	}
}

func TestGraphStoreHandle_ResolvesFreshEachCall(t *testing.T) {
	var calls int32
	h := NewGraphStoreHandle("graph", func() (any, error) {
		atomic.AddInt32(&calls, 1)
		return &fakeGS{id: 1}, nil
	})
	if _, err := h.Get(); err != nil {
		t.Fatal(err)
	}
	if _, err := h.Get(); err != nil {
		t.Fatal(err)
	}
	if calls != 2 {
		t.Fatalf("expected 2 factory calls, got %d", calls)
	}
}

type hybridVS struct {
	fakeVS
	written int
}

func (h *hybridVS) AddNodesWithVectors(ctx context.Context, nodes any) error {
	h.written++
	return nil
}
func (h *hybridVS) AddEdgesWithVectors(ctx context.Context, edges any) error {
	h.written++
	return nil
}

func TestHybridWriteSupported(t *testing.T) {
	if HybridWriteSupported(&hybridVS{}) != true {
		t.Fatal("expected hybrid support")
	}
	if HybridWriteSupported(&fakeVS{}) != false {
		t.Fatal("expected no hybrid support")
	}
	if HybridWriteSupported(nil) != false {
		t.Fatal("nil should not be hybrid-capable")
	}
}

func TestEngineRegistry_RoundTrip(t *testing.T) {
	name := "test-registry-roundtrip"
	RegisterVector(name, NewVectorStoreHandle(name, func() (any, error) {
		return &fakeVS{}, nil
	}))
	got, ok := LookupVector(name)
	if !ok {
		t.Fatalf("expected handle under %q", name)
	}
	if _, err := got.Get(); err != nil {
		t.Fatal(err)
	}
}
