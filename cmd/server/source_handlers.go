package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"

	"agent-memory/internal/sources"
)

func (s *APIServer) sourceIngestHandler(w http.ResponseWriter, r *http.Request) {
	if s.sourcesSvc == nil {
		jsonError(w, "sources service not initialized", http.StatusServiceUnavailable)
		return
	}

	var req sources.IngestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	result, err := s.sourcesSvc.Ingest(r.Context(), req)
	if err != nil {
		safeHTTPError(w, r, fmt.Errorf("source ingest: %w", err), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(result)
}

func (s *APIServer) sourceUploadHandler(w http.ResponseWriter, r *http.Request) {
	if s.sourcesSvc == nil {
		jsonError(w, "sources service not initialized", http.StatusServiceUnavailable)
		return
	}
	if err := r.ParseMultipartForm(64 << 20); err != nil {
		safeHTTPError(w, r, fmt.Errorf("parse upload: %w", err), http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		safeHTTPError(w, r, fmt.Errorf("file required: %w", err), http.StatusBadRequest)
		return
	}
	defer file.Close()

	meta := map[string]interface{}{}
	if raw := r.FormValue("metadata"); raw != "" {
		if err := json.Unmarshal([]byte(raw), &meta); err != nil {
			safeHTTPError(w, r, fmt.Errorf("invalid metadata json: %w", err), http.StatusBadRequest)
			return
		}
	}

	tenantID := effectiveTenantID(r)
	userID := r.FormValue("user_id")
	if userID == "" {
		userID = getUserID(r)
	}
	orgID := r.FormValue("org_id")
	if orgID == "" {
		orgID = tenantID
	}
	if meta == nil {
		meta = map[string]interface{}{}
	}
	meta["tenant_id"] = tenantID

	result, err := s.sourcesSvc.Upload(r.Context(), sources.UploadRequest{
		Filename:    header.Filename,
		ContentType: header.Header.Get("Content-Type"),
		Reader:      file,
		Title:       r.FormValue("title"),
		UserID:      userID,
		OrgID:       orgID,
		AgentID:     r.FormValue("agent_id"),
		Metadata:    meta,
	})
	if err != nil {
		safeHTTPError(w, r, fmt.Errorf("source upload: %w", err), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(result)
}

func (s *APIServer) listSourcesHandler(w http.ResponseWriter, r *http.Request) {
	if s.sourcesSvc == nil {
		jsonError(w, "sources service not initialized", http.StatusServiceUnavailable)
		return
	}
	limit := parseIntParam(r, "limit", 50)
	offset := parseIntParam(r, "offset", 0)
	// Scope list to auth tenant via org_id convention when client omits filters.
	userID := r.URL.Query().Get("user_id")
	orgID := r.URL.Query().Get("org_id")
	tenantID := effectiveTenantID(r)
	if orgID == "" && !isAdmin(r) {
		orgID = tenantID
	}
	sourcesList, total, err := s.sourcesSvc.List(r.Context(), userID, orgID, limit, offset)
	if err != nil {
		safeHTTPError(w, r, fmt.Errorf("list sources: %w", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"sources":   sourcesList,
		"count":     len(sourcesList),
		"total":     total,
		"limit":     limit,
		"offset":    offset,
		"tenant_id": tenantID,
	})
}

func (s *APIServer) getSourceHandler(w http.ResponseWriter, r *http.Request) {
	if s.sourcesSvc == nil {
		jsonError(w, "sources service not initialized", http.StatusServiceUnavailable)
		return
	}
	sourceID := mux.Vars(r)["sourceID"]
	if sourceID == "" {
		jsonError(w, "source ID required", http.StatusBadRequest)
		return
	}
	source, err := s.sourcesSvc.Get(r.Context(), sourceID)
	if err != nil {
		safeHTTPError(w, r, fmt.Errorf("get source: %w", err), http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(source)
}

func (s *APIServer) deleteSourceHandler(w http.ResponseWriter, r *http.Request) {
	if s.sourcesSvc == nil {
		jsonError(w, "sources service not initialized", http.StatusServiceUnavailable)
		return
	}
	sourceID := mux.Vars(r)["sourceID"]
	if sourceID == "" {
		jsonError(w, "source ID required", http.StatusBadRequest)
		return
	}
	if err := s.sourcesSvc.Delete(r.Context(), sourceID); err != nil {
		safeHTTPError(w, r, fmt.Errorf("delete source: %w", err), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "deleted", "source_id": sourceID})
}
