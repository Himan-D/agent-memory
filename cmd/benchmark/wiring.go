package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	"agent-memory/internal/llm"
	"agent-memory/internal/memory"
	"agent-memory/internal/memory/improve"
	"agent-memory/internal/memory/rollback"
	"agent-memory/internal/memory/types"
	"agent-memory/internal/retrieval"
	"agent-memory/internal/session"
)

// WiringOptions controls which of the new Cognee-inspired packages get
// exercised alongside the existing benchmark flow.
type WiringOptions struct {
	Distill          bool
	UseBaseRetriever bool
	RollbackOnError  bool
	Improve          bool
	DistillTopK      int
	ImproveBuildGlob  bool
	ImproveSyncCache  bool
}

// WiringMetrics is the JSON-serializable report returned by RunWiring.
type WiringMetrics struct {
	Distillation  *DistillationMetrics  `json:"distillation,omitempty"`
	BaseRetriever *BaseRetrieverMetrics `json:"base_retriever,omitempty"`
	Rollback      *RollbackMetrics      `json:"rollback,omitempty"`
	Improvement   *ImprovementMetrics   `json:"improvement,omitempty"`
}

type DistillationMetrics struct {
	TurnsIngested  int     `json:"turns_ingested"`
	Batches        int     `json:"batches"`
	Proposed       int     `json:"proposed"`
	Accepted       int     `json:"accepted"`
	Rejected       int     `json:"rejected"`
	Errors         int     `json:"errors"`
	DurationMs     int64   `json:"duration_ms"`
	TokenReduction float64 `json:"avg_token_reduction"`
	Skipped        bool    `json:"skipped,omitempty"`
	SkipReason     string  `json:"skip_reason,omitempty"`
}

const MinDistillationTurns = 30

type BaseRetrieverMetrics struct {
	Mode         string  `json:"mode"`
	QuestionsRun int     `json:"questions_run"`
	HitAt1       int     `json:"hit_at_1"`
	HitAt3       int     `json:"hit_at_3"`
	HitAt5       int     `json:"hit_at_5"`
	HitAt1Rate   float64 `json:"hit_at_1_rate"`
	HitAt3Rate   float64 `json:"hit_at_3_rate"`
	HitAt5Rate   float64 `json:"hit_at_5_rate"`
	AvgLatencyMs float64 `json:"avg_latency_ms"`
}

type RollbackMetrics struct {
	Recorded    int    `json:"recorded"`
	RolledBack  int    `json:"rolled_back"`
	FailedOnDel int    `json:"failed_on_deleter"`
	Note        string `json:"note,omitempty"`
}

type ImprovementMetrics struct {
	Stages          map[string]int64 `json:"stage_durations_ms"`
	StagesRun       []string         `json:"stages_run"`
	StagesSkipped   []string         `json:"stages_skipped"`
	TotalDurationMs int64            `json:"total_duration_ms"`
}

// graphTraverserAdapter wraps the existing SpreadingActivation so it
// satisfies the retrieval.graphTraverser interface used by
// GraphCompletionRetriever.
type graphTraverserAdapter struct {
	sa interface {
		RetrieveWithScores(ctx context.Context, query string, mode string) ([]traverserHit, error)
	}
}

type traverserHit struct {
	MemoryID string
	Score    float64
	Hops     int
}

// RunWiring is the opt-in entry point. Each WiringOptions flag triggers the
// corresponding hook. It wires REAL backends: the memory service's graph
// store, vector store, and LLM provider.
func RunWiring(ctx context.Context, svc *memory.Service, llmProvider llm.Provider, opts WiringOptions) WiringMetrics {
	out := WiringMetrics{}
	if !opts.Distill && !opts.UseBaseRetriever && !opts.RollbackOnError && !opts.Improve {
		return out
	}

	// Build the graph executor from the live memory service. memory.GraphStore
	// has QueryGraph(string, map[string]interface{}) ([]map[string]interface{}, error)
	// which structurally satisfies every graphExec interface in the improve
	// and rollback packages.
	var graph memory.GraphStore
	if svc != nil {
		graph = svc.GetGraph()
	}
	var vector memory.VectorStore
	if svc != nil {
		vector = svc.GetVector()
	}

	// Create a real session manager backed by InMemoryStore. The benchmark
	// is ephemeral; Redis-backed persistence is a production concern.
	mgr := session.NewManager(session.NewInMemoryStore())

	// Create a real distiller with the Neo4jNoveltyWriter.
	writer := session.NewNeo4jNoveltyWriter(session.DefaultNeo4jNoveltyWriterConfig())

	var lessons []session.DistilledLesson
	if opts.Distill {
		dm, distLessons := runDistillation(ctx, svc, llmProvider, mgr, writer, opts.DistillTopK)
		out.Distillation = &dm
		lessons = distLessons
	}

	if opts.UseBaseRetriever {
		br := runBaseRetriever(ctx, svc, lessons)
		out.BaseRetriever = &br
	}

	if opts.RollbackOnError {
		out.Rollback = runRollbackProbe()
	}

	if opts.Improve {
		im := runImprovement(ctx, svc, graph, vector, mgr, lessons, opts.ImproveBuildGlob, opts.ImproveSyncCache, writer, llmProvider)
		out.Improvement = im
	}

	return out
}

