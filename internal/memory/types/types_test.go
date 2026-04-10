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

func TestMemory_WithTagsAndImportance(t *testing.T) {
	mem := Memory{
		ID:          "mem-1",
		UserID:      "user-1",
		Type:        MemoryTypeConversation,
		Content:     "Test memory content",
		Tags:        []string{"important", "work", "project-x"},
		Importance:  ImportanceHigh,
		Version:     1,
		AccessCount: 42,
	}

	if mem.ID != "mem-1" {
		t.Errorf("expected ID mem-1, got %s", mem.ID)
	}
	if len(mem.Tags) != 3 {
		t.Errorf("expected 3 tags, got %d", len(mem.Tags))
	}
	if mem.Importance != ImportanceHigh {
		t.Errorf("expected ImportanceHigh, got %s", mem.Importance)
	}
	if mem.Version != 1 {
		t.Errorf("expected version 1, got %d", mem.Version)
	}
	if mem.AccessCount != 42 {
		t.Errorf("expected access count 42, got %d", mem.AccessCount)
	}
}

func TestMemory_JSONSerialization_WithNewFields(t *testing.T) {
	now := time.Now()
	mem := Memory{
		ID:               "mem-1",
		TenantID:         "tenant-1",
		UserID:           "user-1",
		OrgID:            "org-1",
		AgentID:          "agent-1",
		SessionID:        "sess-1",
		Type:             MemoryTypeConversation,
		Content:          "Important conversation",
		Category:         "work",
		Tags:             []string{"urgent", "meeting"},
		Importance:       ImportanceCritical,
		Status:           MemoryStatusActive,
		Immutable:        false,
		ParentMemoryID:   "parent-1",
		RelatedMemoryIDs: []string{"rel-1", "rel-2"},
		Version:          3,
		AccessCount:      100,
		CreatedAt:        now,
		UpdatedAt:        now,
		LastAccessed:     &now,
		ExpirationDate:   &now,
	}

	data, err := json.Marshal(mem)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var decoded Memory
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if decoded.ID != mem.ID {
		t.Errorf("expected ID %s, got %s", mem.ID, decoded.ID)
	}
	if len(decoded.Tags) != 2 {
		t.Errorf("expected 2 tags, got %d", len(decoded.Tags))
	}
	if decoded.Importance != ImportanceCritical {
		t.Errorf("expected ImportanceCritical, got %s", decoded.Importance)
	}
	if decoded.Version != 3 {
		t.Errorf("expected version 3, got %d", decoded.Version)
	}
	if len(decoded.RelatedMemoryIDs) != 2 {
		t.Errorf("expected 2 related IDs, got %d", len(decoded.RelatedMemoryIDs))
	}
}

func TestImportanceLevel_Constants(t *testing.T) {
	tests := []struct {
		level    ImportanceLevel
		expected string
	}{
		{ImportanceCritical, "critical"},
		{ImportanceHigh, "high"},
		{ImportanceMedium, "medium"},
		{ImportanceLow, "low"},
	}

	for _, tt := range tests {
		if string(tt.level) != tt.expected {
			t.Errorf("expected %s, got %s", tt.expected, string(tt.level))
		}
	}
}

func TestMemoryLinkType_Constants(t *testing.T) {
	tests := []struct {
		linkType MemoryLinkType
		expected string
	}{
		{MemoryLinkParent, "parent"},
		{MemoryLinkRelated, "related"},
		{MemoryLinkReply, "reply"},
		{MemoryLinkCite, "cite"},
	}

	for _, tt := range tests {
		if string(tt.linkType) != tt.expected {
			t.Errorf("expected %s, got %s", tt.expected, string(tt.linkType))
		}
	}
}

