output "load_balancer_ip" {
  description = "Load Balancer IP address"
  value       = google_compute_global_address.default.address
}

output "ssl_certificate_name" {
  description = "SSL certificate name"
  value       = google_compute_managed_ssl_certificate.default.name
}

output "url_map_name" {
  description = "URL map name"
  value       = google_compute_url_map.default.name
}
