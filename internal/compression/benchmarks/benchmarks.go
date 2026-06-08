package benchmarks

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"agent-memory/internal/compression/algorithm"
	"agent-memory/internal/compression/radix"
	"agent-memory/internal/compression/smart"
	"agent-memory/internal/llm"
)

type Config struct {
	Corpus          string   `json:"corpus"`
	Samples         []string `json:"samples,omitempty"`
	Algorithms      []string `json:"algorithms,omitempty"`
	Iterations      int      `json:"iterations,omitempty"`
	Warmup          int      `json:"warmup,omitempty"`
	MinRetention    float64  `json:"min_retention,omitempty"`
	IncludeExamples bool     `json:"include_examples,omitempty"`
}

type Result struct {
	Corpus           string            `json:"corpus"`
	Algorithms       []AlgorithmResult `json:"algorithms"`
	Winner           string            `json:"winner"`
	WinnerReason     string            `json:"winner_reason"`
	SampleCount      int               `json:"sample_count"`
	Iterations       int               `json:"iterations"`
	MinRetention     float64           `json:"min_retention"`
	Evaluator        string            `json:"evaluator"`
	Timestamp        string            `json:"timestamp"`
	AvailableCorpora []string          `json:"available_corpora"`
}

type AlgorithmResult struct {
	Name                 string          `json:"name"`
	Method               string          `json:"method"`
	Reversible           bool            `json:"reversible"`
	SampleCount          int             `json:"sample_count"`
	Iterations           int             `json:"iterations"`
	AvgOriginalBytes     float64         `json:"avg_original_bytes"`
	AvgCompressedBytes   float64         `json:"avg_compressed_bytes"`
	AvgReduction         float64         `json:"avg_reduction"`
	MedianReduction      float64         `json:"median_reduction"`
	P95Reduction         float64         `json:"p95_reduction"`
	AvgRetention         float64         `json:"avg_retention"`
	MinRetention         float64         `json:"min_retention"`
	AvgLatencyMs         float64         `json:"avg_latency_ms"`
	P50LatencyMs         float64         `json:"p50_latency_ms"`
	P95LatencyMs         float64         `json:"p95_latency_ms"`
	ThroughputPerSecond  float64         `json:"throughput_per_second"`
	BytesSavedTotal      int             `json:"bytes_saved_total"`
	ExpansionCount       int             `json:"expansion_count"`
	ErrorCount           int             `json:"error_count"`
	RetentionBelowTarget int             `json:"retention_below_target"`
	Examples             []ExampleResult `json:"examples,omitempty"`
}

type ExampleResult struct {
	Original      string  `json:"original"`
	Compressed    string  `json:"compressed"`
	Reduction     float64 `json:"reduction"`
	Retention     float64 `json:"retention"`
	LatencyMs     float64 `json:"latency_ms"`
	CompressedLen int     `json:"compressed_len"`
}

type compressorFunc func(context.Context, string) (compressed string, method string, reversible bool, err error)

type Runner struct {
	llmClient llm.Provider
	corpora   map[string][]string
}

func NewRunner(llmClient llm.Provider) *Runner {
	return &Runner{
		llmClient: llmClient,
		corpora:   defaultCorpora(),
	}
}

