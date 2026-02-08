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

// FacilityAdapterIntegrationTestSuite defines the test suite for facility adapter
type FacilityAdapterIntegrationTestSuite struct {
	suite.Suite
	client  *postgres.Client
	adapter repositories.FacilityRepository
	db      *sql.DB
}

// SetupSuite runs once before the suite
func (suite *FacilityAdapterIntegrationTestSuite) SetupSuite() {
	// Load test database configuration
	cfg := &config.DatabaseConfig{
		Host:     getEnv("TEST_DB_HOST", "localhost"),
		Port:     getEnvAsInt("TEST_DB_PORT", 5432),
		User:     getEnv("TEST_DB_USER", "postgres"),
		Password: getEnv("TEST_DB_PASSWORD", "postgres"),
		Database: getEnv("TEST_DB_NAME", "patient_price_discovery_test"),
		SSLMode:  getEnv("TEST_DB_SSLMODE", "disable"),
	}

	// Create database client
	client, err := postgres.NewClient(cfg)
	require.NoError(suite.T(), err, "Failed to create postgres client")

	suite.client = client
	suite.db = client.DB()
	suite.adapter = database.NewFacilityAdapter(client)

	// Run migrations
	suite.runMigrations()
}

// TearDownSuite runs once after the suite
func (suite *FacilityAdapterIntegrationTestSuite) TearDownSuite() {
	if suite.client != nil {
		suite.client.Close()
	}
}

// SetupTest runs before each test
func (suite *FacilityAdapterIntegrationTestSuite) SetupTest() {
	// Clean up test data before each test
	suite.cleanupTestData()
}

// TearDownTest runs after each test
func (suite *FacilityAdapterIntegrationTestSuite) TearDownTest() {
	// Clean up test data after each test
	suite.cleanupTestData()
}

// runMigrations executes the database schema
func (suite *FacilityAdapterIntegrationTestSuite) runMigrations() {
	// Read and execute migration file
	migrationPath := "../../migrations/001_initial_schema.sql"
	migrationSQL, err := os.ReadFile(migrationPath)
	require.NoError(suite.T(), err, "Failed to read migration file")

	_, err = suite.db.Exec(string(migrationSQL))
	require.NoError(suite.T(), err, "Failed to execute migrations")
}

// cleanupTestData removes all test data from tables
func (suite *FacilityAdapterIntegrationTestSuite) cleanupTestData() {
	// Delete in reverse order of dependencies
	tables := []string{
		"reviews",
		"availability_slots",
		"appointments",
		"facility_insurance",
		"insurance_providers",
		"facility_procedures",
		"procedures",
		"facilities",
		"users",
	}

	for _, table := range tables {
		_, err := suite.db.Exec(fmt.Sprintf("DELETE FROM %s", table))
		require.NoError(suite.T(), err, fmt.Sprintf("Failed to clean up %s table", table))
	}
}

// TestCreate tests creating a facility
func (suite *FacilityAdapterIntegrationTestSuite) TestCreate() {
	ctx := context.Background()
	facility := &entities.Facility{
		ID:   "test-facility-1",
		Name: "Test Hospital",
		Address: entities.Address{
			Street:  "123 Test St",
			City:    "Test City",
			State:   "TS",
			ZipCode: "12345",
			Country: "USA",
		},
		Location: entities.Location{
			Latitude:  37.7749,
			Longitude: -122.4194,
		},
		PhoneNumber:  "+1234567890",
		Email:        "test@hospital.com",
		FacilityType: "hospital",
		Rating:       4.5,
		ReviewCount:  100,
		IsActive:     true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	// Act
	err := suite.adapter.Create(ctx, facility)

	// Assert
	require.NoError(suite.T(), err)

	// Verify the facility was created
	retrieved, err := suite.adapter.GetByID(ctx, facility.ID)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), facility.ID, retrieved.ID)
	assert.Equal(suite.T(), facility.Name, retrieved.Name)
	assert.Equal(suite.T(), facility.Address.City, retrieved.Address.City)
	assert.Equal(suite.T(), facility.Location.Latitude, retrieved.Location.Latitude)
}

// TestGetByID tests retrieving a facility by ID
func (suite *FacilityAdapterIntegrationTestSuite) TestGetByID() {
	ctx := context.Background()

	// Arrange - create a test facility
	facility := suite.createTestFacility("get-test-1", "Get Test Hospital")

	// Act
	retrieved, err := suite.adapter.GetByID(ctx, facility.ID)

	// Assert
	require.NoError(suite.T(), err)
	assert.NotNil(suite.T(), retrieved)
	assert.Equal(suite.T(), facility.ID, retrieved.ID)
	assert.Equal(suite.T(), facility.Name, retrieved.Name)
}

// TestGetByID_NotFound tests getting a non-existent facility
func (suite *FacilityAdapterIntegrationTestSuite) TestGetByID_NotFound() {
	ctx := context.Background()

	// Act
	retrieved, err := suite.adapter.GetByID(ctx, "non-existent-id")

	// Assert
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), retrieved)
}

