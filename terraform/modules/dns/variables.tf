variable "project_id" {
  description = "GCP Project ID"
  type        = string
}

variable "domain_name" {
  description = "Domain name for the application"
  type        = string
}

variable "environment" {
  description = "Environment name"
  type        = string
}

variable "load_balancer_ip" {
  description = "Load balancer IP address for DNS records"
  type        = string
  default     = ""
}
