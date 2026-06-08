locals {
  name_prefix  = "${var.name}-${var.environment}"
  common_tags = {
    Name        = "${var.name}-${var.environment}"
    Environment = var.environment
    ManagedBy   = "terraform"
    Service     = "hystersis"
  }
}

# ==============================================================================
# VPC Network
# ==============================================================================
resource "google_compute_network" "vpc" {
  name                    = "${local.name_prefix}-vpc"
  auto_create_subnetworks = false
  routing_mode            = "REGIONAL"
}

resource "google_compute_subnet" "main" {
  name          = "${local.name_prefix}-subnet"
  ip_cidr_range = var.subnet_cidr
  region        = var.region
  network       = google_compute_network.vpc.id

  private_ip_google_access = true
}

# Allow internal traffic
resource "google_compute_firewall" "allow_internal" {
  name    = "${local.name_prefix}-allow-internal"
  network = google_compute_network.vpc.name

  allow {
    protocol = "tcp"
    ports    = ["0-65535"]
  }
  allow {
    protocol = "udp"
    ports    = ["0-65535"]
  }
  allow {
    protocol = "icmp"
  }

  source_ranges = [var.vpc_cidr]
}

# Allow health check ranges
resource "google_compute_firewall" "allow_health_checks" {
  name    = "${local.name_prefix}-allow-health-checks"
  network = google_compute_network.vpc.name

  allow {
    protocol = "tcp"
    ports    = ["7687", "6333", "7474"]
  }

  source_ranges = [
    "35.191.0.0/16",
    "130.211.0.0/22",
    "209.85.152.0/22",
    "209.85.204.0/22",
  ]
}

# Allow Cloud Run to reach internal services via VPC
resource "google_compute_firewall" "allow_cloud_run" {
  name    = "${local.name_prefix}-allow-cloud-run"
  network = google_compute_network.vpc.name

  allow {
    protocol = "tcp"
    ports    = ["7687", "6333", "6379", "7474"]
  }

  source_ranges = [var.subnet_cidr]
}

# Allow SSH (restricted to IAP)
resource "google_compute_firewall" "allow_iap_ssh" {
  name    = "${local.name_prefix}-allow-iap-ssh"
  network = google_compute_network.vpc.name

  allow {
    protocol = "tcp"
    ports    = ["22"]
  }

  source_ranges = ["35.235.240.0/20"]
}

# Cloud NAT for outbound internet (Docker pulls) without public IPs
resource "google_compute_router" "nat" {
  name    = "${local.name_prefix}-nat-router"
  region  = var.region
  network = google_compute_network.vpc.id
}

resource "google_compute_router_nat" "main" {
  name                               = "${local.name_prefix}-nat"
  router                             = google_compute_router.nat.name
  region                             = var.region
  nat_ip_allocate_option             = "AUTO_ONLY"
  source_subnetwork_ip_ranges_to_nat = "ALL_SUBNETWORKS_ALL_IP_RANGERS"

  log_config {
    enable = true
    filter = "ERRORS_ONLY"
  }
}

# ==============================================================================
# Serverless VPC Access (for Cloud Run to reach VPC resources)
# ==============================================================================
resource "google_vpc_access_connector" "main" {
  name          = "${local.name_prefix}-vpc-connector"
  region        = var.region
  network       = google_compute_network.vpc.name
  ip_cidr_range = "10.8.0.0/28"
  min_throughput = 200
  max_throughput = 1000
}

# ==============================================================================
# Service Account
# ==============================================================================
resource "google_service_account" "main" {
  account_id   = "${local.name_prefix}-sa"
  display_name = "Hystersis ${var.environment} service account"
}

resource "google_project_iam_member" "cloud_run_invoker" {
  project = var.project_id
  role    = "roles/run.invoker"
  member  = "serviceAccount:${google_service_account.main.email}"
}

resource "google_project_iam_member" "secret_manager_accessor" {
  project = var.project_id
  role    = "roles/secretmanager.secretAccessor"
  member  = "serviceAccount:${google_service_account.main.email}"
}

resource "google_project_iam_member" "cloud_trace_agent" {
  project = var.project_id
  role    = "roles/cloudtrace.agent"
  member  = "serviceAccount:${google_service_account.main.email}"
}

resource "google_project_iam_member" "monitoring_metric_writer" {
  project = var.project_id
  role    = "roles/monitoring.metricWriter"
  member  = "serviceAccount:${google_service_account.main.email}"
}

