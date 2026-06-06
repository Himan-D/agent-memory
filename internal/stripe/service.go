package stripe

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/stripe/stripe-go/v81"
	"github.com/stripe/stripe-go/v81/checkout/session"
	"github.com/stripe/stripe-go/v81/webhook"
)

type TierQuota struct {
	MaxMemories int64
	MaxSearches int64
	MaxAgents   int
	MaxSkills   int
}

var TierQuotas = map[string]TierQuota{
	"free":       {MaxMemories: 1000, MaxSearches: 10000, MaxAgents: 2, MaxSkills: 10},
	"pro":        {MaxMemories: 50000, MaxSearches: 100000, MaxAgents: 10, MaxSkills: 100},
	"team":       {MaxMemories: 200000, MaxSearches: 500000, MaxAgents: 50, MaxSkills: 500},
	"enterprise": {MaxMemories: -1, MaxSearches: -1, MaxAgents: -1, MaxSkills: -1}, // unlimited
}

type UsageRecord struct {
	TenantID    string    `json:"tenant_id"`
	Tier        string    `json:"tier"`
	MemoryCount int64     `json:"memory_count"`
	SearchCount int64     `json:"search_count"`
	PeriodStart time.Time `json:"period_start"`
	PeriodEnd   time.Time `json:"period_end,omitempty"`
}

type CheckoutRequest struct {
	TenantID   string `json:"tenant_id"`
	PlanID     string `json:"plan"`
	Seats      int    `json:"seats"`
	SuccessURL string `json:"success_url"`
	CancelURL  string `json:"cancel_url"`
	Email      string `json:"email,omitempty"`
}

type SubscriptionInfo struct {
	TenantID      string    `json:"tenant_id"`
	Tier          string    `json:"tier"`
	Status        string    `json:"status"`
	MemoryCount   int64     `json:"memory_count"`
	SearchCount   int64     `json:"search_count"`
	MaxMemories   int64     `json:"max_memories"`
	MaxSearches   int64     `json:"max_searches"`
	MaxAgents     int       `json:"max_agents"`
	MaxSkills     int       `json:"max_skills"`
	PeriodStart   time.Time `json:"period_start"`
	StripeEnabled bool      `json:"stripe_enabled"`
}

type Service struct {
	webhookSecret    string
	usageMap         map[string]*UsageRecord
	usageMu          sync.RWMutex
	usagePersistPath string
}

func NewService() *Service {
	apiKey := os.Getenv("STRIPE_SECRET_KEY")
	webhookSecret := os.Getenv("STRIPE_WEBHOOK_SECRET")

	if apiKey != "" {
		stripe.Key = apiKey
	}

	persistPath := os.Getenv("STRIPE_USAGE_PERSIST_PATH")
	if persistPath == "" {
		persistPath = "/tmp/stripe_usage.json"
	}

	svc := &Service{
		webhookSecret:    webhookSecret,
		usageMap:         make(map[string]*UsageRecord),
		usagePersistPath: persistPath,
	}
	svc.loadUsage()
	return svc
}

// persistUsage writes the current usage map to disk (best-effort).
func (s *Service) persistUsage() {
	s.usageMu.RLock()
	defer s.usageMu.RUnlock()
	data, err := json.Marshal(s.usageMap)
	if err != nil {
		return
	}
	_ = os.WriteFile(s.usagePersistPath, data, 0644)
}

// loadUsage reads a previously persisted usage map from disk.
func (s *Service) loadUsage() {
	data, err := os.ReadFile(s.usagePersistPath)
	if err != nil {
		return // file doesn't exist yet — that's fine
	}
	var m map[string]*UsageRecord
	if err := json.Unmarshal(data, &m); err != nil {
		return
	}
	for k, v := range m {
		s.usageMap[k] = v
	}
}

func (s *Service) CheckQuota(tenantID, operation string) error {
	s.usageMu.RLock()
	usage, exists := s.usageMap[tenantID]
	s.usageMu.RUnlock()

	if !exists {
		// Default to free tier
		usage = &UsageRecord{TenantID: tenantID, Tier: "free", PeriodStart: time.Now()}
	}

	quota, ok := TierQuotas[usage.Tier]
	if !ok {
		quota = TierQuotas["free"]
	}

	switch operation {
	case "memory_create":
		if quota.MaxMemories >= 0 && usage.MemoryCount >= quota.MaxMemories {
			return fmt.Errorf("quota exceeded: %d/%d memories for %s tier", usage.MemoryCount, quota.MaxMemories, usage.Tier)
		}
	case "search":
		if quota.MaxSearches >= 0 && usage.SearchCount >= quota.MaxSearches {
			return fmt.Errorf("quota exceeded: %d/%d searches for %s tier", usage.SearchCount, quota.MaxSearches, usage.Tier)
		}
	}
	return nil
}

