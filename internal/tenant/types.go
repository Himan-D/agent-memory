package tenant

import "time"

// Role within a tenant (membership role, not platform admin).
type Role string

const (
	RoleOwner  Role = "owner"
	RoleAdmin  Role = "admin"
	RoleMember Role = "member"
	RoleViewer Role = "viewer"
)

// IsolationMode controls how strictly tenant filters are enforced.
type IsolationMode string

const (
	// IsolationStrict requires a tenant on every request and filters all data access.
	IsolationStrict IsolationMode = "strict"
	// IsolationOff disables hard isolation (legacy single-tenant / migration).
	IsolationOff IsolationMode = "off"
)

// Status of a tenant organization.
type Status string

const (
	StatusActive    Status = "active"
	StatusSuspended Status = "suspended"
	StatusDeleted   Status = "deleted"
)

// Tenant is a first-class multi-tenant organization.
type Tenant struct {
	ID          string            `json:"id"`
	Slug        string            `json:"slug"`
	Name        string            `json:"name"`
	Status      Status            `json:"status"`
	Plan        string            `json:"plan,omitempty"` // free | pro | team | enterprise
	Metadata    map[string]string `json:"metadata,omitempty"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
	CreatedBy   string            `json:"created_by,omitempty"`
	MemberCount int               `json:"member_count,omitempty"`
}

// Membership links a user to a tenant with a role.
type Membership struct {
	TenantID  string    `json:"tenant_id"`
	UserID    string    `json:"user_id"`
	Email     string    `json:"email,omitempty"`
	Role      Role      `json:"role"`
	CreatedAt time.Time `json:"created_at"`
}

// Invite is a pending membership invitation.
type Invite struct {
	ID        string    `json:"id"`
	TenantID  string    `json:"tenant_id"`
	Email     string    `json:"email"`
	Role      Role      `json:"role"`
	Token     string    `json:"token"`
	InvitedBy string    `json:"invited_by,omitempty"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}

// TenantContext is request-scoped identity for isolation.
type TenantContext struct {
	TenantID string
	UserID   string
	Role     Role   // membership role within the tenant
	IsAdmin  bool   // platform admin (can switch tenants)
	KeyScope string // API key scope: read | write | admin
}

// CanWrite returns true if the context may mutate tenant data.
func (tc TenantContext) CanWrite() bool {
	if tc.IsAdmin {
		return true
	}
	switch tc.Role {
	case RoleOwner, RoleAdmin, RoleMember, "":
		// Empty role: API-key path without membership model still allowed if key scope permits.
		return tc.KeyScope == "write" || tc.KeyScope == "admin" || tc.KeyScope == ""
	default:
		return false
	}
}

// CanManage returns true if the context may manage members/keys.
func (tc TenantContext) CanManage() bool {
	if tc.IsAdmin {
		return true
	}
	return tc.Role == RoleOwner || tc.Role == RoleAdmin
}
