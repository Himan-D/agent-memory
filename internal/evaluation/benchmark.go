package evaluation

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"agent-memory/internal/llm"
)

type BenchmarkConfig struct {
	Model         string
	MaxTokens     int
	ParallelLimit int
}

type BenchmarkResult struct {
	Dataset             string   `json:"dataset"`
	OverallScore        float64  `json:"overall_score"`
	SingleHopScore      float64  `json:"single_hop_score"`
	MultiHopScore       float64  `json:"multi_hop_score"`
	MemoryHitRate       float64  `json:"memory_hit_rate"`
	MRR                 float64  `json:"mrr"`
	AvgRetrievedItems   float64  `json:"avg_retrieved_items"`
	TokensRetrieved     int      `json:"tokens_retrieved"`
	MaxContextTokens    int      `json:"max_context_tokens,omitempty"`
	LatencyP50Ms        float64  `json:"latency_p50_ms"`
	LatencyP95Ms        float64  `json:"latency_p95_ms"`
	QuestionsAnswered   int      `json:"questions_answered"`
	TotalQuestions      int      `json:"total_questions"`
	MemoriesIngested    int      `json:"memories_ingested"`
	IngestErrors        int      `json:"ingest_errors"`
	SearchErrors        int      `json:"search_errors"`
	ScoredQuestions     int      `json:"scored_questions"`
	ScoringErrors       int      `json:"scoring_errors"`
	EvaluatorConfigured bool     `json:"evaluator_configured"`
	Publishable         bool     `json:"publishable"`
	ScoreMethod         string   `json:"score_method"`
	Warnings            []string `json:"warnings,omitempty"`
	Timestamp           string   `json:"timestamp"`
}

type BenchmarkQuestion struct {
	ID          string `json:"id"`
	Question    string `json:"question"`
	SessionID   string `json:"session_id"`
	MemoryID    string `json:"memory_id,omitempty"`
	Category    string `json:"category"`
	GroundTruth string `json:"ground_truth,omitempty"`
}

type BenchmarkDataset struct {
	Name             string              `json:"name"`
	Questions        []BenchmarkQuestion `json:"questions"`
	Memories         []BenchmarkMemory   `json:"memories"`
	MaxContextTokens int                 `json:"max_context_tokens,omitempty"` // token budget per query; 0 = unlimited
}

type BenchmarkMemory struct {
	ID      string `json:"id"`
	Content string `json:"content"`
	UserID  string `json:"user_id"`
}

type Scorer struct {
	llmClient llm.Provider
	config    BenchmarkConfig
}

func NewScorer(llmClient llm.Provider, config BenchmarkConfig) *Scorer {
	return &Scorer{
		llmClient: llmClient,
		config:    config,
	}
}

func (s *Scorer) ScoreAnswer(ctx context.Context, question, answer, groundTruth string) (float64, error) {
	if s.llmClient == nil {
		return 0, fmt.Errorf("benchmark scorer: no LLM provider configured")
	}

	prompt := fmt.Sprintf(`You are evaluating AI memory retrieval quality.
Compare the Retrieved Answer against the Expected Answer (Ground Truth) to see if they convey the same key facts.
The original Question is provided as context.

Question: %s
Retrieved Answer: %s
Expected Answer: %s

Rate the semantic match quality between the Retrieved Answer and the Expected Answer from 0-100 where:
- 100: Retrieved Answer completely matches the Expected Answer in semantic meaning or contains the key fact.
- 75: Retrieved Answer is mostly correct and matches the Expected Answer but misses minor details.
- 50: Retrieved Answer is partially correct compared to the Expected Answer.
- 25: Retrieved Answer has some correct information but is mostly wrong or unrelated to the Expected Answer.
- 0: Retrieved Answer is completely wrong, unrelated, or missing.

Return ONLY a number between 0-100.`, question, answer, groundTruth)


	resp, err := s.llmClient.Complete(ctx, &llm.CompletionRequest{
		Model:       s.config.Model,
		Messages:    []llm.Message{{Role: "system", Content: prompt}},
		Temperature: 0.1,
		MaxTokens:   s.config.MaxTokens,
	})
	if err != nil {
		return 0, err
	}

	var score float64
	fmt.Sscanf(resp.Content, "%f", &score)
	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}
	return score / 100.0, nil
}

type BenchmarkRunner struct {
	scorer  *Scorer
	config  BenchmarkConfig
	results []BenchmarkResult
	mu      sync.Mutex
}

