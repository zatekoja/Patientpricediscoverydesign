package providerapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Client interface {
	GetCurrentData(ctx context.Context, req CurrentDataRequest) (*CurrentDataResponse, error)
}

type HTTPClient struct {
	baseURL    string
	httpClient *http.Client
}

type CurrentDataRequest struct {
	ProviderID string
	Limit      int
	Offset     int
}

type CurrentDataResponse struct {
	Data      []PriceRecord        `json:"data"`
	Timestamp time.Time            `json:"timestamp"`
	Metadata  *CurrentDataMetadata `json:"metadata"`
}

type CurrentDataMetadata struct {
	Source string `json:"source"`
	Count  int    `json:"count"`
	Total  int    `json:"total"`
}

type PriceRecord struct {
	ID                   string    `json:"id"`
	FacilityName         string    `json:"facilityName"`
	ProcedureCode        string    `json:"procedureCode"`
	ProcedureDescription string    `json:"procedureDescription"`
	Price                float64   `json:"price"`
	Currency             string    `json:"currency"`
	EffectiveDate        time.Time `json:"effectiveDate"`
	LastUpdated          time.Time `json:"lastUpdated"`
	Source               string    `json:"source"`
	Tags                 []string  `json:"tags"`
}

func NewClient(baseURL string) *HTTPClient {
	trimmed := strings.TrimRight(baseURL, "/")
	return &HTTPClient{
		baseURL: trimmed,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (c *HTTPClient) GetCurrentData(ctx context.Context, req CurrentDataRequest) (*CurrentDataResponse, error) {
	endpoint := fmt.Sprintf("%s/data/current", c.baseURL)
	parsed, err := url.Parse(endpoint)
	if err != nil {
		return nil, err
	}

	query := parsed.Query()
	if req.ProviderID != "" {
		query.Set("providerId", req.ProviderID)
	}
	if req.Limit > 0 {
		query.Set("limit", fmt.Sprintf("%d", req.Limit))
	}
	if req.Offset > 0 {
		query.Set("offset", fmt.Sprintf("%d", req.Offset))
	}
	parsed.RawQuery = query.Encode()

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, parsed.String(), nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("provider api returned status %d", resp.StatusCode)
	}

	var decoded CurrentDataResponse
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return nil, err
	}

	return &decoded, nil
}
