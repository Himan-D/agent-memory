package roles

import "testing"

func TestNewChecker(t *testing.T) {
	c := NewChecker()
	if c == nil {
		t.Fatal("NewChecker() returned nil")
	}
	if c.rolePerms == nil {
		t.Error("rolePerms map is nil")
	}
}

func TestHasPermission_Admin(t *testing.T) {
	c := NewChecker()

	allPerms := []Permission{
		PermReadMemory, PermWriteMemory, PermDeleteMemory,
		PermReadEntity, PermWriteEntity, PermDeleteEntity,
		PermManageSkills, PermExecuteSkills, PermManageAgents,
		PermManageUsers, PermManageAPIKeys, PermManageWebhooks,
		PermViewAnalytics, PermManageCompress, PermBenchmark, PermAdmin,
	}

	for _, perm := range allPerms {
		if !c.HasPermission(RoleAdmin, perm) {
			t.Errorf("admin should have permission %s", perm)
		}
	}
}

func TestHasPermission_Viewer(t *testing.T) {
	c := NewChecker()

	viewerAllow := map[Permission]bool{
		PermReadMemory:    true,
		PermReadEntity:    true,
		PermViewAnalytics: true,
	}

	tests := []struct {
		perm     Permission
		expected bool
	}{
		{PermReadMemory, true},
		{PermReadEntity, true},
		{PermViewAnalytics, true},
		{PermWriteMemory, false},
		{PermDeleteMemory, false},
		{PermWriteEntity, false},
		{PermDeleteEntity, false},
		{PermManageSkills, false},
		{PermExecuteSkills, false},
		{PermManageAgents, false},
		{PermManageUsers, false},
		{PermManageAPIKeys, false},
		{PermManageWebhooks, false},
		{PermManageCompress, false},
		{PermBenchmark, false},
		{PermAdmin, false},
	}

	for _, tt := range tests {
		result := c.HasPermission(RoleViewer, tt.perm)
		if result != tt.expected {
			t.Errorf("HasPermission(viewer, %s) = %v, want %v", tt.perm, result, tt.expected)
		}
	}

	for perm, expected := range viewerAllow {
		if c.HasPermission(RoleViewer, perm) != expected {
			t.Errorf("viewer HasPermission(%s) mismatch", perm)
		}
	}
}

func TestHasPermission_Editor(t *testing.T) {
	c := NewChecker()

	editorAllowed := []Permission{
		PermReadMemory, PermWriteMemory, PermDeleteMemory,
		PermReadEntity, PermWriteEntity, PermDeleteEntity,
		PermManageSkills, PermExecuteSkills, PermManageAgents,
		PermManageAPIKeys, PermManageWebhooks,
		PermViewAnalytics,
	}

	editorDenied := []Permission{
		PermManageUsers,
		PermManageCompress, PermBenchmark,
	}

	for _, perm := range editorAllowed {
		if !c.HasPermission(RoleEditor, perm) {
			t.Errorf("editor should have permission %s", perm)
		}
	}

	for _, perm := range editorDenied {
		if c.HasPermission(RoleEditor, perm) {
			t.Errorf("editor should not have permission %s", perm)
		}
	}
}

func TestHasPermission_Agent(t *testing.T) {
	c := NewChecker()

	agentAllowed := []Permission{
		PermReadMemory, PermWriteMemory,
		PermReadEntity, PermWriteEntity,
		PermExecuteSkills, PermManageAgents,
	}

	agentDenied := []Permission{
		PermDeleteMemory, PermDeleteEntity, PermManageSkills,
		PermManageUsers, PermViewAnalytics, PermManageCompress,
		PermBenchmark, PermAdmin,
	}

	for _, perm := range agentAllowed {
		if !c.HasPermission(RoleAgent, perm) {
			t.Errorf("agent should have permission %s", perm)
		}
	}

	for _, perm := range agentDenied {
		if c.HasPermission(RoleAgent, perm) {
			t.Errorf("agent should not have permission %s", perm)
		}
	}
}

func TestHasPermission_User(t *testing.T) {
	c := NewChecker()

	userAllowed := []Permission{
		PermReadMemory, PermWriteMemory,
		PermReadEntity, PermManageAPIKeys, PermManageWebhooks,
	}

	userDenied := []Permission{
		PermDeleteMemory, PermDeleteEntity, PermManageSkills,
		PermExecuteSkills, PermManageAgents, PermViewAnalytics,
		PermManageUsers,
		PermManageCompress, PermBenchmark, PermAdmin,
	}

	for _, perm := range userAllowed {
		if !c.HasPermission(RoleUser, perm) {
			t.Errorf("user should have permission %s", perm)
		}
	}

	for _, perm := range userDenied {
		if c.HasPermission(RoleUser, perm) {
			t.Errorf("user should not have permission %s", perm)
		}
	}
}

func TestHasPermission_UnknownRole(t *testing.T) {
	c := NewChecker()

	if c.HasPermission("unknown", PermReadMemory) {
		t.Error("unknown role should have no permissions")
	}
}

