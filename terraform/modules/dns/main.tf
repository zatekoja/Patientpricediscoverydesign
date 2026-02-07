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

# Create A record for frontend (e.g., dev.ohealth-ng.com)
resource "google_dns_record_set" "frontend" {
  count        = var.load_balancer_ip != "" ? 1 : 0
  name         = "${var.environment}.${var.domain_name}."
  type         = "A"
  ttl          = 300
  managed_zone = google_dns_managed_zone.main.name
  project      = var.project_id

  rrdatas = [var.load_balancer_ip]
}

# Create A record for backend API (e.g., dev.api.ohealth-ng.com)
resource "google_dns_record_set" "api" {
  count        = var.load_balancer_ip != "" ? 1 : 0
  name         = "${var.environment}.api.${var.domain_name}."
  type         = "A"
  ttl          = 300
  managed_zone = google_dns_managed_zone.main.name
  project      = var.project_id

  rrdatas = [var.load_balancer_ip]
}

# Create CNAME for www subdomain (optional)
resource "google_dns_record_set" "www" {
  count        = var.load_balancer_ip != "" ? 1 : 0
  name         = "www.${var.environment}.${var.domain_name}."
  type         = "CNAME"
  ttl          = 300
  managed_zone = google_dns_managed_zone.main.name
  project      = var.project_id

  rrdatas = ["${var.environment}.${var.domain_name}."]
}
