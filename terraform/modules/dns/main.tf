# DNS Module - Manages Cloud DNS hosted zones and records

# Create managed zone for the domain
resource "google_dns_managed_zone" "main" {
  name        = "${var.environment}-${replace(var.domain_name, ".", "-")}"
  dns_name    = "${var.domain_name}."
  description = "DNS zone for ${var.domain_name} - ${var.environment} environment"
  project     = var.project_id

  dnssec_config {
    state = "on"
  }

  labels = {
    environment = var.environment
    managed-by  = "terraform"
  }
}

# Create A record for dev.ohealth-ng.com (frontend)
resource "google_dns_record_set" "frontend" {
  name         = "dev.${var.domain_name}."
  type         = "A"
  ttl          = 300
  managed_zone = google_dns_managed_zone.main.name
  project      = var.project_id

  # This will be updated with the actual load balancer IP
  # For now, it's a placeholder that will be set via outputs
  rrdatas = [var.load_balancer_ip != "" ? var.load_balancer_ip : "0.0.0.0"]
}

# Create A record for dev.api.ohealth-ng.com (backend API)
resource "google_dns_record_set" "api" {
  name         = "dev.api.${var.domain_name}."
  type         = "A"
  ttl          = 300
  managed_zone = google_dns_managed_zone.main.name
  project      = var.project_id

  # This will be updated with the actual load balancer IP
  rrdatas = [var.load_balancer_ip != "" ? var.load_balancer_ip : "0.0.0.0"]
}

# Create CNAME for www subdomain (optional)
resource "google_dns_record_set" "www" {
  name         = "www.dev.${var.domain_name}."
  type         = "CNAME"
  ttl          = 300
  managed_zone = google_dns_managed_zone.main.name
  project      = var.project_id

  rrdatas = ["dev.${var.domain_name}."]
}
