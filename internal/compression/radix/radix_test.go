package radix

import (
	"testing"
)

func TestRadixTree_InsertAndSearch(t *testing.T) {
	tree := New()

	tree.Insert("hello", "world")
	tree.Insert("help", "me")
	tree.Insert("hero", "zero")

	tests := []struct {
		key      string
		wantVal  string
		wantOk   bool
	}{
		{"hello", "world", true},
		{"help", "me", true},
		{"hero", "zero", true},
		{"hell", "", false},
		{"xyz", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			val, ok := tree.Search(tt.key)
			if ok != tt.wantOk {
				t.Errorf("Search(%q) ok = %v, want %v", tt.key, ok, tt.wantOk)
			}
			if val != tt.wantVal {
				t.Errorf("Search(%q) = %v, want %v", tt.key, val, tt.wantVal)
			}
		})
	}
}

func TestRadixTree_Delete(t *testing.T) {
	tree := New()

	tree.Insert("hello", "world")
	tree.Insert("help", "me")

	if _, ok := tree.Search("hello"); !ok {
		t.Fatal("expected hello to exist before delete")
	}

	tree.Delete("hello")

	if _, ok := tree.Search("hello"); ok {
		t.Error("expected hello to be deleted")
	}

	if _, ok := tree.Search("help"); !ok {
		t.Error("expected help to still exist")
	}
}

func TestRadixTree_FindByPrefix(t *testing.T) {
	tree := New()

	tree.Insert("hello", "world")
	tree.Insert("help", "me")
	tree.Insert("hero", "zero")
	tree.Insert("hell", "fire")
	tree.Insert("test", "case")

	tests := []struct {
		prefix  string
		wantLen int
	}{
		{"hel", 3},
		{"her", 1},
		{"test", 1},
		{"xyz", 0},
		{"", 5},
	}

	for _, tt := range tests {
		t.Run(tt.prefix, func(t *testing.T) {
			results := tree.FindByPrefix(tt.prefix)
			if len(results) != tt.wantLen {
				t.Errorf("FindByPrefix(%q) = %d results, want %d", tt.prefix, len(results), tt.wantLen)
			}
		})
	}
}

func TestMemoryCompressor_Basic(t *testing.T) {
	compressor := NewMemoryCompressor()

	compressor.AddPattern("machine learning", "ML")
	compressor.AddPattern("natural language processing", "NLP")

	compressed := compressor.Compress("I work with machine learning and natural language processing")
	
	if compressed == "" {
		t.Error("expected compressed string, got empty")
	}

	t.Logf("Compressed: %s", compressed)
}

func TestMemoryCompressor_Deduplication(t *testing.T) {
	compressor := NewMemoryCompressor()

	text := "first line\nsecond line\nfirst line\nthird line\nsecond line"
	compressed := compressor.Compress(text)

	lines := 0
	for _, c := range compressed {
		if c == '\n' {
			lines++
		}
	}
	lines++

	if lines > 3 {
		t.Errorf("expected deduplicated lines, got %d lines from: %s", lines, compressed)
	}
}

func TestMemoryCompressor_LearnFromMemories(t *testing.T) {
	compressor := NewMemoryCompressor()

	memories := []string{
		"machine learning is great",
		"machine learning requires data",
		"machine learning models",
		"random text here",
		"another random text",
	}

	compressor.LearnFromMemories(memories)

	if len(compressor.patterns) == 0 {
		t.Error("expected patterns to be learned")
	}

	t.Logf("Learned patterns: %v", compressor.patterns)
}

func TestMemoryCompressor_CompressDecompress(t *testing.T) {
	compressor := NewMemoryCompressor()

	compressor.AddPattern("artificial intelligence", "AI")
	compressor.AddPattern("machine learning", "ML")

	original := "AI and ML are related fields"
	compressed := compressor.Compress(original)

	if compressed == original {
		t.Logf("Compression may not have replaced anything: %s", compressed)
	}

	decompressed := compressor.Decompress(compressed)

	if decompressed == "" {
		t.Error("expected decompressed string, got empty")
	}

	t.Logf("Original: %s", original)
	t.Logf("Compressed: %s", compressed)
	t.Logf("Decompressed: %s", decompressed)
}

func TestMemoryCompressor_GetStats(t *testing.T) {
	compressor := NewMemoryCompressor()

	compressor.AddPattern("test", "t")
	compressor.AddPattern("example", "ex")

	text := "test example test data"
	stats := compressor.GetStats(text)

	if stats.OriginalSize == 0 {
		t.Error("expected non-zero original size")
	}

	if stats.PatternsUsed != 2 {
		t.Errorf("expected 2 patterns used, got %d", stats.PatternsUsed)
	}

	if stats.Reduction < 0 || stats.Reduction > 1 {
		t.Errorf("reduction should be between 0 and 1, got %f", stats.Reduction)
	}

	t.Logf("Stats: Original=%d, Compressed=%d, Reduction=%.2f%%", 
		stats.OriginalSize, stats.CompressedSize, stats.Reduction*100)
}

func TestGenerateAbbreviation(t *testing.T) {
	tests := []struct {
		word string
		want string
	}{
		{"hello", "helo"},
		{"world", "wrld"},
		{"test", "test"},
		{"ai", "ai"},
		{"machine", "mche"},
	}

	for _, tt := range tests {
		t.Run(tt.word, func(t *testing.T) {
			abbrev := generateAbbreviation(tt.word)
			if len(abbrev) > len(tt.word) {
				t.Errorf("abbreviation %s is longer than original %s", abbrev, tt.word)
			}
		})
	}
}

func TestRadixTree_Empty(t *testing.T) {
	tree := New()

	if _, ok := tree.Search("anything"); ok {
		t.Error("expected empty tree to return false")
	}

	results := tree.FindByPrefix("any")
	if len(results) != 0 {
		t.Errorf("expected no results from empty tree, got %d", len(results))
	}
}

func TestRadixTree_CaseInsensitive(t *testing.T) {
	tree := New()

	tree.Insert("Hello", "World")
	tree.Insert("HELLO", "WORLD")

	val, ok := tree.Search("hello")
	if !ok {
		t.Error("expected to find hello (case insensitive)")
	}
	if val != "World" && val != "WORLD" {
		t.Errorf("unexpected value: %s", val)
	}
}

func BenchmarkRadixTree_Insert(b *testing.B) {
	tree := New()
	words := []string{"hello", "world", "test", "example", "data", "compression", "memory", "learning"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, w := range words {
			tree.Insert(w, w+"_value")
		}
	}
}

func BenchmarkRadixTree_Search(b *testing.B) {
	tree := New()
	for i := 0; i < 1000; i++ {
		tree.Insert("word"+string(rune(i)), "value")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tree.Search("word500")
	}
}

func BenchmarkMemoryCompressor_Compress(b *testing.B) {
	compressor := NewMemoryCompressor()
	memories := make([]string, 100)
	for i := 0; i < 100; i++ {
		memories[i] = "machine learning and artificial intelligence are great"
	}
	compressor.LearnFromMemories(memories)

	text := "machine learning is great artificial intelligence compression test"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		compressor.Compress(text)
	}
}
