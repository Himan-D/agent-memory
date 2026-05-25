# Plan: Production-Ready Storage — GCP, Persistent Stores, Multi-Replica Safety

## Context

The system stores data across Neo4j (graph), Qdrant (vectors), and Redis (tiers/sync). But **5 critical data stores are in-memory only** — users, sessions, Stripe usage, audit events, and analytics counters all vanish on restart. In a multi-replica k8s deployment, each pod has divergent state. No GCP/AWS integration exists. This plan makes every store persistent and adds GCP as a first-class deployment target.

---

## Phase 1: Persist All In-Memory Stores to Neo4j

The fastest path to persistence uses Neo4j (already running, already connected). No new infrastructure needed.

### 1.1 Neo4j User Store
- **File**: `internal/users/neo4j_store.go` — already exists with some methods
- **Problem**: `NewService` uses `NewInMemoryStore()` — needs to use Neo4j store instead
- **Fix**: Read the existing `Neo4jStore`. Ensure it implements the full `Store` interface (ListUsers, GetUser, CreateUser, UpdateUser, DeleteUser, ListInvites, etc.). Wire it in `cmd/server/main.go` when Neo4j is available.
- **Seed admin**: Move the `seed()` logic to check Neo4j first — only seed if no admin exists.

### 1.2 Redis Session Store
- **File**: `cmd/server/session.go` — currently in-memory `map[string]*Session`
- **New file**: `cmd/server/session_redis.go`
- **Fix**: Create a `RedisSessionStore` that stores sessions as JSON in Redis with TTL (24h). Falls back to in-memory if Redis unavailable.
- Redis keys: `session:<token>` → JSON session, `user_session:<userID>` → token (for one-session-per-user enforcement)

### 1.3 Stripe Usage to Neo4j
- **File**: `internal/stripe/service.go`
- **Fix**: Replace `usageMap map[string]*UsageRecord` with Neo4j-backed persistence. On `RecordUsage`, update a `(:UsageRecord)` node. On `CheckQuota`, read from Neo4j. Cache in memory with 60s TTL to avoid per-request queries.

### 1.4 Audit Logger to Neo4j
- **File**: `internal/audit/logger.go`
- **Fix**: Wire `Neo4jAuditStorage` (if it exists) or create one implementing the `Storage` interface. Store audit events as `(:AuditEvent)` nodes. Wire in `cmd/server/main.go`.

### 1.5 Analytics to Redis
- **File**: `internal/analytics/service.go`
- **Fix**: Replace atomic counters with Redis INCR commands. `RecordSearch` → `INCR analytics:search_count`. `GetDashboard` reads from Redis. Atomic, shared across replicas, persists across restarts.

---

## Phase 2: Add GCP Config + Storage

### 2.1 GCP Config in config.go
- **File**: `internal/config/config.go`
- **Add**:
```go
type GCPConfig struct {
    ProjectID      string `env:"GCP_PROJECT_ID" envDefault:""`
    Region         string `env:"GCP_REGION" envDefault:"us-central1"`
    CredentialsFile string `env:"GOOGLE_APPLICATION_CREDENTIALS" envDefault:""`
    
    // Cloud Storage
    BucketName     string `env:"GCS_BUCKET" envDefault:""`
    
    // Cloud SQL (if using managed Postgres instead of Neo4j)
    CloudSQLInstance string `env:"CLOUD_SQL_INSTANCE" envDefault:""`
    
    // Pub/Sub (alternative to Redis for multi-agent sync)
    PubSubTopic    string `env:"GCP_PUBSUB_TOPIC" envDefault:"hystersis-events"`
    
    // Secret Manager
    UseSecretManager bool `env:"GCP_USE_SECRET_MANAGER" envDefault:"false"`
}
```
- Add `GCP GCPConfig` to the main `Config` struct

### 2.2 GCS Backup Storage
- **New file**: `internal/storage/gcs.go`
- **What**: Upload backup exports to Google Cloud Storage instead of local filesystem
- Interface: `BlobStore` with `Upload(ctx, key, data)`, `Download(ctx, key)`, `List(ctx, prefix)`, `Delete(ctx, key)`
- Uses `cloud.google.com/go/storage` SDK
- Wired into backup scheduler as alternative to local file writes

### 2.3 GCP Pub/Sub for Multi-Agent Sync
- **New file**: `internal/memory/sync/pubsub.go`
- **What**: Alternative to Redis pub/sub for multi-agent memory sharing
- Same interface as `redis.go` but uses GCP Pub/Sub
- Better for serverless/Cloud Run deployments where Redis isn't available

### 2.4 GCP Secret Manager Integration
- **New file**: `internal/config/secrets.go`
- **What**: Load secrets from GCP Secret Manager instead of env vars
- When `GCP_USE_SECRET_MANAGER=true`, fetch `NEO4J_PASSWORD`, `QDRANT_API_KEY`, `LLM_API_KEY`, `JWT_SECRET`, `STRIPE_SECRET_KEY` from Secret Manager
- Reduces secret exposure in k8s ConfigMaps

---

## Phase 3: Multi-Cloud Storage Abstraction

### 3.1 Blob Store Interface
- **New file**: `internal/storage/store.go`
```go
type BlobStore interface {
    Upload(ctx context.Context, key string, data []byte) error
    Download(ctx context.Context, key string) ([]byte, error)
    List(ctx context.Context, prefix string) ([]string, error)
    Delete(ctx context.Context, key string) error
}
```

### 3.2 S3 Implementation
- **New file**: `internal/storage/s3.go`
- Uses AWS SDK v2 (`github.com/aws/aws-sdk-go-v2`)
- Configured via `AWS_REGION`, `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`, `S3_BUCKET`

