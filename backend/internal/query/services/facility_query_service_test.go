package services

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/entities"
)

// MockSearchRepository is a mock implementation of search repository
type MockSearchRepository struct {
	mock.Mock
}

func (m *MockSearchRepository) SearchWithFacets(ctx context.Context, params interface{}) ([]*entities.Facility, *Facets, int, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, nil, 0, args.Error(3)
	}
	return args.Get(0).([]*entities.Facility), args.Get(1).(*Facets), args.Int(2), args.Error(3)
}

func (m *MockSearchRepository) Index(ctx context.Context, facility *entities.Facility) error {
	args := m.Called(ctx, facility)
	return args.Error(0)
}

func (m *MockSearchRepository) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

// MockFacilityRepository is a mock implementation of facility repository
type MockFacilityRepository struct {
	mock.Mock
}

func (m *MockFacilityRepository) GetByID(ctx context.Context, id string) (*entities.Facility, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entities.Facility), args.Error(1)
}

func (m *MockFacilityRepository) Create(ctx context.Context, facility *entities.Facility) error {
	args := m.Called(ctx, facility)
	return args.Error(0)
}

func (m *MockFacilityRepository) Update(ctx context.Context, facility *entities.Facility) error {
	args := m.Called(ctx, facility)
	return args.Error(0)
}

func (m *MockFacilityRepository) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockFacilityRepository) List(ctx context.Context, filter interface{}) ([]*entities.Facility, error) {
	args := m.Called(ctx, filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entities.Facility), args.Error(1)
}

func (m *MockFacilityRepository) Search(ctx context.Context, params interface{}) ([]*entities.Facility, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entities.Facility), args.Error(1)
}

// MockCacheProvider is a mock implementation of cache provider
type MockCacheProvider struct {
	mock.Mock
}

func (m *MockCacheProvider) Get(ctx context.Context, key string) (interface{}, error) {
	args := m.Called(ctx, key)
	return args.Get(0), args.Error(1)
}

func (m *MockCacheProvider) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	args := m.Called(ctx, key, value, ttl)
	return args.Error(0)
}

func (m *MockCacheProvider) Delete(ctx context.Context, key string) error {
	args := m.Called(ctx, key)
	return args.Error(0)
}

// TestFacilityQueryService_Search_Success tests successful search
func TestFacilityQueryService_Search_Success(t *testing.T) {
	// Arrange
	mockSearchRepo := new(MockSearchRepository)
	mockDBRepo := new(MockFacilityRepository)
	mockCache := new(MockCacheProvider)

	service := NewFacilityQueryService(mockSearchRepo, mockDBRepo, mockCache)

	ctx := context.Background()
	params := SearchParams{
		Query:     "hospital",
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

	expectedFacets := &Facets{
		FacilityTypes: map[string]int{
			"hospital": 1,
		},
		Cities: map[string]int{
			"San Francisco": 1,
		},
	}

	mockSearchRepo.On("SearchWithFacets", ctx, mock.Anything).Return(
		expectedFacilities,
		expectedFacets,
		1,
		nil,
	)

	// Act
	result, err := service.Search(ctx, params)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result.Facilities, 1)
	assert.Equal(t, "fac-1", result.Facilities[0].ID)
	assert.Equal(t, "Test Hospital", result.Facilities[0].Name)
	assert.Equal(t, 1, result.TotalCount)
	assert.GreaterOrEqual(t, result.SearchTime, 0.0)
	assert.NotNil(t, result.Facets)
	assert.Equal(t, 1, result.Facets.FacilityTypes["hospital"])

	mockSearchRepo.AssertExpectations(t)
}

// TestFacilityQueryService_Search_EmptyResults tests search with no results
func TestFacilityQueryService_Search_EmptyResults(t *testing.T) {
	// Arrange
	mockSearchRepo := new(MockSearchRepository)
	mockDBRepo := new(MockFacilityRepository)
	mockCache := new(MockCacheProvider)

	service := NewFacilityQueryService(mockSearchRepo, mockDBRepo, mockCache)

	ctx := context.Background()
	params := SearchParams{
		Query:     "nonexistent",
		Latitude:  37.7749,
		Longitude: -122.4194,
		RadiusKm:  10,
		Limit:     20,
		Offset:    0,
	}

	mockSearchRepo.On("SearchWithFacets", ctx, mock.Anything).Return(
		[]*entities.Facility{},
		&Facets{},
		0,
		nil,
	)

	// Act
	result, err := service.Search(ctx, params)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Empty(t, result.Facilities)
	assert.Equal(t, 0, result.TotalCount)

	mockSearchRepo.AssertExpectations(t)
}

// TestFacilityQueryService_GetByID_CacheHit tests getting facility from cache
func TestFacilityQueryService_GetByID_CacheHit(t *testing.T) {
	// Arrange
	mockSearchRepo := new(MockSearchRepository)
	mockDBRepo := new(MockFacilityRepository)
	mockCache := new(MockCacheProvider)

	service := NewFacilityQueryService(mockSearchRepo, mockDBRepo, mockCache)

	ctx := context.Background()
	facilityID := "fac-1"

	cachedFacility := &entities.Facility{
		ID:           facilityID,
		Name:         "Cached Hospital",
		FacilityType: "hospital",
		IsActive:     true,
	}

	mockCache.On("Get", ctx, "facility:"+facilityID).Return(cachedFacility, nil)

	// Act
	result, err := service.GetByID(ctx, facilityID)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, facilityID, result.ID)
	assert.Equal(t, "Cached Hospital", result.Name)

	mockCache.AssertExpectations(t)
	// DB should not be called on cache hit
	mockDBRepo.AssertNotCalled(t, "GetByID")
}

