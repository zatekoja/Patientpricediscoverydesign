# Main Terraform configuration for Patient Price Discovery on GCP
# This is the root module that orchestrates all infrastructure components

terraform {
  required_version = ">= 1.0"

  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "~> 6.0"
    }
    google-beta = {
      source  = "hashicorp/google-beta"
      version = "~> 6.0"
    }
  }

  # Backend configuration for storing Terraform state
  # Uncomment and configure when ready to use remote state
  # backend "gcs" {
  #   bucket = "open-health-index-dev-tfstate"
  #   prefix = "terraform/state"
  # }
}

# Configure the Google Cloud Provider
provider "google" {
  project = var.project_id
  region  = var.region
}

provider "google-beta" {
  project = var.project_id
  region  = var.region
}

# Enable required GCP APIs
resource "google_project_service" "required_apis" {
  for_each = toset([
    "compute.googleapis.com",
    "run.googleapis.com",
    "sqladmin.googleapis.com",
    "redis.googleapis.com",
    "dns.googleapis.com",
    "containerregistry.googleapis.com",
    "cloudbuild.googleapis.com",
    "vpcaccess.googleapis.com",
    "servicenetworking.googleapis.com",
    "certificatemanager.googleapis.com",
    "cloudresourcemanager.googleapis.com",
    "secretmanager.googleapis.com",
  ])

  project = var.project_id
  service = each.value

  disable_on_destroy = false
}

# DNS Configuration Module
module "dns" {
  source = "./modules/dns"

  project_id       = var.project_id
  domain_name      = var.domain_name
  environment      = var.environment
  load_balancer_ip = module.load_balancer.load_balancer_ip

  depends_on = [google_project_service.required_apis, module.load_balancer]
}

# Networking Module
module "networking" {
  source = "./modules/networking"

  project_id  = var.project_id
  region      = var.region
  environment = var.environment

  depends_on = [google_project_service.required_apis]
}

# Databases Module
module "databases" {
  source = "./modules/databases"

  project_id        = var.project_id
  region            = var.region
  environment       = var.environment
  network_id        = module.networking.network_id
  network_self_link = module.networking.network_self_link
  postgres_password = var.postgres_password

  depends_on = [google_project_service.required_apis]
}

# Cloud Run Services Module
module "cloud_run" {
  source = "./modules/cloud-run"

  project_id                = var.project_id
  region                    = var.region
  environment               = var.environment
  domain_name               = var.domain_name
  vpc_connector_id          = module.networking.vpc_connector_id
  
  # Database connections
  postgres_connection_name  = module.databases.postgres_connection_name
  postgres_database_name    = module.databases.postgres_database_name
  postgres_password_secret_id = module.databases.postgres_password_secret_id
  postgres_private_ip       = module.databases.postgres_private_ip
  redis_host                = module.databases.redis_host
  redis_port                = module.databases.redis_port

  # API Keys (to be stored in Secret Manager)
  google_maps_api_key       = var.google_maps_api_key
  typesense_api_key         = var.typesense_api_key
  typesense_url             = var.typesense_url
  openai_api_key            = var.openai_api_key

  depends_on = [
    google_project_service.required_apis,
    module.databases,
    module.networking
  ]
}

# Load Balancer Module
module "load_balancer" {
  source = "./modules/load-balancer"

  project_id              = var.project_id
  region                  = var.region
  environment             = var.environment
  domain_name             = var.domain_name
  
  # Cloud Run service names
  frontend_service_name   = module.cloud_run.frontend_service_name
  api_service_name        = module.cloud_run.api_service_name
  graphql_service_name    = module.cloud_run.graphql_service_name
  sse_service_name        = module.cloud_run.sse_service_name

  depends_on = [
    google_project_service.required_apis,
    module.cloud_run,
  ]
}

# Output important values
output "dns_nameservers" {
  description = "Name servers for the DNS zone - configure these in your domain registrar"
  value       = module.dns.nameservers
}

output "frontend_url" {
  description = "Frontend application URL"
  value       = "https://${var.environment}.${var.domain_name}"
}

output "api_url" {
  description = "API endpoint URL"
  value       = "https://${var.environment}.api.${var.domain_name}"
}

output "postgres_connection_name" {
  description = "PostgreSQL Cloud SQL connection name"
  value       = module.databases.postgres_connection_name
}

output "redis_host" {
  description = "Redis Memorystore host"
  value       = module.databases.redis_host
  sensitive   = true
}

output "load_balancer_ip" {
  description = "Load Balancer IP address"
  value       = module.load_balancer.load_balancer_ip
}
