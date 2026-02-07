# Load Balancer Module - Global HTTPS Load Balancer with SSL

# Reserve a global static IP address
resource "google_compute_global_address" "default" {
  name    = "${var.environment}-ppd-lb-ip"
  project = var.project_id
}

# Create SSL certificate
resource "google_compute_managed_ssl_certificate" "default" {
  name    = "${var.environment}-ppd-ssl-cert"
  project = var.project_id

  managed {
    domains = [
      "${var.environment}.${var.domain_name}",
      "api.${var.environment}.${var.domain_name}",
      "www.${var.environment}.${var.domain_name}"
    ]
  }
}

# Create backend service for Frontend
resource "google_compute_backend_service" "frontend" {
  name                  = "${var.environment}-ppd-frontend-backend"
  project               = var.project_id
  protocol              = "HTTP"
  port_name             = "http"
  timeout_sec           = 30
  enable_cdn            = true
  compression_mode      = "AUTOMATIC"
  load_balancing_scheme = "EXTERNAL_MANAGED"
  
  backend {
    group = google_compute_region_network_endpoint_group.frontend.id
  }

  cdn_policy {
    cache_mode = "CACHE_ALL_STATIC"
    default_ttl = 3600
    max_ttl     = 86400
    client_ttl  = 3600
  }

  log_config {
    enable = true
    sample_rate = 1.0
  }
}

# Create backend service for API
resource "google_compute_backend_service" "api" {
  name                  = "${var.environment}-ppd-api-backend"
  project               = var.project_id
  protocol              = "HTTP"
  port_name             = "http"
  timeout_sec           = 30
  load_balancing_scheme = "EXTERNAL_MANAGED"

  backend {
    group = google_compute_region_network_endpoint_group.api.id
  }

  log_config {
    enable = true
    sample_rate = 1.0
  }
}

# Create backend service for GraphQL
resource "google_compute_backend_service" "graphql" {
  name                  = "${var.environment}-ppd-graphql-backend"
  project               = var.project_id
  protocol              = "HTTP"
  port_name             = "http"
  timeout_sec           = 30
  load_balancing_scheme = "EXTERNAL_MANAGED"

  backend {
    group = google_compute_region_network_endpoint_group.graphql.id
  }

  log_config {
    enable = true
    sample_rate = 1.0
  }
}

# Create backend service for SSE
resource "google_compute_backend_service" "sse" {
  name                  = "${var.environment}-ppd-sse-backend"
  project               = var.project_id
  protocol              = "HTTP"
  port_name             = "http"
  timeout_sec           = 300 # Longer timeout for SSE connections
  load_balancing_scheme = "EXTERNAL_MANAGED"

  backend {
    group = google_compute_region_network_endpoint_group.sse.id
  }

  log_config {
    enable = true
    sample_rate = 1.0
  }
}

# Create Network Endpoint Groups for Cloud Run services
resource "google_compute_region_network_endpoint_group" "frontend" {
  name                  = "${var.environment}-ppd-frontend-neg"
  project               = var.project_id
  network_endpoint_type = "SERVERLESS"
  region                = var.region

  cloud_run {
    service = var.frontend_service_name
  }
}

resource "google_compute_region_network_endpoint_group" "api" {
  name                  = "${var.environment}-ppd-api-neg"
  project               = var.project_id
  network_endpoint_type = "SERVERLESS"
  region                = var.region

  cloud_run {
    service = var.api_service_name
  }
}

resource "google_compute_region_network_endpoint_group" "graphql" {
  name                  = "${var.environment}-ppd-graphql-neg"
  project               = var.project_id
  network_endpoint_type = "SERVERLESS"
  region                = var.region

  cloud_run {
    service = var.graphql_service_name
  }
}

resource "google_compute_region_network_endpoint_group" "sse" {
  name                  = "${var.environment}-ppd-sse-neg"
  project               = var.project_id
  network_endpoint_type = "SERVERLESS"
  region                = var.region

  cloud_run {
    service = var.sse_service_name
  }
}

# Create URL map for routing
resource "google_compute_url_map" "default" {
  name            = "${var.environment}-ppd-url-map"
  project         = var.project_id
  default_service = google_compute_backend_service.frontend.id

  host_rule {
    hosts        = ["api.${var.environment}.${var.domain_name}"]
    path_matcher = "api"
  }

  host_rule {
    hosts        = ["${var.environment}.${var.domain_name}", "www.${var.environment}.${var.domain_name}"]
    path_matcher = "frontend"
  }

  path_matcher {
    name            = "api"
    default_service = google_compute_backend_service.api.id

    path_rule {
      paths   = ["/graphql", "/graphql/*"]
      service = google_compute_backend_service.graphql.id
    }

    path_rule {
      paths   = ["/sse", "/sse/*"]
      service = google_compute_backend_service.sse.id
    }
  }

  path_matcher {
    name            = "frontend"
    default_service = google_compute_backend_service.frontend.id
  }
}

# Create HTTPS proxy
resource "google_compute_target_https_proxy" "default" {
  name             = "${var.environment}-ppd-https-proxy"
  project          = var.project_id
  url_map          = google_compute_url_map.default.id
  ssl_certificates = [google_compute_managed_ssl_certificate.default.id]
}

# Create global forwarding rule
resource "google_compute_global_forwarding_rule" "https" {
  name                  = "${var.environment}-ppd-https-forwarding-rule"
  project               = var.project_id
  ip_protocol           = "TCP"
  load_balancing_scheme = "EXTERNAL_MANAGED"
  port_range            = "443"
  target                = google_compute_target_https_proxy.default.id
  ip_address            = google_compute_global_address.default.id
}

# Create HTTP to HTTPS redirect
resource "google_compute_url_map" "https_redirect" {
  name    = "${var.environment}-ppd-https-redirect"
  project = var.project_id

  default_url_redirect {
    https_redirect         = true
    redirect_response_code = "MOVED_PERMANENTLY_DEFAULT"
    strip_query            = false
  }
}

resource "google_compute_target_http_proxy" "https_redirect" {
  name    = "${var.environment}-ppd-http-proxy"
  project = var.project_id
  url_map = google_compute_url_map.https_redirect.id
}

resource "google_compute_global_forwarding_rule" "http" {
  name                  = "${var.environment}-ppd-http-forwarding-rule"
  project               = var.project_id
  ip_protocol           = "TCP"
  load_balancing_scheme = "EXTERNAL_MANAGED"
  port_range            = "80"
  target                = google_compute_target_http_proxy.https_redirect.id
  ip_address            = google_compute_global_address.default.id
}
