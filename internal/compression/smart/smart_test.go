package smart

import (
	"testing"
)

func TestSmartCompressor_New(t *testing.T) {
	compressor := NewSmartCompressor(nil, 2)

	if compressor == nil {
		t.Fatal("expected non-nil compressor")
	}

	if compressor.radix == nil {
		t.Error("expected radix compressor to be initialized")
	}

	compressor.Stop()
}

func TestSmartCompressor_CompressEmpty(t *testing.T) {
	compressor := NewSmartCompressor(nil, 1)

	compressed, reduction, err := compressor.Compress(nil, "", ModeHybrid)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if compressed != "" {
		t.Errorf("expected empty result for empty input, got %s", compressed)
	}

	if reduction != 0.0 {
		t.Errorf("expected 0 reduction for empty input, got %f", reduction)
	}

	compressor.Stop()
}

func TestSmartCompressor_CompressRadix(t *testing.T) {
	compressor := NewSmartCompressor(nil, 1)

	compressor.radix.AddPattern("machine learning", "ML")
	compressor.radix.AddPattern("natural language processing", "NLP")

	content := "Machine learning and natural language processing are great technologies"
	compressed, reduction, err := compressor.Compress(nil, content, ModeRadix)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if compressed == "" {
		t.Error("expected compressed result")
	}

	t.Logf("Original: %s", content)
	t.Logf("Compressed: %s (reduction: %.2f%%)", compressed, reduction*100)

	compressor.Stop()
}

func TestSmartCompressor_StatisticsTracking(t *testing.T) {
	compressor := NewSmartCompressor(nil, 1)

	content := "Test content for compression"
	compressor.Compress(nil, content, ModeRadix)
	compressor.Compress(nil, content, ModeRadix)
	compressor.Compress(nil, content, ModeRadix)

	stats := compressor.GetStats()

	if stats.TotalCompressions != 3 {
		t.Errorf("expected 3 compressions, got %d", stats.TotalCompressions)
	}

	if stats.TotalTokensSaved == 0 {
		t.Error("expected some tokens saved")
	}

	t.Logf("Stats: %+v", stats)

	compressor.Stop()
}

func TestSmartCompressor_MultipleModes(t *testing.T) {
	compressor := NewSmartCompressor(nil, 1)

	content := "This is a test about machine learning and AI"

	modes := []Mode{ModeRadix, ModeExtraction, ModeRelational, ModeHybrid}

	for _, mode := range modes {
		compressed, reduction, err := compressor.Compress(nil, content, mode)

		if err != nil {
			t.Logf("Mode %s error: %v", mode, err)
		}

		t.Logf("Mode %s: compressed=%s (reduction=%.2f)", mode, compressed[:min(50, len(compressed))], reduction)
	}

	compressor.Stop()
}

func TestSmartCompressor_LearnPatterns(t *testing.T) {
	compressor := NewSmartCompressor(nil, 1)

	memories := []string{
		"Machine learning requires data",
		"Python is used for ML",
		"Neural networks are part of deep learning",
	}

	compressor.LearnPatterns(memories)

	stats := compressor.GetStats()
	if stats.TotalCompressions == 0 {
		t.Error("expected learn patterns to work")
	}

	t.Logf("Stats: %+v", stats)

	compressor.Stop()
}

func TestSmartCompressor_ExtractionFallback(t *testing.T) {
	compressor := NewSmartCompressor(nil, 1)

	content := "I love machine learning and natural language processing"

	compressed, reduction, _ := compressor.Compress(nil, content, ModeExtraction)

	if compressed == "" {
		t.Error("expected fallback compression to work")
	}

	t.Logf("Extraction mode result: %s (reduction: %.2f)", compressed, reduction*100)

	compressor.Stop()
}

func TestMode_Values(t *testing.T) {
	modes := []Mode{ModeExtraction, ModeRelational, ModeHybrid, ModeRadix}

	for _, mode := range modes {
		if string(mode) == "" {
			t.Error("expected non-empty mode string")
		}
	}
}

func BenchmarkSmartCompressor_Radix(b *testing.B) {
	compressor := NewSmartCompressor(nil, 4)
	compressor.radix.AddPattern("machine learning", "ML")
	compressor.radix.AddPattern("natural language processing", "NLP")

	content := "Machine learning and natural language processing are fundamental technologies"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		compressor.Compress(nil, content, ModeRadix)
	}

	compressor.Stop()
}

func BenchmarkSmartCompressor_Hybrid(b *testing.B) {
	compressor := NewSmartCompressor(nil, 4)

	content := "Machine learning represents a fundamental approach to artificial intelligence"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		compressor.Compress(nil, content, ModeHybrid)
	}

	compressor.Stop()
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}