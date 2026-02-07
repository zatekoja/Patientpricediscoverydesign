# Networking Module - VPC, Subnets, VPC Access Connector

# Create VPC network
resource "google_compute_network" "main" {
  name                    = "${var.environment}-ppd-network"
  project                 = var.project_id
  auto_create_subnetworks = false
  description             = "VPC network for Patient Price Discovery ${var.environment}"
}

# Create subnet for the application
resource "google_compute_subnetwork" "main" {
  name          = "${var.environment}-ppd-subnet"
  project       = var.project_id
  ip_cidr_range = "10.0.0.0/24"
  region        = var.region
  network       = google_compute_network.main.id

  private_ip_google_access = true

  log_config {
    aggregation_interval = "INTERVAL_10_MIN"
    flow_sampling        = 0.5
    metadata             = "INCLUDE_ALL_METADATA"
  }
}

# Create subnet for VPC Access Connector (for Cloud Run to access VPC resources)
resource "google_compute_subnetwork" "vpc_connector" {
  name          = "${var.environment}-ppd-vpc-connector-subnet"
  project       = var.project_id
  ip_cidr_range = "10.8.0.0/28"
  region        = var.region
  network       = google_compute_network.main.id
}

# Create VPC Access Connector for Cloud Run services to access VPC resources
resource "google_vpc_access_connector" "connector" {
  name          = "${var.environment}-ppd-connector"
  project       = var.project_id
  region        = var.region
  network       = google_compute_network.main.name
  ip_cidr_range = "10.8.0.0/28"

  # Performance configuration
  min_instances = 2
  max_instances = 10
  machine_type  = "e2-micro"

  depends_on = [google_compute_subnetwork.vpc_connector]
}

# Reserve IP range for private service connection (for Cloud SQL)
resource "google_compute_global_address" "private_ip_range" {
  name          = "${var.environment}-ppd-private-ip-range"
  project       = var.project_id
  purpose       = "VPC_PEERING"
  address_type  = "INTERNAL"
  prefix_length = 16
  network       = google_compute_network.main.id
}

# Create private VPC connection for Google services (Cloud SQL)
resource "google_service_networking_connection" "private_vpc_connection" {
  network                 = google_compute_network.main.id
  service                 = "servicenetworking.googleapis.com"
  reserved_peering_ranges = [google_compute_global_address.private_ip_range.name]
}

# Firewall rule to allow internal communication
resource "google_compute_firewall" "allow_internal" {
  name    = "${var.environment}-ppd-allow-internal"
  project = var.project_id
  network = google_compute_network.main.name

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

  source_ranges = ["10.0.0.0/8"]
}

# Firewall rule to allow health checks from Google Load Balancers
resource "google_compute_firewall" "allow_health_checks" {
  name    = "${var.environment}-ppd-allow-health-checks"
  project = var.project_id
  network = google_compute_network.main.name

  allow {
    protocol = "tcp"
  }

  source_ranges = [
    "35.191.0.0/16",
    "130.211.0.0/22"
  ]

  target_tags = ["allow-health-check"]
}

# Firewall rule to allow HTTPS traffic
resource "google_compute_firewall" "allow_https" {
  name    = "${var.environment}-ppd-allow-https"
  project = var.project_id
  network = google_compute_network.main.name

  allow {
    protocol = "tcp"
    ports    = ["443"]
  }

  source_ranges = ["0.0.0.0/0"]
  target_tags   = ["https-server"]
}

# Firewall rule to allow HTTP traffic (for redirects)
resource "google_compute_firewall" "allow_http" {
  name    = "${var.environment}-ppd-allow-http"
  project = var.project_id
  network = google_compute_network.main.name

  allow {
    protocol = "tcp"
    ports    = ["80"]
  }

  source_ranges = ["0.0.0.0/0"]
  target_tags   = ["http-server"]
}
