package tenant

import (
	"context"
	"errors"
)

type contextKey int

const (
	ctxKeyTenant contextKey = iota + 1
)

// ErrNoTenant is returned when strict isolation requires a tenant and none is set.
var ErrNoTenant = errors.New("tenant: no tenant in context")

// ErrTenantMismatch is returned when a resource belongs to a different tenant.
var ErrTenantMismatch = errors.New("tenant: resource belongs to another tenant")

// ErrTenantSpoof is returned when a non-admin tries to override the bound tenant.
var ErrTenantSpoof = errors.New("tenant: cannot override tenant for this credential")

// ErrForbidden is returned when the caller lacks permission for the tenant action.
var ErrForbidden = errors.New("tenant: forbidden")

// WithContext attaches a TenantContext to ctx.
func WithContext(ctx context.Context, tc TenantContext) context.Context {
	return context.WithValue(ctx, ctxKeyTenant, tc)
}

// FromContext returns the TenantContext if present.
func FromContext(ctx context.Context) (TenantContext, bool) {
	if ctx == nil {
		return TenantContext{}, false
	}
	tc, ok := ctx.Value(ctxKeyTenant).(TenantContext)
	return tc, ok
}

// IDFromContext returns the tenant ID from context, or empty string.
func IDFromContext(ctx context.Context) string {
	tc, ok := FromContext(ctx)
	if !ok {
		return ""
	}
	return tc.TenantID
}

// Require returns the TenantContext or ErrNoTenant if missing/empty.
func Require(ctx context.Context) (TenantContext, error) {
	tc, ok := FromContext(ctx)
	if !ok || tc.TenantID == "" {
		return TenantContext{}, ErrNoTenant
	}
	return tc, nil
}

// RequireOrDefault returns the tenant from context, or defaultID when isolation is off.
func RequireOrDefault(ctx context.Context, mode IsolationMode, defaultID string) (TenantContext, error) {
	tc, ok := FromContext(ctx)
	if ok && tc.TenantID != "" {
		return tc, nil
	}
	if mode == IsolationOff && defaultID != "" {
		return TenantContext{TenantID: defaultID}, nil
	}
	return TenantContext{}, ErrNoTenant
}
