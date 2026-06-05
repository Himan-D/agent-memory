package roles

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// RBACEnforcer validates permissions and access control
type RBACEnforcer struct {
	roleMu      sync.RWMutex
	roleCache   map[string]*CachedRole
	cacheTTL    time.Duration
	checker     *Checker
}

// CachedRole stores role with expiration
type CachedRole struct {
	role      Role
	expiresAt time.Time
}

// NewRBACEnforcer creates a new enforcer with 5min cache TTL
func NewRBACEnforcer() *RBACEnforcer {
	return &RBACEnforcer{
		roleCache: make(map[string]*CachedRole),
		cacheTTL:  5 * time.Minute,
		checker:   NewChecker(),
	}
}

// CacheRole stores a role in cache
func (re *RBACEnforcer) CacheRole(key string, role Role) {
	re.roleMu.Lock()
	defer re.roleMu.Unlock()
	re.roleCache[key] = &CachedRole{
		role:      role,
		expiresAt: time.Now().Add(re.cacheTTL),
	}
}

// GetCachedRole retrieves a role from cache if not expired
func (re *RBACEnforcer) GetCachedRole(key string) (Role, bool) {
	re.roleMu.RLock()
	defer re.roleMu.RUnlock()

	cached, ok := re.roleCache[key]
	if !ok {
		return "", false
	}

	if time.Now().After(cached.expiresAt) {
		re.roleMu.RUnlock()
		re.roleMu.Lock()
		delete(re.roleCache, key)
		re.roleMu.Unlock()
		re.roleMu.RLock()
		return "", false
	}

	return cached.role, true
}

// CheckPermissionContext validates if context has required permission
func (re *RBACEnforcer) CheckPermissionContext(ctx context.Context, perm Permission) (bool, error) {
	roleStr, ok := ctx.Value("role").(string)
	if !ok {
		roleStr = "user"
	}

	isAdmin, ok := ctx.Value("is_admin").(bool)
	if ok && isAdmin {
		return true, nil
	}

	role := Role(roleStr)
	if re.checker.HasPermission(role, perm) {
		return true, nil
	}

	return false, fmt.Errorf("insufficient permissions: %s required", perm)
}

// ValidateAPIKeyRole checks if API key role has permission
func (re *RBACEnforcer) ValidateAPIKeyRole(apiKey string, role Role, perm Permission) bool {
	if re.checker.HasPermission(role, perm) {
		re.CacheRole(apiKey, role)
		return true
	}
	return false
}

// InvalidateCache clears all cached roles
func (re *RBACEnforcer) InvalidateCache() {
	re.roleMu.Lock()
	defer re.roleMu.Unlock()
	re.roleCache = make(map[string]*CachedRole)
}
