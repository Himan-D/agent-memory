package evaluations

import (
	"context"
	"fmt"
	"math"
	"time"
)

type Evaluator interface {
	Run(ctx context.Context, testset *TestSet) (*EvaluationResult, error)
	RunRecall(ctx context.Context, testset *TestSet) (*RecallMetrics, error)
	RunPrecision(ctx context.Context, testset *TestSet) (*PrecisionMetrics, error)
	RunFaithfulness(ctx context.Context, testset *TestSet) (*FaithfulnessMetrics, error)
}

type TestSet struct {
	ID          string        `json:"id"`
	Name        string        `json:"name"`
	Description string        `json:"description"`
	Queries     []TestQuery   `json:"queries"`
	GroundTruth []GroundTruth `json:"ground_truth"`
	CreatedAt   time.Time     `json:"created_at"`
}

type TestQuery struct {
	ID       string                 `json:"id"`
	Query    string                 `json:"query"`
	Expected []string               `json:"expected"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

type GroundTruth struct {
	QueryID    string `json:"query_id"`
	MemoryID   string `json:"memory_id"`
	Content    string `json:"content"`
	IsRelevant bool   `json:"is_relevant"`
}

type EvaluationResult struct {
	TestSetID       string              `json:"test_set_id"`
	Timestamp       time.Time           `json:"timestamp"`
	OverallScore    float64             `json:"overall_score"`
	Recall          RecallMetrics       `json:"recall"`
	Precision       PrecisionMetrics    `json:"precision"`
	Faithfulness    FaithfulnessMetrics `json:"faithfulness"`
	Latency         LatencyMetrics      `json:"latency"`
	PerQueryResults []QueryResult       `json:"per_query_results"`
}

type RecallMetrics struct {
	Score             float64 `json:"score"`
	RelevantRetrieved int     `json:"relevant_retrieved"`
	TotalRelevant     int     `json:"total_relevant"`
	MRR               float64 `json:"mrr"`
	NDCG              float64 `json:"ndcg"`
}

type PrecisionMetrics struct {
	Score             float64 `json:"score"`
	RetrievedRelevant int     `json:"retrieved_relevant"`
	TotalRetrieved    int     `json:"total_retrieved"`
	AP                float64 `json:"ap"`
}

type FaithfulnessMetrics struct {
	Score          float64 `json:"score"`
	TotalClaims    int     `json:"total_claims"`
	AccurateClaims int     `json:"accurate_claims"`
	LLMJudgment    string  `json:"llm_judgment,omitempty"`
}

type LatencyMetrics struct {
	AvgMs      float64 `json:"avg_ms"`
	P50Ms      float64 `json:"p50_ms"`
	P95Ms      float64 `json:"p95_ms"`
	P99Ms      float64 `json:"p99_ms"`
	TotalCalls int     `json:"total_calls"`
}

type QueryResult struct {
	QueryID   string         `json:"query_id"`
	Query     string         `json:"query"`
	Recall    float64        `json:"recall"`
	Precision float64        `json:"precision"`
	LatencyMs float64        `json:"latency_ms"`
	Retrieved []RetrievedDoc `json:"retrieved"`
}

type RetrievedDoc struct {
	MemoryID string  `json:"memory_id"`
	Content  string  `json:"content"`
	Score    float64 `json:"score"`
	Relevant bool    `json:"relevant"`
}

type DefaultEvaluator struct {
	retriever func(ctx context.Context, query string, limit int) ([]RetrievedDoc, error)
	llmJudge  func(ctx context.Context, claim, context string) (bool, error)
}

func NewDefaultEvaluator(
	retriever func(ctx context.Context, query string, limit int) ([]RetrievedDoc, error),
	llmJudge func(ctx context.Context, claim, context string) (bool, error),
) *DefaultEvaluator {
	return &DefaultEvaluator{
		retriever: retriever,
		llmJudge:  llmJudge,
	}
}

func (e *DefaultEvaluator) Run(ctx context.Context, testset *TestSet) (*EvaluationResult, error) {
	result := &EvaluationResult{
		TestSetID:       testset.ID,
		Timestamp:       time.Now(),
		PerQueryResults: make([]QueryResult, 0, len(testset.Queries)),
	}

	var totalRecall, totalPrecision, totalLatency float64
	var latencies []float64

	for _, query := range testset.Queries {
		start := time.Now()

		retrieved, err := e.retriever(ctx, query.Query, 10)
		if err != nil {
			continue
		}

		latency := time.Since(start).Seconds() * 1000
		latencies = append(latencies, latency)

		relevantSet := make(map[string]bool)
		for _, gt := range testset.GroundTruth {
			if gt.QueryID == query.ID && gt.IsRelevant {
				relevantSet[gt.MemoryID] = true
			}
		}

		retrievedRelevant := 0
		for i, doc := range retrieved {
			doc.Relevant = relevantSet[doc.MemoryID]
			if doc.Relevant {
				retrievedRelevant++
			}
			retrieved[i] = doc
		}

		recall := 0.0
		if len(relevantSet) > 0 {
			recall = float64(retrievedRelevant) / float64(len(relevantSet))
		}

		precision := 0.0
		if len(retrieved) > 0 {
			precision = float64(retrievedRelevant) / float64(len(retrieved))
		}

		qr := QueryResult{
			QueryID:   query.ID,
			Query:     query.Query,
			Recall:    recall,
			Precision: precision,
			LatencyMs: latency,
			Retrieved: retrieved,
		}
		result.PerQueryResults = append(result.PerQueryResults, qr)

		totalRecall += recall
		totalPrecision += precision
		totalLatency += latency
	}

	if len(testset.Queries) > 0 {
		result.Recall.Score = totalRecall / float64(len(testset.Queries))
		result.Precision.Score = totalPrecision / float64(len(testset.Queries))
		result.Latency.AvgMs = totalLatency / float64(len(testset.Queries))
	}

	result.Latency.TotalCalls = len(testset.Queries)
	if len(latencies) > 0 {
		result.Latency.P50Ms = percentile(latencies, 50)
		result.Latency.P95Ms = percentile(latencies, 95)
		result.Latency.P99Ms = percentile(latencies, 99)
	}

	result.OverallScore = (result.Recall.Score + result.Precision.Score) / 2

	return result, nil
}

func (e *DefaultEvaluator) RunRecall(ctx context.Context, testset *TestSet) (*RecallMetrics, error) {
	metrics := &RecallMetrics{}

	gtByQuery := make(map[string]map[string]bool)
	for _, gt := range testset.GroundTruth {
		if gt.IsRelevant {
			if gtByQuery[gt.QueryID] == nil {
				gtByQuery[gt.QueryID] = make(map[string]bool)
			}
			gtByQuery[gt.QueryID][gt.MemoryID] = true
		}
	}

	var totalRelevant, totalRetrievedRelevant int
	var mrrSum, ndcgSum float64

	for _, query := range testset.Queries {
		relevant := gtByQuery[query.ID]
		if relevant == nil {
			continue
		}

		retrieved, err := e.retriever(ctx, query.Query, 10)
		if err != nil {
			continue
		}

		retrievedRelevant := 0
		rank := 0
		for _, doc := range retrieved {
			rank++
			if relevant[doc.MemoryID] {
				retrievedRelevant++
				mrrSum += 1.0 / float64(rank)
				ndcgSum += 1.0 / math.Log2(float64(rank+1))
			}
		}

		totalRelevant += len(relevant)
		totalRetrievedRelevant += retrievedRelevant
	}

	if totalRelevant > 0 {
		metrics.Score = float64(totalRetrievedRelevant) / float64(totalRelevant)
	}
	metrics.TotalRelevant = totalRelevant
	metrics.RelevantRetrieved = totalRetrievedRelevant

	if len(testset.Queries) > 0 {
		metrics.MRR = mrrSum / float64(len(testset.Queries))
		metrics.NDCG = ndcgSum / float64(len(testset.Queries))
	}

	return metrics, nil
}

func (e *DefaultEvaluator) RunPrecision(ctx context.Context, testset *TestSet) (*PrecisionMetrics, error) {
	metrics := &PrecisionMetrics{}

	gtByQuery := make(map[string]map[string]bool)
	for _, gt := range testset.GroundTruth {
		if gt.IsRelevant {
			if gtByQuery[gt.QueryID] == nil {
				gtByQuery[gt.QueryID] = make(map[string]bool)
			}
			gtByQuery[gt.QueryID][gt.MemoryID] = true
		}
	}

	var totalRetrieved, totalRetrievedRelevant int
	var apSum float64

	for _, query := range testset.Queries {
		relevant := gtByQuery[query.ID]

		retrieved, err := e.retriever(ctx, query.Query, 10)
		if err != nil {
			continue
		}

		retrievedRelevant := 0
		relevantCount := 0
		if relevant != nil {
			relevantCount = len(relevant)
		}

		for i, doc := range retrieved {
			if relevant != nil && relevant[doc.MemoryID] {
				retrievedRelevant++
			}
			if relevantCount > 0 {
				apSum += float64(retrievedRelevant) / float64(i+1)
			}
		}

		totalRetrieved += len(retrieved)
		totalRetrievedRelevant += retrievedRelevant
	}

	if totalRetrieved > 0 {
		metrics.Score = float64(totalRetrievedRelevant) / float64(totalRetrieved)
	}
	metrics.TotalRetrieved = totalRetrieved
	metrics.RetrievedRelevant = totalRetrievedRelevant

	if len(testset.Queries) > 0 {
		metrics.AP = apSum / float64(len(testset.Queries))
	}

	return metrics, nil
}

func (e *DefaultEvaluator) RunFaithfulness(ctx context.Context, testset *TestSet) (*FaithfulnessMetrics, error) {
	metrics := &FaithfulnessMetrics{}

	var totalClaims, accurateClaims int

	for _, query := range testset.Queries {
		retrieved, err := e.retriever(ctx, query.Query, 5)
		if err != nil {
			continue
		}

		var context string
		for _, doc := range retrieved {
			context += doc.Content + " "
		}

		if e.llmJudge != nil {
			claims := extractClaims(query.Query)
			for _, claim := range claims {
				totalClaims++
				accurate, _ := e.llmJudge(ctx, claim, context)
				if accurate {
					accurateClaims++
				}
			}
		}
	}

	metrics.TotalClaims = totalClaims
	metrics.AccurateClaims = accurateClaims
	if totalClaims > 0 {
		metrics.Score = float64(accurateClaims) / float64(totalClaims)
	}

	return metrics, nil
}

func extractClaims(text string) []string {
	return []string{text}
}

func percentile(values []float64, p int) float64 {
	if len(values) == 0 {
		return 0
	}

	sorted := make([]float64, len(values))
	copy(sorted, values)

	for i := 0; i < len(sorted)-1; i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[j] < sorted[i] {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

	idx := (float64(p) / 100.0) * float64(len(sorted)-1)
	lower := int(idx)
	upper := lower + 1
	if upper >= len(sorted) {
		return sorted[len(sorted)-1]
	}

	frac := idx - float64(lower)
	return sorted[lower]*(1-frac) + sorted[upper]*frac
}

type BenchmarkSuite struct {
	Name        string     `json:"name"`
	Description string     `json:"description"`
	TestSets    []*TestSet `json:"test_sets"`
	Evaluator   Evaluator  `json:"-"`
}

func (s *BenchmarkSuite) Run(ctx context.Context) (*BenchmarkResult, error) {
	result := &BenchmarkResult{
		SuiteName:   s.Name,
		Timestamp:   time.Now(),
		TestResults: make([]*EvaluationResult, 0, len(s.TestSets)),
	}

	for _, ts := range s.TestSets {
		evalResult, err := s.Evaluator.Run(ctx, ts)
		if err != nil {
			continue
		}
		result.TestResults = append(result.TestResults, evalResult)
	}

	var totalScore float64
	for _, r := range result.TestResults {
		totalScore += r.OverallScore
	}
	if len(result.TestResults) > 0 {
		result.OverallScore = totalScore / float64(len(result.TestResults))
	}

	return result, nil
}

type BenchmarkResult struct {
	SuiteName    string              `json:"suite_name"`
	Timestamp    time.Time           `json:"timestamp"`
	OverallScore float64             `json:"overall_score"`
	TestResults  []*EvaluationResult `json:"test_results"`
}

type TestSetManager struct {
	testsets map[string]*TestSet
}

func NewTestSetManager() *TestSetManager {
	return &TestSetManager{
		testsets: make(map[string]*TestSet),
	}
}

func (m *TestSetManager) Add(testset *TestSet) error {
	if testset.ID == "" {
		return fmt.Errorf("testset ID is required")
	}
	m.testsets[testset.ID] = testset
	return nil
}

func (m *TestSetManager) Get(id string) (*TestSet, error) {
	ts, ok := m.testsets[id]
	if !ok {
		return nil, fmt.Errorf("testset not found: %s", id)
	}
	return ts, nil
}

func (m *TestSetManager) List() []*TestSet {
	result := make([]*TestSet, 0, len(m.testsets))
	for _, ts := range m.testsets {
		result = append(result, ts)
	}
	return result
}

func (m *TestSetManager) Delete(id string) error {
	if _, ok := m.testsets[id]; !ok {
		return fmt.Errorf("testset not found: %s", id)
	}
	delete(m.testsets, id)
	return nil
}
