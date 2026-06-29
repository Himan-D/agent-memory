package improve

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

// Triplet is a (subject, predicate, object) triple extracted from a
// DistilledLesson statement. Used for graph-enrichment queries that need
// relational facts rather than free-form text.
type Triplet struct {
	Subject   string
	Predicate string
	Object    string
	Source    string // lesson id (for traceability)
	Confidence float64
}

// graphExec is reused from feedback_weights.go (same package).

// tripletIndexer is the minimal vector-store surface used by
// MemifyEnrichment. Defined as an interface so tests can inject a fake
// without spinning up Qdrant.
type tripletIndexer interface {
	IndexTriplets(ctx context.Context, triplets []Triplet) error
}

// MemifyEnrichment generates (subject, predicate, object) triplets from
// DistilledLesson statements and indexes them in Qdrant. This is stage 4
// of the improvement pipeline: lessons become queryable as relational
// facts so future retrievers can answer "X relates to Y" questions.
//
// The default generator is a heuristic that extracts capitalized noun
// phrases from the statement. Callers wanting higher-quality triplets
// should provide a Generator backed by an LLM call.
type MemifyEnrichment struct {
	Graph     graphExec
	Vector    tripletIndexer
	Generator TripletGenerator
	// BatchSize caps how many lessons are processed per query. Zero uses
	// default (200).
	BatchSize int
}

// TripletGenerator converts a lesson statement into one or more
// Triplets. Implementations may use any strategy (heuristic, LLM, NER).
type TripletGenerator interface {
	Generate(ctx context.Context, statement string, source string) ([]Triplet, error)
}

// HeuristicTripletGenerator produces triplets by pairing capitalized noun
// phrases in the statement. Quality is lower than an LLM-backed generator
// but the implementation is deterministic and free of network calls.
type HeuristicTripletGenerator struct {
	// Predicate is used when the statement has no obvious relation
	// indicator. Default: "is_related_to".
	Predicate string
}

// Generate implements TripletGenerator.
func (h HeuristicTripletGenerator) Generate(_ context.Context, statement, source string) ([]Triplet, error) {
	predicate := h.Predicate
	if predicate == "" {
		predicate = "is_related_to"
	}
	phrases := extractCapitalizedPhrases(statement)
	if len(phrases) < 2 {
		return nil, nil
	}
	// Pair the first phrase with each subsequent phrase. This is a
	// conservative heuristic -- an LLM-backed generator would produce
	// higher-quality relational triplets.
	out := make([]Triplet, 0, len(phrases)-1)
	for i := 1; i < len(phrases); i++ {
		out = append(out, Triplet{
			Subject:    phrases[0],
			Predicate:  predicate,
			Object:     phrases[i],
			Source:     source,
			Confidence: 0.5,
		})
	}
	return out, nil
}

// WithVectorIndexer enables triplet vector indexing. Pass nil to disable
// (the stage will return an error but not abort the pipeline).
func (s *MemifyEnrichment) WithVectorIndexer(v tripletIndexer) *MemifyEnrichment {
	s.Vector = v
	return s
}

// Name implements Stage.
func (s MemifyEnrichment) Name() string { return "memify_enrichment" }

// Run implements Stage.
func (s MemifyEnrichment) Run(ctx context.Context, in Input) (StageResult, error) {
	start := time.Now().UTC()
	res := StageResult{Name: s.Name(), Started: start}

	if s.Graph == nil {
		res.Ended = time.Now().UTC()
		return res, fmt.Errorf("memify_enrichment: graph executor is nil")
	}
	if s.Generator == nil {
		s.Generator = HeuristicTripletGenerator{}
	}
	if s.Vector == nil {
		// Triplet indexing is best-effort: log and continue without
		// failing the whole stage.
		res.Ended = time.Now().UTC()
		return res, fmt.Errorf("memify_enrichment: vector indexer is nil")
	}

	batchSize := s.BatchSize
	if batchSize <= 0 {
		batchSize = 200
	}

	cypher := `
MATCH (l:DistilledLesson)
WHERE l.user_id = $user_id OR $user_id = ""
RETURN l.id AS id, l.statement AS statement
LIMIT $limit
`
	rows, err := s.Graph.QueryGraph(cypher, map[string]interface{}{
		"user_id": in.UserID,
		"limit":   int64(batchSize),
	})
	if err != nil {
		res.Ended = time.Now().UTC()
		return res, fmt.Errorf("memify_enrichment: query lessons: %w", err)
	}

	var allTriplets []Triplet
	for _, r := range rows {
		id := asString(r["id"])
		stmt := asString(r["statement"])
		if stmt == "" || id == "" {
			continue
		}
		tr, err := s.Generator.Generate(ctx, stmt, id)
		if err != nil {
			res.Ended = time.Now().UTC()
			return res, fmt.Errorf("memify_enrichment: generate for %s: %w", id, err)
		}
		allTriplets = append(allTriplets, tr...)
	}

	indexed := 0
	if len(allTriplets) > 0 {
		if err := s.Vector.IndexTriplets(ctx, allTriplets); err != nil {
			res.Ended = time.Now().UTC()
			return res, fmt.Errorf("memify_enrichment: index: %w", err)
		}
		indexed = len(allTriplets)
	}

	res.Items = indexed
	res.Ended = time.Now().UTC()
	return res, nil
}

// extractCapitalizedPhrases returns the runs of consecutive capitalized
// words in s. It is shared with the entity extractor in
// internal/session/writer_neo4j.go but lives here so the improve package
// has no compile-time dependency on the session package internals.
func extractCapitalizedPhrases(s string) []string {
	out := []string{}
	words := strings.Fields(s)
	current := []string{}
	for _, w := range words {
		clean := strings.TrimFunc(w, func(r rune) bool {
			return !(r >= 'A' && r <= 'Z') && !(r >= 'a' && r <= 'z') && !(r >= '0' && r <= '9')
		})
		if clean == "" {
			flush(&current, &out)
			continue
		}
		if clean[0] >= 'A' && clean[0] <= 'Z' {
			current = append(current, clean)
		} else {
			flush(&current, &out)
		}
	}
	flush(&current, &out)
	return out
}

func flush(current *[]string, out *[]string) {
	if len(*current) > 0 {
		*out = append(*out, strings.Join(*current, " "))
		*current = (*current)[:0]
	}
}

// asString is shared with internal/session/store_neo4j.go (same package).
func asString(v interface{}) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

// Ensure errors is referenced even if the file evolves.
var _ = errors.New
