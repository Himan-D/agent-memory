package tenant

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Store persists tenants and memberships.
type Store interface {
	CreateTenant(ctx context.Context, t *Tenant) error
	GetTenant(ctx context.Context, id string) (*Tenant, error)
	GetTenantBySlug(ctx context.Context, slug string) (*Tenant, error)
	UpdateTenant(ctx context.Context, t *Tenant) error
	ListTenantsForUser(ctx context.Context, userID string) ([]Tenant, error)
	ListAllTenants(ctx context.Context, limit, offset int) ([]Tenant, int, error)

	AddMember(ctx context.Context, m *Membership) error
	RemoveMember(ctx context.Context, tenantID, userID string) error
	GetMember(ctx context.Context, tenantID, userID string) (*Membership, error)
	ListMembers(ctx context.Context, tenantID string) ([]Membership, error)

	SaveInvite(ctx context.Context, inv *Invite) error
	GetInviteByToken(ctx context.Context, token string) (*Invite, error)
	DeleteInvite(ctx context.Context, id string) error
	ListInvites(ctx context.Context, tenantID string) ([]Invite, error)
}

// Service provides multi-tenant business operations.
type Service struct {
	store Store
	mu    sync.RWMutex // for in-memory rate of creates if needed
}

// NewService creates a tenant service.
func NewService(store Store) *Service {
	return &Service{store: store}
}

// CreateTenant creates a tenant and adds the creator as owner.
func (s *Service) CreateTenant(ctx context.Context, name, slug, createdBy string) (*Tenant, error) {
	if name == "" {
		return nil, fmt.Errorf("tenant: name is required")
	}
	slug = strings.TrimSpace(strings.ToLower(slug))
	if slug == "" {
		slug = sanitizeSlug(name)
	}
	if msg := ValidateSlug(slug); msg != "" {
		return nil, fmt.Errorf("tenant: %s", msg)
	}
	if existing, _ := s.store.GetTenantBySlug(ctx, slug); existing != nil {
		return nil, fmt.Errorf("tenant: slug %q already exists", slug)
	}

	now := time.Now().UTC()
	t := &Tenant{
		ID:        "ten_" + uuid.New().String()[:8] + "_" + sanitizeSlug(slug),
		Slug:      sanitizeSlug(slug),
		Name:      name,
		Status:    StatusActive,
		Plan:      "free",
		CreatedAt: now,
		UpdatedAt: now,
		CreatedBy: createdBy,
	}
	// Prefer stable ID from slug when short enough
	if len(t.Slug) >= 2 && len(t.Slug) <= 32 {
		t.ID = t.Slug
	}

	if err := s.store.CreateTenant(ctx, t); err != nil {
		return nil, err
	}
	if createdBy != "" {
		_ = s.store.AddMember(ctx, &Membership{
			TenantID:  t.ID,
			UserID:    createdBy,
			Role:      RoleOwner,
			CreatedAt: now,
		})
	}
	return t, nil
}

