package typesense

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/pkg/config"
)

func TestClient_Integration(t *testing.T) {
	// Skip if not running integration tests (optional, but good practice)
	// if os.Getenv("TEST_INTEGRATION") != "true" { t.Skip("Skipping integration test") }

	cfg := &config.Config{
		Typesense: config.TypesenseConfig{
			URL:    "http://localhost:8108",
			APIKey: "xyz",
		},
	}

	client, err := NewClient(&cfg.Typesense)
	assert.NoError(t, err)
	assert.NotNil(t, client)

	ctx := context.Background()

	// Test InitSchema
	err = client.InitSchema(ctx)
	assert.NoError(t, err)

	// Test Indexing
	doc := map[string]interface{}{
		"id":            "test-facility-1",
		"name":          "Test Facility",
		"facility_type": "Hospital",
		"location":      []float64{37.7749, -122.4194},
		"rating":        4.5,
		"created_at":    time.Now().Unix(),
		"review_count":  0,
		"is_active":     true,
	}
	err = client.IndexFacility(ctx, doc)
	assert.NoError(t, err)

	// Allow some time for indexing
	time.Sleep(1 * time.Second)

	// Test Search (optional validation)
	// We'd need to import the api package to construct search params,
	// or expose a simplified Search method.
}
