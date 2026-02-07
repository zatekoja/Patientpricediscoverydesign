output "postgres_connection_name" {
  description = "Connection name for Cloud SQL PostgreSQL"
  value       = google_sql_database_instance.postgres.connection_name
}

output "postgres_instance_name" {
  description = "Name of the PostgreSQL instance"
  value       = google_sql_database_instance.postgres.name
}

output "postgres_database_name" {
  description = "Name of the PostgreSQL database"
  value       = google_sql_database.main.name
}

output "postgres_private_ip" {
  description = "Private IP address of PostgreSQL instance"
  value       = google_sql_database_instance.postgres.private_ip_address
}

output "redis_host" {
  description = "Redis instance host"
  value       = google_redis_instance.cache.host
}

output "redis_port" {
  description = "Redis instance port"
  value       = google_redis_instance.cache.port
}

output "redis_instance_name" {
  description = "Name of the Redis instance"
  value       = google_redis_instance.cache.name
}

output "postgres_password_secret_id" {
  description = "Secret Manager secret ID for PostgreSQL password"
  value       = google_secret_manager_secret.postgres_password.secret_id
}
