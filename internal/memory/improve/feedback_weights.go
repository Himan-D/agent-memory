package improve

import (
	"context"
	"fmt"
	"time"
)

// graphExec is the minimal graph-store surface used by FeedbackWeights.
// Defined here (not imported from internal/memory/neo4j) so tests can
// inject a fake without spinning up Neo4j, and so production code can
// pass any executor that implements QueryGraph.
type graphExec interface {
	QueryGraph(cypher string, params map[string]interface{}) ([]map[string]interface{}, error)
}

// FeedbackWeights reads session QA feedback from Neo4j and updates the
// feedback_weight property on relationships that were used in those turns.
//
// This stage is the first half of the feedback loop: user feedback
// (thumbs up/down on a turn) accumulates on the edges that contributed to
// the turn's answer, so future retrievers can boost or penalize those edges.
//
// Cypher summary:
//
//   MATCH (t:QATurn)
//   WHERE t.feedback_score IS NOT NULL
//   UNWIND t.used_graph_element_ids AS edgeRef
//   MATCH ()-[r {id: edgeRef}]-()
//   SET r.feedback_weight = COALESCE(r.feedback_weight, 0) + t.feedback_score
type FeedbackWeights struct {
	Graph graphExec
	// BatchSize caps the number of turns updated per query. Zero means
	// default (500).
	BatchSize int
}

// Name implements Stage.
func (s FeedbackWeights) Name() string { return "feedback_weights" }

// Run implements Stage.
func (s FeedbackWeights) Run(ctx context.Context, in Input) (StageResult, error) {
	start := time.Now().UTC()
	res := StageResult{Name: s.Name(), Started: start}

	if s.Graph == nil {
		res.Ended = time.Now().UTC()
		return res, fmt.Errorf("feedback_weights: graph executor is nil")
	}

	batchSize := s.BatchSize
	if batchSize <= 0 {
		batchSize = 500
	}

	cypher := `
MATCH (t:QATurn)
WHERE t.feedback_score IS NOT NULL
  AND (t.user_id = $user_id OR $user_id = "")
  AND size(t.used_graph_element_ids) > 0
WITH t LIMIT $limit
UNWIND t.used_graph_element_ids AS edgeRef
WITH t, edgeRef
WHERE edgeRef IS NOT NULL AND edgeRef <> ""
MATCH ()-[r]->()
WHERE r.id = edgeRef OR elementId(r) = edgeRef
SET r.feedback_weight = COALESCE(r.feedback_weight, 0) + t.feedback_score
RETURN count(DISTINCT r) AS updated_edges, count(DISTINCT t) AS processed_turns
`
	params := map[string]interface{}{
		"user_id": in.UserID,
		"limit":   int64(batchSize),
	}
	rows, err := s.Graph.QueryGraph(cypher, params)
	if err != nil {
		res.Ended = time.Now().UTC()
		return res, fmt.Errorf("feedback_weights: query: %w", err)
	}

	items := 0
	if len(rows) > 0 {
		if v, ok := rows[0]["updated_edges"].(int64); ok {
			items = int(v)
		} else if v, ok := rows[0]["updated_edges"].(int); ok {
			items = v
		}
	}
	res.Items = items
	res.Ended = time.Now().UTC()
	return res, nil
}
