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

// mockMemory for isolation logic without Neo4j
func TestListMemoriesTenantFilterLogic(t *testing.T) {
	type mem struct{ TenantID string }
	all := []mem{{"a"}, {"b"}, {"a"}, {""}}
	tenantID := "a"
	var scoped []mem
	for _, m := range all {
		if m.TenantID == tenantID || (m.TenantID == "" && tenantID == "default") {
			scoped = append(scoped, m)
		}
	}
	if len(scoped) != 2 {
		t.Fatalf("expected 2 for tenant a, got %d", len(scoped))
	}
}

func TestAuthMiddlewareSpoofHTTP(t *testing.T) {
	store := NewSessionStore()
	cfg := &config.Config{}
	cfg.Auth.APIKeys = []string{"key_a:tenant_a"}
	cfg.Tenant.Isolation = "strict"
	cfg.Tenant.DefaultTenantID = "default"

	mw := store.routerAuthMiddleware(cfg, nil)
	var gotTenant string
	var gotStatus int
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotTenant = getTenantID(r)
		w.WriteHeader(http.StatusOK)
	}))

	// Valid bound tenant
	req := httptest.NewRequest(http.MethodGet, "/memories", nil)
	req.Header.Set("X-API-Key", "key_a")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK || gotTenant != "tenant_a" {
		t.Fatalf("bound tenant: status=%d tenant=%q", rr.Code, gotTenant)
	}

	// Spoof attempt
	req2 := httptest.NewRequest(http.MethodGet, "/memories", nil)
	req2.Header.Set("X-API-Key", "key_a")
	req2.Header.Set("X-Tenant-ID", "tenant_b")
	rr2 := httptest.NewRecorder()
	handler.ServeHTTP(rr2, req2)
	gotStatus = rr2.Code
	if gotStatus != http.StatusForbidden {
		t.Fatalf("expected 403 spoof, got %d", gotStatus)
	}
}

func TestTenantServiceMemoryStoreRoundTrip(t *testing.T) {
	ctx := context.Background()
	svc := tenantpkg.NewService(tenantpkg.NewMemoryStore())
	ten, err := svc.CreateTenant(ctx, "Acme Corp", "acme", "u1")
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.AddMember(ctx, ten.ID, "u2", "u2@x.com", tenantpkg.RoleMember); err != nil {
		t.Fatal(err)
	}
	members, err := svc.ListMembers(ctx, ten.ID)
	if err != nil || len(members) != 2 {
		t.Fatalf("members: %v err %v", members, err)
	}
	tc, err := svc.SwitchTenant(ctx, "u2", ten.ID, false)
	if err != nil || tc.Role != tenantpkg.RoleMember {
		t.Fatalf("switch: %+v %v", tc, err)
	}
}
