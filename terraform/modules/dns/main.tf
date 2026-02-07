# DNS Module - Manages Cloud DNS hosted zones and records

# NOTE: This creates an environment-specific subdomain zone (e.g., dev.ohealth-ng.com)
# to allow multiple environments to run in parallel without nameserver conflicts.
# You will need to delegate the subdomain from your main domain registrar by creating
# NS records in the parent zone pointing to the nameservers output by this module.
# Example: In the main ohealth-ng.com zone, create NS records for dev.ohealth-ng.com
# pointing to the nameservers shown in the Terraform output.

# Create managed zone for the environment-specific subdomain
resource "google_dns_managed_zone" "main" {
  name        = "${var.environment}-${replace(var.domain_name, ".", "-")}"
  dns_name    = "${var.environment}.${var.domain_name}."
  description = "DNS zone for ${var.environment}.${var.domain_name}"
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
# Note: This creates dev.api.ohealth-ng.com, which is outside the dev.ohealth-ng.com zone
# For this to work, you need to either:
# 1. Create this record in the parent ohealth-ng.com zone, OR
# 2. Change the hostname pattern to api.dev.ohealth-ng.com (within this zone)
# Currently using pattern: ${environment}.api.${domain} to match load balancer config
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
