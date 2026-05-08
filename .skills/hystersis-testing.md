---
name: hystersis-testing
description: End-to-end testing and verification of the Hystersis platform. Covers API endpoint testing, proxy verification, frontend smoke tests, and production health checks.
triggers: [test, testing, verify, verification, smoke, health, check, e2e, integration, endpoint, curl]
tools: [bash]
model: auto
memory_blocks: none
---

# Hystersis Testing Skill

## Quick Health Check (Run After Every Deployment)

```bash
#!/bin/bash
# Hystersis Platform Health Check
echo "=== Backend Health ==="
curl -s http://localhost:8080/health && echo ""
curl -s http://localhost:8080/ready && echo ""

echo "=== Backend CRUD ==="
API_KEY="am_AYQh3k5V47AVVoyY_1776234755"
curl -s -H "X-API-Key: $API_KEY" "http://localhost:8080/memories?limit=1" | python3 -c "import sys,json; d=json.load(sys.stdin); print(f'Memories: {len(d.get(\"memories\",[]))}')" 2>/dev/null || echo "FAIL"
curl -s -H "X-API-Key: $API_KEY" "http://localhost:8080/agents?limit=1" | python3 -c "import sys,json; d=json.load(sys.stdin); print(f'Agents: {len(d.get(\"agents\",[]))}')" 2>/dev/null || echo "FAIL"
curl -s -H "X-API-Key: $API_KEY" "http://localhost:8080/skills?limit=1" | python3 -c "import sys,json; d=json.load(sys.stdin); print(f'Skills: {len(d.get(\"skills\",[]))}')" 2>/dev/null || echo "FAIL"
curl -s -H "X-API-Key: $API_KEY" "http://localhost:8080/compression/stats" | python3 -c "import sys,json; d=json.load(sys.stdin); print(f'Compression: {d.get(\"accuracy_retention\", 0)} accuracy')" 2>/dev/null || echo "FAIL"

echo "=== Dashboard Proxy ==="
curl -s -H "X-API-Key: $API_KEY" "http://localhost:3000/api/proxy?endpoint=%2Fmemories%3Flimit%3D1" | python3 -c "import sys,json; d=json.load(sys.stdin); print(f'Proxy: {len(d.get(\"memories\",[]))} memories')" 2>/dev/null || echo "FAIL"
curl -s -w "\nHTTP: %{http_code}\n" "http://localhost:3000/api/proxy?endpoint=%2Fmemories%2Fnonexistent" -H "X-API-Key: $API_KEY" | head -2

echo "=== Landing Page ==="
curl -s https://hystersis.ai/ | grep -o "Hystersis" | head -1 || echo "FAIL"

echo "=== Dashboard ==="
curl -s -o /dev/null -w "Dashboard signin: %{http_code}\n" https://dashboard.hystersis.ai/auth/signin

echo "=== PM2 ==="
pm2 list
```

## Full Endpoint Test Suite

### Authentication Tests
```bash
# Without API key → 401
curl -s -w "\nHTTP: %{http_code}" "http://localhost:8080/memories" | tail -2
# Expected: {"error":"Unauthorized: Invalid or missing API key"} / HTTP: 401

# With wrong API key → 401
curl -s -w "\nHTTP: %{http_code}" -H "X-API-Key: wrong-key" "http://localhost:8080/memories" | tail -2
# Expected: {"error":"Unauthorized: Invalid or missing API key"} / HTTP: 401

# With correct API key → 200
curl -s -w "\nHTTP: %{http_code}" -H "X-API-Key: am_AYQh3k5V47AVVoyY_1776234755" "http://localhost:8080/memories?limit=1" | head -1
# Expected: {"memories":[...]} / HTTP: 200
```

### CRUD Endpoint Tests
```bash
API_KEY="am_AYQh3k5V47AVVoyY_1776234755"

# Create
MEM_ID=$(curl -s -X POST "http://localhost:8080/memories" -H "X-API-Key: $API_KEY" -H "Content-Type: application/json" -d '{"content":"test memory","agent_id":"test"}' | python3 -c "import sys,json; print(json.load(sys.stdin).get('id',''))" 2>/dev/null)
echo "Created memory: $MEM_ID"

# Read
curl -s -H "X-API-Key: $API_KEY" "http://localhost:8080/memories/$MEM_ID" | python3 -c "import sys,json; d=json.load(sys.stdin); print(f'Content: {d.get(\"content\",\"\")}')" 2>/dev/null

# Update
curl -s -X PUT "http://localhost:8080/memories/$MEM_ID" -H "X-API-Key: $API_KEY" -H "Content-Type: application/json" -d '{"content":"updated test memory"}' | python3 -c "import sys,json; d=json.load(sys.stdin); print(f'Updated: {d.get(\"content\",\"\")}')" 2>/dev/null

# Delete
curl -s -X DELETE "http://localhost:8080/memories/$MEM_ID" -H "X-API-Key: $API_KEY" | python3 -c "import sys,json; print(json.load(sys.stdin))" 2>/dev/null
```

