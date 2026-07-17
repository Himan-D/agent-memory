package roles

import "context"

// Context keys used by the auth middleware to pass identity information
// downstream. Using typed keys avoids accidental collisions with the
// string-based context values used elsewhere in the codebase.
type ctxKey int

const (
	roleKey ctxKey = iota
	isAdminKey
	userIDKey
	emailKey
)

// WithRole returns a new context carrying the given role.
func WithRole(ctx context.Context, role Role) context.Context {
	return context.WithValue(ctx, roleKey, string(role))
}

// RoleFromContext extracts the role from the context. If no role is set,
// RoleUser is returned as a safe default for endpoints that still expect
// authenticated traffic.
func RoleFromContext(ctx context.Context) Role {
	if ctx == nil {
		return RoleUser
	}
	if v, ok := ctx.Value(roleKey).(string); ok && v != "" {
		return Role(v)
	}
	// Backward compatibility: read legacy "role" string value.
	if v, ok := ctx.Value("role").(string); ok && v != "" {
		return Role(v)
	}
	return RoleUser
}

// WithAdmin returns a new context with the admin flag.
func WithAdmin(ctx context.Context, isAdmin bool) context.Context {
	return context.WithValue(ctx, isAdminKey, isAdmin)
}

// IsAdminFromContext reports whether the request is being made by an admin
// user. The legacy "is_admin" string-based flag is also honoured for
// compatibility with the existing middleware chain.
func IsAdminFromContext(ctx context.Context) bool {
	if ctx == nil {
		return false
	}
	if v, ok := ctx.Value(isAdminKey).(bool); ok {
		return v
	}
	if v, ok := ctx.Value("is_admin").(bool); ok {
		return v
	}
	return false
}

// WithUserID returns a new context carrying the authenticated user ID.
func WithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, userIDKey, userID)
}

// UserIDFromContext returns the authenticated user ID, falling back to the
// legacy "tenant_id" value used by older handlers.
func UserIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if v, ok := ctx.Value(userIDKey).(string); ok && v != "" {
		return v
	}
	if v, ok := ctx.Value("tenant_id").(string); ok {
		return v
	}
	return ""
}

// WithEmail returns a new context carrying the user's email address.
func WithEmail(ctx context.Context, email string) context.Context {
	return context.WithValue(ctx, emailKey, email)
}

// EmailFromContext returns the authenticated user's email if known.
func EmailFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if v, ok := ctx.Value(emailKey).(string); ok {
		return v
	}
	return ""
}

// IsAdminRole reports whether a role is the admin role.
func IsAdminRole(role Role) bool {
	return role == RoleAdmin
}
