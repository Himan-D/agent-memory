package chunking

import (
	"strings"
	"testing"
)

// ─── RecursiveChunker ─────────────────────────────────────────────────────────

func TestRecursiveChunker_ShortText(t *testing.T) {
	c := NewRecursiveChunker(1000, 100, "")
	chunks := c.Chunk("Hello world")
	if len(chunks) != 1 {
		t.Errorf("expected 1 chunk for short text, got %d", len(chunks))
	}
	if chunks[0] != "Hello world" {
		t.Errorf("unexpected chunk content: %q", chunks[0])
	}
}

func TestRecursiveChunker_EmptyText(t *testing.T) {
	c := NewRecursiveChunker(1000, 100, "")
	chunks := c.Chunk("")
	if len(chunks) != 0 {
		t.Errorf("expected 0 chunks for empty text, got %d", len(chunks))
	}
}

func TestRecursiveChunker_LargeText(t *testing.T) {
	// Build text that is definitely larger than 200 bytes
	var sb strings.Builder
	for i := 0; i < 20; i++ {
		sb.WriteString("This is paragraph number one with some content. ")
		sb.WriteString("It has multiple sentences. ")
		sb.WriteString("\n\n")
	}
	text := sb.String()

	c := NewRecursiveChunker(200, 20, "")
	chunks := c.Chunk(text)

	if len(chunks) < 2 {
		t.Errorf("expected multiple chunks for text of length %d, got %d", len(text), len(chunks))
	}
	// Each chunk should be non-empty
	for i, chunk := range chunks {
		if strings.TrimSpace(chunk) == "" {
			t.Errorf("chunk %d is empty", i)
		}
	}
}

func TestRecursiveChunker_CustomSeparator(t *testing.T) {
	c := NewRecursiveChunker(100, 10, "|")
	text := "part one|part two|part three and more content here"
	chunks := c.Chunk(text)
	if len(chunks) == 0 {
		t.Error("expected at least one chunk")
	}
}

func TestRecursiveChunker_NoOverlap(t *testing.T) {
	c := NewRecursiveChunker(50, 0, "")
	text := strings.Repeat("ABCDE", 30) // 150 chars
	chunks := c.Chunk(text)
	if len(chunks) == 0 {
		t.Fatal("expected chunks")
	}
	// All chunks should be <= 50 chars
	for i, chunk := range chunks {
		if len(chunk) > 55 { // small tolerance for word boundary
			t.Errorf("chunk %d too long: %d chars", i, len(chunk))
		}
	}
}

// ─── FixedSizeChunker ────────────────────────────────────────────────────────

func TestFixedSizeChunker_Basic(t *testing.T) {
	c := NewFixedSizeChunker(10, 0)
	text := "ABCDEFGHIJKLMNOPQRSTUVWXYZ" // 26 chars
	chunks := c.Chunk(text)

	if len(chunks) == 0 {
		t.Fatal("expected at least one chunk")
	}
	// First chunk should be 10 chars
	if len(chunks[0]) > 10 {
		t.Errorf("expected first chunk <= 10 chars, got %d: %q", len(chunks[0]), chunks[0])
	}
}

func TestFixedSizeChunker_Empty(t *testing.T) {
	c := NewFixedSizeChunker(100, 10)
	if len(c.Chunk("")) != 0 {
		t.Error("expected 0 chunks for empty text")
	}
}

func TestFixedSizeChunker_WithOverlap(t *testing.T) {
	c := NewFixedSizeChunker(10, 3)
	text := "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	chunks := c.Chunk(text)
	// With overlap, we get more chunks than without
	c2 := NewFixedSizeChunker(10, 0)
	chunks2 := c2.Chunk(text)
	if len(chunks) < len(chunks2) {
		t.Errorf("expected overlap to produce more chunks: %d vs %d", len(chunks), len(chunks2))
	}
}

// ─── SemanticChunker ─────────────────────────────────────────────────────────

func TestSemanticChunker_SingleSentence(t *testing.T) {
	c := NewSemanticChunker(nil)
	text := "This is a single short sentence."
	chunks := c.Chunk(text)
	if len(chunks) == 0 {
		t.Error("expected at least one chunk")
	}
}

func TestSemanticChunker_MultipleSentences(t *testing.T) {
	c := NewSemanticChunker(nil)
	text := "The sky is blue. The ocean is deep. " +
		"Machine learning is fascinating. " +
		"Dogs are good pets. Cats are also great companions."
	chunks := c.Chunk(text)
	if len(chunks) == 0 {
		t.Error("expected at least one chunk")
	}
}

func TestSemanticChunker_EmptyText(t *testing.T) {
	c := NewSemanticChunker(nil)
	if len(c.Chunk("")) != 0 {
		t.Error("expected 0 chunks for empty text")
	}
}

// ─── jaccardSimilarity ───────────────────────────────────────────────────────

func TestJaccardSimilarity_Identical(t *testing.T) {
	sim := jaccardSimilarity("the quick brown fox", "the quick brown fox")
	if sim != 1.0 {
		t.Errorf("expected 1.0 for identical texts, got %v", sim)
	}
}

func TestJaccardSimilarity_Disjoint(t *testing.T) {
	sim := jaccardSimilarity("hello world", "foo bar baz")
	if sim != 0.0 {
		t.Errorf("expected 0.0 for disjoint texts, got %v", sim)
	}
}

func TestJaccardSimilarity_Partial(t *testing.T) {
	sim := jaccardSimilarity("cat dog fish", "cat bird snake")
	if sim <= 0.0 || sim >= 1.0 {
		t.Errorf("expected partial similarity between 0 and 1, got %v", sim)
	}
}

// ─── splitIntoSentences ───────────────────────────────────────────────────────

func TestSplitIntoSentences_Basic(t *testing.T) {
	sentences := splitIntoSentences("Hello world. How are you? Fine!")
	if len(sentences) < 2 {
		t.Errorf("expected at least 2 sentences, got %d: %v", len(sentences), sentences)
	}
}

func TestSplitIntoSentences_Empty(t *testing.T) {
	if len(splitIntoSentences("")) != 0 {
		t.Error("expected 0 sentences for empty string")
	}
}

// ─── Chunker interface ────────────────────────────────────────────────────────

func TestChunkerInterface(t *testing.T) {
	var c Chunker
	c = NewRecursiveChunker(100, 10, "")
	result := c.Chunk("test text")
	if len(result) == 0 {
		t.Error("expected at least one chunk from interface call")
	}

	c = NewFixedSizeChunker(100, 10)
	result = c.Chunk("test text")
	if len(result) == 0 {
		t.Error("expected at least one chunk from FixedSizeChunker via interface")
	}
}
