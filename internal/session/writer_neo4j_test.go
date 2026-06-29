package session

import (
	"context"
	"testing"
)

func TestNeo4jNoveltyWriter_AcceptsWithEntity(t *testing.T) {
	w := NewNeo4jNoveltyWriter(DefaultNeo4jNoveltyWriterConfig())
	got, err := w.Evaluate(context.Background(),
		ProposedLesson{WorkingStatement: "Alice lives in Paris"},
		[]string{"m1"}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !got.Accept {
		t.Fatalf("expected accept, got reason=%s", got.Reason)
	}
	if len(got.Entities) < 2 {
		t.Fatalf("expected >= 2 entities (Alice, Paris), got %v", got.Entities)
	}
}

func TestNeo4jNoveltyWriter_RejectsEmptyStatement(t *testing.T) {
	w := NewNeo4jNoveltyWriter(DefaultNeo4jNoveltyWriterConfig())
	got, _ := w.Evaluate(context.Background(),
		ProposedLesson{WorkingStatement: "   "},
		nil, nil, nil)
	if got.Accept {
		t.Fatal("expected reject for empty statement")
	}
	if got.Reason != "not_durable" {
		t.Fatalf("expected reason=not_durable, got %s", got.Reason)
	}
}

func TestNeo4jNoveltyWriter_RejectsNoEntities(t *testing.T) {
	w := NewNeo4jNoveltyWriter(DefaultNeo4jNoveltyWriterConfig())
	got, _ := w.Evaluate(context.Background(),
		ProposedLesson{WorkingStatement: "no entities here at all"},
		nil, nil, nil)
	if got.Accept {
		t.Fatal("expected reject for no entities")
	}
	if got.Reason != "not_durable" {
		t.Fatalf("expected reason=not_durable, got %s", got.Reason)
	}
}

func TestNeo4jNoveltyWriter_RejectsDuplicate(t *testing.T) {
	w := NewNeo4jNoveltyWriter(DefaultNeo4jNoveltyWriterConfig())
	prior := []string{"Alice lives in Paris"}
	got, _ := w.Evaluate(context.Background(),
		ProposedLesson{WorkingStatement: "Alice lives in Paris"},
		nil, prior, nil)
	if got.Accept {
		t.Fatal("expected reject for duplicate")
	}
	if got.Reason != "already_known" {
		t.Fatalf("expected reason=already_known, got %s", got.Reason)
	}
}

func TestNeo4jNoveltyWriter_RejectsSubstringDuplicate(t *testing.T) {
	w := NewNeo4jNoveltyWriter(DefaultNeo4jNoveltyWriterConfig())
	prior := []string{"Alice lives in Paris"}
	got, _ := w.Evaluate(context.Background(),
		ProposedLesson{WorkingStatement: "Alice lives in Paris and works there"},
		nil, prior, nil)
	if got.Accept {
		t.Fatal("expected reject for substring duplicate")
	}
}

func TestNeo4jNoveltyWriter_DisabledThresholdAlwaysPasses(t *testing.T) {
	cfg := DefaultNeo4jNoveltyWriterConfig()
	cfg.SimilarityThreshold = 0
	w := NewNeo4jNoveltyWriter(cfg)
	got, _ := w.Evaluate(context.Background(),
		ProposedLesson{WorkingStatement: "Alice lives in Paris"},
		nil, []string{"Alice lives in Paris"}, nil)
	if !got.Accept {
		t.Fatalf("threshold=0 should disable duplicate check, got reason=%s", got.Reason)
	}
}

func TestNeo4jNoveltyWriter_TruncatesLongStatements(t *testing.T) {
	cfg := DefaultNeo4jNoveltyWriterConfig()
	cfg.MaxStatementChars = 30
	w := NewNeo4jNoveltyWriter(cfg)
	long := "Alice lives in Paris and has many friends who also live in Paris"
	got, _ := w.Evaluate(context.Background(),
		ProposedLesson{WorkingStatement: long},
		nil, nil, nil)
	if !got.Accept {
		t.Fatal("expected accept after truncation")
	}
	if len(got.Statement) > 30+3 { // 30 chars + "..."
		t.Fatalf("statement not truncated: %q (len=%d)", got.Statement, len(got.Statement))
	}
}

func TestNeo4jNoveltyWriter_ExtractsMultiWordEntities(t *testing.T) {
	w := NewNeo4jNoveltyWriter(DefaultNeo4jNoveltyWriterConfig())
	got, _ := w.Evaluate(context.Background(),
		ProposedLesson{WorkingStatement: "Alice met Bob Smith in New York"},
		nil, nil, nil)
	if !got.Accept {
		t.Fatalf("expected accept, got reason=%s", got.Reason)
	}
	// Expect at least: Alice, Bob Smith, New York
	if len(got.Entities) < 3 {
		t.Fatalf("expected >= 3 entities, got %v", got.Entities)
	}
}

func TestNeo4jNoveltyWriter_FiltersStopWords(t *testing.T) {
	w := NewNeo4jNoveltyWriter(DefaultNeo4jNoveltyWriterConfig())
	got, _ := w.Evaluate(context.Background(),
		ProposedLesson{WorkingStatement: "The man walked"},
		nil, nil, nil)
	if got.Accept {
		t.Fatalf("expected reject (no entities after stop-word filter), got %+v", got)
	}
}

func TestNeo4jNoveltyWriter_ExtractEntitiesDedup(t *testing.T) {
	w := NewNeo4jNoveltyWriter(DefaultNeo4jNoveltyWriterConfig())
	got, _ := w.Evaluate(context.Background(),
		ProposedLesson{WorkingStatement: "Alice met Alice today"},
		nil, nil, nil)
	count := 0
	for _, e := range got.Entities {
		if e == "Alice" {
			count++
		}
	}
	if count > 1 {
		t.Fatalf("expected dedup of Alice, got %v", got.Entities)
	}
}

func TestNeo4jNoveltyWriter_ContextCanceled(t *testing.T) {
	w := NewNeo4jNoveltyWriter(DefaultNeo4jNoveltyWriterConfig())
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := w.Evaluate(ctx, ProposedLesson{WorkingStatement: "Alice"}, nil, nil, nil)
	if err == nil {
		t.Fatal("expected context.Canceled error")
	}
}

func TestNeo4jNoveltyWriter_EvaluateAsFunc(t *testing.T) {
	w := NewNeo4jNoveltyWriter(DefaultNeo4jNoveltyWriterConfig())
	fn := w.EvaluateAsFunc()
	got, err := fn(context.Background(),
		ProposedLesson{WorkingStatement: "Alice lives in Paris"},
		nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !got.Accept {
		t.Fatal("expected accept via WriterFunc adapter")
	}
}

func TestNormalizeForCompare(t *testing.T) {
	cases := map[string]string{
		"Hello, World!":   "hello world",
		"  spaces  ":      "spaces",
		"ALICE-LIVES-IN":  "alicelivesin",
		"":                "",
	}
	for in, want := range cases {
		if got := normalizeForCompare(in); got != want {
			t.Fatalf("normalizeForCompare(%q) = %q, want %q", in, got, want)
		}
	}
}
