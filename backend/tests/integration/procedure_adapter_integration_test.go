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

type ProcedureAdapterIntegrationTestSuite struct {
	suite.Suite
	client        *postgres.Client
	procAdapter   repositories.ProcedureRepository
	fpAdapter     repositories.FacilityProcedureRepository
	db            *sql.DB
}

func (suite *ProcedureAdapterIntegrationTestSuite) SetupSuite() {
	cfg := &config.DatabaseConfig{
		Host:     getEnv("TEST_DB_HOST", "localhost"),
		Port:     getEnvAsInt("TEST_DB_PORT", 5432),
		User:     getEnv("TEST_DB_USER", "postgres"),
		Password: getEnv("TEST_DB_PASSWORD", "postgres"),
		Database: getEnv("TEST_DB_NAME", "patient_price_discovery_test"),
		SSLMode:  getEnv("TEST_DB_SSLMODE", "disable"),
	}

	client, err := postgres.NewClient(cfg)
	require.NoError(suite.T(), err)

	suite.client = client
	suite.db = client.DB()
	suite.procAdapter = database.NewProcedureAdapter(client)
	suite.fpAdapter = database.NewFacilityProcedureAdapter(client)

	suite.runMigrations()
}

func (suite *ProcedureAdapterIntegrationTestSuite) TearDownSuite() {
	if suite.client != nil {
		suite.client.Close()
	}
}

func (suite *ProcedureAdapterIntegrationTestSuite) SetupTest() {
	suite.cleanupTestData()
}

func (suite *ProcedureAdapterIntegrationTestSuite) TearDownTest() {
	suite.cleanupTestData()
}

func (suite *ProcedureAdapterIntegrationTestSuite) runMigrations() {
	migrationPath := "../../migrations/001_initial_schema.sql"
	migrationSQL, err := os.ReadFile(migrationPath)
	require.NoError(suite.T(), err)
	_, err = suite.db.Exec(string(migrationSQL))
	require.NoError(suite.T(), err)

	migrationPath2 := "../../migrations/002_add_service_normalization.sql"
	migrationSQL2, err := os.ReadFile(migrationPath2)
	require.NoError(suite.T(), err)
	_, err = suite.db.Exec(string(migrationSQL2))
	require.NoError(suite.T(), err)
}

func (suite *ProcedureAdapterIntegrationTestSuite) cleanupTestData() {
	tables := []string{"facility_procedures", "procedures", "facilities"}
	for _, table := range tables {
		_, err := suite.db.Exec(fmt.Sprintf("DELETE FROM %s", table))
		require.NoError(suite.T(), err)
	}
}

func (suite *ProcedureAdapterIntegrationTestSuite) TestProcedureCRUD() {
	ctx := context.Background()
	proc := &entities.Procedure{
		ID:          "proc-test-1",
		Name:        "Test Procedure",
		DisplayName: "Test Procedure",
		Code:        "99213",
		Category:    "Evaluation",
		Description: "Office visit",
		IsActive:    true,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	// Create
	err := suite.procAdapter.Create(ctx, proc)
	require.NoError(suite.T(), err)

	// GetByID
	got, err := suite.procAdapter.GetByID(ctx, proc.ID)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), proc.Name, got.Name)

	// GetByCode
	gotCode, err := suite.procAdapter.GetByCode(ctx, "99213")
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), proc.ID, gotCode.ID)

	// Update
	proc.Name = "Updated Procedure"
	err = suite.procAdapter.Update(ctx, proc)
	require.NoError(suite.T(), err)
	
	gotUpdated, err := suite.procAdapter.GetByID(ctx, proc.ID)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), "Updated Procedure", gotUpdated.Name)

	// List
	list, err := suite.procAdapter.List(ctx, repositories.ProcedureFilter{})
	require.NoError(suite.T(), err)
	assert.Len(suite.T(), list, 1)

	// Delete
	err = suite.procAdapter.Delete(ctx, proc.ID)
	require.NoError(suite.T(), err)
	
	// Should be soft deleted (not found by ID if GetByID filters active? 
	// Wait, GetByID in adapter usually filters by ID only unless specified. 
	// Let's check adapter implementation. It filters by ID only. 
	// But Delete sets is_active=false. 
	// Let's check if GetByID returns inactive.
	// The implementation of GetByID uses `Where(goqu.Ex{field: value})` - no active check.
	// So it should still return it but with IsActive=false.
	
	deleted, err := suite.procAdapter.GetByID(ctx, proc.ID)
	require.NoError(suite.T(), err)
	assert.False(suite.T(), deleted.IsActive)
}

func (suite *ProcedureAdapterIntegrationTestSuite) TestFacilityProcedureCRUD() {
	ctx := context.Background()

	// Setup: Need Facility and Procedure
	facID := "fac-fp-test"
	procID := "proc-fp-test"
	
	_, err := suite.db.Exec("INSERT INTO facilities (id, name, created_at, updated_at) VALUES ($1, 'Test Fac', NOW(), NOW())", facID)
	require.NoError(suite.T(), err)
	
	_, err = suite.db.Exec("INSERT INTO procedures (id, name, display_name, code, created_at, updated_at) VALUES ($1, 'Test Proc', 'Test Proc', 'TP002', NOW(), NOW())", procID)
	require.NoError(suite.T(), err)

	fp := &entities.FacilityProcedure{
		ID:                uuid.New().String(),
		FacilityID:        facID,
		ProcedureID:       procID,
		Price:             150.00,
		Currency:          "USD",
		EstimatedDuration: 30,
		IsAvailable:       true,
		CreatedAt:         time.Now().UTC(),
		UpdatedAt:         time.Now().UTC(),
	}

	// Create
	err = suite.fpAdapter.Create(ctx, fp)
	require.NoError(suite.T(), err)

	// Get
	got, err := suite.fpAdapter.GetByID(ctx, fp.ID)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), fp.Price, got.Price)

	// GetByFacilityAndProcedure
	got2, err := suite.fpAdapter.GetByFacilityAndProcedure(ctx, facID, procID)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), fp.ID, got2.ID)

	// ListByFacility
	list, err := suite.fpAdapter.ListByFacility(ctx, facID)
	require.NoError(suite.T(), err)
	assert.Len(suite.T(), list, 1)
}

func TestProcedureAdapterIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	suite.Run(t, new(ProcedureAdapterIntegrationTestSuite))
}
