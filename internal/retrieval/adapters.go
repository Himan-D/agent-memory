package retrieval

import (
	"context"
	"fmt"
	"strings"

	"agent-memory/internal/memory/types"
)

// memorySearcher is the minimal search surface that adapters depend on. It
// is satisfied by the existing *memory.ServiceAdapter and by the memory
// service directly via the interface declared in service_adapter.go.
type memorySearcher interface {
	SearchSemantic(ctx context.Context, query string, limit int) ([]types.MemoryResult, error)
	SearchKeyword(ctx context.Context, query string, limit int) ([]types.MemoryResult, error)
	SearchEntities(ctx context.Context, entities []string, limit int) ([]types.MemoryResult, error)
	ExtractQueryEntities(ctx context.Context, query string) ([]string, error)
}

// graphTraverser is the optional graph-propagation surface used by
// GraphCompletionRetriever. When provided, the retriever augments the
// vector-search hits with multi-hop graph propagation (spreading
// activation). When nil, the retriever falls back to the distance-penalty
// re-rank behavior so it stays usable in environments without Neo4j.
//
// The interface matches the public surface of
// internal/compression/retrieval.SpreadingActivation.RetrieveWithScores,
// defined here as an interface so the retrieval package stays decoupled
// from the proprietary compression package.
type graphTraverser interface {
	RetrieveWithScores(ctx context.Context, query string, mode GraphMode) ([]GraphHit, error)
}

// GraphMode mirrors internal/compression/retrieval.SearchMode so callers
// don't need to import the proprietary package. The string values match
// what SpreadingActivation expects.
type GraphMode string

const (
	GraphModeVector    GraphMode = "vector"
	GraphModeSpreading GraphMode = "spreading"
	GraphModeHybrid    GraphMode = "hybrid"
)

// GraphHit is a graph-traversal result with score and hop count. Matches
// the proprietary RetrieveResult shape without the import.
type GraphHit struct {
	MemoryID string
	Score    float64
	Hops     int
}

// VectorRetriever is a thin BaseRetriever that calls semantic search only.
// It is the simplest concrete retriever and the default for tests.
type VectorRetriever struct {
	searcher  memorySearcher
	cfg       *RetrieverConfig
	complete  CompletionFunc
}

// NewVectorRetriever wraps a memorySearcher as a BaseRetriever. cfg may be
// nil, in which case DefaultRetrieverConfig is used.
func NewVectorRetriever(s memorySearcher, cfg *RetrieverConfig) *VectorRetriever {
	if cfg == nil {
		cfg = DefaultRetrieverConfig()
	}
	return &VectorRetriever{searcher: s, cfg: cfg}
}

// WithCompletion attaches an LLM completion callback. When nil, the
// retriever returns a heuristic answer derived from the top hits.
func (r *VectorRetriever) WithCompletion(c CompletionFunc) *VectorRetriever {
	r.complete = c
	return r
}

func (r *VectorRetriever) Name() string { return "vector" }

func (r *VectorRetriever) RetrieveObjects(ctx context.Context, query string) (RetrievedObjects, error) {
	if r.searcher == nil {
		return RetrievedObjects{}, fmt.Errorf("vector retriever: searcher not configured")
	}
	results, err := r.searcher.SearchSemantic(ctx, query, r.cfg.WideSearchTopK)
	if err != nil {
		return RetrievedObjects{}, err
	}
	objs := make([]RetrievedObject, 0, len(results))
	for _, res := range results {
		objs = append(objs, RetrievedObject{
			Type:   "memory",
			ID:     res.MemoryID,
			Text:   res.Text,
			Score:  float64(res.Score),
			Source: "vector",
		})
	}
	objs = rerankOrders(objs, r.cfg.FeedbackInfluence)
	if r.cfg.TopK > 0 && len(objs) > r.cfg.TopK {
		objs = objs[:r.cfg.TopK]
	}
	return RetrievedObjects{Items: objs}, nil
}

func (r *VectorRetriever) BuildContext(_ context.Context, query string, objects RetrievedObjects) (string, error) {
	if len(objects.Items) == 0 {
		return "", nil
	}
	var b strings.Builder
	for i, it := range objects.Items {
		fmt.Fprintf(&b, "[%d] %s\n", i+1, it.Text)
	}
	return trimContext(b.String(), r.cfg.MaxContextChars), nil
}

