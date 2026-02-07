//go:build integration

package integration

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/api/handlers"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/infrastructure/clients/providerapi"
)

func TestProviderRestCurrentDataIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	baseURL := getEnv("PROVIDER_API_BASE_URL", "http://localhost:3002/api/v1")
	providerID := getEnv("PROVIDER_ID", "file_price_list")

	waitForProviderHealthy(t, baseURL)

	client := providerapi.NewClient(baseURL)
	handler := handlers.NewProviderPriceHandler(client)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/provider/prices/current", handler.GetCurrentData)

	server := httptest.NewServer(mux)
	defer server.Close()

	endpoint := fmt.Sprintf("%s/api/provider/prices/current?providerId=%s&limit=5", server.URL, providerID)
	resp, err := http.Get(endpoint)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	var payload providerapi.CurrentDataResponse
	err = json.NewDecoder(resp.Body).Decode(&payload)
	require.NoError(t, err)
	require.NotEmpty(t, payload.Data, "expected provider data in REST response")
	require.NotEmpty(t, payload.Timestamp, "expected timestamp in REST response")
}
