// Package cogni provides HTTP handlers that expose the session, distillation,
// improvement, and rollback subsystems added by the Cognee-inspired work.
// The package is purely additive: registering its routes does not change any
// existing handler or response format. To opt in, call RegisterRoutes on a
// gorilla/mux router from cmd/server.
package cogni

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/gorilla/mux"

	"agent-memory/internal/memory/improve"
	"agent-memory/internal/memory/rollback"
	"agent-memory/internal/retrieval"
	"agent-memory/internal/session"
)

// Deps wires the handlers to their dependencies. Any field may be nil; the
// corresponding endpoints return 503 Service Unavailable in that case.
type Deps struct {
	SessionManager *session.Manager
	Distiller      *session.Distiller
	Improver       *improve.Pipeline
	Ledger         rollback.Ledger
	// Retriever is the optional enhanced retriever exposed via
	// /search/enhanced?mode=vector|multisignal|graph.
	Retriever        retrieval.BaseRetriever
	DefaultTopK      int
}

// RegisterRoutes installs all Cognee-inspired endpoints on r. The router's
// prefix middleware (if any) still applies.
func RegisterRoutes(r *mux.Router, deps Deps) {
	if r == nil {
		return
	}
	h := &handler{deps: deps}

	r.HandleFunc("/sessions/{id}/qa", h.addQA).Methods(http.MethodPost, http.MethodOptions)
	r.HandleFunc("/sessions/{id}/qa", h.listQA).Methods(http.MethodGet)
	r.HandleFunc("/sessions/{id}/improve", h.improveSession).Methods(http.MethodPost)
	r.HandleFunc("/sessions/{id}/distill", h.distillSession).Methods(http.MethodPost)
	r.HandleFunc("/pipeline/runs/{id}/rollback", h.rollbackPipelineRun).Methods(http.MethodPost)
	r.HandleFunc("/search/enhanced", h.searchEnhanced).Methods(http.MethodGet)
}

// handler holds dependencies and serves the routes. Methods are on the value
// receiver because the handler carries no mutable state.
type handler struct {
	deps Deps
}

// safeHTTPError writes an error JSON response. Kept inline (not imported
// from cmd/server) so this package has zero compile-time coupling to the
// server binary.
func safeHTTPError(w http.ResponseWriter, _ *http.Request, err error, status int) {
	if err == nil {
		err = errors.New("unknown error")
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"error":  err.Error(),
		"status": status,
	})
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

// ---- Request/response shapes ----

type addQARequest struct {
	UserID     string   `json:"user_id"`
	Question   string   `json:"question"`
	Answer     string   `json:"answer"`
	Context    string   `json:"context,omitempty"`
	Feedback   *float64 `json:"feedback_score,omitempty"`
	UsedGraphIDs []string `json:"used_graph_element_ids,omitempty"`
}

type addQAResponse struct {
	Turn session.QATurn `json:"turn"`
}

// ---- Handlers ----

func (h *handler) addQA(w http.ResponseWriter, r *http.Request) {
	if h.deps.SessionManager == nil {
		safeHTTPError(w, r, errors.New("session manager not configured"), http.StatusServiceUnavailable)
		return
	}
	var req addQARequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		safeHTTPError(w, r, err, http.StatusBadRequest)
		return
	}
	if req.UserID == "" {
		safeHTTPError(w, r, errors.New("user_id is required"), http.StatusBadRequest)
		return
	}
	if req.Question == "" && req.Answer == "" {
		safeHTTPError(w, r, errors.New("question and answer cannot both be empty"), http.StatusBadRequest)
		return
	}
	sessionID := mux.Vars(r)["id"]
	turn, err := h.deps.SessionManager.AddQATurn(
		r.Context(), req.UserID, sessionID,
		req.Question, req.Answer, req.Context,
		req.Feedback, req.UsedGraphIDs,
	)
	if err != nil {
		safeHTTPError(w, r, err, http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusCreated, addQAResponse{Turn: turn})
}

func (h *handler) listQA(w http.ResponseWriter, r *http.Request) {
	if h.deps.SessionManager == nil {
		safeHTTPError(w, r, errors.New("session manager not configured"), http.StatusServiceUnavailable)
		return
	}
	sessionID := mux.Vars(r)["id"]
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		safeHTTPError(w, r, errors.New("user_id query param is required"), http.StatusBadRequest)
		return
	}
	lastN := 0
	if v := r.URL.Query().Get("last_n"); v != "" {
		var n int
		if _, err := jsonDecodeInt(v, &n); err != nil {
			safeHTTPError(w, r, err, http.StatusBadRequest)
			return
		}
		lastN = n
	}
	turns, err := h.deps.SessionManager.GetSession(r.Context(), userID, sessionID, lastN)
	if err != nil {
		safeHTTPError(w, r, err, http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"session_id": sessionID,
		"turns":      turns,
		"count":      len(turns),
	})
}