resource "google_project_iam_member" "logging_log_writer" {
  project = var.project_id
  role    = "roles/logging.logWriter"
  member  = "serviceAccount:${google_service_account.main.email}"
}

# ==============================================================================
# Cloud Run Service (Monolith Backend)
# ==============================================================================
resource "google_cloud_run_v2_service" "main" {
  name     = local.name_prefix
  location = var.region
  client   = "terraform"

  template {
    service_account = google_service_account.main.email
    timeout         = "300s"
    max_instance_request_concurrency = var.cloud_run_container_concurrency

    scaling {
      min_instance_count = var.cloud_run_min_instances
      max_instance_count = var.cloud_run_max_instances
    }

    containers {
      image = var.container_image

      resources {
        limits = {
          cpu    = "${var.cloud_run_cpu}"
          memory = var.cloud_run_memory
        }
      }

      ports {
        container_port = 8080
      }

      env {
        name  = "ENVIRONMENT"
        value = var.environment
      }
      env {
        name  = "NEO4J_URI"
        value = "bolt://${google_compute_instance.neo4j.network_interface[0].network_ip}:7687"
      }
      env {
        name  = "QDRANT_URL"
        value = "http://${google_compute_instance.qdrant.network_interface[0].network_ip}:6333"
      }
      env {
        name  = "REDIS_URL"
        value = "redis://${google_redis_instance.cache.host}:${google_redis_instance.cache.port}"
      }
      env {
        name  = "COMPRESSION_ENABLED"
        value = "true"
      }
      env {
        name  = "COMPRESSION_MODE"
        value = "extract"
      }
      env {
        name  = "MULTI_SIGNAL_ENABLED"
        value = "true"
      }
      env {
        name  = "AUTH_ENABLED"
        value = "true"
      }
      env {
        name  = "API_BASE_URL"
        value = "https://api.${var.domain_name}"
      }
      env {
        name  = "ALLOWED_ORIGINS"
        value = "https://${var.domain_name},https://www.${var.domain_name},https://app.${var.domain_name}"
      }
      env {
        name  = "STORAGE_PROVIDER"
        value = "gcs"
      }
      env {
        name  = "DATA_DIR"
        value = "/app/data"
      }

      # Secrets from Secret Manager
      env {
        name = "NEO4J_PASSWORD"
        value_source {
          secret_key_ref {
            secret  = google_secret_manager_secret.neo4j_password.secret_id
            version = "latest"
          }
        }
      }
      env {
        name = "LLM_API_KEY"
        value_source {
          secret_key_ref {
            secret  = google_secret_manager_secret.llm_api_key.secret_id
            version = "latest"
          }
        }
      }
      env {
        name = "JWT_SECRET"
        value_source {
          secret_key_ref {
            secret  = google_secret_manager_secret.jwt_secret.secret_id
            version = "latest"
          }
        }
      }
      env {
        name = "SENTRY_DSN"
        value_source {
          secret_key_ref {
            secret  = google_secret_manager_secret.sentry_dsn.secret_id
            version = "latest"
          }
        }
      }

      startup_probe {
        initial_delay_seconds = 10
        timeout_seconds       = 5
        period_seconds        = 10
        failure_threshold     = 3
        tcp_socket {
          port = 8080
        }
      }

      liveness_probe {
        initial_delay_seconds = 30
        timeout_seconds       = 5
        period_seconds        = 15
        failure_threshold     = 3
        http_get {
          path = "/health"
          port = 8080
        }
      }
    }

    vpc_access {
      connector = google_vpc_access_connector.main.id
      egress    = "PRIVATE_RANGES_ONLY"
    }
  }

  lifecycle {
    prevent_destroy = false
  }

  depends_on = [
    google_vpc_access_connector.main,
    google_secret_manager_secret_iam_member.neo4j_password,
    google_secret_manager_secret_iam_member.llm_api_key,
    google_secret_manager_secret_iam_member.jwt_secret,
    google_secret_manager_secret_iam_member.sentry_dsn,
  ]
}

# ==============================================================================
# Cloud Run IAM (Allow public unauthenticated access)
# ==============================================================================
resource "google_cloud_run_v2_service_iam_member" "public_access" {
  count    = var.enable_private_api ? 0 : 1
  name     = google_cloud_run_v2_service.main.name
  location = google_cloud_run_v2_service.main.location
  role     = "roles/run.invoker"
  member   = "allUsers"
}

