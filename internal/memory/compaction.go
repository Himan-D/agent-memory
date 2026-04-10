package memory

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"
	"sync"
	"time"

	"agent-memory/internal/memory/types"
)

type CompactionConfig struct {
	SimilarityThreshold float64
	MaxMemoryAge        time.Duration
	MinMemoryLength     int
	MaxMemoriesPerUser  int
	CompressionRatio    float64
}

type CompactionService struct {
	neo4j      Neo4jClient
	qdrant     QdrantClient
	embedder   Embedder
	config     CompactionConfig
	mu         sync.RWMutex
	lastRun    time.Time
	compacting bool
}

type Neo4jClient interface {
	GetMemoriesByUser(ctx context.Context, userID string) ([]*types.Memory, error)
	GetMemoriesByOrg(ctx context.Context, orgID string) ([]*types.Memory, error)
	GetMemory(id string) (*types.Memory, error)
	UpdateMemory(mem *types.Memory) error
	DeleteMemory(id string) error
	SearchMemories(ctx context.Context, req *types.SearchRequest) ([]types.MemoryResult, error)
	CreateMemory(mem *types.Memory) error
	RecordHistory(memoryID, action, oldValue, newValue, changedBy, reason string) error
}

type QdrantClient interface {
	Search(ctx context.Context, query []float32, limit int, threshold float32, filters map[string]interface{}) ([]types.MemoryResult, error)
	DeleteMemory(ctx context.Context, id string) error
}

type Embedder interface {
	GenerateEmbedding(text string) ([]float32, error)
	GenerateBatchEmbeddings(texts []string) ([][]float32, error)
}

type CompactionResult struct {
	MergedCount     int
	ArchivedCount   int
	DeletedCount    int
	SummarizedCount int
	TotalMemories   int
	Duration        time.Duration
}

type MemoryWithEmbedding struct {
	Memory    *types.Memory
	Embedding []float32
}

type MemoryGroup struct {
	Memories []*MemoryWithEmbedding
	Centroid []float32
	Combined string
	CanMerge bool
}

func NewCompactionService(neo4j Neo4jClient, qdrant QdrantClient, embedder Embedder, cfg *CompactionConfig) *CompactionService {
	if cfg == nil {
		cfg = &CompactionConfig{
			SimilarityThreshold: 0.92,
			MaxMemoryAge:        30 * 24 * time.Hour,
			MinMemoryLength:     100,
			MaxMemoriesPerUser:  1000,
			CompressionRatio:    0.6,
		}
	}

	return &CompactionService{
		neo4j:    neo4j,
		qdrant:   qdrant,
		embedder: embedder,
		config:   *cfg,
	}
}

func (s *CompactionService) RunCompaction(ctx context.Context, userID, orgID string) (*CompactionResult, error) {
	s.mu.Lock()
	if s.compacting {
		s.mu.Unlock()
		return nil, fmt.Errorf("compaction already in progress")
	}
	s.compacting = true
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		s.compacting = false
		s.lastRun = time.Now()
		s.mu.Unlock()
	}()

	start := time.Now()
	result := &CompactionResult{}

	var memories []*types.Memory
	var err error

	if userID != "" {
		memories, err = s.neo4j.GetMemoriesByUser(ctx, userID)
	} else if orgID != "" {
		memories, err = s.neo4j.GetMemoriesByOrg(ctx, orgID)
	} else {
		return nil, fmt.Errorf("either userID or orgID required")
	}

	if err != nil {
		return nil, fmt.Errorf("fetch memories: %w", err)
	}

	result.TotalMemories = len(memories)

	groups, err := s.groupSimilarMemories(ctx, memories)
	if err != nil {
		return nil, fmt.Errorf("group memories: %w", err)
	}

	for _, group := range groups {
		if group.CanMerge && len(group.Memories) > 1 {
			if _, err := s.mergeMemoryGroup(ctx, group); err == nil {
				result.MergedCount++
			}
		}
	}

	oldMemories := s.findOldMemories(memories)
	for _, mem := range oldMemories {
		if err := s.archiveMemory(ctx, mem); err == nil {
			result.ArchivedCount++
		}
	}

	redundant := s.findRedundantMemories(memories)
	for _, mem := range redundant {
		if err := s.deleteMemory(ctx, mem); err == nil {
			result.DeletedCount++
		}
	}

	longMemories := s.findLongMemories(memories)
	for _, mem := range longMemories {
		if summarized, err := s.summarizeMemory(ctx, mem); err == nil && summarized {
			result.SummarizedCount++
		}
	}

	result.Duration = time.Since(start)
	return result, nil
}

