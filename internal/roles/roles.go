package roles

type Role string

const (
	RoleAdmin      Role = "admin"
	RoleEditor     Role = "editor"
	RoleViewer     Role = "viewer"
	RoleAgent      Role = "agent"
	RoleUser       Role = "user"
)

type Permission string

const (
	PermReadMemory      Permission = "memory:read"
	PermWriteMemory     Permission = "memory:write"
	PermDeleteMemory    Permission = "memory:delete"
	PermReadEntity      Permission = "entity:read"
	PermWriteEntity     Permission = "entity:write"
	PermDeleteEntity    Permission = "entity:delete"
	PermManageSkills    Permission = "skills:manage"
	PermExecuteSkills   Permission = "skills:execute"
	PermManageAgents    Permission = "agents:manage"
	PermManageUsers     Permission = "users:manage"
	PermManageAPIKeys   Permission = "apikeys:manage"
	PermManageWebhooks  Permission = "webhooks:manage"
	PermViewAnalytics   Permission = "analytics:view"
	PermManageCompress  Permission = "compression:manage"
	PermBenchmark       Permission = "benchmark:run"
	PermAdmin           Permission = "admin:all"
)

var rolePermissions = map[Role][]Permission{
	RoleAdmin:  {PermAdmin},
	RoleEditor: {PermReadMemory, PermWriteMemory, PermDeleteMemory, PermReadEntity, PermWriteEntity, PermDeleteEntity, PermManageSkills, PermExecuteSkills, PermManageAgents, PermViewAnalytics},
	RoleViewer: {PermReadMemory, PermReadEntity, PermViewAnalytics},
	RoleAgent:  {PermReadMemory, PermWriteMemory, PermReadEntity, PermWriteEntity, PermExecuteSkills, PermManageAgents},
	RoleUser:   {PermReadMemory, PermWriteMemory, PermReadEntity},
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