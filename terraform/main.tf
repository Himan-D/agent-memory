terraform {
  required_version = ">= 1.6"

  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "~> 6.0"
    }
    google-beta = {
      source  = "hashicorp/google-beta"
      version = "~> 6.0"
    }
    random = {
      source  = "hashicorp/random"
      version = "~> 3.6"
    }
  }

  backend "gcs" {
    # Configure via backend.hcl or -backend-config
    # bucket = "hystersis-terraform-state"
    # prefix = "prod"
  }
}

provider "google" {
  project = var.project_id
  region  = var.region
}

provider "google-beta" {
  project = var.project_id
  region  = var.region
}

locals {
  container_image = var.container_image != "" ? var.container_image : "gcr.io/${var.project_id}/hystersis:latest"
}

module "gcp" {
  source = "./modules/gcp"

  project_id = var.project_id
  region     = var.region
  name       = var.name

  environment = var.environment

  # Network
  vpc_cidr           = var.vpc_cidr
  subnet_cidr        = var.subnet_cidr
  enable_private_api = var.enable_private_api

  # Cloud Run
  cloud_run_memory               = var.cloud_run_memory
  cloud_run_cpu                  = var.cloud_run_cpu
  cloud_run_min_instances        = var.cloud_run_min_instances
  cloud_run_max_instances        = var.cloud_run_max_instances
  cloud_run_container_concurrency = var.cloud_run_container_concurrency
  cloud_run_service_account_email = var.cloud_run_service_account_email

  # Redis
  redis_memory_size_gb    = var.redis_memory_size_gb
  redis_tier              = var.redis_tier
  redis_version           = var.redis_version

  # Neo4j
  neo4j_machine_type  = var.neo4j_machine_type
  neo4j_disk_size_gb  = var.neo4j_disk_size_gb
  neo4j_password      = var.neo4j_password
  neo4j_heap_initial  = var.neo4j_heap_initial
  neo4j_heap_max      = var.neo4j_heap_max

  # Qdrant
  qdrant_machine_type = var.qdrant_machine_type
  qdrant_disk_size_gb = var.qdrant_disk_size_gb

  # Image
  container_image = local.container_image

  # DNS
  domain_name = var.domain_name

  # Secrets
  llm_api_key = var.llm_api_key
  jwt_secret  = var.jwt_secret
  sentry_dsn  = var.sentry_dsn

  # Observability
  enable_otel_collector = var.enable_otel_collector
}