// TestFacilityQueryService_GetByID_CacheMiss_DBFallback tests cache miss and DB fallback
func TestFacilityQueryService_GetByID_CacheMiss_DBFallback(t *testing.T) {
	// Arrange
	mockSearchRepo := new(MockSearchRepository)
	mockDBRepo := new(MockFacilityRepository)
	mockCache := new(MockCacheProvider)

	service := NewFacilityQueryService(mockSearchRepo, mockDBRepo, mockCache)

	ctx := context.Background()
	facilityID := "fac-1"

	dbFacility := &entities.Facility{
		ID:           facilityID,
		Name:         "DB Hospital",
		FacilityType: "hospital",
		IsActive:     true,
	}

	// Cache miss
	mockCache.On("Get", ctx, "facility:"+facilityID).Return(nil, assert.AnError)

	// DB hit
	mockDBRepo.On("GetByID", ctx, facilityID).Return(dbFacility, nil)

	// Cache set after DB fetch
	mockCache.On("Set", ctx, "facility:"+facilityID, dbFacility, 5*time.Minute).Return(nil)

	// Act
	result, err := service.GetByID(ctx, facilityID)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, facilityID, result.ID)
	assert.Equal(t, "DB Hospital", result.Name)

	mockCache.AssertExpectations(t)
	mockDBRepo.AssertExpectations(t)
}

// TestFacilityQueryService_GetByID_NotFound tests facility not found
func TestFacilityQueryService_GetByID_NotFound(t *testing.T) {
	// Arrange
	mockSearchRepo := new(MockSearchRepository)
	mockDBRepo := new(MockFacilityRepository)
	mockCache := new(MockCacheProvider)

	service := NewFacilityQueryService(mockSearchRepo, mockDBRepo, mockCache)

	ctx := context.Background()
	facilityID := "non-existent"

	// Cache miss
	mockCache.On("Get", ctx, "facility:"+facilityID).Return(nil, assert.AnError)

	// DB miss
	mockDBRepo.On("GetByID", ctx, facilityID).Return(nil, assert.AnError)

	// Act
	result, err := service.GetByID(ctx, facilityID)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)

	mockCache.AssertExpectations(t)
	mockDBRepo.AssertExpectations(t)
}

// TestFacilityQueryService_Search_WithFilters tests search with multiple filters
func TestFacilityQueryService_Search_WithFilters(t *testing.T) {
	// Arrange
	mockSearchRepo := new(MockSearchRepository)
	mockDBRepo := new(MockFacilityRepository)
	mockCache := new(MockCacheProvider)

	service := NewFacilityQueryService(mockSearchRepo, mockDBRepo, mockCache)

	ctx := context.Background()
	minRating := 4.0
	maxPrice := 500.0
	params := SearchParams{
		Query:              "hospital",
		Latitude:           37.7749,
		Longitude:          -122.4194,
		RadiusKm:           10,
		FacilityTypes:      []string{"hospital", "clinic"},
		InsuranceProviders: []string{"ins-1", "ins-2"},
		MinRating:          &minRating,
		MaxPrice:           &maxPrice,
		HasParking:         boolPtr(true),
		Limit:              20,
		Offset:             0,
	}

	expectedFacilities := []*entities.Facility{
		{
			ID:           "fac-1",
			Name:         "Premium Hospital",
			FacilityType: "hospital",
			Rating:       4.8,
			IsActive:     true,
		},
	}

	mockSearchRepo.On("SearchWithFacets", ctx, mock.Anything).Return(
		expectedFacilities,
		&Facets{},
		1,
		nil,
	)

	// Act
	result, err := service.Search(ctx, params)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result.Facilities, 1)
	assert.Equal(t, "Premium Hospital", result.Facilities[0].Name)

	mockSearchRepo.AssertExpectations(t)
}

// TestFacilityQueryService_Suggest_Success tests autocomplete suggestions
func TestFacilityQueryService_Suggest_Success(t *testing.T) {
	// Arrange
	mockSearchRepo := new(MockSearchRepository)
	mockDBRepo := new(MockFacilityRepository)
	mockCache := new(MockCacheProvider)

	service := NewFacilityQueryService(mockSearchRepo, mockDBRepo, mockCache)

	ctx := context.Background()

	// This test will pass once we implement Suggest method
	// For now, we expect it to return nil, nil (not implemented)

	// Act
	result, err := service.Suggest(ctx, "hosp", 37.7749, -122.4194, 5)

	// Assert
	// Currently returns nil, nil as not implemented
	assert.NoError(t, err)
	assert.Nil(t, result)
}

// Helper function
func boolPtr(b bool) *bool {
	return &b
}
