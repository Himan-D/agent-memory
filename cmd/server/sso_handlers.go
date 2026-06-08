package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/gorilla/mux"

	"agent-memory/internal/sso"
)

type ssoProviderRequest struct {
	TenantID     string `json:"tenant_id"`
	ProviderType string `json:"provider_type"`
	ClientID     string `json:"client_id,omitempty"`
	ClientSecret string `json:"client_secret,omitempty"`
	IssuerURL    string `json:"issuer_url"`
	CallbackURL  string `json:"callback_url,omitempty"`
	Certificate  string `json:"certificate,omitempty"`
}

func (s *APIServer) registerSSOProviderHandler(w http.ResponseWriter, r *http.Request) {
	var req ssoProviderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		safeHTTPError(w, r, fmt.Errorf("invalid request body: %w", err), http.StatusBadRequest)
		return
	}
	if req.TenantID == "" {
		safeHTTPError(w, r, fmt.Errorf("tenant_id required"), http.StatusBadRequest)
		return
	}
	if req.ProviderType == "" {
		safeHTTPError(w, r, fmt.Errorf("provider_type required"), http.StatusBadRequest)
		return
	}

	cfg := &sso.Config{
		ProviderType: sso.ProviderType(strings.ToLower(req.ProviderType)),
		ClientID:     req.ClientID,
		ClientSecret: req.ClientSecret,
		IssuerURL:    req.IssuerURL,
		CallbackURL:  req.CallbackURL,
		TenantID:     req.TenantID,
		Certificate:  req.Certificate,
	}
	if err := s.ssoManager.RegisterProvider(req.TenantID, cfg); err != nil {
		safeHTTPError(w, r, err, http.StatusBadRequest)
		return
	}
	if s.ssoStore != nil {
		if err := s.ssoStore.Save(r.Context(), req.TenantID, cfg); err != nil {
			_ = s.ssoManager.UnregisterProvider(req.TenantID)
			safeHTTPError(w, r, fmt.Errorf("persist SSO provider: %w", err), http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":       true,
		"tenant_id":     req.TenantID,
		"provider_type": cfg.ProviderType,
	})
}

func (s *APIServer) listSSOProvidersHandler(w http.ResponseWriter, r *http.Request) {
	providers := s.ssoManager.ListProviders()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"providers": providers,
		"count":     len(providers),
	})
}

func (s *APIServer) deleteSSOProviderHandler(w http.ResponseWriter, r *http.Request) {
	tenantID := mux.Vars(r)["tenantID"]
	if tenantID == "" {
		safeHTTPError(w, r, fmt.Errorf("tenantID required"), http.StatusBadRequest)
		return
	}
	if err := s.ssoManager.UnregisterProvider(tenantID); err != nil {
		safeHTTPError(w, r, err, http.StatusNotFound)
		return
	}
	if s.ssoStore != nil {
		if err := s.ssoStore.Delete(r.Context(), tenantID); err != nil {
			safeHTTPError(w, r, fmt.Errorf("delete persisted SSO provider: %w", err), http.StatusInternalServerError)
			return
		}
	}
	json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "tenant_id": tenantID})
}

func (s *APIServer) ssoLoginHandler(w http.ResponseWriter, r *http.Request) {
	tenantID := mux.Vars(r)["tenantID"]
	provider, err := s.ssoManager.GetProvider(tenantID)
	if err != nil {
		safeHTTPError(w, r, err, http.StatusNotFound)
		return
	}
	cfg, err := s.ssoManager.GetConfig(tenantID)
	if err != nil {
		safeHTTPError(w, r, err, http.StatusNotFound)
		return
	}

	redirectURL := cfg.CallbackURL
	if redirectURL == "" {
		redirectURL = publicURL(r) + "/auth/sso/" + url.PathEscape(tenantID) + "/callback"
	}

	if samlProvider, ok := provider.(interface {
		InitiateLogin(callbackURL string) (string, string, error)
	}); ok {
		loginURL, requestID, err := samlProvider.InitiateLogin(redirectURL)
		if err != nil {
			safeHTTPError(w, r, err, http.StatusBadRequest)
			return
		}
		if requestID != "" {
			http.SetCookie(w, &http.Cookie{
				Name:     "hystersis_sso_request",
				Value:    requestID,
				Path:     "/",
				HttpOnly: true,
				Secure:   r.TLS != nil,
				SameSite: http.SameSiteLaxMode,
			})
		}
		http.Redirect(w, r, loginURL, http.StatusTemporaryRedirect)
		return
	}

	authURL := buildSSOAuthURL(cfg, redirectURL, r.URL.Query().Get("state"))
	http.Redirect(w, r, authURL, http.StatusTemporaryRedirect)
}

