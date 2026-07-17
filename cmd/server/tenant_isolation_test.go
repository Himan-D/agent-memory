package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"agent-memory/internal/config"
	tenantpkg "agent-memory/internal/tenant"
)

func TestResolveActingTenantViaMiddlewareHelpers(t *testing.T) {
	// Non-admin spoof blocked
	_, err := tenantpkg.ResolveActingTenant("tenant_a", "tenant_b", false, tenantpkg.IsolationStrict, "default")
	if err != tenantpkg.ErrTenantSpoof {
		t.Fatalf("expected spoof error, got %v", err)
	}

	// Admin override allowed
	got, err := tenantpkg.ResolveActingTenant("admin", "tenant_b", true, tenantpkg.IsolationStrict, "default")
	if err != nil || got != "tenant_b" {
		t.Fatalf("admin override failed: %q %v", got, err)
	}
}

func TestEffectiveTenantIgnoresHeaderWhenNotInContext(t *testing.T) {
	// Without middleware setting context, effectiveTenantID falls back to default
	// (X-Tenant-ID is no longer trusted at the helper layer).
	req := httptest.NewRequest(http.MethodGet, "/memories", nil)
	req.Header.Set("X-Tenant-ID", "evil-tenant")
	if got := effectiveTenantID(req); got != "default" {
		t.Fatalf("expected default without auth context, got %q", got)
	}

	ctx := context.WithValue(req.Context(), "tenant_id", "bound-tenant")
	req = req.WithContext(ctx)
	if got := effectiveTenantID(req); got != "bound-tenant" {
		t.Fatalf("expected bound tenant, got %q", got)
	}
}

func TestRequestContextWithTenant(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/search", nil)
	ctx := context.WithValue(req.Context(), "tenant_id", "acme")
	req = req.WithContext(ctx)
	out := requestContextWithTenant(req)
	if tenantpkg.IDFromContext(out) != "acme" {
		t.Fatalf("expected acme in context, got %q", tenantpkg.IDFromContext(out))
	}
}

func TestTenantServiceBootstrapDefault(t *testing.T) {
	cfg := &config.Config{}
	cfg.Tenant.DefaultTenantID = "default"
	cfg.Tenant.Isolation = "strict"
	store := tenantpkg.NewMemoryStore()
	svc := tenantpkg.NewService(store)
	tnt, err := svc.EnsureDefaultTenant(context.Background(), "default")
	if err != nil || tnt.ID != "default" {
		t.Fatalf("ensure default: %+v %v", tnt, err)
	}
}

func TestCollectionNamePerTenant(t *testing.T) {
	a := tenantpkg.CollectionName("agent_memory", "tenant_a")
	b := tenantpkg.CollectionName("agent_memory", "tenant_b")
	if a == b {
		t.Fatal("expected distinct collections")
	}
	if a != "agent_memory_tenant_a" {
		t.Fatalf("got %q", a)
	}
}
