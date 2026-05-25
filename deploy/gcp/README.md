# GCP Deployment

## Prerequisites
- GCP Project with billing enabled
- Cloud Run API enabled
- Cloud Build API enabled
- Neo4j Aura or self-managed Neo4j instance
- Qdrant Cloud or self-managed Qdrant instance
- Memorystore for Redis

## Setup

1. Create secrets:
```bash
gcloud secrets create neo4j-uri --data-file=- <<< "bolt://your-neo4j:7687"
gcloud secrets create neo4j-password --data-file=- <<< "your-password"
gcloud secrets create qdrant-url --data-file=- <<< "https://your-qdrant:6333"
gcloud secrets create qdrant-api-key --data-file=- <<< "your-key"
gcloud secrets create llm-api-key --data-file=- <<< "your-openai-key"
gcloud secrets create redis-url --data-file=- <<< "redis://your-redis:6379"
```

2. Deploy:
```bash
gcloud builds submit --config deploy/gcp/cloudbuild.yaml
```

3. Or deploy directly:
```bash
gcloud run deploy hystersis \
  --source . \
  --region us-central1 \
  --port 8080 \
  --memory 1Gi
```

## Architecture
```
Cloud Run (hystersis) → Neo4j Aura (graph)
                      → Qdrant Cloud (vectors)
                      → Memorystore Redis (sessions, tiers, sync)
                      → Cloud Storage (backups)
                      → Secret Manager (credentials)
```
