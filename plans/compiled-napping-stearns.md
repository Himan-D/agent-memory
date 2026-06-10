# Plan: Best-in-Class Dashboard & Webhooks — Production Polish

## Context

The `dashboard/` directory contains a substantial Next.js 15 dashboard (App Router, TypeScript, Tailwind, shadcn/ui, TanStack Query) with 20+ pages already implemented. The Go backend has 150+ API endpoints covering every resource. However, many pages have bugs, missing features, and incomplete functionality that prevent this from being a production-quality product. The webhook system (both backend and frontend) needs significant enhancement.

This plan addresses every gap found during the audit to make this a best-in-class SaaS dashboard.

---

## Phase 1: Foundation Fixes (sidebar, overview, API client)

### 1.1 Fix Sidebar Bugs
**File:** `dashboard/src/components/dashboard/sidebar.tsx`
- Fix `/` active state — it matches every path because `pathname.startsWith("/")` is always true. Use `pathname === item.href` for exact match on `/`.
- Fix Billing dropdown link — currently points to `/settings`, should point to `/billing`.

### 1.2 Fix Overview/Home Page
**File:** `dashboard/src/app/(dashboard)/page.tsx`
- Fix memory growth chart — currently maps `by_category` (categorical data) to a time-series chart. Use `daily_trend` from the analytics response instead.
- Add agent activity section using `agent_activity` from analytics.
- Add more quick action links (Webhooks, Alerts, Chains, Analytics).
- Show real agent count in stats cards.

---

## Phase 2: Webhook System Overhaul

### 2.1 Backend: Expand Webhook Events
**File:** `internal/webhook/service.go`, `internal/memory/types/types.go`
- Add event types: `search.performed`, `entity.created`, `entity.updated`, `entity.deleted`, `session.created`, `session.ended`, `skill.executed`, `alert.triggered`, `agent.connected`, `agent.disconnected`
- Add `EmitEvent` calls in relevant service methods across the backend
- Add per-webhook delivery stats tracking (success count, failure count, last delivery time, last status code)
- Persist dead-letter queue entries to Neo4j (currently in-memory, lost on restart)

### 2.2 Backend: Webhook Delivery Enhancements
**File:** `internal/webhook/service.go`
- Add `GET /webhooks/{webhookID}/deliveries` endpoint for per-webhook delivery history
- Add `POST /webhooks/{webhookID}/retry` to retry a failed delivery
- Add `PATCH /webhooks/{webhookID}` for partial updates (toggle active, change URL)
- Return delivery stats in webhook list/get responses

### 2.3 Frontend: Full Webhook Management
**File:** `dashboard/src/app/(dashboard)/webhooks/page.tsx`
- Add edit dialog (update URL, toggle events, toggle active/inactive)
- Add delivery history tab per webhook (status codes, timestamps, payloads, retry button)
- Add dead-letter queue viewer with retry capability
- Add webhook health indicator (green/yellow/red based on recent delivery success rate)
- Format `last_triggered` as relative time
- Add event type badges with color coding
- Add "Copy Secret" button for webhook secrets

### 2.4 API Client: Webhook Extensions
**File:** `dashboard/src/lib/api.ts`
- Add `webhooksApi.getDeliveries(id)`, `webhooksApi.retryDelivery(id, deliveryId)`, `webhooksApi.getDeadLetter()`
- Add webhook stats types

---

## Phase 3: Page-by-Page Fixes

### 3.1 Notifications Page Enhancement
**File:** `dashboard/src/app/(dashboard)/notifications/page.tsx`
- Add notification preferences panel (toggle in-app, email, webhook; set email address, webhook URL, mute types)
- Add type filter chips (info/success/warning/error)
- Add search functionality
- Fix missing toast feedback on delete action
- Add notification icon by type (info=blue, success=green, warning=yellow, error=red)

### 3.2 Skills Page Fix
**File:** `dashboard/src/app/(dashboard)/skills/page.tsx`
- Add `prompt` field to edit dialog (it's dropped on edit, so system prompt can't be updated)
- Guard delete button for `is_builtin` skills (hide or disable)
- Clear `deletingId` state on mutation success

### 3.3 Chains Page Enhancement
**File:** `dashboard/src/app/(dashboard)/chains/page.tsx`
- Add step editor (add/reorder/remove steps with skill selection)
- Add execution history tab using `chainsApi.getExecutions()`
- Fix empty `typeOptions` on `FilterComponent`
- Add null guard on `chain.confidence`

### 3.4 Groups Page Enhancement
**File:** `dashboard/src/app/(dashboard)/groups/page.tsx`
- Add role selector when adding members (admin/leader/member)
- Add group skills tab (view skills shared with group)
- Add group shared memories tab
- Fix variable naming (`isCreating` used for edit)
- Fix empty `typeOptions` on filter

### 3.5 Projects Page Fix
**File:** `dashboard/src/app/(dashboard)/projects/page.tsx`
- Fix memory fetching — use project-scoped query or pass `project_id` as filter param instead of client-filtering all memories
- Fix `isCreating` variable naming
- Fix empty `typeOptions`

