package tenant

import "strings"

// AssertSameTenant ensures resourceTenant matches the auth tenant (or admin).
// Empty resourceTenant is treated as mismatch under strict mode (legacy data should be backfilled).
func AssertSameTenant(auth TenantContext, resourceTenant string) error {
	if auth.IsAdmin {
		return nil
	}
	if auth.TenantID == "" {
		return ErrNoTenant
	}
	if resourceTenant == "" || resourceTenant != auth.TenantID {
		return ErrTenantMismatch
	}
	return nil
}

// ResolveActingTenant picks the effective tenant for a request.
// boundTenant is from the API key / session.
// headerTenant is X-Tenant-ID (or query tenant_id).
// Under strict isolation, non-admins cannot override; mismatch returns ErrTenantSpoof.
func ResolveActingTenant(boundTenant, headerTenant string, isAdmin bool, mode IsolationMode, defaultTenant string) (string, error) {
	boundTenant = strings.TrimSpace(boundTenant)
	headerTenant = strings.TrimSpace(headerTenant)

	if mode == IsolationOff {
		if boundTenant != "" {
			return boundTenant, nil
		}
		if headerTenant != "" {
			return headerTenant, nil
		}
		if defaultTenant != "" {
			return defaultTenant, nil
		}
		return "default", nil
	}

	// Strict mode
	if isAdmin {
		if headerTenant != "" {
			return headerTenant, nil
		}
		if boundTenant != "" && boundTenant != "admin" {
			return boundTenant, nil
		}
		if defaultTenant != "" {
			return defaultTenant, nil
		}
		return "admin", nil
	}

	if boundTenant == "" {
		if defaultTenant != "" {
			return defaultTenant, nil
		}
		return "", ErrNoTenant
	}

	if headerTenant != "" && headerTenant != boundTenant {
		return "", ErrTenantSpoof
	}
	return boundTenant, nil
}

// CanRoleManage returns whether role may manage members/settings.
func CanRoleManage(role Role) bool {
	return role == RoleOwner || role == RoleAdmin
}

// CanRoleWrite returns whether role may write data.
func CanRoleWrite(role Role) bool {
	switch role {
	case RoleOwner, RoleAdmin, RoleMember, "":
		return true
	default:
		return false
	}
}
