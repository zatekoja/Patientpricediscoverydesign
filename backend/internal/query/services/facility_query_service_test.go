package services

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/entities"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/repositories"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/tests/mocks"
)

// Test 1: Search via Typesense succeeds
func TestFacilityQueryServiceImpl_Search_Success(t *testing.T) {
	// Arrange
	mockSearch := mocks.NewMockSearchAdapter(t)
	mockDB := mocks.NewMockFacilityRepository(t)
	mockCache := mocks.NewMockQueryCacheProvider(t)

	service := NewFacilityQueryServiceImpl(mockSearch, mockDB, mockCache)

	ctx := context.Background()
	params := repositories.SearchParams{
		Latitude:  37.7749,
		Longitude: -122.4194,
		RadiusKm:  10,
		Limit:     20,
		Offset:    0,
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
			CreatedAt:   time.Now(),
		},
	}

	mockSearch.EXPECT().Search(ctx, params).Return(expectedFacilities, nil)

	// Act
	result, err := service.Search(ctx, params)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result, 1)
	assert.Equal(t, "fac-1", result[0].ID)
	assert.Equal(t, "Test Hospital", result[0].Name)
}

// Test 2: Search falls back to DB when Typesense fails
func TestFacilityQueryServiceImpl_Search_FallbackToDB(t *testing.T) {
	// Arrange
	mockSearch := mocks.NewMockSearchAdapter(t)
	mockDB := mocks.NewMockFacilityRepository(t)
	mockCache := mocks.NewMockQueryCacheProvider(t)

	service := NewFacilityQueryServiceImpl(mockSearch, mockDB, mockCache)

	ctx := context.Background()
	params := repositories.SearchParams{
		Latitude:  37.7749,
		Longitude: -122.4194,
		RadiusKm:  10,
		Limit:     20,
	}

	dbFacilities := []*entities.Facility{
		{
			ID:   "fac-from-db",
			Name: "DB Hospital",
		},
	}

	// Typesense fails
	mockSearch.EXPECT().Search(ctx, params).Return(nil, assert.AnError)
	// Fallback to DB
	mockDB.EXPECT().Search(ctx, params).Return(dbFacilities, nil)

	// Act
	result, err := service.Search(ctx, params)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result, 1)
	assert.Equal(t, "fac-from-db", result[0].ID)
}

// Test 3: GetByID returns from cache
func TestFacilityQueryServiceImpl_GetByID_CacheHit(t *testing.T) {
	// Arrange
	mockSearch := mocks.NewMockSearchAdapter(t)
	mockDB := mocks.NewMockFacilityRepository(t)
	mockCache := mocks.NewMockQueryCacheProvider(t)

	service := NewFacilityQueryServiceImpl(mockSearch, mockDB, mockCache)

	ctx := context.Background()
	facilityID := "fac-1"

	cachedFacility := &entities.Facility{
		ID:   facilityID,
		Name: "Cached Hospital",
	}

	mockCache.EXPECT().Get(ctx, "facility:"+facilityID).Return(cachedFacility, nil)

	// Act
	result, err := service.GetByID(ctx, facilityID)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "Cached Hospital", result.Name)
}

// Test 4: GetByID falls back to DB and caches result
func TestFacilityQueryServiceImpl_GetByID_DBFallback(t *testing.T) {
	// Arrange
	mockSearch := mocks.NewMockSearchAdapter(t)
	mockDB := mocks.NewMockFacilityRepository(t)
	mockCache := mocks.NewMockQueryCacheProvider(t)

	service := NewFacilityQueryServiceImpl(mockSearch, mockDB, mockCache)

	ctx := context.Background()
	facilityID := "fac-1"

	dbFacility := &entities.Facility{
		ID:   facilityID,
		Name: "DB Hospital",
	}

	// Cache miss
	mockCache.EXPECT().Get(ctx, "facility:"+facilityID).Return(nil, assert.AnError)
	// DB hit
	mockDB.EXPECT().GetByID(ctx, facilityID).Return(dbFacility, nil)
	// Cache set
	mockCache.EXPECT().Set(ctx, "facility:"+facilityID, dbFacility, 5*time.Minute).Return(nil)

	// Act
	result, err := service.GetByID(ctx, facilityID)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "DB Hospital", result.Name)
}

// Test 5: GetByID returns error when not found
func TestFacilityQueryServiceImpl_GetByID_NotFound(t *testing.T) {
	// Arrange
	mockSearch := mocks.NewMockSearchAdapter(t)
	mockDB := mocks.NewMockFacilityRepository(t)
	mockCache := mocks.NewMockQueryCacheProvider(t)

	service := NewFacilityQueryServiceImpl(mockSearch, mockDB, mockCache)

	ctx := context.Background()
	facilityID := "non-existent"

	// Cache miss
	mockCache.EXPECT().Get(ctx, "facility:"+facilityID).Return(nil, assert.AnError)
	// DB miss
	mockDB.EXPECT().GetByID(ctx, facilityID).Return(nil, assert.AnError)

	// Act
	result, err := service.GetByID(ctx, facilityID)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
}
