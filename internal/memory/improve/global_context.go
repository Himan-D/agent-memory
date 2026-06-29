package improve

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// GlobalContextIndex builds per-tenant summaries from DistilledLesson
// statements and stores them as GlobalContext nodes in Neo4j. This is
// stage 5 of the improvement pipeline: lessons aggregate into a
// retrieval-ready "current themes" prelude so the next session starts
// with high-level context without paying the cost of graph traversal.
//
// Node shape:
//
//   (:GlobalContext {
//     id: string,
//     tenant_id: string,
//     dataset_id: string,
//     summary: string,         // concatenated top-K lesson statements
//     entity_glossary: []string,
//     created_at: datetime,
//     updated_at: datetime
//   })
type GlobalContextIndex struct {
	Graph    graphExec
	TenantID string
	// DatasetID, when set, scopes the index to a specific dataset.
	DatasetID string
	// TopK is the number of recent lessons to include in the summary.
	// Zero means default (50).
	TopK int
	// MaxSummaryChars truncates the summary to keep it LLM-friendly.
	// Zero means default (4000).
	MaxSummaryChars int
}

// Name implements Stage.
func (s GlobalContextIndex) Name() string { return "global_context_index" }

// Run implements Stage.
func (s GlobalContextIndex) Run(ctx context.Context, in Input) (StageResult, error) {
	start := time.Now().UTC()
	res := StageResult{Name: s.Name(), Started: start}

	if s.Graph == nil {
		res.Ended = time.Now().UTC()
		return res, fmt.Errorf("global_context_index: graph executor is nil")
	}

	tenant := s.TenantID
	if tenant == "" {
		tenant = in.UserID // fallback: use user_id as tenant scope
	}
	if tenant == "" {
		res.Ended = time.Now().UTC()
		return res, fmt.Errorf("global_context_index: tenant_id is required")
	}

	topK := s.TopK
	if topK <= 0 {
		topK = 50
	}
	maxChars := s.MaxSummaryChars
	if maxChars <= 0 {
		maxChars = 4000
	}

	// Fetch recent lessons for the tenant.
	cypher := `
MATCH (l:DistilledLesson)
WHERE l.user_id = $tenant OR $tenant = ""
RETURN l.id AS id, l.statement AS statement, l.entities AS entities,
       l.distilled_on AS distilled_on
ORDER BY l.distilled_on DESC
LIMIT $limit
`
	rows, err := s.Graph.QueryGraph(cypher, map[string]interface{}{
		"tenant": tenant,
		"limit":  int64(topK),
	})
	if err != nil {
		res.Ended = time.Now().UTC()
		return res, fmt.Errorf("global_context_index: query lessons: %w", err)
	}

	summary, glossary := buildSummary(rows, maxChars)

	id := "gc:" + tenant
	if s.DatasetID != "" {
		id = "gc:" + tenant + ":" + s.DatasetID
	}

	writeCypher := `
MERGE (g:GlobalContext {id: $id})
  ON CREATE SET g.tenant_id      = $tenant,
                g.dataset_id     = $dataset_id,
                g.summary        = $summary,
                g.entity_glossary = $glossary,
                g.created_at     = datetime($now),
                g.updated_at     = datetime($now)
  ON MATCH  SET g.summary        = $summary,
                g.entity_glossary = $glossary,
                g.updated_at     = datetime($now)
`
	_, err = s.Graph.QueryGraph(writeCypher, map[string]interface{}{
		"id":         id,
		"tenant":     tenant,
		"dataset_id": s.DatasetID,
		"summary":    summary,
		"glossary":   glossary,
		"now":        time.Now().UTC().Format(time.RFC3339Nano),
	})
	if err != nil {
		res.Ended = time.Now().UTC()
		return res, fmt.Errorf("global_context_index: write node: %w", err)
	}

	res.Items = len(rows)
	res.Ended = time.Now().UTC()
	return res, nil
}

// buildSummary concatenates the top statements into a single string and
// collects unique entities across them.
func buildSummary(rows []map[string]interface{}, maxChars int) (string, []string) {
	var sb strings.Builder
	entitySet := make(map[string]struct{})
	for _, r := range rows {
		stmt := asString(r["statement"])
		if stmt == "" {
			continue
		}
		if sb.Len() > 0 {
			sb.WriteString(" | ")
		}
		sb.WriteString(stmt)
		if entities, ok := r["entities"].([]interface{}); ok {
			for _, e := range entities {
				if s, ok := e.(string); ok && s != "" {
					entitySet[s] = struct{}{}
				}
			}
		}
		if sb.Len() >= maxChars {
			break
		}
	}
	summary := sb.String()
	if len(summary) > maxChars {
		summary = summary[:maxChars] + "..."
	}
	glossary := make([]string, 0, len(entitySet))
	for e := range entitySet {
		glossary = append(glossary, e)
	}
	return summary, glossary
}