func TestHasAnyPermission(t *testing.T) {
	c := NewChecker()

	tests := []struct {
		name     string
		role     Role
		perms    []Permission
		expected bool
	}{
		{
			name:     "viewer has any read perm",
			role:     RoleViewer,
			perms:    []Permission{PermReadMemory, PermWriteMemory},
			expected: true,
		},
		{
			name:     "viewer has no manage perms",
			role:     RoleViewer,
			perms:    []Permission{PermManageUsers, PermManageSkills},
			expected: false,
		},
		{
			name:     "admin has any perm",
			role:     RoleAdmin,
			perms:    []Permission{PermManageUsers},
			expected: true,
		},
		{
			name:     "empty perms list",
			role:     RoleViewer,
			perms:    []Permission{},
			expected: false,
		},
		{
			name:     "unknown role returns false",
			role:     "unknown",
			perms:    []Permission{PermReadMemory},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := c.HasAnyPermission(tt.role, tt.perms)
			if result != tt.expected {
				t.Errorf("HasAnyPermission(%s, %v) = %v, want %v", tt.role, tt.perms, result, tt.expected)
			}
		})
	}
}

func TestGetPermissions(t *testing.T) {
	c := NewChecker()

	tests := []struct {
		name          string
		role          Role
		expectedCount int
	}{
		{"admin", RoleAdmin, 1},
		{"editor", RoleEditor, 12},
		{"viewer", RoleViewer, 3},
		{"agent", RoleAgent, 6},
		{"user", RoleUser, 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			perms := c.GetPermissions(tt.role)
			if len(perms) != tt.expectedCount {
				t.Errorf("GetPermissions(%s) returned %d perms, expected %d", tt.role, len(perms), tt.expectedCount)
			}
		})
	}
}

func TestGetPermissions_UnknownRole(t *testing.T) {
	c := NewChecker()
	perms := c.GetPermissions("unknown")
	if perms != nil {
		t.Errorf("expected nil for unknown role, got %v", perms)
	}
}

func TestGetPermissions_ReturnsCopy(t *testing.T) {
	c := NewChecker()
	perms := c.GetPermissions(RoleViewer)
	perms[0] = PermAdmin
	originalPerms := c.GetPermissions(RoleViewer)
	if originalPerms[0] == PermAdmin {
		t.Error("GetPermissions should return a copy, not reference to internal slice")
	}
}

func TestValidateRole(t *testing.T) {
	c := NewChecker()

	tests := []struct {
		input    string
		valid    bool
		expected Role
	}{
		{"admin", true, RoleAdmin},
		{"editor", true, RoleEditor},
		{"viewer", true, RoleViewer},
		{"agent", true, RoleAgent},
		{"user", true, RoleUser},
		{"superadmin", false, Role("superadmin")},
		{"guest", false, Role("guest")},
		{"", false, Role("")},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			role, ok := c.ValidateRole(tt.input)
			if ok != tt.valid {
				t.Errorf("ValidateRole(%q) ok=%v, want %v", tt.input, ok, tt.valid)
			}
			if role != tt.expected {
				t.Errorf("ValidateRole(%q) role=%v, want %v", tt.input, role, tt.expected)
			}
		})
	}
}

func TestCheckScope(t *testing.T) {
	tests := []struct {
		name      string
		keyScopes []string
		required  string
		expected  bool
	}{
		{"admin can read", []string{"admin"}, "read", true},
		{"admin can write", []string{"admin"}, "write", true},
		{"admin can admin", []string{"admin"}, "admin", true},
		{"write can read", []string{"write"}, "read", true},
		{"write can write", []string{"write"}, "write", true},
		{"write cannot admin", []string{"write"}, "admin", false},
		{"read can read", []string{"read"}, "read", true},
		{"read cannot write", []string{"read"}, "write", false},
		{"read cannot admin", []string{"read"}, "admin", false},
		{"no scopes deny all", nil, "read", false},
		{"unknown scope deny all", []string{"unknown"}, "read", false},
		{"invalid required scope", []string{"admin"}, "invalid", false},
		{"resource read can read resource", []string{"memories:read"}, "memories:read", true},
		{"resource write can read resource", []string{"memories:write"}, "memories:read", true},
		{"resource write can write resource", []string{"memories:write"}, "memories:write", true},
		{"resource read cannot write resource", []string{"memories:read"}, "memories:write", false},
		{"resource write cannot write another resource", []string{"memories:write"}, "webhooks:write", false},
		{"wildcard resource read can read any resource", []string{"*:read"}, "webhooks:read", true},
		{"wildcard resource read cannot write any resource", []string{"*:read"}, "webhooks:write", false},
		{"resource wildcard can write resource", []string{"webhooks:*"}, "webhooks:write", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CheckScope(tt.keyScopes, tt.required)
			if result != tt.expected {
				t.Errorf("CheckScope(%v, %s) = %v, want %v", tt.keyScopes, tt.required, result, tt.expected)
			}
		})
	}
}
