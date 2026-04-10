package license

import (
	"context"
	"testing"
	"time"

	"agent-memory/internal/memory/types"
)

func TestValidator_RegisterLicense(t *testing.T) {
	v := NewValidator(nil)

	license := &types.License{
		Tier:     types.LicenseTierDeveloper,
		TenantID: "tenant-1",
		Key:      "dev_12345678901234567890",
	}

	if err := v.RegisterLicense(license); err != nil {
		t.Fatalf("RegisterLicense failed: %v", err)
	}
}

func TestValidator_RegisterLicense_Nil(t *testing.T) {
	v := NewValidator(nil)

	if err := v.RegisterLicense(nil); err == nil {
		t.Error("expected error for nil license")
	}
}

func TestValidator_RegisterLicense_EmptyKey(t *testing.T) {
	v := NewValidator(nil)

	license := &types.License{
		Tier:     types.LicenseTierDeveloper,
		TenantID: "tenant-1",
		Key:      "",
	}

	if err := v.RegisterLicense(license); err == nil {
		t.Error("expected error for empty key on non-AGPL tier")
	}
}

func TestValidator_RegisterLicense_EmptyTenantID(t *testing.T) {
	v := NewValidator(nil)

	license := &types.License{
		Tier: types.LicenseTierDeveloper,
		Key:  "dev_12345678901234567890",
	}

	if err := v.RegisterLicense(license); err == nil {
		t.Error("expected error for empty tenant ID")
	}
}

func TestValidator_Validate(t *testing.T) {
	v := NewValidator(nil)

	result, err := v.Validate(context.Background(), "unknown-tenant")
	if err != nil {
		t.Fatalf("Validate failed: %v", err)
	}

	if !result.Valid {
		t.Error("expected valid result for unknown tenant (defaults to AGPL)")
	}
	if result.Tier != types.LicenseTierOpenSource {
		t.Errorf("expected AGPL tier, got %s", result.Tier)
	}
}

func TestValidator_Validate_Expired(t *testing.T) {
	v := NewValidator(nil)

	expiredTime := time.Now().Add(-24 * time.Hour)
	license := &types.License{
		Tier:      types.LicenseTierDeveloper,
		TenantID:  "tenant-expired",
		Key:       "dev_12345678901234567890",
		ExpiresAt: &expiredTime,
	}

	v.RegisterLicense(license)

	result, err := v.Validate(context.Background(), "tenant-expired")
	if err != nil {
		t.Fatalf("Validate failed: %v", err)
	}

	if result.Valid {
		t.Error("expected invalid result for expired license")
	}
}

func TestValidator_CheckFeature(t *testing.T) {
	v := NewValidator(nil)

	check, err := v.CheckFeature("tenant-1", types.FeatureProceduralMemory)
	if err != nil {
		t.Fatalf("CheckFeature failed: %v", err)
	}

	if !check.Allowed {
		t.Error("procedural memory should be allowed in all tiers")
	}
}

func TestValidator_CheckFeature_MultiAgent(t *testing.T) {
	v := NewValidator(nil)

	check, err := v.CheckFeature("tenant-1", types.FeatureMultiAgent)
	if err != nil {
		t.Fatalf("CheckFeature failed: %v", err)
	}

	if !check.Allowed {
		t.Error("multi-agent should be allowed")
	}
}

func TestValidator_CheckFeature_HumanReview(t *testing.T) {
	v := NewValidator(nil)

	license := &types.License{
		Tier:     types.LicenseTierTeam,
		TenantID: "tenant-team",
		Key:      "team_12345678901234567890",
	}
	v.RegisterLicense(license)

	check, err := v.CheckFeature("tenant-team", types.FeatureHumanReview)
	if err != nil {
		t.Fatalf("CheckFeature failed: %v", err)
	}

	if !check.Allowed {
		t.Error("human review should be allowed for Team tier")
	}
}

func TestValidator_CheckFeature_AuditLogging(t *testing.T) {
	v := NewValidator(nil)

	check, err := v.CheckFeature("tenant-1", types.FeatureAuditLogging)
	if err != nil {
		t.Fatalf("CheckFeature failed: %v", err)
	}

	if check.Allowed {
		t.Error("audit logging should not be allowed for AGPL tier")
	}
}

func TestValidator_CheckFeature_IndustryModules(t *testing.T) {
	v := NewValidator(nil)

	check, err := v.CheckFeature("tenant-1", types.FeatureIndustryModules)
	if err != nil {
		t.Fatalf("CheckFeature failed: %v", err)
	}

	if check.Allowed {
		t.Error("industry modules should not be allowed for AGPL tier")
	}
}

func TestValidator_CheckAgentLimit(t *testing.T) {
	v := NewValidator(nil)

	check, err := v.CheckAgentLimit("tenant-1", 0)
	if err != nil {
		t.Fatalf("CheckAgentLimit failed: %v", err)
	}

	if !check.Allowed {
		t.Error("should allow agent when under limit")
	}
}

func TestValidator_CheckGroupLimit(t *testing.T) {
	v := NewValidator(nil)

	check, err := v.CheckGroupLimit("tenant-1", 0)
	if err != nil {
		t.Fatalf("CheckGroupLimit failed: %v", err)
	}

	if !check.Allowed {
		t.Error("should allow group when under limit")
	}
}

func TestValidator_CheckSkillLimit(t *testing.T) {
	v := NewValidator(nil)

	check, err := v.CheckSkillLimit("tenant-1", 0)
	if err != nil {
		t.Fatalf("CheckSkillLimit failed: %v", err)
	}

	if !check.Allowed {
		t.Error("should allow skill when under limit")
	}
}

