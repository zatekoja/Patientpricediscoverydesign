# Cloud Run Services Module - Deploys containerized services

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

      env {
        name  = "VITE_API_URL"
        value = "https://${var.environment}.api.${var.domain_name}"
      }

      env {
        name  = "VITE_GRAPHQL_URL"
        value = "https://${var.environment}.api.${var.domain_name}/graphql"
      }

      env {
        name  = "VITE_SSE_URL"
        value = "https://${var.environment}.api.${var.domain_name}/sse"
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
        value = var.typesense_url != "" ? var.typesense_url : "http://typesense:8108"
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
        value = var.typesense_url != "" ? var.typesense_url : "http://typesense:8108"
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
resource "google_cloud_run_service_iam_member" "frontend_noauth" {
  project  = var.project_id
  location = var.region
  service  = google_cloud_run_v2_service.frontend.name
  role     = "roles/run.invoker"
  member   = "allUsers"
}

resource "google_cloud_run_service_iam_member" "api_noauth" {
  project  = var.project_id
  location = var.region
  service  = google_cloud_run_v2_service.api.name
  role     = "roles/run.invoker"
  member   = "allUsers"
}

resource "google_cloud_run_service_iam_member" "graphql_noauth" {
  project  = var.project_id
  location = var.region
  service  = google_cloud_run_v2_service.graphql.name
  role     = "roles/run.invoker"
  member   = "allUsers"
}

resource "google_cloud_run_service_iam_member" "sse_noauth" {
  project  = var.project_id
  location = var.region
  service  = google_cloud_run_v2_service.sse.name
  role     = "roles/run.invoker"
  member   = "allUsers"
}