func (r *Runner) Corpora() []string {
	names := make([]string, 0, len(r.corpora))
	for name := range r.corpora {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func (r *Runner) Run(ctx context.Context, cfg Config) (*Result, error) {
	cfg = normalizeConfig(cfg)
	samples, corpusName, err := r.samplesFor(cfg)
	if err != nil {
		return nil, err
	}
	algorithms := r.buildAlgorithms(samples, cfg.Algorithms)
	if len(algorithms) == 0 {
		return nil, fmt.Errorf("no compression algorithms selected")
	}

	results := make([]AlgorithmResult, 0, len(algorithms))
	for _, alg := range algorithms {
		result := runAlgorithm(ctx, alg.name, alg.fn, samples, cfg)
		results = append(results, result)
		if alg.close != nil {
			alg.close()
		}
	}
	sort.SliceStable(results, func(i, j int) bool {
		return score(results[i], cfg.MinRetention) > score(results[j], cfg.MinRetention)
	})

	winner, reason := pickWinner(results, cfg.MinRetention)
	return &Result{
		Corpus:           corpusName,
		Algorithms:       results,
		Winner:           winner,
		WinnerReason:     reason,
		SampleCount:      len(samples),
		Iterations:       cfg.Iterations,
		MinRetention:     cfg.MinRetention,
		Evaluator:        "lexical-retention+byte-size",
		Timestamp:        time.Now().Format(time.RFC3339),
		AvailableCorpora: r.Corpora(),
	}, nil
}

type namedAlgorithm struct {
	name  string
	fn    compressorFunc
	close func()
}

func (r *Runner) buildAlgorithms(samples []string, names []string) []namedAlgorithm {
	nameSet := map[string]bool{}
	for _, name := range names {
		nameSet[strings.ToLower(strings.TrimSpace(name))] = true
	}
	include := func(name string) bool {
		return len(nameSet) == 0 || nameSet[name]
	}

	var algorithms []namedAlgorithm
	if include("radix") {
		rc := radix.NewMemoryCompressor()
		rc.LearnFromMemories(samples)
		algorithms = append(algorithms, namedAlgorithm{name: "radix", fn: func(ctx context.Context, text string) (string, string, bool, error) {
			return rc.Compress(text), "radix", false, nil
		}})
	}
	if include("smart_radix") {
		sc := smart.NewSmartCompressor(r.llmClient, 1)
		sc.LearnPatterns(samples)
		algorithms = append(algorithms, namedAlgorithm{name: "smart_radix", fn: func(ctx context.Context, text string) (string, string, bool, error) {
			compressed, _, err := sc.Compress(ctx, text, smart.ModeRadix)
			return compressed, "smart_radix", false, err
		}, close: sc.Stop})
	}
	if include("smart_hybrid") {
		sc := smart.NewSmartCompressor(r.llmClient, 1)
		sc.LearnPatterns(samples)
		algorithms = append(algorithms, namedAlgorithm{name: "smart_hybrid", fn: func(ctx context.Context, text string) (string, string, bool, error) {
			compressed, _, err := sc.Compress(ctx, text, smart.ModeHybrid)
			return compressed, "smart_hybrid", false, err
		}, close: sc.Stop})
	}
	if include("real_best") {
		real := algorithm.NewRealCompressor()
		real.LearnFromMemories(samples)
		algorithms = append(algorithms, namedAlgorithm{name: "real_best", fn: func(ctx context.Context, text string) (string, string, bool, error) {
			result, err := real.Compress(text)
			if err != nil {
				return "", "real_best", false, err
			}
			reversible := false
			if result.Method != "" {
				decompressed, err := real.Decompress(result.Compressed, result.Method)
				reversible = err == nil && decompressed == text
			}
			return result.Compressed, result.Method, reversible, nil
		}})
	}
	if include("gzip") {
		algorithms = append(algorithms, namedAlgorithm{name: "gzip", fn: func(ctx context.Context, text string) (string, string, bool, error) {
			var buf bytes.Buffer
			zw := gzip.NewWriter(&buf)
			if _, err := zw.Write([]byte(text)); err != nil {
				_ = zw.Close()
				return "", "gzip", true, err
			}
			if err := zw.Close(); err != nil {
				return "", "gzip", true, err
			}
			return buf.String(), "gzip", true, nil
		}})
	}
	return algorithms
}

func runAlgorithm(ctx context.Context, name string, fn compressorFunc, samples []string, cfg Config) AlgorithmResult {
	result := AlgorithmResult{
		Name:         name,
		MinRetention: 1,
		SampleCount:  len(samples),
		Iterations:   cfg.Iterations,
	}
	var reductions []float64
	var latencies []float64
	var totalOriginal, totalCompressed, totalLatency float64

	for i := 0; i < cfg.Warmup; i++ {
		for _, sample := range samples {
			_, _, _, _ = fn(ctx, sample)
		}
	}

	for i := 0; i < cfg.Iterations; i++ {
		for sampleIndex, sample := range samples {
			start := time.Now()
			compressed, method, reversible, err := fn(ctx, sample)
			latencyMs := float64(time.Since(start).Microseconds()) / 1000.0
			if result.Method == "" && method != "" {
				result.Method = method
			}
			if reversible {
				result.Reversible = true
			}
			if err != nil {
				result.ErrorCount++
				continue
			}

			originalBytes := len([]byte(sample))
			compressedBytes := len([]byte(compressed))
			reduction := 0.0
			if originalBytes > 0 {
				reduction = 1 - float64(compressedBytes)/float64(originalBytes)
			}
			retention := lexicalRetention(sample, compressed)
			if reversible {
				retention = 1
			}

			totalOriginal += float64(originalBytes)
			totalCompressed += float64(compressedBytes)
			totalLatency += latencyMs
			reductions = append(reductions, reduction)
			latencies = append(latencies, latencyMs)
			result.BytesSavedTotal += originalBytes - compressedBytes
			if compressedBytes > originalBytes {
				result.ExpansionCount++
			}
			result.AvgRetention += retention
			if retention < result.MinRetention {
				result.MinRetention = retention
			}
			if retention < cfg.MinRetention {
				result.RetentionBelowTarget++
			}
			if cfg.IncludeExamples && i == 0 && sampleIndex < 3 {
				compressedPreview := preview(compressed, 500)
				if name == "gzip" {
					compressedPreview = fmt.Sprintf("<binary gzip payload: %d bytes>", compressedBytes)
				}
				result.Examples = append(result.Examples, ExampleResult{
					Original:      sample,
					Compressed:    compressedPreview,
					Reduction:     round(reduction),
					Retention:     round(retention),
					LatencyMs:     round(latencyMs),
					CompressedLen: compressedBytes,
				})
			}
		}
	}

	ops := len(reductions)
	if ops == 0 {
		result.MinRetention = 0
		return result
	}
	result.AvgOriginalBytes = round(totalOriginal / float64(ops))
	result.AvgCompressedBytes = round(totalCompressed / float64(ops))
	result.AvgReduction = round(average(reductions))
	result.MedianReduction = round(percentile(reductions, 50))
	result.P95Reduction = round(percentile(reductions, 95))
	result.AvgRetention = round(result.AvgRetention / float64(ops))
	result.MinRetention = round(result.MinRetention)
	result.AvgLatencyMs = round(totalLatency / float64(ops))
	result.P50LatencyMs = round(percentile(latencies, 50))
	result.P95LatencyMs = round(percentile(latencies, 95))
	if totalLatency > 0 {
		result.ThroughputPerSecond = round(float64(ops) / (totalLatency / 1000.0))
	}
	return result
}

func normalizeConfig(cfg Config) Config {
	if cfg.Corpus == "" {
		cfg.Corpus = "agent_memory"
	}
	if cfg.Iterations < 1 {
		cfg.Iterations = 3
	}
	if cfg.Warmup < 0 {
		cfg.Warmup = 0
	}
	if cfg.MinRetention <= 0 {
		cfg.MinRetention = 0.9
	}
	return cfg
}

func (r *Runner) samplesFor(cfg Config) ([]string, string, error) {
	if len(cfg.Samples) > 0 {
		samples := nonEmpty(cfg.Samples)
		if len(samples) == 0 {
			return nil, "custom", fmt.Errorf("custom compression benchmark samples are empty")
		}
		return samples, "custom", nil
	}
	samples, ok := r.corpora[cfg.Corpus]
	if !ok {
		return nil, cfg.Corpus, fmt.Errorf("unknown compression benchmark corpus %q", cfg.Corpus)
	}
	return append([]string(nil), samples...), cfg.Corpus, nil
}

func nonEmpty(samples []string) []string {
	var result []string
	for _, sample := range samples {
		if strings.TrimSpace(sample) != "" {
			result = append(result, sample)
		}
	}
	return result
}

func lexicalRetention(original, compressed string) float64 {
	orig := tokenCounts(original)
	if len(orig) == 0 {
		return 1
	}
	comp := tokenCounts(compressed)
	kept := 0
	total := 0
	for token, count := range orig {
		total += count
		if c := comp[token]; c > 0 {
			kept += minInt(count, c)
		}
	}
	return float64(kept) / float64(total)
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func tokenCounts(text string) map[string]int {
	counts := map[string]int{}
	for _, token := range strings.FieldsFunc(strings.ToLower(text), func(r rune) bool {
		return !(r >= 'a' && r <= 'z' || r >= '0' && r <= '9')
	}) {
		if len(token) > 1 {
			counts[token]++
		}
	}
	return counts
}

func pickWinner(results []AlgorithmResult, minRetention float64) (string, string) {
	if len(results) == 0 {
		return "", "no algorithms executed"
	}
	winner := results[0]
	if winner.AvgRetention < minRetention {
		return winner.Name, fmt.Sprintf("highest score, but retention %.2f is below target %.2f", winner.AvgRetention, minRetention)
	}
	return winner.Name, fmt.Sprintf("best reduction %.2f with retention %.2f and p95 latency %.2fms", winner.AvgReduction, winner.AvgRetention, winner.P95LatencyMs)
}

func score(result AlgorithmResult, minRetention float64) float64 {
	retentionPenalty := 0.0
	if result.AvgRetention < minRetention {
		retentionPenalty = (minRetention - result.AvgRetention) * 2
	}
	expansionPenalty := float64(result.ExpansionCount) / math.Max(1, float64(result.SampleCount*result.Iterations))
	errorPenalty := float64(result.ErrorCount) / math.Max(1, float64(result.SampleCount*result.Iterations))
	return result.AvgReduction + result.AvgRetention - retentionPenalty - expansionPenalty - errorPenalty
}

func percentile(values []float64, p float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sorted := append([]float64(nil), values...)
	sort.Float64s(sorted)
	idx := int(float64(len(sorted)-1) * p / 100)
	return sorted[idx]
}

func average(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	var total float64
	for _, value := range values {
		total += value
	}
	return total / float64(len(values))
}

func round(value float64) float64 {
	return math.Round(value*10000) / 10000
}

func preview(value string, limit int) string {
	if len(value) <= limit {
		return value
	}
	return value[:limit]
}

func defaultCorpora() map[string][]string {
	return map[string][]string{
		"agent_memory": {
			"User prefers Python over JavaScript and is building a memory backend for AI agents with Neo4j, Qdrant, and Redis.",
			"The agent should remember project conventions, API contracts, deployment settings, error handling rules, and benchmark evidence.",
			"Enterprise tenants need audit logs, SSO provider configuration, RBAC policies, webhook delivery state, and source-attributed retrieval results.",
			"Compression must preserve facts about users, teams, skills, entities, preferences, and temporal interaction history while reducing token load.",
			"Search combines vector similarity, keyword matching, graph propagation, reranking, metadata filters, and source attribution.",
		},
		"repetitive": {
			"memory memory memory retrieval retrieval retrieval compression compression compression graph graph graph",
			"Neo4j stores entities. Neo4j stores relationships. Neo4j stores audit state. Qdrant stores vectors. Qdrant stores embeddings.",
			"OpenTelemetry metrics, OpenTelemetry traces, OpenTelemetry logs, Prometheus metrics, Grafana dashboards.",
		},
		"long_context": {
			strings.Repeat("The user is building a self-hosted graph memory platform for agents with hybrid search and enterprise controls. ", 16),
			strings.Repeat("Compression should reduce repeated context while preserving facts, source attribution, and operational instructions. ", 16),
			strings.Repeat("Benchmarks must report measured retention, token reduction, latency, errors, and corpus details before marketing claims are used. ", 12),
		},
	}
}