# ==============================================================================
# Redis (Memorystore)
# ==============================================================================
resource "google_redis_instance" "cache" {
  name           = "${local.name_prefix}-redis"
  region         = var.region
  memory_size_gb = var.redis_memory_size_gb
  tier           = var.redis_tier
  redis_version  = var.redis_version

  connect_mode            = "PRIVATE_SERVICE_ACCESS"
  authorized_network      = google_compute_network.vpc.id
  transit_encryption_mode = "SERVER_AUTHENTICATION"
  replica_count           = var.redis_tier == "STANDARD_HA" ? 1 : 0

  display_name = "Hystersis ${var.environment} Redis"
  labels       = local.common_tags
}

# ==============================================================================
# Neo4j (Compute Engine)
# ==============================================================================
resource "google_compute_disk" "neo4j_data" {
  name  = "${local.name_prefix}-neo4j-data"
  type  = "pd-ssd"
  zone  = "${var.region}-a"
  size  = var.neo4j_disk_size_gb
  labels = local.common_tags
}

resource "google_compute_address" "neo4j_internal" {
  name         = "${local.name_prefix}-neo4j-ip"
  subnetwork   = google_compute_subnet.main.id
  address_type = "INTERNAL"
  region       = var.region
}

resource "google_compute_instance" "neo4j" {
  name         = "${local.name_prefix}-neo4j"
  machine_type = var.neo4j_machine_type
  zone         = "${var.region}-a"

  tags = ["neo4j", local.name_prefix]

  boot_disk {
    initialize_params {
      image = "ubuntu-os-cloud/ubuntu-2204-lts"
      size  = 20
      type  = "pd-standard"
    }
  }

  attached_disk {
    source      = google_compute_disk.neo4j_data.id
    device_name = "neo4j-data"
  }

  network_interface {
    network    = google_compute_network.vpc.id
    subnetwork = google_compute_subnet.main.id
    network_ip = google_compute_address.neo4j_internal.address
    # No public IP - internet access via Cloud NAT for Docker pulls
  }

  service_account {
    email  = google_service_account.main.email
    scopes = ["cloud-platform"]
  }

  metadata = {
    neo4j_password   = random_password.neo4j.result
    neo4j_heap_init  = var.neo4j_heap_initial
    neo4j_heap_max   = var.neo4j_heap_max
    enable-oslogin   = "TRUE"
  }

  metadata_startup_script = <<-EOF
    #!/bin/bash
    set -euo pipefail

    # Mount data disk
    if ! mountpoint -q /data; then
      mkfs.ext4 -F /dev/disk/by-id/google-neo4j-data || true
      mkdir -p /data
      mount /dev/disk/by-id/google-neo4j-data /data
      echo "/dev/disk/by-id/google-neo4j-data /data ext4 defaults 0 0" >> /etc/fstab
    fi

    # Install Docker (internet via Cloud NAT)
    if ! command -v docker &>/dev/null; then
      apt-get update -qq
      apt-get install -y -qq ca-certificates curl
      install -m 0755 -d /etc/apt/keyrings
      curl -fsSL https://download.docker.com/linux/ubuntu/gpg -o /etc/apt/keyrings/docker.asc
      echo "deb [arch=amd64 signed-by=/etc/apt/keyrings/docker.asc] https://download.docker.com/linux/ubuntu jammy stable" > /etc/apt/sources.list.d/docker.list
      apt-get update -qq
      apt-get install -y -qq docker-ce docker-ce-cli containerd.io
      systemctl enable docker
    fi

    # Run Neo4j
    PASSWORD=$(curl -H "Metadata-Flavor: Google" http://metadata.google.internal/computeMetadata/v1/instance/attributes/neo4j_password)
    HEAP_INIT=$(curl -H "Metadata-Flavor: Google" http://metadata.google.internal/computeMetadata/v1/instance/attributes/neo4j_heap_init)
    HEAP_MAX=$(curl -H "Metadata-Flavor: Google" http://metadata.google.internal/computeMetadata/v1/instance/attributes/neo4j_heap_max)

    docker rm -f neo4j 2>/dev/null || true
    docker run -d \
      --name neo4j \
      --restart unless-stopped \
      --network host \
      -v /data/neo4j:/data \
      -v /data/neo4j/logs:/logs \
      -e NEO4J_AUTH=neo4j/$PASSWORD \
      -e NEO4J_PLUGINS='["apoc"]' \
      -e NEO4J_server_memory_heap_initial__size=$HEAP_INIT \
      -e NEO4J_server_memory_heap_max__size=$HEAP_MAX \
      -e NEO4J_server_bolt_listen__address=0.0.0.0:7687 \
      neo4j:5.14-community
  EOF

  labels = local.common_tags

  depends_on = [
    google_compute_disk.neo4j_data,
    google_compute_address.neo4j_internal,
    google_compute_router_nat.main,
  ]
}

# ==============================================================================
# Qdrant (Compute Engine)
# ==============================================================================
resource "google_compute_disk" "qdrant_data" {
  name  = "${local.name_prefix}-qdrant-data"
  type  = "pd-ssd"
  zone  = "${var.region}-a"
  size  = var.qdrant_disk_size_gb
  labels = local.common_tags
}

resource "google_compute_address" "qdrant_internal" {
  name         = "${local.name_prefix}-qdrant-ip"
  subnetwork   = google_compute_subnet.main.id
  address_type = "INTERNAL"
  region       = var.region
}

resource "google_compute_instance" "qdrant" {
  name         = "${local.name_prefix}-qdrant"
  machine_type = var.qdrant_machine_type
  zone         = "${var.region}-a"

  tags = ["qdrant", local.name_prefix]

  boot_disk {
    initialize_params {
      image = "ubuntu-os-cloud/ubuntu-2204-lts"
      size  = 20
      type  = "pd-standard"
    }
  }

  attached_disk {
    source      = google_compute_disk.qdrant_data.id
    device_name = "qdrant-data"
  }

  network_interface {
    network    = google_compute_network.vpc.id
    subnetwork = google_compute_subnet.main.id
    network_ip = google_compute_address.qdrant_internal.address
    # No public IP - internet access via Cloud NAT for Docker pulls
  }

  service_account {
    email  = google_service_account.main.email
    scopes = ["cloud-platform"]
  }

  metadata = {
    enable-oslogin = "TRUE"
  }

  metadata_startup_script = <<-EOF
    #!/bin/bash
    set -euo pipefail

    # Mount data disk
    if ! mountpoint -q /data; then
      mkfs.ext4 -F /dev/disk/by-id/google-qdrant-data || true
      mkdir -p /data
      mount /dev/disk/by-id/google-qdrant-data /data
      echo "/dev/disk/by-id/google-qdrant-data /data ext4 defaults 0 0" >> /etc/fstab
    fi

    # Install Docker (internet via Cloud NAT)
    if ! command -v docker &>/dev/null; then
      apt-get update -qq
      apt-get install -y -qq ca-certificates curl
      install -m 0755 -d /etc/apt/keyrings
      curl -fsSL https://download.docker.com/linux/ubuntu/gpg -o /etc/apt/keyrings/docker.asc
      echo "deb [arch=amd64 signed-by=/etc/apt/keyrings/docker.asc] https://download.docker.com/linux/ubuntu jammy stable" > /etc/apt/sources.list.d/docker.list
      apt-get update -qq
      apt-get install -y -qq docker-ce docker-ce-cli containerd.io
      systemctl enable docker
    fi

    # Run Qdrant
    docker rm -f qdrant 2>/dev/null || true
    docker run -d \
      --name qdrant \
      --restart unless-stopped \
      --network host \
      -v /data/qdrant:/qdrant/storage \
      qdrant/qdrant:v1.7.4
  EOF

  labels = local.common_tags

  depends_on = [
    google_compute_disk.qdrant_data,
    google_compute_address.qdrant_internal,
    google_compute_router_nat.main,
  ]
}

# ==============================================================================
# Random Passwords & Secrets
# ==============================================================================
resource "random_password" "neo4j" {
  length  = 24
  special = false
}

resource "random_password" "jwt" {
  length  = 32
  special = false
}

resource "google_secret_manager_secret" "neo4j_password" {
  secret_id = "${local.name_prefix}-neo4j-password"
  replication {
    auto {}
  }
  labels = local.common_tags
}

resource "google_secret_manager_secret_version" "neo4j_password" {
  secret      = google_secret_manager_secret.neo4j_password.id
  secret_data = var.neo4j_password != "" ? var.neo4j_password : random_password.neo4j.result
}

resource "google_secret_manager_secret_iam_member" "neo4j_password" {
  secret_id = google_secret_manager_secret.neo4j_password.id
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${google_service_account.main.email}"
}

resource "google_secret_manager_secret" "llm_api_key" {
  secret_id = "${local.name_prefix}-llm-api-key"
  replication {
    auto {}
  }
  labels = local.common_tags
}

resource "google_secret_manager_secret_version" "llm_api_key" {
  secret      = google_secret_manager_secret.llm_api_key.id
  secret_data = var.llm_api_key
}

resource "google_secret_manager_secret_iam_member" "llm_api_key" {
  secret_id = google_secret_manager_secret.llm_api_key.id
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${google_service_account.main.email}"
}

resource "google_secret_manager_secret" "jwt_secret" {
  secret_id = "${local.name_prefix}-jwt-secret"
  replication {
    auto {}
  }
  labels = local.common_tags
}

resource "google_secret_manager_secret_version" "jwt_secret" {
  secret      = google_secret_manager_secret.jwt_secret.id
  secret_data = var.jwt_secret != "" ? var.jwt_secret : random_password.jwt.result
}

resource "google_secret_manager_secret_iam_member" "jwt_secret" {
  secret_id = google_secret_manager_secret.jwt_secret.id
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${google_service_account.main.email}"
}

resource "google_secret_manager_secret" "sentry_dsn" {
  secret_id = "${local.name_prefix}-sentry-dsn"
  replication {
    auto {}
  }
  labels = local.common_tags
}

resource "google_secret_manager_secret_version" "sentry_dsn" {
  secret      = google_secret_manager_secret.sentry_dsn.id
  secret_data = var.sentry_dsn
}

resource "google_secret_manager_secret_iam_member" "sentry_dsn" {
  secret_id = google_secret_manager_secret.sentry_dsn.id
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${google_service_account.main.email}"
}

# ==============================================================================
# Cloud DNS (Optional)
# ==============================================================================
resource "google_dns_managed_zone" "main" {
  count       = var.domain_name != "" ? 1 : 0
  name        = "${local.name_prefix}-zone"
  dns_name    = "${var.domain_name}."
  description = "Hystersis ${var.environment} DNS zone"
  labels      = local.common_tags
}

resource "google_dns_record_set" "api" {
  count        = var.domain_name != "" ? 1 : 0
  name         = "api.${google_dns_managed_zone.main[0].dns_name}"
  type         = "A"
  ttl          = 300
  managed_zone = google_dns_managed_zone.main[0].name
  rrdatas      = [google_cloud_run_v2_service.main.uri]
}

# ==============================================================================
# Monitoring: Uptime Check
# ==============================================================================
resource "google_monitoring_uptime_check_config" "api_health" {
  display_name = "${local.name_prefix}-api-health"
  timeout      = "10s"
  period       = "60s"

  http_check {
    path         = "/health"
    port         = 443
    use_ssl      = true
    validate_ssl = true
  }

  monitored_resource {
    type = "uptime_url"
    labels = {
      project_id = var.project_id
      host       = replace(replace(google_cloud_run_v2_service.main.uri, "https://", ""), "/.*", "")
    }
  }
}

resource "google_monitoring_alert_policy" "api_down" {
  display_name = "${local.name_prefix}-api-down"
  combiner     = "OR"
  enabled      = true

  conditions {
    display_name = "Uptime check failure"
    condition_threshold {
      filter     = "metric.type=\"monitoring.googleapis.com/uptime_check/check_passed\" AND resource.type=\"uptime_url\" AND metric.labels.check_id=\"${google_monitoring_uptime_check_config.api_health.uptime_check_id}\""
      duration   = "120s"
      comparison = "COMPARISON_LT"
      threshold_value = 1
      aggregations {
        alignment_period     = "60s"
        per_series_aligner   = "ALIGN_COUNT_TRUE"
        cross_series_reducer = "REDUCE_COUNT_FALSE"
      }
      trigger {
        count = 2
      }
    }
  }

  alert_strategy {
    auto_close = "1800s"
  }

  notification_channels = []
}

# ==============================================================================
 # OTel Collector Sidecar (optional)
# ==============================================================================
resource "google_cloud_run_v2_service_iam_member" "otel" {
  count    = var.enable_otel_collector ? 1 : 0
  name     = google_cloud_run_v2_service.main.name
  location = google_cloud_run_v2_service.main.location
  role     = "roles/run.invoker"
  member   = "allUsers"
}
