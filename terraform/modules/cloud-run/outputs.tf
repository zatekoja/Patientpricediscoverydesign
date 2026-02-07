output "frontend_service_url" {
  description = "URL of the frontend Cloud Run service"
  value       = google_cloud_run_v2_service.frontend.uri
}

output "frontend_service_name" {
  description = "Name of the frontend Cloud Run service"
  value       = google_cloud_run_v2_service.frontend.name
}

output "api_service_url" {
  description = "URL of the API Cloud Run service"
  value       = google_cloud_run_v2_service.api.uri
}

output "api_service_name" {
  description = "Name of the API Cloud Run service"
  value       = google_cloud_run_v2_service.api.name
}

output "graphql_service_url" {
  description = "URL of the GraphQL Cloud Run service"
  value       = google_cloud_run_v2_service.graphql.uri
}

output "graphql_service_name" {
  description = "Name of the GraphQL Cloud Run service"
  value       = google_cloud_run_v2_service.graphql.name
}

output "sse_service_url" {
  description = "URL of the SSE Cloud Run service"
  value       = google_cloud_run_v2_service.sse.uri
}

output "sse_service_name" {
  description = "Name of the SSE Cloud Run service"
  value       = google_cloud_run_v2_service.sse.name
}
