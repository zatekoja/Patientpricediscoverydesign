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

variable "frontend_service_name" {
  description = "Frontend Cloud Run service name"
  type        = string
}

variable "api_service_name" {
  description = "API Cloud Run service name"
  type        = string
}

variable "graphql_service_name" {
  description = "GraphQL Cloud Run service name"
  type        = string
}

variable "sse_service_name" {
  description = "SSE Cloud Run service name"
  type        = string
}
