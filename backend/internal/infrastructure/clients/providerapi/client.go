package providerapi

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Client interface {
	GetCurrentData(ctx context.Context, req CurrentDataRequest) (*CurrentDataResponse, error)
	GetPreviousData(ctx context.Context, req CurrentDataRequest) (*CurrentDataResponse, error)
	GetHistoricalData(ctx context.Context, req HistoricalDataRequest) (*CurrentDataResponse, error)
	GetProviderHealth(ctx context.Context, providerID string) (*ProviderHealthResponse, error)
	ListProviders(ctx context.Context) (*ProviderListResponse, error)
	TriggerSync(ctx context.Context, providerID string) (*SyncResponse, error)
	GetSyncStatus(ctx context.Context, providerID string) (*SyncResponse, error)
	GetFacilityProfile(ctx context.Context, facilityID string) (*FacilityProfile, error)
	ListFacilityProfiles(ctx context.Context, limit, offset int) ([]FacilityProfile, error)
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
	Source  string `json:"source"`
	Count   int    `json:"count"`
	Total   int    `json:"total"`
	BatchID string `json:"batchId,omitempty"`
	Type    string `json:"type,omitempty"`
	Message string `json:"message,omitempty"`
	HasMore bool   `json:"hasMore,omitempty"`
}

type PriceRecord struct {
	ID                   string    `json:"id"`
	FacilityName         string    `json:"facilityName"`
	FacilityID           string    `json:"facilityId"`
	ProcedureCode        string    `json:"procedureCode"`
	ProcedureDescription string    `json:"procedureDescription"`
	ProcedureCategory    string    `json:"procedureCategory"`
	ProcedureDetails     string    `json:"procedureDetails"`
	Price                float64   `json:"price"`
	Currency             string    `json:"currency"`
	EstimatedDurationMin *int      `json:"estimatedDurationMinutes"`
	EffectiveDate        time.Time `json:"effectiveDate"`
	LastUpdated          time.Time `json:"lastUpdated"`
	Source               string    `json:"source"`
	Tags                 []string  `json:"tags"`
}

type FacilityProfile struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	FacilityType string   `json:"facilityType"`
	Description  string   `json:"description"`
	Tags         []string `json:"tags"`
	CapacityStatus *string `json:"capacityStatus"`
	AvgWaitMinutes *int    `json:"avgWaitMinutes"`
	UrgentCareAvailable *bool `json:"urgentCareAvailable"`
	Address      struct {
		Street  string `json:"street"`
		City    string `json:"city"`
		State   string `json:"state"`
		ZipCode string `json:"zipCode"`
		Country string `json:"country"`
	} `json:"address"`
	Location struct {
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
	} `json:"location"`
	PhoneNumber string    `json:"phoneNumber"`
	Email       string    `json:"email"`
	Website     string    `json:"website"`
	LastUpdated time.Time `json:"lastUpdated"`
	Source      string    `json:"source"`
}

type HistoricalDataRequest struct {
	ProviderID string
	TimeWindow string
	StartDate  *time.Time
	EndDate    *time.Time
	Limit      int
	Offset     int
}

type ProviderHealthResponse struct {
	Healthy  bool      `json:"healthy"`
	LastSync time.Time `json:"lastSync,omitempty"`
	Message  string    `json:"message,omitempty"`
}

type ProviderInfo struct {
	ID       string    `json:"id"`
	Name     string    `json:"name"`
	Type     string    `json:"type"`
	Healthy  bool      `json:"healthy"`
	LastSync time.Time `json:"lastSync,omitempty"`
}

type ProviderListResponse struct {
	Providers []ProviderInfo `json:"providers"`
}

type SyncResponse struct {
	Success          bool      `json:"success"`
	RecordsProcessed int       `json:"recordsProcessed"`
	Timestamp        time.Time `json:"timestamp"`
	Error            string    `json:"error,omitempty"`
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
	return c.getData(ctx, "/data/current", req)
}

func (c *HTTPClient) GetPreviousData(ctx context.Context, req CurrentDataRequest) (*CurrentDataResponse, error) {
	return c.getData(ctx, "/data/previous", req)
}

