package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	stripePkg "agent-memory/internal/stripe"
)

func (s *APIServer) getBillingPlansHandler(w http.ResponseWriter, r *http.Request) {
	if s.stripeSvc == nil {
		jsonError(w, "billing service not initialized", http.StatusServiceUnavailable)
		return
	}

	plans := s.stripeSvc.GetPlans()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"plans":           plans,
		"stripe_enabled":  s.stripeSvc.IsConfigured(),
	})
}

func (s *APIServer) getBillingUsageHandler(w http.ResponseWriter, r *http.Request) {
	if s.stripeSvc == nil {
		jsonError(w, "billing service not initialized", http.StatusServiceUnavailable)
		return
	}

	tenantID := resolveBillingTenantID(r)
	usage := s.stripeSvc.GetUsage(tenantID)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(usage)
}

func (s *APIServer) getBillingSubscriptionHandler(w http.ResponseWriter, r *http.Request) {
	if s.stripeSvc == nil {
		jsonError(w, "billing service not initialized", http.StatusServiceUnavailable)
		return
	}

	tenantID := resolveBillingTenantID(r)
	sub := s.stripeSvc.GetSubscription(tenantID)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sub)
}

func (s *APIServer) stripeCheckoutHandler(w http.ResponseWriter, r *http.Request) {
	if s.stripeSvc == nil {
		jsonError(w, "billing service not initialized", http.StatusServiceUnavailable)
		return
	}

	var req stripePkg.CheckoutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.PlanID == "" {
		jsonError(w, "plan is required", http.StatusBadRequest)
		return
	}

	if req.TenantID == "" {
		req.TenantID = resolveBillingTenantID(r)
	}

	url, err := s.stripeSvc.CreateCheckoutSession(r.Context(), req)
	if err != nil {
		safeHTTPError(w, r, fmt.Errorf("create checkout session: %w", err), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"url": url})
}

func resolveBillingTenantID(r *http.Request) string {
	tenantID := getTenantID(r)
	if tenantID != "" {
		return tenantID
	}
	if tenantID = r.URL.Query().Get("tenant_id"); tenantID != "" {
		return tenantID
	}
	return "default"
}
