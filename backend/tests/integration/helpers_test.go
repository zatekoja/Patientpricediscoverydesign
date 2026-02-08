//go:build integration

package integration

import (
	"os"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/infrastructure/clients/postgres"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/infrastructure/clients/redis"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/pkg/config"
)

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

func newTestRedisClient(t *testing.T) *redis.Client {
	t.Helper()

	cfg := &config.RedisConfig{
		Host:     getEnv("TEST_REDIS_HOST", "localhost"),
		Port:     getEnvAsInt("TEST_REDIS_PORT", 6379),
		Password: getEnv("TEST_REDIS_PASSWORD", ""),
		DB:       getEnvAsInt("TEST_REDIS_DB", 0),
	}

	client, err := redis.NewClient(cfg)
	require.NoError(t, err, "Failed to create redis client")
	return client
}

func maybeTestRedisClient(t *testing.T) *redis.Client {
	t.Helper()

	cfg := &config.RedisConfig{
		Host:     getEnv("TEST_REDIS_HOST", "localhost"),
		Port:     getEnvAsInt("TEST_REDIS_PORT", 6379),
		Password: getEnv("TEST_REDIS_PASSWORD", ""),
		DB:       getEnvAsInt("TEST_REDIS_DB", 0),
	}

	client, err := redis.NewClient(cfg)
	if err != nil {
		t.Logf("Redis unavailable: %v", err)
		return nil
	}
	return client
}

func newTestPostgresClient(t *testing.T) *postgres.Client {
	t.Helper()

	cfg := &config.DatabaseConfig{
		Host:     getEnv("TEST_DB_HOST", "localhost"),
		Port:     getEnvAsInt("TEST_DB_PORT", 5432),
		User:     getEnv("TEST_DB_USER", "postgres"),
		Password: getEnv("TEST_DB_PASSWORD", "postgres"),
		Database: getEnv("TEST_DB_NAME", "patient_price_discovery_test"),
		SSLMode:  getEnv("TEST_DB_SSLMODE", "disable"),
	}

	client, err := postgres.NewClient(cfg)
	require.NoError(t, err, "Failed to create postgres client")
	return client
}
