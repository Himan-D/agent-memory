variable "project_id" {
  description = "GCP project ID"
  type        = string
}

variable "region" {
  description = "GCP region"
  type        = string
  default     = "us-central1"
}

variable "name" {
  description = "Resource name prefix"
  type        = string
  default     = "hystersis"
}

variable "environment" {
  description = "Deployment environment (dev, staging, prod)"
  type        = string
  default     = "dev"

  validation {
    condition     = contains(["dev", "staging", "prod"], var.environment)
    error_message = "Must be dev, staging, or prod."
  }
}

variable "container_image" {
  description = "Container image for the monolith backend (built by Cloud Build)"
  type        = string
  default     = ""
}

# Network
variable "vpc_cidr" {
  description = "VPC CIDR range"
  type        = string
  default     = "10.0.0.0/16"
}

variable "subnet_cidr" {
  description = "Subnet CIDR range"
  type        = string
  default     = "10.0.1.0/24"
}

variable "enable_private_api" {
  description = "Restrict API to private VPC (no public internet)"
  type        = bool
  default     = false
}

# Cloud Run
variable "cloud_run_memory" {
  description = "Cloud Run container memory"
  type        = string
  default     = "1Gi"
}

variable "cloud_run_cpu" {
  description = "Cloud Run container CPU count"
  type        = number
  default     = 2
}

variable "cloud_run_min_instances" {
  description = "Cloud Run minimum instances"
  type        = number
  default     = 0
}

variable "cloud_run_max_instances" {
  description = "Cloud Run maximum instances"
  type        = number
  default     = 10
}

variable "cloud_run_container_concurrency" {
  description = "Cloud Run max concurrent requests per instance"
  type        = number
  default     = 80
}

variable "cloud_run_service_account_email" {
  description = "Service account email for Cloud Run (empty = default compute)"
  type        = string
  default     = ""
}

# Redis
variable "redis_memory_size_gb" {
  description = "Redis instance memory size in GB"
  type        = number
  default     = 1
}

variable "redis_tier" {
  description = "Redis tier: STANDARD_HA or BASIC"
  type        = string
  default     = "STANDARD_HA"
}

variable "redis_version" {
  description = "Redis version"
  type        = string
  default     = "REDIS_7_0"
}

# Neo4j (Compute Engine self-hosted)
variable "neo4j_machine_type" {
  description = "Neo4j Compute Engine machine type"
  type        = string
  default     = "e2-standard-4"
}

variable "neo4j_disk_size_gb" {
  description = "Neo4j persistent disk size in GB"
  type        = number
  default     = 100
}

variable "neo4j_password" {
  description = "Neo4j password (auto-generated if empty)"
  type        = string
  default     = ""
  sensitive   = true
}

variable "neo4j_heap_initial" {
  description = "Neo4j initial heap size"
  type        = string
  default     = "512m"
}

variable "neo4j_heap_max" {
  description = "Neo4j max heap size"
  type        = string
  default     = "2G"
}

# Qdrant (Compute Engine self-hosted)
variable "qdrant_machine_type" {
  description = "Qdrant Compute Engine machine type"
  type        = string
  default     = "e2-standard-2"
}

variable "qdrant_disk_size_gb" {
  description = "Qdrant persistent disk size in GB"
  type        = number
  default     = 50
}

# DNS
variable "domain_name" {
  description = "Custom domain for the API (optional)"
  type        = string
  default     = ""
}

# Secrets
variable "llm_api_key" {
  description = "LLM provider API key"
  type        = string
  default     = ""
  sensitive   = true
}

variable "jwt_secret" {
  description = "JWT signing secret (auto-generated if empty)"
  type        = string
  default     = ""
  sensitive   = true
}

variable "sentry_dsn" {
  description = "Sentry DSN for error tracking"
  type        = string
  default     = ""
  sensitive   = true
}

# Observability
variable "enable_otel_collector" {
  description = "Deploy OpenTelemetry Collector sidecar"
  type        = bool
  default     = false
}
