package metrics

import (
	"sync"
	"testing"
)

func TestNewMetricsCollector(t *testing.T) {
	m := NewMetricsCollector()
	if m == nil {
		t.Fatal("NewMetricsCollector() returned nil")
	}
	if m.ExtractionsByProvider == nil {
		t.Error("ExtractionsByProvider map is nil")
	}
	if m.SpreadingActivationHops == nil {
		t.Error("SpreadingActivationHops map is nil")
	}
	if m.TierHits == nil {
		t.Error("TierHits map is nil")
	}
	if m.ExtractionsTotal != 0 {
		t.Errorf("expected ExtractionsTotal=0, got %d", m.ExtractionsTotal)
	}
}

func TestRecordExtraction(t *testing.T) {
	m := NewMetricsCollector()

	tests := []struct {
		name        string
		provider    string
		tokensSaved int64
		latencyMs   float64
	}{
		{"single extraction", "openai", 500, 120.5},
		{"another provider", "anthropic", 300, 95.0},
		{"zero tokens", "groq", 0, 10.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m.RecordExtraction(tt.provider, tt.tokensSaved, tt.latencyMs)
		})
	}

	if m.ExtractionsTotal != 3 {
		t.Errorf("expected ExtractionsTotal=3, got %d", m.ExtractionsTotal)
	}
	if m.ExtractionsByProvider["openai"] != 1 {
		t.Errorf("expected openai count=1, got %d", m.ExtractionsByProvider["openai"])
	}
	if m.ExtractionsByProvider["anthropic"] != 1 {
		t.Errorf("expected anthropic count=1, got %d", m.ExtractionsByProvider["anthropic"])
	}
	if m.TokensSavedTotal != 800 {
		t.Errorf("expected TokensSavedTotal=800, got %d", m.TokensSavedTotal)
	}
}

func TestRecordExtractionLatencyAverage(t *testing.T) {
	m := NewMetricsCollector()
	m.RecordExtraction("openai", 100, 60.0)
	m.RecordExtraction("openai", 200, 100.0)

	if m.CompressionLatencyMs != 80.0 {
		t.Errorf("expected average latency=80.0, got %f", m.CompressionLatencyMs)
	}
}

func TestRecordSpreadingActivation(t *testing.T) {
	m := NewMetricsCollector()

	m.RecordSpreadingActivation(2)
	m.RecordSpreadingActivation(3)
	m.RecordSpreadingActivation(2)

	if m.SpreadingActivationsTotal != 3 {
		t.Errorf("expected SpreadingActivationsTotal=3, got %d", m.SpreadingActivationsTotal)
	}
	if m.SpreadingActivationHops[2] != 2 {
		t.Errorf("expected hops[2]=2, got %d", m.SpreadingActivationHops[2])
	}
	if m.SpreadingActivationHops[3] != 1 {
		t.Errorf("expected hops[3]=1, got %d", m.SpreadingActivationHops[3])
	}
}

func TestRecordTierHit(t *testing.T) {
	m := NewMetricsCollector()

	tests := []struct {
		tier string
	}{
		{"working"},
		{"hot"},
		{"cold"},
		{"working"},
	}

	for _, tt := range tests {
		m.RecordTierHit(tt.tier)
	}

	if m.TierHits["working"] != 2 {
		t.Errorf("expected working=2, got %d", m.TierHits["working"])
	}
	if m.TierHits["hot"] != 1 {
		t.Errorf("expected hot=1, got %d", m.TierHits["hot"])
	}
	if m.TierHits["cold"] != 1 {
		t.Errorf("expected cold=1, got %d", m.TierHits["cold"])
	}
}

func TestRecordCacheHit(t *testing.T) {
	m := NewMetricsCollector()

	m.RecordCacheHit(true)
	m.RecordCacheHit(true)
	m.RecordCacheHit(false)

	if m.CacheHits != 2 {
		t.Errorf("expected CacheHits=2, got %d", m.CacheHits)
	}
	if m.CacheMisses != 1 {
		t.Errorf("expected CacheMisses=1, got %d", m.CacheMisses)
	}
}

func TestSetAccuracyRetention(t *testing.T) {
	m := NewMetricsCollector()

	tests := []struct {
		value float64
	}{
		{0.97},
		{0.85},
		{1.0},
		{0.0},
	}

	for _, tt := range tests {
		m.SetAccuracyRetention(tt.value)
		if m.AccuracyRetention != tt.value {
			t.Errorf("expected AccuracyRetention=%f, got %f", tt.value, m.AccuracyRetention)
		}
	}
}

