package pipeline

import (
	"context"
	"testing"
	"time"
)

func TestCompressionPipeline_New(t *testing.T) {
	pipeline := NewCompressionPipeline(2, nil)

	if pipeline == nil {
		t.Fatal("expected non-nil pipeline")
	}

	if pipeline.workers != 2 {
		t.Errorf("expected 2 workers, got %d", pipeline.workers)
	}

	pipeline.Stop()
}

func TestCompressionPipeline_StartStop(t *testing.T) {
	pipeline := NewCompressionPipeline(2, nil)

	pipeline.Start()

	time.Sleep(10 * time.Millisecond)

	if pipeline.jobQueue == nil {
		t.Error("expected job queue to be initialized")
	}

	pipeline.Stop()

	time.Sleep(10 * time.Millisecond)
}

func TestCompressionPipeline_CompressAsync(t *testing.T) {
	pipeline := NewCompressionPipeline(2, nil)
	pipeline.Start()

	done := make(chan Result, 1)
	job := CompressionJob{
		MemoryID: "test-1",
		Content: "This is a test memory about machine learning and AI",
		Done:    done,
	}

	pipeline.CompressAsync(job)

	select {
	case result := <-done:
		t.Logf("Compressed: %s (reduction: %.2f)", result.Compressed, result.TokenReduction)
	case <-time.After(2 * time.Second):
		t.Error("timeout waiting for compression")
	}

	pipeline.Stop()
}

func TestCompressionPipeline_GetStats(t *testing.T) {
	pipeline := NewCompressionPipeline(2, nil)
	pipeline.Start()

	pipeline.GetStats()

	pipeline.Stop()
}

func TestCompressionPipeline_GetQueueDepth(t *testing.T) {
	pipeline := NewCompressionPipeline(2, nil)

	depth := pipeline.GetQueueDepth()
	if depth != 0 {
		t.Errorf("expected empty queue, got %d", depth)
	}
}

func TestCompressionPipeline_MultipleJobs(t *testing.T) {
	pipeline := NewCompressionPipeline(4, nil)
	pipeline.Start()

	jobCount := 10
	results := make(chan Result, jobCount)

	for i := 0; i < jobCount; i++ {
		job := CompressionJob{
			MemoryID: "test-" + string(rune(i)),
			Content: "Test memory content " + string(rune(i)),
			Done:    results,
		}
		pipeline.CompressAsync(job)
	}

	for i := 0; i < jobCount; i++ {
		select {
		case <-results:
			t.Logf("Received result %d", i+1)
		case <-time.After(5 * time.Second):
			t.Errorf("timeout waiting for job %d", i+1)
			break
		}
	}

	pipeline.Stop()
}

func TestCompressionPipeline_FullQueue(t *testing.T) {
	pipeline := NewCompressionPipeline(1, nil)
	pipeline.Start()

	for i := 0; i < 1000; i++ {
		job := CompressionJob{
			Content: "test",
			Done:    make(chan Result, 1),
		}
		pipeline.CompressAsync(job)
	}

	pipeline.Stop()
}

func TestCompressionPipeline_LearnPatterns(t *testing.T) {
	pipeline := NewCompressionPipeline(1, nil)

	memories := []string{
		"machine learning is great",
		"machine learning models",
		"artificial intelligence",
	}

	pipeline.LearnPatterns(memories)

	pipeline.AddPattern("ml", "machine learning")

	pipeline.Stop()
}

func TestCompressionConfig(t *testing.T) {
	config := DefaultCompressionConfig()

	config.SetMode(CompressionModeBalanced)
	if config.GetMode() != CompressionModeBalanced {
		t.Error("failed to set compression mode")
	}

	config.SetTierPolicy(TierPolicyAggressive)
	if config.GetTierPolicy() != TierPolicyAggressive {
		t.Error("failed to set tier policy")
	}
}

func TestCompressionMode_Values(t *testing.T) {
	tests := []struct {
		mode CompressionMode
		val string
	}{
		{CompressionModeExtract, "extract"},
		{CompressionModeBalanced, "balanced"},
		{CompressionModeAggressive, "aggressive"},
	}

	for _, tt := range tests {
		if string(tt.mode) != tt.val {
			t.Errorf("expected %s, got %s", tt.val, tt.mode)
		}
	}
}

func TestTierPolicy_Values(t *testing.T) {
	tests := []struct {
		policy TierPolicy
		val    string
	}{
		{TierPolicyAggressive, "aggressive"},
		{TierPolicyBalanced, "balanced"},
		{TierPolicyConservative, "conservative"},
	}

	for _, tt := range tests {
		if string(tt.policy) != tt.val {
			t.Errorf("expected %s, got %s", tt.val, tt.policy)
		}
	}
}

func TestContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	pipeline := NewCompressionPipeline(2, nil)
	pipeline.ctx = ctx

	pipeline.Start()
	cancel()

	time.Sleep(10 * time.Millisecond)

	pipeline.Stop()
}

func BenchmarkCompressionPipeline_Throughput(b *testing.B) {
	pipeline := NewCompressionPipeline(4, nil)
	pipeline.Start()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		done := make(chan Result, 1)
		job := CompressionJob{
			Content: "Test memory content for benchmarking",
			Done:    done,
		}
		pipeline.CompressAsync(job)
		<-done
	}

	pipeline.Stop()
}