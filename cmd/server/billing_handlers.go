package main

import (
	"encoding/json"
	"fmt"
	"net/http"
)

func (s *APIServer) getBillingPlansHandler(w http.ResponseWriter, r *http.Request) {
	if s.stripeSvc == nil {
		jsonError(w, "billing service not initialized", http.StatusServiceUnavailable)
		return
	}

	plans := s.stripeSvc.GetPlans()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"plans": plans,
	})
}

func (s *APIServer) stripeCheckoutHandler(w http.ResponseWriter, r *http.Request) {
	if s.stripeSvc == nil {
		jsonError(w, "billing service not initialized", http.StatusServiceUnavailable)
		return
	}

	var req struct {
		Plan       string `json:"plan"`
		Seats      int    `json:"seats"`
		SuccessURL string `json:"success_url"`
		CancelURL  string `json:"cancel_url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.Plan == "" {
		jsonError(w, "plan is required", http.StatusBadRequest)
		return
	}
	if req.Seats <= 0 {
		req.Seats = 1
	}
	if req.SuccessURL == "" {
		req.SuccessURL = "https://app.hystersis.com?success=true"
	}
	if req.CancelURL == "" {
		req.CancelURL = "https://hystersis.com?canceled=true"
	}

	url, err := s.stripeSvc.CreateCheckoutSession(r.Context(), req.Plan, req.Seats, req.SuccessURL, req.CancelURL)
	if err != nil {
		safeHTTPError(w, r, fmt.Errorf("create checkout session: %w", err), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"url": url})
}
