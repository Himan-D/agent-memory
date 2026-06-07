package memory

import (
	"context"
	"log"
	"strings"
	"time"

	"agent-memory/internal/compression/pipeline"
	"agent-memory/internal/memory/tier"
	"agent-memory/internal/memory/types"
	"agent-memory/internal/retrieval"
)

func (s *Service) SetCompressionPipeline(p *pipeline.CompressionPipeline) {
	s.compressionPipeline = p
}

func (s *Service) initMultiSignal() {
	if s.config == nil || !s.config.Memory.MultiSignalEnabled {
		return
	}
	s.multiSignalAdapter = retrieval.NewServiceAdapter(s)
	s.multiSignal = retrieval.NewMultiSignalRetrieval(s.multiSignalAdapter, retrieval.DefaultRetrievalConfig())
}

func (s *Service) routeMemoryTier(ctx context.Context, mem *types.Memory) {
	if s.tierRouter == nil || mem == nil {
		return
	}
	tierLevel, err := s.tierRouter.DetermineTier(ctx, mem)
	if err != nil {
		log.Printf("service: determine tier: %v", err)
		return
	}
	mem.Tier = string(tierLevel)
}

func (s *Service) scheduleCompression(memoryID, content string) {
	if s.compressionPipeline == nil || memoryID == "" || content == "" {
		return
	}

	job := pipeline.CompressionJob{
		MemoryID: memoryID,
		Content:  content,
		Priority: 2,
		Done:     make(chan pipeline.Result, 1),
	}

	s.compressionPipeline.CompressAsync(job)

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		select {
		case result := <-job.Done:
			if result.Error != nil {
				log.Printf("service: compression failed for %s: %v", memoryID, result.Error)
				return
			}
			if result.Compressed == "" || result.Compressed == content {
				return
			}
			if err := s.applyCompression(context.Background(), memoryID, result.Compressed, result.TokenReduction); err != nil {
				log.Printf("service: apply compression for %s: %v", memoryID, err)
			}
		case <-time.After(5 * time.Minute):
			log.Printf("service: compression timeout for memory %s", memoryID)
		}
	}()
}

func (s *Service) applyCompression(ctx context.Context, id, compressed string, ratio float64) error {
	if s.graph == nil {
		return nil
	}
	mem, err := s.graph.GetMemory(id)
	if err != nil || mem == nil {
		return err
	}
	mem.Compressed = compressed
	mem.CompressionRatio = ratio
	if mem.Metadata == nil {
		mem.Metadata = make(map[string]interface{})
	}
	mem.Metadata["compressed"] = true
	mem.Metadata["compression_ratio"] = ratio
	return s.graph.UpdateMemory(mem)
}

func (s *Service) indexMemoryForSearch(mem *types.Memory) {
	if s.multiSignalAdapter == nil || mem == nil || mem.Content == "" {
		return
	}
	s.multiSignalAdapter.AppendDocumentWithID(mem.ID, mem.Content)
}

func (s *Service) ensureBM25Index(ctx context.Context) {
	if s.multiSignalAdapter == nil || s.graph == nil {
		return
	}
	s.bm25Once.Do(func() {
		memories, err := s.graph.GetAllMemories()
		if err != nil {
			log.Printf("service: bm25 index build failed: %v", err)
			return
		}
		s.multiSignalAdapter.UpdateMemoryDocuments(memories)
	})
}

func (s *Service) searchWithMultiSignal(ctx context.Context, query string, limit int) ([]types.MemoryResult, error) {
	if s.multiSignal == nil {
		return nil, nil
	}
	s.ensureBM25Index(ctx)
	if limit <= 0 {
		limit = 10
	}
	cfg := s.multiSignal.GetConfig()
	if cfg != nil && cfg.TopK != limit {
		updated := *cfg
		updated.TopK = limit
		s.multiSignal.SetConfig(&updated)
	}
	return s.multiSignal.Retrieve(ctx, query)
}

func (s *Service) syncTierPolicyToRouter() {
	if s.tierRouter == nil {
		return
	}
	s.configMu.RLock()
	policy := s.tierPolicy
	s.configMu.RUnlock()
	s.tierRouter.SetTierPolicy(tier.TierPolicy(policy))
}

// SummarizeMemories builds a lightweight text summary from recent memories.
func (s *Service) SummarizeMemories(ctx context.Context, userID, orgID string, maxItems int) (string, error) {
	if s.graph == nil {
		return "", nil
	}
	if maxItems <= 0 {
		maxItems = 10
	}

	var memories []*types.Memory
	var err error
	switch {
	case userID != "":
		memories, err = s.graph.GetMemoriesByUser(userID)
	case orgID != "":
		memories, err = s.graph.GetMemoriesByOrg(orgID)
	default:
		memories, err = s.graph.GetAllMemories()
	}
	if err != nil {
		return "", err
	}
	if len(memories) == 0 {
		return "No memories found.", nil
	}

	if len(memories) > maxItems {
		memories = memories[:maxItems]
	}

	var parts []string
	for _, m := range memories {
		if m == nil || m.Content == "" {
			continue
		}
		content := m.Content
		if len(content) > 200 {
			content = content[:200] + "..."
		}
		parts = append(parts, "- "+content)
	}
	if len(parts) == 0 {
		return "No memory content available.", nil
	}
	return strings.Join(parts, "\n"), nil
}