func NewBenchmarkRunner(scorer *Scorer, config BenchmarkConfig) *BenchmarkRunner {
	return &BenchmarkRunner{
		scorer:  scorer,
		config:  config,
		results: make([]BenchmarkResult, 0),
	}
}

func (r *BenchmarkRunner) LoadDataset(name string) (*BenchmarkDataset, error) {
	paths := []string{
		filepath.Join("internal", "evaluation", name, "dataset.json"),
		filepath.Join("evaluation", name, "dataset.json"),
	}
	if strings.HasPrefix(name, "beam_") {
		scale := strings.TrimPrefix(name, "beam_")
		paths = append(paths,
			filepath.Join("internal", "evaluation", "beam", "beam_"+scale+"_dataset.json"),
			filepath.Join("evaluation", "beam", "beam_"+scale+"_dataset.json"),
			// fallback to the shared beam dataset
			filepath.Join("internal", "evaluation", "beam", "dataset.json"),
			filepath.Join("evaluation", "beam", "dataset.json"),
		)
	}

	var data []byte
	var lastErr error
	for _, path := range paths {
		var err error
		data, err = os.ReadFile(path)
		if err == nil {
			lastErr = nil
			break
		}
		lastErr = err
	}
	if lastErr != nil {
		return nil, fmt.Errorf("load dataset %s: %w", name, lastErr)
	}

	var dataset BenchmarkDataset
	if err := json.Unmarshal(data, &dataset); err != nil {
		return nil, fmt.Errorf("parse dataset: %w", err)
	}

	return &dataset, nil
}

func (r *BenchmarkRunner) RunLoCoMo(ctx context.Context, memSvc MemoryService, searchFn SearchFunc) (*BenchmarkResult, error) {
	dataset, err := r.LoadDataset("locomo")
	if err != nil {
		return nil, err
	}

	results := r.runBenchmark(ctx, dataset, memSvc, searchFn)
	return r.summarizeResults("locomo", results), nil
}

func (r *BenchmarkRunner) RunLongMemEval(ctx context.Context, memSvc MemoryService, searchFn SearchFunc) (*BenchmarkResult, error) {
	dataset, err := r.LoadDataset("longmemeval")
	if err != nil {
		return nil, err
	}

	results := r.runBenchmark(ctx, dataset, memSvc, searchFn)
	return r.summarizeResults("longmemeval", results), nil
}

func (r *BenchmarkRunner) RunBEAM(ctx context.Context, memSvc MemoryService, searchFn SearchFunc, scale string) (*BenchmarkResult, error) {
	dataset, err := r.LoadDataset(fmt.Sprintf("beam_%s", scale))
	if err != nil {
		return nil, err
	}

	results := r.runBenchmark(ctx, dataset, memSvc, searchFn)
	result := r.summarizeResults(fmt.Sprintf("beam_%s", scale), results)
	result.MaxContextTokens = dataset.MaxContextTokens
	return result, nil
}

type questionResult struct {
	QuestionID string
	Score      float64
	Latency    time.Duration
	Tokens     int
	Category   string
	SearchErr  error
	ScoreErr   error
	Scored     bool
	ExpectedID string
	HitRank    int
	Retrieved  int
	Ingested   bool
	IngestErr  error
}