func (c *HTTPClient) GetHistoricalData(ctx context.Context, req HistoricalDataRequest) (*CurrentDataResponse, error) {
	parsed, err := url.Parse(fmt.Sprintf("%s/data/historical", c.baseURL))
	if err != nil {
		return nil, err
	}

	query := parsed.Query()
	if req.ProviderID != "" {
		query.Set("providerId", req.ProviderID)
	}
	if req.TimeWindow != "" {
		query.Set("timeWindow", req.TimeWindow)
	}
	if req.StartDate != nil {
		query.Set("startDate", req.StartDate.Format(time.RFC3339))
	}
	if req.EndDate != nil {
		query.Set("endDate", req.EndDate.Format(time.RFC3339))
	}
	if req.Limit > 0 {
		query.Set("limit", fmt.Sprintf("%d", req.Limit))
	}
	if req.Offset > 0 {
		query.Set("offset", fmt.Sprintf("%d", req.Offset))
	}
	parsed.RawQuery = query.Encode()

	out := &CurrentDataResponse{}
	if err := c.doJSON(ctx, http.MethodGet, parsed.String(), nil, out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *HTTPClient) GetProviderHealth(ctx context.Context, providerID string) (*ProviderHealthResponse, error) {
	endpoint := fmt.Sprintf("%s/provider/health", c.baseURL)
	if providerID != "" {
		endpoint = fmt.Sprintf("%s?providerId=%s", endpoint, url.QueryEscape(providerID))
	}
	out := &ProviderHealthResponse{}
	if err := c.doJSON(ctx, http.MethodGet, endpoint, nil, out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *HTTPClient) ListProviders(ctx context.Context) (*ProviderListResponse, error) {
	endpoint := fmt.Sprintf("%s/provider/list", c.baseURL)
	out := &ProviderListResponse{}
	if err := c.doJSON(ctx, http.MethodGet, endpoint, nil, out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *HTTPClient) TriggerSync(ctx context.Context, providerID string) (*SyncResponse, error) {
	endpoint := fmt.Sprintf("%s/sync/trigger", c.baseURL)
	if providerID != "" {
		endpoint = fmt.Sprintf("%s?providerId=%s", endpoint, url.QueryEscape(providerID))
	}
	out := &SyncResponse{}
	if err := c.doJSON(ctx, http.MethodPost, endpoint, nil, out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *HTTPClient) GetSyncStatus(ctx context.Context, providerID string) (*SyncResponse, error) {
	endpoint := fmt.Sprintf("%s/sync/status", c.baseURL)
	if providerID != "" {
		endpoint = fmt.Sprintf("%s?providerId=%s", endpoint, url.QueryEscape(providerID))
	}
	out := &SyncResponse{}
	if err := c.doJSON(ctx, http.MethodGet, endpoint, nil, out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *HTTPClient) GetFacilityProfile(ctx context.Context, facilityID string) (*FacilityProfile, error) {
	if strings.TrimSpace(facilityID) == "" {
		return nil, fmt.Errorf("facility id is required")
	}
	endpoint := fmt.Sprintf("%s/facilities/%s", c.baseURL, url.PathEscape(facilityID))
	out := &FacilityProfile{}
	if err := c.doJSON(ctx, http.MethodGet, endpoint, nil, out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *HTTPClient) ListFacilityProfiles(ctx context.Context, limit, offset int) ([]FacilityProfile, error) {
	parsed, err := url.Parse(fmt.Sprintf("%s/facilities", c.baseURL))
	if err != nil {
		return nil, err
	}
	query := parsed.Query()
	if limit > 0 {
		query.Set("limit", fmt.Sprintf("%d", limit))
	}
	if offset > 0 {
		query.Set("offset", fmt.Sprintf("%d", offset))
	}
	parsed.RawQuery = query.Encode()
	var response struct {
		Data []FacilityProfile `json:"data"`
	}
	if err := c.doJSON(ctx, http.MethodGet, parsed.String(), nil, &response); err != nil {
		return nil, err
	}
	return response.Data, nil
}

func (c *HTTPClient) getData(ctx context.Context, path string, req CurrentDataRequest) (*CurrentDataResponse, error) {
	parsed, err := url.Parse(fmt.Sprintf("%s%s", c.baseURL, path))
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

	out := &CurrentDataResponse{}
	if err := c.doJSON(ctx, http.MethodGet, parsed.String(), nil, out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *HTTPClient) doJSON(ctx context.Context, method, endpoint string, body io.Reader, out interface{}) error {
	httpReq, err := http.NewRequestWithContext(ctx, method, endpoint, body)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("provider api returned status %d", resp.StatusCode)
	}

	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return err
	}

	return nil
}
