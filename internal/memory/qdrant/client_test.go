package qdrant

import "testing"

func TestQdrantPointIDUsesUUIDMemoryID(t *testing.T) {
	id := "11111111-1111-1111-1111-111111111111"

	if got := qdrantPointID(id); got != id {
		t.Fatalf("expected UUID memory ID to be used as point ID, got %q", got)
	}
}

func TestQdrantPointIDIsStableForNonUUIDMemoryID(t *testing.T) {
	first := qdrantPointID("memory-123")
	second := qdrantPointID("memory-123")

	if first == "" {
		t.Fatal("expected non-empty point ID")
	}
	if first != second {
		t.Fatalf("expected deterministic point ID, got %q then %q", first, second)
	}
}

func TestQdrantValuePreservesStringSlices(t *testing.T) {
	value := toQdrantValue([]string{"entity-1", "entity-2"})

	got, ok := fromQdrantValue(value).([]interface{})
	if !ok {
		t.Fatalf("expected []interface{} round trip, got %T", fromQdrantValue(value))
	}
	if len(got) != 2 || got[0] != "entity-1" || got[1] != "entity-2" {
		t.Fatalf("unexpected round trip values: %#v", got)
	}
}
