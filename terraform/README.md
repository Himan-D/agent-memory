# Hystersis Terraform

Infrastructure as Code for deploying Hystersis to Google Cloud Platform.

```
                     ┌──────────────┐
                     │   Cloud Run   │
                     │  (Monolith)   │
                     └──────┬───────┘
                            │
              ┌─────────────┼─────────────┐
              │             │             │
              ▼             ▼             ▼
       ┌──────────┐  ┌──────────┐  ┌──────────┐
       │  Neo4j   │  │  Qdrant  │  │  Redis   │
       │ (GCE VM) │  │ (GCE VM) │  │ Memstore │
       └──────────┘  └──────────┘  └──────────┘
              │             │             │
              └─────────────┼─────────────┘
                            │
                     ┌──────┴──────┐
                     │    VPC      │
                     │   Network   │
                     └─────────────┘

  Internet → Cloud NAT (outbound only, for Docker pulls on VMs)
  Admin    → IAP Tunnel (SSH via gcloud compute ssh)
```

## Prerequisites

- Terraform >= 1.6
- Google Cloud project with billing enabled
- `gcloud` CLI installed and authenticated

## Quick Start (with gcloud)

```bash
# 1. Set your project
export PROJECT_ID="your-gcp-project-id"
gcloud config set project $PROJECT_ID

# 2. Create state bucket
gcloud storage buckets create gs://${PROJECT_ID}-tf-state \
  --location=us-central1

# 3. Enable required APIs
gcloud services enable \
  run.googleapis.com \
  compute.googleapis.com \
  redis.googleapis.com \
  secretmanager.googleapis.com \
  vpcaccess.googleapis.com \
  monitoring.googleapis.com \
  dns.googleapis.com

# 4. Authenticate Terraform with GCP
gcloud auth application-default login

# 5. Copy and edit config
cp backend.hcl.example backend.hcl
# Edit backend.hcl: set bucket and prefix
cp terraform.tfvars.example terraform.tfvars
# Edit terraform.tfvars: set project_id and secrets

# 6. Initialize
terraform init -backend-config=backend.hcl

# 7. Review plan
terraform plan

# 8. Deploy
terraform apply

# 9. Get outputs (connection strings, passwords)
terraform output -json
```

## Build the Container Image

Before `terraform apply`, build and push the Docker image:

```bash
gcloud builds submit \
  --config deploy/gcp/cloudbuild.yaml \
  --substitutions=_REGION=us-central1
```

Or build locally and push:

```bash
docker build -t gcr.io/$PROJECT_ID/hystersis:latest .
docker push gcr.io/$PROJECT_ID/hystersis:latest
```

## Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `project_id` | (required) | GCP project ID |
| `region` | `us-central1` | GCP region |
| `environment` | `dev` | dev/staging/prod |
| `container_image` | `gcr.io/{project_id}/hystersis:latest` | Backend image |
| `cloud_run_memory` | `1Gi` | Container memory |
| `cloud_run_cpu` | `2` | Container vCPUs |
| `neo4j_machine_type` | `e2-standard-4` | Neo4j VM type |
| `neo4j_disk_size_gb` | `100` | Neo4j data disk (SSD) |
| `qdrant_machine_type` | `e2-standard-2` | Qdrant VM type |
| `qdrant_disk_size_gb` | `50` | Qdrant data disk (SSD) |
| `redis_memory_size_gb` | `1` | Redis memory |
| `domain_name` | `""` | Custom domain (optional) |
| `llm_api_key` | `""` | LLM API key (Secret Manager) |
| `enable_otel_collector` | `false` | Deploy OTel collector |

## Outputs

| Output | Description | gcloud Equivalent |
|--------|-------------|-------------------|
| `cloud_run_url` | Public API URL | `gcloud run services describe hystersis-{env}` |
| `redis_host` | Redis instance host | `gcloud redis instances describe` |
| `neo4j_bolt_uri` | Neo4j connection URI | `gcloud compute instances describe` |
| `qdrant_http_url` | Qdrant HTTP URL | `gcloud compute instances describe` |
| `neo4j_password` | Auto-generated Neo4j password | Stored in Secret Manager |
| `jwt_secret` | Auto-generated JWT secret | Stored in Secret Manager |

## Environments

```bash
# Dev workspace
terraform workspace new dev || terraform workspace select dev
terraform apply -var-file=dev.tfvars

# Staging
terraform workspace new staging || terraform workspace select staging
terraform apply -var-file=staging.tfvars

# Production
terraform workspace new prod || terraform workspace select prod
terraform apply -var-file=prod.tfvars
```

## Security

- Neo4j and Qdrant have no public IPs — internet access via Cloud NAT only
- Cloud Run connects via Serverless VPC Access (private networking)
- Secrets stored in Google Secret Manager, referenced by Cloud Run at runtime
- Redis uses transit encryption + private service access
- SSH access via IAP tunnel only: `gcloud compute ssh --tunnel-through-iap`
- All VMs use OS Login for SSH key management

## Troubleshooting

```bash
# Check Terraform state
terraform state list

# Import existing resources
terraform import module.gcp.google_compute_network.vpc projects/$PROJECT_ID/global/networks/hystersis-dev-vpc

# SSH into Neo4j VM (via IAP)
gcloud compute ssh hystersis-dev-neo4j --zone us-central1-a --tunnel-through-iap

# Check startup script logs
sudo journalctl -u google-startup-scripts.service

# View Cloud Run logs
gcloud logging read "resource.type=cloud_run_revision AND resource.labels.service_name=hystersis-dev" --limit 20
```

## Updating

After code changes, rebuild and push the image, then update Terraform:

```bash
gcloud builds submit --config deploy/gcp/cloudbuild.yaml
terraform plan
terraform apply
```