func (s *Service) RecordUsage(tenantID, operation string) {
	s.usageMu.Lock()
	defer s.usageMu.Unlock()
	usage, exists := s.usageMap[tenantID]
	if !exists {
		usage = &UsageRecord{TenantID: tenantID, Tier: "free", PeriodStart: time.Now()}
		s.usageMap[tenantID] = usage
	}
	switch operation {
	case "memory_create":
		usage.MemoryCount++
	case "search":
		usage.SearchCount++
	}
	go s.persistUsage()
}

func (s *Service) GetUsage(tenantID string) *UsageRecord {
	s.usageMu.RLock()
	defer s.usageMu.RUnlock()
	if usage, ok := s.usageMap[tenantID]; ok {
		copy := *usage
		return &copy
	}
	return &UsageRecord{TenantID: tenantID, Tier: "free", PeriodStart: time.Now()}
}

func (s *Service) GetSubscription(tenantID string) *SubscriptionInfo {
	usage := s.GetUsage(tenantID)
	quota, ok := TierQuotas[usage.Tier]
	if !ok {
		quota = TierQuotas["free"]
	}
	return &SubscriptionInfo{
		TenantID:      tenantID,
		Tier:          usage.Tier,
		Status:        "active",
		MemoryCount:   usage.MemoryCount,
		SearchCount:   usage.SearchCount,
		MaxMemories:   quota.MaxMemories,
		MaxSearches:   quota.MaxSearches,
		MaxAgents:     quota.MaxAgents,
		MaxSkills:     quota.MaxSkills,
		PeriodStart:   usage.PeriodStart,
		StripeEnabled: s.IsConfigured(),
	}
}

func (s *Service) SetTier(tenantID, tier string) {
	s.usageMu.Lock()
	defer s.usageMu.Unlock()
	if usage, ok := s.usageMap[tenantID]; ok {
		usage.Tier = tier
	} else {
		s.usageMap[tenantID] = &UsageRecord{TenantID: tenantID, Tier: tier, PeriodStart: time.Now()}
	}
	go s.persistUsage()
}

