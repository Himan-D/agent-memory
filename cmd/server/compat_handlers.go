package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"

	"agent-memory/internal/memory/types"
)

type operationEvent struct {
	ID         string                 `json:"id"`
	Type       string                 `json:"type"`
	Status     string                 `json:"status"`
	Resource   string                 `json:"resource,omitempty"`
	ResourceID string                 `json:"resource_id,omitempty"`
	Error      string                 `json:"error,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt  time.Time              `json:"created_at"`
	UpdatedAt  time.Time              `json:"updated_at"`
}

type operationEventStore struct {
	mu     sync.RWMutex
	ttl    time.Duration
	events map[string]*operationEvent
}

func newOperationEventStore(ttl time.Duration) *operationEventStore {
	store := &operationEventStore{ttl: ttl, events: map[string]*operationEvent{}}
	go store.cleanupLoop()
	return store
}

func (s *operationEventStore) create(eventType, resource, resourceID string, metadata map[string]interface{}) *operationEvent {
	now := time.Now()
	event := &operationEvent{
		ID:         uuid.New().String(),
		Type:       eventType,
		Status:     "completed",
		Resource:   resource,
		ResourceID: resourceID,
		Metadata:   metadata,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	s.mu.Lock()
	s.events[event.ID] = event
	s.mu.Unlock()
	return event
}

func (s *operationEventStore) get(id string) (*operationEvent, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	event, ok := s.events[id]
	if !ok {
		return nil, false
	}
	cp := *event
	return &cp, true
}

func (s *operationEventStore) cleanupLoop() {
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()
	for range ticker.C {
		cutoff := time.Now().Add(-s.ttl)
		s.mu.Lock()
		for id, event := range s.events {
			if event.UpdatedAt.Before(cutoff) {
				delete(s.events, id)
			}
		}
		s.mu.Unlock()
	}
}

type v3MemoryInput struct {
	Memory            string                 `json:"memory"`
	Content           string                 `json:"content"`
	Messages          []v3Message            `json:"messages"`
	UserID            string                 `json:"user_id"`
	AgentID           string                 `json:"agent_id"`
	AppID             string                 `json:"app_id"`
	RunID             string                 `json:"run_id"`
	OrgID             string                 `json:"org_id"`
	Categories        []string               `json:"categories"`
	Metadata          map[string]interface{} `json:"metadata"`
	CustomInstruction string                 `json:"custom_instructions"`
	SkipProcessing    bool                   `json:"skip_processing"`
}

type v3Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

func (s *APIServer) v3AddMemoriesHandler(w http.ResponseWriter, r *http.Request) {
	var req v3MemoryInput
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		safeHTTPError(w, r, fmt.Errorf("invalid request body: %w", err), http.StatusBadRequest)
		return
	}
	content := v3Content(req)
	if content == "" {
		jsonError(w, "memory, content, or messages are required", http.StatusBadRequest)
		return
	}

	mem := &types.Memory{
		ID:        uuid.New().String(),
		Content:   content,
		Type:      types.MemoryTypeUser,
		UserID:    req.UserID,
		AgentID:   req.AgentID,
		OrgID:     req.OrgID,
		SessionID: req.RunID,
		Category:  firstCategory(req.Categories),
		Tags:      req.Categories,
		Metadata:  v3Metadata(req),
		Status:    types.MemoryStatusActive,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if tenantID := getTenantID(r); tenantID != "" {
		mem.TenantID = tenantID
	}
	created, err := s.memSvc.CreateMemoryWithOptions(context.Background(), mem, req.SkipProcessing)
	if err != nil {
		safeHTTPError(w, r, fmt.Errorf("v3 add memory: %w", err), http.StatusInternalServerError)
		return
	}

	event := s.eventStore.create("memory.add", "memory", created.ID, map[string]interface{}{
		"memory_id": created.ID,
		"user_id":   created.UserID,
		"agent_id":  created.AgentID,
		"app_id":    req.AppID,
		"run_id":    req.RunID,
	})
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"event_id":   event.ID,
		"status":     event.Status,
		"memory_ids": []string{created.ID},
		"results":    []interface{}{v3MemoryEnvelope(created)},
	})
}

func (s *APIServer) v3SearchMemoriesHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Query        string                 `json:"query"`
		Q            string                 `json:"q"`
		UserID       string                 `json:"user_id"`
		AgentID      string                 `json:"agent_id"`
		AppID        string                 `json:"app_id"`
		RunID        string                 `json:"run_id"`
		OrgID        string                 `json:"org_id"`
		Categories   []string               `json:"categories"`
		Limit        int                    `json:"limit"`
		Threshold    float32                `json:"threshold"`
		Rerank       bool                   `json:"rerank"`
		RewriteQuery bool                   `json:"rewrite_query"`
		Filters      map[string]interface{} `json:"filters"`
		Include      map[string]bool        `json:"include"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		safeHTTPError(w, r, fmt.Errorf("invalid request body: %w", err), http.StatusBadRequest)
		return
	}
	query := req.Query
	if query == "" {
		query = req.Q
	}
	if query == "" {
		jsonError(w, "query is required", http.StatusBadRequest)
		return
	}
	if req.Limit <= 0 {
		req.Limit = 10
	}
	if req.Limit > 100 {
		req.Limit = 100
	}
	if req.Threshold == 0 {
		req.Threshold = 0.5
	}
	searchReq := &types.SearchRequest{
		Query:     query,
		Limit:     req.Limit,
		Threshold: req.Threshold,
		UserID:    req.UserID,
		AgentID:   req.AgentID,
		OrgID:     req.OrgID,
		Category:  firstCategory(req.Categories),
		Mode:      "hybrid",
		Rerank:    req.Rerank,
	}
	results, err := s.memSvc.SearchMemories(context.Background(), searchReq)
	if err != nil {
		safeHTTPError(w, r, fmt.Errorf("v3 search memories: %w", err), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(map[string]interface{}{
		"results": v3SearchResults(results, req.Include),
		"count":   len(results),
		"query":   query,
		"mode":    "hybrid",
	})
}

func (s *APIServer) v3ListMemoriesHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserID     string   `json:"user_id"`
		AgentID    string   `json:"agent_id"`
		AppID      string   `json:"app_id"`
		RunID      string   `json:"run_id"`
		OrgID      string   `json:"org_id"`
		Categories []string `json:"categories"`
		Page       int      `json:"page"`
		PageSize   int      `json:"page_size"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		safeHTTPError(w, r, fmt.Errorf("invalid request body: %w", err), http.StatusBadRequest)
		return
	}
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 50
	}
	if req.PageSize > 100 {
		req.PageSize = 100
	}

	memories, err := s.listScopedMemories(req.UserID, req.OrgID)
	if err != nil {
		safeHTTPError(w, r, fmt.Errorf("v3 list memories: %w", err), http.StatusInternalServerError)
		return
	}
	memories = filterV3Memories(memories, req.AgentID, req.AppID, req.RunID, firstCategory(req.Categories))
	total := len(memories)
	start := (req.Page - 1) * req.PageSize
	end := start + req.PageSize
	if start > total {
		start = total
	}
	if end > total {
		end = total
	}
	results := make([]map[string]interface{}, 0, end-start)
	for _, mem := range memories[start:end] {
		results = append(results, v3MemoryEnvelope(mem))
	}
	json.NewEncoder(w).Encode(map[string]interface{}{
		"count":    total,
		"next":     nextPageURL(r, req.Page, req.PageSize, total),
		"previous": previousPageURL(r, req.Page, req.PageSize),
		"results":  results,
	})
}

func (s *APIServer) getOperationEventHandler(w http.ResponseWriter, r *http.Request) {
	eventID := mux.Vars(r)["eventID"]
	event, ok := s.eventStore.get(eventID)
	if !ok {
		safeHTTPError(w, r, fmt.Errorf("event not found"), http.StatusNotFound)
		return
	}
	json.NewEncoder(w).Encode(event)
}

func (s *APIServer) createExportHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserID string `json:"user_id"`
		OrgID  string `json:"org_id"`
		Format string `json:"format"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	if req.UserID == "" {
		req.UserID = r.URL.Query().Get("user_id")
	}
	if req.OrgID == "" {
		req.OrgID = r.URL.Query().Get("org_id")
	}
	if req.UserID == "" && req.OrgID == "" {
		jsonError(w, "user_id or org_id is required", http.StatusBadRequest)
		return
	}
	export, err := s.memSvc.ExportMemories(r.Context(), req.UserID, req.OrgID)
	if err != nil {
		safeHTTPError(w, r, fmt.Errorf("export memories: %w", err), http.StatusInternalServerError)
		return
	}
	event := s.eventStore.create("memory.export", "export", "", map[string]interface{}{"user_id": req.UserID, "org_id": req.OrgID})
	json.NewEncoder(w).Encode(map[string]interface{}{
		"event_id": event.ID,
		"status":   event.Status,
		"format":   "json",
		"export":   export,
	})
}