func (s *CompactionService) groupSimilarMemories(ctx context.Context, memories []*types.Memory) ([]*MemoryGroup, error) {
	activeMemories := make([]*types.Memory, 0)
	for _, m := range memories {
		if m.Status == types.MemoryStatusActive && m.Content != "" {
			activeMemories = append(activeMemories, m)
		}
	}

	if len(activeMemories) < 2 {
		return nil, nil
	}

	texts := make([]string, len(activeMemories))
	for i, m := range activeMemories {
		texts[i] = m.Content
	}

	embeddings, err := s.embedder.GenerateBatchEmbeddings(texts)
	if err != nil {
		return nil, err
	}

	memWithEmbed := make([]*MemoryWithEmbedding, len(activeMemories))
	for i, m := range activeMemories {
		memWithEmbed[i] = &MemoryWithEmbedding{
			Memory:    m,
			Embedding: embeddings[i],
		}
	}

	sort.Slice(memWithEmbed, func(i, j int) bool {
		return memWithEmbed[i].Memory.CreatedAt.Before(memWithEmbed[j].Memory.CreatedAt)
	})

	var groups []*MemoryGroup
	used := make(map[string]bool)

	for i, mwe := range memWithEmbed {
		if used[mwe.Memory.ID] || len(mwe.Embedding) == 0 {
			continue
		}

		group := &MemoryGroup{
			Memories: []*MemoryWithEmbedding{mwe},
			Centroid: mwe.Embedding,
		}
		used[mwe.Memory.ID] = true

		for j := i + 1; j < len(memWithEmbed); j++ {
			if used[memWithEmbed[j].Memory.ID] || len(memWithEmbed[j].Embedding) == 0 {
				continue
			}

			similarity := cosineSimilarity(mwe.Embedding, memWithEmbed[j].Embedding)
			if similarity >= s.config.SimilarityThreshold {
				group.Memories = append(group.Memories, memWithEmbed[j])
				used[memWithEmbed[j].Memory.ID] = true
				group.Centroid = addVectors(group.Centroid, memWithEmbed[j].Embedding)
			}
		}

		if len(group.Memories) > 1 {
			group.Centroid = scaleVector(group.Centroid, 1.0/float64(len(group.Memories)))
			group.Combined = s.combineMemoryContents(group.Memories)
			group.CanMerge = s.shouldMerge(group)
			groups = append(groups, group)
		}
	}

	return groups, nil
}

func (s *CompactionService) shouldMerge(group *MemoryGroup) bool {
	if len(group.Memories) < 2 {
		return false
	}

	var totalLen int
	for _, m := range group.Memories {
		totalLen += len(m.Memory.Content)
	}

	combinedLen := len(group.Combined)
	if combinedLen == 0 {
		return false
	}

	ratio := float64(combinedLen) / float64(totalLen)
	return ratio < s.config.CompressionRatio
}

func (s *CompactionService) combineMemoryContents(memories []*MemoryWithEmbedding) string {
	sort.Slice(memories, func(i, j int) bool {
		return memories[i].Memory.CreatedAt.Before(memories[j].Memory.CreatedAt)
	})

	seen := make(map[string]bool)
	var parts []string

	for _, m := range memories {
		normalized := normalizeContent(m.Memory.Content)
		if seen[normalized] {
			continue
		}
		seen[normalized] = true
		parts = append(parts, m.Memory.Content)
	}

	return strings.Join(parts, " | ")
}

func (s *CompactionService) mergeMemoryGroup(ctx context.Context, group *MemoryGroup) (*types.Memory, error) {
	primary := group.Memories[0].Memory

	merged := &types.Memory{
		ID:        primary.ID,
		UserID:    primary.UserID,
		OrgID:     primary.OrgID,
		AgentID:   primary.AgentID,
		SessionID: primary.SessionID,
		Type:      primary.Type,
		Content:   group.Combined,
		Category:  primary.Category,
		Metadata:  primary.Metadata,
		Status:    types.MemoryStatusActive,
		CreatedAt: primary.CreatedAt,
		UpdatedAt: time.Now(),
	}

	if merged.Metadata == nil {
		merged.Metadata = make(map[string]interface{})
	}
	merged.Metadata["merged_from"] = len(group.Memories)
	merged.Metadata["merged_ids"] = extractIDs(group.Memories)

	if err := s.neo4j.UpdateMemory(merged); err != nil {
		return nil, err
	}

	history := fmt.Sprintf("Merged %d memories", len(group.Memories))
	_ = s.neo4j.RecordHistory(primary.ID, string(types.HistoryActionUpdate), "", group.Combined, "compaction", history)

	for _, mwe := range group.Memories[1:] {
		_ = s.qdrant.DeleteMemory(ctx, mwe.Memory.ID)
		_ = s.neo4j.DeleteMemory(mwe.Memory.ID)
		_ = s.neo4j.RecordHistory(mwe.Memory.ID, string(types.HistoryActionDelete), mwe.Memory.Content, "", "compaction", "Merged into "+primary.ID)
	}

	return merged, nil
}

func (s *CompactionService) findOldMemories(memories []*types.Memory) []*types.Memory {
	var old []*types.Memory
	cutoff := time.Now().Add(-s.config.MaxMemoryAge)

	for _, m := range memories {
		if m.Status != types.MemoryStatusActive {
			continue
		}
		if m.Immutable {
			continue
		}
		if m.CreatedAt.Before(cutoff) {
			old = append(old, m)
		}
	}

	return old
}

