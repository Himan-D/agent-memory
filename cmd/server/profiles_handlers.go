package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"

	"agent-memory/internal/profiles"
)

func (s *APIServer) getProfileHandler(w http.ResponseWriter, r *http.Request) {
	if s.profileSvc == nil {
		jsonError(w, "profiles service not initialized", http.StatusServiceUnavailable)
		return
	}

	userID := mux.Vars(r)["userID"]
	profile, err := s.profileSvc.GetUserProfile(r.Context(), userID)
	if err != nil {
		safeHTTPError(w, r, fmt.Errorf("get profile: %w", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(profile)
}

func (s *APIServer) updateProfileHandler(w http.ResponseWriter, r *http.Request) {
	if s.profileSvc == nil {
		jsonError(w, "profiles service not initialized", http.StatusServiceUnavailable)
		return
	}

	userID := mux.Vars(r)["userID"]
	var profile profiles.UserProfile
	if err := json.NewDecoder(r.Body).Decode(&profile); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}
	profile.ID = userID

	if err := s.profileSvc.UpsertProfile(r.Context(), &profile); err != nil {
		safeHTTPError(w, r, fmt.Errorf("update profile: %w", err), http.StatusInternalServerError)
		return
	}

	updated, err := s.profileSvc.GetUserProfile(r.Context(), userID)
	if err != nil {
		safeHTTPError(w, r, fmt.Errorf("get updated profile: %w", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(updated)
}

func (s *APIServer) getProfilePreferencesHandler(w http.ResponseWriter, r *http.Request) {
	if s.profileSvc == nil {
		jsonError(w, "profiles service not initialized", http.StatusServiceUnavailable)
		return
	}

	userID := mux.Vars(r)["userID"]
	prefs, err := s.profileSvc.GetUserPreferences(r.Context(), userID)
	if err != nil {
		safeHTTPError(w, r, fmt.Errorf("get preferences: %w", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"user_id":     userID,
		"preferences": prefs,
	})
}

func (s *APIServer) updateProfilePreferencesHandler(w http.ResponseWriter, r *http.Request) {
	if s.profileSvc == nil {
		jsonError(w, "profiles service not initialized", http.StatusServiceUnavailable)
		return
	}

	userID := mux.Vars(r)["userID"]
	var req struct {
		Preferences map[string]interface{} `json:"preferences"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.Preferences == nil {
		jsonError(w, "preferences required", http.StatusBadRequest)
		return
	}

	if err := s.profileSvc.UpdateUserPreferences(r.Context(), userID, req.Preferences); err != nil {
		safeHTTPError(w, r, fmt.Errorf("update preferences: %w", err), http.StatusInternalServerError)
		return
	}

	prefs, _ := s.profileSvc.GetUserPreferences(r.Context(), userID)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"user_id":     userID,
		"preferences": prefs,
	})
}

func (s *APIServer) getProfileBehaviorHandler(w http.ResponseWriter, r *http.Request) {
	if s.profileSvc == nil {
		jsonError(w, "profiles service not initialized", http.StatusServiceUnavailable)
		return
	}

	userID := mux.Vars(r)["userID"]
	behavior, err := s.profileSvc.BuildBehaviorProfile(r.Context(), userID)
	if err != nil {
		safeHTTPError(w, r, fmt.Errorf("build behavior profile: %w", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"user_id":  userID,
		"behavior": behavior,
	})
}

func (s *APIServer) getProfileRecommendationsHandler(w http.ResponseWriter, r *http.Request) {
	if s.profileSvc == nil {
		jsonError(w, "profiles service not initialized", http.StatusServiceUnavailable)
		return
	}

	userID := mux.Vars(r)["userID"]
	limit := 5
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
		}
	}

	recs, err := s.profileSvc.GetRecommendations(r.Context(), userID, limit)
	if err != nil {
		safeHTTPError(w, r, fmt.Errorf("get recommendations: %w", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"user_id":         userID,
		"recommendations": recs,
	})
}

func (s *APIServer) getProfileSummaryHandler(w http.ResponseWriter, r *http.Request) {
	if s.profileSvc == nil {
		jsonError(w, "profiles service not initialized", http.StatusServiceUnavailable)
		return
	}

	userID := mux.Vars(r)["userID"]
	summary, err := s.profileSvc.GenerateUserSummary(r.Context(), userID)
	if err != nil {
		safeHTTPError(w, r, fmt.Errorf("generate summary: %w", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"user_id": userID,
		"summary": summary,
	})
}

func (s *APIServer) trackProfileActivityHandler(w http.ResponseWriter, r *http.Request) {
	if s.profileSvc == nil {
		jsonError(w, "profiles service not initialized", http.StatusServiceUnavailable)
		return
	}

	userID := mux.Vars(r)["userID"]
	var req struct {
		Type     string                 `json:"type"`
		Metadata map[string]interface{} `json:"metadata"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.Type == "" {
		jsonError(w, "type is required", http.StatusBadRequest)
		return
	}

	if err := s.profileSvc.TrackUserActivity(r.Context(), userID, req.Type, req.Metadata); err != nil {
		safeHTTPError(w, r, fmt.Errorf("track activity: %w", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "recorded"})
}

func (s *APIServer) v4ProfileHandler(w http.ResponseWriter, r *http.Request) {
	if s.profileSvc == nil {
		jsonError(w, "profiles service not initialized", http.StatusServiceUnavailable)
		return
	}

	var profile profiles.UserProfile
	if err := json.NewDecoder(r.Body).Decode(&profile); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if profile.ID == "" {
		jsonError(w, "id is required", http.StatusBadRequest)
		return
	}

	if err := s.profileSvc.UpsertProfile(r.Context(), &profile); err != nil {
		safeHTTPError(w, r, fmt.Errorf("upsert profile: %w", err), http.StatusInternalServerError)
		return
	}

	updated, err := s.profileSvc.GetUserProfile(r.Context(), profile.ID)
	if err != nil {
		safeHTTPError(w, r, fmt.Errorf("get profile: %w", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(updated)
}
