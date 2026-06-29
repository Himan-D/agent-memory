package session

import (
	"sort"
	"strings"
	"time"
)

// Batch is one chronological slice of QA turns + candidate entries, packed
// under BatchCharBudget. The curator LLM processes one Batch at a time.
type Batch struct {
	Index   int
	Text    string
	Members []string // entry IDs included in this batch
}

// Batcher packs QA turns and candidate context entries into size-safe
// chronological batches. Mirrors Cognee's build_curator_batches.
type Batcher struct {
	CharBudget        int
	BlocksPerBatch    int
	MaxQuestionChars  int
	MaxAnswerChars    int
	MaxCandidateChars int
}

// NewBatcher returns a Batcher with package-default constants.
func NewBatcher() *Batcher {
	return &Batcher{
		CharBudget:        BatchCharBudget,
		BlocksPerBatch:    CuratorBlocksPerBatch,
		MaxQuestionChars:  MaxQAQuestionChars,
		MaxAnswerChars:    MaxQAAnswerChars,
		MaxCandidateChars: MaxCandidateChars,
	}
}

// Build packs turns and context entries into batches. Each batch contains up
// to BlocksPerBatch blocks. Blocks are sorted by timestamp first so the
// curator sees the timeline in order.
func (b *Batcher) Build(turns []QATurn, context []ContextEntry) []Batch {
	timeline := make([]timelineItem, 0, len(turns)+len(context))
	for _, t := range turns {
		q := truncate(collapseWS(t.Question), b.maxQuestionChars())
		a := truncate(collapseWS(t.Answer), b.maxAnswerChars())
		text := "User: " + q + "\nAssistant: " + a
		timeline = append(timeline, timelineItem{
			ts:    t.CreatedAt,
			block: text,
			id:    t.ID,
		})
	}
	for _, e := range context {
		c := truncate(collapseWS(e.Content), b.maxCandidateChars())
		text := "Candidate " + e.ID + " (" + e.Section + "): " + c
		timeline = append(timeline, timelineItem{
			ts:    e.CreatedAt,
			block: text,
			id:    e.ID,
		})
	}
	sort.SliceStable(timeline, func(i, j int) bool {
		return timeline[i].ts.Before(timeline[j].ts)
	})

	blocksPer := b.blocksPerBatch()
	budget := b.charBudget()

	var (
		batches []Batch
		current []timelineItem
		currentSize int
		flush = func(idx int) {
			if len(current) == 0 {
				return
			}
			strs := make([]string, len(current))
			members := make([]string, len(current))
			for i, it := range current {
				strs[i] = it.block
				members[i] = it.id
			}
			batches = append(batches, Batch{
				Index:   idx,
				Text:    strings.Join(strs, "\n\n"),
				Members: members,
			})
			current = current[:0]
			currentSize = 0
		}
	)

	idx := 0
	for _, item := range timeline {
		// Stop accumulating when we hit the block cap.
		if len(current) >= blocksPer {
			flush(idx)
			idx++
		}
		// If a single block would overflow the budget, emit it alone.
		if currentSize > 0 && currentSize+len(item.block)+2 > budget {
			flush(idx)
			idx++
		}
		current = append(current, item)
		currentSize += len(item.block) + 2
	}
	flush(idx)
	return batches
}

func (b *Batcher) charBudget() int {
	if b.CharBudget <= 0 {
		return BatchCharBudget
	}
	return b.CharBudget
}

func (b *Batcher) blocksPerBatch() int {
	if b.BlocksPerBatch <= 0 {
		return CuratorBlocksPerBatch
	}
	return b.BlocksPerBatch
}

func (b *Batcher) maxQuestionChars() int {
	if b.MaxQuestionChars <= 0 {
		return MaxQAQuestionChars
	}
	return b.MaxQuestionChars
}

func (b *Batcher) maxAnswerChars() int {
	if b.MaxAnswerChars <= 0 {
		return MaxQAAnswerChars
	}
	return b.MaxAnswerChars
}

func (b *Batcher) maxCandidateChars() int {
	if b.MaxCandidateChars <= 0 {
		return MaxCandidateChars
	}
	return b.MaxCandidateChars
}

type timelineItem struct {
	ts    time.Time
	block string
	id    string
}

// collapseWS collapses runs of whitespace into single spaces. Mirrors
// " ".join(text.split()) in Python.
func collapseWS(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	inWS := false
	for _, r := range s {
		if r == ' ' || r == '\n' || r == '\t' || r == '\r' {
			if !inWS {
				b.WriteByte(' ')
				inWS = true
			}
			continue
		}
		b.WriteRune(r)
		inWS = false
	}
	return strings.TrimSpace(b.String())
}

// truncate returns s clipped to max runes (not bytes, for human-readable
// limits). If max <= 0 it returns s unchanged.
func truncate(s string, max int) string {
	if max <= 0 || len(s) <= max {
		return s
	}
	// Walk runes so we don't cut a multi-byte sequence.
	count := 0
	for i := range s {
		if count == max {
			return s[:i]
		}
		count++
	}
	return s
}
