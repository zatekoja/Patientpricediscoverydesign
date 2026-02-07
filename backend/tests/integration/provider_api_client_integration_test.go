//go:build integration

package integration

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/infrastructure/clients/providerapi"
)

func TestProviderAPIClientCurrentData(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	baseURL := getEnv("PROVIDER_API_BASE_URL", "http://localhost:3002/api/v1")
	providerID := getEnv("PROVIDER_ID", "file_price_list")
	waitForProviderHealthy(t, baseURL)

	client := providerapi.NewClient(baseURL)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := client.GetCurrentData(ctx, providerapi.CurrentDataRequest{
		ProviderID: providerID,
		Limit:      5,
		Offset:     0,
	})
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotEmpty(t, resp.Data, "expected provider data")

	record := resp.Data[0]
	require.NotEmpty(t, record.ID)
	require.NotEmpty(t, record.FacilityName)
	require.NotEmpty(t, record.ProcedureDescription)
	require.NotNil(t, record.Tags)
}