func (s *APIServer) createImportHandler(w http.ResponseWriter, r *http.Request) {
	var req types.MemoryImport
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		safeHTTPError(w, r, fmt.Errorf("invalid request body: %w", err), http.StatusBadRequest)
		return
	}
	count, err := s.memSvc.ImportMemories(r.Context(), &req)
	if err != nil {
		safeHTTPError(w, r, fmt.Errorf("import memories: %w", err), http.StatusInternalServerError)
		return
	}
	event := s.eventStore.create("memory.import", "import", "", map[string]interface{}{"imported": count})
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"event_id": event.ID,
		"status":   event.Status,
		"imported": count,
	})
}

func (s *APIServer) listScopedMemories(userID, orgID string) ([]*types.Memory, error) {
	if userID != "" {
		return s.memSvc.GetMemoriesByUser(context.Background(), userID)
	}
	if orgID != "" {
		return s.memSvc.GetMemoriesByOrg(context.Background(), orgID)
	}
	return s.memSvc.GetAllMemories(context.Background())
}

func v3Content(req v3MemoryInput) string {
	if req.Memory != "" {
		return req.Memory
	}
	if req.Content != "" {
		return req.Content
	}
	var content string
	for _, msg := range req.Messages {
		if msg.Content == "" {
			continue
		}
		if content != "" {
			content += "\n"
		}
		if msg.Role != "" {
			content += msg.Role + ": "
		}
		content += msg.Content
	}
	return content
}

