<!-- BEGIN:nextjs-agent-rules -->
# This is NOT the Next.js you know

This version has breaking changes — APIs, conventions, and file structure may all differ from your training data. Read the relevant guide in `node_modules/next/dist/docs/` before writing any code. Heed deprecation notices.
<!-- END:nextjs-agent-rules -->

# Hystersis Dashboard - Developer Guide

## Overview
The dashboard provides a UI for managing the Hystersis memory platform. It connects to the backend API via `/api/proxy` routes and must integrate with the proprietary compression engine.

---

## Compression Engine Integration (PROPRIETARY)

### Backend API Endpoints Required

All compression endpoints are accessed via `/api/proxy` using the `ADMIN_API_KEY`:

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/proxy/compression/stats` | GET | Get compression metrics |
| `/api/proxy/compression/mode` | GET | Get current compression mode |
| `/api/proxy/compression/mode` | PUT | Set compression mode |
| `/api/proxy/tier/policy` | GET | Get tier policy |
| `/api/proxy/tier/policy` | PUT | Set tier policy |
| `/api/proxy/search/enhanced` | GET | Spreading activation search |

### Compression Stats Response Format

```typescript
interface CompressionStats {
  accuracy_retention: number;      // Target: ≥0.97
  token_reduction: number;         // Target: 0.80-0.85
  total_tokens_saved: number;
  extractions_performed: number;
  spreading_activations: number;
  avg_latency_ms: number;
  p95_latency_ms: number;
}
```

### Compression Modes

```typescript
enum CompressionMode {
  EXTRACT = "extract",      // 97%+ accuracy, 80-85% reduction
  BALANCED = "balanced",    // 95%+ accuracy, 85-90% reduction
  AGGRESSIVE = "aggressive" // 92%+ accuracy, 90-93% reduction
}
```

### Tier Policies

```typescript
enum TierPolicy {
  AGGRESSIVE = "aggressive",    // 1-day hot storage
  BALANCED = "balanced",        // 7-day hot storage (default)
  CONSERVATIVE = "conservative" // 30-day hot storage
}
```

---

## Required Frontend Components

### 1. Compression Stats Card (Dashboard Home)

Location: `/src/app/(dashboard)/page.tsx`

Add to the stats section:
- `accuracy_retention` - Show as percentage (97.3%)
- `token_reduction` - Show as percentage (84%)
- `total_tokens_saved` - Show formatted number (1.5M tokens saved)
- Visual indicator showing target vs actual

### 2. Compression Mode Selector (Settings)

Location: `/src/app/(dashboard)/settings/page.tsx` (or create new)

```
┌─────────────────────────────────────────┐
│  Compression Mode                       │
├─────────────────────────────────────────┤
│  ○ EXTRACT   (97%+ accuracy) [Default] │
│  ○ BALANCED  (95%+ accuracy)            │
│  ○ AGGRESSIVE (92%+ accuracy)           │
└─────────────────────────────────────────┘
```

API: `PUT /api/proxy/compression/mode` with `{ "mode": "extract" }`

### 3. Tier Policy Selector (Settings)

```
┌─────────────────────────────────────────┐
│  Memory Tier Policy                     │
├─────────────────────────────────────────┤
│  ○ AGGRESSIVE  (1-day hot)              │
│  ● BALANCED    (7-day hot) [Default]    │
│  ○ CONSERVATIVE (30-day hot)            │
└─────────────────────────────────────────┘
```

API: `PUT /api/proxy/tier/policy` with `{ "policy": "balanced" }`

### 4. Enhanced Search Toggle (Memories Page)

Location: `/src/app/(dashboard)/memories/page.tsx`

Add search mode toggle:
- Standard (vector similarity) - default
- Enhanced (spreading activation) - uses proprietary graph traversal

```
┌─────────────────────────────────────────┐
│  [Search Input...]  [Mode: ▼ Enhanced] │
└─────────────────────────────────────────┘
```

API: `GET /api/proxy/search/enhanced?mode=spreading&query=...`

---

## API Integration

### Environment Variables

```bash
# .env.local
NEXT_PUBLIC_API_URL=http://localhost:8080
ADMIN_API_KEY=your-admin-api-key  # For compression endpoints
```

### API Client Usage

```typescript
// lib/api/compression.ts
import { apiClient } from './client';

export const compressionApi = {
  getStats: () => apiClient.get('/compression/stats'),
  getMode: () => apiClient.get('/compression/mode'),
  setMode: (mode: string) => apiClient.put('/compression/mode', { mode }),
  getTierPolicy: () => apiClient.get('/tier/policy'),
  setTierPolicy: (policy: string) => apiClient.put('/tier/policy', { policy }),
  searchEnhanced: (query: string, mode: string) => 
    apiClient.get(`/search/enhanced?mode=${mode}&query=${encodeURIComponent(query)}`)
};
```

---

## Frontend Components to Create/Update

| Component | Location | Action |
|-----------|----------|--------|
| CompressionStatsCard | `/src/components/dashboard/compression-stats.tsx` | CREATE |
| CompressionModeSelector | `/src/components/settings/compression-mode.tsx` | CREATE |
| TierPolicySelector | `/src/components/settings/tier-policy.tsx` | CREATE |
| SearchModeToggle | `/src/app/(dashboard)/memories/` | UPDATE |
| DashboardPage | `/src/app/(dashboard)/page.tsx` | UPDATE |

---

## Project Structure

```
src/
├── app/
│   ├── (dashboard)/
│   │   ├── page.tsx              # Dashboard - add compression stats
│   │   ├── memories/page.tsx    # Add search mode toggle
│   │   ├── settings/page.tsx    # Add compression/tier settings
│   │   └── ...
│   └── api/proxy/               # Backend proxy routes
├── components/
│   ├── dashboard/
│   │   └── compression-stats.tsx  # NEW
│   └── settings/
│       ├── compression-mode.tsx   # NEW
│       └── tier-policy.tsx         # NEW
└── lib/
    ├── api/
    │   └── compression.ts         # NEW
    └── utils/
```

---

## Build & Test

```bash
npm run dev     # Development server
npm run build   # Production build
npm run lint   # Lint check
```

---

## Key Points

1. **Compression is PROPRIETARY** - We show the results in UI but don't expose the algorithm
2. **Admin API key required** - Compression endpoints need ADMIN_API_KEY
3. **Stats polling** - Poll `/compression/stats` every 30 seconds for real-time updates
4. **Mode changes** - Should trigger a toast notification confirming the change

---

## Integration with Backend

The dashboard communicates with backend via `/api/proxy` routes which forward to the Go backend. Ensure:

1. Backend has compression endpoints implemented
2. ADMIN_API_KEY is set in both backend and frontend
3. Neo4j and Qdrant are running for full functionality

---

## Existing Conventions

- Use `safeHTTPError()` not `http.Error()` in API routes
- Component naming: PascalCase (e.g., `CompressionStatsCard`)
- API client: Always use `/api/proxy` prefix for backend calls
- UI components: Use Radix UI primitives (Dialog, Popover, DropdownMenu)
- Styling: Tailwind CSS