package provenance

import (
	"sync"
	"time"
)

// DAG tracks which memories were used to create new memories.
// It enables credit assignment — when a downstream memory succeeds, credit flows upstream.
// The graph is directed and acyclic: edges flow from source memories to derived memories.
type DAG struct {
	mu       sync.RWMutex
	edges    map[string][]Edge // keyed by fromID
	reverse  map[string][]Edge // keyed by toID (for ancestor lookups)
	allEdges map[string]Edge   // keyed by "fromID->toID"
}

// Edge represents a provenance link between two memories.
type Edge struct {
	FromID    string // source memory
	ToID      string // derived memory
	CreatedAt time.Time
	Depth     int // hop count from root
}

// NewDAG creates an empty provenance DAG.
func NewDAG() *DAG {
	return &DAG{
		edges:    make(map[string][]Edge),
		reverse:  make(map[string][]Edge),
		allEdges: make(map[string]Edge),
	}
}

// AddEdge creates a provenance edge from fromID to toID.
// Returns the created edge. If the edge already exists, returns the existing one.
func (d *DAG) AddEdge(fromID, toID string) *Edge {
	d.mu.Lock()
	defer d.mu.Unlock()

	key := fromID + "->" + toID
	if existing, ok := d.allEdges[key]; ok {
		return &existing
	}

	// Compute depth: max depth of any ancestor of fromID + 1
	depth := 0
	if ancestors := d.getAncestorsLocked(fromID, 100); len(ancestors) > 0 {
		// Find the edge with maximum depth pointing to fromID
		for _, e := range d.reverse[fromID] {
			if e.Depth >= depth {
				depth = e.Depth + 1
			}
		}
	}

	edge := Edge{
		FromID:    fromID,
		ToID:      toID,
		CreatedAt: time.Now(),
		Depth:     depth,
	}

	d.edges[fromID] = append(d.edges[fromID], edge)
	d.reverse[toID] = append(d.reverse[toID], edge)
	d.allEdges[key] = edge

	return &edge
}

// GetAncestors returns all ancestor memory IDs up to maxDepth hops upstream.
// The result is ordered nearest-first (direct parents before grandparents).
func (d *DAG) GetAncestors(memoryID string, maxDepth int) []string {
	d.mu.RLock()
	defer d.mu.RUnlock()

	return d.getAncestorsLocked(memoryID, maxDepth)
}

// getAncestorsLocked performs BFS upstream without holding a lock (caller must hold at least RLock).
func (d *DAG) getAncestorsLocked(memoryID string, maxDepth int) []string {
	if maxDepth <= 0 {
		return nil
	}

	visited := make(map[string]bool)
	var result []string

	// BFS queue: (nodeID, currentDepth)
	type item struct {
		id    string
		depth int
	}
	queue := []item{{id: memoryID, depth: 0}}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		if current.depth >= maxDepth {
			continue
		}

		// Look at edges pointing TO current node (reverse edges)
		for _, edge := range d.reverse[current.id] {
			if !visited[edge.FromID] {
				visited[edge.FromID] = true
				result = append(result, edge.FromID)
				queue = append(queue, item{id: edge.FromID, depth: current.depth + 1})
			}
		}
	}

	return result
}

// GetDescendants returns all descendant memory IDs up to maxDepth hops downstream.
// The result is ordered nearest-first (direct children before grandchildren).
func (d *DAG) GetDescendants(memoryID string, maxDepth int) []string {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if maxDepth <= 0 {
		return nil
	}

	visited := make(map[string]bool)
	var result []string

	type item struct {
		id    string
		depth int
	}
	queue := []item{{id: memoryID, depth: 0}}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		if current.depth >= maxDepth {
			continue
		}

		// Look at edges going FROM current node (forward edges)
		for _, edge := range d.edges[current.id] {
			if !visited[edge.ToID] {
				visited[edge.ToID] = true
				result = append(result, edge.ToID)
				queue = append(queue, item{id: edge.ToID, depth: current.depth + 1})
			}
		}
	}

	return result
}