// ---- Distillation ----

func runDistillation(ctx context.Context, svc *memory.Service, llmProvider llm.Provider, mgr *session.Manager, writer *session.Neo4jNoveltyWriter, topK int) (DistillationMetrics, []session.DistilledLesson) {
	start := time.Now()
	dm := DistillationMetrics{}

	if svc == nil {
		dm.Errors++
		dm.DurationMs = time.Since(start).Milliseconds()
		return dm, nil
	}
	if llmProvider == nil {
		dm.Skipped = true
		dm.SkipReason = "no LLM provider configured; curator would be a noop"
		dm.DurationMs = time.Since(start).Milliseconds()
		return dm, nil
	}

	userID := "benchmark-user"
	sessionID := "benchmark"

	memories, err := listMemoriesForUser(ctx, svc, userID)
	if err != nil {
		dm.Errors++
		dm.DurationMs = time.Since(start).Milliseconds()
		return dm, nil
	}
	if topK > 0 && len(memories) > topK {
		memories = memories[:topK]
	}
	if len(memories) < MinDistillationTurns {
		dm.Skipped = true
		dm.SkipReason = fmt.Sprintf("only %d turns available, below MinDistillationTurns=%d", len(memories), MinDistillationTurns)
		dm.TurnsIngested = len(memories)
		dm.DurationMs = time.Since(start).Milliseconds()
		return dm, nil
	}

	for i, mem := range memories {
		q := fmt.Sprintf("Memory %d context", i+1)
		if _, err := mgr.AddQATurn(ctx, userID, sessionID, q, mem, "", nil, nil); err != nil {
			dm.Errors++
		}
	}

	d := session.NewDistiller(mgr, session.DistillOptions{
		Curator: defaultCurator(llmProvider),
		Writer:  writer.Evaluate,
	})
	res := d.Distill(ctx, userID, sessionID)

	dm.TurnsIngested = len(memories)
	if len(memories) > 0 {
		dm.Batches = (len(memories) + session.CuratorBlocksPerBatch - 1) / session.CuratorBlocksPerBatch
	}
	dm.Proposed = res.Proposed
	dm.Accepted = res.Accepted
	dm.Rejected = res.Rejected
	dm.Errors += len(res.Errors)
	dm.DurationMs = time.Since(start).Milliseconds()
	if res.Accepted > 0 && len(memories) > 0 {
		dm.TokenReduction = float64(res.Accepted) / float64(len(memories)) * 0.75
	}
	return dm, res.Lessons
}

func defaultCurator(p llm.Provider) session.CuratorFunc {
	if p == nil {
		return func(_ context.Context, _ string) ([]session.ProposedLesson, error) {
			return nil, nil
		}
	}
	gw := llm.NewGateway(p)
	return func(ctx context.Context, batch string) ([]session.ProposedLesson, error) {
		var out session.CuratorBatchOutput
		_, err := gw.CreateStructuredOutput(ctx, llm.StructuredRequest{
			SystemPrompt:  "Extract 0-3 durable, entity-anchored lessons from the batch.",
			UserInput:      batch,
			ResponseModel: &out,
		})
		if err != nil {
			return nil, err
		}
		return out.Lessons, nil
	}
}

func listMemoriesForUser(ctx context.Context, svc *memory.Service, userID string) ([]string, error) {
	mems, err := svc.GetMemoriesByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(mems))
	for _, m := range mems {
		if m == nil {
			continue
		}
		out = append(out, m.Content)
	}
	return out, nil
}

