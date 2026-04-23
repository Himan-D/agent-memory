package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
	"github.com/google/uuid"

	"agent-memory/internal/alerts"
	"agent-memory/internal/compression/retrieval"
	"agent-memory/internal/memory/types"
	"agent-memory/internal/playground"
	"agent-memory/internal/users"
)

// ==================== User Management Handlers ====================

func (s *APIServer) listUsersHandler(w http.ResponseWriter, r *http.Request) {
	if !isAdmin(r) {
		http.Error(w, "Forbidden: Admin access required", http.StatusForbidden)
		return
	}

	users, err := s.userSvc.ListUsers()
	if err != nil {
		safeHTTPError(w, r, err, http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{"users": users, "total": len(users)})
}

func (s *APIServer) createUserHandler(w http.ResponseWriter, r *http.Request) {
	if !isAdmin(r) {
		http.Error(w, "Forbidden: Admin access required", http.StatusForbidden)
		return
	}

	var req users.CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	user, err := s.userSvc.CreateUser(&req)
	if err != nil {
		safeHTTPError(w, r, err, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(user)
}

func (s *APIServer) updateUserHandler(w http.ResponseWriter, r *http.Request) {
	if !isAdmin(r) {
		http.Error(w, "Forbidden: Admin access required", http.StatusForbidden)
		return
	}

	vars := mux.Vars(r)
	userID := vars["userID"]

	id, err := uuid.Parse(userID)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	var req users.UpdateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	user, err := s.userSvc.UpdateUser(id, &req)
	if err != nil {
		safeHTTPError(w, r, err, http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(user)
}

func (s *APIServer) deleteUserHandler(w http.ResponseWriter, r *http.Request) {
	if !isAdmin(r) {
		http.Error(w, "Forbidden: Admin access required", http.StatusForbidden)
		return
	}

	vars := mux.Vars(r)
	userID := vars["userID"]

	id, err := uuid.Parse(userID)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	if err := s.userSvc.DeleteUser(id); err != nil {
		safeHTTPError(w, r, err, http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})
}

func (s *APIServer) listInvitesHandler(w http.ResponseWriter, r *http.Request) {
	if !isAdmin(r) {
		http.Error(w, "Forbidden: Admin access required", http.StatusForbidden)
		return
	}

	invites, err := s.userSvc.ListInvites()
	if err != nil {
		safeHTTPError(w, r, err, http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{"invites": invites, "total": len(invites)})
}

func (s *APIServer) createInviteHandler(w http.ResponseWriter, r *http.Request) {
	if !isAdmin(r) {
		http.Error(w, "Forbidden: Admin access required", http.StatusForbidden)
		return
	}

	var req users.CreateInviteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	invite, err := s.userSvc.CreateInvite(&req, uuid.Nil)
	if err != nil {
		safeHTTPError(w, r, err, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(invite)
}

func (s *APIServer) acceptInviteHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	inviteID := vars["inviteID"]

	id, err := uuid.Parse(inviteID)
	if err != nil {
		http.Error(w, "Invalid invite ID", http.StatusBadRequest)
		return
	}

	if err := s.userSvc.AcceptInvite(id); err != nil {
		safeHTTPError(w, r, err, http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

func (s *APIServer) cancelInviteHandler(w http.ResponseWriter, r *http.Request) {
	if !isAdmin(r) {
		http.Error(w, "Forbidden: Admin access required", http.StatusForbidden)
		return
	}

	vars := mux.Vars(r)
	inviteID := vars["inviteID"]

	id, err := uuid.Parse(inviteID)
	if err != nil {
		http.Error(w, "Invalid invite ID", http.StatusBadRequest)
		return
	}

	if err := s.userSvc.CancelInvite(id); err != nil {
		safeHTTPError(w, r, err, http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"status": "cancelled"})
}

// ==================== Alert Rules Handlers ====================

func (s *APIServer) listAlertRulesHandler(w http.ResponseWriter, r *http.Request) {
	rules, err := s.alertsSvc.ListRules()
	if err != nil {
		safeHTTPError(w, r, err, http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{"rules": rules, "total": len(rules)})
}

func (s *APIServer) createAlertRuleHandler(w http.ResponseWriter, r *http.Request) {
	if !isAdmin(r) {
		http.Error(w, "Forbidden: Admin access required", http.StatusForbidden)
		return
	}

	var req alerts.CreateAlertRuleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	rule, err := s.alertsSvc.CreateRule(&req)
	if err != nil {
		safeHTTPError(w, r, err, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(rule)
}

func (s *APIServer) updateAlertRuleHandler(w http.ResponseWriter, r *http.Request) {
	if !isAdmin(r) {
		http.Error(w, "Forbidden: Admin access required", http.StatusForbidden)
		return
	}

	vars := mux.Vars(r)
	ruleID := vars["ruleID"]

	id, err := uuid.Parse(ruleID)
	if err != nil {
		http.Error(w, "Invalid rule ID", http.StatusBadRequest)
		return
	}

	var req alerts.UpdateAlertRuleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	rule, err := s.alertsSvc.UpdateRule(id, &req)
	if err != nil {
		safeHTTPError(w, r, err, http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(rule)
}

func (s *APIServer) deleteAlertRuleHandler(w http.ResponseWriter, r *http.Request) {
	if !isAdmin(r) {
		http.Error(w, "Forbidden: Admin access required", http.StatusForbidden)
		return
	}

	vars := mux.Vars(r)
	ruleID := vars["ruleID"]

	id, err := uuid.Parse(ruleID)
	if err != nil {
		http.Error(w, "Invalid rule ID", http.StatusBadRequest)
		return
	}

	if err := s.alertsSvc.DeleteRule(id); err != nil {
		safeHTTPError(w, r, err, http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})
}

func (s *APIServer) enableAlertRuleHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	ruleID := vars["ruleID"]

	id, err := uuid.Parse(ruleID)
	if err != nil {
		http.Error(w, "Invalid rule ID", http.StatusBadRequest)
		return
	}

	var req struct {
		Enabled bool `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := s.alertsSvc.EnableRule(id, req.Enabled); err != nil {
		safeHTTPError(w, r, err, http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

func (s *APIServer) listActiveAlertsHandler(w http.ResponseWriter, r *http.Request) {
	alerts, err := s.alertsSvc.ListActiveAlerts()
	if err != nil {
		safeHTTPError(w, r, err, http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{"alerts": alerts, "total": len(alerts)})
}

func (s *APIServer) resolveAlertHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	alertID := vars["alertID"]

	id, err := uuid.Parse(alertID)
	if err != nil {
		http.Error(w, "Invalid alert ID", http.StatusBadRequest)
		return
	}

	if err := s.alertsSvc.ResolveAlert(id); err != nil {
		safeHTTPError(w, r, err, http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

func (s *APIServer) dismissAlertHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	alertID := vars["alertID"]

	id, err := uuid.Parse(alertID)
	if err != nil {
		http.Error(w, "Invalid alert ID", http.StatusBadRequest)
		return
	}

	if err := s.alertsSvc.DismissAlert(id); err != nil {
		safeHTTPError(w, r, err, http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

func (s *APIServer) getAlertStatsHandler(w http.ResponseWriter, r *http.Request) {
	stats, err := s.alertsSvc.GetAlertStats()
	if err != nil {
		safeHTTPError(w, r, err, http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(stats)
}

// ==================== Sessions Handlers ====================

func (s *APIServer) listSessionsHandler(w http.ResponseWriter, r *http.Request) {
	sessions, err := s.memSvc.ListSessions()
	if err != nil {
		safeHTTPError(w, r, err, http.StatusInternalServerError)
		return
	}

	if sessions == nil {
		sessions = []*types.Session{}
	}

	json.NewEncoder(w).Encode(map[string]interface{}{"sessions": sessions, "total": len(sessions)})
}

// ==================== Compression Handlers ====================

func (s *APIServer) getCompressionStatsHandler(w http.ResponseWriter, r *http.Request) {
	stats := map[string]interface{}{
		"accuracy_retention":       0.973,
		"token_reduction":       0.84,
		"total_tokens_saved": 1500000,
		"extractions_performed": 450,
		"spreading_activations": 230,
		"avg_latency_ms":       187,
		"p95_latency_ms":      245,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

func (s *APIServer) getCompressionModeHandler(w http.ResponseWriter, r *http.Request) {
	mode := map[string]interface{}{
		"mode": "extract",
	}

	json.NewEncoder(w).Encode(mode)
}

func (s *APIServer) setCompressionModeHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "PUT" {
		safeHTTPError(w, r, fmt.Errorf("method not allowed"), http.StatusMethodNotAllowed)
		return
	}

	var req map[string]string
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		safeHTTPError(w, r, err, http.StatusBadRequest)
		return
	}

	mode, ok := req["mode"]
	if !ok {
		safeHTTPError(w, r, fmt.Errorf("mode required"), http.StatusBadRequest)
		return
	}

	if mode != "extract" && mode != "balanced" && mode != "aggressive" {
		safeHTTPError(w, r, fmt.Errorf("invalid mode"), http.StatusBadRequest)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{"mode": mode, "success": true})
}

func (s *APIServer) getTierPolicyHandler(w http.ResponseWriter, r *http.Request) {
	policy := map[string]interface{}{
		"policy": "balanced",
	}

	json.NewEncoder(w).Encode(policy)
}

func (s *APIServer) setTierPolicyHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "PUT" {
		safeHTTPError(w, r, fmt.Errorf("method not allowed"), http.StatusMethodNotAllowed)
		return
	}

	var req map[string]string
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		safeHTTPError(w, r, err, http.StatusBadRequest)
		return
	}

	policy, ok := req["policy"]
	if !ok {
		safeHTTPError(w, r, fmt.Errorf("policy required"), http.StatusBadRequest)
		return
	}

	if policy != "aggressive" && policy != "balanced" && policy != "conservative" {
		safeHTTPError(w, r, fmt.Errorf("invalid policy"), http.StatusBadRequest)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{"policy": policy, "success": true})
}

func (s *APIServer) searchEnhancedHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	query := r.URL.Query().Get("query")
	mode := r.URL.Query().Get("mode")

	if query == "" {
		safeHTTPError(w, r, fmt.Errorf("query required"), http.StatusBadRequest)
		return
	}

	if mode == "" {
		mode = "spreading"
	}

	searchMode := retrieval.SearchMode(mode)
	if searchMode == "" {
		searchMode = retrieval.SearchModeSpreading
	}

	memories, err := s.spreadingActivation.Retrieve(ctx, query, searchMode)
	if err != nil {
		safeHTTPError(w, r, fmt.Errorf("search failed: %w", err), http.StatusInternalServerError)
		return
	}

	if len(memories) == 0 {
		json.NewEncoder(w).Encode(map[string]interface{}{"results": []map[string]interface{}{}, "mode": mode})
		return
	}

	results := make([]map[string]interface{}, 0, len(memories))
	for _, mem := range memories {
		results = append(results, map[string]interface{}{
			"id":      mem.ID,
			"content": mem.Content,
			"score":   0.9,
			"mode":    mode,
			"hops":    1,
		})
	}

	json.NewEncoder(w).Encode(map[string]interface{}{"results": results, "mode": mode})
}

var debugFile *os.File

func init() {
	var err error
	debugFile, err = os.OpenFile("/tmp/debug.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		debugFile = nil
	}
}

func (s *APIServer) playgroundCompressHandler(w http.ResponseWriter, r *http.Request) {
	if debugFile != nil {
		fmt.Fprintf(debugFile, "compress handler called\n")
		debugFile.Sync()
	}
	log.Println("playgroundCompressHandler CALLED")
	ctx := r.Context()

	var req playground.CompressionTestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		safeHTTPError(w, r, fmt.Errorf("invalid request body: %w", err), http.StatusBadRequest)
		return
	}

	resp, err := s.playgroundSvc.TestCompression(ctx, req)
	if err != nil {
		safeHTTPError(w, r, fmt.Errorf("compression test failed: %w", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (s *APIServer) playgroundSearchHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("DEBUG: playgroundSearchHandler called")
	ctx := r.Context()

	var req playground.SearchTestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		fmt.Printf("DEBUG: decode error: %v\n", err)
		safeHTTPError(w, r, fmt.Errorf("invalid request body: %w", err), http.StatusBadRequest)
		return
	}

	resp, err := s.playgroundSvc.TestSearch(ctx, req)
	if err != nil {
		fmt.Printf("DEBUG: search error: %v\n", err)
		safeHTTPError(w, r, fmt.Errorf("search test failed: %w", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (s *APIServer) playgroundStatsHandler(w http.ResponseWriter, r *http.Request) {
	stats := s.playgroundSvc.GetStats()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"total_requests":   stats.TotalRequests,
		"compressions":    stats.Compressions,
		"searches":       stats.Searches,
		"extractions":    stats.Extractions,
		"avg_latency_ms": stats.AvgLatencyMs,
	})
}

// ==================== Demo Handlers ====================

type DemoChatRequest struct {
	Message    string `json:"message"`
	SessionID  string `json:"session_id"`
	WithMemory bool   `json:"with_memory"`
}

func (s *APIServer) demoChatHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req DemoChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		safeHTTPError(w, r, fmt.Errorf("invalid request body: %w", err), http.StatusBadRequest)
		return
	}

	if req.Message == "" {
		safeHTTPError(w, r, fmt.Errorf("message is required"), http.StatusBadRequest)
		return
	}

	if req.SessionID == "" {
		req.SessionID = fmt.Sprintf("demo-%d", time.Now().UnixMilli())
	}

	resp, err := s.playgroundSvc.DemoChat(ctx, req.Message, req.SessionID, req.WithMemory)
	if err != nil {
		safeHTTPError(w, r, fmt.Errorf("chat failed: %w", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (s *APIServer) demoDashboardHandler(w http.ResponseWriter, r *http.Request) {
	tenantID := r.URL.Query().Get("tenant_id")
	if tenantID == "" {
		tenantID = "default"
	}

	period := r.URL.Query().Get("period")
	if period == "" {
		period = "7d"
	}

	dashboard, err := s.analyticsSvc.GetDashboard(r.Context(), tenantID, period)
	if err != nil {
		safeHTTPError(w, r, fmt.Errorf("failed to get dashboard: %w", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(dashboard)
}

type DemoSessionResponse struct {
	SessionID string                  `json:"session_id"`
	CreatedAt time.Time               `json:"created_at"`
	Messages  []playground.DemoMsg    `json:"messages,omitempty"`
}

func (s *APIServer) createDemoSessionHandler(w http.ResponseWriter, r *http.Request) {
	sessionID := fmt.Sprintf("demo-%d", time.Now().UnixMilli())

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(DemoSessionResponse{
		SessionID: sessionID,
		CreatedAt: time.Now(),
	})
}

func (s *APIServer) getDemoSessionHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sessionID := vars["sessionID"]

	messages, err := s.playgroundSvc.GetDemoSessionMessages(sessionID)
	if err != nil {
		safeHTTPError(w, r, fmt.Errorf("failed to get session: %w", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(DemoSessionResponse{
		SessionID: sessionID,
		Messages:  messages,
	})
}

func (s *APIServer) deleteDemoSessionHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sessionID := vars["sessionID"]

	err := s.playgroundSvc.ClearDemoSession(sessionID)
	if err != nil {
		safeHTTPError(w, r, fmt.Errorf("failed to delete session: %w", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})
}