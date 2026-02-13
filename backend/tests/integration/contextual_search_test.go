//go:build integration

package integration

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/adapters/database"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/application/services"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/entities"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/repositories"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/infrastructure/clients/postgres"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/pkg/config"
)

type ContextualSearchTestSuite struct {
	suite.Suite
	client      *postgres.Client
	fpAdapter   repositories.FacilityProcedureRepository
	procAdapter repositories.ProcedureRepository
	facAdapter  repositories.FacilityRepository
	termService *services.TermExpansionService
}

func (suite *ContextualSearchTestSuite) SetupSuite() {
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

	suite.fpAdapter = database.NewFacilityProcedureAdapter(client)
	suite.procAdapter = database.NewProcedureAdapter(client)
	suite.facAdapter = database.NewFacilityAdapter(client)

	// Create a temp config file for term expansion
	content := `{"baby": ["maternity", "delivery"]}`
	tmpFile, err := os.CreateTemp("", "terms.json")
	require.NoError(suite.T(), err)

	_, err = tmpFile.WriteString(content)
	require.NoError(suite.T(), err)
	tmpFile.Close()

	suite.termService, err = services.NewTermExpansionService(tmpFile.Name())
	require.NoError(suite.T(), err)

	os.Remove(tmpFile.Name()) // Clean up file

	// Drop tables to ensure clean slate for migrations
	_, err = suite.client.DB().Exec("DROP TABLE IF EXISTS facility_procedures CASCADE")
	require.NoError(suite.T(), err)
	_, err = suite.client.DB().Exec("DROP TABLE IF EXISTS procedures CASCADE")
	require.NoError(suite.T(), err)
	_, err = suite.client.DB().Exec("DROP TABLE IF EXISTS facilities CASCADE")
	require.NoError(suite.T(), err)
	_, err = suite.client.DB().Exec("DROP TABLE IF EXISTS insurance_providers CASCADE")
	require.NoError(suite.T(), err)
	_, err = suite.client.DB().Exec("DROP TABLE IF EXISTS facility_insurance CASCADE")
	require.NoError(suite.T(), err)
	_, err = suite.client.DB().Exec("DROP TABLE IF EXISTS users CASCADE")
	require.NoError(suite.T(), err)
	_, err = suite.client.DB().Exec("DROP TABLE IF EXISTS appointments CASCADE")
	require.NoError(suite.T(), err)
	_, err = suite.client.DB().Exec("DROP TABLE IF EXISTS availability_slots CASCADE")
	require.NoError(suite.T(), err)
	_, err = suite.client.DB().Exec("DROP TABLE IF EXISTS reviews CASCADE")
	require.NoError(suite.T(), err)

	suite.runMigrations()
}

func (suite *ContextualSearchTestSuite) runMigrations() {
	files, err := os.ReadDir("../../migrations")
	require.NoError(suite.T(), err)

	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".sql") {
			// Skip seed data as it might conflict with tests
			if strings.Contains(file.Name(), "seed") {
				continue
			}

			migrationSQL, err := os.ReadFile("../../migrations/" + file.Name())
			require.NoError(suite.T(), err)
			_, err = suite.client.DB().Exec(string(migrationSQL))
			require.NoError(suite.T(), err)
		}
	}
}

func (suite *ContextualSearchTestSuite) TearDownSuite() {
	if suite.client != nil {
		// Clean up schema so other tests are not affected by our migrations
		suite.client.DB().Exec("DROP SCHEMA public CASCADE; CREATE SCHEMA public;")
		suite.client.Close()
	}
}

func (suite *ContextualSearchTestSuite) SetupTest() {
	suite.client.DB().Exec("DELETE FROM facility_procedures")
	suite.client.DB().Exec("DELETE FROM procedures")
	suite.client.DB().Exec("DELETE FROM facilities")
}

func (suite *ContextualSearchTestSuite) TestContextualSearch() {
	ctx := context.Background()

	// 1. Create Procedure "Normal Delivery" with tag "maternity"
	proc := &entities.Procedure{
		ID:             "proc-delivery",
		Name:           "Normal Delivery",
		DisplayName:    "Normal Delivery",
		Code:           "ND001",
		NormalizedTags: []string{"delivery", "maternity", "birth"},
		IsActive:       true,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	require.NoError(suite.T(), suite.procAdapter.Create(ctx, proc))

	// 2. Create Facility
	fac := &entities.Facility{
		ID:        "fac-main",
		Name:      "Main Hospital",
		IsActive:  true,
		Location:  entities.Location{Latitude: 0, Longitude: 0},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	require.NoError(suite.T(), suite.facAdapter.Create(ctx, fac))

	// 3. Link them
	fp := &entities.FacilityProcedure{
		ID:          "fp-1",
		FacilityID:  fac.ID,
		ProcedureID: proc.ID,
		Price:       1000,
		IsAvailable: true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	require.NoError(suite.T(), suite.fpAdapter.Create(ctx, fp))

	// 4. Search for "baby"
	// "baby" expands to ["baby", "maternity", "delivery"]
	query := "baby"
	expanded := suite.termService.Expand(query)

	// Verify expansion works
	assert.Contains(suite.T(), expanded, "maternity")

	filter := repositories.FacilityProcedureFilter{
		SearchQuery: query,
		SearchTerms: expanded,
	}

	results, count, err := suite.fpAdapter.ListByFacilityWithCount(ctx, fac.ID, filter)
	require.NoError(suite.T(), err)

	// Should find it because "maternity" or "delivery" is in normalized tags
	assert.Equal(suite.T(), 1, count)
	if assert.Len(suite.T(), results, 1) {
		assert.Equal(suite.T(), "Normal Delivery", results[0].ProcedureName)
	}

	// 5. Search for "fracture" (unrelated)
	// "fracture" -> ["fracture"]
	query2 := "fracture"
	expanded2 := suite.termService.Expand(query2)

	filter2 := repositories.FacilityProcedureFilter{
		SearchQuery: query2,
		SearchTerms: expanded2,
	}

	results2, count2, err := suite.fpAdapter.ListByFacilityWithCount(ctx, fac.ID, filter2)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), 0, count2)
	assert.Empty(suite.T(), results2)
}

func TestContextualSearchTestSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	suite.Run(t, new(ContextualSearchTestSuite))
}