// TestUpdate tests updating a facility
func (suite *FacilityAdapterIntegrationTestSuite) TestUpdate() {
	ctx := context.Background()

	// Arrange - create a test facility
	facility := suite.createTestFacility("update-test-1", "Original Name")

	// Act - update the facility
	facility.Name = "Updated Name"
	facility.Rating = 4.8
	err := suite.adapter.Update(ctx, facility)

	// Assert
	require.NoError(suite.T(), err)

	// Verify the update
	retrieved, err := suite.adapter.GetByID(ctx, facility.ID)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), "Updated Name", retrieved.Name)
	assert.Equal(suite.T(), 4.8, retrieved.Rating)
}

// TestDelete tests soft deleting a facility
func (suite *FacilityAdapterIntegrationTestSuite) TestDelete() {
	ctx := context.Background()

	// Arrange - create a test facility
	facility := suite.createTestFacility("delete-test-1", "Delete Test Hospital")

	// Act
	err := suite.adapter.Delete(ctx, facility.ID)

	// Assert
	require.NoError(suite.T(), err)

	// Verify it's soft deleted (not returned by GetByID which filters active)
	retrieved, err := suite.adapter.GetByID(ctx, facility.ID)
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), retrieved)
}

// TestList tests listing facilities with filters
func (suite *FacilityAdapterIntegrationTestSuite) TestList() {
	ctx := context.Background()

	// Arrange - create multiple test facilities
	suite.createTestFacility("list-test-1", "Hospital 1")
	suite.createTestFacility("list-test-2", "Hospital 2")
	suite.createTestFacility("list-test-3", "Hospital 3")

	// Act
	filter := repositories.FacilityFilter{
		Limit:  10,
		Offset: 0,
	}
	facilities, err := suite.adapter.List(ctx, filter)

	// Assert
	require.NoError(suite.T(), err)
	assert.GreaterOrEqual(suite.T(), len(facilities), 3)
}

// TestSearch tests searching facilities by location
func (suite *FacilityAdapterIntegrationTestSuite) TestSearch() {
	ctx := context.Background()

	// Arrange - create facilities at different locations
	// San Francisco area
	f1 := suite.createTestFacility("search-test-1", "SF Hospital")
	f1.Location.Latitude = 37.7749
	f1.Location.Longitude = -122.4194
	suite.adapter.Update(ctx, f1)

	// Los Angeles area (far from SF)
	f2 := suite.createTestFacility("search-test-2", "LA Hospital")
	f2.Location.Latitude = 34.0522
	f2.Location.Longitude = -118.2437
	suite.adapter.Update(ctx, f2)

	// Act - search near San Francisco
	params := repositories.SearchParams{
		Latitude:  37.7749,
		Longitude: -122.4194,
		RadiusKm:  50.0, // 50km radius
		Limit:     10,
		Offset:    0,
	}
	facilities, err := suite.adapter.Search(ctx, params)

	// Assert
	require.NoError(suite.T(), err)
	// Should find SF hospital but not LA hospital
	assert.GreaterOrEqual(suite.T(), len(facilities), 1)

	// Verify SF hospital is in results
	found := false
	for _, f := range facilities {
		if f.ID == f1.ID {
			found = true
			break
		}
	}
	assert.True(suite.T(), found, "SF hospital should be in search results")
}

// TestNullableFields tests handling of nullable fields
func (suite *FacilityAdapterIntegrationTestSuite) TestNullableFields() {
	ctx := context.Background()

	// Arrange - create facility with minimal required fields
	facility := &entities.Facility{
		ID:      "nullable-test-1",
		Name:    "Minimal Hospital",
		Address: entities.Address{
			// Leave nullable fields empty
		},
		Location: entities.Location{
			Latitude:  37.7749,
			Longitude: -122.4194,
		},
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Act
	err := suite.adapter.Create(ctx, facility)
	require.NoError(suite.T(), err)

	// Retrieve and verify nullable fields are handled correctly
	retrieved, err := suite.adapter.GetByID(ctx, facility.ID)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), "", retrieved.Address.Street)
	assert.Equal(suite.T(), "", retrieved.PhoneNumber)
	assert.Equal(suite.T(), "", retrieved.Email)
}

// createTestFacility is a helper to create a test facility
func (suite *FacilityAdapterIntegrationTestSuite) createTestFacility(id, name string) *entities.Facility {
	ctx := context.Background()
	facility := &entities.Facility{
		ID:   id,
		Name: name,
		Address: entities.Address{
			Street:  "123 Test St",
			City:    "Test City",
			State:   "TS",
			ZipCode: "12345",
			Country: "USA",
		},
		Location: entities.Location{
			Latitude:  37.7749,
			Longitude: -122.4194,
		},
		PhoneNumber:  "+1234567890",
		Email:        "test@hospital.com",
		FacilityType: "hospital",
		Rating:       4.5,
		ReviewCount:  100,
		IsActive:     true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	err := suite.adapter.Create(ctx, facility)
	require.NoError(suite.T(), err)
	return facility
}

// Helper functions
// TestFacilityAdapterIntegration runs the test suite
func TestFacilityAdapterIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	suite.Run(t, new(FacilityAdapterIntegrationTestSuite))
}
