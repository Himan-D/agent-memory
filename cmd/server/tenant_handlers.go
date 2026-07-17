package main

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/gorilla/mux"

	tenantpkg "agent-memory/internal/tenant"
)

func (s *APIServer) createTenantHandler(w http.ResponseWriter, r *http.Request) {
	if s.tenantSvc == nil {
		http.Error(w, "tenant service unavailable", http.StatusServiceUnavailable)
		return
	}
	var req struct {
		Name string `json:"name"`
		Slug string `json:"slug"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	userID := getUserID(r)
	ten, err := s.tenantSvc.CreateTenant(r.Context(), req.Name, req.Slug, userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(ten)
}

func (s *APIServer) listTenantsHandler(w http.ResponseWriter, r *http.Request) {
	if s.tenantSvc == nil {
		http.Error(w, "tenant service unavailable", http.StatusServiceUnavailable)
		return
	}
	if isAdmin(r) {
		list, total, err := s.tenantSvc.ListAll(r.Context(), 100, 0)
		if err != nil {
			http.Error(w, "Failed to list tenants", http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(map[string]interface{}{"tenants": list, "total": total})
		return
	}
	userID := getUserID(r)
	if userID == "" {
		// API-key only: return the bound tenant
		tid := effectiveTenantID(r)
		if t, err := s.tenantSvc.Get(r.Context(), tid); err == nil && t != nil {
			json.NewEncoder(w).Encode(map[string]interface{}{"tenants": []interface{}{t}, "total": 1})
			return
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"tenants": []map[string]string{{"id": tid, "slug": tid, "name": tid}},
			"total":   1,
		})
		return
	}
	list, err := s.tenantSvc.ListForUser(r.Context(), userID)
	if err != nil {
		http.Error(w, "Failed to list tenants", http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(map[string]interface{}{"tenants": list, "total": len(list)})
}

func (s *APIServer) getTenantHandler(w http.ResponseWriter, r *http.Request) {
	if s.tenantSvc == nil {
		http.Error(w, "tenant service unavailable", http.StatusServiceUnavailable)
		return
	}
	tenantID := mux.Vars(r)["tenantID"]
	if !s.canAccessTenant(r, tenantID) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}
	t, err := s.tenantSvc.Get(r.Context(), tenantID)
	if err != nil || t == nil {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}
	json.NewEncoder(w).Encode(t)
}

func (s *APIServer) updateTenantHandler(w http.ResponseWriter, r *http.Request) {
	if s.tenantSvc == nil {
		http.Error(w, "tenant service unavailable", http.StatusServiceUnavailable)
		return
	}
	tenantID := mux.Vars(r)["tenantID"]
	if !s.canManageTenant(r, tenantID) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}
	var req struct {
		Name string `json:"name"`
		Plan string `json:"plan"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	t, err := s.tenantSvc.Update(r.Context(), tenantID, req.Name, req.Plan)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	json.NewEncoder(w).Encode(t)
}

func (s *APIServer) listTenantMembersHandler(w http.ResponseWriter, r *http.Request) {
	if s.tenantSvc == nil {
		http.Error(w, "tenant service unavailable", http.StatusServiceUnavailable)
		return
	}
	tenantID := mux.Vars(r)["tenantID"]
	if !s.canAccessTenant(r, tenantID) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}
	members, err := s.tenantSvc.ListMembers(r.Context(), tenantID)
	if err != nil {
		http.Error(w, "Failed to list members", http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(map[string]interface{}{"members": members, "total": len(members)})
}

func (s *APIServer) addTenantMemberHandler(w http.ResponseWriter, r *http.Request) {
	if s.tenantSvc == nil {
		http.Error(w, "tenant service unavailable", http.StatusServiceUnavailable)
		return
	}
	tenantID := mux.Vars(r)["tenantID"]
	if !s.canManageTenant(r, tenantID) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}
	var req struct {
		UserID string `json:"user_id"`
		Email  string `json:"email"`
		Role   string `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if req.UserID == "" {
		http.Error(w, "user_id is required", http.StatusBadRequest)
		return
	}
	role := tenantpkg.Role(req.Role)
	if role == "" {
		role = tenantpkg.RoleMember
	}
	if err := s.tenantSvc.AddMember(r.Context(), tenantID, req.UserID, req.Email, role); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (s *APIServer) removeTenantMemberHandler(w http.ResponseWriter, r *http.Request) {
	if s.tenantSvc == nil {
		http.Error(w, "tenant service unavailable", http.StatusServiceUnavailable)
		return
	}
	vars := mux.Vars(r)
	tenantID := vars["tenantID"]
	userID := vars["userID"]
	if !s.canManageTenant(r, tenantID) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}
	if err := s.tenantSvc.RemoveMember(r.Context(), tenantID, userID); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (s *APIServer) createTenantInviteHandler(w http.ResponseWriter, r *http.Request) {
	if s.tenantSvc == nil {
		http.Error(w, "tenant service unavailable", http.StatusServiceUnavailable)
		return
	}
	tenantID := mux.Vars(r)["tenantID"]
	if !s.canManageTenant(r, tenantID) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}
	var req struct {
		Email string `json:"email"`
		Role  string `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	role := tenantpkg.Role(req.Role)
	if role == "" {
		role = tenantpkg.RoleMember
	}
	inv, err := s.tenantSvc.CreateInvite(r.Context(), tenantID, req.Email, getUserID(r), role)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(inv)
}

func (s *APIServer) acceptTenantInviteHandler(w http.ResponseWriter, r *http.Request) {
	if s.tenantSvc == nil {
		http.Error(w, "tenant service unavailable", http.StatusServiceUnavailable)
		return
	}
	token := mux.Vars(r)["token"]
	userID := getUserID(r)
	if userID == "" {
		http.Error(w, "session required", http.StatusUnauthorized)
		return
	}
	// Optional email from body
	var body struct {
		Email string `json:"email"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	m, err := s.tenantSvc.AcceptInvite(r.Context(), token, userID, body.Email)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	json.NewEncoder(w).Encode(m)
}

func (s *APIServer) switchSessionTenantHandler(w http.ResponseWriter, r *http.Request) {
	if s.tenantSvc == nil {
		http.Error(w, "tenant service unavailable", http.StatusServiceUnavailable)
		return
	}
	var req struct {
		TenantID string `json:"tenant_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.TenantID == "" {
		http.Error(w, "tenant_id required", http.StatusBadRequest)
		return
	}
	userID := getUserID(r)
	tc, err := s.tenantSvc.SwitchTenant(r.Context(), userID, req.TenantID, isAdmin(r))
	if err != nil {
		if err == tenantpkg.ErrForbidden {
			http.Error(w, "Forbidden: not a member of this tenant", http.StatusForbidden)
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Persist on session if bearer token present
	authHeader := r.Header.Get("Authorization")
	if strings.HasPrefix(strings.ToLower(authHeader), "bearer ") {
		tok := strings.TrimSpace(authHeader[7:])
		s.sessionStore.SetActiveTenant(tok, req.TenantID)
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"tenant_id": tc.TenantID,
		"role":      tc.Role,
		"status":    "ok",
	})
}

func (s *APIServer) adminListTenantsHandler(w http.ResponseWriter, r *http.Request) {
	if s.tenantSvc == nil {
		http.Error(w, "tenant service unavailable", http.StatusServiceUnavailable)
		return
	}
	list, total, err := s.tenantSvc.ListAll(r.Context(), 200, 0)
	if err != nil {
		http.Error(w, "Failed to list tenants", http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(map[string]interface{}{"tenants": list, "total": total})
}

func (s *APIServer) adminSuspendTenantHandler(w http.ResponseWriter, r *http.Request) {
	if s.tenantSvc == nil {
		http.Error(w, "tenant service unavailable", http.StatusServiceUnavailable)
		return
	}
	tenantID := mux.Vars(r)["tenantID"]
	if err := s.tenantSvc.Suspend(r.Context(), tenantID); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	json.NewEncoder(w).Encode(map[string]string{"status": "suspended", "tenant_id": tenantID})
}

func (s *APIServer) canAccessTenant(r *http.Request, tenantID string) bool {
	if isAdmin(r) {
		return true
	}
	if effectiveTenantID(r) == tenantID {
		return true
	}
	userID := getUserID(r)
	if userID != "" && s.tenantSvc != nil {
		return s.tenantSvc.IsMember(r.Context(), tenantID, userID)
	}
	return false
}

func (s *APIServer) canManageTenant(r *http.Request, tenantID string) bool {
	if isAdmin(r) {
		return true
	}
	userID := getUserID(r)
	if userID == "" || s.tenantSvc == nil {
		// API keys with write scope on bound tenant may manage
		return effectiveTenantID(r) == tenantID && (getKeyScope(r) == "write" || getKeyScope(r) == "admin")
	}
	m, err := s.tenantSvc.GetMembership(r.Context(), tenantID, userID)
	if err != nil || m == nil {
		return false
	}
	return tenantpkg.CanRoleManage(m.Role)
}

func getUserID(r *http.Request) string {
	if ctx := r.Context(); ctx != nil {
		if uid, ok := ctx.Value("user_id").(string); ok {
			return uid
		}
	}
	if tc, ok := tenantpkg.FromContext(r.Context()); ok {
		return tc.UserID
	}
	return ""
}
