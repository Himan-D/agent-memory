package license

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"agent-memory/internal/memory/types"
)

type Validator struct {
	entitlements map[types.LicenseTier]types.Entitlement
	tiers        map[string]*types.License
	mu           sync.RWMutex
	config       *ValidatorConfig
}

type ValidatorConfig struct {
	CacheDuration time.Duration
	StrictMode    bool
}

type ValidationResult struct {
	Valid       bool
	Tier        types.LicenseTier
	Entitlement types.Entitlement
	Errors      []string
	Warnings    []string
}

type FeatureCheck struct {
	Feature string
	Allowed bool
	Reason  string
}

func NewValidator(cfg *ValidatorConfig) *Validator {
	if cfg == nil {
		cfg = &ValidatorConfig{
			CacheDuration: 5 * time.Minute,
			StrictMode:    false,
		}
	}

	v := &Validator{
		entitlements: make(map[types.LicenseTier]types.Entitlement),
		tiers:        make(map[string]*types.License),
		config:       cfg,
	}

	for tier, ent := range types.DefaultEntitlements {
		v.entitlements[tier] = ent
	}

	return v
}

func (v *Validator) RegisterLicense(license *types.License) error {
	if license == nil {
		return fmt.Errorf("license is nil")
	}

	if license.Key == "" && license.Tier != types.LicenseTierOpenSource {
		return fmt.Errorf("license key is required for non-AGPL tiers")
	}

	if license.TenantID == "" {
		return fmt.Errorf("tenant_id is required")
	}

	if err := v.validateLicenseKey(license); err != nil {
		return err
	}

	v.mu.Lock()
	defer v.mu.Unlock()

	v.tiers[license.TenantID] = license

	return nil
}

func (v *Validator) validateLicenseKey(license *types.License) error {
	if license.Tier == types.LicenseTierOpenSource {
		return nil
	}

	if len(license.Key) < 16 {
		return fmt.Errorf("invalid license key format")
	}

	keyHash := v.hashKey(license.Key)
	expectedPrefix := v.getExpectedPrefix(license.Tier)

	if !strings.HasPrefix(keyHash, expectedPrefix) {
		if v.config.StrictMode {
			return fmt.Errorf("license key does not match expected format for tier %s", license.Tier)
		}
	}

	return nil
}

func (v *Validator) hashKey(key string) string {
	hash := sha256.Sum256([]byte(key + v.getSalt()))
	return hex.EncodeToString(hash[:])
}

func (v *Validator) getSalt() string {
	return "agent-memory-license-salt-v1"
}

func (v *Validator) getExpectedPrefix(tier types.LicenseTier) string {
	switch tier {
	case types.LicenseTierDeveloper:
		return "dev_"
	case types.LicenseTierTeam:
		return "team_"
	case types.LicenseTierEnterprise:
		return "ent_"
	default:
		return ""
	}
}

func (v *Validator) Validate(ctx context.Context, tenantID string) (*ValidationResult, error) {
	v.mu.RLock()
	license, ok := v.tiers[tenantID]
	v.mu.RUnlock()

	if !ok {
		license = &types.License{
			Tier:     types.LicenseTierOpenSource,
			TenantID: tenantID,
		}
	}

	if license.ExpiresAt != nil && license.ExpiresAt.Before(time.Now()) {
		return &ValidationResult{
			Valid:  false,
			Tier:   license.Tier,
			Errors: []string{"license has expired"},
		}, nil
	}

	entitlement, ok := v.entitlements[license.Tier]
	if !ok {
		return &ValidationResult{
			Valid:  false,
			Tier:   license.Tier,
			Errors: []string{"unknown license tier"},
		}, fmt.Errorf("unknown license tier: %s", license.Tier)
	}

	return &ValidationResult{
		Valid:       true,
		Tier:        license.Tier,
		Entitlement: entitlement,
	}, nil
}

func (v *Validator) CheckFeature(tenantID, feature string) (*FeatureCheck, error) {
	result, err := v.Validate(context.Background(), tenantID)
	if err != nil {
		return &FeatureCheck{
			Feature: feature,
			Allowed: false,
			Reason:  err.Error(),
		}, err
	}

	entitlement := result.Entitlement

	allowed := v.isFeatureAllowed(result.Tier, feature, entitlement)

	return &FeatureCheck{
		Feature: feature,
		Allowed: allowed,
		Reason:  v.getFeatureReason(feature, result.Tier, entitlement),
	}, nil
}

