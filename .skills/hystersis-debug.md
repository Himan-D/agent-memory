---
name: hystersis-debug
description: Debug and fix issues across the Hystersis platform. Covers backend errors, frontend crashes, proxy failures, deployment issues, and nginx misconfiguration.
triggers: [debug, error, crash, 500, 404, 401, fix, broken, not working, failing, timeout, deploy issue]
tools: [Read, Grep, Glob, bash]
model: auto
memory_blocks: none
---

# Hystersis Debug Skill

## Debug Decision Tree

### Backend 500 Error
```bash
# 1. Check if backend is running
pm2 status backend

# 2. Check recent logs (most common issues)
pm2 logs backend --lines 50 --nostream

# 3. Common causes:
#    - OpenAI API key invalid/expired → "openai API error (401)"
#    - Neo4j connection lost → "connection refused" on port 7687
#    - Qdrant down → "connection refused" on port 6334
#    - Panic in handler → "panic: runtime error"

# 4. Restart if needed
pm2 restart backend && sleep 5 && curl -s http://localhost:8080/health
```

### Frontend Build Failure
```bash
# 1. Build with verbose output
cd /home/ubuntu/agent-memory/dashboard && rm -rf .next && npm run build 2>&1 | tail -40

# 2. Common TypeScript errors:
#    - Type mismatch in Select onValueChange → cast: (v: string | null) => { if (v) setMode(v) }
#    - Missing import → add to import statement
#    - Duplicate interface → remove the duplicate, keep the one with all fields

# 3. Common fixes:
#    - `ADMIN_API_KEY` in client code → remove, it's server-only
#    - `searchEnhanced` not found → use `api.search.enhanced()` instead
#    - `alertsApi.compression.searchEnhanced` → replaced by `api.search.enhanced`
```

### Proxy 500 Error
```bash
# 1. Test backend directly (bypass proxy)
curl -s -H "X-API-Key: <YOUR_ADMIN_API_KEY>" "http://localhost:8080/ENDPOINT" | head -5

# 2. Test through proxy
curl -s "http://localhost:3000/api/proxy?endpoint=%2FENDPOINT" | head -5

# 3. If proxy returns 500 but backend works:
#    - Check dashboard logs: pm2 logs dashboard --lines 20 --nostream
#    - Common cause: response.json() on non-JSON response
#    - Fix: safeFetchResponse() helper in route.ts handles both JSON and plain text

# 4. If proxy returns 401:
#    - Check if endpoint is in ADMIN_ENDPOINTS, WRITE_ENDPOINTS, or USER_ENDPOINTS
#    - Check if admin key is being injected properly
```

### Landing Page Not Rendering
```bash
# 1. Check nginx serves static files
ls -la /var/www/hystersis/assets/

# 2. Check nginx config for SPA routing
cat /etc/nginx/sites-enabled/hystersis | grep "try_files"
# Should be: try_files $uri $uri/ /index.html;

# 3. Rebuild and deploy
cd /home/ubuntu/agent-memory/landing && npm run build && sudo cp -r dist/* /var/www/hystersis/

# 4. Test
curl -s https://hystersis.com/ | grep -o "Hystersis" | head -1
```

### Hash Anchor Navigation (e.g., #for-agents)
```bash
# Problem: React Router <Link to="#for-agents"> doesn't scroll
# Fix: Use scrollToSection() function in Navbar.jsx
# Pattern: { path: 'section:for-agents', label: 'For Developers' }
# This calls navigate('/#for-agents') which triggers ScrollToHash component
# ScrollToHash uses document.getElementById(id).scrollIntoView()
```

## Log Analysis Quick Reference

| Pattern | Cause | Fix |
|---------|-------|-----|
| `json: cannot unmarshal` | Client sending wrong JSON shape | Check request body matches backend struct |
| `connection refused :7687` | Neo4j down | `docker-compose up -d neo4j` |
| `connection refused :6334` | Qdrant down | `docker-compose up -d qdrant` |
| `openai API error (401)` | Invalid/expired API key | Update `.env` LLM_API_KEY or OPENAI_API_KEY |
| `panic: runtime error` | Nil pointer in handler | Check handler for nil checks |
| `Unexpected token 'U'` | Backend returned plain text, proxy tried JSON | Ensure `jsonError()` used (not `http.Error()`) |
| `ADMIN_API_KEY empty` | Client-side code referenced removed var | Remove `ADMIN_API_KEY` from client code, use proxy |
| `Cannot find module './XXX.js'` | Stale .next cache | `rm -rf .next && npm run build` |