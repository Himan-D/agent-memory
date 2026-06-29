package session

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"agent-memory/internal/resilience"
)

// CuratorFunc produces a list of ProposedLesson for a single batch. It is
// pluggable so the distiller can run with any LLM backend (or a fake in
// tests).
type CuratorFunc func(ctx context.Context, batchText string) ([]ProposedLesson, error)

// WriterFunc decides whether a single ProposedLesson becomes a durable
// WrittenLesson. It receives the proposed lesson, the source member IDs, and
// any prior lessons / entity glossary hits (used for novelty).
type WriterFunc func(ctx context.Context, proposed ProposedLesson, members []string, priorLessons []string, glossary []string) (WrittenLesson, error)

// NoveltySearchFunc returns prior lesson statements and entity glossary
// entries that are semantically near the candidate. It is invoked once per
// proposed lesson by the writer.
type NoveltySearchFunc func(ctx context.Context, statement string) (priorLessons []string, glossary []string, err error)

// DistillOptions configures the Distiller.
type DistillOptions struct {
	Curator             CuratorFunc
	Writer              WriterFunc
	Novelty             NoveltySearchFunc
	CuratorConcurrency  int
	WriterConcurrency   int
	Logger              *log.Logger
	// Batcher is the timeline batcher. When nil, NewBatcher() is used.
	Batcher *Batcher
}

// Distiller turns ephemeral session turns + candidate context into durable
// DistilledLessons. It mirrors Cognee's session distillation pipeline:
// gate → batch → curator (parallel) → writer (parallel) → render.
//
// Fail-open per unit: a single batch or lesson failure is logged and
// skipped; it does not abort the whole job.
type Distiller struct {
	mgr     *Manager
	batcher *Batcher
	opts    DistillOptions
}

// NewDistiller returns a Distiller backed by the given Manager. The options
// must include at least a Writer; the Curator is required to produce
// proposals. Novelty may be nil (no novelty check).
func NewDistiller(mgr *Manager, opts DistillOptions) *Distiller {
	if opts.Curator == nil {
		opts.Curator = noopCurator
	}
	if opts.Writer == nil {
		opts.Writer = acceptAllWriter
	}
	if opts.CuratorConcurrency <= 0 {
		opts.CuratorConcurrency = CuratorConcurrency
	}
	if opts.WriterConcurrency <= 0 {
		opts.WriterConcurrency = WriterConcurrency
	}
	if opts.Logger == nil {
		opts.Logger = log.Default()
	}
	if opts.Batcher == nil {
		opts.Batcher = NewBatcher()
	}
	return &Distiller{mgr: mgr, batcher: opts.Batcher, opts: opts}
}

// Result summarizes one Distill call.
type Result struct {
	Proposed   int
	Accepted   int
	Rejected   int
	Lessons    []DistilledLesson
	Errors     []error
	DurationMs int64
}

