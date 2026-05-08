---
name: hystersis-api
description: Design, implement, and verify REST API endpoints in the Hystersis Go backend and Next.js proxy. Covers route registration, handler implementation, proxy configuration, and API client integration.
triggers: [api, endpoint, route, handler, REST, CRUD, proxy, register, handler, http, mux]
tools: [Read, Grep, Glob, bash, edit, write]
model: auto
memory_blocks: none
---

# Hystersis API Skill

## Adding a New Endpoint — Step by Step

### Step 1: Backend Route Registration

File: `cmd/server/api.go`

```go
// In setupRoutes() function, add:
s.router.HandleFunc("/your-resource", s.yourListHandler).Methods("GET")
s.router.HandleFunc("/your-resource", s.yourCreateHandler).Methods("POST")
s.router.HandleFunc("/your-resource/{id}", s.yourGetHandler).Methods("GET")
s.router.HandleFunc("/your-resource/{id}", s.yourUpdateHandler).Methods("PUT")
s.router.HandleFunc("/your-resource/{id}", s.yourDeleteHandler).Methods("DELETE")
```

### Step 2: Backend Handler Implementation

File: `cmd/server/api_handlers.go`

```go
func (s *APIServer) yourListHandler(w http.ResponseWriter, r *http.Request) {
    if !isAdmin(r) {
        jsonError(w, "Forbidden: Admin access required", http.StatusForbidden)
        return
    }
    // ... implementation
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(result)
}
```

**IMPORTANT**: Always use `jsonError()` for errors, NEVER `http.Error()`.
**IMPORTANT**: Always set `Content-Type: application/json` before writing response body.

### Step 3: Proxy Configuration

File: `dashboard/src/app/api/proxy/route.ts`

```typescript
// Add to ADMIN_ENDPOINTS (for auto admin-key injection on GET):
const ADMIN_ENDPOINTS = [
  // ... existing entries
  "/your-resource/",
];

// Add to WRITE_ENDPOINTS (for auto key injection on POST/PUT/DELETE):
const WRITE_ENDPOINTS = [
  // ... existing entries
  "/your-resource",
];
```

### Step 4: API Client Method

File: `dashboard/src/lib/api.ts`

```typescript
// Add inside the api object:
yourResource: {
  list: (params?: Record<string, string | number | boolean | undefined>) =>
    request<{ items: YourType[]; count: number }>("/your-resource", { params }),
  get: (id: string) =>
    request<YourType>(`/your-resource/${id}`),
  create: (data: Partial<YourType>) =>
    request<YourType>("/your-resource", { method: "POST", body: JSON.stringify(data) }),
  update: (id: string, data: Partial<YourType>) =>
    request<YourType>(`/your-resource/${id}`, { method: "PUT", body: JSON.stringify(data) }),
  delete: (id: string) =>
    request<void>(`/your-resource/${id}`, { method: "DELETE" }),
},
```

### Step 5: Dashboard Page (if needed)

File: `dashboard/src/app/(dashboard)/your-resource/page.tsx`

Follow the CRUD pattern from `memories/page.tsx`:
- `useQuery` for listing
- `useMutation` for create/update/delete
- Dialog components for create/edit/view
- Toast notifications for success/error
- DropdownMenu for row actions

### Step 6: Sidebar Entry (if needed)

File: `dashboard/src/components/dashboard/sidebar.tsx`

Add to the appropriate group in `sidebarGroups` array:
```typescript
{ href: "/your-resource", label: "Your Resource", icon: YourIcon },
```

### Step 7: Middleware Update (if public page)

File: `dashboard/src/middleware.ts`

```typescript
// Add before auth check:
if (pathname.startsWith("/your-public-route")) {
  return NextResponse.next();
}
```

## API Design Rules

1. **Always use JSON responses** — Never plain text, always `{"error": "message"}` format
2. **Always set Content-Type header** — `w.Header().Set("Content-Type", "application/json")` before Encode
3. **Always use admin checks** — `if !isAdmin(r) { jsonError(w, "Forbidden: Admin access required", http.StatusForbidden); return }`
4. **Always validate input** — `if err := json.NewDecoder(r.Body).Decode(&req); err != nil { jsonError(w, "Invalid request body", http.StatusBadRequest); return }`
5. **Always log errors** — Use `safeHTTPError(w, r, err, http.StatusInternalServerError)` for internal errors
6. **RESTful URL patterns** — List: GET /resource, Create: POST /resource, Get: GET /resource/{id}, Update: PUT /resource/{id}, Delete: DELETE /resource/{id}

## Debugging API Issues

```bash
# Test backend directly
curl -s -H "X-API-Key: am_AYQh3k5V47AVVoyY_1776234755" "http://localhost:8080/your-resource" | python3 -c "import sys,json; print(json.dumps(json.load(sys.stdin), indent=2)[:200])"

# Test through proxy
curl -s "http://localhost:3000/api/proxy?endpoint=%2Fyour-resource" | python3 -c "import sys,json; print(json.dumps(json.load(sys.stdin), indent=2)[:200])"

# Check proxy logs
pm2 logs dashboard --lines 20 --nostream
```