### 3.6 Documents Page Enhancement
**File:** `dashboard/src/app/(dashboard)/documents/page.tsx`
- Add "Save to Memory" action after extraction (create memory from extracted content)
- Add source document list from `sourcesApi` (if available) or persist extraction history
- Add supported file type display

---

## Phase 4: Real-Time & Polish

### 4.1 Real-Time Dashboard Updates
**File:** `dashboard/src/hooks/use-realtime.ts` (exists but unused)
- Wire SSE connection to `GET /events` endpoint
- Add real-time indicators on overview page (new memories, active agents)
- Show live webhook delivery status updates
- Add connection status indicator in header

### 4.2 Audit Trail Page (New)
**File:** `dashboard/src/app/(dashboard)/audit/page.tsx` (new)
- List audit log entries from backend (`/audit/logs` if available, or via analytics)
- Filter by action type, user, date range
- Add to sidebar under Operations group

### 4.3 Global UX Polish
- Add breadcrumbs on all pages using the existing shadcn breadcrumb component
- Add empty state illustrations on all list pages
- Consistent loading skeletons across all pages (some have them, some don't)
- Add keyboard shortcuts (Cmd+K for search, Escape to close dialogs)
- Add CSV/JSON export on memories, entities, sessions, alerts pages

---

## Phase 5: Production Hardening

### 5.1 Error Handling
- Add error boundaries per page section
- Add retry buttons on failed API calls
- Add offline detection banner

### 5.2 Performance
- Add route-level code splitting (already using lazy for charts, extend to other heavy components)
- Add pagination on all list pages (some already have it, standardize)
- Add virtual scrolling for large lists (memories can be 50K+)

### 5.3 API Proxy Security
**File:** `dashboard/src/app/api/proxy/route.ts`
- Review SSRF allowlist
- Add rate limiting headers forwarding
- Add proper error response formatting

---

## Files to Modify (Summary)

### Backend (Go)
| File | Changes |
|------|---------|
| `internal/memory/types/types.go` | Add new webhook event type constants |
| `internal/webhook/service.go` | Expand events, add delivery stats, persist dead-letter |
| `internal/webhook/neo4j_store.go` | Add delivery log + dead-letter persistence |
| `cmd/server/api.go` | Add new webhook routes |
| `cmd/server/api_handlers.go` | Add webhook delivery/retry handlers |

### Frontend (Next.js)
| File | Changes |
|------|---------|
| `dashboard/src/components/dashboard/sidebar.tsx` | Fix active state, fix billing link |
| `dashboard/src/app/(dashboard)/page.tsx` | Fix chart, add agent activity, quick actions |
| `dashboard/src/app/(dashboard)/webhooks/page.tsx` | Edit, delivery history, dead-letter, health indicator |
| `dashboard/src/app/(dashboard)/notifications/page.tsx` | Preferences, type filter, search |
| `dashboard/src/app/(dashboard)/skills/page.tsx` | Fix edit dialog, guard builtin delete |
| `dashboard/src/app/(dashboard)/chains/page.tsx` | Step editor, execution history |
| `dashboard/src/app/(dashboard)/groups/page.tsx` | Role selector, skills/memories tabs |
| `dashboard/src/app/(dashboard)/projects/page.tsx` | Fix memory fetch |
| `dashboard/src/app/(dashboard)/documents/page.tsx` | Save-to-memory, source list |
| `dashboard/src/app/(dashboard)/audit/page.tsx` | New audit trail page |
| `dashboard/src/lib/api.ts` | Webhook delivery APIs, audit API |
| `dashboard/src/hooks/use-realtime.ts` | Wire SSE for live updates |

---

## Verification

1. `cd dashboard && npm run build` — confirm no TypeScript errors
2. `cd dashboard && npm run dev` — start dev server
3. Start Go backend: `go run cmd/server/*.go`
4. Test each page in browser:
   - Overview loads stats, chart shows daily trend, quick actions work
   - Webhooks: create, edit, toggle active, test, view deliveries, dead-letter
   - Alerts: create rule, toggle, resolve/dismiss active alerts
   - Notifications: filter by type, manage preferences, search
   - Skills: create with prompt, edit with prompt, guard builtins
   - Chains: create with steps, execute, view history
   - Groups: add member with role, view skills/memories
   - Projects: view doesn't fetch all memories
   - Documents: extract → save to memory
   - Sidebar: `/` doesn't highlight everything, billing link goes to `/billing`
5. `go test ./internal/webhook/...` — webhook service tests pass
6. Check SSE real-time updates on overview page

---

## Implementation Order

Execute phases in order (1→5). Within each phase, items are independent and can be parallelized across agents. Backend webhook changes (Phase 2.1-2.2) must complete before frontend webhook work (Phase 2.3-2.4).

Estimated scope: ~15-20 files modified, ~2000-3000 lines of changes.
