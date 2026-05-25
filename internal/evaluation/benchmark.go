package evaluation

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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
	Dataset          string  `json:"dataset"`
	OverallScore    float64 `json:"overall_score"`
	SingleHopScore   float64 `json:"single_hop_score"`
	MultiHopScore    float64 `json:"multi_hop_score"`
	TokensRetrieved  int     `json:"tokens_retrieved"`
	LatencyP50Ms     float64 `json:"latency_p50_ms"`
	LatencyP95Ms     float64 `json:"latency_p95_ms"`
	QuestionsAnswered int    `json:"questions_answered"`
	TotalQuestions   int     `json:"total_questions"`
	Timestamp        string  `json:"timestamp"`
}

type BenchmarkQuestion struct {
	ID         string `json:"id"`
	Question   string `json:"question"`
	SessionID  string `json:"session_id"`
	MemoryID   string `json:"memory_id,omitempty"`
	Category   string `json:"category"`
	GroundTruth string `json:"ground_truth,omitempty"`
}

type BenchmarkDataset struct {
	Name       string             `json:"name"`
	Questions  []BenchmarkQuestion `json:"questions"`
	Memories    []BenchmarkMemory  `json:"memories"`
}

type BenchmarkMemory struct {
	ID      string `json:"id"`
	Content string `json:"content"`
	UserID  string `json:"user_id"`
}

type Scorer struct {
	llmClient llm.Provider
	config   BenchmarkConfig
}

func NewScorer(llmClient llm.Provider, config BenchmarkConfig) *Scorer {
	return &Scorer{
		llmClient: llmClient,
		config:   config,
	}
}

func (s *Scorer) ScoreAnswer(ctx context.Context, question, answer, groundTruth string) (float64, error) {
	prompt := fmt.Sprintf(`You are evaluating AI memory retrieval quality.

Question: %s
Retrieved Answer: %s
Expected Answer: %s

Rate the quality from 0-100 where:
- 100: Answer is complete and correct
- 75: Answer is mostly correct but missing details
- 50: Answer is partially correct
- 25: Answer has some correct info but mostly wrong
- 0: Answer is completely wrong or missing

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
	scorer     *Scorer
	config     BenchmarkConfig
	results    []BenchmarkResult
	mu         sync.Mutex
}

func NewBenchmarkRunner(scorer *Scorer, config BenchmarkConfig) *BenchmarkRunner {
	return &BenchmarkRunner{
		scorer:  scorer,
		config:  config,
		results: make([]BenchmarkResult, 0),
	}
}

func (r *BenchmarkRunner) LoadDataset(name string) (*BenchmarkDataset, error) {
	path := filepath.Join("evaluation", name, "dataset.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("load dataset: %w", err)
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
	return r.summarizeResults(fmt.Sprintf("beam_%s", scale), results), nil
}

type questionResult struct {
	QuestionID string
	Score      float64
	Latency    time.Duration
	Tokens     int
	Category   string
}

func (r *BenchmarkRunner) runBenchmark(ctx context.Context, dataset *BenchmarkDataset, memSvc MemoryService, searchFn SearchFunc) []questionResult {
	results := make([]questionResult, 0, len(dataset.Questions))
	
	sem := make(chan struct{}, r.config.ParallelLimit)
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

			latency := time.Since(start)

			var score float64
			if question.GroundTruth != "" {
				score, _ = r.scorer.ScoreAnswer(ctx, question.Question, answer, question.GroundTruth)
			}

			mu.Lock()
			results = append(results, questionResult{
				QuestionID: question.ID,
				Score:      score,
				Latency:    latency,
				Tokens:     len(answer) / 4,
				Category:   question.Category,
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
	var latencies []float64
	var totalTokens int

	for _, qr := range qResults {
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

	n := len(qResults)
	if n == 0 {
		return &BenchmarkResult{Dataset: name}
	}

	result := &BenchmarkResult{
		Dataset:           name,
		OverallScore:     totalScore / float64(n),
		TokensRetrieved:  totalTokens / n,
		QuestionsAnswered: n,
		TotalQuestions:   n,
		Timestamp:        time.Now().Format(time.RFC3339),
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

func percentile(values []float64, p float64) float64 {
	if len(values) == 0 {
		return 0
	}
	
	sorted := make([]float64, len(values))
	copy(sorted, values)
	
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

type SearchFunc func(ctx context.Context, sessionID, query string) ([]MemoryResult, error)

type MemoryResult struct {
	ID      string
	Content string
	Score   float32
}

type RunAllResult struct {
	LoCoMo      *BenchmarkResult `json:"locomo"`
	LongMemEval *BenchmarkResult `json:"longmemeval"`
	BEAM1M      *BenchmarkResult `json:"beam_1m"`
	BEAM10M     *BenchmarkResult `json:"beam_10m"`
	Timestamp   string           `json:"timestamp"`
}

func (r *BenchmarkRunner) RunAll(ctx context.Context, memSvc MemoryService, searchFn SearchFunc) *RunAllResult {
	result := &RunAllResult{
		Timestamp: time.Now().Format(time.RFC3339),
	}

	if loCoMo, err := r.RunLoCoMo(ctx, memSvc, searchFn); err == nil {
		result.LoCoMo = loCoMo
	}

	if longMem, err := r.RunLongMemEval(ctx, memSvc, searchFn); err == nil {
		result.LongMemEval = longMem
	}

	if beam1m, err := r.RunBEAM(ctx, memSvc, searchFn, "1m"); err == nil {
		result.BEAM1M = beam1m
	}

	if beam10m, err := r.RunBEAM(ctx, memSvc, searchFn, "10m"); err == nil {
		result.BEAM10M = beam10m
	}

	return result
}