func v3Metadata(req v3MemoryInput) map[string]interface{} {
	meta := map[string]interface{}{}
	for k, v := range req.Metadata {
		meta[k] = v
	}
	if req.AppID != "" {
		meta["app_id"] = req.AppID
	}
	if req.RunID != "" {
		meta["run_id"] = req.RunID
	}
	if req.CustomInstruction != "" {
		meta["custom_instructions"] = req.CustomInstruction
	}
	if len(req.Categories) > 0 {
		meta["categories"] = req.Categories
	}
	meta["extraction_mode"] = "add_only"
	meta["api_version"] = "v3"
	return meta
}

func firstCategory(categories []string) string {
	if len(categories) == 0 {
		return ""
	}
	return categories[0]
}

func v3MemoryEnvelope(mem *types.Memory) map[string]interface{} {
	meta := map[string]interface{}{}
	for k, v := range mem.Metadata {
		meta[k] = v
	}
	categories := mem.Tags
	if len(categories) == 0 && mem.Category != "" {
		categories = []string{mem.Category}
	}
	return map[string]interface{}{
		"id":         mem.ID,
		"memory":     mem.Content,
		"user_id":    mem.UserID,
		"agent_id":   mem.AgentID,
		"app_id":     stringFromMeta(meta, "app_id"),
		"run_id":     stringFromMeta(meta, "run_id", mem.SessionID),
		"metadata":   meta,
		"categories": categories,
		"hash":       mem.ContentHash,
		"created_at": mem.CreatedAt,
		"updated_at": mem.UpdatedAt,
	}
}

func v3SearchResults(results []types.MemoryResult, include map[string]bool) []map[string]interface{} {
	out := make([]map[string]interface{}, 0, len(results))
	for _, result := range results {
		item := map[string]interface{}{
			"id":        result.MemoryID,
			"memory":    result.Text,
			"score":     result.Score,
			"metadata":  result.Metadata,
			"entity":    result.Entity,
			"source":    result.Source,
			"documents": nil,
		}
		if result.Metadata != nil {
			item["id"] = result.Metadata.ID
			item["memory"] = result.Metadata.Content
			item["metadata"] = result.Metadata.Metadata
			if include["documents"] || include["summaries"] {
				item["document"] = map[string]interface{}{
					"source_id": result.Metadata.Metadata["source_id"],
					"title":     result.Metadata.Metadata["title"],
					"provider":  result.Metadata.Metadata["provider"],
					"url":       result.Metadata.Metadata["url"],
				}
			}
		}
		out = append(out, item)
	}
	return out
}

func filterV3Memories(memories []*types.Memory, agentID, appID, runID, category string) []*types.Memory {
	filtered := make([]*types.Memory, 0, len(memories))
	for _, mem := range memories {
		if agentID != "" && mem.AgentID != agentID {
			continue
		}
		if category != "" && mem.Category != category {
			continue
		}
		if appID != "" && stringFromMeta(mem.Metadata, "app_id") != appID {
			continue
		}
		if runID != "" && stringFromMeta(mem.Metadata, "run_id", mem.SessionID) != runID {
			continue
		}
		filtered = append(filtered, mem)
	}
	return filtered
}

func stringFromMeta(meta map[string]interface{}, key string, fallback ...string) string {
	if value, ok := meta[key].(string); ok {
		return value
	}
	if len(fallback) > 0 {
		return fallback[0]
	}
	return ""
}

func nextPageURL(r *http.Request, page, pageSize, total int) interface{} {
	if page*pageSize >= total {
		return nil
	}
	return fmt.Sprintf("%s?page=%d&page_size=%d", r.URL.Path, page+1, pageSize)
}

func previousPageURL(r *http.Request, page, pageSize int) interface{} {
	if page <= 1 {
		return nil
	}
	return fmt.Sprintf("%s?page=%d&page_size=%d", r.URL.Path, page-1, pageSize)
}

func parsePageParam(r *http.Request, key string, fallback int) int {
	if raw := r.URL.Query().Get(key); raw != "" {
		if value, err := strconv.Atoi(raw); err == nil && value > 0 {
			return value
		}
	}
	return fallback
}
