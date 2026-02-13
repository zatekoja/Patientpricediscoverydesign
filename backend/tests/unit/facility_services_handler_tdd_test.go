package handlers_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/api/handlers"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/entities"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/repositories"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/tests/mocks"
)

// TDD Test: Verify that facility services endpoint implements search-across-all-data
func TestFacilityHandler_GetFacilityServices_TDD_SearchAcrossAllData(t *testing.T) {
	mockFacilityService := new(mocks.MockFacilityService)
	mockProcedureService := new(mocks.MockFacilityProcedureService)

	handler := handlers.NewFacilityHandlerWithServices(mockFacilityService, mockProcedureService)

	facilityID := "fac-123"

	// Mock data: simulates finding 25 total procedures with "MRI" in entire dataset
	// But only returning 10 due to pagination limit
	expectedProcedures := []*entities.FacilityProcedure{
		{
			ID:                "fp-1",
			FacilityID:        facilityID,
			ProcedureID:       "proc-mri-brain",
			Price:             400.0,
			Currency:          "NGN",
			EstimatedDuration: 45,
			IsAvailable:       true,
		},
		{
			ID:                "fp-2",
			FacilityID:        facilityID,
			ProcedureID:       "proc-mri-knee",
			Price:             350.0,
			Currency:          "NGN",
			EstimatedDuration: 30,
			IsAvailable:       true,
		},
		// ... 8 more procedures to make 10 total returned
	}

	// Critical TDD assertion: total count should be 25 (all matches), returned should be 10 (page limit)
	expectedTotalCount := 25
	expectedReturnedCount := len(expectedProcedures)

	// Set up mock expectation - verifies the filter is passed correctly
	mockProcedureService.On("ListByFacilityWithCount", mock.Anything, facilityID, mock.MatchedBy(func(filter repositories.FacilityProcedureFilter) bool {
		return filter.SearchQuery == "MRI" &&
			filter.Limit == 10 &&
			filter.Offset == 0 &&
			filter.SortBy == "price" &&
			filter.SortOrder == "asc"
	})).Return(expectedProcedures, expectedTotalCount, nil)

	// Create request with TDD parameters
	req := httptest.NewRequest("GET", "/api/facilities/fac-123/services?search=MRI&limit=10&offset=0&sort=price&order=asc", nil)
	req.SetPathValue("id", facilityID)
	w := httptest.NewRecorder()

	// Execute the handler
	handler.GetFacilityServices(w, req)

	// TDD Assertions
	require.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)

	// Critical assertion: total_count reflects search across ALL data
	assert.Equal(t, float64(expectedTotalCount), response["total_count"],
		"Total count must reflect search across ENTIRE dataset, not just current page")

	// Verify returned services respect pagination
	services, ok := response["services"].([]interface{})
	require.True(t, ok, "Services should be an array")
	assert.Equal(t, expectedReturnedCount, len(services),
		"Returned services should respect pagination limit")

	// Verify pagination metadata is correct
	assert.Equal(t, float64(1), response["current_page"])
	assert.Equal(t, float64(3), response["total_pages"]) // ceil(25/10) = 3
	assert.Equal(t, float64(10), response["page_size"])
	assert.Equal(t, true, response["has_next"])
	assert.Equal(t, false, response["has_prev"])

	// Verify filters are preserved in response
	filtersApplied, ok := response["filters_applied"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "MRI", filtersApplied["search"])
	assert.Equal(t, "price", filtersApplied["sort_by"])
	assert.Equal(t, "asc", filtersApplied["sort_order"])

	mockProcedureService.AssertExpectations(t)
}

func TestFacilityHandler_GetFacilityServices_TDD_ComplexFiltering(t *testing.T) {
	mockFacilityService := new(mocks.MockFacilityService)
	mockProcedureService := new(mocks.MockFacilityProcedureService)

	handler := handlers.NewFacilityHandlerWithServices(mockFacilityService, mockProcedureService)

	facilityID := "fac-456"

	// Test complex filtering: search + category + price range + availability
	expectedProcedures := []*entities.FacilityProcedure{
		{
			ID:                "fp-therapy-1",
			FacilityID:        facilityID,
			ProcedureID:       "proc-therapy-basic",
			Price:             125.0,
			Currency:          "NGN",
			EstimatedDuration: 60,
			IsAvailable:       true,
		},
	}

	// Mock expectation with complex filter validation
	mockProcedureService.On("ListByFacilityWithCount", mock.Anything, facilityID, mock.MatchedBy(func(filter repositories.FacilityProcedureFilter) bool {
		return filter.SearchQuery == "therapy" &&
			filter.Category == "therapeutic" &&
			filter.MinPrice != nil && *filter.MinPrice == 100.0 &&
			filter.MaxPrice != nil && *filter.MaxPrice == 200.0 &&
			filter.IsAvailable != nil && *filter.IsAvailable == true &&
			filter.Limit == 20 &&
			filter.Offset == 0
	})).Return(expectedProcedures, 1, nil)

	// Request with complex filters
	req := httptest.NewRequest("GET",
		"/api/facilities/fac-456/services?search=therapy&category=therapeutic&min_price=100&max_price=200&available=true",
		nil)
	req.SetPathValue("id", facilityID)
	w := httptest.NewRecorder()

	handler.GetFacilityServices(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)

	// Verify all filters were applied correctly
	assert.Equal(t, float64(1), response["total_count"])

	filtersApplied, ok := response["filters_applied"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "therapy", filtersApplied["search"])
	assert.Equal(t, "therapeutic", filtersApplied["category"])
	assert.Equal(t, float64(100), filtersApplied["min_price"])
	assert.Equal(t, float64(200), filtersApplied["max_price"])
	assert.Equal(t, true, filtersApplied["available"])

	mockProcedureService.AssertExpectations(t)
}

