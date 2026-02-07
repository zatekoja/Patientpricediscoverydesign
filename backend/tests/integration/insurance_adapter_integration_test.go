//go:build integration

package integration

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

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

type InsuranceAdapterIntegrationTestSuite struct {
	suite.Suite
	client  *postgres.Client
	adapter repositories.InsuranceRepository
	db      *sql.DB
}

func (suite *InsuranceAdapterIntegrationTestSuite) SetupSuite() {
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
	suite.adapter = database.NewInsuranceAdapter(client)

	suite.runMigrations()
}

func (suite *InsuranceAdapterIntegrationTestSuite) TearDownSuite() {
	if suite.client != nil {
		suite.client.Close()
	}
}

func (suite *InsuranceAdapterIntegrationTestSuite) SetupTest() {
	suite.cleanupTestData()
}

func (suite *InsuranceAdapterIntegrationTestSuite) TearDownTest() {
	suite.cleanupTestData()
}

func (suite *InsuranceAdapterIntegrationTestSuite) runMigrations() {
	migrationPath := "../../migrations/001_initial_schema.sql"
	migrationSQL, err := os.ReadFile(migrationPath)
	require.NoError(suite.T(), err)
	_, err = suite.db.Exec(string(migrationSQL))
	require.NoError(suite.T(), err)
}

func (suite *InsuranceAdapterIntegrationTestSuite) cleanupTestData() {
	tables := []string{"facility_insurance", "insurance_providers", "facilities"}
	for _, table := range tables {
		_, err := suite.db.Exec(fmt.Sprintf("DELETE FROM %s", table))
		require.NoError(suite.T(), err)
	}
}

func (suite *InsuranceAdapterIntegrationTestSuite) TestInsuranceCRUD() {
	ctx := context.Background()
	provider := &entities.InsuranceProvider{
		ID:          "ins-test-1",
		Name:        "Test Insurance",
		Code:        "TI001",
		PhoneNumber: "1-800-TEST",
		Website:     "https://test.com",
		IsActive:    true,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	// Create
	err := suite.adapter.Create(ctx, provider)
	require.NoError(suite.T(), err)

	// GetByID
	got, err := suite.adapter.GetByID(ctx, provider.ID)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), provider.Name, got.Name)

	// GetByCode
	gotCode, err := suite.adapter.GetByCode(ctx, "TI001")
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), provider.ID, gotCode.ID)

	// Update
	provider.Name = "Updated Insurance"
	err = suite.adapter.Update(ctx, provider)
	require.NoError(suite.T(), err)

	gotUpdated, err := suite.adapter.GetByID(ctx, provider.ID)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), "Updated Insurance", gotUpdated.Name)

	// List
	list, err := suite.adapter.List(ctx, repositories.InsuranceFilter{})
	require.NoError(suite.T(), err)
	assert.Len(suite.T(), list, 1)

	// Delete
	err = suite.adapter.Delete(ctx, provider.ID)
	require.NoError(suite.T(), err)

	deleted, err := suite.adapter.GetByID(ctx, provider.ID)
	require.NoError(suite.T(), err)
	assert.False(suite.T(), deleted.IsActive)
}

func (suite *InsuranceAdapterIntegrationTestSuite) TestGetFacilityInsurance() {
	ctx := context.Background()

	// Setup: Facility and Insurance + Link
	facID := "fac-ins-test"
	insID := "ins-rel-test"

	_, err := suite.db.Exec("INSERT INTO facilities (id, name, created_at, updated_at) VALUES ($1, 'Test Fac', NOW(), NOW())", facID)
	require.NoError(suite.T(), err)

	_, err = suite.db.Exec("INSERT INTO insurance_providers (id, name, code, is_active, created_at, updated_at) VALUES ($1, 'Linked Insurance', 'LI001', true, NOW(), NOW())", insID)
	require.NoError(suite.T(), err)

	_, err = suite.db.Exec("INSERT INTO facility_insurance (id, facility_id, insurance_provider_id, is_accepted, created_at, updated_at) VALUES ($1, $2, $3, true, NOW(), NOW())", "rel-1", facID, insID)
	require.NoError(suite.T(), err)

	// Test
	providers, err := suite.adapter.GetFacilityInsurance(ctx, facID)
	require.NoError(suite.T(), err)
	assert.Len(suite.T(), providers, 1)
	assert.Equal(suite.T(), insID, providers[0].ID)
}

func TestInsuranceAdapterIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	suite.Run(t, new(InsuranceAdapterIntegrationTestSuite))
}
