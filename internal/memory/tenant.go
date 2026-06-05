package memory

import "context"

type contextKey string

const tenantCtxKey contextKey = "tenant_id"

// ContextWithTenant returns a new context with the tenant ID set.
func ContextWithTenant(ctx context.Context, tenantID string) context.Context {
	return context.WithValue(ctx, tenantCtxKey, tenantID)
}

// TenantFromContext extracts the tenant ID from the context.
// Returns empty string if no tenant is set.
func TenantFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(tenantCtxKey).(string); ok {
		return v
	}
	return ""
}