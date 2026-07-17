package evaluation

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"agent-memory/internal/llm"
)

func benchmarkDebug(format string, args ...any) {
	if os.Getenv("BENCHMARK_DEBUG") == "1" {
		fmt.Printf("DEBUG: "+format+"\n", args...)
	}
}

type BenchmarkConfig struct {
	Model         string
	MaxTokens     int
	ParallelLimit int
	SearchLimit   int // max results to retrieve per search (default 10)
	Limit         int // limit number of questions
}

type BenchmarkResult struct {
	Dataset             string             `json:"dataset"`
	OverallScore        float64            `json:"overall_score"`
	SingleHopScore      float64            `json:"single_hop_score"`
	MultiHopScore       float64            `json:"multi_hop_score"`
	MemoryHitRate       float64            `json:"memory_hit_rate"`
	MRR                 float64            `json:"mrr"`
	HitAt1              float64            `json:"hit_at_1"`
	HitAt3              float64            `json:"hit_at_3"`
	HitAt5              float64            `json:"hit_at_5"`
	HitAt10             float64            `json:"hit_at_10"`
	MRRAt5              float64            `json:"mrr_at_5"`
	QACorrectness       float64            `json:"qa_correctness,omitempty"`
	QACompleteness      float64            `json:"qa_completeness,omitempty"`
	QARelevance         float64            `json:"qa_relevance,omitempty"`
	FAMAScore           float64            `json:"fama_score,omitempty"`
	PerCategoryScore    map[string]float64 `json:"per_category_score,omitempty"`
	AvgRetrievedItems   float64            `json:"avg_retrieved_items"`
	TokensRetrieved     int                `json:"tokens_retrieved"`
	MaxContextTokens    int                `json:"max_context_tokens,omitempty"`
	LatencyP50Ms        float64            `json:"latency_p50_ms"`
	LatencyP95Ms        float64            `json:"latency_p95_ms"`
	QuestionsAnswered   int                `json:"questions_answered"`
	TotalQuestions      int                `json:"total_questions"`
	MemoriesIngested    int                `json:"memories_ingested"`
	IngestErrors        int                `json:"ingest_errors"`
	SearchErrors        int                `json:"search_errors"`
	ScoredQuestions     int                `json:"scored_questions"`
	ScoringErrors       int                `json:"scoring_errors"`
	EvaluatorConfigured bool               `json:"evaluator_configured"`
	Publishable         bool               `json:"publishable"`
	ScoreMethod         string             `json:"score_method"`
	Warnings            []string           `json:"warnings,omitempty"`
	Timestamp           string             `json:"timestamp"`
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
	ID        string    `json:"id"`
	Content   string    `json:"content"`
	UserID    string    `json:"user_id"`
	CreatedAt time.Time `json:"created_at,omitempty"`
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

// QARubricResult holds structured QA evaluation dimensions.
type QARubricResult struct {
	Correctness  float64 `json:"correctness"`
	Completeness float64 `json:"completeness"`
	Relevance    float64 `json:"relevance"`
	Overall      float64 `json:"overall"`
}

func (s *Scorer) ScoreAnswer(ctx context.Context, question, answer, groundTruth string) (float64, error) {
	if s.llmClient == nil {
		// Fallback: token-level F1 when no LLM is configured
		return TokenF1Score(answer, groundTruth), nil
	}

	prompt := fmt.Sprintf(`You are evaluating an AI memory retrieval system.
Your job is to determine if the Retrieved Memory Context contains the key facts required by the Expected Fact.
The original Question is provided as context.

Question: %s
Retrieved Context: %s
Expected Fact: %s

Rate the quality of the Retrieved Context from 0-100 where:
- 100: The Retrieved Context clearly and explicitly contains the Expected Fact.
- 75: The Retrieved Context contains most of the Expected Fact but misses minor details.
- 50: The Retrieved Context is partially relevant but misses key information needed to confirm the Expected Fact.
- 25: The Retrieved Context is mostly unrelated but mentions similar entities.
- 0: The Retrieved Context is completely unrelated or missing the Expected Fact.

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

	// Filter out <think>...</think> tags if they exist
	cleanContent := regexp.MustCompile(`(?s)<think>.*?</think>`).ReplaceAllString(resp.Content, "")

	var score float64
	fmt.Sscanf(cleanContent, "%f", &score)
	if score == 0 {
		reNum := regexp.MustCompile(`([0-9]{1,3}(?:\.[0-9]+)?)`)
		if m := reNum.FindStringSubmatch(cleanContent); len(m) > 1 {
			score, _ = strconv.ParseFloat(m[1], 64)
		}
	}

	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}
	return score / 100.0, nil
}

// ScoreAnswerRubric performs structured multi-dimension QA evaluation.
// Returns correctness, completeness, relevance as separate 0-1 scores.
// Falls back to TokenF1Score when no LLM is configured.
func (s *Scorer) ScoreAnswerRubric(ctx context.Context, question, answer, groundTruth string) (*QARubricResult, error) {
	if s.llmClient == nil {
		f1 := TokenF1Score(answer, groundTruth)
		return &QARubricResult{
			Correctness:  f1,
			Completeness: f1,
			Relevance:    f1,
			Overall:      f1,
		}, nil
	}

	prompt := fmt.Sprintf(`You are evaluating an AI memory retrieval system on three dimensions.
Your job is to determine if the Retrieved Context contains the information needed to confirm the Expected Fact.

Question: %s
Retrieved Context: %s
Expected Fact: %s

Score each dimension from 0-100:
1. Correctness: Does the retrieved context explicitly contain the true information from the expected fact? (Ignore extra unrelated chatter in the context).
2. Completeness: Does the retrieved context cover all key points of the expected fact?
3. Relevance: Does the retrieved context actually help answer the original question?

CRITICAL: Return ONLY valid JSON. Do not include any markdown, explanations, or conversational text. Your response must start with { and end with }.
Example: {"correctness": 85, "completeness": 90, "relevance": 95, "overall": 90}`,
		question, answer, groundTruth)

	resp, err := s.llmClient.Complete(ctx, &llm.CompletionRequest{
		Model:       s.config.Model,
		Messages:    []llm.Message{{Role: "user", Content: prompt}},
		Temperature: 0.1,
		MaxTokens:   s.config.MaxTokens,
	})
	if err != nil {
		return nil, err
	}
	
	// Filter out <think>...</think> tags if they exist
	cleanContent := regexp.MustCompile(`(?s)<think>.*?</think>`).ReplaceAllString(resp.Content, "")
	
	benchmarkDebug("Retrieved Context for %q:\n%s", question, answer)

	var rubric QARubricResult
	content := strings.ToLower(cleanContent)

	extractScore := func(key string) float64 {
		re := regexp.MustCompile(fmt.Sprintf(`"%s"\s*:\s*"?([0-9.]+)"?`, key))
		matches := re.FindStringSubmatch(content)
		if len(matches) > 1 {
			if val, err := strconv.ParseFloat(matches[1], 64); err == nil {
				return val
			}
		}
		// Try to find the number even without quotes, just in case
		reBackup := regexp.MustCompile(fmt.Sprintf(`%s[^0-9]+([0-9.]+)`, key))
		matches = reBackup.FindStringSubmatch(content)
		if len(matches) > 1 {
			if val, err := strconv.ParseFloat(matches[1], 64); err == nil {
				return val
			}
		}
		return 0.0
	}

	rubric.Correctness = extractScore("correctness")
	rubric.Completeness = extractScore("completeness")
	rubric.Relevance = extractScore("relevance")
	rubric.Overall = extractScore("overall")

	// If everything is 0 but we found numbers, it might be a format issue.
	if rubric.Overall == 0 && rubric.Correctness == 0 && rubric.Completeness == 0 && rubric.Relevance == 0 {
		var score float64
		fmt.Sscanf(resp.Content, "%f", &score)
		if score == 0 {
			reNum := regexp.MustCompile(`([0-9]{1,3}(?:\.[0-9]+)?)`)
			if m := reNum.FindStringSubmatch(resp.Content); len(m) > 1 {
				score, _ = strconv.ParseFloat(m[1], 64)
			}
		}
		rubric.Overall = score
		rubric.Correctness = score
		rubric.Completeness = score
		rubric.Relevance = score
	}

	// Auto-scale if the LLM decided to use a 0-1 or 0-10 scale instead of 0-100
	maxScore := rubric.Overall
	if rubric.Correctness > maxScore { maxScore = rubric.Correctness }
	if rubric.Completeness > maxScore { maxScore = rubric.Completeness }
	if rubric.Relevance > maxScore { maxScore = rubric.Relevance }
	
	if maxScore > 0 && maxScore <= 1.0 {
		rubric.Correctness *= 100.0
		rubric.Completeness *= 100.0
		rubric.Relevance *= 100.0
		rubric.Overall *= 100.0
	} else if maxScore > 1.0 && maxScore <= 10.0 {
		rubric.Correctness *= 10.0
		rubric.Completeness *= 10.0
		rubric.Relevance *= 10.0
		rubric.Overall *= 10.0
	}

	rubric.Correctness = clampScore(rubric.Correctness) / 100.0
	rubric.Completeness = clampScore(rubric.Completeness) / 100.0
	rubric.Relevance = clampScore(rubric.Relevance) / 100.0
	rubric.Overall = clampScore(rubric.Overall) / 100.0

	return &rubric, nil
}

func clampScore(s float64) float64 {
	if s < 0 {
		return 0
	}
	if s > 100 {
		return 100
	}
	return s
}

// TokenF1Score computes token-level F1 between predicted and ground truth text.
// Used as a fallback when no LLM judge is configured.
func TokenF1Score(predicted, groundTruth string) float64 {
	predTokens := tokenize(strings.ToLower(predicted))
	truthTokens := tokenize(strings.ToLower(groundTruth))

	if len(predTokens) == 0 && len(truthTokens) == 0 {
		return 1.0
	}
	if len(predTokens) == 0 || len(truthTokens) == 0 {
		return 0.0
	}

	// Count overlapping tokens
	common := 0
	truthSet := make(map[string]int, len(truthTokens))
	for _, t := range truthTokens {
		truthSet[t]++
	}
	for _, t := range predTokens {
		if truthSet[t] > 0 {
			common++
			truthSet[t]--
		}
	}

	if common == 0 {
		return 0.0
	}

	precision := float64(common) / float64(len(predTokens))
	recall := float64(common) / float64(len(truthTokens))
	return 2 * precision * recall / (precision + recall)
}

// tokenize splits text into word tokens, stripping punctuation.
func tokenize(text string) []string {
	text = strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == ' ' {
			return r
		}
		if r >= 'A' && r <= 'Z' {
			return r + 32 // lowercase
		}
		return ' '
	}, text)
	words := strings.Fields(text)
	// Filter stop words
	filtered := make([]string, 0, len(words))
	for _, w := range words {
		if !isStopWord(w) {
			filtered = append(filtered, w)
		}
	}
	return filtered
}

func isStopWord(w string) bool {
	switch w {
	case "the", "a", "an", "is", "are", "was", "were", "be", "been",
		"being", "have", "has", "had", "do", "does", "did", "will",
		"would", "could", "should", "may", "might", "shall", "can",
		"to", "of", "in", "for", "on", "with", "at", "by", "from",
		"as", "into", "through", "during", "before", "after", "and",
		"but", "or", "not", "no", "it", "its", "this", "that":
		return true
	}
	return false
}

type BenchmarkRunner struct {
	scorer  *Scorer
	config  BenchmarkConfig
	results []BenchmarkResult
	mu      sync.Mutex
}

func NewBenchmarkRunner(scorer *Scorer, config BenchmarkConfig) *BenchmarkRunner {
	if config.SearchLimit <= 0 {
		config.SearchLimit = 10
	}
	return &BenchmarkRunner{
		scorer:  scorer,
		config:  config,
		results: make([]BenchmarkResult, 0),
	}
}

// SearchLimit returns the configured max results per search query.
func (r *BenchmarkRunner) SearchLimit() int {
	if r.config.SearchLimit <= 0 {
		return 10
	}
	return r.config.SearchLimit
}

func (r *BenchmarkRunner) LoadDataset(name string) (*BenchmarkDataset, error) {
	paths := []string{
		// Prefer converted official datasets under data/benchmarks/
		filepath.Join("data", "benchmarks", name, "dataset.json"),
		filepath.Join("data", "benchmarks", name, "dataset.full.json"),
		// Fall back to packaged fixtures under internal/evaluation/
		filepath.Join("internal", "evaluation", name, "dataset.json"),
		filepath.Join("evaluation", name, "dataset.json"),
	}
	if base := os.Getenv("BENCHMARK_DATASET_PATH"); base != "" {
		paths = append([]string{filepath.Join(base, name, "dataset.json")}, paths...)
	}
	if strings.HasPrefix(name, "beam_") {
		scale := strings.TrimPrefix(name, "beam_")
		// Scale-specific files must be tried before the shared beam_1m fallback.
		paths = append(paths,
			filepath.Join("internal", "evaluation", "beam", "beam_"+scale+"_dataset.json"),
			filepath.Join("evaluation", "beam", "beam_"+scale+"_dataset.json"),
			filepath.Join("internal", "evaluation", "beam", "dataset.json"),
			filepath.Join("evaluation", "beam", "dataset.json"),
			filepath.Join("data", "benchmarks", "beam", "dataset.json"),
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
	QuestionID   string
	Score        float64
	Correctness  float64
	Completeness float64
	Relevance    float64
	Latency      time.Duration
	Tokens       int
	Category     string
	SearchErr    error
	ScoreErr     error
	Scored       bool
	ExpectedID   string
	HitRank      int
	Retrieved    int
	Ingested     bool
	IngestErr    error
	MemoryAge    time.Duration // age of the expected memory at query time
}

func (r *BenchmarkRunner) runBenchmark(ctx context.Context, dataset *BenchmarkDataset, memSvc MemoryService, searchFn SearchFunc) []questionResult {
	if cleanupSvc, ok := memSvc.(interface{ CleanupBenchmarkMemories(context.Context) error }); ok {
		_ = cleanupSvc.CleanupBenchmarkMemories(ctx)
	}

	if r.config.Limit > 0 {
		if len(dataset.Questions) > r.config.Limit {
			dataset.Questions = dataset.Questions[:r.config.Limit]
		}
		if len(dataset.Memories) > r.config.Limit {
			dataset.Memories = dataset.Memories[:r.config.Limit]
		}
	}

	// Track when each memory was ingested for FAMA computation
	memoryIngestTime := make(map[string]time.Time, len(dataset.Memories))

	results := make([]questionResult, 0, len(dataset.Questions))
	for _, mem := range dataset.Memories {
		userID := mem.UserID
		if userID == "" {
			userID = "benchmark-user"
		}

		// Split long conversation memories into per-turn chunks so each chunk
		// gets a focused embedding. This is critical for LongMemEval where a
		// single memory can contain 10+ conversation turns totalling thousands
		// of tokens — a single embedding for that would dilute all the facts.
		chunks := chunkConversationMemory(mem.Content, mem.ID)

		var ingestErr error
		for i, chunk := range chunks {
			chunkMem := BenchmarkMemory{
				ID:      fmt.Sprintf("%s_c%d", mem.ID, i),
				Content: chunk,
				UserID:  userID,
			}
			var err error
			if labeledSvc, ok := memSvc.(LabeledMemoryService); ok {
				_, err = labeledSvc.CreateBenchmarkMemory(ctx, chunkMem)
			} else {
				_, err = memSvc.CreateMemory(ctx, chunk, userID)
			}
			if err != nil {
				ingestErr = err
				benchmarkDebug("Ingest Error for memory %s chunk %d: %v", mem.ID, i, err)
				break
			}
		}
		if ingestErr != nil {
			results = append(results, questionResult{
				QuestionID: mem.ID,
				IngestErr:  fmt.Errorf("ingest memory: %w", ingestErr),
			})
			continue
		}
		memoryIngestTime[mem.ID] = time.Now()
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
			var correctness, completeness, relevance float64
			var scoreErr error
			scored := false
			if question.GroundTruth != "" && r.scorer != nil && r.scorer.llmClient != nil {
				rubric, rubricErr := r.scorer.ScoreAnswerRubric(ctx, question.Question, answer, question.GroundTruth)
				if rubricErr == nil && rubric != nil {
					score = rubric.Overall
					correctness = rubric.Correctness
					completeness = rubric.Completeness
					relevance = rubric.Relevance
					scored = true
				} else {
					if rubricErr != nil {
						benchmarkDebug("ScoreAnswerRubric failed: %v", rubricErr)
					}
					// Fallback to simple scoring
					score, scoreErr = r.scorer.ScoreAnswer(ctx, question.Question, answer, question.GroundTruth)
					scored = scoreErr == nil
					correctness = score
					completeness = score
					relevance = score
				}
			} else if question.GroundTruth != "" && r.scorer != nil {
				// No LLM: use token F1
				score = TokenF1Score(answer, question.GroundTruth)
				correctness = score
				completeness = score
				relevance = score
				scored = true
			}
			if scored {
				benchmarkDebug("Question %s\n - Query: %q\n - Retrieved: %q\n - Expected: %q\n - Score: %.2f", question.ID, question.Question, answer, question.GroundTruth, score)
			} else if scoreErr != nil {
				benchmarkDebug("Question %s - Scoring Error: %v", question.ID, scoreErr)
			}
			hitRank := hitRank(memoryResults, question.MemoryID)

			// Compute memory age for FAMA scoring
			var memoryAge time.Duration
			if ingestTime, ok := memoryIngestTime[question.MemoryID]; ok {
				memoryAge = time.Since(ingestTime)
			}

			mu.Lock()
			results = append(results, questionResult{
				QuestionID:   question.ID,
				Score:        score,
				Correctness:  correctness,
				Completeness: completeness,
				Relevance:    relevance,
				Latency:      latency,
				Tokens:       len(answer) / 4,
				Category:     question.Category,
				SearchErr:    err,
				ScoreErr:     scoreErr,
				Scored:       scored,
				ExpectedID:   question.MemoryID,
				HitRank:      hitRank,
				Retrieved:    len(memoryResults),
				MemoryAge:    memoryAge,
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
	var reciprocalRankTotal, reciprocalRankAt5Total float64
	var hitAt1Count, hitAt3Count, hitAt5Count, hitAt10Count int
	var latencies []float64
	var totalTokens int
	var totalCorrectness, totalCompleteness, totalRelevance float64

	// Per-category score tracking
	categoryScores := make(map[string]float64)
	categoryCounts := make(map[string]int)

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
				// Hit@K counting
				if qr.HitRank <= 1 {
					hitAt1Count++
				}
				if qr.HitRank <= 3 {
					hitAt3Count++
				}
				if qr.HitRank <= 5 {
					hitAt5Count++
					reciprocalRankAt5Total += 1 / float64(qr.HitRank)
				}
				if qr.HitRank <= 10 {
					hitAt10Count++
				}
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
		totalCorrectness += qr.Correctness
		totalCompleteness += qr.Completeness
		totalRelevance += qr.Relevance
		latencies = append(latencies, qr.Latency.Seconds()*1000)
		totalTokens += qr.Tokens

		switch qr.Category {
		case "single_hop", "user", "knowledge_update", "multi_session", "old", "new":
			singleHopScore += qr.Score
			singleHopCount++
		case "multi_hop", "temporal", "temporal-reasoning", "temporal_reasoning":
			multiHopScore += qr.Score
			multiHopCount++
		}

		// Per-category tracking (all categories)
		if qr.Category != "" {
			categoryScores[qr.Category] += qr.Score
			categoryCounts[qr.Category]++
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
		result.QACorrectness = totalCorrectness / float64(scoredCount)
		result.QACompleteness = totalCompleteness / float64(scoredCount)
		result.QARelevance = totalRelevance / float64(scoredCount)
		if r.scorer != nil && r.scorer.llmClient != nil {
			result.ScoreMethod = "llm_judge"
		} else {
			result.ScoreMethod = "token_f1"
		}
	} else if questionCount > 0 {
		result.Warnings = append(result.Warnings, "no questions were scored; configure an evaluator LLM before publishing benchmark numbers")
	}
	if questionCount > 0 {
		result.AvgRetrievedItems = float64(retrievedTotal) / float64(questionCount)
	}
	if expectedIDCount > 0 {
		result.MemoryHitRate = float64(memoryHits) / float64(expectedIDCount)
		result.MRR = reciprocalRankTotal / float64(expectedIDCount)
		result.HitAt1 = float64(hitAt1Count) / float64(expectedIDCount)
		result.HitAt3 = float64(hitAt3Count) / float64(expectedIDCount)
		result.HitAt5 = float64(hitAt5Count) / float64(expectedIDCount)
		result.HitAt10 = float64(hitAt10Count) / float64(expectedIDCount)
		result.MRRAt5 = reciprocalRankAt5Total / float64(expectedIDCount)
	} else {
		result.Warnings = append(result.Warnings, "dataset has no expected memory IDs; memory_hit_rate and mrr cannot be computed")
	}

	// Per-category score breakdown
	if len(categoryCounts) > 0 {
		result.PerCategoryScore = make(map[string]float64, len(categoryCounts))
		for cat, total := range categoryScores {
			if categoryCounts[cat] > 0 {
				result.PerCategoryScore[cat] = total / float64(categoryCounts[cat])
			}
		}
	}

	// FAMA (Forgetting-Aware Memory Accuracy) scoring
	var famaInputs []FAMAInput
	for _, qr := range qResults {
		if qr.Scored && qr.ExpectedID != "" && qr.HitRank > 0 {
			famaInputs = append(famaInputs, FAMAInput{
				Correct:        qr.Score >= 0.5,
				MemoryAge:      qr.MemoryAge,
				ValidityWindow: 7 * 24 * time.Hour, // default 7-day validity window
			})
		}
	}
	if len(famaInputs) > 0 {
		famaScorer := NewFAMAScorer()
		result.FAMAScore = famaScorer.BatchScore(famaInputs)
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

// hitRank returns the 1-based rank of the first result that matches expectedID.
// It also matches chunk IDs produced by chunkConversationMemory, i.e.
// a result with ID "mem_0_c3" is counted as a hit for expectedID "mem_0".
func hitRank(results []MemoryResult, expectedID string) int {
	if expectedID == "" {
		return 0
	}
	chunkPrefix := expectedID + "_c"
	for idx, result := range results {
		if result.ID == expectedID || strings.HasPrefix(result.ID, chunkPrefix) {
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

// chunkConversationMemory splits a long multi-turn conversation into per-turn
// chunks so each chunk gets a focused embedding. For short memories (< 500
// chars) it returns the content unchanged as a single chunk.
//
// LongMemEval memories are entire conversations like:
//
//	"user: I got my car serviced...\nassistant: ...\nuser: Also, the GPS broke...\nassistant: ..."
//
// Embedding this whole thing as one vector buries every individual fact.
// Splitting by turn lets the vector search pinpoint "GPS broke on 3/22".
func chunkConversationMemory(content, _ string) []string {
	const minChunkLen = 50   // skip trivially short fragments
	const maxChunkLen = 1500 // ~300-400 tokens, safe for Qwen3-Embedding-4B

	// If short enough, return as-is.
	if len(content) <= maxChunkLen {
		return []string{content}
	}

	// Split on conversation turn boundaries.
	var chunks []string
	var current strings.Builder

	flushCurrent := func() {
		s := strings.TrimSpace(current.String())
		if len(s) >= minChunkLen {
			chunks = append(chunks, s)
		}
		current.Reset()
	}

	lines := strings.Split(content, "\n")
	for _, line := range lines {
		isTurnStart := strings.HasPrefix(line, "user:") || strings.HasPrefix(line, "assistant:")
		if isTurnStart && current.Len() > 0 {
			// If the accumulated turn is already large, flush before adding next.
			if current.Len() >= maxChunkLen {
				flushCurrent()
			}
		}
		current.WriteString(line)
		current.WriteByte('\n')

		// Flush if current chunk has grown large enough.
		if current.Len() >= maxChunkLen {
			flushCurrent()
		}
	}
	flushCurrent()

	if len(chunks) == 0 {
		return []string{content}
	}
	return chunks
}

