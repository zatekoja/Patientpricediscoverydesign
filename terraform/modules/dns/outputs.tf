output "managed_zone_name" {
  description = "The name of the managed DNS zone"
  value       = google_dns_managed_zone.main.name
}

output "managed_zone_dns_name" {
  description = "The DNS name of the managed zone"
  value       = google_dns_managed_zone.main.dns_name
}

output "nameservers" {
  description = "List of nameservers for the zone"
  value       = google_dns_managed_zone.main.name_servers
}
