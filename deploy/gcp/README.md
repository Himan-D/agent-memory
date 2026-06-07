# GCP Deployment

Two deployment options exist:

## Option 1: Quick Deploy (Cloud Run + external services)

Uses existing managed services (Neo4j Aura, Qdrant Cloud, Redis Memorystore).

```bash
# Create secrets
gcloud secrets create neo4j-uri --data-file=- <<< "bolt://your-neo4j:7687"
gcloud secrets create neo4j-password --data-file=- <<< "your-password"
gcloud secrets create qdrant-url --data-file=- <<< "https://your-qdrant:6333"
gcloud secrets create qdrant-api-key --data-file=- <<< "your-key"
gcloud secrets create llm-api-key --data-file=- <<< "your-openai-key"
gcloud secrets create redis-url --data-file=- <<< "redis://your-redis:6379"

# Deploy
gcloud builds submit --config deploy/gcp/cloudbuild.yaml
```

## Option 2: Full Infrastructure (Terraform)

Provisions everything: VPC, Cloud Run, Neo4j VM, Qdrant VM, Redis, Secret Manager.

```bash
# See terraform/README.md for full instructions
cd terraform
gcloud auth application-default login
terraform init -backend-config=backend.hcl
terraform plan
terraform apply
```

## Architecture

### Option 1 (Cloud Build + Cloud Run)
```
Cloud Run (hystersis) → Neo4j Aura (graph)
                      → Qdrant Cloud (vectors)
                      → Memorystore Redis (sessions, tiers, sync)
                      → Cloud Storage (backups)
                      → Secret Manager (credentials)
```

### Option 2 (Terraform)
```
Cloud Run (hystersis) → Neo4j (Compute Engine, no public IP)
                      → Qdrant (Compute Engine, no public IP)
                      → Memorystore Redis (private service access)
                      → Secret Manager (credentials)
                      → Cloud NAT (outbound internet for VMs)
```

## Container Image

Both options use the same Dockerfile:

```bash
# Via Cloud Build (CI)
gcloud builds submit --config deploy/gcp/cloudbuild.yaml

# Manual
docker build -t gcr.io/$PROJECT_ID/hystersis:latest -f docker/Dockerfile .
docker push gcr.io/$PROJECT_ID/hystersis:latest
```
