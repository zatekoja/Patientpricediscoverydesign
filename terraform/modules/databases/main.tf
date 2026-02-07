# Databases Module - Cloud SQL PostgreSQL, Memorystore Redis

# Create Cloud SQL PostgreSQL instance
resource "google_sql_database_instance" "postgres" {
  name             = "${var.environment}-ppd-postgres"
  project          = var.project_id
  region           = var.region
  database_version = "POSTGRES_15"

  settings {
    tier              = "db-custom-2-7680" # 2 vCPUs, 7.5GB RAM
    availability_type = "REGIONAL"          # High availability with failover replica
    disk_type         = "PD_SSD"
    disk_size         = 50
    disk_autoresize   = true

    backup_configuration {
      enabled                        = true
      start_time                     = "03:00"
      point_in_time_recovery_enabled = true
      transaction_log_retention_days = 7
      backup_retention_settings {
        retained_backups = 7
      }
    }

    ip_configuration {
      ipv4_enabled    = false
      private_network = var.network_self_link
      require_ssl     = true
    }

    maintenance_window {
      day  = 7 # Sunday
      hour = 3
    }

    insights_config {
      query_insights_enabled  = true
      query_string_length     = 1024
      record_application_tags = true
    }

    database_flags {
      name  = "max_connections"
      value = "100"
    }
  }

  deletion_protection = true
}

# Create PostgreSQL database
resource "google_sql_database" "main" {
  name     = "patient_price_discovery"
  project  = var.project_id
  instance = google_sql_database_instance.postgres.name
}

# Create PostgreSQL user
resource "google_sql_user" "postgres_user" {
  name     = "postgres"
  project  = var.project_id
  instance = google_sql_database_instance.postgres.name
  password = var.postgres_password != "" ? var.postgres_password : random_password.postgres_password.result
}

# Generate random password for PostgreSQL
resource "random_password" "postgres_password" {
  length  = 32
  special = true
}

# Store PostgreSQL password in Secret Manager
resource "google_secret_manager_secret" "postgres_password" {
  secret_id = "${var.environment}-ppd-postgres-password"
  project   = var.project_id

  replication {
    auto {}
  }
}

resource "google_secret_manager_secret_version" "postgres_password" {
  secret      = google_secret_manager_secret.postgres_password.id
  secret_data = var.postgres_password != "" ? var.postgres_password : random_password.postgres_password.result
}

# Create Memorystore Redis instance
resource "google_redis_instance" "cache" {
  name               = "${var.environment}-ppd-redis"
  project            = var.project_id
  region             = var.region
  tier               = "STANDARD_HA" # High availability
  memory_size_gb     = 5
  redis_version      = "REDIS_7_0"
  display_name       = "Patient Price Discovery Redis Cache - ${var.environment}"
  authorized_network = var.network_id

  redis_configs = {
    maxmemory-policy = "allkeys-lru"
  }

  maintenance_policy {
    weekly_maintenance_window {
      day = "SUNDAY"
      start_time {
        hours   = 3
        minutes = 0
      }
    }
  }
}
