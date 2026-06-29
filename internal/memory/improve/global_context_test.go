package improve

import (
	"context"
	"strings"
	"testing"
	"time"
)

type fakeCache struct {
	get     map[string]string
	setErr  error
	getErr  error
	keys    map[string]string
	ttls    map[string]time.Duration
}

func (f *fakeCache) Get(ctx context.Context, key string) (string, error) {
	if f.getErr != nil {
		return "", f.getErr
	}
	if f.get == nil {
		return "", nil
	}
	return f.get[key], nil
}

func (f *fakeCache) Set(ctx context.Context, key, value string, ttl time.Duration) error {
	if f.setErr != nil {
		return f.setErr
	}
	if f.keys == nil {
		f.keys = map[string]string{}
		f.ttls = map[string]time.Duration{}
	}
	f.keys[key] = value
	f.ttls[key] = ttl
	return nil
}

func TestGlobalContextIndex_NilGraphErrors(t *testing.T) {
	s := GlobalContextIndex{TenantID: "t1"}
	if _, err := s.Run(context.Background(), Input{UserID: "u1"}); err == nil {
		t.Fatal("expected error for nil graph")
	}
}

func TestGlobalContextIndex_MissingTenantErrors(t *testing.T) {
	s := GlobalContextIndex{Graph: &fakeGraph{}}
	if _, err := s.Run(context.Background(), Input{}); err == nil {
		t.Fatal("expected error for missing tenant_id")
	}
}

func TestGlobalContextIndex_BuildsSummaryAndWrites(t *testing.T) {
	fg := &fakeGraph{
		rows: []map[string]interface{}{
			{"id": "l1", "statement": "Alice lives in Paris", "entities": []interface{}{"Alice", "Paris"}, "distilled_on": time.Now()},
			{"id": "l2", "statement": "Bob works at Acme", "entities": []interface{}{"Bob", "Acme"}, "distilled_on": time.Now()},
		},
	}
	s := GlobalContextIndex{Graph: fg, TenantID: "t1", TopK: 10, MaxSummaryChars: 500}
	res, err := s.Run(context.Background(), Input{UserID: "u1"})
	if err != nil {
		t.Fatal(err)
	}
	if res.Items != 2 {
		t.Fatalf("expected Items=2, got %d", res.Items)
	}
	if len(fg.cyphers) < 2 {
		t.Fatalf("expected at least 2 cypher calls (read + write), got %d", len(fg.cyphers))
	}
	writeC := fg.cyphers[1]
	if !strings.Contains(writeC, "MERGE (g:GlobalContext") {
		t.Fatalf("expected MERGE GlobalContext, got %s", writeC)
	}
	summary := fg.params[1]["summary"].(string)
	if !strings.Contains(summary, "Alice") || !strings.Contains(summary, "Bob") {
		t.Fatalf("summary missing lesson statements: %s", summary)
	}
	glossary := fg.params[1]["entity_glossary"].([]string)
	if len(glossary) < 4 {
		t.Fatalf("expected 4 entities, got %v", glossary)
	}
}

func TestGlobalContextIndex_QueryErrorPropagates(t *testing.T) {
	fg := &fakeGraph{err: errFake("query failed")}
	s := GlobalContextIndex{Graph: fg, TenantID: "t1"}
	if _, err := s.Run(context.Background(), Input{UserID: "u1"}); err == nil {
		t.Fatal("expected error from failed query")
	}
}

func TestGlobalContextIndex_WriteErrorPropagates(t *testing.T) {
	// First call (read) succeeds; second call (write) fails.
	fg := &callingErrGraph{
		rowsByCall: [][]map[string]interface{}{{
			{"id": "l1", "statement": "Alice", "entities": []interface{}{"Alice"}},
		}},
		errOnCall: 1, // 0-indexed: write is the second call (index 1)
	}
	s := GlobalContextIndex{Graph: fg, TenantID: "t1"}
	if _, err := s.Run(context.Background(), Input{UserID: "u1"}); err == nil {
		t.Fatal("expected error from write failure")
	}
}

func TestGlobalContextIndex_Name(t *testing.T) {
	if (GlobalContextIndex{}).Name() != "global_context_index" {
		t.Fatal("unexpected name")
	}
}

func TestSyncToCache_NilGraphErrors(t *testing.T) {
	s := SyncToCache{Cache: &fakeCache{}}
	if _, err := s.Run(context.Background(), Input{UserID: "u1"}); err == nil {
		t.Fatal("expected error for nil graph")
	}
}

func TestSyncToCache_NilCacheErrors(t *testing.T) {
	s := SyncToCache{Graph: &fakeGraph{}}
	if _, err := s.Run(context.Background(), Input{UserID: "u1"}); err == nil {
		t.Fatal("expected error for nil cache")
	}
}

func TestSyncToCache_MissingTenantErrors(t *testing.T) {
	s := SyncToCache{Graph: &fakeGraph{}, Cache: &fakeCache{}}
	if _, err := s.Run(context.Background(), Input{}); err == nil {
		t.Fatal("expected error for missing tenant")
	}
}