func TestValidator_GetEntitlement(t *testing.T) {
	v := NewValidator(nil)

	ent, err := v.GetEntitlement("tenant-1")
	if err != nil {
		t.Fatalf("GetEntitlement failed: %v", err)
	}

	if ent.MaxAgents <= 0 {
		t.Error("expected positive max agents for AGPL")
	}
}

func TestValidator_RevokeLicense(t *testing.T) {
	v := NewValidator(nil)

	license := &types.License{
		Tier:     types.LicenseTierDeveloper,
		TenantID: "tenant-revoke",
		Key:      "dev_12345678901234567890",
	}
	v.RegisterLicense(license)

	if err := v.RevokeLicense("tenant-revoke"); err != nil {
		t.Fatalf("RevokeLicense failed: %v", err)
	}

	if err := v.RevokeLicense("non-existent"); err == nil {
		t.Error("expected error revoking non-existent license")
	}
}

func TestValidator_GenerateLicenseKey(t *testing.T) {
	v := NewValidator(nil)

	key, err := v.GenerateLicenseKey(types.LicenseTierDeveloper, "tenant-1")
	if err != nil {
		t.Fatalf("GenerateLicenseKey failed: %v", err)
	}

	if key == "" {
		t.Error("expected non-empty key")
	}

	if len(key) < 16 {
		t.Error("key should be at least 16 characters")
	}
}

func TestValidator_GenerateLicenseKey_AGPL(t *testing.T) {
	v := NewValidator(nil)

	_, err := v.GenerateLicenseKey(types.LicenseTierOpenSource, "tenant-1")
	if err == nil {
		t.Error("expected error generating key for AGPL tier")
	}
}

func TestValidator_ListTenants(t *testing.T) {
	v := NewValidator(nil)

	v.RegisterLicense(&types.License{
		Tier:     types.LicenseTierDeveloper,
		TenantID: "tenant-1",
		Key:      "dev_12345678901234567890",
	})

	v.RegisterLicense(&types.License{
		Tier:     types.LicenseTierTeam,
		TenantID: "tenant-2",
		Key:      "team_12345678901234567890",
	})

	tenants := v.ListTenants()
	if len(tenants) != 2 {
		t.Errorf("expected 2 tenants, got %d", len(tenants))
	}
}

func TestValidator_GetLicense(t *testing.T) {
	v := NewValidator(nil)

	v.RegisterLicense(&types.License{
		Tier:     types.LicenseTierDeveloper,
		TenantID: "tenant-1",
		Key:      "dev_12345678901234567890",
	})

	license, err := v.GetLicense("tenant-1")
	if err != nil {
		t.Fatalf("GetLicense failed: %v", err)
	}

	if license.Tier != types.LicenseTierDeveloper {
		t.Errorf("expected developer tier, got %s", license.Tier)
	}

	_, err = v.GetLicense("non-existent")
	if err == nil {
		t.Error("expected error for non-existent tenant")
	}
}

func TestTierChecker_RequireTier(t *testing.T) {
	v := NewValidator(nil)
	checker := NewTierChecker(v)

	v.RegisterLicense(&types.License{
		Tier:     types.LicenseTierTeam,
		TenantID: "tenant-team",
		Key:      "team_12345678901234567890",
	})

	if err := checker.RequireTier(context.Background(), "tenant-team", types.LicenseTierDeveloper); err != nil {
		t.Error("Team tier should satisfy Developer requirement")
	}

	if err := checker.RequireTier(context.Background(), "tenant-team", types.LicenseTierEnterprise); err == nil {
		t.Error("Team tier should not satisfy Enterprise requirement")
	}
}

func TestTierChecker_RequireFeature(t *testing.T) {
	v := NewValidator(nil)
	checker := NewTierChecker(v)

	if err := checker.RequireFeature(context.Background(), "tenant-1", types.FeatureProceduralMemory); err != nil {
		t.Error("procedural memory should be available")
	}
}

func TestMiddleware_SkipPath(t *testing.T) {
	v := NewValidator(nil)
	m := NewMiddleware(v)

	if err := m.RequireValidLicense(context.Background(), "/health", "tenant-1"); err != nil {
		t.Error("health path should be skipped")
	}

	if err := m.RequireValidLicense(context.Background(), "/admin/license", "tenant-1"); err != nil {
		t.Error("admin/license path should be skipped")
	}
}

func TestMiddleware_RequireFeature(t *testing.T) {
	v := NewValidator(nil)
	m := NewMiddleware(v)

	v.RegisterLicense(&types.License{
		Tier:     types.LicenseTierTeam,
		TenantID: "tenant-team",
		Key:      "team_12345678901234567890",
	})

	if err := m.RequireFeature(context.Background(), "/api", "tenant-team", types.FeatureHumanReview); err != nil {
		t.Error("human review should be allowed for Team tier")
	}
}

func TestValidator_ValidateLicenseKey_ShortKey(t *testing.T) {
	v := NewValidator(nil)

	license := &types.License{
		Tier:     types.LicenseTierDeveloper,
		TenantID: "tenant-1",
		Key:      "short",
	}

	if err := v.validateLicenseKey(license); err == nil {
		t.Error("expected error for short license key")
	}
}

func TestValidator_Validate_AGPL_NoKey(t *testing.T) {
	v := NewValidator(nil)

	license := &types.License{
		Tier:     types.LicenseTierOpenSource,
		TenantID: "tenant-1",
	}

	if err := v.RegisterLicense(license); err != nil {
		t.Errorf("AGPL should not require key: %v", err)
	}
}