// Distill runs the full distillation pipeline for a user's session. It
// returns a Result with counts and any per-lesson errors. A non-nil
// Result.Error is never returned; the pipeline is fail-open by design.
func (d *Distiller) Distill(ctx context.Context, userID, sessionID string) Result {
	start := time.Now()
	res := Result{}

	turns, err := d.mgr.GetSession(ctx, userID, sessionID, 0)
	if err != nil {
		res.Errors = append(res.Errors, fmt.Errorf("distill: list turns: %w", err))
		res.DurationMs = time.Since(start).Milliseconds()
		return res
	}
	if len(turns) == 0 {
		res.DurationMs = time.Since(start).Milliseconds()
		return res
	}

	ctxEntries, err := d.mgr.FilteredContext(userID)
	if err != nil {
		res.Errors = append(res.Errors, fmt.Errorf("distill: filter context: %w", err))
		// Continue with empty context - the curator can still see the QA timeline.
		ctxEntries = nil
	}

	batches := d.batcher.Build(turns, ctxEntries)
	if len(batches) == 0 {
		res.DurationMs = time.Since(start).Milliseconds()
		return res
	}

	// Stage 1: parallel curator calls.
	curatorTasks := make([]func(ctx context.Context) ([]ProposedLesson, error), len(batches))
	for i, b := range batches {
		batch := b
		curatorTasks[i] = func(ctx context.Context) ([]ProposedLesson, error) {
			lessons, err := d.opts.Curator(ctx, batch.Text)
			if err != nil {
				d.opts.Logger.Printf("distill: curator batch %d failed: %v", batch.Index, err)
				return nil, nil
			}
			return lessons, nil
		}
	}

	perBatch := resilience.GatherTyped(ctx, resilience.GatherOptions{Limit: d.opts.CuratorConcurrency}, curatorTasks)

	var proposals []proposedWithMembers
	for i, r := range perBatch {
		if r.Err != nil {
			res.Errors = append(res.Errors, fmt.Errorf("distill: curator batch %d: %w", i, r.Err))
			continue
		}
		for _, lesson := range r.Value {
			proposals = append(proposals, proposedWithMembers{
				proposed: lesson,
				members:  batches[i].Members,
			})
		}
	}
	res.Proposed = len(proposals)

	// Stage 2: parallel writer calls with novelty check.
	writerTasks := make([]func(ctx context.Context) (WrittenLesson, error), len(proposals))
	for i := range proposals {
		p := proposals[i]
		writerTasks[i] = func(ctx context.Context) (WrittenLesson, error) {
			var prior []string
			var gloss []string
			if d.opts.Novelty != nil {
				pl, gl, err := d.opts.Novelty(ctx, p.proposed.WorkingStatement)
				if err != nil {
					d.opts.Logger.Printf("distill: novelty check failed: %v", err)
					// Continue without novelty info rather than failing the lesson.
				}
				prior = pl
				gloss = gl
			}
			out, err := d.opts.Writer(ctx, p.proposed, p.members, prior, gloss)
			if err != nil {
				d.opts.Logger.Printf("distill: writer failed: %v", err)
				return WrittenLesson{}, err
			}
			return out, nil
		}
	}

	decisions := resilience.GatherTyped(ctx, resilience.GatherOptions{Limit: d.opts.WriterConcurrency}, writerTasks)

	for i, dec := range decisions {
		if dec.Err != nil {
			res.Errors = append(res.Errors, fmt.Errorf("distill: writer proposal %d: %w", i, dec.Err))
			continue
		}
		w := dec.Value
		if !w.Accept {
			res.Rejected++
			continue
		}
		lesson := DistilledLesson{
			UserID:         userID,
			SessionID:      sessionID,
			Statement:      w.Statement,
			Entities:       w.Entities,
			WhyLearned:     w.WhyLearned,
			MemberEntryIDs: proposals[i].members,
			DistilledOn:    time.Now().UTC(),
		}
		if err := d.mgr.SaveLesson(ctx, userID, lesson); err != nil {
			res.Errors = append(res.Errors, fmt.Errorf("distill: save lesson: %w", err))
			continue
		}
		res.Accepted++
		res.Lessons = append(res.Lessons, lesson)
	}

	res.DurationMs = time.Since(start).Milliseconds()
	return res
}

// RenderLesson returns the markdown document body for a distilled lesson.
// It is exported so callers can re-ingest the lesson into the existing
// ProMem extractor.
func RenderLesson(l DistilledLesson) string {
	statement := strings.TrimSpace(l.Statement)
	why := strings.TrimSpace(l.WhyLearned)
	why = strings.TrimRight(why, ".")
	var body string
	if why != "" {
		body = fmt.Sprintf("%s (%s.)", statement, why)
	} else {
		body = statement
	}
	title := l.DistilledOn.Format("2006-01-02")
	return fmt.Sprintf("# Session learning — %s (session %s)\n\n%s\n", title, l.SessionID, body)
}

type proposedWithMembers struct {
	proposed ProposedLesson
	members  []string
}

// noopCurator returns no proposals. It is the default when the caller does
// not configure an LLM curator.
func noopCurator(_ context.Context, _ string) ([]ProposedLesson, error) {
	return nil, nil
}

// acceptAllWriter accepts every proposed lesson verbatim.
func acceptAllWriter(_ context.Context, p ProposedLesson, _ []string, _ []string, _ []string) (WrittenLesson, error) {
	return WrittenLesson{
		Accept:    true,
		Statement: p.WorkingStatement,
	}, nil
}

// AcceptAllWriter is the exported alias of acceptAllWriter. It is convenient
// for tests and callers that want to accept every proposal without writing a
// custom WriterFunc.
func AcceptAllWriter() WriterFunc {
	return acceptAllWriter
}

// ErrDistillCanceled indicates the caller canceled the distillation.
var ErrDistillCanceled = errors.New("distill canceled")
