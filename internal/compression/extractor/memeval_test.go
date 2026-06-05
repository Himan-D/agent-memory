package extractor

import (
	"context"
	"io/ioutil"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestMemEval runs the extractor over a dataset directory pointed to by MEMEVAL_DIR or MEM0_DIR env var.
// It is safe to run locally by setting MEMEVAL_DIR to a local copy of the MemEval / Mem0 dataset.
// The test is skipped when the env var is not provided.
func TestMemEval(t *testing.T) {
	// Environment variable checked: MEMEVAL_DIR then MEM0_DIR
	dir := getenv("MEMEVAL_DIR")
	if dir == "" {
		dir = getenv("MEM0_DIR")
	}
	if dir == "" {
		t.Skip("MEMEVAL_DIR or MEM0_DIR not set; skipping memeval integration test")
	}

	// Read all .txt files (or .md) under dir
	files, err := filepath.Glob(filepath.Join(dir, "**", "*.txt"))
	if err != nil || len(files) == 0 {
		// Try md files
		files, _ = filepath.Glob(filepath.Join(dir, "**", "*.md"))
	}
	if len(files) == 0 {
		t.Skipf("No text files found in dataset dir %s; skipping", dir)
	}

	provider := &mockProvider{}
	extr := NewMemoryExtractor(provider)

	var total float64
	var count int
	var totalReduction float64
	startAll := time.Now()

	for i, f := range files {
		// limit to reasonable number in CI unless MEMEVAL_RUN_ALL=1
		if i >= 500 && getenv("MEMEVAL_RUN_ALL") != "1" {
			break
		}
		b, err := ioutil.ReadFile(f)
		if err != nil {
			t.Logf("failed read %s: %v", f, err)
			continue
		}
		content := strings.TrimSpace(string(b))
		if len(content) == 0 {
			continue
		}
		t0 := time.Now()
		res, err := extr.Extract(context.Background(), content)
		lat := time.Since(t0).Milliseconds()
		if err != nil {
			t.Logf("extract error %s: %v", f, err)
			continue
		}
		total += float64(lat)
		count++
		totalReduction += res.TokenReduction
	}

	if count == 0 {
		t.Skip("no files processed")
	}

	avgLatency := total / float64(count)
	avgReduction := totalReduction / float64(count)
	t.Logf("Processed %d files from %s in %s. Avg latency=%.2fms, Avg token reduction=%.2f", count, dir, time.Since(startAll), avgLatency, avgReduction)
}

// getenv reads environment variable and returns empty string if not set
func getenv(k string) string {
	return strings.TrimSpace(strings.ReplaceAll(strings.TrimSpace(getenvRaw(k)), "\n", ""))
}

func getenvRaw(k string) string {
	return lookupEnv(k)
}
