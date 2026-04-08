package types

import (
	"encoding/json"
	"testing"
	"time"
)

func TestMessage_JSONSerialization(t *testing.T) {
	msg := Message{
		ID:        "msg-1",
		SessionID: "sess-1",
		Role:      "user",
		Content:   "Hello world",
		Timestamp: time.Now().Truncate(time.Second),
	}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var decoded Message
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if decoded.ID != msg.ID {
		t.Errorf("expected ID %s, got %s", msg.ID, decoded.ID)
	}
	if decoded.Role != msg.Role {
		t.Errorf("expected Role %s, got %s", msg.Role, decoded.Role)
	}
}

func TestEntity_JSONSerialization(t *testing.T) {
	entity := Entity{
		ID:   "entity-1",
		Type: "Concept",
		Name: "Machine Learning",
		Properties: map[string]interface{}{
			"field": "AI",
			"year":  1959,
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	data, err := json.Marshal(entity)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var decoded Entity
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if decoded.ID != entity.ID {
		t.Errorf("expected ID %s, got %s", entity.ID, decoded.ID)
	}
	if decoded.Type != entity.Type {
		t.Errorf("expected Type %s, got %s", entity.Type, decoded.Type)
	}
	if decoded.Properties["field"] != "AI" {
		t.Errorf("expected field=AI, got %v", decoded.Properties["field"])
	}
}

func TestEntity_LastSynced(t *testing.T) {
	now := time.Now()
	entity := Entity{
		ID:         "entity-1",
		Type:       "Test",
		Name:       "TestEntity",
		LastSynced: &now,
	}

	data, err := json.Marshal(entity)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var decoded Entity
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if decoded.LastSynced == nil {
		t.Error("LastSynced should not be nil")
	}
}

func TestRelation_Fields(t *testing.T) {
	rel := Relation{
		ID:     "rel-1",
		FromID: "entity-1",
		ToID:   "entity-2",
		Type:   "KNOWS",
		Weight: 0.8,
	}

	if rel.ID != "rel-1" {
		t.Errorf("expected ID rel-1, got %s", rel.ID)
	}
	if rel.FromID != "entity-1" {
		t.Errorf("expected FromID entity-1, got %s", rel.FromID)
	}
	if rel.Weight != 0.8 {
		t.Errorf("expected Weight 0.8, got %f", rel.Weight)
	}
}

func TestSession_Fields(t *testing.T) {
	sess := Session{
		ID:       "sess-1",
		AgentID:  "agent-1",
		Metadata: map[string]interface{}{"purpose": "testing"},
	}

	if sess.ID != "sess-1" {
		t.Errorf("expected ID sess-1, got %s", sess.ID)
	}
	if sess.Metadata["purpose"] != "testing" {
		t.Errorf("expected purpose=testing, got %v", sess.Metadata["purpose"])
	}
}

func TestMemoryResult_Source(t *testing.T) {
	result := MemoryResult{
		Entity: Entity{ID: "e1", Name: "Test"},
		Score:  0.95,
		Text:   "relevant text",
		Source: "qdrant",
	}

	if result.Source != "qdrant" {
		t.Errorf("expected source qdrant, got %s", result.Source)
	}
	if result.Score != 0.95 {
		t.Errorf("expected score 0.95, got %f", result.Score)
	}
}
