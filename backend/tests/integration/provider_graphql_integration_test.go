//go:build integration

package integration

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/99designs/gqlgen/client"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/stretchr/testify/require"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/graphql/generated"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/graphql/resolvers"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/infrastructure/clients/providerapi"
)

type providerHealthResponse struct {
	Status string `json:"status"`
}

func waitForProviderHealthy(t *testing.T, baseURL string) {
	t.Helper()

	client := &http.Client{Timeout: 3 * time.Second}
	deadline := time.Now().Add(45 * time.Second)
	url := baseURL + "/health"
	var lastStatus string
	var lastErr string

	for time.Now().Before(deadline) {
		req, err := http.NewRequest(http.MethodGet, url, nil)
		require.NoError(t, err)

		resp, err := client.Do(req)
		if err != nil {
			lastErr = err.Error()
		} else if resp != nil {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()

			if resp.StatusCode == http.StatusOK {
				var payload providerHealthResponse
				if decodeErr := json.Unmarshal(body, &payload); decodeErr == nil && payload.Status == "ok" {
					return
				}

				if bytes.Contains(body, []byte(`"status":"ok"`)) {
					return
				}
			}

			lastStatus = string(body)
		}

		time.Sleep(1 * time.Second)
	}

	if lastStatus != "" {
		t.Fatalf("provider API not healthy at %s (last response: %s)", url, lastStatus)
	}
	if lastErr != "" {
		t.Fatalf("provider API not healthy at %s (last error: %s)", url, lastErr)
	}
	t.Fatalf("provider API not healthy at %s", url)
}

func TestProviderPriceCurrentGraphQLIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	baseURL := getEnv("PROVIDER_API_BASE_URL", "http://localhost:3002/api/v1")
	providerID := getEnv("PROVIDER_ID", "file_price_list")

	waitForProviderHealthy(t, baseURL)

	providerClient := providerapi.NewClient(baseURL)
	resolver := resolvers.NewResolver(nil, nil, nil, nil, nil, nil, nil, providerClient)
	srv := handler.NewDefaultServer(generated.NewExecutableSchema(generated.Config{
		Resolvers: resolver,
	}))

	gql := client.New(srv)

	var resp struct {
		ProviderPriceCurrent struct {
			Data []struct {
				ID                   string   `json:"id"`
				FacilityName         string   `json:"facilityName"`
				ProcedureCode        string   `json:"procedureCode"`
				ProcedureDescription string   `json:"procedureDescription"`
				Price                float64  `json:"price"`
				Currency             string   `json:"currency"`
				EffectiveDate        string   `json:"effectiveDate"`
				LastUpdated          string   `json:"lastUpdated"`
				Source               string   `json:"source"`
				Tags                 []string `json:"tags"`
			} `json:"data"`
			Timestamp string `json:"timestamp"`
		} `json:"providerPriceCurrent"`
	}

	query := `
		query ProviderPriceCurrent($providerId: String, $limit: Int) {
			providerPriceCurrent(providerId: $providerId, limit: $limit) {
				data {
					id
					facilityName
					procedureCode
					procedureDescription
					price
					currency
					effectiveDate
					lastUpdated
					source
					tags
				}
				timestamp
			}
		}
	`

	gql.MustPost(query, &resp,
		client.Var("providerId", providerID),
		client.Var("limit", 5),
	)

	require.NotEmpty(t, resp.ProviderPriceCurrent.Timestamp, "timestamp should be set")
	require.NotEmpty(t, resp.ProviderPriceCurrent.Data, "expected provider data in GraphQL response")

	record := resp.ProviderPriceCurrent.Data[0]
	require.NotEmpty(t, record.ID, "record id should be set")
	require.NotEmpty(t, record.FacilityName, "facility name should be set")
	require.NotEmpty(t, record.ProcedureDescription, "procedure description should be set")
	require.NotNil(t, record.Tags, "tags should be present in GraphQL response")
}
