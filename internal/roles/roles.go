package roles

type Role string

const (
	RoleAdmin  Role = "admin"
	RoleEditor Role = "editor"
	RoleViewer Role = "viewer"
	RoleAgent  Role = "agent"
	RoleUser   Role = "user"
)

type Permission string

const (
	PermReadMemory     Permission = "memory:read"
	PermWriteMemory    Permission = "memory:write"
	PermDeleteMemory   Permission = "memory:delete"
	PermReadEntity     Permission = "entity:read"
	PermWriteEntity    Permission = "entity:write"
	PermDeleteEntity   Permission = "entity:delete"
	PermManageSkills   Permission = "skills:manage"
	PermExecuteSkills  Permission = "skills:execute"
	PermManageAgents   Permission = "agents:manage"
	PermManageUsers    Permission = "users:manage"
	PermManageAPIKeys  Permission = "apikeys:manage"
	PermManageWebhooks Permission = "webhooks:manage"
	PermViewAnalytics  Permission = "analytics:view"
	PermManageCompress Permission = "compression:manage"
	PermBenchmark      Permission = "benchmark:run"
	PermAdmin          Permission = "admin:all"
)

var rolePermissions = map[Role][]Permission{
	RoleAdmin:  {PermAdmin},
	RoleEditor: {PermReadMemory, PermWriteMemory, PermDeleteMemory, PermReadEntity, PermWriteEntity, PermDeleteEntity, PermManageSkills, PermExecuteSkills, PermManageAgents, PermManageAPIKeys, PermManageWebhooks, PermViewAnalytics},
	RoleViewer: {PermReadMemory, PermReadEntity, PermViewAnalytics},
	RoleAgent:  {PermReadMemory, PermWriteMemory, PermReadEntity, PermWriteEntity, PermExecuteSkills, PermManageAgents},
	RoleUser:   {PermReadMemory, PermWriteMemory, PermReadEntity, PermManageAPIKeys, PermManageWebhooks},
}

type Checker struct {
	rolePerms map[Role][]Permission
}

func NewChecker() *Checker {
	return &Checker{
		rolePerms: rolePermissions,
	}
}

func (c *Checker) HasPermission(role Role, perm Permission) bool {
	perms, ok := c.rolePerms[role]
	if !ok {
		return false
	}

	for _, p := range perms {
		if p == PermAdmin {
			return true
		}
		if p == perm {
			return true
		}
	}
	return false
}

func (c *Checker) HasAnyPermission(role Role, perms []Permission) bool {
	for _, p := range perms {
		if c.HasPermission(role, p) {
			return true
		}
	}
	return false
}

func (c *Checker) GetPermissions(role Role) []Permission {
	perms, ok := c.rolePerms[role]
	if !ok {
		return nil
	}
	return append([]Permission{}, perms...)
}

func (c *Checker) ValidateRole(role string) (Role, bool) {
	r := Role(role)
	_, ok := c.rolePerms[r]
	return r, ok
}

// CheckScope verifies that the provided key has at least the required scope.
// Supports legacy scopes (admin, write, read), resource scopes
// (memories:read, webhooks:write), and wildcards (*, *:read, memories:*).
func CheckScope(keyScopes []string, required string) bool {
	if !validRequiredScope(required) {
		return false
	}

	for _, s := range keyScopes {
		if scopeAllows(s, required) {
			return true
		}
	}
	return false
}

func validRequiredScope(scope string) bool {
	if scope == "read" || scope == "write" || scope == "admin" || scope == "*" {
		return true
	}
	resource, action := splitScope(scope)
	return resource != "" && action != "" && legacyScopeLevel(action) > 0
}

func scopeAllows(granted string, required string) bool {
	if granted == "" {
		return false
	}
	if granted == "admin" || granted == "*" {
		return true
	}
	if granted == required {
		return true
	}

	resource, action := splitScope(required)
	grantedResource, grantedAction := splitScope(granted)

	if resource == "" {
		return legacyScopeLevel(granted) >= legacyScopeLevel(required) && legacyScopeLevel(required) > 0
	}

	if grantedResource == "*" && actionAllows(grantedAction, action) {
		return true
	}
	if grantedResource == resource && actionAllows(grantedAction, action) {
		return true
	}

	if grantedResource == "" {
		return legacyScopeLevel(grantedAction) >= legacyScopeLevel(action) && legacyScopeLevel(action) > 0
	}

	return false
}

func splitScope(scope string) (string, string) {
	for i, r := range scope {
		if r == ':' {
			return scope[:i], scope[i+1:]
		}
	}
	return "", scope
}

func actionAllows(granted string, required string) bool {
	if granted == "*" || granted == "admin" {
		return true
	}
	return legacyScopeLevel(granted) >= legacyScopeLevel(required) && legacyScopeLevel(required) > 0
}

func legacyScopeLevel(scope string) int {
	switch scope {
	case "read":
		return 1
	case "write", "delete", "manage":
		return 2
	case "admin":
		return 3
	default:
		return 0
	}
}