func (h *handler) improveSession(w http.ResponseWriter, r *http.Request) {
	if h.deps.Improver == nil {
		safeHTTPError(w, r, errors.New("improver not configured"), http.StatusServiceUnavailable)
		return
	}
	var req struct {
		UserID             string   `json:"user_id"`
		BuildGlobalContext bool     `json:"build_global_context"`
		RunSyncToCache     bool     `json:"run_sync_to_cache"`
		SessionIDs         []string `json:"session_ids,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		safeHTTPError(w, r, err, http.StatusBadRequest)
		return
	}
	if req.UserID == "" {
		safeHTTPError(w, r, errors.New("user_id is required"), http.StatusBadRequest)
		return
	}
	sessionID := mux.Vars(r)["id"]
	if len(req.SessionIDs) == 0 {
		req.SessionIDs = []string{sessionID}
	}
	in := improve.Input{
		UserID:             req.UserID,
		SessionIDs:         req.SessionIDs,
		BuildGlobalContext: req.BuildGlobalContext,
		RunSyncToCache:     req.RunSyncToCache,
	}
	out := h.deps.Improver.Run(r.Context(), in)
	writeJSON(w, http.StatusOK, out)
}

func (h *handler) distillSession(w http.ResponseWriter, r *http.Request) {
	if h.deps.Distiller == nil {
		safeHTTPError(w, r, errors.New("distiller not configured"), http.StatusServiceUnavailable)
		return
	}
	var req struct {
		UserID string `json:"user_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && !errors.Is(err, io.EOF) {
		safeHTTPError(w, r, err, http.StatusBadRequest)
		return
	}
	if req.UserID == "" {
		req.UserID = r.URL.Query().Get("user_id")
	}
	if req.UserID == "" {
		safeHTTPError(w, r, errors.New("user_id is required"), http.StatusBadRequest)
		return
	}
	sessionID := mux.Vars(r)["id"]
	res := h.deps.Distiller.Distill(r.Context(), req.UserID, sessionID)
	writeJSON(w, http.StatusOK, res)
}

func (h *handler) rollbackPipelineRun(w http.ResponseWriter, r *http.Request) {
	if h.deps.Ledger == nil {
		safeHTTPError(w, r, errors.New("ledger not configured"), http.StatusServiceUnavailable)
		return
	}
	runID := mux.Vars(r)["id"]
	if runID == "" {
		safeHTTPError(w, r, errors.New("pipeline run id is required"), http.StatusBadRequest)
		return
	}
	nodes, err := h.deps.Ledger.Rollback(r.Context(), runID)
	if err != nil {
		safeHTTPError(w, r, err, http.StatusConflict)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"pipeline_run_id": runID,
		"rolled_back_nodes": len(nodes),
		"node_ids":        nodes,
	})
}

func (h *handler) searchEnhanced(w http.ResponseWriter, r *http.Request) {
	if h.deps.Retriever == nil {
		safeHTTPError(w, r, errors.New("retriever not configured"), http.StatusServiceUnavailable)
		return
	}
	q := r.URL.Query().Get("query")
	if q == "" {
		safeHTTPError(w, r, errors.New("query is required"), http.StatusBadRequest)
		return
	}
	comp, err := h.deps.Retriever.GetCompletion(r.Context(), q)
	if err != nil {
		safeHTTPError(w, r, err, http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"mode":       h.deps.Retriever.Name(),
		"query":      comp.Query,
		"answer":     comp.Answer,
		"context":    comp.Context,
		"citations":  comp.Citations,
	})
}

// ---- helpers ----

// jsonDecodeInt parses a string into an int. Local helper to avoid pulling
// strconv into the handler hot path.
func jsonDecodeInt(s string, dst *int) (int, error) {
	if s == "" {
		return 0, errors.New("empty integer")
	}
	n := 0
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c < '0' || c > '9' {
			return 0, errors.New("invalid integer: " + s)
		}
		n = n*10 + int(c-'0')
	}
	*dst = n
	return n, nil
}
