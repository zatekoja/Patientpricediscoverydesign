# Cloud Run Services Module - Deploys containerized services

# SECURITY NOTE: Secret Manager secrets are created with secret_data from Terraform variables.
# This means secret values will be stored in Terraform state (potentially in plaintext).
# For production use, consider:
# 1. Creating secrets/versions out-of-band (via gcloud or console)
# 2. Using remote state backend with encryption (GCS with encryption at rest)
# 3. Implementing strict state access controls
# 4. Using Terraform Cloud/Enterprise with encrypted state
# Alternatively, reference existing secrets instead of creating them in Terraform.

# Dedicated service account for Cloud Run services
resource "google_service_account" "cloud_run" {
  account_id   = "${var.environment}-cloud-run-sa"
  display_name = "Cloud Run service account for ${var.environment}"
  project      = var.project_id
}

# Create Secret Manager secrets for API keys
resource "google_secret_manager_secret" "google_maps_api_key" {
  secret_id = "${var.environment}-google-maps-api-key"
  project   = var.project_id

  replication {
    auto {}
  }
}

resource "google_secret_manager_secret_version" "google_maps_api_key" {
  secret      = google_secret_manager_secret.google_maps_api_key.id
  secret_data = var.google_maps_api_key
}

resource "google_secret_manager_secret" "typesense_api_key" {
  secret_id = "${var.environment}-typesense-api-key"
  project   = var.project_id

  replication {
    auto {}
  }
}

resource "google_secret_manager_secret_version" "typesense_api_key" {
  secret      = google_secret_manager_secret.typesense_api_key.id
  secret_data = var.typesense_api_key
}

resource "google_secret_manager_secret" "openai_api_key" {
  secret_id = "${var.environment}-openai-api-key"
  project   = var.project_id

  replication {
    auto {}
  }
}

resource "google_secret_manager_secret_version" "openai_api_key" {
  secret      = google_secret_manager_secret.openai_api_key.id
  secret_data = var.openai_api_key
}

# Cloud Run service for Frontend
resource "google_cloud_run_v2_service" "frontend" {
  name     = "${var.environment}-ppd-frontend"
  project  = var.project_id
  location = var.region

  template {
    service_account = google_service_account.cloud_run.email
    
    containers {
      image = "gcr.io/${var.project_id}/${var.environment}-ppd-frontend:latest"

      ports {
        container_port = 80
      }

      resources {
        limits = {
          cpu    = "1"
          memory = "512Mi"
        }
      }

      # NOTE:
      # Vite environment variables (VITE_*) are resolved at build time via import.meta.env.
      # They must be provided to the image build (for example via Docker build args in CI)
      # and cannot be overridden via Cloud Run runtime environment variables.
      # The values below are placeholders and will not affect the built frontend.
      env {
        name  = "VITE_API_BASE_URL"
        value = "https://api.${var.environment}.${var.domain_name}"
      }

      env {
        name  = "VITE_SSE_BASE_URL"
        value = "https://api.${var.environment}.${var.domain_name}"
      }
    }

    scaling {
      min_instance_count = 1
      max_instance_count = 10
    }
  }

  traffic {
    type    = "TRAFFIC_TARGET_ALLOCATION_TYPE_LATEST"
    percent = 100
  }
}

# Cloud Run service for API
resource "google_cloud_run_v2_service" "api" {
  name     = "${var.environment}-ppd-api"
  project  = var.project_id
  location = var.region

  template {
    vpc_access {
      connector = var.vpc_connector_id
      egress    = "PRIVATE_RANGES_ONLY"
    }

    containers {
      image = "gcr.io/${var.project_id}/${var.environment}-ppd-api:latest"

      ports {
        container_port = 8080
      }

      resources {
        limits = {
          cpu    = "2"
          memory = "2Gi"
        }
      }

      env {
        name  = "SERVER_HOST"
        value = "0.0.0.0"
      }

      env {
        name  = "SERVER_PORT"
        value = "8080"
      }

      env {
        name  = "DB_HOST"
        value = var.postgres_private_ip
      }

      env {
        name  = "DB_PORT"
        value = "5432"
      }

      env {
        name  = "DB_NAME"
        value = var.postgres_database_name
      }

      env {
        name  = "DB_USER"
        value = "postgres"
      }

      env {
        name = "DB_PASSWORD"
        value_source {
          secret_key_ref {
            secret  = var.postgres_password_secret_id
            version = "latest"
          }
        }
      }

      env {
        name  = "DB_SSLMODE"
        value = "require"
      }

      env {
        name  = "REDIS_HOST"
        value = var.redis_host
      }

      env {
        name  = "REDIS_PORT"
        value = tostring(var.redis_port)
      }

      env {
        name  = "TYPESENSE_URL"
        value = var.typesense_url != "" ? var.typesense_url : "http://localhost:8108"
      }

      env {
        name = "TYPESENSE_API_KEY"
        value_source {
          secret_key_ref {
            secret  = google_secret_manager_secret.typesense_api_key.secret_id
            version = "latest"
          }
        }
      }

      env {
        name = "GEOLOCATION_API_KEY"
        value_source {
          secret_key_ref {
            secret  = google_secret_manager_secret.google_maps_api_key.secret_id
            version = "latest"
          }
        }
      }

      env {
        name = "OPENAI_API_KEY"
        value_source {
          secret_key_ref {
            secret  = google_secret_manager_secret.openai_api_key.secret_id
            version = "latest"
          }
        }
      }
    }

    scaling {
      min_instance_count = 1
      max_instance_count = 20
    }
  }

  traffic {
    type    = "TRAFFIC_TARGET_ALLOCATION_TYPE_LATEST"
    percent = 100
  }
}

