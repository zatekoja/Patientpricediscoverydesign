//go:build integration

package integration

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/adapters/database"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/entities"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/repositories"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/infrastructure/clients/postgres"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/pkg/config"
)

type AppointmentAdapterIntegrationTestSuite struct {
	suite.Suite
	client  *postgres.Client
	adapter repositories.AppointmentRepository
	db      *sql.DB
}

func (suite *AppointmentAdapterIntegrationTestSuite) SetupSuite() {
	cfg := &config.DatabaseConfig{
		Host:     getEnv("TEST_DB_HOST", "localhost"),
		Port:     getEnvAsInt("TEST_DB_PORT", 5432),
		User:     getEnv("TEST_DB_USER", "postgres"),
		Password: getEnv("TEST_DB_PASSWORD", "postgres"),
		Database: getEnv("TEST_DB_NAME", "patient_price_discovery_test"),
		SSLMode:  getEnv("TEST_DB_SSLMODE", "disable"),
	}

	client, err := postgres.NewClient(cfg)
	require.NoError(suite.T(), err, "Failed to create postgres client")

	suite.client = client
	suite.db = client.DB()
	suite.adapter = database.NewAppointmentAdapter(client)

	suite.runMigrations()
}

func (suite *AppointmentAdapterIntegrationTestSuite) TearDownSuite() {
	if suite.client != nil {
		suite.client.Close()
	}
}

func (suite *AppointmentAdapterIntegrationTestSuite) SetupTest() {
	suite.cleanupTestData()
	// Need reference data (Facilities, Procedures, Users) for foreign keys
	suite.seedReferenceData()
}

func (suite *AppointmentAdapterIntegrationTestSuite) TearDownTest() {
	suite.cleanupTestData()
}

func (suite *AppointmentAdapterIntegrationTestSuite) runMigrations() {
	migrationPath := "../../migrations/001_initial_schema.sql"
	migrationSQL, err := os.ReadFile(migrationPath)
	require.NoError(suite.T(), err)
	_, err = suite.db.Exec(string(migrationSQL))
	require.NoError(suite.T(), err)
}

func (suite *AppointmentAdapterIntegrationTestSuite) cleanupTestData() {
	tables := []string{
		"appointments",
		"users",
		"facility_procedures",
		"procedures",
		"facilities",
	}
	for _, table := range tables {
		_, err := suite.db.Exec(fmt.Sprintf("DELETE FROM %s", table))
		require.NoError(suite.T(), err)
	}
}

func (suite *AppointmentAdapterIntegrationTestSuite) seedReferenceData() {
	// 1. Create Facility
	_, err := suite.db.Exec(`
		INSERT INTO facilities (id, name, is_active, created_at, updated_at)
		VALUES ('test-fac-1', 'Integration Test Facility', true, NOW(), NOW())
	`)
	require.NoError(suite.T(), err)

	// 2. Create Procedure
	_, err = suite.db.Exec(`
		INSERT INTO procedures (id, name, code, is_active, created_at, updated_at)
		VALUES ('test-proc-1', 'Test Procedure', 'TP001', true, NOW(), NOW())
	`)
	require.NoError(suite.T(), err)

	// 3. Create User
	_, err = suite.db.Exec(`
		INSERT INTO users (id, email, first_name, last_name, created_at, updated_at)
		VALUES ('test-user-1', 'test@user.com', 'Test', 'User', NOW(), NOW())
	`)
	require.NoError(suite.T(), err)
}

func (suite *AppointmentAdapterIntegrationTestSuite) TestCreateAndGet() {
	ctx := context.Background()
	appointment := &entities.Appointment{
		ID:           uuid.New().String(),
		FacilityID:   "test-fac-1",
		ProcedureID:  "test-proc-1",
		UserID:       stringPointer("test-user-1"),
		ScheduledAt:  time.Now().Add(24 * time.Hour).UTC(),
		Status:       entities.AppointmentStatusPending,
		PatientName:  "Test Patient",
		PatientEmail: "patient@example.com",
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}

	// Create
	err := suite.adapter.Create(ctx, appointment)
	require.NoError(suite.T(), err)

	// Get
	retrieved, err := suite.adapter.GetByID(ctx, appointment.ID)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), appointment.ID, retrieved.ID)
	assert.Equal(suite.T(), appointment.PatientName, retrieved.PatientName)
	assert.WithinDuration(suite.T(), appointment.ScheduledAt, retrieved.ScheduledAt, time.Second)
}

func (suite *AppointmentAdapterIntegrationTestSuite) TestUpdate() {
	ctx := context.Background()
	appointment := &entities.Appointment{
		ID:           uuid.New().String(),
		FacilityID:   "test-fac-1",
		ProcedureID:  "test-proc-1",
		ScheduledAt:  time.Now().Add(48 * time.Hour).UTC(),
		Status:       entities.AppointmentStatusPending,
		PatientName:  "Update Test",
		PatientEmail: "update@example.com",
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}

	err := suite.adapter.Create(ctx, appointment)
	require.NoError(suite.T(), err)

	// Update
	appointment.Status = entities.AppointmentStatusConfirmed
	appointment.Notes = "Updated notes"
	err = suite.adapter.Update(ctx, appointment)
	require.NoError(suite.T(), err)

	// Verify
	retrieved, err := suite.adapter.GetByID(ctx, appointment.ID)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), entities.AppointmentStatusConfirmed, retrieved.Status)
	assert.Equal(suite.T(), "Updated notes", retrieved.Notes)
}

func (suite *AppointmentAdapterIntegrationTestSuite) TestListByFacility() {
	ctx := context.Background()
	// Create multiple appointments
	for i := 0; i < 3; i++ {
		appt := &entities.Appointment{
			ID:           uuid.New().String(),
			FacilityID:   "test-fac-1",
			ProcedureID:  "test-proc-1",
			ScheduledAt:  time.Now().Add(time.Duration(i+1) * 24 * time.Hour).UTC(),
			Status:       entities.AppointmentStatusPending,
			PatientName:  fmt.Sprintf("Patient %d", i),
			PatientEmail: fmt.Sprintf("patient%d@example.com", i),
			CreatedAt:    time.Now().UTC(),
			UpdatedAt:    time.Now().UTC(),
		}
		require.NoError(suite.T(), suite.adapter.Create(ctx, appt))
	}

	// List
	filter := repositories.AppointmentFilter{
		Limit: 10,
	}
	results, err := suite.adapter.ListByFacility(ctx, "test-fac-1", filter)
	require.NoError(suite.T(), err)
	assert.Len(suite.T(), results, 3)
}

func stringPointer(s string) *string {
	return &s
}

func TestAppointmentAdapterIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	suite.Run(t, new(AppointmentAdapterIntegrationTestSuite))
}
