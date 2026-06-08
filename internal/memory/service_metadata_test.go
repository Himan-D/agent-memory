package memory

import (
	"testing"

	"agent-memory/internal/memory/types"
)

func TestBuildMemoryMetadataIncludesGraphEntityIDs(t *testing.T) {
	svc := &Service{}
	mem := &types.Memory{
		ID:         "mem-1",
		EntityID:   "entity-primary",
		UserID:     "user-1",
		OrgID:      "org-1",
		AgentID:    "agent-1",
		Type:       types.MemoryTypeUser,
		Importance: types.ImportanceHigh,
		Version:    3,
		Metadata: map[string]interface{}{
			"entity_ids":        []string{"entity-primary", "entity-secondary"},
			"primary_entity_id": "entity-primary",
		},
	}

	meta := svc.buildMemoryMetadata(mem)

	if got := meta["memory_id"]; got != "mem-1" {
		t.Fatalf("expected memory_id to be mem-1, got %#v", got)
	}
	if got := meta["entity_id"]; got != "entity-primary" {
		t.Fatalf("expected entity_id to be primary graph entity, got %#v", got)
	}
	entityIDs, ok := meta["entity_ids"].([]string)
	if !ok {
		t.Fatalf("expected entity_ids []string, got %T", meta["entity_ids"])
	}
	if len(entityIDs) != 2 || entityIDs[0] != "entity-primary" || entityIDs[1] != "entity-secondary" {
		t.Fatalf("unexpected entity_ids: %#v", entityIDs)
	}
}
