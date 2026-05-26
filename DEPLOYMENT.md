# Hystersis Authentication System Deployment Guide

## Overview
This guide covers the deployment of the complete Hystersis authentication system including:
- Landing page with authentication
- Dashboard with NextAuth integration  
- Backend API with authentication endpoints

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                    AUTHENTICATION SYSTEM                          │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌─────────────┐    ┌─────────────────┐    ┌─────────────────┐ │
│  │  Landing    │    │   Dashboard     │    │   Backend API   │ │
│  │  Page       │◄──►│   (NextAuth)   │◄──►│    (Go)         │ │
│  │             │    │                 │    │                 │ │
│  │ - AuthModal │    │ - Sign In/Up    │    │ - /auth/login   │ │
│  │ - UserMenu  │    │ - User Sessions │    │ - /auth/reg    │ │
│  │ - Context   │    │ - API Keys      │    │ - User Store    │ │
│  └─────────────┘    └─────────────────┘    └─────────────────┘ │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

## Prerequisites

### Infrastructure
- Docker & Docker Compose
- Node.js v18+ (for local development)
- Go 1.21+ (for backend development)

### Environment Variables

#### Backend (api.env)
```bash
# Neo4j
NEO4J_URI=bolt://localhost:7687
NEO4J_USER=neo4j
NEO4J_PASSWORD=your-neo4j-password

# Qdrant
QDRANT_URL=http://localhost:6333

# LLM (Optional for compression)
LLM_PROVIDER=openai
LLM_API_KEY=your-llm-api-key

# Compression Engine
COMPRESSION_ENABLED=true
COMPRESSION_LLM_FAST_PROVIDER=openai
COMPRESSION_LLM_FAST_MODEL=gpt-4o-mini
COMPRESSION_LLM_VERIFY_PROVIDER=anthropic
COMPRESSION_LLM_VERIFY_MODEL=claude-3-5-sonnet
COMPRESSION_COMPLEXITY_THRESHOLD=0.6
TIER_POLICY=balanced

# Auth
NEXTAUTH_SECRET=your-nextauth-secret
NEXTAUTH_URL=https://hystersis.ai
```

#### Dashboard (.env.local)
```bash
NEXT_PUBLIC_API_URL=https://api.hystersis.ai
NEXTAUTH_URL=https://dashboard.hystersis.ai
NEXTAUTH_SECRET=your-nextauth-secret
ADMIN_API_KEY=<YOUR_ADMIN_API_KEY>
```

#### Landing Page (No additional env needed)

## Deployment Options

### Option 1: Docker Compose (Development)

```yaml
# docker-compose.yml
version: '3.8'

services:
  # Backend API
  api:
    build: ./cmd/server
    ports:
      - "8080:8080"
    environment:
      - NEO4J_URI=bolt://neo4j:7687
      - NEO4J_USER=neo4j
      - NEO4J_PASSWORD=neo4j
      - QDRANT_URL=http://qdrant:6333
    depends_on:
      - neo4j
      - qdrant

  # Neo4j
  neo4j:
    image: neo4j:latest
    environment:
      - NEO4J_AUTH=neo4j/neo4j
    ports:
      - "7687:7687"
      - "7474:7474"
    volumes:
      - neo4j_data:/data

  # Qdrant
  qdrant:
    image: qdrant/qdrant:latest
    ports:
      - "6333:6333"
    volumes:
      - qdrant_data:/qdrant/data

  # Dashboard
  dashboard:
    build: ./dashboard
    ports:
      - "3000:3000"
    environment:
      - NEXT_PUBLIC_API_URL=http://localhost:8080
      - NEXTAUTH_URL=http://localhost:3000
      - NEXTAUTH_SECRET=dev-secret-key
    depends_on:
      - api

  # Landing Page
  landing:
    build: ./landing
    ports:
      - "5000:5000"

volumes:
  neo4j_data:
  qdrant_data:
```

### Option 2: Production Deployment