func (r *BenchmarkRunner) runBenchmark(ctx context.Context, dataset *BenchmarkDataset, memSvc MemoryService, searchFn SearchFunc) []questionResult {
	if cleanupSvc, ok := memSvc.(interface{ CleanupBenchmarkMemories(context.Context) error }); ok {
		_ = cleanupSvc.CleanupBenchmarkMemories(ctx)
	}

	results := make([]questionResult, 0, len(dataset.Questions))
	for _, mem := range dataset.Memories {
		userID := mem.UserID
		if userID == "" {
			userID = "benchmark-user"
		}
		var err error
		if labeledSvc, ok := memSvc.(LabeledMemoryService); ok {
			_, err = labeledSvc.CreateBenchmarkMemory(ctx, mem)
		} else {
			_, err = memSvc.CreateMemory(ctx, mem.Content, userID)
		}
		if err != nil {
			results = append(results, questionResult{
				QuestionID: mem.ID,
				IngestErr:  fmt.Errorf("ingest memory: %w", err),
			})
			continue
		}
		results = append(results, questionResult{
			QuestionID: mem.ID,
			Ingested:   true,
		})
	}

	if flusher, ok := memSvc.(interface{ Flush(context.Context) error }); ok {
		_ = flusher.Flush(ctx)
	}
	time.Sleep(100 * time.Millisecond)

	parallelLimit := r.config.ParallelLimit
	if parallelLimit < 1 {
		parallelLimit = 1
	}
	sem := make(chan struct{}, parallelLimit)
	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, q := range dataset.Questions {
		wg.Add(1)
		go func(question BenchmarkQuestion) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			start := time.Now()

			memoryResults, err := searchFn(ctx, question.SessionID, question.Question)
			var answer string
			if err == nil && len(memoryResults) > 0 {
				answer = memoryResults[0].Content
			} else {
				answer = "No relevant memory found."
			}

			// Enforce context token budget if set by dataset (e.g. BEAM 10M <7K tokens)
			if dataset.MaxContextTokens > 0 {
				maxChars := dataset.MaxContextTokens * 4 // approx 4 chars per token
				if len(answer) > maxChars {
					answer = answer[:maxChars]
				}
			}

			latency := time.Since(start)

			var score float64
			var scoreErr error
			scored := false
			if question.GroundTruth != "" && r.scorer != nil && r.scorer.llmClient != nil {
				score, scoreErr = r.scorer.ScoreAnswer(ctx, question.Question, answer, question.GroundTruth)
				scored = scoreErr == nil
			}
			if scored {
				fmt.Printf("DEBUG: Question %s\n - Query: %q\n - Retrieved: %q\n - Expected: %q\n - Score: %.2f\n", question.ID, question.Question, answer, question.GroundTruth, score)
			} else if scoreErr != nil {
				fmt.Printf("DEBUG: Question %s - Scoring Error: %v\n", question.ID, scoreErr)
			}
			hitRank := hitRank(memoryResults, question.MemoryID)

			mu.Lock()
			results = append(results, questionResult{
				QuestionID: question.ID,
				Score:      score,
				Latency:    latency,
				Tokens:     len(answer) / 4,
				Category:   question.Category,
				SearchErr:  err,
				ScoreErr:   scoreErr,
				Scored:     scored,
				ExpectedID: question.MemoryID,
				HitRank:    hitRank,
				Retrieved:  len(memoryResults),
			})
			mu.Unlock()
		}(q)
	}

	wg.Wait()
	return results
}

func (r *BenchmarkRunner) summarizeResults(name string, qResults []questionResult) *BenchmarkResult {
	var totalScore, singleHopScore, multiHopScore float64
	var singleHopCount, multiHopCount int
	var scoredCount, searchErrors, scoringErrors, ingestErrors, memoriesIngested int
	var expectedIDCount, memoryHits, retrievedTotal int
	var reciprocalRankTotal float64
	var latencies []float64
	var totalTokens int

	for _, qr := range qResults {
		if qr.Ingested {
			memoriesIngested++
			continue
		}
		if qr.IngestErr != nil {
			ingestErrors++
			continue
		}
		if qr.SearchErr != nil {
			searchErrors++
		}
		retrievedTotal += qr.Retrieved
		if qr.ExpectedID != "" {
			expectedIDCount++
			if qr.HitRank > 0 {
				memoryHits++
				reciprocalRankTotal += 1 / float64(qr.HitRank)
			}
		}
		if qr.ScoreErr != nil {
			scoringErrors++
		}
		if !qr.Scored {
			continue
		}
		scoredCount++
		totalScore += qr.Score
		latencies = append(latencies, qr.Latency.Seconds()*1000)
		totalTokens += qr.Tokens

		switch qr.Category {
		case "single_hop", "user":
			singleHopScore += qr.Score
			singleHopCount++
		case "multi_hop", "temporal":
			multiHopScore += qr.Score
			multiHopCount++
		}
	}

	resultCount := len(qResults)
	if resultCount == 0 {
		return &BenchmarkResult{Dataset: name, Timestamp: time.Now().Format(time.RFC3339)}
	}
	questionCount := resultCount - memoriesIngested - ingestErrors

	result := &BenchmarkResult{
		Dataset:             name,
		QuestionsAnswered:   questionCount - searchErrors,
		TotalQuestions:      questionCount,
		MemoriesIngested:    memoriesIngested,
		IngestErrors:        ingestErrors,
		SearchErrors:        searchErrors,
		ScoredQuestions:     scoredCount,
		ScoringErrors:       scoringErrors,
		EvaluatorConfigured: r.scorer != nil && r.scorer.llmClient != nil,
		ScoreMethod:         "unscored",
		Timestamp:           time.Now().Format(time.RFC3339),
	}
	if scoredCount > 0 {
		result.OverallScore = totalScore / float64(scoredCount)
		result.TokensRetrieved = totalTokens / scoredCount
		result.ScoreMethod = "llm_judge"
	} else if questionCount > 0 {
		result.Warnings = append(result.Warnings, "no questions were scored; configure an evaluator LLM before publishing benchmark numbers")
	}
	if questionCount > 0 {
		result.AvgRetrievedItems = float64(retrievedTotal) / float64(questionCount)
	}
	if expectedIDCount > 0 {
		result.MemoryHitRate = float64(memoryHits) / float64(expectedIDCount)
		result.MRR = reciprocalRankTotal / float64(expectedIDCount)
	} else {
		result.Warnings = append(result.Warnings, "dataset has no expected memory IDs; memory_hit_rate and mrr cannot be computed")
	}
	result.Publishable = result.EvaluatorConfigured && result.ScoredQuestions == result.TotalQuestions && result.SearchErrors == 0 && result.IngestErrors == 0
	if !result.Publishable {
		result.Warnings = append(result.Warnings, "result is not publishable as competitive evidence until all questions are LLM-scored with zero ingest/search errors")
	}

	if singleHopCount > 0 {
		result.SingleHopScore = singleHopScore / float64(singleHopCount)
	}
	if multiHopCount > 0 {
		result.MultiHopScore = multiHopScore / float64(multiHopCount)
	}

	if len(latencies) > 0 {
		result.LatencyP50Ms = percentile(latencies, 50)
		result.LatencyP95Ms = percentile(latencies, 95)
	}

	return result
}

