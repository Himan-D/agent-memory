---
name: hystersis-deploy
description: Deploy the Hystersis platform (backend, dashboard, landing page) to production. Covers PM2 process management, nginx configuration, SSL, and verification.
triggers: [deploy, deployment, production, pm2, nginx, restart, rebuild, publish, release, staging]
tools: [Read, bash]
model: auto
memory_blocks: none
---

# Hystersis Deploy Skill

## Production Architecture

```
hystersis.ai (Nginx :443 SSL)
├── / → Landing Page (Vite SPA, /var/www/hystersis/)
├── /demo → Dashboard (Next.js, :3000)
├── /api/ → Dashboard proxy → Backend (:8080)
├── /_next/ → Dashboard static assets (:3000)
├── /playground → Dashboard (:3000)
├── /webhook → Webhook handler (:9000)
└── /llms.txt, /agents.md → Backend (:8080)

dashboard.hystersis.ai (Nginx)
└── / → Dashboard (Next.js, :3000)

api.hystersis.ai (Nginx)
└── / → Backend (Go, :8080)
```

## PM2 Process Management

```bash
# Status
pm2 status

# Restart individual services
pm2 restart backend
pm2 restart dashboard

# Restart both (most common after deployment)
pm2 restart backend && pm2 restart dashboard

# Save process list (persists across reboot)
pm2 save

# View logs
pm2 logs backend --lines 50 --nostream
pm2 logs dashboard --lines 50 --nostream

# Delete and recreate (only if binary path changes)
pm2 delete backend
cd /home/ubuntu/agent-memory && pm2 start ./server --name backend
```

## Full Deployment Sequence

### 1. Backend (Go)
```bash
cd /home/ubuntu/agent-memory
export PATH="/usr/local/go/bin:$PATH"
go build ./cmd/server
pm2 restart backend
sleep 3
curl -s http://localhost:8080/health  # Should return {"status":"ok"}
```

### 2. Dashboard (Next.js)
```bash
cd /home/ubuntu/agent-memory/dashboard
rm -rf .next
npm run build
pm2 restart dashboard
sleep 5
curl -s -o /dev/null -w "%{http_code}" https://dashboard.hystersis.ai/auth/signin  # Should return 200 or 302
```

### 3. Landing Page (Vite)
```bash
cd /home/ubuntu/agent-memory/landing
npm run build
sudo cp -r dist/* /var/www/hystersis/
curl -s https://hystersis.ai/ | grep -o "Hystersis" | head -1  # Should return "Hystersis"
```

### 4. Verify All Services
```bash
# Backend health
curl -s http://localhost:8080/health
curl -s http://localhost:8080/ready

# Backend CRUD
curl -s -H "X-API-Key: am_AYQh3k5V47AVVoyY_1776234755" "http://localhost:8080/memories?limit=1" | python3 -c "import sys,json; d=json.load(sys.stdin); print(f'Memories: {len(d.get(\"memories\",[]))}')"

# Proxy
curl -s -H "X-API-Key: am_AYQh3k5V47AVVoyY_1776234755" "http://localhost:3000/api/proxy?endpoint=%2Fmemories%3Flimit%3D1" | python3 -c "import sys,json; d=json.load(sys.stdin); print(f'Proxy: {len(d.get(\"memories\",[]))} memories')"

# Landing page
curl -s https://hystersis.ai/ | grep -o "Hystersis" | head -1

# Dashboard
curl -s -o /dev/null -w "%{http_code}" https://dashboard.hystersis.ai/auth/signin
```

## Nginx Configuration

Location: `/etc/nginx/sites-enabled/hystersis`

Key directives:
- `try_files $uri $uri/ /index.html;` — SPA routing for landing page
- `/demo` → proxy to `:3000` — Dashboard demo
- `/api/` → proxy to `:3000` — Dashboard API proxy
- `/_next/` → proxy to `:3000` — Dashboard static assets
- `/webhook` → proxy to `:9000` — Webhook handler

After editing nginx config:
```bash
sudo nginx -t && sudo systemctl reload nginx
```

## Environment Variables

### Dashboard (.env.local)
```bash
NEXT_PUBLIC_API_URL=http://localhost:8080
NEXTAUTH_URL=https://dashboard.hystersis.ai
NEXTAUTH_SECRET=<secret>
ADMIN_API_KEY=am_AYQh3k5V47AVVoyY_1776234755
NEXT_PUBLIC_AMPLITUDE_API_KEY=5a684520b5dcd448c4fd3874a8a9b663
```

### Backend (.env)
```bash
NEO4J_URI=bolt://localhost:7687
NEO4J_USER=neo4j
NEO4J_PASSWORD=secret
QDRANT_URL=http://localhost:6333
LLM_API_KEY=<key>
OPENAI_API_KEY=<key>
ADMIN_API_KEY=am_AYQh3k5V47AVVoyY_1776234755
```

## Rollback Procedure

If deployment breaks:
1. `pm2 restart backend` (restarts with existing binary)
2. `cd ~/agent-memory/dashboard && rm -rf .next && npm run build && pm2 restart dashboard`
3. If landing page broken: `cd ~/agent-memory/landing && npm run build && sudo cp -r dist/* /var/www/hystersis/`
4. If nginx broken: `sudo nginx -t && sudo systemctl reload nginx`
5. If DB broken: `docker-compose up -d neo4j qdrant`

## Known Issues and Fixes

| Issue | Symptoms | Fix |
|-------|----------|-----|
| Stale .next cache | `Cannot find module './XXX.js'` | `rm -rf .next && npm run build` |
| Backend down | curl returns connection refused | `pm2 restart backend && sleep 3 && curl localhost:8080/health` |
| OpenAI key expired | Search returns 500, "invalid_api_key" | Update `.env` and `pm2 restart backend` |
| Neo4j down | Entities/sessions return 500 | `docker-compose up -d neo4j` |
| Landing page blank | curl returns HTML but no content | Rebuild: `cd landing && npm run build && sudo cp -r dist/* /var/www/hystersis/` |
| Proxy returns empty body | 500 with no JSON | Backend returned plain text error — check `jsonError()` is used |