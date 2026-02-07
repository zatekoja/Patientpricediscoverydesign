# Development Environment - Main Configuration

terraform {
  # Uncomment to use remote state backend
  # backend "gcs" {
  #   bucket = "open-health-index-dev-tfstate"
  #   prefix = "terraform/dev"
  # }
}

module "infrastructure" {
  source = "../../"

  project_id  = var.project_id
  region      = var.region
  environment = var.environment
  domain_name = var.domain_name

  google_maps_api_key = var.google_maps_api_key
  typesense_api_key   = var.typesense_api_key
  openai_api_key      = var.openai_api_key
  postgres_password   = var.postgres_password
}

output "dns_nameservers" {
  description = "Name servers for the DNS zone"
  value       = module.infrastructure.dns_nameservers
}

output "frontend_url" {
  description = "Frontend application URL"
  value       = module.infrastructure.frontend_url
}

output "api_url" {
  description = "API endpoint URL"
  value       = module.infrastructure.api_url
}

output "load_balancer_ip" {
  description = "Load Balancer IP address"
  value       = module.infrastructure.load_balancer_ip
}