func TestMemoryLink_Fields(t *testing.T) {
	link := MemoryLink{
		ID:       "link-1",
		TenantID: "tenant-1",
		FromID:   "mem-1",
		ToID:     "mem-2",
		Type:     MemoryLinkRelated,
		Weight:   0.75,
		Metadata: map[string]interface{}{
			"context": "related discussion",
		},
	}

	if link.ID != "link-1" {
		t.Errorf("expected ID link-1, got %s", link.ID)
	}
	if link.FromID != "mem-1" {
		t.Errorf("expected FromID mem-1, got %s", link.FromID)
	}
	if link.Type != MemoryLinkRelated {
		t.Errorf("expected type related, got %s", link.Type)
	}
	if link.Weight != 0.75 {
		t.Errorf("expected weight 0.75, got %f", link.Weight)
	}
}

func TestMemoryVersion_Fields(t *testing.T) {
	now := time.Now()
	version := MemoryVersion{
		ID:        "ver-1",
		MemoryID:  "mem-1",
		Version:   2,
		Content:   "Original content that was updated",
		Metadata:  map[string]interface{}{"edited_by": "admin"},
		CreatedBy: "admin",
		CreatedAt: now,
	}

	if version.ID != "ver-1" {
		t.Errorf("expected ID ver-1, got %s", version.ID)
	}
	if version.MemoryID != "mem-1" {
		t.Errorf("expected MemoryID mem-1, got %s", version.MemoryID)
	}
	if version.Version != 2 {
		t.Errorf("expected version 2, got %d", version.Version)
	}
	if version.CreatedBy != "admin" {
		t.Errorf("expected created_by admin, got %s", version.CreatedBy)
	}
}

func TestHybridSearchRequest_Fields(t *testing.T) {
	now := time.Now()
	req := HybridSearchRequest{
		Query:         "machine learning",
		SemanticLimit: 20,
		KeywordLimit:  10,
		Threshold:     0.7,
		Boost:         1.5,
		MemoryType:    MemoryTypeConversation,
		UserID:        "user-1",
		Tags:          []string{"ai", "ml"},
		Importance:    ImportanceHigh,
		DateFrom:      &now,
		DateTo:        &now,
	}

	if req.Query != "machine learning" {
		t.Errorf("expected query 'machine learning', got %s", req.Query)
	}
	if req.SemanticLimit != 20 {
		t.Errorf("expected semantic limit 20, got %d", req.SemanticLimit)
	}
	if req.Boost != 1.5 {
		t.Errorf("expected boost 1.5, got %f", req.Boost)
	}
	if len(req.Tags) != 2 {
		t.Errorf("expected 2 tags, got %d", len(req.Tags))
	}
	if req.Importance != ImportanceHigh {
		t.Errorf("expected importance high, got %s", req.Importance)
	}
}

func TestPaginatedResponse_Fields(t *testing.T) {
	resp := PaginatedResponse{
		Items:      []string{"item1", "item2", "item3"},
		Page:       2,
		PageSize:   10,
		TotalItems: 25,
		TotalPages: 3,
		HasMore:    true,
	}

	if resp.Page != 2 {
		t.Errorf("expected page 2, got %d", resp.Page)
	}
	if resp.TotalPages != 3 {
		t.Errorf("expected 3 total pages, got %d", resp.TotalPages)
	}
	if !resp.HasMore {
		t.Error("expected HasMore to be true")
	}
}

func TestMemoryExport_Fields(t *testing.T) {
	now := time.Now()
	export := MemoryExport{
		Version:    "1.0",
		ExportedAt: now,
		Memories: []Memory{
			{ID: "mem-1", Content: "Memory 1"},
			{ID: "mem-2", Content: "Memory 2"},
		},
	}

	if export.Version != "1.0" {
		t.Errorf("expected version 1.0, got %s", export.Version)
	}
	if len(export.Memories) != 2 {
		t.Errorf("expected 2 memories, got %d", len(export.Memories))
	}
}

func TestMemoryImport_Fields(t *testing.T) {
	imp := MemoryImport{
		Memories: []Memory{
			{ID: "mem-1", Content: "Memory 1"},
		},
		Overwrite: true,
		MergeMode: "append",
	}

	if !imp.Overwrite {
		t.Error("expected Overwrite to be true")
	}
	if imp.MergeMode != "append" {
		t.Errorf("expected merge mode append, got %s", imp.MergeMode)
	}
}

