package memory

import (
	"strings"
	"testing"
)

func TestParagraphChunker_EmptyText(t *testing.T) {
	c := NewParagraphChunker(100)
	chunks := c.Chunk("doc-1", "")
	if len(chunks) != 1 {
		t.Fatalf("expected 1 chunk for empty text, got %d", len(chunks))
	}
	if chunks[0].DocumentID != "doc-1" || chunks[0].Index != 0 {
		t.Fatalf("unexpected chunk: %+v", chunks[0])
	}
}

func TestParagraphChunker_SingleParagraph(t *testing.T) {
	c := NewParagraphChunker(100)
	chunks := c.Chunk("doc-1", "hello world")
	if len(chunks) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(chunks))
	}
	if chunks[0].Text != "hello world" {
		t.Fatalf("unexpected text: %q", chunks[0].Text)
	}
}

func TestParagraphChunker_MultipleParagraphs(t *testing.T) {
	c := NewParagraphChunker(50)
	text := "First paragraph here.\n\nSecond paragraph here.\n\nThird paragraph here."
	chunks := c.Chunk("doc-2", text)
	if len(chunks) < 2 {
		t.Fatalf("expected multiple chunks, got %d", len(chunks))
	}
	for i, ch := range chunks {
		if ch.Index != i {
			t.Fatalf("chunk[%d].Index = %d", i, ch.Index)
		}
		if ch.DocumentID != "doc-2" {
			t.Fatalf("chunk[%d].DocumentID = %q", i, ch.DocumentID)
		}
	}
}

func TestParagraphChunker_DeterministicIDs(t *testing.T) {
	c := NewParagraphChunker(50)
	text := "A.\n\nB.\n\nC."
	a := c.Chunk("doc-3", text)
	b := c.Chunk("doc-3", text)
	if len(a) != len(b) {
		t.Fatalf("chunk counts differ: %d vs %d", len(a), len(b))
	}
	for i := range a {
		if a[i].ID != b[i].ID {
			t.Fatalf("chunk[%d] ID mismatch: %s vs %s", i, a[i].ID, b[i].ID)
		}
	}
}

func TestParagraphChunker_OversizedParagraphStaysIntact(t *testing.T) {
	c := NewParagraphChunker(20)
	big := strings.Repeat("x", 100)
	chunks := c.Chunk("doc-4", big)
	if len(chunks) != 1 {
		t.Fatalf("expected 1 chunk for oversized paragraph, got %d", len(chunks))
	}
	if chunks[0].Text != big {
		t.Fatalf("oversized paragraph was modified")
	}
}

func TestParagraphChunker_NeverSplitsParagraphs(t *testing.T) {
	c := NewParagraphChunker(40)
	text := "AAA BBB CCC.\n\nDDD EEE FFF.\n\nGGG HHH III."
	chunks := c.Chunk("doc-5", text)
	for i, ch := range chunks {
		if strings.Contains(ch.Text, "\n\n") {
			t.Fatalf("chunk[%d] contains a paragraph break: %q", i, ch.Text)
		}
	}
}
