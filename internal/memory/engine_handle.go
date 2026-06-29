package memory

import (
	"context"
	"sync"
)

// EngineFactory creates an engine on demand. It is the indirection used by
// handle types so a fresh engine is returned after config reloads or cache
// invalidations. Modeled after Cognee's _VectorEngineHandle.
type EngineFactory func() (any, error)

// VectorStore is the subset of the vector-store interface used by the
// handle. The concrete type satisfies it transparently; the handle simply
// defers resolution so a stale reference can be replaced without touching
// callers.
type VectorStoreHandle interface {
	Get() (VectorStore, error)
	Name() string
}

// GraphStoreHandle is the graph-store analog.
type GraphStoreHandle interface {
	Get() (GraphStore, error)
	Name() string
}

// _VectorEngineHandle is a stable reference that survives cache
// invalidation. Each method call re-resolves the underlying engine via the
// factory, so a config reload transparently picks up the new engine.
type _VectorEngineHandle struct {
	name    string
	factory EngineFactory
}

// NewVectorStoreHandle wraps a factory in a handle. The factory is invoked
// on every Get so the returned engine always reflects current configuration.
func NewVectorStoreHandle(name string, factory EngineFactory) VectorStoreHandle {
	return &_VectorEngineHandle{name: name, factory: factory}
}

func (h *_VectorEngineHandle) Name() string { return h.name }

// Get returns the current vector store or an error if the factory fails.
func (h *_VectorEngineHandle) Get() (VectorStore, error) {
	v, err := h.factory()
	if err != nil {
		return nil, err
	}
	vs, ok := v.(VectorStore)
	if !ok {
		return nil, ErrEngineTypeMismatch
	}
	return vs, nil
}

// _GraphEngineHandle is the graph-store analog of _VectorEngineHandle.
type _GraphEngineHandle struct {
	name    string
	factory EngineFactory
}

// NewGraphStoreHandle wraps a factory in a handle.
func NewGraphStoreHandle(name string, factory EngineFactory) GraphStoreHandle {
	return &_GraphEngineHandle{name: name, factory: factory}
}

func (h *_GraphEngineHandle) Name() string { return h.name }

// Get returns the current graph store or an error if the factory fails.
func (h *_GraphEngineHandle) Get() (GraphStore, error) {
	v, err := h.factory()
	if err != nil {
		return nil, err
	}
	gs, ok := v.(GraphStore)
	if !ok {
		return nil, ErrEngineTypeMismatch
	}
	return gs, nil
}

// ErrEngineTypeMismatch is returned when a handle's factory returns a value
// that does not satisfy the expected engine interface.
var ErrEngineTypeMismatch = errEngineTypeMismatch("engine factory returned wrong type")

type errEngineTypeMismatch string

func (e errEngineTypeMismatch) Error() string { return string(e) }

// HybridWriteCapable is implemented by engines that can write nodes and
// their vectors in a single operation. Hystersis uses this capability to
// avoid the race window between graph and vector writes.
type HybridWriteCapable interface {
	AddNodesWithVectors(ctx context.Context, nodes any) error
	AddEdgesWithVectors(ctx context.Context, edges any) error
}

// HybridWriteSupported reports whether the engine implements
// HybridWriteCapable. Use it to decide between the hybrid write path and
// the separate-graph-then-vector path.
func HybridWriteSupported(engine any) bool {
	if engine == nil {
		return false
	}
	_, ok := engine.(HybridWriteCapable)
	return ok
}

// engineRegistry keeps a process-wide map of named engine handles so callers
// can look up "graph" or "vector" without importing the factory. It is
// intentionally tiny and additive; it does not change how engines are
// constructed elsewhere in the codebase.
type engineRegistry struct {
	mu     sync.RWMutex
	vector map[string]VectorStoreHandle
	graph  map[string]GraphStoreHandle
}

var defaultEngineRegistry = &engineRegistry{
	vector: make(map[string]VectorStoreHandle),
	graph:  make(map[string]GraphStoreHandle),
}

// RegisterVector adds a vector store handle under the given name.
func RegisterVector(name string, h VectorStoreHandle) {
	defaultEngineRegistry.mu.Lock()
	defer defaultEngineRegistry.mu.Unlock()
	defaultEngineRegistry.vector[name] = h
}

// RegisterGraph adds a graph store handle under the given name.
func RegisterGraph(name string, h GraphStoreHandle) {
	defaultEngineRegistry.mu.Lock()
	defer defaultEngineRegistry.mu.Unlock()
	defaultEngineRegistry.graph[name] = h
}

// LookupVector returns the vector store handle registered under name.
func LookupVector(name string) (VectorStoreHandle, bool) {
	defaultEngineRegistry.mu.RLock()
	defer defaultEngineRegistry.mu.RUnlock()
	h, ok := defaultEngineRegistry.vector[name]
	return h, ok
}

// LookupGraph returns the graph store handle registered under name.
func LookupGraph(name string) (GraphStoreHandle, bool) {
	defaultEngineRegistry.mu.RLock()
	defer defaultEngineRegistry.mu.RUnlock()
	h, ok := defaultEngineRegistry.graph[name]
	return h, ok
}