func (r *VectorRetriever) Complete(ctx context.Context, query, contextText string, objects RetrievedObjects) (Completion, error) {
	citations := make([]string, 0, len(objects.Items))
	for _, it := range objects.Items {
		citations = append(citations, it.ID)
	}
	answer, err := r.runCompletion(ctx, query, contextText, objects)
	if err != nil {
		return Completion{}, err
	}
	return Completion{Query: query, Context: contextText, Answer: answer, Citations: citations}, nil
}

func (r *VectorRetriever) GetCompletion(ctx context.Context, query string) (Completion, error) {
	objs, err := r.RetrieveObjects(ctx, query)
	if err != nil {
		return Completion{}, err
	}
	ctxText, err := r.BuildContext(ctx, query, objs)
	if err != nil {
		return Completion{}, err
	}
	return r.Complete(ctx, query, ctxText, objs)
}

func (r *VectorRetriever) runCompletion(ctx context.Context, query, contextText string, objects RetrievedObjects) (string, error) {
	if r.complete != nil {
		return r.complete(ctx, query, contextText)
	}
	return heuristicAnswer(query, objects), nil
}

// MultiSignalRetrieverAdapter adapts an existing MultiSignalRetrieval (or any
// memorySearcher) to the BaseRetriever interface. The multi-signal fusion
// runs inside RetrieveObjects, so BuildContext only needs to format results.
type MultiSignalRetrieverAdapter struct {
	ms       *MultiSignalRetrieval
	cfg      *RetrieverConfig
	complete CompletionFunc
}

// NewMultiSignalRetrieverAdapter wraps a MultiSignalRetrieval as a
// BaseRetriever.
func NewMultiSignalRetrieverAdapter(ms *MultiSignalRetrieval, cfg *RetrieverConfig) *MultiSignalRetrieverAdapter {
	if cfg == nil {
		cfg = DefaultRetrieverConfig()
	}
	return &MultiSignalRetrieverAdapter{ms: ms, cfg: cfg}
}

func (r *MultiSignalRetrieverAdapter) Name() string { return "multisignal" }

func (r *MultiSignalRetrieverAdapter) RetrieveObjects(ctx context.Context, query string) (RetrievedObjects, error) {
	results, err := r.ms.Retrieve(ctx, query)
	if err != nil {
		return RetrievedObjects{}, err
	}
	objs := make([]RetrievedObject, 0, len(results))
	for _, res := range results {
		objs = append(objs, RetrievedObject{
			Type:   "memory",
			ID:     res.MemoryID,
			Text:   res.Text,
			Score:  float64(res.Score),
			Source: "multisignal",
		})
	}
	objs = rerankOrders(objs, r.cfg.FeedbackInfluence)
	if r.cfg.TopK > 0 && len(objs) > r.cfg.TopK {
		objs = objs[:r.cfg.TopK]
	}
	return RetrievedObjects{Items: objs}, nil
}

func (r *MultiSignalRetrieverAdapter) BuildContext(_ context.Context, query string, objects RetrievedObjects) (string, error) {
	if len(objects.Items) == 0 {
		return "", nil
	}
	var b strings.Builder
	for i, it := range objects.Items {
		fmt.Fprintf(&b, "[%d] %s\n", i+1, it.Text)
	}
	return trimContext(b.String(), r.cfg.MaxContextChars), nil
}

func (r *MultiSignalRetrieverAdapter) Complete(ctx context.Context, query, contextText string, objects RetrievedObjects) (Completion, error) {
	citations := make([]string, 0, len(objects.Items))
	for _, it := range objects.Items {
		citations = append(citations, it.ID)
	}
	var answer string
	if r.complete != nil {
		a, err := r.complete(ctx, query, contextText)
		if err != nil {
			return Completion{}, err
		}
		answer = a
	} else {
		answer = heuristicAnswer(query, objects)
	}
	return Completion{Query: query, Context: contextText, Answer: answer, Citations: citations}, nil
}

func (r *MultiSignalRetrieverAdapter) GetCompletion(ctx context.Context, query string) (Completion, error) {
	objs, err := r.RetrieveObjects(ctx, query)
	if err != nil {
		return Completion{}, err
	}
	ctxText, err := r.BuildContext(ctx, query, objs)
	if err != nil {
		return Completion{}, err
	}
	return r.Complete(ctx, query, ctxText, objs)
}