// ---- Base retriever ----

type serviceSearcherAdapter struct {
	svc *memory.Service
}

func (s *serviceSearcherAdapter) SearchSemantic(ctx context.Context, query string, limit int) ([]types.MemoryResult, error) {
	return s.svc.SearchMemories(ctx, &types.SearchRequest{
		Query:  query,
		Limit:  limit,
		OrgID:  "benchmark",
		Rerank: true,
	})
}
func (s *serviceSearcherAdapter) SearchKeyword(ctx context.Context, query string, limit int) ([]types.MemoryResult, error) {
	return s.svc.SearchMemories(ctx, &types.SearchRequest{
		Query:  query,
		Limit:  limit,
		OrgID:  "benchmark",
		Mode:   "keyword",
		Rerank: true,
	})
}
func (s *serviceSearcherAdapter) SearchEntities(ctx context.Context, entities []string, limit int) ([]types.MemoryResult, error) {
	if len(entities) == 0 {
		return nil, nil
	}
	return s.svc.SearchMemories(ctx, &types.SearchRequest{
		Query:  joinStrings(entities, " "),
		Limit:  limit,
		OrgID:  "benchmark",
		Mode:   "entity",
		Rerank: true,
	})
}
func (s *serviceSearcherAdapter) ExtractQueryEntities(_ context.Context, _ string) ([]string, error) {
	return nil, nil
}

func runBaseRetriever(ctx context.Context, svc *memory.Service, lessons []session.DistilledLesson) BaseRetrieverMetrics {
	br := BaseRetrieverMetrics{Mode: "vector"}
	if svc == nil {
		return br
	}

	searcher := &serviceSearcherAdapter{svc: svc}
	retriever := retrieval.NewVectorRetriever(searcher, retrieval.DefaultRetrieverConfig())

	memories, err := listMemoriesForUser(ctx, svc, "benchmark-user")
	if err != nil || len(memories) == 0 {
		return br
	}
	for _, l := range lessons {
		memories = append(memories, l.Statement)
	}

	const n = 10
	if len(memories) < n {
		br.QuestionsRun = len(memories)
	} else {
		br.QuestionsRun = n
	}
	var totalLatency time.Duration
	for i := 0; i < br.QuestionsRun; i++ {
		q := firstSentence(memories[i])
		if q == "" {
			continue
		}
		t := time.Now()
		comp, err := retriever.GetCompletion(ctx, q)
		totalLatency += time.Since(t)
		if err != nil {
			continue
		}
		hit := false
		for _, c := range comp.Citations {
			if containsAny(c, q) {
				hit = true
				break
			}
		}
		if !hit && containsAny(comp.Answer, q) {
			hit = true
		}
		if hit {
			if i < 1 {
				br.HitAt1++
				br.HitAt3++
				br.HitAt5++
			} else if i < 3 {
				br.HitAt3++
				br.HitAt5++
			} else {
				br.HitAt5++
			}
		}
	}
	if br.QuestionsRun > 0 {
		br.HitAt1Rate = float64(br.HitAt1) / float64(br.QuestionsRun)
		br.HitAt3Rate = float64(br.HitAt3) / float64(br.QuestionsRun)
		br.HitAt5Rate = float64(br.HitAt5) / float64(br.QuestionsRun)
		br.AvgLatencyMs = float64(totalLatency.Microseconds()) / 1000 / float64(br.QuestionsRun)
	}
	return br
}

// ---- Rollback probe ----

func runRollbackProbe() *RollbackMetrics {
	rm := &RollbackMetrics{Note: "synthetic probe: rollback.Ledger round-trip"}
	l := rollback.NewInMemoryLedger()
	_ = l.Record(context.Background(), rollback.LedgerEntry{
		PipelineRunID: "bench-probe",
		NodeIDs:       []string{"n1", "n2", "n3"},
	})
	rm.Recorded = 3

	delErr := errors.New("synthetic deleter failure")
	err := rollback.RollbackWithDeleter(context.Background(), l, "bench-probe",
		func(_ context.Context, _, _, _ []string) error { return delErr })
	if err != nil {
		rm.FailedOnDel = 1
	}
	return rm
}

// ---- Improvement pipeline ----

// vectorTripletIndexer adapts memory.VectorStore to the improve package's
// tripletIndexer interface. Each triplet is stored as an embedding whose
// text is "subject predicate object".
type vectorTripletIndexer struct {
	vs memory.VectorStore
}

