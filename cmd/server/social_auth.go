package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"

	"agent-memory/internal/sso"
)

func (s *APIServer) getSocialProvider(provider string, tenantID string) (*sso.SocialProvider, error) {
	var clientID, clientSecret, callbackURL string
	switch provider {
	case "google":
		clientID = s.cfg.SSO.Google.ClientID
		clientSecret = s.cfg.SSO.Google.ClientSecret
		callbackURL = s.cfg.SSO.Google.CallbackURL
	case "github":
		clientID = s.cfg.SSO.GitHub.ClientID
		clientSecret = s.cfg.SSO.GitHub.ClientSecret
		callbackURL = s.cfg.SSO.GitHub.CallbackURL
	default:
		return nil, fmt.Errorf("unsupported provider: %s", provider)
	}
	if clientID == "" {
		return nil, fmt.Errorf("%s OAuth not configured", provider)
	}
	cfg := &sso.Config{
		ProviderType: sso.ProviderTypeOAuth,
		ClientID:     clientID,
		ClientSecret: clientSecret,
		CallbackURL:  callbackURL,
		TenantID:     tenantID,
	}
	if provider == "google" {
		return sso.NewGoogleProvider(cfg)
	}
	return sso.NewGitHubProvider(cfg)
}

func (s *APIServer) socialOAuthHandler(provider string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		prov, err := s.getSocialProvider(provider, getTenantID(r))
		if err != nil {
			safeHTTPError(w, r, err, http.StatusNotImplemented)
			return
		}

		state := fmt.Sprintf("%s-%s", getTenantID(r), r.URL.Query().Get("redirect"))
		url := prov.GetAuthorizationURL(state)
		http.Redirect(w, r, url, http.StatusTemporaryRedirect)
	}
}

func (s *APIServer) socialOAuthCallbackHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	provider := vars["provider"]
	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "missing authorization code", http.StatusBadRequest)
		return
	}

	prov, err := s.getSocialProvider(provider, getTenantID(r))
	if err != nil {
		safeHTTPError(w, r, err, http.StatusBadRequest)
		return
	}

	user, err := prov.Authenticate(r.Context(), code)
	if err != nil {
		safeHTTPError(w, r, err, http.StatusUnauthorized)
		return
	}

	session := s.sessionStore.CreateSession(user.ID, user.Email, user.Name, "member")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"token":   session.Token,
		"user": map[string]string{
			"id":    user.ID,
			"email": user.Email,
			"name":  user.Name,
		},
	})
}

func (s *APIServer) webhookDeliveryLogsHandler(w http.ResponseWriter, r *http.Request) {
	logs := s.whSvc.GetDeliveryLogs(100)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"logs":  logs,
		"count": len(logs),
	})
}

func (s *APIServer) webhookDeadLetterHandler(w http.ResponseWriter, r *http.Request) {
	dlq := s.whSvc.GetDeadLetterQueue()
	json.NewEncoder(w).Encode(map[string]interface{}{
		"dead_letter": dlq,
		"count":       len(dlq),
	})
}