func (v *Validator) isFeatureAllowed(tier types.LicenseTier, feature string, ent types.Entitlement) bool {
	switch feature {
	case types.FeatureProceduralMemory:
		return true
	case types.FeatureMultiAgent:
		return tier != types.LicenseTierOpenSource || ent.MaxAgents > 1
	case types.FeatureSharedMemoryPool:
		return tier != types.LicenseTierOpenSource
	case types.FeatureHumanReview:
		return ent.HumanReviewEnabled
	case types.FeatureAuditLogging:
		return ent.AuditLogging
	case types.FeatureIndustryModules:
		return tier == types.LicenseTierEnterprise
	case types.FeatureCustomBranding:
		return ent.CustomDomains
	case types.FeaturePrioritySupport:
		return ent.SupportLevel == "priority" || ent.SupportLevel == "dedicated"
	default:
		return tier == types.LicenseTierEnterprise
	}
}

func (v *Validator) getFeatureReason(feature string, tier types.LicenseTier, ent types.Entitlement) string {
	switch feature {
	case types.FeatureProceduralMemory:
		return "available in all tiers"
	case types.FeatureMultiAgent:
		if tier == types.LicenseTierOpenSource && ent.MaxAgents <= 1 {
			return "requires Developer tier or higher"
		}
		return "available"
	case types.FeatureSharedMemoryPool:
		if tier == types.LicenseTierOpenSource {
			return "requires Team tier or higher"
		}
		return "available"
	case types.FeatureHumanReview:
		if !ent.HumanReviewEnabled {
			return "requires Team tier or higher"
		}
		return "available"
	case types.FeatureAuditLogging:
		if !ent.AuditLogging {
			return "requires Team tier or higher"
		}
		return "available"
	case types.FeatureIndustryModules:
		return "requires Enterprise tier"
	case types.FeatureCustomBranding:
		if !ent.CustomDomains {
			return "requires Enterprise tier"
		}
		return "available"
	case types.FeaturePrioritySupport:
		if ent.SupportLevel != "priority" && ent.SupportLevel != "dedicated" {
			return "requires Priority or Dedicated support"
		}
		return "available"
	default:
		return "requires Enterprise tier"
	}
}

func (v *Validator) CheckAgentLimit(tenantID string, currentAgents int) (*FeatureCheck, error) {
	result, err := v.Validate(context.Background(), tenantID)
	if err != nil {
		return &FeatureCheck{
			Feature: "agent_limit",
			Allowed: false,
			Reason:  err.Error(),
		}, err
	}

	limit := result.Entitlement.MaxAgents
	if limit == -1 {
		return &FeatureCheck{
			Feature: "agent_limit",
			Allowed: true,
			Reason:  "unlimited agents",
		}, nil
	}

	allowed := currentAgents < limit

	return &FeatureCheck{
		Feature: "agent_limit",
		Allowed: allowed,
		Reason:  fmt.Sprintf("limit is %d, current is %d", limit, currentAgents),
	}, nil
}

func (v *Validator) CheckGroupLimit(tenantID string, currentGroups int) (*FeatureCheck, error) {
	result, err := v.Validate(context.Background(), tenantID)
	if err != nil {
		return &FeatureCheck{
			Feature: "group_limit",
			Allowed: false,
			Reason:  err.Error(),
		}, nil
	}

	limit := result.Entitlement.MaxGroups
	if limit == -1 {
		return &FeatureCheck{
			Feature: "group_limit",
			Allowed: true,
			Reason:  "unlimited groups",
		}, nil
	}

	allowed := currentGroups < limit

	return &FeatureCheck{
		Feature: "group_limit",
		Allowed: allowed,
		Reason:  fmt.Sprintf("limit is %d, current is %d", limit, currentGroups),
	}, nil
}

func (v *Validator) CheckSkillLimit(tenantID string, currentSkills int) (*FeatureCheck, error) {
	result, err := v.Validate(context.Background(), tenantID)
	if err != nil {
		return &FeatureCheck{
			Feature: "skill_limit",
			Allowed: false,
			Reason:  err.Error(),
		}, nil
	}

	limit := result.Entitlement.MaxSkills
	if limit == -1 {
		return &FeatureCheck{
			Feature: "skill_limit",
			Allowed: true,
			Reason:  "unlimited skills",
		}, nil
	}

	allowed := currentSkills < limit

	return &FeatureCheck{
		Feature: "skill_limit",
		Allowed: allowed,
		Reason:  fmt.Sprintf("limit is %d, current is %d", limit, currentSkills),
	}, nil
}