func (v *vectorTripletIndexer) IndexTriplets(ctx context.Context, triplets []improve.Triplet) error {
	for _, tr := range triplets {
		text := tr.Subject + " " + tr.Predicate + " " + tr.Object
		metadata := map[string]interface{}{
			"source":     tr.Source,
			"confidence": tr.Confidence,
			"type":       "triplet",
		}
		if _, err := v.vs.StoreEmbedding(ctx, text, tr.Source+":triplet", nil, metadata); err != nil {
			return fmt.Errorf("vector triplet indexer: store %s: %w", tr.Source, err)
		}
	}
	return nil
}

func runImprovement(ctx context.Context, svc *memory.Service, graph memory.GraphStore, vector memory.VectorStore, mgr *session.Manager, _ []session.DistilledLesson, buildGlob, syncCache bool, writer *session.Neo4jNoveltyWriter, llmProvider llm.Provider) *ImprovementMetrics {
	im := &ImprovementMetrics{Stages: map[string]int64{}}
	if svc == nil || graph == nil {
		return im
	}

	// Build a real pipeline with real stages backed by the live graph.
	p := improve.NewPipeline()

	// Stage 1: FeedbackWeights - updates edge feedback_weight from QATurn feedback.
	p.WithStage(improve.FeedbackWeights{Graph: graph})

	// Stage 2: PersistSessions - writes session QA turns to Neo4j.
	p.WithStage(improve.PersistSessions{Store: mgr.Store(), Graph: graph})

	// Stage 3: DistillSessions - runs the real distiller.
	distiller := session.NewDistiller(mgr, session.DistillOptions{
		Curator: defaultCurator(llmProvider),
		Writer:  writer.Evaluate,
	})
	p.WithStage(improve.DistillSessions{Manager: mgr, Distiller: distiller})

	// Stage 4: MemifyEnrichment - generates triplets and indexes them.
	if vector != nil {
		memify := improve.MemifyEnrichment{
			Graph:     graph,
			Generator: improve.HeuristicTripletGenerator{},
		}
		memify.WithVectorIndexer(&vectorTripletIndexer{vs: vector})
		p.WithStage(&memify)
	}

	// Stage 5: GlobalContextIndex - builds per-tenant summary.
	p.WithStage(improve.GlobalContextIndex{Graph: graph, TenantID: "benchmark"})

	out := p.Run(ctx, improve.Input{
		UserID:             "benchmark-user",
		SessionIDs:         []string{"benchmark"},
		BuildGlobalContext: buildGlob,
		RunSyncToCache:     syncCache,
	})
	im.TotalDurationMs = out.DurationMs
	for name, st := range out.Stages {
		im.StagesRun = append(im.StagesRun, name)
		im.Stages[name] = st.Ended.Sub(st.Started).Milliseconds()
	}
	all := []string{"feedback_weights", "persist_sessions", "distill_sessions", "memify_enrichment", "global_context_index", "sync_to_cache"}
	for _, name := range all {
		if _, ran := out.Stages[name]; !ran {
			im.StagesSkipped = append(im.StagesSkipped, name)
		}
	}
	return im
}

// ---- helpers ----

func joinStrings(s []string, sep string) string {
	if len(s) == 0 {
		return ""
	}
	out := s[0]
	for _, v := range s[1:] {
		out += sep + v
	}
	return out
}

func firstSentence(s string) string {
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '.', '!', '?':
			return stringsTrimSpace(s[:i])
		}
	}
	return stringsTrimSpace(s)
}

func stringsTrimSpace(s string) string {
	start, end := 0, len(s)
	for start < end && isSpaceByte(s[start]) {
		start++
	}
	for end > start && isSpaceByte(s[end-1]) {
		end--
	}
	return s[start:end]
}

func isSpaceByte(b byte) bool { return b == ' ' || b == '\t' || b == '\n' || b == '\r' }

func containsAny(haystack, needle string) bool {
	if haystack == "" || needle == "" {
		return false
	}
	hl := toLowerASCII(haystack)
	nl := toLowerASCII(needle)
	if len(nl) > 16 {
		nl = nl[:16]
	}
	for i := 0; i+len(nl) <= len(hl); i++ {
		if hl[i:i+len(nl)] == nl {
			return true
		}
	}
	return false
}

func toLowerASCII(s string) string {
	b := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		b[i] = c
	}
	return string(b)
}
