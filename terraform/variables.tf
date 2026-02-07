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
  description = "Google Maps API key for geolocation (required)"
  type        = string
  sensitive   = true
  default     = ""

  validation {
    condition     = var.google_maps_api_key != ""
    error_message = "Google Maps API key is required. Please set google_maps_api_key variable."
  }
}

variable "typesense_api_key" {
  description = "Typesense API key for search (required)"
  type        = string
  sensitive   = true
  default     = ""

  validation {
    condition     = var.typesense_api_key != ""
    error_message = "Typesense API key is required. Please set typesense_api_key variable."
  }
}

variable "openai_api_key" {
  description = "OpenAI API key for AI features (required)"
  type        = string
  sensitive   = true
  default     = ""

  validation {
    condition     = var.environment != "prod" || var.openai_api_key != ""
    error_message = "OpenAI API key is required in the prod environment. Please set openai_api_key variable."
  }
}

variable "postgres_password" {
  description = "Password for PostgreSQL database"
  type        = string
  sensitive   = true
  default     = ""
}

variable "typesense_url" {
  description = "Typesense URL (optional, defaults to localhost)"
  type        = string
  default     = ""
}