func hitRank(results []MemoryResult, expectedID string) int {
	if expectedID == "" {
		return 0
	}
	for idx, result := range results {
		if result.ID == expectedID {
			return idx + 1
		}
	}
	return 0
}

func percentile(values []float64, p float64) float64 {
	if len(values) == 0 {
		return 0
	}

	sorted := make([]float64, len(values))
	copy(sorted, values)
	sort.Float64s(sorted)

	n := len(sorted)
	k := int(float64(n-1) * p / 100)
	if k >= n {
		k = n - 1
	}
	return sorted[k]
}

type MemoryService interface {
	CreateMemory(ctx context.Context, content, userID string) (string, error)
	GetMemories(ctx context.Context, sessionID string) ([]MemoryResult, error)
}

type LabeledMemoryService interface {
	CreateBenchmarkMemory(ctx context.Context, mem BenchmarkMemory) (string, error)
}

type SearchFunc func(ctx context.Context, sessionID, query string) ([]MemoryResult, error)

type MemoryResult struct {
	ID      string
	Content string
	Score   float32
}

type RunAllResult struct {
	LoCoMo      *BenchmarkResult  `json:"locomo"`
	LongMemEval *BenchmarkResult  `json:"longmemeval"`
	BEAM1M      *BenchmarkResult  `json:"beam_1m"`
	BEAM10M     *BenchmarkResult  `json:"beam_10m"`
	Errors      map[string]string `json:"errors,omitempty"`
	Timestamp   string            `json:"timestamp"`
}

func (r *BenchmarkRunner) RunAll(ctx context.Context, memSvc MemoryService, searchFn SearchFunc) *RunAllResult {
	result := &RunAllResult{
		Timestamp: time.Now().Format(time.RFC3339),
	}

	if loCoMo, err := r.RunLoCoMo(ctx, memSvc, searchFn); err == nil {
		result.LoCoMo = loCoMo
	} else {
		addRunAllError(result, "locomo", err)
	}

	if longMem, err := r.RunLongMemEval(ctx, memSvc, searchFn); err == nil {
		result.LongMemEval = longMem
	} else {
		addRunAllError(result, "longmemeval", err)
	}

	if beam1m, err := r.RunBEAM(ctx, memSvc, searchFn, "1m"); err == nil {
		result.BEAM1M = beam1m
	} else {
		addRunAllError(result, "beam_1m", err)
	}

	if beam10m, err := r.RunBEAM(ctx, memSvc, searchFn, "10m"); err == nil {
		result.BEAM10M = beam10m
	} else {
		addRunAllError(result, "beam_10m", err)
	}

	return result
}

func addRunAllError(result *RunAllResult, dataset string, err error) {
	if result.Errors == nil {
		result.Errors = map[string]string{}
	}
	result.Errors[dataset] = err.Error()
}
