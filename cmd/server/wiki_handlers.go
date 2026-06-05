package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"

	"agent-memory/internal/wiki"
)

// ==================== Wiki Handlers ====================

func parseIntParam(r *http.Request, name string, defaultValue int) int {
	if val := r.URL.Query().Get(name); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil {
			return parsed
		}
	}
	return defaultValue
}

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

	var pages []*wiki.Page
	var total int64
	var err error
	if pageType == "" {
		pages, total, err = s.wikiSvc.ListPages(r.Context(), limit, offset)
	} else {
		// Filter by page type
		allPages, _, err2 := s.wikiSvc.ListPages(r.Context(), 0, 0)
		if err2 != nil {
			safeHTTPError(w, r, err2, http.StatusInternalServerError)
			return
		}
		for _, p := range allPages {
			if p.Type == pageType {
				pages = append(pages, p)
			}
		}
		total = int64(len(pages))
		// Apply pagination
		start := offset
		end := offset + limit
		if start > len(pages) {
			start = len(pages)
		}
		if end > len(pages) {
			end = len(pages)
		}
		pages = pages[start:end]
	}

	if err != nil {
		safeHTTPError(w, r, err, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"pages": pages,
		"count": len(pages),
		"total": total,
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

	page, err := s.wikiSvc.GetPage(r.Context(), pageID)
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

	page, err := s.wikiSvc.GetPage(r.Context(), pageID)
	if err != nil {
		jsonError(w, err.Error(), http.StatusNotFound)
		return
	}

	if title, ok := updates["title"].(string); ok {
		page.Title = title
	}
	if content, ok := updates["content"].(string); ok {
		page.Content = content
	}
	if raw, ok := updates["links"].([]interface{}); ok {
		var links []string
		for _, v := range raw {
			if s, ok := v.(string); ok {
				links = append(links, s)
			}
		}
		page.Links = links
	}
	if raw, ok := updates["tags"].([]interface{}); ok {
		var tags []string
		for _, v := range raw {
			if s, ok := v.(string); ok {
				tags = append(tags, s)
			}
		}
		page.Tags = tags
	}
	page.UpdatedAt = time.Now()

	if err := s.wikiSvc.UpdatePage(r.Context(), page); err != nil {
		safeHTTPError(w, r, err, http.StatusInternalServerError)
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

	if err := s.wikiSvc.DeletePage(r.Context(), pageID); err != nil {
		safeHTTPError(w, r, err, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *APIServer) wikiListSourcesHandler(w http.ResponseWriter, r *http.Request) {
	if s.wikiSvc == nil {
		jsonError(w, "wiki service not initialized", http.StatusServiceUnavailable)
		return
	}

	limit := parseIntParam(r, "limit", 50)
	offset := parseIntParam(r, "offset", 0)

	sources, total, err := s.wikiSvc.ListSources(r.Context(), limit, offset)
	if err != nil {
		safeHTTPError(w, r, err, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"sources": sources,
		"count":   len(sources),
		"total":   total,
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

	source, err := s.wikiSvc.GetSource(r.Context(), sourceID)
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

	stats, err := s.wikiSvc.GetStats(r.Context())
	if err != nil {
		safeHTTPError(w, r, err, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

func (s *APIServer) wikiIndexHandler(w http.ResponseWriter, r *http.Request) {
	if s.wikiSvc == nil {
		jsonError(w, "wiki service not initialized", http.StatusServiceUnavailable)
		return
	}

	index, err := s.wikiSvc.GetIndex(r.Context())
	if err != nil {
		safeHTTPError(w, r, err, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(index)
}

func (s *APIServer) wikiLogHandler(w http.ResponseWriter, r *http.Request) {
	if s.wikiSvc == nil {
		jsonError(w, "wiki service not initialized", http.StatusServiceUnavailable)
		return
	}

	limit := parseIntParam(r, "limit", 100)
	logs, err := s.wikiSvc.GetLog(r.Context(), limit)
	if err != nil {
		safeHTTPError(w, r, err, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(logs)
}