func TestSyncToCache_NoGlobalContextIsNoOp(t *testing.T) {
	fg := &fakeGraph{} // no rows
	fc := &fakeCache{}
	s := SyncToCache{Graph: fg, Cache: fc, TenantID: "t1"}
	res, err := s.Run(context.Background(), Input{UserID: "u1"})
	if err != nil {
		t.Fatal(err)
	}
	if res.Items != 0 {
		t.Fatalf("expected Items=0 for missing global_context, got %d", res.Items)
	}
	if len(fc.keys) != 0 {
		t.Fatalf("expected no cache writes, got %d", len(fc.keys))
	}
}

func TestSyncToCache_WritesPayload(t *testing.T) {
	fg := &fakeGraph{rows: []map[string]interface{}{
		{
			"summary":        "Alice lives in Paris",
			"entity_glossary": []interface{}{"Alice", "Paris"},
			"tenant_id":      "t1",
			"dataset_id":     "d1",
			"updated_at":     time.Now(),
		},
	}}
	fc := &fakeCache{}
	s := SyncToCache{Graph: fg, Cache: fc, TenantID: "t1", DatasetID: "d1"}
	res, err := s.Run(context.Background(), Input{UserID: "u1"})
	if err != nil {
		t.Fatal(err)
	}
	if res.Items != 1 {
		t.Fatalf("expected Items=1, got %d", res.Items)
	}
	if v, ok := fc.keys["global_context:t1:d1"]; !ok || !strings.Contains(v, "Alice") {
		t.Fatalf("expected cache key global_context:t1:d1 with Alice payload, got %v", v)
	}
	if !strings.Contains(fc.keys["global_context:t1:d1"], "entities:") {
		t.Fatalf("expected payload to include entities header, got %s", fc.keys["global_context:t1:d1"])
	}
	if fc.ttls["global_context:t1:d1"] != 24*time.Hour {
		t.Fatalf("expected default TTL=24h, got %v", fc.ttls["global_context:t1:d1"])
	}
}

func TestSyncToCache_CustomTTL(t *testing.T) {
	fg := &fakeGraph{rows: []map[string]interface{}{
		{"summary": "x", "entity_glossary": nil, "tenant_id": "t1", "updated_at": time.Now()},
	}}
	fc := &fakeCache{}
	s := SyncToCache{Graph: fg, Cache: fc, TenantID: "t1", CacheTTL: time.Hour}
	if _, err := s.Run(context.Background(), Input{UserID: "u1"}); err != nil {
		t.Fatal(err)
	}
	if fc.ttls["global_context:t1"] != time.Hour {
		t.Fatalf("expected custom TTL=1h, got %v", fc.ttls["global_context:t1"])
	}
}

func TestSyncToCache_Name(t *testing.T) {
	if (SyncToCache{}).Name() != "sync_to_cache" {
		t.Fatal("unexpected name")
	}
}

func TestBuildSummary_TruncatesAtMaxChars(t *testing.T) {
	rows := []map[string]interface{}{
		{"statement": "aaaa", "entities": []interface{}{"A"}},
		{"statement": "bbbb", "entities": []interface{}{"B"}},
		{"statement": "cccc", "entities": []interface{}{"C"}},
	}
	summary, glossary := buildSummary(rows, 8)
	if len(summary) > 8+3 { // 8 chars + "..." truncation
		t.Fatalf("summary not truncated: %q", summary)
	}
	if len(glossary) != 3 {
		t.Fatalf("expected 3 glossary entities, got %v", glossary)
	}
}

func TestBuildSummary_EmptyRows(t *testing.T) {
	summary, glossary := buildSummary(nil, 100)
	if summary != "" || glossary != nil {
		t.Fatalf("expected empty result, got %q %v", summary, glossary)
	}
}

func TestAsStringSlice(t *testing.T) {
	if v := asStringSlice(nil); v != nil {
		t.Fatalf("nil -> nil, got %v", v)
	}
	if v := asStringSlice([]interface{}{"a", 1, "b"}); len(v) != 2 || v[0] != "a" || v[1] != "b" {
		t.Fatalf("mixed -> only strings, got %v", v)
	}
}

func TestJoinStrings(t *testing.T) {
	if v := joinStrings(nil, ","); v != "" {
		t.Fatalf("nil -> empty, got %q", v)
	}
	if v := joinStrings([]string{"a", "b", "c"}, ","); v != "a,b,c" {
		t.Fatalf("expected a,b,c, got %q", v)
	}
}

// callingErrGraph returns rowsByCall[i] on the i-th call; returns errOnCall's
// error on that index.
type callingErrGraph struct {
	rowsByCall [][]map[string]interface{}
	errOnCall  int
	err        error
	calls      int
}

func (g *callingErrGraph) QueryGraph(_ string, _ map[string]interface{}) ([]map[string]interface{}, error) {
	idx := g.calls
	g.calls++
	if idx == g.errOnCall {
		return nil, errFake("write failed")
	}
	if idx < len(g.rowsByCall) {
		return g.rowsByCall[idx], nil
	}
	return nil, nil
}