func (v *Validator) GetEntitlement(tenantID string) (types.Entitlement, error) {
	result, err := v.Validate(context.Background(), tenantID)
	if err != nil {
		return types.Entitlement{}, err
	}
	return result.Entitlement, nil
}

func (v *Validator) RevokeLicense(tenantID string) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	if _, ok := v.tiers[tenantID]; !ok {
		return fmt.Errorf("license not found for tenant: %s", tenantID)
	}

	delete(v.tiers, tenantID)
	return nil
}

func (v *Validator) GenerateLicenseKey(tier types.LicenseTier, tenantID string) (string, error) {
	if tier == types.LicenseTierOpenSource {
		return "", fmt.Errorf("AGPL tier does not require a license key")
	}

	rawKey := fmt.Sprintf("%s-%s-%s-%s", tier, tenantID, uuid.New().String(), time.Now().Format("20060102"))
	hash := v.hashKey(rawKey)

	var prefix string
	switch tier {
	case types.LicenseTierDeveloper:
		prefix = "dev_"
	case types.LicenseTierTeam:
		prefix = "team_"
	case types.LicenseTierEnterprise:
		prefix = "ent_"
	}

	return prefix + hash[:32], nil
}

func (v *Validator) ListTenants() []string {
	v.mu.RLock()
	defer v.mu.RUnlock()

	tenants := make([]string, 0, len(v.tiers))
	for tenantID := range v.tiers {
		tenants = append(tenants, tenantID)
	}
	return tenants
}

func (v *Validator) GetLicense(tenantID string) (*types.License, error) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	if license, ok := v.tiers[tenantID]; ok {
		return license, nil
	}

	return nil, fmt.Errorf("license not found for tenant: %s", tenantID)
}

type TierChecker struct {
	validator *Validator
}

func NewTierChecker(v *Validator) *TierChecker {
	return &TierChecker{validator: v}
}

func (c *TierChecker) RequireTier(ctx context.Context, tenantID string, requiredTier types.LicenseTier) error {
	result, err := c.validator.Validate(ctx, tenantID)
	if err != nil {
		return err
	}

	if !c.meetsMinimumTier(result.Tier, requiredTier) {
		return fmt.Errorf("tier %s is required, but tenant has tier %s", requiredTier, result.Tier)
	}

	return nil
}

func (c *TierChecker) RequireFeature(ctx context.Context, tenantID, feature string) error {
	check, err := c.validator.CheckFeature(tenantID, feature)
	if err != nil {
		return err
	}

	if !check.Allowed {
		return fmt.Errorf("feature %s is not available: %s", feature, check.Reason)
	}

	return nil
}

func (c *TierChecker) meetsMinimumTier(current, minimum types.LicenseTier) bool {
	tierOrder := map[types.LicenseTier]int{
		types.LicenseTierOpenSource: 0,
		types.LicenseTierDeveloper:  1,
		types.LicenseTierTeam:       2,
		types.LicenseTierEnterprise: 3,
	}

	currentLevel, ok1 := tierOrder[current]
	minimumLevel, ok2 := tierOrder[minimum]

	if !ok1 || !ok2 {
		return current == minimum
	}

	return currentLevel >= minimumLevel
}

type Middleware struct {
	validator    *Validator
	skippedPaths map[string]bool
}

func NewMiddleware(v *Validator) *Middleware {
	return &Middleware{
		validator: v,
		skippedPaths: map[string]bool{
			"/health":        true,
			"/ready":         true,
			"/metrics":       true,
			"/admin/license": true,
		},
	}
}

func (m *Middleware) SkipPath(path string) {
	m.skippedPaths[path] = true
}

func (m *Middleware) RequireValidLicense(ctx context.Context, path, tenantID string) error {
	if m.skippedPaths[path] {
		return nil
	}

	result, err := m.validator.Validate(ctx, tenantID)
	if err != nil {
		return fmt.Errorf("license validation failed: %w", err)
	}

	if !result.Valid {
		return fmt.Errorf("invalid license: %v", result.Errors)
	}

	return nil
}

func (m *Middleware) RequireFeature(ctx context.Context, path, tenantID, feature string) error {
	if m.skippedPaths[path] {
		return nil
	}

	if err := m.RequireValidLicense(ctx, path, tenantID); err != nil {
		return err
	}

	check, err := m.validator.CheckFeature(tenantID, feature)
	if err != nil {
		return err
	}

	if !check.Allowed {
		return fmt.Errorf("feature %s not allowed: %s", feature, check.Reason)
	}

	return nil
}
