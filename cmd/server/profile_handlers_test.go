package main

import (
	"strings"
	"testing"
	"time"

	"agent-memory/internal/memory/types"
	"agent-memory/internal/sources"
)

func TestBuildProfileDerivesSignals(t *testing.T) {
	now := time.Now().UTC()
	memories := []*types.Memory{
		{
			ID:        "mem_1",
			Content:   "User likes concise technical explanations and wants to improve retrieval benchmarks",
			Category:  "preference",
			Tags:      []string{"benchmarks", "retrieval"},
			CreatedAt: now.Add(-time.Hour),
		},
		{
			ID:        "mem_2",
			Content:   "Source document covers Notion GitHub connector ingestion and source attribution",
			Category:  sources.CategorySource,
			Tags:      []string{"connectors"},
			CreatedAt: now,
		},
	}

	profile := buildProfile("user_1", "", memories, 2)
	if profile.UserID != "user_1" {
		t.Fatalf("user_id = %q", profile.UserID)
	}
	if profile.MemoryCount != 2 {
		t.Fatalf("memory_count = %d, want 2", profile.MemoryCount)
	}
	if len(profile.Preferences["likes"]) == 0 {
		t.Fatalf("likes were not derived: %#v", profile.Preferences)
	}
	if profile.FrequentCategories[sources.CategorySource] != 1 {
		t.Fatalf("source category count missing: %#v", profile.FrequentCategories)
	}
	if profile.TopTags["benchmarks"] != 1 {
		t.Fatalf("top tags missing benchmark: %#v", profile.TopTags)
	}
	if !profile.Signals["has_sources"].(bool) {
		t.Fatalf("has_sources signal false: %#v", profile.Signals)
	}
}

func TestRenderAgentContextIncludesRecentMemories(t *testing.T) {
	now := time.Now().UTC()
	memories := []*types.Memory{{
		ID:        "mem_1",
		Content:   "User wants stronger source ingestion",
		Category:  "goal",
		CreatedAt: now,
	}}
	profile := buildProfile("user_1", "", memories, 1)
	content := renderAgentContext(profile, recentActivity(memories, 1))
	if !strings.Contains(content, "Hystersis memory context") {
		t.Fatalf("missing context header: %s", content)
	}
	if !strings.Contains(content, "User wants stronger source ingestion") {
		t.Fatalf("missing recent memory: %s", content)
	}
}
