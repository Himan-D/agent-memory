package main

import (
	"testing"
	"time"

	"agent-memory/internal/memory/types"
)

func TestV3ContentPrefersExplicitMemory(t *testing.T) {
	content := v3Content(v3MemoryInput{
		Memory:  "explicit memory",
		Content: "fallback content",
		Messages: []v3Message{
			{Role: "user", Content: "message content"},
		},
	})
	if content != "explicit memory" {
		t.Fatalf("expected explicit memory, got %q", content)
	}
}

func TestV3ContentBuildsFromMessages(t *testing.T) {
	content := v3Content(v3MemoryInput{
		Messages: []v3Message{
			{Role: "user", Content: "Alice likes graph memory"},
			{Role: "assistant", Content: "Remembered."},
		},
	})
	want := "user: Alice likes graph memory\nassistant: Remembered."
	if content != want {
		t.Fatalf("content mismatch\nwant: %q\ngot:  %q", want, content)
	}
}

func TestV3MetadataAddsCompatibilityFields(t *testing.T) {
	meta := v3Metadata(v3MemoryInput{
		AppID:             "app-1",
		RunID:             "run-1",
		Categories:        []string{"preference", "work"},
		CustomInstruction: "extract preferences",
		Metadata:          map[string]interface{}{"source": "test"},
	})
	if meta["source"] != "test" || meta["app_id"] != "app-1" || meta["run_id"] != "run-1" {
		t.Fatalf("missing compatibility metadata: %#v", meta)
	}
	if meta["extraction_mode"] != "add_only" || meta["api_version"] != "v3" {
		t.Fatalf("missing v3 markers: %#v", meta)
	}
}

func TestV3MemoryEnvelope(t *testing.T) {
	now := time.Now()
	mem := &types.Memory{
		ID:        "mem-1",
		Content:   "Alice prefers dark mode",
		UserID:    "alice",
		AgentID:   "agent-1",
		SessionID: "run-fallback",
		Category:  "preference",
		Metadata:  map[string]interface{}{"app_id": "app-1"},
		CreatedAt: now,
		UpdatedAt: now,
	}
	env := v3MemoryEnvelope(mem)
	if env["memory"] != mem.Content || env["app_id"] != "app-1" || env["run_id"] != "run-fallback" {
		t.Fatalf("unexpected envelope: %#v", env)
	}
	categories, ok := env["categories"].([]string)
	if !ok || len(categories) != 1 || categories[0] != "preference" {
		t.Fatalf("unexpected categories: %#v", env["categories"])
	}
}

func TestOperationEventStore(t *testing.T) {
	store := newOperationEventStore(time.Hour)
	event := store.create("memory.add", "memory", "mem-1", map[string]interface{}{"memory_id": "mem-1"})
	got, ok := store.get(event.ID)
	if !ok {
		t.Fatal("expected event to be stored")
	}
	if got.Type != "memory.add" || got.Status != "completed" || got.ResourceID != "mem-1" {
		t.Fatalf("unexpected event: %#v", got)
	}
}