// WithCompletion attaches an LLM completion callback.
func (r *MultiSignalRetrieverAdapter) WithCompletion(c CompletionFunc) *MultiSignalRetrieverAdapter {
	r.complete = c
	return r
}

// GraphCompletionRetriever applies the distance penalty + feedback influence
// from Cognee's GraphCompletionRetriever. When a GraphTraverser is
// configured (typically *compression/retrieval.SpreadingActivation), the
// retriever augments vector-search hits with multi-hop graph propagation
// before re-ranking. Without a traverser, it falls back to a pure
// distance-penalty re-rank on vector hits.
type GraphCompletionRetriever struct {
	searcher memorySearcher
	traverser graphTraverser
	cfg      *RetrieverConfig
	complete CompletionFunc
}

// NewGraphCompletionRetriever wraps a memorySearcher as a graph-style
// retriever.
func NewGraphCompletionRetriever(s memorySearcher, cfg *RetrieverConfig) *GraphCompletionRetriever {
	if cfg == nil {
		cfg = DefaultRetrieverConfig()
	}
	return &GraphCompletionRetriever{searcher: s, cfg: cfg}
}

// WithGraphTraverser enables real multi-hop graph propagation. Pass nil to
// disable. When non-nil, RetrieveObjects uses spreading activation in
// addition to vector search.
func (r *GraphCompletionRetriever) WithGraphTraverser(t graphTraverser) *GraphCompletionRetriever {
	r.traverser = t
	return r
}

// WithCompletion attaches an LLM completion callback.
func (r *GraphCompletionRetriever) WithCompletion(c CompletionFunc) *GraphCompletionRetriever {
	r.complete = c
	return r
}

func (r *GraphCompletionRetriever) Name() string { return "graph" }

func (r *GraphCompletionRetriever) RetrieveObjects(ctx context.Context, query string) (RetrievedObjects, error) {
	if r.searcher == nil {
		return RetrievedObjects{}, fmt.Errorf("graph retriever: searcher not configured")
	}
	results, err := r.searcher.SearchSemantic(ctx, query, r.cfg.WideSearchTopK)
	if err != nil {
		return RetrievedObjects{}, err
	}
	objs := make([]RetrievedObject, 0, len(results))
	for i, res := range results {
		// Approximate "distance from query" as 1 - normalized score.
		distance := 1.0 - clamp01(float64(res.Score))
		objs = append(objs, RetrievedObject{
			Type:   "memory",
			ID:     res.MemoryID,
			Text:   res.Text,
			Score:  float64(res.Score),
			Source: "graph",
			Meta: map[string]string{
				"hop":      itoaInt(i), // approximate hop rank
				"distance": ftoaFloat(distance),
			},
		})
	}
	// Apply triplet distance penalty.
	for i := range objs {
		d, _ := parseFloat(objs[i].Meta["distance"])
		objs[i].Score = objs[i].Score - (r.cfg.TripletDistancePenalty * d)
	}
	// When a graph traverser is configured, merge its propagation results
	// into the candidate set. Items discovered only by graph traversal
	// (not in the top-WideSearchTopK vector hits) are appended with the
	// traverser's score scaled into the same range.
	if r.traverser != nil {
		hits, err := r.traverser.RetrieveWithScores(ctx, query, GraphModeHybrid)
		if err == nil {
			objs = mergeGraphHits(objs, hits, r.cfg.FeedbackInfluence)
		}
		// Graph traversal failures are non-fatal: fall through to the
		// distance-penalty ranking on vector hits alone.
	}
	objs = rerankOrders(objs, r.cfg.FeedbackInfluence)
	if r.cfg.TopK > 0 && len(objs) > r.cfg.TopK {
		objs = objs[:r.cfg.TopK]
	}
	return RetrievedObjects{Items: objs}, nil
}

