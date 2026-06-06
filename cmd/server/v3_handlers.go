package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"agent-memory/internal/memory/types"
)

type v3Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

func (s *APIServer) v3AddHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Messages []v3Message            `json:"messages"`
		UserID   string                 `json:"user_id"`
		AgentID  string                 `json:"agent_id"`
		Metadata map[string]interface{} `json:"metadata"`
		RunID    string                 `json:"run_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if len(req.Messages) == 0 {
		jsonError(w, "messages required", http.StatusBadRequest)
		return
	}
	if req.UserID == "" {
		req.UserID = "default"
	}

	tenantID := getTenantID(r)
	if tenantID == "" {
		tenantID = "default"
	}

	input := formatV3Messages(req.Messages)
	var created []*types.Memory

	if s.v3Extractor != nil {
		result, err := s.v3Extractor.Extract(r.Context(), req.UserID, input)
		if err == nil && len(result.Facts) > 0 {
			for _, fact := range result.Facts {
				mem := &types.Memory{
					ID:       uuid.New().String(),
					Content:  fact.Content,
					UserID:   req.UserID,
					AgentID:  req.AgentID,
					TenantID: tenantID,
					OrgID:    "default",
					Type:     types.MemoryTypeUser,
					Metadata: req.Metadata,
				}
				createdMem, createErr := s.memSvc.CreateMemory(r.Context(), mem)
				if createErr != nil {
					continue
				}
				created = append(created, createdMem)
			}
		}
	}

	if len(created) == 0 {
		mem := &types.Memory{
			Content:  input,
			UserID:   req.UserID,
			AgentID:  req.AgentID,
			TenantID: tenantID,
			OrgID:    "default",
			Type:     types.MemoryTypeUser,
			Metadata: req.Metadata,
		}
		createdMem, err := s.memSvc.CreateMemory(r.Context(), mem)
		if err != nil {
			safeHTTPError(w, r, fmt.Errorf("v3 add: %w", err), http.StatusInternalServerError)
			return
		}
		created = append(created, createdMem)
	}

	results := make([]map[string]interface{}, len(created))
	for i, m := range created {
		results[i] = map[string]interface{}{
			"id":      m.ID,
			"memory":  m.Content,
			"event":   "ADD",
			"user_id": m.UserID,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"results": results,
		"count":   len(results),
	})
}

func (s *APIServer) v3SearchHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Query  string `json:"query"`
		UserID string `json:"user_id"`
		Limit  int    `json:"limit"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.Query == "" {
		jsonError(w, "query required", http.StatusBadRequest)
		return
	}
	if req.UserID == "" {
		req.UserID = "default"
	}
	if req.Limit <= 0 {
		req.Limit = 10
	}

	memories, err := s.memSvc.SearchMemories(r.Context(), &types.SearchRequest{
		Query:  req.Query,
		UserID: req.UserID,
		Limit:  req.Limit,
	})
	if err != nil {
		safeHTTPError(w, r, fmt.Errorf("v3 search: %w", err), http.StatusInternalServerError)
		return
	}

	results := make([]map[string]interface{}, len(memories))
	for i, m := range memories {
		entry := map[string]interface{}{
			"id":      m.MemoryID,
			"memory":  m.Text,
			"score":   m.Score,
			"user_id": req.UserID,
		}
		if m.Metadata != nil && m.Metadata.Metadata != nil {
			entry["metadata"] = m.Metadata.Metadata
		}
		results[i] = entry
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"results": results,
		"count":   len(results),
	})
}

func formatV3Messages(messages []v3Message) string {
	var parts []string
	for _, msg := range messages {
		role := msg.Role
		if role == "" {
			role = "user"
		}
		parts = append(parts, fmt.Sprintf("%s: %s", role, msg.Content))
	}
	return strings.Join(parts, "\n")
}