# Cloud Run service for GraphQL
resource "google_cloud_run_v2_service" "graphql" {
  name     = "${var.environment}-ppd-graphql"
  project  = var.project_id
  location = var.region

  template {
    vpc_access {
      connector = var.vpc_connector_id
      egress    = "PRIVATE_RANGES_ONLY"
    }

    containers {
      image = "gcr.io/${var.project_id}/${var.environment}-ppd-graphql:latest"

      ports {
        container_port = 8081
      }

      resources {
        limits = {
          cpu    = "2"
          memory = "2Gi"
        }
      }

      env {
        name  = "SERVER_HOST"
        value = "0.0.0.0"
      }

      env {
        name  = "SERVER_PORT"
        value = "8081"
      }

      env {
        name  = "DB_HOST"
        value = var.postgres_private_ip
      }

      env {
        name  = "DB_PORT"
        value = "5432"
      }

      env {
        name  = "DB_NAME"
        value = var.postgres_database_name
      }

      env {
        name  = "DB_USER"
        value = "postgres"
      }

      env {
        name = "DB_PASSWORD"
        value_source {
          secret_key_ref {
            secret  = var.postgres_password_secret_id
            version = "latest"
          }
        }
      }

      env {
        name  = "DB_SSLMODE"
        value = "require"
      }

      env {
        name  = "REDIS_HOST"
        value = var.redis_host
      }

      env {
        name  = "REDIS_PORT"
        value = tostring(var.redis_port)
      }

      env {
        name  = "TYPESENSE_URL"
        value = var.typesense_url != "" ? var.typesense_url : "http://localhost:8108"
      }

      env {
        name = "TYPESENSE_API_KEY"
        value_source {
          secret_key_ref {
            secret  = google_secret_manager_secret.typesense_api_key.secret_id
            version = "latest"
          }
        }
      }
    }

    scaling {
      min_instance_count = 1
      max_instance_count = 20
    }
  }

  traffic {
    type    = "TRAFFIC_TARGET_ALLOCATION_TYPE_LATEST"
    percent = 100
  }
}

# Cloud Run service for SSE
resource "google_cloud_run_v2_service" "sse" {
  name     = "${var.environment}-ppd-sse"
  project  = var.project_id
  location = var.region

  template {
    vpc_access {
      connector = var.vpc_connector_id
      egress    = "PRIVATE_RANGES_ONLY"
    }

    containers {
      image = "gcr.io/${var.project_id}/${var.environment}-ppd-sse:latest"

      ports {
        container_port = 8082
      }

      resources {
        limits = {
          cpu    = "1"
          memory = "1Gi"
        }
      }

      env {
        name  = "SERVER_HOST"
        value = "0.0.0.0"
      }

      env {
        name  = "SERVER_PORT"
        value = "8082"
      }

      env {
        name  = "REDIS_HOST"
        value = var.redis_host
      }

      env {
        name  = "REDIS_PORT"
        value = tostring(var.redis_port)
      }
    }

    scaling {
      min_instance_count = 1
      max_instance_count = 10
    }
  }

  traffic {
    type    = "TRAFFIC_TARGET_ALLOCATION_TYPE_LATEST"
    percent = 100
  }
}

# IAM policy to allow unauthenticated access (for public-facing services)
# Using v2 IAM resources to match v2 Cloud Run services
resource "google_cloud_run_v2_service_iam_member" "frontend_noauth" {
  project  = var.project_id
  location = var.region
  name     = google_cloud_run_v2_service.frontend.name
  role     = "roles/run.invoker"
  member   = "allUsers"
}

resource "google_cloud_run_v2_service_iam_member" "api_noauth" {
  project  = var.project_id
  location = var.region
  name     = google_cloud_run_v2_service.api.name
  role     = "roles/run.invoker"
  member   = "allUsers"
}

resource "google_cloud_run_v2_service_iam_member" "graphql_noauth" {
  project  = var.project_id
  location = var.region
  name     = google_cloud_run_v2_service.graphql.name
  role     = "roles/run.invoker"
  member   = "allUsers"
}

resource "google_cloud_run_v2_service_iam_member" "sse_noauth" {
  project  = var.project_id
  location = var.region
  name     = google_cloud_run_v2_service.sse.name
  role     = "roles/run.invoker"
  member   = "allUsers"
}

# Grant the Cloud Run service account access to Secret Manager secrets
resource "google_secret_manager_secret_iam_member" "google_maps_access" {
  secret_id = google_secret_manager_secret.google_maps_api_key.id
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${google_service_account.cloud_run.email}"
}

resource "google_secret_manager_secret_iam_member" "typesense_access" {
  secret_id = google_secret_manager_secret.typesense_api_key.id
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${google_service_account.cloud_run.email}"
}

resource "google_secret_manager_secret_iam_member" "openai_access" {
  secret_id = google_secret_manager_secret.openai_api_key.id
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${google_service_account.cloud_run.email}"
}

# Grant access to the PostgreSQL password secret (created in databases module)
resource "google_secret_manager_secret_iam_member" "postgres_password_access" {
  secret_id = var.postgres_password_secret_id
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${google_service_account.cloud_run.email}"
}
