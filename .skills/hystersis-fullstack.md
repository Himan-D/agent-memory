---
name: hystersis-fullstack
description: Full-stack Hystersis platform expert covering both Go backend and Next.js dashboard. Use this skill for any cross-cutting change that touches both backend and frontend, API design, proxy configuration, or data flow between services.
triggers: [hystersis, platform, fullstack, backend, frontend, api, proxy, crud, endpoint, route, handler, migration, deploy]
tools: [Read, Glob, Grep, bash, edit, write]
model: auto
memory_blocks: none
---

# Hystersis Full-Stack Development Skill

## Architecture Overview

```
Browser → Nginx (hystersis.ai/:443) → Landing Page (Vite SPA, /var/www/hystersis/)
                                    → Dashboard (Next.js, :3000)
                                    → Backend (Go, :8080)
```

## Critical File Locations

### Backend (Go)
- **API routes**: `cmd/server/api.go` — all route registrations (search for `HandleFunc`)
- **API handlers**: `cmd/server/api_handlers.go` — handler implementations
- **Core service**: `internal/memory/service.go` — memory CRUD, search, compression
- **Graph store**: `internal/memory/neo4j/` — Neo4j operations
- **Vector store**: `internal/vector/` — Qdrant operations
- **Types**: `internal/memory/types/` — all type definitions

### Frontend (Next.js Dashboard)
- **Sidebar**: `dashboard/src/components/dashboard/sidebar.tsx` — 19 items in 5 groups
- **API client**: `dashboard/src/lib/api.ts` — all API calls via proxy
- **Proxy**: `dashboard/src/app/api/proxy/route.ts` — backend proxy with admin key injection
- **Pages**: `dashboard/src/app/(dashboard)/*/page.tsx` — CRUD pages
- **Demo**: `dashboard/src/app/demo/page.tsx` — public playground (no auth)
- **Middleware**: `dashboard/src/middleware.ts` — auth routes, public routes

### Landing Page (Vite SPA)
- **App**: `landing/src/App.jsx` — routes and scroll handling
- **Navbar**: `landing/src/components/Navbar.jsx` — navigation links
- **Deploy**: `landing/dist/` → `/var/www/hystersis/`

## Proxy Configuration Rules

When adding a new backend endpoint:

1. Add to `ADMIN_ENDPOINTS` in `dashboard/src/app/api/proxy/route.ts` if it needs admin key auto-injection
2. Add to `WRITE_ENDPOINTS` if it needs key injection for POST/PUT/DELETE
3. Add to `USER_ENDPOINTS` if it's a user-level operation (api-keys, notifications)
4. Add API method in `dashboard/src/lib/api.ts` using `request<T>()` helper
5. Use `useAdminKey: true` for admin-only calls
6. Use FormData for file uploads (Content-Type is auto-detected)

## Error Handling Rules

### Backend (Go)
- Use `jsonError(w, "message", statusCode)` for ALL error responses
- NEVER use `http.Error()` — it returns plain text which crashes the proxy
- Use `safeHTTPError(w, r, err, statusCode)` for internal errors (logs real error, sends generic message)

### Frontend (TypeScript)
- Use `useMutation` for writes with toast notifications
- Use `useQuery` for reads with loading/error states
- All API errors are now JSON: `{ "error": "message" }`

## Build & Deploy Commands

```bash
# Backend
cd /home/ubuntu/agent-memory && export PATH="/usr/local/go/bin:$PATH" && go build ./cmd/server

# Dashboard
cd /home/ubuntu/agent-memory/dashboard && rm -rf .next && npm run build

# Landing Page
cd /home/ubuntu/agent-memory/landing && npm run build && sudo cp -r dist/* /var/www/hystersis/

# PM2
pm2 restart backend && pm2 restart dashboard && pm2 save
```

## Testing Checklist (Run After Every Change)

```bash
# Backend health
curl -s http://localhost:8080/health  # → {"status":"ok"}

# Dashboard proxy (key endpoints)
curl -s -H "X-API-Key: <YOUR_ADMIN_API_KEY>" "http://localhost:3000/api/proxy?endpoint=%2Fmemories%3Flimit%3D1" | python3 -c "import sys,json; d=json.load(sys.stdin); print(f'Memories: {len(d.get(\"memories\",[]))}')"

# 404 returns JSON (not plain text)
curl -s -w "\nHTTP: %{http_code}" -H "X-API-Key: <YOUR_ADMIN_API_KEY>" "http://localhost:3000/api/proxy?endpoint=%2Fmemories%2Fnonexistent" | head -3

# Landing page
curl -s https://hystersis.ai/ | grep -o "Hystersis" | head -1

# Dashboard page loads
curl -s -o /dev/null -w "%{http_code}" https://dashboard.hystersis.ai/auth/signin
```

## Common Pitfalls

1. **NEVER use `http.Error()` in Go** — use `jsonError()` instead
2. **NEVER put `ADMIN_API_KEY` in client-side code** — it's server-only, proxy injects it
3. **NEVER set `Content-Type: application/json` for FormData** — let the browser set the boundary
4. **ALWAYS add new endpoints to proxy lists** — otherwise 401 or 404
5. **ALWAYS use `pathname.startsWith(item.href + "/")` for sidebar active state** — exact match breaks on detail pages
6. **Graph queries use `cypher` field** — NOT `query` (matches backend struct)
7. **Landing page is a Vite SPA** — nginx must `try_files $uri $uri/ /index.html` for client-side routing