func (s *APIServer) ssoCallbackHandler(w http.ResponseWriter, r *http.Request) {
	tenantID := mux.Vars(r)["tenantID"]
	provider, err := s.ssoManager.GetProvider(tenantID)
	if err != nil {
		safeHTTPError(w, r, err, http.StatusNotFound)
		return
	}

	code := r.URL.Query().Get("code")
	if code == "" {
		code = r.FormValue("SAMLResponse")
	}
	if code == "" {
		safeHTTPError(w, r, fmt.Errorf("missing SSO authorization response"), http.StatusBadRequest)
		return
	}

	user, err := provider.Authenticate(r.Context(), code)
	if err != nil {
		safeHTTPError(w, r, err, http.StatusUnauthorized)
		return
	}
	s.writeSSOSession(w, user)
}

func (s *APIServer) ssoLDAPLoginHandler(w http.ResponseWriter, r *http.Request) {
	tenantID := mux.Vars(r)["tenantID"]
	provider, err := s.ssoManager.GetProvider(tenantID)
	if err != nil {
		safeHTTPError(w, r, err, http.StatusNotFound)
		return
	}
	if provider.Type() != sso.ProviderTypeLDAP {
		safeHTTPError(w, r, fmt.Errorf("tenant is not configured for LDAP"), http.StatusBadRequest)
		return
	}

	var req struct {
		Username string `json:"username"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		safeHTTPError(w, r, fmt.Errorf("invalid request body: %w", err), http.StatusBadRequest)
		return
	}
	username := req.Username
	if username == "" {
		username = req.Email
	}
	if username == "" || req.Password == "" {
		safeHTTPError(w, r, fmt.Errorf("username/email and password required"), http.StatusBadRequest)
		return
	}

	user, err := provider.Authenticate(r.Context(), username+":"+req.Password)
	if err != nil {
		safeHTTPError(w, r, err, http.StatusUnauthorized)
		return
	}
	s.writeSSOSession(w, user)
}

func (s *APIServer) writeSSOSession(w http.ResponseWriter, user *sso.User) {
	role := "member"
	if len(user.Roles) > 0 && user.Roles[0] != "" {
		role = user.Roles[0]
	}
	session := s.sessionStore.CreateSession(user.ID, user.Email, user.Name, role)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"token":   session.Token,
		"user": map[string]interface{}{
			"id":        user.ID,
			"email":     user.Email,
			"name":      user.Name,
			"tenant_id": user.TenantID,
			"roles":     user.Roles,
			"groups":    user.Groups,
		},
	})
}

func buildSSOAuthURL(cfg *sso.Config, callbackURL, state string) string {
	base := strings.TrimRight(cfg.IssuerURL, "/")
	if cfg.ProviderType == sso.ProviderTypeOIDC {
		base += "/authorize"
	} else {
		base += "/oauth/authorize"
	}
	q := url.Values{}
	q.Set("client_id", cfg.ClientID)
	q.Set("redirect_uri", callbackURL)
	q.Set("response_type", "code")
	q.Set("scope", "openid email profile")
	if state != "" {
		q.Set("state", state)
	}
	return base + "?" + q.Encode()
}

func publicURL(r *http.Request) string {
	scheme := "https"
	if r.TLS == nil {
		scheme = "http"
	}
	if proto := r.Header.Get("X-Forwarded-Proto"); proto != "" {
		scheme = proto
	}
	return scheme + "://" + r.Host
}
