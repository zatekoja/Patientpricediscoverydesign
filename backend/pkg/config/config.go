package config

import (
	"fmt"
	"os"
	"strconv"
)

// Config holds all application configuration
type Config struct {
	Server      ServerConfig
	Database    DatabaseConfig
	Redis       RedisConfig
	Typesense   TypesenseConfig
	Geolocation GeolocationConfig
	OpenAI      OpenAIConfig
	OTEL        OTELConfig
}

// ServerConfig holds server configuration
type ServerConfig struct {
	Host string
	Port int
}

// DatabaseConfig holds database configuration
type DatabaseConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	Database string
	SSLMode  string
}

// RedisConfig holds Redis configuration
type RedisConfig struct {
	Host     string
	Port     int
	Password string
	DB       int
}

// TypesenseConfig holds Typesense configuration
type TypesenseConfig struct {
	URL    string
	APIKey string
}

// GeolocationConfig holds geolocation provider configuration
type GeolocationConfig struct {
	Provider string
	APIKey   string
}

// OpenAIConfig holds OpenAI configuration
type OpenAIConfig struct {
	APIKey string
	Model  string
}

// OTELConfig holds OpenTelemetry configuration
type OTELConfig struct {
	ServiceName    string
	ServiceVersion string
	Endpoint       string
	Enabled        bool
}

// Load loads configuration from environment variables
func Load() (*Config, error) {
	return &Config{
		Server: ServerConfig{
			Host: getEnv("SERVER_HOST", "0.0.0.0"),
			Port: getEnvAsInt("SERVER_PORT", 8080),
		},
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnvAsInt("DB_PORT", 5432),
			User:     getEnv("DB_USER", "postgres"),
			Password: getEnv("DB_PASSWORD", ""),
			Database: getEnv("DB_NAME", "patient_price_discovery"),
			SSLMode:  getEnv("DB_SSLMODE", "disable"),
		},
		Redis: RedisConfig{
			Host:     getEnv("REDIS_HOST", "localhost"),
			Port:     getEnvAsInt("REDIS_PORT", 6379),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getEnvAsInt("REDIS_DB", 0),
		},
		Typesense: TypesenseConfig{
			URL:    getEnv("TYPESENSE_URL", "http://localhost:8108"),
			APIKey: getEnv("TYPESENSE_API_KEY", "xyz"),
		},
		Geolocation: GeolocationConfig{
			Provider: getEnv("GEOLOCATION_PROVIDER", "mock"),
			APIKey:   getEnv("GEOLOCATION_API_KEY", ""),
		},
		OpenAI: OpenAIConfig{
			APIKey: getEnv("OPENAI_API_KEY", ""),
			Model:  getEnv("OPENAI_MODEL", "gpt-4o-mini"),
		},
		OTEL: OTELConfig{
			ServiceName:    getEnv("OTEL_SERVICE_NAME", "patient-price-discovery"),
			ServiceVersion: getEnv("OTEL_SERVICE_VERSION", "1.0.0"),
			Endpoint:       getEnv("OTEL_ENDPOINT", ""),
			Enabled:        getEnvAsBool("OTEL_ENABLED", false),
		},
	}, nil
}

// DatabaseDSN returns the PostgreSQL connection string
func (c *DatabaseConfig) DatabaseDSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.Database, c.SSLMode,
	)
}

// RedisAddr returns the Redis address
func (c *RedisConfig) RedisAddr() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}

func getEnvAsBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolVal, err := strconv.ParseBool(value); err == nil {
			return boolVal
		}
	}
	return defaultValue
}
