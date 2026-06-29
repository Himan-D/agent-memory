package improve

import (
	"context"
	"fmt"
	"time"

	"agent-memory/internal/session"
)

// DistillSessions runs the real session.Distiller on each input session and
// returns the total accepted-lesson count. Lessons are persisted by the
// session.Manager's Store, so when Manager is wired with TieredStore +
// Neo4jStore, accepted lessons land in Neo4j as :DistilledLesson nodes
// automatically.
//
// This is stage 3 of the improvement pipeline: ephemeral session QA turns
// become durable, entity-anchored, novelty-checked lessons.
type DistillSessions struct {
	Manager  *session.Manager
	Distiller *session.Distiller

	// Concurrency limits for the distiller; zero uses package defaults.
	CuratorConcurrency int
	WriterConcurrency  int
}

// Name implements Stage.
func (s DistillSessions) Name() string { return "distill_sessions" }

// Run implements Stage.
func (s DistillSessions) Run(ctx context.Context, in Input) (StageResult, error) {
	start := time.Now().UTC()
	res := StageResult{Name: s.Name(), Started: start}

	if s.Manager == nil {
		res.Ended = time.Now().UTC()
		return res, fmt.Errorf("distill_sessions: session manager is nil")
	}
	if s.Distiller == nil {
		res.Ended = time.Now().UTC()
		return res, fmt.Errorf("distill_sessions: distiller is nil")
	}
	if in.UserID == "" {
		res.Ended = time.Now().UTC()
		return res, fmt.Errorf("distill_sessions: input user_id is required")
	}

	total := 0
	for _, sid := range in.SessionIDs {
		dr := s.Distiller.Distill(ctx, in.UserID, sid)
		total += dr.Accepted
		if len(dr.Errors) > 0 {
			// Surface the first error but continue with remaining sessions.
			res.Ended = time.Now().UTC()
			return res, fmt.Errorf("distill_sessions: session %s had %d errors", sid, len(dr.Errors))
		}
	}

	res.Items = total
	res.Ended = time.Now().UTC()
	return res, nil
}
