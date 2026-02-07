package resolvers

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/entities"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/repositories"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/graphql/generated"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/tests/mocks"
)

// TestQueryResolver_SearchFacilities_Success tests successful facility search with query
func TestQueryResolver_SearchFacilities_Success(t *testing.T) {
	// Arrange
	mockSearch := mocks.NewMockSearchAdapter(t)
	mockDB := mocks.NewMockFacilityRepository(t)
	mockAppt := mocks.NewMockAppointmentRepository(t)
	mockProc := mocks.NewMockProcedureRepository(t)
	mockFacProc := mocks.NewMockFacilityProcedureRepository(t)
	mockIns := mocks.NewMockInsuranceRepository(t)
	mockCache := mocks.NewMockQueryCacheProvider(t)

	resolver := NewResolver(mockSearch, mockDB, mockAppt, mockProc, mockFacProc, mockIns, mockCache, nil)
	queryResolver := resolver.Query()

	ctx := context.Background()
	query := "ct scan"
	location := generated.LocationInput{
		Latitude:  37.7749,
		Longitude: -122.4194,
	}
	radiusKm := 10.0

	expectedFacilities := []*entities.Facility{
		{
			ID:           "fac-1",
			Name:         "Imaging Center",
			FacilityType: "imaging_center",
			Location: entities.Location{
				Latitude:  37.7749,
				Longitude: -122.4194,
			},
			Rating:      4.8,
			ReviewCount: 150,
			IsActive:    true,
		},
	}

	mockSearch.EXPECT().Search(ctx, repositories.SearchParams{
		Query:     query,
		Latitude:  location.Latitude,
		Longitude: location.Longitude,
		RadiusKm:  radiusKm,
		Limit:     20,
		Offset:    0,
	}).Return(expectedFacilities, nil)

	// Act
	result, err := queryResolver.SearchFacilities(ctx, query, location, &radiusKm, nil)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result.FacilitiesData, 1)
	assert.Equal(t, "fac-1", result.FacilitiesData[0].ID)
	assert.Equal(t, 1, result.TotalCountValue)
}

// TestQueryResolver_SearchFacilities_WithFilters tests search with additional filters
func TestQueryResolver_SearchFacilities_WithFilters(t *testing.T) {
	// Arrange
	mockSearch := mocks.NewMockSearchAdapter(t)
	mockDB := mocks.NewMockFacilityRepository(t)
	mockAppt := mocks.NewMockAppointmentRepository(t)
	mockProc := mocks.NewMockProcedureRepository(t)
	mockFacProc := mocks.NewMockFacilityProcedureRepository(t)
	mockIns := mocks.NewMockInsuranceRepository(t)
	mockCache := mocks.NewMockQueryCacheProvider(t)

	resolver := NewResolver(mockSearch, mockDB, mockAppt, mockProc, mockFacProc, mockIns, mockCache, nil)
	queryResolver := resolver.Query()

	ctx := context.Background()
	query := "hospital"
	location := generated.LocationInput{
		Latitude:  37.7749,
		Longitude: -122.4194,
	}
	radiusKm := 5.0
	limit := 10
	filters := &generated.FacilitySearchInput{
		Location:  &location,
		RadiusKm:  radiusKm,
		Limit:     &limit,
		MinRating: floatPtr(4.0),
	}

	expectedFacilities := []*entities.Facility{
		{
			ID:       "fac-1",
			Name:     "Central Hospital",
			Rating:   4.5,
			IsActive: true,
		},
	}

	mockSearch.EXPECT().Search(ctx, repositories.SearchParams{
		Query:     query,
		Latitude:  location.Latitude,
		Longitude: location.Longitude,
		RadiusKm:  radiusKm,
		Limit:     10,
		Offset:    0,
	}).Return(expectedFacilities, nil)

	// Act
	result, err := queryResolver.SearchFacilities(ctx, query, location, &radiusKm, filters)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result.FacilitiesData, 1)
}

// TestQueryResolver_SearchFacilities_NoResults tests search with no results
func TestQueryResolver_SearchFacilities_NoResults(t *testing.T) {
	// Arrange
	mockSearch := mocks.NewMockSearchAdapter(t)
	mockDB := mocks.NewMockFacilityRepository(t)
	mockAppt := mocks.NewMockAppointmentRepository(t)
	mockProc := mocks.NewMockProcedureRepository(t)
	mockFacProc := mocks.NewMockFacilityProcedureRepository(t)
	mockIns := mocks.NewMockInsuranceRepository(t)
	mockCache := mocks.NewMockQueryCacheProvider(t)

	resolver := NewResolver(mockSearch, mockDB, mockAppt, mockProc, mockFacProc, mockIns, mockCache, nil)
	queryResolver := resolver.Query()

	ctx := context.Background()
	query := "nonexistent"
	location := generated.LocationInput{
		Latitude:  37.7749,
		Longitude: -122.4194,
	}
	radiusKm := 5.0

	mockSearch.EXPECT().Search(ctx, repositories.SearchParams{
		Query:     query,
		Latitude:  location.Latitude,
		Longitude: location.Longitude,
		RadiusKm:  radiusKm,
		Limit:     20,
		Offset:    0,
	}).Return([]*entities.Facility{}, nil)

	// Act
	result, err := queryResolver.SearchFacilities(ctx, query, location, &radiusKm, nil)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result.FacilitiesData, 0)
	assert.Equal(t, 0, result.TotalCountValue)
}

// TestQueryResolver_FacilitySuggestions_Success tests autocomplete suggestions
func TestQueryResolver_FacilitySuggestions_Success(t *testing.T) {
	// Arrange
	mockSearch := mocks.NewMockSearchAdapter(t)
	mockDB := mocks.NewMockFacilityRepository(t)
	mockAppt := mocks.NewMockAppointmentRepository(t)
	mockProc := mocks.NewMockProcedureRepository(t)
	mockFacProc := mocks.NewMockFacilityProcedureRepository(t)
	mockIns := mocks.NewMockInsuranceRepository(t)
	mockCache := mocks.NewMockQueryCacheProvider(t)

	resolver := NewResolver(mockSearch, mockDB, mockAppt, mockProc, mockFacProc, mockIns, mockCache, nil)
	queryResolver := resolver.Query()

	ctx := context.Background()
	query := "imaging"
	location := generated.LocationInput{
		Latitude:  37.7749,
		Longitude: -122.4194,
	}
	limit := 5

	expectedFacilities := []*entities.Facility{
		{
			ID:           "fac-1",
			Name:         "Imaging Center",
			FacilityType: "imaging_center",
			Address: entities.Address{
				City:  "San Francisco",
				State: "CA",
			},
			Location: entities.Location{
				Latitude:  37.7749,
				Longitude: -122.4194,
			},
			Rating:      4.8,
			ReviewCount: 150,
			IsActive:    true,
		},
	}

	mockSearch.EXPECT().Search(ctx, repositories.SearchParams{
		Query:     query,
		Latitude:  location.Latitude,
		Longitude: location.Longitude,
		RadiusKm:  50.0,
		Limit:     limit,
		Offset:    0,
	}).Return(expectedFacilities, nil)

	// Act
	result, err := queryResolver.FacilitySuggestions(ctx, query, location, &limit)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result, 1)
	assert.Equal(t, "fac-1", result[0].ID)
	assert.Equal(t, "Imaging Center", result[0].Name)
}

// Helper functions
func floatPtr(v float64) *float64 {
	return &v
}
