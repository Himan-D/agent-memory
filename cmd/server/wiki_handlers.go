package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"

	"agent-memory/internal/wiki"
)

func (s *APIServer) wikiIngestHandler(w http.ResponseWriter, r *http.Request) {
	if s.wikiSvc == nil {
		jsonError(w, "wiki service not initialized", http.StatusServiceUnavailable)
		return
	}

	var req wiki.IngestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	if req.Content == "" {
		jsonError(w, "content is required", http.StatusBadRequest)
		return
	}

	result, err := s.wikiSvc.Ingest(r.Context(), req)
	if err != nil {
		safeHTTPError(w, r, fmt.Errorf("wiki ingest: %w", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (s *APIServer) wikiQueryHandler(w http.ResponseWriter, r *http.Request) {
	if s.wikiSvc == nil {
		jsonError(w, "wiki service not initialized", http.StatusServiceUnavailable)
		return
	}

	var req wiki.QueryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	if req.Query == "" {
		jsonError(w, "query is required", http.StatusBadRequest)
		return
	}

	result, err := s.wikiSvc.Query(r.Context(), req)
	if err != nil {
		safeHTTPError(w, r, fmt.Errorf("wiki query: %w", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (s *APIServer) wikiLintHandler(w http.ResponseWriter, r *http.Request) {
	if s.wikiSvc == nil {
		jsonError(w, "wiki service not initialized", http.StatusServiceUnavailable)
		return
	}

	var req wiki.LintRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	result, err := s.wikiSvc.Lint(r.Context(), req)
	if err != nil {
		safeHTTPError(w, r, fmt.Errorf("wiki lint: %w", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (s *APIServer) wikiListPagesHandler(w http.ResponseWriter, r *http.Request) {
	if s.wikiSvc == nil {
		jsonError(w, "wiki service not initialized", http.StatusServiceUnavailable)
		return
	}

	pageType := wiki.PageType(r.URL.Query().Get("type"))
	limit := parseIntParam(r, "limit", 50)
	offset := parseIntParam(r, "offset", 0)

	pages := s.wikiSvc.ListPages(pageType, limit, offset)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"pages": pages,
		"total": len(pages),
	})
}

func (s *APIServer) wikiGetPageHandler(w http.ResponseWriter, r *http.Request) {
	if s.wikiSvc == nil {
		jsonError(w, "wiki service not initialized", http.StatusServiceUnavailable)
		return
	}

	pageID := mux.Vars(r)["pageID"]
	if pageID == "" {
		jsonError(w, "page ID required", http.StatusBadRequest)
		return
	}

	page, err := s.wikiSvc.GetPage(pageID)
	if err != nil {
		jsonError(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(page)
}

func (s *APIServer) wikiUpdatePageHandler(w http.ResponseWriter, r *http.Request) {
	if s.wikiSvc == nil {
		jsonError(w, "wiki service not initialized", http.StatusServiceUnavailable)
		return
	}

	pageID := mux.Vars(r)["pageID"]
	if pageID == "" {
		jsonError(w, "page ID required", http.StatusBadRequest)
		return
	}

	var updates map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		jsonError(w, "invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	page, err := s.wikiSvc.UpdatePage(pageID, updates)
	if err != nil {
		jsonError(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(page)
}

func (s *APIServer) wikiDeletePageHandler(w http.ResponseWriter, r *http.Request) {
	if s.wikiSvc == nil {
		jsonError(w, "wiki service not initialized", http.StatusServiceUnavailable)
		return
	}

	pageID := mux.Vars(r)["pageID"]
	if pageID == "" {
		jsonError(w, "page ID required", http.StatusBadRequest)
		return
	}

	if err := s.wikiSvc.DeletePage(pageID); err != nil {
		jsonError(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})
}

func (s *APIServer) wikiListSourcesHandler(w http.ResponseWriter, r *http.Request) {
	if s.wikiSvc == nil {
		jsonError(w, "wiki service not initialized", http.StatusServiceUnavailable)
		return
	}

	limit := parseIntParam(r, "limit", 50)
	offset := parseIntParam(r, "offset", 0)

	sources := s.wikiSvc.ListSources(limit, offset)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"sources": sources,
		"total":   len(sources),
	})
}

func (s *APIServer) wikiGetSourceHandler(w http.ResponseWriter, r *http.Request) {
	if s.wikiSvc == nil {
		jsonError(w, "wiki service not initialized", http.StatusServiceUnavailable)
		return
	}

	sourceID := mux.Vars(r)["sourceID"]
	if sourceID == "" {
		jsonError(w, "source ID required", http.StatusBadRequest)
		return
	}

	source, err := s.wikiSvc.GetSource(sourceID)
	if err != nil {
		jsonError(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(source)
}

func (s *APIServer) wikiStatsHandler(w http.ResponseWriter, r *http.Request) {
	if s.wikiSvc == nil {
		jsonError(w, "wiki service not initialized", http.StatusServiceUnavailable)
		return
	}

	stats := s.wikiSvc.GetStats()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

func (s *APIServer) wikiIndexHandler(w http.ResponseWriter, r *http.Request) {
	if s.wikiSvc == nil {
		jsonError(w, "wiki service not initialized", http.StatusServiceUnavailable)
		return
	}

	index := s.wikiSvc.GetIndex()

	w.Header().Set("Content-Type", "text/markdown")
	w.Write([]byte(index))
}

func (s *APIServer) wikiLogHandler(w http.ResponseWriter, r *http.Request) {
	if s.wikiSvc == nil {
		jsonError(w, "wiki service not initialized", http.StatusServiceUnavailable)
		return
	}

	limit := parseIntParam(r, "limit", 50)
	offset := parseIntParam(r, "offset", 0)

	entries := s.wikiSvc.GetLog(limit, offset)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"entries": entries,
		"total":   len(entries),
	})
}

func parseIntParam(r *http.Request, name string, defaultVal int) int {
	val := r.URL.Query().Get(name)
	if val == "" {
		return defaultVal
	}
	var result int
	fmt.Sscanf(val, "%d", &result)
	if result <= 0 {
		return defaultVal
	}
	return result
}
