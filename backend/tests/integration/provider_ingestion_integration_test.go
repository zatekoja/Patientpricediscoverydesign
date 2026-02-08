//go:build integration

package integration

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/adapters/database"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/application/services"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/repositories"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/infrastructure/clients/postgres"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/infrastructure/clients/providerapi"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/pkg/config"
)

func TestProviderIngestionServiceIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	baseURL := getEnv("PROVIDER_API_BASE_URL", "http://localhost:3002/api/v1")
	providerID := getEnv("PROVIDER_ID", "file_price_list")
	waitForProviderHealthy(t, baseURL)

	cfg := &config.DatabaseConfig{
		Host:     getEnv("TEST_DB_HOST", "localhost"),
		Port:     getEnvAsInt("TEST_DB_PORT", 5432),
		User:     getEnv("TEST_DB_USER", "postgres"),
		Password: getEnv("TEST_DB_PASSWORD", "postgres"),
		Database: getEnv("TEST_DB_NAME", "patient_price_discovery_test"),
		SSLMode:  getEnv("TEST_DB_SSLMODE", "disable"),
	}

	client, err := postgres.NewClient(cfg)
	require.NoError(t, err)
	defer client.Close()

	runAllMigrations(t, client)
	cleanupProviderIngestionTables(t, client)

	facilityRepo := database.NewFacilityAdapter(client)
	facilityWardRepo := database.NewFacilityWardAdapter(client)
	procedureRepo := database.NewProcedureAdapter(client)
	facilityProcedureRepo := database.NewFacilityProcedureAdapter(client)

	facilityService := services.NewFacilityService(
		facilityRepo,
		nil,
		facilityProcedureRepo,
		procedureRepo,
		nil,
	)

	providerClient := providerapi.NewClient(baseURL)
	enrichmentAdapter := database.NewProcedureEnrichmentAdapter(client)
	facilityWardRepo := database.NewFacilityWardAdapter(client)
	ingestion := services.NewProviderIngestionService(
		providerClient,
		facilityRepo,
		facilityWardRepo,
		facilityService,
		procedureRepo,
		facilityProcedureRepo,
		enrichmentAdapter,
		nil, // no enrichment provider for test - enrichment will be skipped
		nil, // no geolocation provider
		nil, // no cache provider
		200,
	)

	summary, err := ingestion.SyncCurrentData(context.Background(), providerID)
	require.NoError(t, err)
	require.Greater(t, summary.RecordsProcessed, 0)

	facilities, err := facilityRepo.List(context.Background(), repositories.FacilityFilter{})
	require.NoError(t, err)
	require.NotEmpty(t, facilities)

	procedures, err := procedureRepo.List(context.Background(), repositories.ProcedureFilter{})
	require.NoError(t, err)
	require.NotEmpty(t, procedures)
}

func runAllMigrations(t *testing.T, client *postgres.Client) {
	t.Helper()

	migrationDir := filepath.Join("..", "..", "migrations")
	entries, err := os.ReadDir(migrationDir)
	require.NoError(t, err)

	files := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if filepath.Ext(entry.Name()) != ".sql" {
			continue
		}
		files = append(files, filepath.Join(migrationDir, entry.Name()))
	}
	sort.Strings(files)

	for _, file := range files {
		sqlBytes, err := os.ReadFile(file)
		require.NoError(t, err)
		_, err = client.DB().Exec(string(sqlBytes))
		require.NoError(t, err)
	}
}

func cleanupProviderIngestionTables(t *testing.T, client *postgres.Client) {
	t.Helper()

	tables := []string{
		"facility_procedures",
		"procedures",
		"facilities",
	}
	for _, table := range tables {
		_, err := client.DB().Exec("DELETE FROM " + table)
		require.NoError(t, err)
	}
}
