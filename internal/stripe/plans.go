package stripe

import (
	"fmt"
	"strings"
)

// NormalizePlanID maps client plan aliases to canonical tier IDs.
func NormalizePlanID(planID string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(planID)) {
	case "free", "starter", "selfhosted", "self-hosted", "self_hosted":
		return "free", nil
	case "pro":
		return "pro", nil
	case "team":
		return "team", nil
	case "enterprise":
		return "enterprise", nil
	default:
		return "", fmt.Errorf("invalid plan: %s", planID)
	}
}

// IsCheckoutPlan returns true if the plan can be purchased via Stripe Checkout.
func IsCheckoutPlan(planID string) bool {
	tier, err := NormalizePlanID(planID)
	if err != nil {
		return false
	}
	return tier == "pro" || tier == "team"
}
