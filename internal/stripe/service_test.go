package stripe

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNormalizePlanID(t *testing.T) {
	tests := []struct {
		input    string
		expected string
		wantErr  bool
	}{
		{"pro", "pro", false},
		{"Pro", "pro", false},
		{"team", "team", false},
		{"starter", "free", false},
		{"self-hosted", "free", false},
		{"enterprise", "enterprise", false},
		{"invalid", "", true},
	}

	for _, tc := range tests {
		got, err := NormalizePlanID(tc.input)
		if tc.wantErr {
			if err == nil {
				t.Fatalf("expected error for %q", tc.input)
			}
			continue
		}
		if err != nil {
			t.Fatalf("unexpected error for %q: %v", tc.input, err)
		}
		if got != tc.expected {
			t.Fatalf("NormalizePlanID(%q) = %q, want %q", tc.input, got, tc.expected)
		}
	}
}

func TestIsCheckoutPlan(t *testing.T) {
	if !IsCheckoutPlan("pro") {
		t.Fatal("pro should be checkoutable")
	}
	if !IsCheckoutPlan("team") {
		t.Fatal("team should be checkoutable")
	}
	if IsCheckoutPlan("free") {
		t.Fatal("free should not be checkoutable")
	}
	if IsCheckoutPlan("enterprise") {
		t.Fatal("enterprise should not be checkoutable")
	}
}

func TestSetTierAndGetSubscription(t *testing.T) {
	tmp := filepath.Join(t.TempDir(), "usage.json")
	os.Setenv("STRIPE_USAGE_PERSIST_PATH", tmp)
	defer os.Unsetenv("STRIPE_USAGE_PERSIST_PATH")

	svc := NewService()
	svc.SetTier("tenant-a", "pro")

	sub := svc.GetSubscription("tenant-a")
	if sub.Tier != "pro" {
		t.Fatalf("tier = %q, want pro", sub.Tier)
	}
	if sub.MaxMemories != TierQuotas["pro"].MaxMemories {
		t.Fatalf("max memories = %d, want %d", sub.MaxMemories, TierQuotas["pro"].MaxMemories)
	}
}

func TestCheckQuota(t *testing.T) {
	tmp := filepath.Join(t.TempDir(), "usage.json")
	os.Setenv("STRIPE_USAGE_PERSIST_PATH", tmp)
	defer os.Unsetenv("STRIPE_USAGE_PERSIST_PATH")

	svc := NewService()
	svc.SetTier("tenant-b", "free")

	for i := 0; i < int(TierQuotas["free"].MaxMemories); i++ {
		svc.RecordUsage("tenant-b", "memory_create")
	}

	err := svc.CheckQuota("tenant-b", "memory_create")
	if err == nil {
		t.Fatal("expected quota exceeded error")
	}
}

func TestGetPlans(t *testing.T) {
	svc := NewService()
	plans := svc.GetPlans()
	if len(plans) != 4 {
		t.Fatalf("expected 4 plans, got %d", len(plans))
	}
	if plans[0].ID != "free" {
		t.Fatalf("first plan id = %q, want free", plans[0].ID)
	}
}
