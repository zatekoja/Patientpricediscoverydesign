variable "project_id" {
  description = "GCP Project ID"
  type        = string
}

variable "region" {
  description = "GCP region"
  type        = string
}

variable "environment" {
  description = "Environment name"
  type        = string
}

variable "domain_name" {
  description = "Domain name"
  type        = string
}

variable "frontend_service_url" {
  description = "Frontend Cloud Run service URL"
  type        = string
}

variable "api_service_url" {
  description = "API Cloud Run service URL"
  type        = string
}

variable "graphql_service_url" {
  description = "GraphQL Cloud Run service URL"
  type        = string
}

variable "sse_service_url" {
  description = "SSE Cloud Run service URL"
  type        = string
}
