package handlers_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/api/handlers"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/entities"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/repositories"
)

type MockFacilityService struct {
	mock.Mock
}

func (m *MockFacilityService) GetByID(ctx context.Context, id string) (*entities.Facility, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entities.Facility), args.Error(1)
}

func (m *MockFacilityService) List(ctx context.Context, filter repositories.FacilityFilter) ([]*entities.Facility, error) {
	args := m.Called(ctx, filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entities.Facility), args.Error(1)
}

func (m *MockFacilityService) Search(ctx context.Context, params repositories.SearchParams) ([]*entities.Facility, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entities.Facility), args.Error(1)
}

func (m *MockFacilityService) SearchResults(ctx context.Context, params repositories.SearchParams) ([]entities.FacilitySearchResult, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]entities.FacilitySearchResult), args.Error(1)
}

func (m *MockFacilityService) Suggest(ctx context.Context, query string, lat, lon float64, limit int) ([]*entities.Facility, error) {
	args := m.Called(ctx, query, lat, lon, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entities.Facility), args.Error(1)
}

type searchFacilitiesResponse struct {
	Facilities []entities.FacilitySearchResult `json:"facilities"`
	Count      int                             `json:"count"`
}

type suggestFacilitiesResponse struct {
	Suggestions []handlers.FacilitySuggestion `json:"suggestions"`
	Count       int                           `json:"count"`
}

func TestFacilityHandler_SearchFacilities_ReturnsContract(t *testing.T) {
	mockService := new(MockFacilityService)
	handler := handlers.NewFacilityHandler(mockService)

	now := time.Date(2026, 2, 7, 12, 30, 0, 0, time.UTC)
	next := now.Add(2 * time.Hour)

	expected := []entities.FacilitySearchResult{
		{
			ID:           "fac-1",
			Name:         "Lagos General Hospital",
			FacilityType: "Hospital",
			Address: entities.Address{
				Street:  "1 Marina",
				City:    "Lagos",
				State:   "Lagos",
				Country: "Nigeria",
			},
			Location: entities.Location{
				Latitude:  6.5244,
				Longitude: 3.3792,
			},
			PhoneNumber: "0800-000-0000",
			Website:     "https://lagos.example.com",
			Rating:      4.7,
			ReviewCount: 120,
			DistanceKm:  2.4,
			Price: &entities.FacilityPriceRange{
				Min:      15000,
				Max:      45000,
				Currency: "NGN",
			},
			Services: []string{"MRI", "CT Scan"},
			ServicePrices: []entities.ServicePrice{
				{
					ProcedureID: "proc-1",
					Name:        "MRI",
					Price:       15000,
					Currency:    "NGN",
				},
			},
			AcceptedInsurance: []string{"NHIS", "AXA"},
			NextAvailableAt:   &next,
			UpdatedAt:         now,
		},
	}

	mockService.On("SearchResults", mock.Anything, mock.MatchedBy(func(p repositories.SearchParams) bool {
		return p.Latitude == 6.5244 && p.Longitude == 3.3792 && p.RadiusKm == 20 && p.Limit == 5 && p.Offset == 10 && p.Query == "clinic"
	})).Return(expected, nil)

	req := httptest.NewRequest("GET", "/api/facilities/search?lat=6.5244&lon=3.3792&radius=20&limit=5&offset=10&query=clinic", nil)
	w := httptest.NewRecorder()

	handler.SearchFacilities(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp searchFacilitiesResponse
	err := json.NewDecoder(w.Body).Decode(&resp)
	assert.NoError(t, err)
	assert.Len(t, resp.Facilities, 1)
	assert.Equal(t, 1, resp.Count)
	assert.Equal(t, expected[0].ID, resp.Facilities[0].ID)
	assert.Equal(t, expected[0].Price.Currency, resp.Facilities[0].Price.Currency)
	assert.Equal(t, expected[0].AcceptedInsurance, resp.Facilities[0].AcceptedInsurance)
	assert.Equal(t, expected[0].Services, resp.Facilities[0].Services)
	assert.Equal(t, expected[0].ServicePrices, resp.Facilities[0].ServicePrices)
}

func TestFacilityHandler_SuggestFacilities_ReturnsServicePrices(t *testing.T) {
	mockService := new(MockFacilityService)
	handler := handlers.NewFacilityHandler(mockService)

	expected := []entities.FacilitySearchResult{
		{
			ID:           "fac-1",
			Name:         "Lagos General Hospital",
			FacilityType: "Hospital",
			Address: entities.Address{
				City:    "Lagos",
				State:   "Lagos",
				Country: "Nigeria",
			},
			Location: entities.Location{
				Latitude:  6.5244,
				Longitude: 3.3792,
			},
			Rating: 4.7,
			Price: &entities.FacilityPriceRange{
				Min:      15000,
				Max:      45000,
				Currency: "NGN",
			},
			ServicePrices: []entities.ServicePrice{
				{
					ProcedureID: "proc-1",
					Name:        "MRI",
					Price:       15000,
					Currency:    "NGN",
				},
			},
		},
	}

	mockService.On("SearchResults", mock.Anything, mock.MatchedBy(func(p repositories.SearchParams) bool {
		return p.Latitude == 6.5244 && p.Longitude == 3.3792 && p.RadiusKm == 50 && p.Limit == 3 && p.Query == "reliance"
	})).Return(expected, nil)

	req := httptest.NewRequest("GET", "/api/facilities/suggest?lat=6.5244&lon=3.3792&limit=3&query=reliance", nil)
	w := httptest.NewRecorder()

	handler.SuggestFacilities(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp suggestFacilitiesResponse
	err := json.NewDecoder(w.Body).Decode(&resp)
	assert.NoError(t, err)
	assert.Len(t, resp.Suggestions, 1)
	assert.Equal(t, expected[0].ID, resp.Suggestions[0].ID)
	assert.Equal(t, expected[0].ServicePrices[0].Name, resp.Suggestions[0].ServicePrices[0].Name)
	assert.Equal(t, expected[0].ServicePrices[0].Price, resp.Suggestions[0].ServicePrices[0].Price)
	assert.Equal(t, expected[0].Price.Currency, resp.Suggestions[0].Price.Currency)
}