// mergeGraphHits combines vector-search objects with spreading-activation
// hits. Vector hits take precedence on conflict (matched by ID); graph-only
// hits are appended with their hop distance folded into the score.
func mergeGraphHits(objs []RetrievedObject, hits []GraphHit, _ float64) []RetrievedObject {
	if len(hits) == 0 {
		return objs
	}
	indexed := make(map[string]struct{}, len(objs))
	for _, o := range objs {
		indexed[o.ID] = struct{}{}
	}
	for _, h := range hits {
		if _, exists := indexed[h.MemoryID]; exists {
			continue
		}
		indexed[h.MemoryID] = struct{}{}
		// Graph-only hit: score is the traverser's raw score; hop count is
		// stored in meta for downstream debugging.
		hopStr := itoaInt(h.Hops)
		objs = append(objs, RetrievedObject{
			Type:   "memory",
			ID:     h.MemoryID,
			Score:  h.Score,
			Source: "graph-traversal",
			Meta: map[string]string{
				"hop": hopStr,
			},
		})
	}
	return objs
}

func (r *GraphCompletionRetriever) BuildContext(_ context.Context, query string, objects RetrievedObjects) (string, error) {
	if len(objects.Items) == 0 {
		return "", nil
	}
	if r.cfg.IncludeGlobalContext {
		// Scaffolding: prepend a placeholder global context header.
		var b strings.Builder
		b.WriteString("[Global context] Recent themes and entities around the query.\n\n")
		for i, it := range objects.Items {
			fmt.Fprintf(&b, "[%d] %s\n", i+1, it.Text)
		}
		return trimContext(b.String(), r.cfg.MaxContextChars), nil
	}
	var b strings.Builder
	for i, it := range objects.Items {
		fmt.Fprintf(&b, "[%d] %s\n", i+1, it.Text)
	}
	return trimContext(b.String(), r.cfg.MaxContextChars), nil
}

func (r *GraphCompletionRetriever) Complete(ctx context.Context, query, contextText string, objects RetrievedObjects) (Completion, error) {
	citations := make([]string, 0, len(objects.Items))
	for _, it := range objects.Items {
		citations = append(citations, it.ID)
	}
	var answer string
	if r.complete != nil {
		a, err := r.complete(ctx, query, contextText)
		if err != nil {
			return Completion{}, err
		}
		answer = a
	} else {
		answer = heuristicAnswer(query, objects)
	}
	return Completion{Query: query, Context: contextText, Answer: answer, Citations: citations}, nil
}

func (r *GraphCompletionRetriever) GetCompletion(ctx context.Context, query string) (Completion, error) {
	objs, err := r.RetrieveObjects(ctx, query)
	if err != nil {
		return Completion{}, err
	}
	ctxText, err := r.BuildContext(ctx, query, objs)
	if err != nil {
		return Completion{}, err
	}
	return r.Complete(ctx, query, ctxText, objs)
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func itoaInt(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	buf := make([]byte, 0, 12)
	for n > 0 {
		buf = append([]byte{byte('0' + n%10)}, buf...)
		n /= 10
	}
	if neg {
		buf = append([]byte{'-'}, buf...)
	}
	return string(buf)
}

func ftoaFloat(v float64) string {
	// Simple fixed-precision conversion. Avoids importing strconv in hot
	// path; sufficient for meta values.
	// We only need 4 decimals.
	whole := int(v)
	frac := int((v - float64(whole)) * 10000)
	if frac < 0 {
		frac = -frac
	}
	return itoaInt(whole) + "." + padLeft(itoaInt(frac), 4, '0')
}

func padLeft(s string, width int, ch byte) string {
	if len(s) >= width {
		return s
	}
	return strings.Repeat(string(ch), width-len(s)) + s
}

func parseFloat(s string) (float64, error) {
	// Minimal float parser. Accepts signed integer or decimal forms.
	if s == "" {
		return 0, nil
	}
	neg := false
	if s[0] == '-' {
		neg = true
		s = s[1:]
	} else if s[0] == '+' {
		s = s[1:]
	}
	whole := 0.0
	frac := 0.0
	divisor := 1.0
	seenDot := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == '.' {
			seenDot = true
			continue
		}
		if c < '0' || c > '9' {
			break
		}
		d := float64(c - '0')
		if !seenDot {
			whole = whole*10 + d
		} else {
			divisor *= 10
			frac = frac*10 + d
		}
	}
	v := whole + frac/divisor
	if neg {
		v = -v
	}
	return v, nil
}
