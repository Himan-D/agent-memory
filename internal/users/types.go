package users

import (
	"time"

	"github.com/google/uuid"
)

type Role string

const (
	RoleAdmin  Role = "admin"
	RoleMember Role = "member"
	RoleViewer Role = "viewer"
)

type User struct {
	ID        uuid.UUID  `json:"id"`
	Email     string     `json:"email"`
	Name      string     `json:"name"`
	Role      Role       `json:"role"`
	Status    string     `json:"status"` // active, inactive, pending
	AvatarURL string     `json:"avatar_url,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	LastLogin *time.Time `json:"last_login,omitempty"`
}

type Invite struct {
	ID        uuid.UUID `json:"id"`
	Email     string    `json:"email"`
	Role      Role      `json:"role"`
	Status    string    `json:"status"` // pending, accepted, rejected, expired
	InvitedBy uuid.UUID `json:"invited_by"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}

type CreateUserRequest struct {
	Email string `json:"email"`
	Name  string `json:"name"`
	Role  Role   `json:"role"`
}

type UpdateUserRequest struct {
	Name   string `json:"name,omitempty"`
	Role   Role   `json:"role,omitempty"`
	Status string `json:"status,omitempty"`
}

type CreateInviteRequest struct {
	Email string `json:"email"`
	Role  Role   `json:"role"`
}

type UserListResponse struct {
	Users     []User `json:"users"`
	Total     int    `json:"total"`
	Page      int    `json:"page"`
	PageSize  int    `json:"page_size"`
}

type InviteListResponse struct {
	Invites []Invite `json:"invites"`
	Total   int     `json:"total"`
}

type Permission struct {
	Resource string `json:"resource"` // memories, entities, agents, etc.
	Action   string `json:"action"`   // create, read, update, delete
}

var RolePermissions = map[Role][]Permission{
	RoleAdmin: {
		{Resource: "memories", Action: "create"}, {Resource: "memories", Action: "read"}, {Resource: "memories", Action: "update"}, {Resource: "memories", Action: "delete"},
		{Resource: "entities", Action: "create"}, {Resource: "entities", Action: "read"}, {Resource: "entities", Action: "update"}, {Resource: "entities", Action: "delete"},
		{Resource: "agents", Action: "create"}, {Resource: "agents", Action: "read"}, {Resource: "agents", Action: "update"}, {Resource: "agents", Action: "delete"},
		{Resource: "groups", Action: "create"}, {Resource: "groups", Action: "read"}, {Resource: "groups", Action: "update"}, {Resource: "groups", Action: "delete"},
		{Resource: "projects", Action: "create"}, {Resource: "projects", Action: "read"}, {Resource: "projects", Action: "update"}, {Resource: "projects", Action: "delete"},
		{Resource: "skills", Action: "create"}, {Resource: "skills", Action: "read"}, {Resource: "skills", Action: "update"}, {Resource: "skills", Action: "delete"},
		{Resource: "chains", Action: "create"}, {Resource: "chains", Action: "read"}, {Resource: "chains", Action: "update"}, {Resource: "chains", Action: "delete"},
		{Resource: "webhooks", Action: "create"}, {Resource: "webhooks", Action: "read"}, {Resource: "webhooks", Action: "update"}, {Resource: "webhooks", Action: "delete"},
		{Resource: "api_keys", Action: "create"}, {Resource: "api_keys", Action: "read"}, {Resource: "api_keys", Action: "update"}, {Resource: "api_keys", Action: "delete"},
		{Resource: "analytics", Action: "read"},
		{Resource: "settings", Action: "read"}, {Resource: "settings", Action: "update"},
		{Resource: "users", Action: "create"}, {Resource: "users", Action: "read"}, {Resource: "users", Action: "update"}, {Resource: "users", Action: "delete"},
		{Resource: "audit", Action: "read"},
	},
	RoleMember: {
		{Resource: "memories", Action: "create"}, {Resource: "memories", Action: "read"}, {Resource: "memories", Action: "update"}, {Resource: "memories", Action: "delete"},
		{Resource: "entities", Action: "create"}, {Resource: "entities", Action: "read"}, {Resource: "entities", Action: "update"}, {Resource: "entities", Action: "delete"},
		{Resource: "agents", Action: "create"}, {Resource: "agents", Action: "read"}, {Resource: "agents", Action: "update"}, {Resource: "agents", Action: "delete"},
		{Resource: "groups", Action: "create"}, {Resource: "groups", Action: "read"}, {Resource: "groups", Action: "update"}, {Resource: "groups", Action: "delete"},
		{Resource: "projects", Action: "create"}, {Resource: "projects", Action: "read"}, {Resource: "projects", Action: "update"}, {Resource: "projects", Action: "delete"},
		{Resource: "skills", Action: "create"}, {Resource: "skills", Action: "read"}, {Resource: "skills", Action: "update"}, {Resource: "skills", Action: "delete"},
		{Resource: "chains", Action: "create"}, {Resource: "chains", Action: "read"}, {Resource: "chains", Action: "update"}, {Resource: "chains", Action: "delete"},
		{Resource: "webhooks", Action: "create"}, {Resource: "webhooks", Action: "read"}, {Resource: "webhooks", Action: "update"}, {Resource: "webhooks", Action: "delete"},
		{Resource: "api_keys", Action: "create"}, {Resource: "api_keys", Action: "read"}, {Resource: "api_keys", Action: "delete"},
		{Resource: "analytics", Action: "read"},
		{Resource: "settings", Action: "read"}, {Resource: "settings", Action: "update"},
	},
	RoleViewer: {
		{Resource: "memories", Action: "read"},
		{Resource: "entities", Action: "read"},
		{Resource: "agents", Action: "read"},
		{Resource: "groups", Action: "read"},
		{Resource: "projects", Action: "read"},
		{Resource: "skills", Action: "read"},
		{Resource: "chains", Action: "read"},
		{Resource: "webhooks", Action: "read"},
		{Resource: "analytics", Action: "read"},
		{Resource: "settings", Action: "read"},
	},
}

func HasPermission(role Role, resource, action string) bool {
	perms, ok := RolePermissions[role]
	if !ok {
		return false
	}
	for _, p := range perms {
		if p.Resource == resource && p.Action == action {
			return true
		}
	}
	return false
}

func GetPermissionsForRole(role Role) []Permission {
	return RolePermissions[role]
}