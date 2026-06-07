variable "project_id" {
  description = "GCP project ID"
  type        = string
}

variable "region" {
  description = "GCP region"
  type        = string
}

variable "name" {
  description = "Resource name prefix"
  type        = string
}

variable "environment" {
  description = "Deployment environment"
  type        = string
}

# Network
variable "vpc_cidr" {
  description = "VPC CIDR range"
  type        = string
}

variable "subnet_cidr" {
  description = "Subnet CIDR range"
  type        = string
}

variable "enable_private_api" {
  description = "Restrict API to private VPC"
  type        = bool
}

# Cloud Run
variable "cloud_run_memory" {
  description = "Cloud Run container memory"
  type        = string
}

variable "cloud_run_cpu" {
  description = "Cloud Run container CPU count"
  type        = number
}

variable "cloud_run_min_instances" {
  description = "Cloud Run minimum instances"
  type        = number
}

variable "cloud_run_max_instances" {
  description = "Cloud Run maximum instances"
  type        = number
}

variable "cloud_run_container_concurrency" {
  description = "Cloud Run max concurrent requests"
  type        = number
}

variable "cloud_run_service_account_email" {
  description = "Cloud Run service account email"
  type        = string
}

# Redis
variable "redis_memory_size_gb" {
  description = "Redis memory size in GB"
  type        = number
}

variable "redis_tier" {
  description = "Redis tier"
  type        = string
}

variable "redis_version" {
  description = "Redis version"
  type        = string
}

# Neo4j
variable "neo4j_machine_type" {
  description = "Neo4j machine type"
  type        = string
}

variable "neo4j_disk_size_gb" {
  description = "Neo4j disk size in GB"
  type        = number
}

variable "neo4j_password" {
  description = "Neo4j password"
  type        = string
}

variable "neo4j_heap_initial" {
  description = "Neo4j initial heap"
  type        = string
}

variable "neo4j_heap_max" {
  description = "Neo4j max heap"
  type        = string
}

# Qdrant
variable "qdrant_machine_type" {
  description = "Qdrant machine type"
  type        = string
}

variable "qdrant_disk_size_gb" {
  description = "Qdrant disk size in GB"
  type        = number
}

# Container
variable "container_image" {
  description = "Container image"
  type        = string
}

# DNS
variable "domain_name" {
  description = "Custom domain"
  type        = string
}

# Secrets
variable "llm_api_key" {
  description = "LLM API key"
  type        = string
}

variable "jwt_secret" {
  description = "JWT secret"
  type        = string
}

variable "sentry_dsn" {
  description = "Sentry DSN"
  type        = string
}

# Observability
variable "enable_otel_collector" {
  description = "Enable OTel Collector"
  type        = bool
}
