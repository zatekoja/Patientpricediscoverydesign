//go:build integration

package integration

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/adapters/search"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/entities"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/repositories"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/infrastructure/clients/typesense"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/pkg/config"
)

func TestTypesenseAdapter(t *testing.T) {
	// Skip if not running integration tests
	if os.Getenv("TEST_DB_HOST") == "" {
		t.Skip("Skipping integration test: TEST_DB_HOST not set")
	}

	// Config
	cfg := &config.TypesenseConfig{
		URL:    "http://localhost:8109", // Port mapped in docker-compose.test.yml
		APIKey: "xyz",
	}

	// Client
	client, err := typesense.NewClient(cfg)
	require.NoError(t, err)

	// Adapter
	adapter := search.NewTypesenseAdapter(client)

	// Context
	ctx := context.Background()

	// Ensure clean collection for schema updates
	_, _ = client.Client().Collection(typesense.FacilitiesCollection).Delete(ctx)

	// 1. Init Schema
	err = adapter.InitSchema(ctx)
	require.NoError(t, err)

	// 2. Index Facility
	facility := &entities.Facility{
		ID:           "test-facility-ts-1",
		Name:         "Typesense Search Hospital",
		FacilityType: "hospital",
		IsActive:     true,
		Address: entities.Address{
			City:    "Ikeja",
			State:   "Lagos",
			Country: "Nigeria",
		},
		AcceptedInsurance: []string{"NHIS"},
		Tags:              []string{"mri", "cardiology"},
		Location: entities.Location{
			Latitude:  37.7749,
			Longitude: -122.4194,
		},
		Rating:      4.9,
		ReviewCount: 50,
		CreatedAt:   time.Now(),
	}

	err = adapter.Index(ctx, facility)
	require.NoError(t, err)

	// Allow Typesense to index
	time.Sleep(1 * time.Second)

	// 3. Search
	params := repositories.SearchParams{
		Latitude:  37.7749,
		Longitude: -122.4194,
		RadiusKm:  10,
		Limit:     10,
		Offset:    0,
	}

	results, err := adapter.Search(ctx, params)
	require.NoError(t, err)
	assert.NotEmpty(t, results)
	assert.Equal(t, facility.ID, results[0].ID)
	assert.Equal(t, facility.Name, results[0].Name)

	// 3b. Search by tag (city)
	tagParams := repositories.SearchParams{
		Query:     "ikeja",
		Latitude:  37.7749,
		Longitude: -122.4194,
		RadiusKm:  10,
		Limit:     10,
		Offset:    0,
	}

	tagResults, err := adapter.Search(ctx, tagParams)
	require.NoError(t, err)
	assert.True(t, containsFacility(tagResults, facility.ID))

	// 3c. Suggest by tag
	suggestions, err := adapter.Suggest(ctx, "ikeja", 37.7749, -122.4194, 5)
	require.NoError(t, err)
	assert.True(t, containsFacility(suggestions, facility.ID))

	// 3c.1 Search by custom tag
	customTagParams := repositories.SearchParams{
		Query:     "mri",
		Latitude:  37.7749,
		Longitude: -122.4194,
		RadiusKm:  10,
		Limit:     10,
		Offset:    0,
	}

	customTagResults, err := adapter.Search(ctx, customTagParams)
	require.NoError(t, err)
	assert.True(t, containsFacility(customTagResults, facility.ID))

	customTagSuggestions, err := adapter.Suggest(ctx, "mri", 37.7749, -122.4194, 5)
	require.NoError(t, err)
	assert.True(t, containsFacility(customTagSuggestions, facility.ID))

	// 3d. Suggest with typo tolerance
	typoSuggestions, err := adapter.Suggest(ctx, "ikejx", 37.7749, -122.4194, 5)
	require.NoError(t, err)
	assert.True(t, containsFacility(typoSuggestions, facility.ID))

	// 4. Delete
	err = adapter.Delete(ctx, facility.ID)
	require.NoError(t, err)
}

func containsFacility(results []*entities.Facility, id string) bool {
	for _, facility := range results {
		if facility != nil && facility.ID == id {
			return true
		}
	}
	return false
}
