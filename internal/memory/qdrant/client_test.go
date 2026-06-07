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
