# Feature: RBAC Enforcement

**Priority**: P0 — All endpoints are currently open to any authenticated API key.
**Status**: `internal/roles/roles.go` has types and tests (89 lines) but is never called by the API.
**Estimated effort**: 1-2 days

---

## What Needs to Be Built

1. A `requireScope(scope string)` middleware in `cmd/server/api.go`
2. Scope stored on each API key (already has `Scopes []string` in `AdminAPIKey`)
3. Default scope assignment when creating keys without explicit scopes
4. Three scopes: `read`, `write`, `admin`

---

## Scope Definitions

| Scope | Can access |
|-------|-----------|
| `read` | GET on all resources: memories, entities, sessions, skills, search |
| `write` | read + POST/PUT/DELETE on memories, entities, sessions, skills, chains, feedback |
| `admin` | write + /admin/*, /api-keys (create/delete), /users, /invites |

---

## Step 1: Update `internal/roles/roles.go`

Add a `HasScope(apiKey string, required string) bool` function. The API key record is looked up from the auth middleware context.

```go
// CheckScope verifies that the provided key has at least the required scope
// Scope hierarchy: admin ⊃ write ⊃ read
func CheckScope(keyScopes []string, required string) bool {
    scopeLevel := map[string]int{"read": 1, "write": 2, "admin": 3}
    
    requiredLevel := scopeLevel[required]
    for _, s := range keyScopes {
        if scopeLevel[s] >= requiredLevel {
            return true
        }
    }
    return false
}
```

---

## Step 2: Add Middleware in `cmd/server/api.go`

Find the existing `requireAuth` middleware. Add `requireScope` below it:

```go
// requireScope checks that the authenticated key has the required scope.
// Must be chained AFTER requireAuth (which sets the key in context).
func (a *API) requireScope(scope string) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            keyScopes, ok := r.Context().Value(contextKeyScopes).([]string)
            if !ok || !roles.CheckScope(keyScopes, scope) {
                safeHTTPError(w, http.StatusForbidden, fmt.Errorf("scope %q required", scope))
                return
            }
            next.ServeHTTP(w, r)
        })
    }
}
```

Set `contextKeyScopes` in the `requireAuth` middleware when validating the key — read scopes from the `AdminAPIKey.Scopes` field.

---

## Step 3: Apply Scopes to Routes

In `cmd/server/api.go`, in the route registration section, wrap handlers:

```go
// Read-only endpoints — require "read" scope
r.With(a.requireAuth, a.requireScope("read")).Get("/memories", a.handleListMemories)
r.With(a.requireAuth, a.requireScope("read")).Get("/memories/{id}", a.handleGetMemory)
r.With(a.requireAuth, a.requireScope("read")).Get("/search", a.handleSearch)
r.With(a.requireAuth, a.requireScope("read")).Get("/entities", a.handleListEntities)
r.With(a.requireAuth, a.requireScope("read")).Get("/skills", a.handleListSkills)
// ... all other GET endpoints

// Write endpoints — require "write" scope
r.With(a.requireAuth, a.requireScope("write")).Post("/memories", a.handleCreateMemory)
r.With(a.requireAuth, a.requireScope("write")).Put("/memories/{id}", a.handleUpdateMemory)
r.With(a.requireAuth, a.requireScope("write")).Delete("/memories/{id}", a.handleDeleteMemory)
r.With(a.requireAuth, a.requireScope("write")).Post("/sessions", a.handleCreateSession)
// ... all other POST/PUT/DELETE endpoints

// Admin endpoints — require "admin" scope
r.With(a.requireAuth, a.requireScope("admin")).Get("/admin/api-keys", a.handleListAPIKeys)
r.With(a.requireAuth, a.requireScope("admin")).Post("/admin/api-keys", a.handleCreateAPIKey)
r.With(a.requireAuth, a.requireScope("admin")).Delete("/admin/api-keys/{id}", a.handleDeleteAPIKey)
r.With(a.requireAuth, a.requireScope("admin")).Get("/admin/users", a.handleListUsers)
// ... all other /admin/* endpoints
```

---

## Step 4: Default Scopes

When `ADMIN_API_KEYS` env var creates keys without explicit scopes, assign `["write"]` by default. Admin keys get `["admin"]`.

In `cmd/server/api.go`, when parsing `ADMIN_API_KEYS`:
```go
for _, keyStr := range strings.Split(config.AdminAPIKeys, ",") {
    parts := strings.SplitN(keyStr, ":", 3)
    key := parts[0]
    tenant := ""
    scopes := []string{"write"}  // default scope
    
    if len(parts) >= 2 {
        tenant = parts[1]
    }
    if len(parts) >= 3 {
        scopes = strings.Split(parts[2], "+")  // e.g. "key:tenant:read+write"
    }
    // store key with scopes
}
```

---

## Step 5: API Key Creation Endpoint

When creating a key via `POST /admin/api-keys`, accept `scopes` in the request body:

```json
{
  "label": "my-readonly-key",
  "scopes": ["read"],
  "expires_in_hours": 8760
}
```

Default to `["write"]` if `scopes` not provided.

---

## Tests

`internal/roles/roles_test.go` (already 292 lines) — verify these cases:

```go
// CheckScope hierarchy
assert.True(t, CheckScope([]string{"admin"}, "read"))    // admin can read
assert.True(t, CheckScope([]string{"admin"}, "write"))   // admin can write
assert.True(t, CheckScope([]string{"write"}, "read"))    // write can read
assert.False(t, CheckScope([]string{"read"}, "write"))   // read cannot write
assert.False(t, CheckScope([]string{"read"}, "admin"))   // read cannot admin
assert.False(t, CheckScope([]string{"write"}, "admin"))  // write cannot admin
assert.False(t, CheckScope(nil, "read"))                 // no scopes → deny all
```

Add integration test in `cmd/server/e2e_test.go`:
```go
// Read key cannot create memory
resp := doRequest("POST", "/memories", readKey, `{"content":"test","user_id":"u1"}`)
assert.Equal(t, 403, resp.StatusCode)

// Write key can create memory
resp = doRequest("POST", "/memories", writeKey, `{"content":"test","user_id":"u1"}`)
assert.Equal(t, 201, resp.StatusCode)

// Write key cannot list admin API keys
resp = doRequest("GET", "/admin/api-keys", writeKey, "")
assert.Equal(t, 403, resp.StatusCode)
```
