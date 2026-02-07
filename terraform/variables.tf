# Variables for Patient Price Discovery GCP Infrastructure

variable "project_id" {
  description = "GCP Project ID"
  type        = string
  default     = "open-health-index-dev"
}

variable "region" {
  description = "GCP region for resources"
  type        = string
  default     = "us-central1"
}

variable "environment" {
  description = "Environment name (dev, staging, prod)"
  type        = string
  default     = "dev"
}

variable "domain_name" {
  description = "Domain name for the application"
  type        = string
  default     = "ohealth-ng.com"
}

variable "google_maps_api_key" {
  description = "Google Maps API key for geolocation"
  type        = string
  sensitive   = true
  default     = ""
}

variable "typesense_api_key" {
  description = "Typesense API key for search"
  type        = string
  sensitive   = true
  default     = ""
}

variable "openai_api_key" {
  description = "OpenAI API key for AI features"
  type        = string
  sensitive   = true
  default     = ""
}

variable "postgres_password" {
  description = "Password for PostgreSQL database"
  type        = string
  sensitive   = true
  default     = ""
}
