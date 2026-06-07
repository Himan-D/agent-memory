package chunking

import (
	"sort"
	"strings"

	"agent-memory/internal/memory/types"
)

// Merger groups chunked search results by their parent memory and returns
// one merged result per parent with the best score across its chunks.
type Merger struct {
	topChunksPerParent int
}

func NewMerger(topChunksPerParent int) *Merger {
	if topChunksPerParent <= 0 {
		topChunksPerParent = 3
	}
	return &Merger{topChunksPerParent: topChunksPerParent}
}

// MergeResults groups MemoryResults by ParentMemoryID.
// Results without a ParentMemoryID pass through unchanged.
// For each parent: keeps top N chunks by score and merges their text.
func (m *Merger) MergeResults(results []types.MemoryResult) []types.MemoryResult {
	// Separate chunks from standalone memories
	byParent := make(map[string][]types.MemoryResult)
	var standalone []types.MemoryResult

	for _, r := range results {
		parentID := ""
		if r.Metadata != nil {
			parentID = r.Metadata.ParentMemoryID
		}
		if parentID != "" {
			byParent[parentID] = append(byParent[parentID], r)
		} else {
			standalone = append(standalone, r)
		}
	}

	merged := make([]types.MemoryResult, 0, len(standalone)+len(byParent))
	merged = append(merged, standalone...)

	for parentID, chunks := range byParent {
		// Sort by score descending
		sort.Slice(chunks, func(i, j int) bool {
			return chunks[i].Score > chunks[j].Score
		})

		// Keep top N
		top := chunks
		if len(top) > m.topChunksPerParent {
			top = top[:m.topChunksPerParent]
		}

		// Build merged text from top chunks in original order (by chunk_index metadata)
		sort.Slice(top, func(i, j int) bool {
			idxI := chunkIndex(top[i])
			idxJ := chunkIndex(top[j])
			return idxI < idxJ
		})

		var texts []string
		for _, c := range top {
			if c.Text != "" {
				texts = append(texts, c.Text)
			}
		}

		merged = append(merged, types.MemoryResult{
			MemoryID: parentID,
			Score:    chunks[0].Score, // best score (already sorted desc)
			Text:     strings.Join(texts, " "),
			Source:   "chunked",
			Metadata: top[0].Metadata,
		})
	}

	// Re-sort everything by score
	sort.Slice(merged, func(i, j int) bool {
		return merged[i].Score > merged[j].Score
	})

	return merged
}

func chunkIndex(r types.MemoryResult) int {
	if r.Metadata == nil {
		return 0
	}
	if idx, ok := r.Metadata.Metadata["chunk_index"].(int); ok {
		return idx
	}
	if idx, ok := r.Metadata.Metadata["chunk_index"].(float64); ok {
		return int(idx)
	}
	return 0
}