### 3.3 Local Filesystem Implementation
- **New file**: `internal/storage/local.go`
- Writes to `./data/` directory
- Default when no cloud config is set

### 3.4 Add AWS Config
- **File**: `internal/config/config.go`
```go
type AWSConfig struct {
    Region          string `env:"AWS_REGION" envDefault:"us-east-1"`
    S3Bucket        string `env:"S3_BUCKET" envDefault:""`
    AccessKeyID     string `env:"AWS_ACCESS_KEY_ID" envDefault:""`
    SecretAccessKey  string `env:"AWS_SECRET_ACCESS_KEY" envDefault:""`
    UseSecretsManager bool  `env:"AWS_USE_SECRETS_MANAGER" envDefault:"false"`
}
```

---

## Phase 4: Production Deployment Configs

### 4.1 GCP Cloud Run Deployment
- **New file**: `deploy/gcp/cloudbuild.yaml` — Cloud Build config
- **New file**: `deploy/gcp/cloudrun.yaml` — Cloud Run service definition
- Connects to: Cloud SQL (Neo4j Aura or Memgraph), Qdrant Cloud, Redis (Memorystore)

### 4.2 Update docker-compose for production
- **File**: `docker-compose.yml`
- Add volume mounts for `./data/` directory (analytics, audit, backups)
- Add OpenSearch service (currently missing despite being in config)
- Add environment variables for all new GCP/AWS config

### 4.3 Update Helm chart
- **File**: `deploy/helm/agent-memory/values.yaml`
- Add GCP config values
- Add Redis URL
- Add persistence volume claims
- Add Sentry DSN
- Add proper secrets via k8s Secrets (not ConfigMap)

### 4.4 Update k8s manifest
- **File**: `deploy/k8s/agent-memory.yaml`
- Add Redis service or connection
- Move secrets from ConfigMap to k8s Secrets
- Add PVC for data directory
- Add readiness/liveness probes that check Neo4j + Qdrant + Redis

---

## Phase 5: Fix Remaining Production Gaps

### 5.1 Fix Qdrant collection name mismatch
- `config.go` defaults `QDRANT_COLLECTION` to `agent_memory`
- `qdrant/client.go` hardcodes `agent_long_term_memory`
- **Fix**: Make client read from config

### 5.2 Add connection health checks
- **File**: `internal/memory/service.go` — `HealthCheck()` method
- Check Neo4j, Qdrant, Redis connectivity
- Return per-store status for `/health` endpoint

### 5.3 Add database migration versioning
- **New file**: `internal/migration/migration.go`
- Track schema version in Neo4j (`:SchemaVersion` node)
- Run migrations on startup (idempotent, version-gated)

---

## Implementation Order

```
Phase 1 (Persist in-memory stores — highest priority):
  ├── 1.1: Neo4j user store [users/neo4j_store.go + main.go]
  ├── 1.2: Redis session store [session_redis.go]
  ├── 1.3: Stripe usage to Neo4j [stripe/service.go]
  ├── 1.4: Audit to Neo4j [audit/logger.go]
  └── 1.5: Analytics to Redis [analytics/service.go]

Phase 2 (GCP integration):
  ├── 2.1: GCP config [config/config.go]
  ├── 2.2: GCS backup storage [storage/gcs.go]
  ├── 2.3: Pub/Sub sync [sync/pubsub.go]
  └── 2.4: Secret Manager [config/secrets.go]

Phase 3 (Multi-cloud abstraction):
  ├── 3.1: BlobStore interface [storage/store.go]
  ├── 3.2: S3 implementation [storage/s3.go]
  ├── 3.3: Local filesystem [storage/local.go]
  └── 3.4: AWS config [config/config.go]

Phase 4 (Deployment):
  ├── 4.1: Cloud Run config [deploy/gcp/]
  ├── 4.2: docker-compose production [docker-compose.yml]
  ├── 4.3: Helm values [deploy/helm/]
  └── 4.4: k8s manifest [deploy/k8s/]

Phase 5 (Polish):
  ├── 5.1: Qdrant collection name fix
  ├── 5.2: Health checks per store
  └── 5.3: Migration versioning
```

## Critical Files

| File | Changes |
|---|---|
| `internal/config/config.go` | Add GCPConfig, AWSConfig, StorageConfig |
| `internal/users/neo4j_store.go` | Complete Store interface implementation |
| `cmd/server/main.go` | Wire Neo4j user store, Redis session store, audit backend |
| `cmd/server/session.go` | Add RedisSessionStore |
| `internal/stripe/service.go` | Neo4j-backed usage persistence |
| `internal/audit/logger.go` | Wire Neo4j backend |
| `internal/analytics/service.go` | Redis-backed counters |
| `internal/storage/store.go` | BlobStore interface (NEW) |
| `internal/storage/gcs.go` | GCS implementation (NEW) |
| `internal/storage/s3.go` | S3 implementation (NEW) |
| `internal/storage/local.go` | Local filesystem (NEW) |
| `internal/memory/sync/pubsub.go` | GCP Pub/Sub sync (NEW) |
| `internal/config/secrets.go` | Secret Manager loader (NEW) |
| `deploy/gcp/cloudrun.yaml` | Cloud Run deployment (NEW) |
| `deploy/gcp/cloudbuild.yaml` | Cloud Build CI (NEW) |

## Verification

```bash
# Build
go build ./... && go vet ./...

# Test with local infra
docker-compose up -d
go test ./internal/users/... -v
go test ./internal/storage/... -v

# Health check
curl http://localhost:8080/health
# Should return per-store status: neo4j=ok, qdrant=ok, redis=ok
```