func TestGetSnapshot(t *testing.T) {
	m := NewMetricsCollector()
	m.RecordExtraction("openai", 500, 100.0)
	m.RecordExtraction("anthropic", 300, 80.0)
	m.RecordSpreadingActivation(2)
	m.RecordTierHit("hot")
	m.RecordCacheHit(true)
	m.SetAccuracyRetention(0.97)

	snap := m.GetSnapshot()

	if snap.ExtractionsTotal != 2 {
		t.Errorf("expected ExtractionsTotal=2, got %d", snap.ExtractionsTotal)
	}
	if snap.ExtractionsByProvider["openai"] != 1 {
		t.Errorf("expected openai=1 in snapshot, got %d", snap.ExtractionsByProvider["openai"])
	}
	if snap.SpreadingActivationsTotal != 1 {
		t.Errorf("expected SpreadingActivationsTotal=1, got %d", snap.SpreadingActivationsTotal)
	}
	if snap.SpreadingActivationHops[2] != 1 {
		t.Errorf("expected hops[2]=1 in snapshot, got %d", snap.SpreadingActivationHops[2])
	}
	if snap.TierHits["hot"] != 1 {
		t.Errorf("expected TierHits[hot]=1 in snapshot, got %d", snap.TierHits["hot"])
	}
	if snap.CacheHits != 1 {
		t.Errorf("expected CacheHits=1, got %d", snap.CacheHits)
	}
	if snap.AccuracyRetention != 0.97 {
		t.Errorf("expected AccuracyRetention=0.97, got %f", snap.AccuracyRetention)
	}
	if snap.TokensSavedTotal != 800 {
		t.Errorf("expected TokensSavedTotal=800, got %d", snap.TokensSavedTotal)
	}
}

func TestGetSnapshotIsCopy(t *testing.T) {
	m := NewMetricsCollector()
	m.RecordExtraction("openai", 100, 50.0)

	snap := m.GetSnapshot()
	snap.ExtractionsByProvider["openai"] = 999

	if m.ExtractionsByProvider["openai"] == 999 {
		t.Error("snapshot should be a copy, modifying it should not affect the original")
	}
}

func TestReset(t *testing.T) {
	m := NewMetricsCollector()
	m.RecordExtraction("openai", 500, 100.0)
	m.RecordSpreadingActivation(3)
	m.RecordTierHit("cold")
	m.RecordCacheHit(false)
	m.SetAccuracyRetention(0.95)

	m.Reset()

	if m.ExtractionsTotal != 0 {
		t.Errorf("expected ExtractionsTotal=0 after reset, got %d", m.ExtractionsTotal)
	}
	if m.SpreadingActivationsTotal != 0 {
		t.Errorf("expected SpreadingActivationsTotal=0 after reset, got %d", m.SpreadingActivationsTotal)
	}
	if m.CompressionLatencyMs != 0 {
		t.Errorf("expected CompressionLatencyMs=0 after reset, got %f", m.CompressionLatencyMs)
	}
	if m.TokensSavedTotal != 0 {
		t.Errorf("expected TokensSavedTotal=0 after reset, got %d", m.TokensSavedTotal)
	}
	if m.AccuracyRetention != 0 {
		t.Errorf("expected AccuracyRetention=0 after reset, got %f", m.AccuracyRetention)
	}
	if m.CacheHits != 0 {
		t.Errorf("expected CacheHits=0 after reset, got %d", m.CacheHits)
	}
	if m.CacheMisses != 0 {
		t.Errorf("expected CacheMisses=0 after reset, got %d", m.CacheMisses)
	}
	if len(m.ExtractionsByProvider) != 0 {
		t.Errorf("expected empty ExtractionsByProvider after reset, got %v", m.ExtractionsByProvider)
	}
	if len(m.SpreadingActivationHops) != 0 {
		t.Errorf("expected empty SpreadingActivationHops after reset, got %v", m.SpreadingActivationHops)
	}
	if len(m.TierHits) != 0 {
		t.Errorf("expected empty TierHits after reset, got %v", m.TierHits)
	}
}

func TestConcurrentAccess(t *testing.T) {
	m := NewMetricsCollector()
	var wg sync.WaitGroup

	const goroutines = 100
	wg.Add(goroutines * 5)

	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			m.RecordExtraction("openai", 10, 5.0)
		}()
		go func() {
			defer wg.Done()
			m.RecordSpreadingActivation(2)
		}()
		go func() {
			defer wg.Done()
			m.RecordTierHit("hot")
		}()
		go func() {
			defer wg.Done()
			m.RecordCacheHit(i%2 == 0)
		}()
		go func() {
			defer wg.Done()
			m.SetAccuracyRetention(0.97)
		}()
	}

	wg.Wait()

	snap := m.GetSnapshot()

	if snap.ExtractionsTotal != goroutines {
		t.Errorf("expected ExtractionsTotal=%d, got %d", goroutines, snap.ExtractionsTotal)
	}
	if snap.SpreadingActivationsTotal != goroutines {
		t.Errorf("expected SpreadingActivationsTotal=%d, got %d", goroutines, snap.SpreadingActivationsTotal)
	}
	if snap.TierHits["hot"] != goroutines {
		t.Errorf("expected TierHits[hot]=%d, got %d", goroutines, snap.TierHits["hot"])
	}
	totalCacheOps := snap.CacheHits + snap.CacheMisses
	if totalCacheOps != goroutines {
		t.Errorf("expected total cache ops=%d, got %d", goroutines, totalCacheOps)
	}
}

func TestResetWithConcurrentAccess(t *testing.T) {
	m := NewMetricsCollector()
	var wg sync.WaitGroup

	wg.Add(2)
	go func() {
		defer wg.Done()
		for i := 0; i < 1000; i++ {
			m.RecordExtraction("openai", 10, 5.0)
		}
	}()
	go func() {
		defer wg.Done()
		m.Reset()
	}()

	wg.Wait()

	snap := m.GetSnapshot()
	if snap.ExtractionsTotal > 1000 {
		t.Errorf("unexpected ExtractionsTotal=%d after concurrent reset", snap.ExtractionsTotal)
	}
}
