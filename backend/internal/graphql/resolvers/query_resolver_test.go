package resolvers

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/entities"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/repositories"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/graphql/generated"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/graphql/loaders"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/tests/mocks"
)

// TestQueryResolver_Facility_Success tests successful facility retrieval by ID
func TestQueryResolver_Facility_Success(t *testing.T) {
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

	// Set up DataLoader
	ldrs := loaders.NewLoaders(mockDB, mockProc)
	ctx := loaders.WithLoaders(context.Background(), ldrs)

	facilityID := "fac-123"

	expectedFacility := &entities.Facility{
		ID:           facilityID,
		Name:         "Test Hospital",
		FacilityType: "hospital",
		Location: entities.Location{
			Latitude:  37.7749,
			Longitude: -122.4194,
		},
		Rating:      4.5,
		ReviewCount: 100,
		IsActive:    true,
		CreatedAt:   time.Now(),
	}

	// Mock: DataLoader will call GetByIDs
	mockDB.EXPECT().GetByIDs(ctx, []string{facilityID}).Return([]*entities.Facility{expectedFacility}, nil)

	// Act
	result, err := queryResolver.Facility(ctx, facilityID)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, facilityID, result.ID)
	assert.Equal(t, "Test Hospital", result.Name)
}

// TestQueryResolver_Facility_NotFound tests facility not found
func TestQueryResolver_Facility_NotFound(t *testing.T) {
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
	facilityID := "non-existent"

	// Cache miss, DB miss
	mockCache.EXPECT().Get(ctx, "facility:"+facilityID).Return(nil, assert.AnError)
	mockDB.EXPECT().GetByID(ctx, facilityID).Return(nil, assert.AnError)

	// Act
	result, err := queryResolver.Facility(ctx, facilityID)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
}

// TestQueryResolver_SearchFacilities_Success tests successful facility search
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
	query := "hospital"
	location := generated.LocationInput{
		Latitude:  37.7749,
		Longitude: -122.4194,
	}
	radiusKm := 10.0

	expectedFacilities := []*entities.Facility{
		{
			ID:           "fac-1",
			Name:         "Test Hospital",
			FacilityType: "hospital",
			Location: entities.Location{
				Latitude:  37.7749,
				Longitude: -122.4194,
			},
			Rating:      4.5,
			ReviewCount: 100,
			IsActive:    true,
		},
	}

	mockSearch.EXPECT().Search(ctx, repositories.SearchParams{
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
	assert.Len(t, result.Facilities, 1)
	assert.Equal(t, "fac-1", result.Facilities[0].ID)
}

// TestQueryResolver_Facilities_Success tests facility search with filter
func TestQueryResolver_Facilities_Success(t *testing.T) {
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
	filter := generated.FacilitySearchInput{
		Location: &generated.LocationInput{
			Latitude:  37.7749,
			Longitude: -122.4194,
		},
		RadiusKm: 10.0,
		Limit:    intPtr(20),
		Offset:   intPtr(0),
	}

	expectedFacilities := []*entities.Facility{
		{
			ID:           "fac-1",
			Name:         "Test Hospital",
			FacilityType: "hospital",
			Location: entities.Location{
				Latitude:  37.7749,
				Longitude: -122.4194,
			},
			Rating:      4.5,
			ReviewCount: 100,
			IsActive:    true,
		},
	}

	// Mock search adapter
	mockSearch.EXPECT().Search(ctx, repositories.SearchParams{
		Latitude:  filter.Location.Latitude,
		Longitude: filter.Location.Longitude,
		RadiusKm:  filter.RadiusKm,
		Limit:     20,
		Offset:    0,
	}).Return(expectedFacilities, nil)

	// Act
	result, err := queryResolver.Facilities(ctx, filter)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotNil(t, result.FacilitiesData)
	assert.Len(t, result.FacilitiesData, 1)
	assert.Equal(t, "fac-1", result.FacilitiesData[0].ID)
	assert.Equal(t, 1, result.TotalCountValue)
}

// TestFacilitySearchResultResolver_Facilities tests field resolver for facilities
func TestFacilitySearchResultResolver_Facilities(t *testing.T) {
	// Arrange
	mockSearch := mocks.NewMockSearchAdapter(t)
	mockDB := mocks.NewMockFacilityRepository(t)
	mockAppt := mocks.NewMockAppointmentRepository(t)
	mockProc := mocks.NewMockProcedureRepository(t)
	mockFacProc := mocks.NewMockFacilityProcedureRepository(t)
	mockIns := mocks.NewMockInsuranceRepository(t)
	mockCache := mocks.NewMockQueryCacheProvider(t)

	resolver := NewResolver(mockSearch, mockDB, mockAppt, mockProc, mockFacProc, mockIns, mockCache, nil)
	fieldResolver := resolver.FacilitySearchResult()

	ctx := context.Background()
	searchResult := &entities.GraphQLFacilitySearchResult{
		FacilitiesData: []*entities.Facility{
			{
				ID:   "fac-1",
				Name: "Hospital A",
			},
			{
				ID:   "fac-2",
				Name: "Hospital B",
			},
		},
		TotalCountValue: 2,
	}

	// Act
	facilities, err := fieldResolver.Facilities(ctx, searchResult)

	// Assert
	assert.NoError(t, err)
	assert.Len(t, facilities, 2)
	assert.Equal(t, "fac-1", facilities[0].ID)
	assert.Equal(t, "fac-2", facilities[1].ID)
}

// TestFacilitySearchResultResolver_TotalCount tests field resolver for totalCount
func TestFacilitySearchResultResolver_TotalCount(t *testing.T) {
	// Arrange
	mockSearch := mocks.NewMockSearchAdapter(t)
	mockDB := mocks.NewMockFacilityRepository(t)
	mockAppt := mocks.NewMockAppointmentRepository(t)
	mockProc := mocks.NewMockProcedureRepository(t)
	mockFacProc := mocks.NewMockFacilityProcedureRepository(t)
	mockIns := mocks.NewMockInsuranceRepository(t)
	mockCache := mocks.NewMockQueryCacheProvider(t)

	resolver := NewResolver(mockSearch, mockDB, mockAppt, mockProc, mockFacProc, mockIns, mockCache, nil)
	fieldResolver := resolver.FacilitySearchResult()

	ctx := context.Background()
	searchResult := &entities.GraphQLFacilitySearchResult{
		TotalCountValue: 42,
	}

	// Act
	count, err := fieldResolver.TotalCount(ctx, searchResult)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, 42, count)
}

// Helper functions
func intPtr(v int) *int {
	return &v
}
