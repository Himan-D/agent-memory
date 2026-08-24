package main

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
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
	if links, ok := updates["links"].([]string); ok {
		page.Links = links
	}
	if tags, ok := updates["tags"].([]string); ok {
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

// wikiExportHandler returns a zip of all wiki pages as Obsidian-compatible .md files.
// Each file is "<page-id>.md" with YAML frontmatter and [[wikilinks]] for linked pages.
func (s *APIServer) wikiExportHandler(w http.ResponseWriter, r *http.Request) {
	if s.wikiSvc == nil {
		jsonError(w, "wiki service not initialized", http.StatusServiceUnavailable)
		return
	}
	pages, _, err := s.wikiSvc.ListPages(r.Context(), 10000, 0)
	if err != nil {
		safeHTTPError(w, r, err, http.StatusInternalServerError)
		return
	}

	buf := new(bytes.Buffer)
	zw := zip.NewWriter(buf)
	for _, p := range pages {
		var md strings.Builder
		md.WriteString("---\n")
		md.WriteString(fmt.Sprintf("id: %s\n", p.ID))
		md.WriteString(fmt.Sprintf("title: %s\n", p.Title))
		md.WriteString(fmt.Sprintf("type: %s\n", p.Type))
		if len(p.Tags) > 0 {
			md.WriteString(fmt.Sprintf("tags: [%s]\n", strings.Join(p.Tags, ", ")))
		}
		md.WriteString(fmt.Sprintf("created: %s\n", p.CreatedAt.Format(time.RFC3339)))
		md.WriteString(fmt.Sprintf("updated: %s\n", p.UpdatedAt.Format(time.RFC3339)))
		md.WriteString("---\n\n")
		md.WriteString(p.Content)
		if len(p.Links) > 0 {
			md.WriteString("\n\n## Links\n")
			for _, l := range p.Links {
				md.WriteString(fmt.Sprintf("- [[%s]]\n", l))
			}
		}
		if f, werr := zw.Create(p.ID + ".md"); werr == nil {
			f.Write([]byte(md.String())) //nolint:errcheck
		}
	}
	zw.Close()

	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", `attachment; filename="wiki-export.zip"`)
	w.Write(buf.Bytes()) //nolint:errcheck
}
