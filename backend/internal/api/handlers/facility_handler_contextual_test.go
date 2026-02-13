package handlers_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/api/handlers"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/application/services"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/entities"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/repositories"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/evaluation"
)

func TestSearchWithConceptualQuery_ReturnsInterpretation(t *testing.T) {
	mockService := new(MockFacilityService)
	handler := handlers.NewFacilityHandler(mockService)

	query := "tooth ache"

	expectedInterpretation := &services.QueryInterpretation{
		OriginalQuery:   query,
		NormalizedQuery: "tooth ache",
		DetectedIntent:  evaluation.IntentSymptom,
		ExpandedTerms:   []string{"tooth", "ache", "dental"},
	}

	expectedFacilities := []entities.FacilitySearchResult{
		{ID: "fac-1", Name: "Dental Clinic"},
	}

	mockService.On("ExpandQuery", query).Return([]string{"tooth", "ache"})
	mockService.On("SearchResultsWithCount", mock.Anything, mock.MatchedBy(func(p repositories.SearchParams) bool {
		return p.Query == query
	})).Return(expectedFacilities, 1, expectedInterpretation, nil)

	req := httptest.NewRequest("GET", "/api/facilities/search?lat=6.5244&lon=3.3792&query=tooth+ache", nil)
	w := httptest.NewRecorder()

	handler.SearchFacilities(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.NewDecoder(w.Body).Decode(&resp)
	assert.NoError(t, err)

	assert.NotNil(t, resp["query_interpretation"])
	interp := resp["query_interpretation"].(map[string]interface{})
	assert.Equal(t, query, interp["original_query"])
	assert.Equal(t, string(evaluation.IntentSymptom), interp["detected_intent"])
}
