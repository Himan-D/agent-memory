output "cloud_run_url" {
  description = "Cloud Run service URL"
  value       = module.gcp.cloud_run_url
}

output "redis_host" {
  description = "Redis instance host"
  value       = module.gcp.redis_host
  sensitive   = true
}

output "redis_port" {
  description = "Redis instance port"
  value       = module.gcp.redis_port
}

output "neo4j_internal_ip" {
  description = "Neo4j internal IP address"
  value       = module.gcp.neo4j_internal_ip
}

output "neo4j_bolt_uri" {
  description = "Neo4j Bolt URI for application config"
  value       = module.gcp.neo4j_bolt_uri
  sensitive   = true
}

output "qdrant_internal_ip" {
  description = "Qdrant internal IP address"
  value       = module.gcp.qdrant_internal_ip
}

output "qdrant_http_url" {
  description = "Qdrant HTTP URL"
  value       = module.gcp.qdrant_http_url
}

output "ne4j_password" {
  description = "Neo4j password (auto-generated)"
  value       = module.gcp.neo4j_password
  sensitive   = true
}

output "jwt_secret" {
  description = "JWT signing secret"
  value       = module.gcp.jwt_secret
  sensitive   = true
}

output "service_account_email" {
  description = "Cloud Run service account email"
  value       = module.gcp.service_account_email
}

output "vpc_name" {
  description = "VPC network name"
  value       = module.gcp.vpc_name
}

output "project_id" {
  description = "GCP project ID"
  value       = var.project_id
}
