package improve

import (
	"context"
	"fmt"
	"time"

	"agent-memory/internal/session"
)

// PersistSessions reads session QA turns from the hot store and writes them
// to Neo4j as Session + QATurn nodes with PART_OF relationships. This is
// stage 2 of the improvement pipeline: ephemeral hot-tier data becomes
// durable cold-tier data the graph can query.
//
// Implementation notes:
//   - Uses session.Store for the read (typically RedisStore or TieredStore).
//   - Uses graphExec for the write (typically *neo4j.Client).
//   - One Cypher MERGE per turn so partial failures don't lose earlier
//     turns in the same session.
type PersistSessions struct {
	Store session.Store
	Graph graphExec
	// MaxTurnsPerSession caps turns persisted per session. Zero means no cap.
	MaxTurnsPerSession int
}

// Name implements Stage.
func (s PersistSessions) Name() string { return "persist_sessions" }

// Run implements Stage.
func (s PersistSessions) Run(ctx context.Context, in Input) (StageResult, error) {
	start := time.Now().UTC()
	res := StageResult{Name: s.Name(), Started: start}

	if s.Store == nil {
		res.Ended = time.Now().UTC()
		return res, fmt.Errorf("persist_sessions: session store is nil")
	}
	if s.Graph == nil {
		res.Ended = time.Now().UTC()
		return res, fmt.Errorf("persist_sessions: graph executor is nil")
	}

	// Determine which sessions to persist. The Input may list specific
	// session IDs; otherwise we persist one session per SessionID-less
	// turn-batch under the user.
	persisted := 0
	for _, sid := range in.SessionIDs {
		turns, err := s.Store.ListTurns(in.UserID, sid, s.MaxTurnsPerSession)
		if err != nil {
			res.Ended = time.Now().UTC()
			return res, fmt.Errorf("persist_sessions: list turns for %s: %w", sid, err)
		}
		for _, turn := range turns {
			if err := s.persistTurn(ctx, in.UserID, sid, turn); err != nil {
				res.Ended = time.Now().UTC()
				return res, fmt.Errorf("persist_sessions: persist turn %s: %w", turn.ID, err)
			}
			persisted++
		}
	}

	res.Items = persisted
	res.Ended = time.Now().UTC()
	return res, nil
}

func (s PersistSessions) persistTurn(_ context.Context, userID, sessionID string, turn session.QATurn) error {
	cypher := `
MERGE (sess:Session {id: $session_id})
  ON CREATE SET sess.user_id = $user_id,
                sess.created_at = datetime($created_at),
                sess.updated_at = datetime($created_at)
  ON MATCH  SET sess.updated_at = datetime($created_at)
MERGE (t:QATurn {id: $turn_id})
  ON CREATE SET t.user_id = $user_id,
                t.session_id = $session_id,
                t.question = $question,
                t.answer = $answer,
                t.context = $context,
                t.created_at = datetime($created_at),
                t.confidence = $confidence,
                t.harmful_count = $harmful_count,
                t.feedback_score = $feedback_score,
                t.used_graph_element_ids = $used_graph_element_ids
MERGE (t)-[:PART_OF]->(sess)
`
	params := map[string]interface{}{
		"session_id":              sessionID,
		"user_id":                 userID,
		"turn_id":                 turn.ID,
		"question":                turn.Question,
		"answer":                  turn.Answer,
		"context":                 turn.Context,
		"created_at":              turn.CreatedAt.Format(time.RFC3339Nano),
		"confidence":              turn.Confidence,
		"harmful_count":           turn.HarmfulCount,
		"feedback_score":          feedbackOrNil(turn.FeedbackScore),
		"used_graph_element_ids":  turn.UsedGraphElementIDs,
	}
	_, err := s.Graph.QueryGraph(cypher, params)
	return err
}

// feedbackOrNil converts *float64 to interface{} so Cypher stores NULL
// instead of 0 when no feedback was recorded.
func feedbackOrNil(f *float64) interface{} {
	if f == nil {
		return nil
	}
	return *f
}