func (s *CompactionService) archiveMemory(ctx context.Context, mem *types.Memory) error {
	mem.Status = types.MemoryStatusArchived
	mem.UpdatedAt = time.Now()

	if err := s.neo4j.UpdateMemory(mem); err != nil {
		return err
	}

	_ = s.qdrant.DeleteMemory(ctx, mem.ID)
	_ = s.neo4j.RecordHistory(mem.ID, string(types.HistoryActionArchive), "", "", "compaction", "Auto-archived due to age")

	return nil
}

func (s *CompactionService) findRedundantMemories(memories []*types.Memory) []*types.Memory {
	var redundant []*types.Memory
	seen := make(map[string]string)

	for _, m := range memories {
		if m.Status != types.MemoryStatusActive {
			continue
		}
		if m.Immutable {
			continue
		}

		normalized := normalizeContent(m.Content)
		if _, exists := seen[normalized]; exists {
			redundant = append(redundant, m)
			continue
		}
		seen[normalized] = m.ID
	}

	return redundant
}

func (s *CompactionService) deleteMemory(ctx context.Context, mem *types.Memory) error {
	_ = s.qdrant.DeleteMemory(ctx, mem.ID)
	if err := s.neo4j.DeleteMemory(mem.ID); err != nil {
		return err
	}
	_ = s.neo4j.RecordHistory(mem.ID, string(types.HistoryActionDelete), mem.Content, "", "compaction", "Duplicate content")
	return nil
}

func (s *CompactionService) findLongMemories(memories []*types.Memory) []*types.Memory {
	var long []*types.Memory

	for _, m := range memories {
		if m.Status != types.MemoryStatusActive {
			continue
		}
		if m.Immutable {
			continue
		}
		if len(m.Content) < s.config.MinMemoryLength {
			continue
		}
		long = append(long, m)
	}

	return long
}

func (s *CompactionService) summarizeMemory(ctx context.Context, mem *types.Memory) (bool, error) {
	if mem.Content == "" {
		return false, nil
	}

	searchReq := &types.SearchRequest{
		Query:     mem.Content,
		Limit:     5,
		Threshold: 0.85,
		UserID:    mem.UserID,
	}

	similar, err := s.neo4j.SearchMemories(ctx, searchReq)
	if err != nil {
		return false, err
	}

	if len(similar) == 0 {
		return false, nil
	}

	var summary strings.Builder
	summary.WriteString("Summary of related memories:\n")

	combined := mem.Content
	for _, r := range similar {
		if r.Metadata != nil && r.Metadata.ID != mem.ID {
			combined += " " + r.Metadata.Content
		}
	}

	if len(combined) > len(mem.Content)*2 {
		words := strings.Fields(combined)
		if len(words) > 200 {
			summary.WriteString(fmt.Sprintf("Key points from %d related memories. ", len(similar)+1))
			summary.WriteString(strings.Join(words[:200], " "))
			summary.WriteString("...")
		} else {
			summary.WriteString(combined)
		}
	} else {
		return false, nil
	}

	mem.Content = summary.String()
	mem.UpdatedAt = time.Now()

	if mem.Metadata == nil {
		mem.Metadata = make(map[string]interface{})
	}
	mem.Metadata["summarized"] = true
	mem.Metadata["original_length"] = len(mem.Content)

	if err := s.neo4j.UpdateMemory(mem); err != nil {
		return false, err
	}

	_ = s.neo4j.RecordHistory(mem.ID, string(types.HistoryActionUpdate), "", summary.String(), "compaction", "Auto-summarized")

	return true, nil
}

func cosineSimilarity(a, b []float32) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}

	var dotProd, normA, normB float64
	for i := range a {
		dotProd += float64(a[i]) * float64(b[i])
		normA += float64(a[i]) * float64(a[i])
		normB += float64(b[i]) * float64(b[i])
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return dotProd / (math.Sqrt(normA) * math.Sqrt(normB))
}

func addVectors(a, b []float32) []float32 {
	result := make([]float32, len(a))
	for i := range a {
		result[i] = a[i] + b[i]
	}
	return result
}

func scaleVector(v []float32, scale float64) []float32 {
	result := make([]float32, len(v))
	for i := range v {
		result[i] = float32(float64(v[i]) * scale)
	}
	return result
}

func normalizeContent(s string) string {
	s = strings.ToLower(s)
	s = strings.Join(strings.Fields(s), " ")
	return s
}

func extractIDs(memories []*MemoryWithEmbedding) []string {
	ids := make([]string, len(memories))
	for i, m := range memories {
		ids[i] = m.Memory.ID
	}
	return ids
}

func (s *CompactionService) IsCompacting() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.compacting
}

func (s *CompactionService) LastRun() time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.lastRun
}

func (s *CompactionService) SetConfig(cfg CompactionConfig) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.config = cfg
}
