package tenant

import (
	"context"
	"testing"
)

func TestCollectionName(t *testing.T) {
	name := CollectionName("agent_memory", "Acme-Corp")
	if name != "agent_memory_acme_corp" {
		t.Fatalf("got %q", name)
	}
	long := CollectionName("agent_memory", "this-is-a-very-long-tenant-id-that-exceeds-limits-aaaaaaaaaaaaaaaaaaaaaaaa")
	if len(long) > maxCollectionNameLen {
		t.Fatalf("collection name too long: %d %q", len(long), long)
	}
}

func TestResolveActingTenant_StrictSpoof(t *testing.T) {
	_, err := ResolveActingTenant("tenant_a", "tenant_b", false, IsolationStrict, "default")
	if err != ErrTenantSpoof {
		t.Fatalf("expected spoof error, got %v", err)
	}
	got, err := ResolveActingTenant("tenant_a", "tenant_a", false, IsolationStrict, "default")
	if err != nil || got != "tenant_a" {
		t.Fatalf("got %q err %v", got, err)
	}
	got, err = ResolveActingTenant("admin", "tenant_b", true, IsolationStrict, "default")
	if err != nil || got != "tenant_b" {
		t.Fatalf("admin override: got %q err %v", got, err)
	}
}

func TestAssertSameTenant(t *testing.T) {
	auth := TenantContext{TenantID: "a"}
	if err := AssertSameTenant(auth, "a"); err != nil {
		t.Fatal(err)
	}
	if err := AssertSameTenant(auth, "b"); err != ErrTenantMismatch {
		t.Fatalf("expected mismatch, got %v", err)
	}
	admin := TenantContext{TenantID: "a", IsAdmin: true}
	if err := AssertSameTenant(admin, "b"); err != nil {
		t.Fatal(err)
	}
}

func TestServiceCreateAndMembership(t *testing.T) {
	ctx := context.Background()
	svc := NewService(NewMemoryStore())
	ten, err := svc.CreateTenant(ctx, "Acme", "acme", "user1")
	if err != nil {
		t.Fatal(err)
	}
	if ten.ID == "" || ten.Slug != "acme" {
		t.Fatalf("bad tenant: %+v", ten)
	}
	members, err := svc.ListMembers(ctx, ten.ID)
	if err != nil || len(members) != 1 || members[0].Role != RoleOwner {
		t.Fatalf("members: %+v err %v", members, err)
	}
	list, err := svc.ListForUser(ctx, "user1")
	if err != nil || len(list) != 1 {
		t.Fatalf("list for user: %+v err %v", list, err)
	}
	tc, err := svc.SwitchTenant(ctx, "user1", ten.ID, false)
	if err != nil || tc.TenantID != ten.ID {
		t.Fatalf("switch: %+v err %v", tc, err)
	}
	if _, err := svc.SwitchTenant(ctx, "user2", ten.ID, false); err != ErrForbidden {
		t.Fatalf("expected forbidden, got %v", err)
	}
}

func TestInviteFlow(t *testing.T) {
	ctx := context.Background()
	svc := NewService(NewMemoryStore())
	ten, err := svc.CreateTenant(ctx, "Beta", "beta", "owner1")
	if err != nil {
		t.Fatal(err)
	}
	inv, err := svc.CreateInvite(ctx, ten.ID, "new@example.com", "owner1", RoleMember)
	if err != nil {
		t.Fatal(err)
	}
	m, err := svc.AcceptInvite(ctx, inv.Token, "user2", "new@example.com")
	if err != nil || m.UserID != "user2" {
		t.Fatalf("accept: %+v err %v", m, err)
	}
}

func TestContextRoundTrip(t *testing.T) {
	ctx := context.Background()
	tc := TenantContext{TenantID: "t1", UserID: "u1", Role: RoleMember}
	ctx = WithContext(ctx, tc)
	got, err := Require(ctx)
	if err != nil || got.TenantID != "t1" {
		t.Fatalf("got %+v err %v", got, err)
	}
	if IDFromContext(ctx) != "t1" {
		t.Fatal("IDFromContext")
	}
}
