output "cloud_run_url" {
  description = "Cloud Run service URL"
  value       = google_cloud_run_v2_service.main.uri
}

output "redis_host" {
  description = "Redis instance host"
  value       = google_redis_instance.cache.host
}

output "redis_port" {
  description = "Redis instance port"
  value       = google_redis_instance.cache.port
}

output "neo4j_internal_ip" {
  description = "Neo4j internal IP"
  value       = google_compute_instance.neo4j.network_interface[0].network_ip
}

output "neo4j_bolt_uri" {
  description = "Neo4j Bolt URI"
  value       = "bolt://${google_compute_instance.neo4j.network_interface[0].network_ip}:7687"
}

output "neo4j_password" {
  description = "Neo4j password"
  value       = random_password.neo4j.result
}

output "qdrant_internal_ip" {
  description = "Qdrant internal IP"
  value       = google_compute_instance.qdrant.network_interface[0].network_ip
}

output "qdrant_http_url" {
  description = "Qdrant HTTP URL"
  value       = "http://${google_compute_instance.qdrant.network_interface[0].network_ip}:6333"
}

output "jwt_secret" {
  description = "JWT signing secret"
  value       = random_password.jwt.result
}

output "service_account_email" {
  description = "Cloud Run service account"
  value       = google_service_account.main.email
}

output "vpc_name" {
  description = "VPC network name"
  value       = google_compute_network.vpc.name
}

output "subnet_name" {
  description = "Subnet name"
  value       = google_compute_subnet.main.name
}