func TestMemoryStats_Fields(t *testing.T) {
	stats := MemoryStats{
		TotalMemories:  150,
		ByCategory:     map[string]int64{"work": 100, "personal": 50},
		ByType:         map[string]int64{"conversation": 120, "user": 30},
		ByImportance:   map[string]int64{"high": 30, "medium": 80, "low": 40},
		ByStatus:       map[string]int64{"active": 140, "archived": 10},
		AvgAccessCount: 5.5,
		TopTags: []TagCount{
			{Tag: "important", Count: 25},
			{Tag: "work", Count: 20},
		},
		RecentMemories:  45,
		ExpiredMemories: 5,
	}

	if stats.TotalMemories != 150 {
		t.Errorf("expected 150 total memories, got %d", stats.TotalMemories)
	}
	if stats.ByImportance["high"] != 30 {
		t.Errorf("expected 30 high importance, got %d", stats.ByImportance["high"])
	}
	if len(stats.TopTags) != 2 {
		t.Errorf("expected 2 top tags, got %d", len(stats.TopTags))
	}
	if stats.AvgAccessCount != 5.5 {
		t.Errorf("expected avg access count 5.5, got %f", stats.AvgAccessCount)
	}
}

func TestMemoryInsight_Fields(t *testing.T) {
	insight := MemoryInsight{
		Type:        "high_memory_volume",
		Description: "You have over 100 memories stored",
		Memories:    []string{"mem-1", "mem-2"},
		Metadata:    map[string]interface{}{"threshold": 100},
	}

	if insight.Type != "high_memory_volume" {
		t.Errorf("expected type high_memory_volume, got %s", insight.Type)
	}
	if len(insight.Memories) != 2 {
		t.Errorf("expected 2 memory IDs, got %d", len(insight.Memories))
	}
}

func TestTagCount_Fields(t *testing.T) {
	tagCount := TagCount{
		Tag:   "machine-learning",
		Count: 42,
	}

	if tagCount.Tag != "machine-learning" {
		t.Errorf("expected tag machine-learning, got %s", tagCount.Tag)
	}
	if tagCount.Count != 42 {
		t.Errorf("expected count 42, got %d", tagCount.Count)
	}
}

func TestPaginationParams_Fields(t *testing.T) {
	params := PaginationParams{
		Page:     5,
		PageSize: 25,
	}

	if params.Page != 5 {
		t.Errorf("expected page 5, got %d", params.Page)
	}
	if params.PageSize != 25 {
		t.Errorf("expected page size 25, got %d", params.PageSize)
	}
}

func TestMemoryHistory_ActionConstants(t *testing.T) {
	actions := []struct {
		action   HistoryAction
		expected string
	}{
		{HistoryActionCreate, "create"},
		{HistoryActionUpdate, "update"},
		{HistoryActionDelete, "delete"},
		{HistoryActionArchive, "archive"},
		{HistoryActionFeedback, "feedback"},
	}

	for _, aa := range actions {
		if string(aa.action) != aa.expected {
			t.Errorf("expected %s, got %s", aa.expected, string(aa.action))
		}
	}
}

func TestMemoryType_Constants(t *testing.T) {
	types := []struct {
		memType  MemoryType
		expected string
	}{
		{MemoryTypeConversation, "conversation"},
		{MemoryTypeSession, "session"},
		{MemoryTypeUser, "user"},
		{MemoryTypeOrg, "org"},
	}

	for _, tt := range types {
		if string(tt.memType) != tt.expected {
			t.Errorf("expected %s, got %s", tt.expected, string(tt.memType))
		}
	}
}

func TestFeedbackType_Constants(t *testing.T) {
	types := []struct {
		fbType   FeedbackType
		expected string
	}{
		{FeedbackPositive, "positive"},
		{FeedbackNegative, "negative"},
		{FeedbackVeryNegative, "very_negative"},
	}

	for _, tt := range types {
		if string(tt.fbType) != tt.expected {
			t.Errorf("expected %s, got %s", tt.expected, string(tt.fbType))
		}
	}
}