### Proxy Tests (Through Dashboard)
```bash
# API key endpoint auto-injection
curl -s "http://localhost:3000/api/proxy?endpoint=%2Fmemories%3Flimit%3D1" | python3 -c "import sys,json; d=json.load(sys.stdin); print(f'Proxy auto-inject: {len(d.get(\"memories\",[]))} memories')" 2>/dev/null

# Admin key endpoint auto-injection
curl -s "http://localhost:3000/api/proxy?endpoint=%2Fcompression%2Fstats" | python3 -c "import sys,json; d=json.load(sys.stdin); print(f'Compression stats: {d}')" 2>/dev/null

# 404 returns JSON (not plain text)
curl -s -w "\nHTTP: %{http_code}" "http://localhost:3000/api/proxy?endpoint=%2Fmemories%2Fnonexistent" -H "X-API-Key: am_AYQh3k5V47AVVoyY_1776234755" | head -2
# Expected: {"error":"resource not found"} / HTTP: 404
```

### Search Tests
```bash
API_KEY="am_AYQh3k5V47AVVoyY_1776234755"

# Vector search
curl -s -H "X-API-Key: $API_KEY" "http://localhost:8080/search/enhanced?query=test&mode=vector" | python3 -c "import sys,json; d=json.load(sys.stdin); print(f'Vector search: mode={d.get(\"mode\")}, results={len(d.get(\"results\",[]))}')" 2>/dev/null

# Spreading activation search
curl -s -H "X-API-Key: $API_KEY" "http://localhost:8080/search/enhanced?query=test&mode=spreading" | python3 -c "import sys,json; d=json.load(sys.stdin); print(f'Spreading search: mode={d.get(\"mode\")}, results={len(d.get(\"results\",[]))}')" 2>/dev/null
```

### Playground Tests
```bash
# Compression test
curl -s -X POST "http://localhost:3000/api/proxy?endpoint=%2Fplayground%2Fcompress" -H "Content-Type: application/json" -d '{"text":"machine learning is a subset of artificial intelligence","modes":["extraction"]}' | python3 -c "import sys,json; d=json.load(sys.stdin); print(f'Compression: best_mode={d.get(\"best_mode\")}, reduction={d.get(\"results\",{}).get(\"extraction\",{}).get(\"reduction_percent\",0)}')" 2>/dev/null

# Playground stats
curl -s "http://localhost:3000/api/proxy?endpoint=%2Fplayground%2Fstats" | python3 -c "import sys,json; d=json.load(sys.stdin); print(f'Stats: {d}')" 2>/dev/null
```

## Regression Test Checklist

After any change to these files, verify:

| Changed File | Test |
|-------------|------|
| `api.go` (routes) | All CRUD endpoints respond 200 |
| `api_handlers.go` | Error responses return JSON, not plain text |
| `api.ts` (client) | Dashboard CRUD pages work without errors |
| `proxy/route.ts` | Admin endpoints auto-inject key; 404 returns JSON |
| `sidebar.tsx` | All 19 sidebar links resolve; nested routes highlight active |
| `middleware.ts` | `/demo` accessible without auth; `/memories` requires auth |
| `Navbar.jsx` | Hash anchors scroll correctly; external links work |
| `App.jsx` | `/#for-agents` scrolls to section on page load |
| `api.go` (jsonError) | 404/401/403 all return `{"error":"..."}` |

## Performance Baseline

| Endpoint | Expected Latency | Max Acceptable |
|----------|-----------------|---------------|
| GET /health | <1ms | 10ms |
| GET /memories?limit=10 | <50ms | 200ms |
| GET /search/enhanced | <200ms | 500ms |
| POST /playground/compress | <500ms | 2000ms |
| Dashboard page load | <2s | 5s |
| Landing page load | <1s | 3s |