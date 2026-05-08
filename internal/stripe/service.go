package stripe

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/stripe/stripe-go/v81"
	"github.com/stripe/stripe-go/v81/checkout/session"
	"github.com/stripe/stripe-go/v81/webhook"
)

type Service struct {
	webhookSecret string
}

func NewService() *Service {
	apiKey := os.Getenv("STRIPE_SECRET_KEY")
	webhookSecret := os.Getenv("STRIPE_WEBHOOK_SECRET")

	if apiKey != "" {
		stripe.Key = apiKey
	}

	return &Service{
		webhookSecret: webhookSecret,
	}
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
	var session stripe.CheckoutSession
	if err := json.Unmarshal(data, &session); err != nil {
		fmt.Printf("Error parsing checkout session: %v\n", err)
		return
	}

	customerID := ""
	if session.Customer != nil {
		customerID = session.Customer.ID
	}
	fmt.Printf("Checkout completed: %s, Customer: %s\n", session.ID, customerID)
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
		fmt.Printf("Error parsing subscription: %v\n", err)
		return
	}

	fmt.Printf("Subscription deleted: %s\n", subscription.ID)
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
	return []Plan{
		{ID: "selfHosted", Name: "Self-Hosted", PricePerSeat: 0},
		{ID: "pro", Name: "Pro", PricePerSeat: 29, PriceID: os.Getenv("STRIPE_PRO_PRICE_ID")},
		{ID: "team", Name: "Team", PricePerSeat: 99, PriceID: os.Getenv("STRIPE_TEAM_PRICE_ID")},
		{ID: "enterprise", Name: "Enterprise", PricePerSeat: 0},
	}
}

func (s *Service) CreateCheckoutSession(ctx context.Context, planID string, seats int, successURL, cancelURL string) (string, error) {
	if !s.IsConfigured() {
		return "", fmt.Errorf("Stripe not configured. Set STRIPE_SECRET_KEY and STRIPE_WEBHOOK_SECRET environment variables")
	}

	priceID := ""
	switch planID {
	case "pro":
		priceID = os.Getenv("STRIPE_PRO_PRICE_ID")
	case "team":
		priceID = os.Getenv("STRIPE_TEAM_PRICE_ID")
	default:
		return "", fmt.Errorf("invalid plan: %s", planID)
	}

	if priceID == "" {
		return "", fmt.Errorf("price ID not configured for plan: %s", planID)
	}

	params := &stripe.CheckoutSessionParams{
		Mode:               stripe.String(string(stripe.CheckoutSessionModeSubscription)),
		PaymentMethodTypes: stripe.StringSlice([]string{"card"}),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				Price:    stripe.String(priceID),
				Quantity: stripe.Int64(int64(seats)),
			},
		},
		SuccessURL: stripe.String(successURL),
		CancelURL:  stripe.String(cancelURL),
	}

	session, err := session.New(params)
	if err != nil {
		return "", fmt.Errorf("create checkout session: %w", err)
	}

	return session.URL, nil
}

func jsonError(w http.ResponseWriter, message string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}