func TestFacilityHandler_GetFacilityServices_TDD_PaginationConsistency(t *testing.T) {
	mockFacilityService := new(mocks.MockFacilityService)
	mockProcedureService := new(mocks.MockFacilityProcedureService)

	handler := handlers.NewFacilityHandlerWithServices(mockFacilityService, mockProcedureService)

	facilityID := "fac-789"
	totalMatchingProcedures := 47 // Total procedures matching search
	pageSize := 15

	// Test second page
	page2Offset := 15
	expectedPage2Procedures := make([]*entities.FacilityProcedure, pageSize)
	for i := 0; i < pageSize; i++ {
		expectedPage2Procedures[i] = &entities.FacilityProcedure{
			ID:          fmt.Sprintf("fp-page2-%d", i),
			FacilityID:  facilityID,
			ProcedureID: fmt.Sprintf("proc-scan-%d", i+page2Offset),
			Price:       float64(100 + (i+page2Offset)*10),
			Currency:    "NGN",
			IsAvailable: true,
		}
	}

	mockProcedureService.On("ListByFacilityWithCount", mock.Anything, facilityID, mock.MatchedBy(func(filter repositories.FacilityProcedureFilter) bool {
		return filter.SearchQuery == "scan" &&
			filter.Limit == pageSize &&
			filter.Offset == page2Offset
	})).Return(expectedPage2Procedures, totalMatchingProcedures, nil)

	// Request second page
	req := httptest.NewRequest("GET",
		fmt.Sprintf("/api/facilities/fac-789/services?search=scan&limit=%d&offset=%d", pageSize, page2Offset),
		nil)
	req.SetPathValue("id", facilityID)
	w := httptest.NewRecorder()

	handler.GetFacilityServices(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)

	// Verify pagination metadata for second page
	assert.Equal(t, float64(totalMatchingProcedures), response["total_count"], "Total count should be consistent")
	assert.Equal(t, float64(2), response["current_page"], "Should be page 2")
	assert.Equal(t, float64(4), response["total_pages"], "ceil(47/15) = 4 total pages")
	assert.Equal(t, true, response["has_next"], "Should have next page")
	assert.Equal(t, true, response["has_prev"], "Should have previous page")

	services, ok := response["services"].([]interface{})
	require.True(t, ok)
	assert.Equal(t, pageSize, len(services), "Should return full page size")

	mockProcedureService.AssertExpectations(t)
}

func TestFacilityHandler_GetFacilityServices_ErrorHandling(t *testing.T) {
	tests := []struct {
		name           string
		url            string
		facilityID     string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "Missing facility ID",
			url:            "/api/facilities//services",
			facilityID:     "",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "facility ID is required",
		},
		{
			name:           "Invalid limit",
			url:            "/api/facilities/fac-123/services?limit=0",
			facilityID:     "fac-123",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "limit must be between 1 and 100",
		},
		{
			name:           "Limit too high",
			url:            "/api/facilities/fac-123/services?limit=150",
			facilityID:     "fac-123",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "limit must be between 1 and 100",
		},
		{
			name:           "Negative offset",
			url:            "/api/facilities/fac-123/services?offset=-5",
			facilityID:     "fac-123",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "offset must be non-negative",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockFacilityService := new(mocks.MockFacilityService)
			mockProcedureService := new(mocks.MockFacilityProcedureService)

			handler := handlers.NewFacilityHandlerWithServices(mockFacilityService, mockProcedureService)

			req := httptest.NewRequest("GET", tt.url, nil)
			if tt.facilityID != "" {
				req.SetPathValue("id", tt.facilityID)
			}
			w := httptest.NewRecorder()

			handler.GetFacilityServices(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedError != "" {
				var response map[string]interface{}
				err := json.NewDecoder(w.Body).Decode(&response)
				require.NoError(t, err)
				assert.Contains(t, response["error"], tt.expectedError)
			}
		})
	}
}
