//go:build integration

package integration

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/adapters/database"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/adapters/events"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/application/services"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/entities"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/providers"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/infrastructure/clients/postgres"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/infrastructure/clients/redis"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/pkg/config"
)

func TestRedisEventBusFanoutIntegration(t *testing.T) {
	if os.Getenv("TEST_REDIS_HOST") == "" {
		t.Skip("Skipping integration test: TEST_REDIS_HOST not set")
	}

	redisClient := newTestRedisClient(t)
	defer redisClient.Close()

	eventBus := events.NewRedisEventBus(redisClient)
	defer eventBus.Close()

	channel := providers.EventChannelFacilityUpdates
	ctx1, cancel1 := context.WithCancel(context.Background())
	ctx2, cancel2 := context.WithCancel(context.Background())
	defer cancel1()
	defer cancel2()

	sub1, err := eventBus.Subscribe(ctx1, channel)
	require.NoError(t, err)
	sub2, err := eventBus.Subscribe(ctx2, channel)
	require.NoError(t, err)
	time.Sleep(50 * time.Millisecond)

	event := entities.NewFacilityEvent(
		"fac-redis-1",
		entities.FacilityEventTypeCapacityUpdate,
		entities.Location{Latitude: 6.5244, Longitude: 3.3792},
		map[string]interface{}{"capacity_status": "high"},
	)

	err = eventBus.Publish(context.Background(), channel, event)
	require.NoError(t, err)

	received1 := waitForFacilityEvent(t, sub1)
	received2 := waitForFacilityEvent(t, sub2)

	assert.Equal(t, event.ID, received1.ID)
	assert.Equal(t, event.ID, received2.ID)
}

func TestFacilityService_UpdateServiceAvailability_PublishesEvent(t *testing.T) {
	if os.Getenv("TEST_DB_HOST") == "" || os.Getenv("TEST_REDIS_HOST") == "" {
		t.Skip("Skipping integration test: TEST_DB_HOST or TEST_REDIS_HOST not set")
	}

	dbClient := newTestPostgresClient(t)
	defer dbClient.Close()

	db := dbClient.DB()
	runMigrations(t, db,
		"../../migrations/001_initial_schema.sql",
		"../../migrations/003_add_capacity_fields.sql",
	)
	cleanupFacilityAvailabilityData(t, db)
	seedFacilityAvailabilityData(t, db)

	redisClient := newTestRedisClient(t)
	defer redisClient.Close()

	eventBus := events.NewRedisEventBus(redisClient)
	defer eventBus.Close()

	facilityRepo := database.NewFacilityAdapter(dbClient)
	facilityProcedureRepo := database.NewFacilityProcedureAdapter(dbClient)
	procedureRepo := database.NewProcedureAdapter(dbClient)

	service := services.NewFacilityService(facilityRepo, nil, facilityProcedureRepo, procedureRepo, nil)
	service.SetEventBus(eventBus)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	channel := providers.GetFacilityChannel("fac-avail-1")
	eventChan, err := eventBus.Subscribe(ctx, channel)
	require.NoError(t, err)
	time.Sleep(50 * time.Millisecond)

	fp, err := service.UpdateServiceAvailability(ctx, "fac-avail-1", "proc-avail-1", true)
	require.NoError(t, err)
	assert.True(t, fp.IsAvailable)

	received := waitForFacilityEvent(t, eventChan)
	assert.Equal(t, entities.FacilityEventTypeServiceAvailabilityUpdate, received.EventType)
	assert.Equal(t, "fac-avail-1", received.FacilityID)
	assert.Equal(t, "proc-avail-1", received.ChangedFields["procedure_id"])
	assert.Equal(t, true, received.ChangedFields["is_available"])

	cleanupFacilityAvailabilityData(t, db)
}

func newTestRedisClient(t *testing.T) *redis.Client {
	t.Helper()

	cfg := &config.RedisConfig{
		Host: getEnv("TEST_REDIS_HOST", "localhost"),
		Port: getEnvAsInt("TEST_REDIS_PORT", 6379),
		DB:   0,
	}

	client, err := redis.NewClient(cfg)
	require.NoError(t, err, "Failed to create redis client")
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

func runMigrations(t *testing.T, db *sql.DB, paths ...string) {
	t.Helper()
	for _, path := range paths {
		migrationSQL, err := os.ReadFile(path)
		require.NoError(t, err)
		_, err = db.Exec(string(migrationSQL))
		require.NoError(t, err)
	}
}

func cleanupFacilityAvailabilityData(t *testing.T, db *sql.DB) {
	t.Helper()
	tables := []string{
		"facility_procedures",
		"procedures",
		"facilities",
	}
	for _, table := range tables {
		_, err := db.Exec("DELETE FROM " + table)
		require.NoError(t, err)
	}
}

func seedFacilityAvailabilityData(t *testing.T, db *sql.DB) {
	t.Helper()

	_, err := db.Exec(`
		INSERT INTO facilities (id, name, latitude, longitude, is_active, created_at, updated_at)
		VALUES ('fac-avail-1', 'Availability Test Facility', 6.5244, 3.3792, true, NOW(), NOW())
	`)
	require.NoError(t, err)

	_, err = db.Exec(`
		INSERT INTO procedures (id, name, code, category, description, is_active, created_at, updated_at)
		VALUES ('proc-avail-1', 'MRI Scan', 'MRI001', 'Imaging', 'MRI description', true, NOW(), NOW())
	`)
	require.NoError(t, err)

	_, err = db.Exec(`
		INSERT INTO facility_procedures (id, facility_id, procedure_id, price, currency, estimated_duration, is_available, created_at, updated_at)
		VALUES ('fp-avail-1', 'fac-avail-1', 'proc-avail-1', 12000, 'NGN', 45, false, NOW(), NOW())
	`)
	require.NoError(t, err)
}

func waitForFacilityEvent(t *testing.T, ch <-chan *entities.FacilityEvent) *entities.FacilityEvent {
	t.Helper()
	select {
	case event := <-ch:
		require.NotNil(t, event)
		return event
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for facility event")
		return nil
	}
}