func (s *Service) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	if s.webhookSecret == "" {
		jsonError(w, "Stripe not configured", http.StatusServiceUnavailable)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		jsonError(w, "Failed to read request body", http.StatusBadRequest)
		return
	}

	signature := r.Header.Get("Stripe-Signature")
	event, err := webhook.ConstructEvent(body, signature, s.webhookSecret)
	if err != nil {
		jsonError(w, fmt.Sprintf("Webhook signature verification failed: %v", err), http.StatusBadRequest)
		return
	}

	switch event.Type {
	case "checkout.session.completed":
		s.handleCheckoutComplete(event)
	case "invoice.payment_succeeded":
		s.handlePaymentSuccess(event)
	case "invoice.payment_failed":
		s.handlePaymentFailed(event)
	case "customer.subscription.deleted":
		s.handleSubscriptionDeleted(event)
	default:
		fmt.Printf("Unhandled Stripe event type: %s\n", event.Type)
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (s *Service) handleCheckoutComplete(event stripe.Event) {
	data, _ := json.Marshal(event.Data.Object)
	var sess stripe.CheckoutSession
	if err := json.Unmarshal(data, &sess); err != nil {
		fmt.Printf("stripe checkout parse error: %v\n", err)
		return
	}

	tenantID := sess.Metadata["tenant_id"]
	if tenantID == "" {
		tenantID = sess.ClientReferenceID
	}
	if tenantID == "" && sess.Customer != nil {
		tenantID = sess.Customer.ID
	}
	if tenantID == "" {
		fmt.Printf("stripe checkout completed without tenant_id: session=%s\n", sess.ID)
		return
	}

	tier := "pro"
	if t, ok := sess.Metadata["tier"]; ok && t != "" {
		tier = t
	}
	if normalized, err := NormalizePlanID(tier); err == nil {
		tier = normalized
	}

	s.SetTier(tenantID, tier)
	fmt.Printf("stripe checkout completed: session=%s tenant=%s tier=%s\n", sess.ID, tenantID, tier)
}

func (s *Service) handlePaymentSuccess(event stripe.Event) {
	data, _ := json.Marshal(event.Data.Object)
	var invoice stripe.Invoice
	if err := json.Unmarshal(data, &invoice); err != nil {
		fmt.Printf("Error parsing invoice: %v\n", err)
		return
	}

	fmt.Printf("Payment succeeded: %s\n", invoice.ID)
}

func (s *Service) handlePaymentFailed(event stripe.Event) {
	data, _ := json.Marshal(event.Data.Object)
	var invoice stripe.Invoice
	if err := json.Unmarshal(data, &invoice); err != nil {
		fmt.Printf("Error parsing invoice: %v\n", err)
		return
	}

	fmt.Printf("Payment failed: %s\n", invoice.ID)
}

func (s *Service) handleSubscriptionDeleted(event stripe.Event) {
	data, _ := json.Marshal(event.Data.Object)
	var subscription stripe.Subscription
	if err := json.Unmarshal(data, &subscription); err != nil {
		fmt.Printf("stripe subscription parse error: %v\n", err)
		return
	}

	tenantID := ""
	if subscription.Metadata != nil {
		tenantID = subscription.Metadata["tenant_id"]
	}
	if tenantID == "" && subscription.Customer != nil {
		tenantID = subscription.Customer.ID
	}
	if tenantID != "" {
		s.SetTier(tenantID, "free")
	}
	fmt.Printf("stripe subscription deleted: id=%s tenant=%s\n", subscription.ID, tenantID)
}

func (s *Service) IsConfigured() bool {
	return stripe.Key != "" && s.webhookSecret != ""
}

type Plan struct {
	ID           string  `json:"id"`
	Name         string  `json:"name"`
	PricePerSeat float64 `json:"price_per_seat"`
	PriceID      string  `json:"price_id,omitempty"`
}

func (s *Service) GetPlans() []Plan {
	proPrice := os.Getenv("STRIPE_PRO_PRICE_ID")
	teamPrice := os.Getenv("STRIPE_TEAM_PRICE_ID")
	return []Plan{
		{ID: "free", Name: "Self-Hosted", PricePerSeat: 0},
		{ID: "pro", Name: "Pro", PricePerSeat: 29, PriceID: proPrice},
		{ID: "team", Name: "Team", PricePerSeat: 99, PriceID: teamPrice},
		{ID: "enterprise", Name: "Enterprise", PricePerSeat: 0},
	}
}

func (s *Service) CreateCheckoutSession(ctx context.Context, req CheckoutRequest) (string, error) {
	if !s.IsConfigured() {
		return "", fmt.Errorf("stripe not configured: set STRIPE_SECRET_KEY and STRIPE_WEBHOOK_SECRET")
	}

	tier, err := NormalizePlanID(req.PlanID)
	if err != nil {
		return "", err
	}
	if !IsCheckoutPlan(tier) {
		return "", fmt.Errorf("plan %s is not available for self-serve checkout", req.PlanID)
	}

	seats := req.Seats
	if seats <= 0 {
		seats = 1
	}

	tenantID := req.TenantID
	if tenantID == "" {
		tenantID = "default"
	}

	priceID := ""
	switch tier {
	case "pro":
		priceID = os.Getenv("STRIPE_PRO_PRICE_ID")
	case "team":
		priceID = os.Getenv("STRIPE_TEAM_PRICE_ID")
	}
	if priceID == "" {
		return "", fmt.Errorf("price ID not configured for plan: %s", tier)
	}

	successURL := req.SuccessURL
	if successURL == "" {
		successURL = "https://app.hystersis.com/billing?success=true"
	}
	cancelURL := req.CancelURL
	if cancelURL == "" {
		cancelURL = "https://hystersis.com/?canceled=true"
	}

	params := &stripe.CheckoutSessionParams{
		Mode:               stripe.String(string(stripe.CheckoutSessionModeSubscription)),
		PaymentMethodTypes: stripe.StringSlice([]string{"card"}),
		ClientReferenceID:  stripe.String(tenantID),
		Metadata: map[string]string{
			"tenant_id": tenantID,
			"tier":      tier,
			"plan":      tier,
		},
		SubscriptionData: &stripe.CheckoutSessionSubscriptionDataParams{
			Metadata: map[string]string{
				"tenant_id": tenantID,
				"tier":      tier,
			},
		},
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				Price:    stripe.String(priceID),
				Quantity: stripe.Int64(int64(seats)),
			},
		},
		SuccessURL: stripe.String(successURL),
		CancelURL:  stripe.String(cancelURL),
	}
	if req.Email != "" {
		params.CustomerEmail = stripe.String(req.Email)
	}

	sess, err := session.New(params)
	if err != nil {
		return "", fmt.Errorf("create checkout session: %w", err)
	}

	return sess.URL, nil
}

func jsonError(w http.ResponseWriter, message string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}