#### Backend
```bash
# Build and run backend
cd cmd/server
go build -o server .
./server
```

#### Dashboard
```bash
cd dashboard
npm run build
npm run start
```

#### Landing Page
```bash
cd landing
npm run build
# Use any static server
npx serve -s dist -l 5000
```

## Authentication Flow

### 1. User Registration
```javascript
// Landing Page
const result = await register(email, password, name)
if (result.success) {
  // User logged in automatically
}
```

### 2. User Login
```javascript
// Landing Page or Dashboard
const result = await login(email, password)
if (result.success) {
  // Token stored in localStorage (landing) 
  // or session (dashboard)
}
```

### 3. Dashboard Integration
```javascript
// Dashboard uses NextAuth with custom credentials provider
// Automatically redirects to /auth/signin when not authenticated
```

### 4. API Protection
```go
// Backend automatically protects all endpoints except:
// - /auth/login
// - /auth/register
// - /health, /ready, /status, /metrics
```

## Testing the Authentication System

### 1. Test Landing Page Authentication
```bash
# Start landing page
cd landing
npm run dev
# Visit http://localhost:5000
# Click "Sign In" -> Try signing up/logging in
```

### 2. Test Dashboard Authentication
```bash
# Start dashboard
cd dashboard  
npm run dev
# Visit http://localhost:3000
# Should redirect to /auth/signin
```

### 3. Test Backend Endpoints
```bash
# Test login
curl -X POST http://localhost:8080/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"demo@hystersis.ai","password":"demo123"}'

# Test registration  
curl -X POST http://localhost:8080/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"test123","name":"Test User"}'
```

## Demo Credentials

### Default Demo User
```json
{
  "email": "demo@hystersis.ai",
  "password": "demo123",
  "name": "Demo User"
}
```

### New Users
- Can register with any email/password combination
- Password is not hashed (demo only)
- User is automatically created on login if not exists

## Security Considerations

### Current Implementation (Demo)
- No password hashing (plaintext storage)
- No rate limiting on auth endpoints
- No session management beyond JWT
- Demo credentials automatically created

### Production Requirements
- Implement password hashing (bcrypt/scrypt)
- Add rate limiting
- Implement proper session management
- Add email verification
- Implement CSRF protection
- Add audit logging

## Monitoring & Logging

### Backend Logs
```bash
# API server logs
./server 2>&1 | tee api.log
```

### Frontend Error Tracking
- Dashboard: Built-in error reporting via Next.js
- Landing Page: Console logging for debugging

## Troubleshooting

### Common Issues

1. **Build Errors**
   ```bash
   # Clear node_modules and rebuild
   rm -rf node_modules package-lock.json
   npm install
   npm run build
   ```

2. **CORS Issues**
   - Ensure NEXTAUTH_URL matches frontend URL
   - Check API_BASE environment variable

3. **Authentication Failures**
   - Verify backend is running on port 8080
   - Check network connectivity
   - Verify environment variables

### Debug Mode
```bash
# Enable debug logging
export DEBUG=true
./server
```

## API Reference

### Authentication Endpoints

#### POST /auth/login
```json
Request: {
  "email": "string",
  "password": "string"
}

Response: {
  "success": boolean,
  "token": "string",
  "user": {
    "id": "string",
    "name": "string", 
    "email": "string"
  }
}
```

#### POST /auth/register
```json
Request: {
  "email": "string",
  "password": "string",
  "name": "string"
}

Response: {
  "success": boolean,
  "token": "string", 
  "user": {
    "id": "string",
    "name": "string",
    "email": "string"
  }
}
```

## Support

For deployment issues:
1. Check logs in each service
2. Verify environment variables
3. Test connectivity between services
4. Check firewall settings

## Next Steps

1. **Enhanced Security**: Implement password hashing
2. **Email Notifications**: Send welcome emails
3. **Session Management**: Add proper session handling
4. **Analytics**: Track authentication events
5. **Admin Panel**: User management interface