// EnsureDefaultTenant creates the default tenant if missing (bootstrap).
func (s *Service) EnsureDefaultTenant(ctx context.Context, id string) (*Tenant, error) {
	if id == "" {
		id = "default"
	}
	if t, err := s.store.GetTenant(ctx, id); err == nil && t != nil {
		return t, nil
	}
	now := time.Now().UTC()
	t := &Tenant{
		ID:        id,
		Slug:      sanitizeSlug(id),
		Name:      "Default",
		Status:    StatusActive,
		Plan:      "free",
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.store.CreateTenant(ctx, t); err != nil {
		// race: re-get
		if existing, e2 := s.store.GetTenant(ctx, id); e2 == nil && existing != nil {
			return existing, nil
		}
		return nil, err
	}
	return t, nil
}

// Get returns a tenant by ID.
func (s *Service) Get(ctx context.Context, id string) (*Tenant, error) {
	return s.store.GetTenant(ctx, id)
}

// ListForUser returns tenants the user is a member of.
func (s *Service) ListForUser(ctx context.Context, userID string) ([]Tenant, error) {
	return s.store.ListTenantsForUser(ctx, userID)
}

// ListAll returns all tenants (platform admin).
func (s *Service) ListAll(ctx context.Context, limit, offset int) ([]Tenant, int, error) {
	if limit <= 0 {
		limit = 50
	}
	return s.store.ListAllTenants(ctx, limit, offset)
}

// Update updates mutable tenant fields.
func (s *Service) Update(ctx context.Context, id, name, plan string) (*Tenant, error) {
	t, err := s.store.GetTenant(ctx, id)
	if err != nil || t == nil {
		return nil, fmt.Errorf("tenant: not found")
	}
	if name != "" {
		t.Name = name
	}
	if plan != "" {
		t.Plan = plan
	}
	t.UpdatedAt = time.Now().UTC()
	if err := s.store.UpdateTenant(ctx, t); err != nil {
		return nil, err
	}
	return t, nil
}

// Suspend marks a tenant suspended (platform admin).
func (s *Service) Suspend(ctx context.Context, id string) error {
	t, err := s.store.GetTenant(ctx, id)
	if err != nil || t == nil {
		return fmt.Errorf("tenant: not found")
	}
	t.Status = StatusSuspended
	t.UpdatedAt = time.Now().UTC()
	return s.store.UpdateTenant(ctx, t)
}

// AddMember adds or updates a membership.
func (s *Service) AddMember(ctx context.Context, tenantID, userID, email string, role Role) error {
	if role == "" {
		role = RoleMember
	}
	return s.store.AddMember(ctx, &Membership{
		TenantID:  tenantID,
		UserID:    userID,
		Email:     email,
		Role:      role,
		CreatedAt: time.Now().UTC(),
	})
}

// RemoveMember removes a membership.
func (s *Service) RemoveMember(ctx context.Context, tenantID, userID string) error {
	return s.store.RemoveMember(ctx, tenantID, userID)
}

// ListMembers lists members of a tenant.
func (s *Service) ListMembers(ctx context.Context, tenantID string) ([]Membership, error) {
	return s.store.ListMembers(ctx, tenantID)
}

// GetMembership returns the membership or nil.
func (s *Service) GetMembership(ctx context.Context, tenantID, userID string) (*Membership, error) {
	return s.store.GetMember(ctx, tenantID, userID)
}

// IsMember returns true if user belongs to tenant.
func (s *Service) IsMember(ctx context.Context, tenantID, userID string) bool {
	m, err := s.store.GetMember(ctx, tenantID, userID)
	return err == nil && m != nil
}

// CreateInvite creates a membership invite.
func (s *Service) CreateInvite(ctx context.Context, tenantID, email, invitedBy string, role Role) (*Invite, error) {
	if role == "" {
		role = RoleMember
	}
	token, err := randomToken(24)
	if err != nil {
		return nil, err
	}
	inv := &Invite{
		ID:        "inv_" + uuid.New().String()[:12],
		TenantID:  tenantID,
		Email:     strings.ToLower(strings.TrimSpace(email)),
		Role:      role,
		Token:     token,
		InvitedBy: invitedBy,
		ExpiresAt: time.Now().UTC().Add(7 * 24 * time.Hour),
		CreatedAt: time.Now().UTC(),
	}
	if err := s.store.SaveInvite(ctx, inv); err != nil {
		return nil, err
	}
	return inv, nil
}

// AcceptInvite accepts an invite for the given user.
func (s *Service) AcceptInvite(ctx context.Context, token, userID, email string) (*Membership, error) {
	inv, err := s.store.GetInviteByToken(ctx, token)
	if err != nil || inv == nil {
		return nil, fmt.Errorf("tenant: invite not found")
	}
	if time.Now().After(inv.ExpiresAt) {
		return nil, fmt.Errorf("tenant: invite expired")
	}
	if email != "" && inv.Email != "" && !strings.EqualFold(email, inv.Email) {
		return nil, fmt.Errorf("tenant: invite email mismatch")
	}
	m := &Membership{
		TenantID:  inv.TenantID,
		UserID:    userID,
		Email:     email,
		Role:      inv.Role,
		CreatedAt: time.Now().UTC(),
	}
	if err := s.store.AddMember(ctx, m); err != nil {
		return nil, err
	}
	_ = s.store.DeleteInvite(ctx, inv.ID)
	return m, nil
}

// SwitchTenant validates that the user may act as tenantID.
func (s *Service) SwitchTenant(ctx context.Context, userID, tenantID string, isAdmin bool) (TenantContext, error) {
	if isAdmin {
		return TenantContext{TenantID: tenantID, UserID: userID, Role: RoleAdmin, IsAdmin: true, KeyScope: "admin"}, nil
	}
	m, err := s.store.GetMember(ctx, tenantID, userID)
	if err != nil || m == nil {
		return TenantContext{}, ErrForbidden
	}
	t, err := s.store.GetTenant(ctx, tenantID)
	if err != nil || t == nil || t.Status != StatusActive {
		return TenantContext{}, fmt.Errorf("tenant: not available")
	}
	return TenantContext{
		TenantID: tenantID,
		UserID:   userID,
		Role:     m.Role,
		IsAdmin:  false,
		KeyScope: "write",
	}, nil
}

func randomToken(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
