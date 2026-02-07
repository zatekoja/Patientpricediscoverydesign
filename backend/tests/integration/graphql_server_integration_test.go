//go:build integration

package integration

import (
	"testing"
	"time"

	"github.com/99designs/gqlgen/client"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/entities"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/repositories"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/graphql/generated"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/graphql/resolvers"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/tests/mocks"
)

// TestGraphQLFacilityQuery tests the facility query end-to-end
func TestGraphQLFacilityQuery(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Set up mocks
	mockSearch := mocks.NewMockSearchAdapter(t)
	mockDB := mocks.NewMockFacilityRepository(t)
	mockAppt := mocks.NewMockAppointmentRepository(t)
	mockProc := mocks.NewMockProcedureRepository(t)
	mockFacProc := mocks.NewMockFacilityProcedureRepository(t)
	mockIns := mocks.NewMockInsuranceRepository(t)
	mockCache := mocks.NewMockQueryCacheProvider(t)

	// Create resolver
	resolver := resolvers.NewResolver(mockSearch, mockDB, mockAppt, mockProc, mockFacProc, mockIns, mockCache, nil)

	// Create GraphQL server
	srv := handler.NewDefaultServer(generated.NewExecutableSchema(generated.Config{
		Resolvers: resolver,
	}))

	// Create test client
	c := client.New(srv)

	// Test facility data
	facilityID := "test-facility-1"
	facility := &entities.Facility{
		ID:           facilityID,
		Name:         "Test Hospital",
		FacilityType: "hospital",
		Location: entities.Location{
			Latitude:  37.7749,
			Longitude: -122.4194,
		},
		Address: entities.Address{
			Street:  "123 Main St",
			City:    "San Francisco",
			State:   "CA",
			ZipCode: "94102",
			Country: "USA",
		},
		PhoneNumber: "555-1234",
		Email:       "contact@testhospital.com",
		Website:     "https://testhospital.com",
		Rating:      4.5,
		ReviewCount: 100,
		IsActive:    true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Mock the GetByIDs call (used by DataLoader)
	mockDB.EXPECT().GetByIDs(mock.Anything, []string{facilityID}).Return([]*entities.Facility{facility}, nil)

	// Execute GraphQL query
	query := `
		query GetFacility($id: ID!) {
			facility(id: $id) {
				id
				name
				facilityType
				rating
				reviewCount
				isActive
			}
		}
	`

	var resp struct {
		Facility struct {
			ID           string  `json:"id"`
			Name         string  `json:"name"`
			FacilityType string  `json:"facilityType"`
			Rating       float64 `json:"rating"`
			ReviewCount  int     `json:"reviewCount"`
			IsActive     bool    `json:"isActive"`
		} `json:"facility"`
	}

	err := c.Post(query, &resp, client.Var("id", facilityID))
	require.NoError(t, err)

	// Assert results
	assert.Equal(t, facilityID, resp.Facility.ID)
	assert.Equal(t, "Test Hospital", resp.Facility.Name)
	assert.Equal(t, "HOSPITAL", resp.Facility.FacilityType)
	assert.Equal(t, 4.5, resp.Facility.Rating)
	assert.Equal(t, 100, resp.Facility.ReviewCount)
	assert.True(t, resp.Facility.IsActive)
}

// TestGraphQLSearchFacilitiesWithFacets tests the search with facets
func TestGraphQLSearchFacilitiesWithFacets(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Set up mocks
	mockSearch := mocks.NewMockSearchAdapter(t)
	mockDB := mocks.NewMockFacilityRepository(t)
	mockAppt := mocks.NewMockAppointmentRepository(t)
	mockProc := mocks.NewMockProcedureRepository(t)
	mockFacProc := mocks.NewMockFacilityProcedureRepository(t)
	mockIns := mocks.NewMockInsuranceRepository(t)
	mockCache := mocks.NewMockQueryCacheProvider(t)

	// Create resolver
	resolver := resolvers.NewResolver(mockSearch, mockDB, mockAppt, mockProc, mockFacProc, mockIns, mockCache, nil)

	// Create GraphQL server
	srv := handler.NewDefaultServer(generated.NewExecutableSchema(generated.Config{
		Resolvers: resolver,
	}))

	// Create test client
	c := client.New(srv)

	// Test data
	facilities := []*entities.Facility{
		{
			ID:           "fac-1",
			Name:         "City Hospital",
			FacilityType: "hospital",
			Location: entities.Location{
				Latitude:  37.7749,
				Longitude: -122.4194,
			},
			Rating:      4.8,
			ReviewCount: 200,
			IsActive:    true,
		},
		{
			ID:           "fac-2",
			Name:         "Medical Clinic",
			FacilityType: "clinic",
			Location: entities.Location{
				Latitude:  37.7849,
				Longitude: -122.4294,
			},
			Rating:      4.6,
			ReviewCount: 150,
			IsActive:    true,
		},
	}

	facets := &entities.SearchFacets{
		FacilityTypes: []entities.FacetCount{
			{Value: "hospital", Count: 1},
			{Value: "clinic", Count: 1},
		},
		InsuranceProviders: []entities.FacetCount{
			{Value: "Blue Cross", Count: 2},
		},
		Specialties:        []entities.FacetCount{},
		Cities:             []entities.FacetCount{},
		States:             []entities.FacetCount{},
		PriceRanges:        []entities.PriceRangeFacet{},
		RatingDistribution: []entities.RatingFacet{},
	}

	// Mock the search call
	mockSearch.EXPECT().SearchWithFacets(
		mock.Anything, // Use mock.Anything for context to match any context type
		repositories.SearchParams{
			Query:         "hospital",
			Latitude:      37.7749,
			Longitude:     -122.4194,
			RadiusKm:      10.0,
			Limit:         20,
			Offset:        0,
			IncludeFacets: true,
		},
	).Return(&repositories.EnhancedSearchResult{
		Facilities: facilities,
		Facets:     facets,
		TotalCount: 2,
		SearchTime: 15.5,
	}, nil)

	// Execute GraphQL query
	query := `
		query SearchFacilities($query: String!, $lat: Float!, $lon: Float!, $radius: Float!) {
			searchFacilities(
				query: $query
				location: { latitude: $lat, longitude: $lon }
				radiusKm: $radius
			) {
				facilities {
					id
					name
					facilityType
					rating
				}
				facets {
					facilityTypes {
						value
						count
					}
					insuranceProviders {
						value
						count
					}
				}
				totalCount
				searchTime
				pagination {
					hasNextPage
					hasPreviousPage
					currentPage
					totalPages
				}
			}
		}
	`

	var resp struct {
		SearchFacilities struct {
			Facilities []struct {
				ID           string  `json:"id"`
				Name         string  `json:"name"`
				FacilityType string  `json:"facilityType"`
				Rating       float64 `json:"rating"`
			} `json:"facilities"`
			Facets struct {
				FacilityTypes []struct {
					Value string `json:"value"`
					Count int    `json:"count"`
				} `json:"facilityTypes"`
				InsuranceProviders []struct {
					Value string `json:"value"`
					Count int    `json:"count"`
				} `json:"insuranceProviders"`
			} `json:"facets"`
			TotalCount int     `json:"totalCount"`
			SearchTime float64 `json:"searchTime"`
			Pagination struct {
				HasNextPage     bool `json:"hasNextPage"`
				HasPreviousPage bool `json:"hasPreviousPage"`
				CurrentPage     int  `json:"currentPage"`
				TotalPages      int  `json:"totalPages"`
			} `json:"pagination"`
		} `json:"searchFacilities"`
	}

	err := c.Post(query, &resp,
		client.Var("query", "hospital"),
		client.Var("lat", 37.7749),
		client.Var("lon", -122.4194),
		client.Var("radius", 10.0),
	)
	require.NoError(t, err)

	// Assert results
	assert.Len(t, resp.SearchFacilities.Facilities, 2)
	assert.Equal(t, "fac-1", resp.SearchFacilities.Facilities[0].ID)
	assert.Equal(t, "City Hospital", resp.SearchFacilities.Facilities[0].Name)

	// Assert facets
	assert.Len(t, resp.SearchFacilities.Facets.FacilityTypes, 2)
	assert.Equal(t, "hospital", resp.SearchFacilities.Facets.FacilityTypes[0].Value)
	assert.Equal(t, 1, resp.SearchFacilities.Facets.FacilityTypes[0].Count)

	// Assert metadata
	assert.Equal(t, 2, resp.SearchFacilities.TotalCount)
	assert.Equal(t, 15.5, resp.SearchFacilities.SearchTime)

	// Assert pagination
	assert.False(t, resp.SearchFacilities.Pagination.HasNextPage)
	assert.False(t, resp.SearchFacilities.Pagination.HasPreviousPage)
	assert.Equal(t, 1, resp.SearchFacilities.Pagination.CurrentPage)
	assert.Equal(t, 1, resp.SearchFacilities.Pagination.TotalPages)
}

// TestGraphQLPaginationBehavior tests pagination logic
func TestGraphQLPaginationBehavior(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Set up mocks
	mockSearch := mocks.NewMockSearchAdapter(t)
	mockDB := mocks.NewMockFacilityRepository(t)
	mockAppt := mocks.NewMockAppointmentRepository(t)
	mockProc := mocks.NewMockProcedureRepository(t)
	mockFacProc := mocks.NewMockFacilityProcedureRepository(t)
	mockIns := mocks.NewMockInsuranceRepository(t)
	mockCache := mocks.NewMockQueryCacheProvider(t)

	// Create resolver
	resolver := resolvers.NewResolver(mockSearch, mockDB, mockAppt, mockProc, mockFacProc, mockIns, mockCache, nil)

	// Create GraphQL server
	srv := handler.NewDefaultServer(generated.NewExecutableSchema(generated.Config{
		Resolvers: resolver,
	}))

	// Create test client
	c := client.New(srv)

	// Test data - simulate 35 facilities with 10 per page
	facilities := make([]*entities.Facility, 10)
	for i := 0; i < 10; i++ {
		facilities[i] = &entities.Facility{
			ID:           string(rune('A' + i)),
			Name:         "Facility " + string(rune('A'+i)),
			FacilityType: "hospital",
			IsActive:     true,
		}
	}

	// Mock page 2 of 4 (offset 10, limit 10, total 35)
	mockSearch.EXPECT().SearchWithFacets(
		mock.Anything, // Use mock.Anything for context
		repositories.SearchParams{
			Latitude:      37.7749,
			Longitude:     -122.4194,
			RadiusKm:      10.0,
			Limit:         10,
			Offset:        10,
			IncludeFacets: true,
		},
	).Return(&repositories.EnhancedSearchResult{
		Facilities: facilities,
		Facets:     &entities.SearchFacets{},
		TotalCount: 35,
		SearchTime: 12.0,
	}, nil)

	query := `
		query Facilities($lat: Float!, $lon: Float!, $radius: Float!, $limit: Int, $offset: Int) {
			facilities(filter: {
				location: { latitude: $lat, longitude: $lon }
				radiusKm: $radius
				limit: $limit
				offset: $offset
			}) {
				facilities {
					id
					name
				}
				totalCount
				pagination {
					hasNextPage
					hasPreviousPage
					currentPage
					totalPages
					limit
					offset
				}
			}
		}
	`

	var resp struct {
		Facilities struct {
			Facilities []struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"facilities"`
			TotalCount int `json:"totalCount"`
			Pagination struct {
				HasNextPage     bool `json:"hasNextPage"`
				HasPreviousPage bool `json:"hasPreviousPage"`
				CurrentPage     int  `json:"currentPage"`
				TotalPages      int  `json:"totalPages"`
				Limit           int  `json:"limit"`
				Offset          int  `json:"offset"`
			} `json:"pagination"`
		} `json:"facilities"`
	}

	err := c.Post(query, &resp,
		client.Var("lat", 37.7749),
		client.Var("lon", -122.4194),
		client.Var("radius", 10.0),
		client.Var("limit", 10),
		client.Var("offset", 10),
	)
	require.NoError(t, err)

	// Assert pagination for page 2 of 4
	assert.Equal(t, 35, resp.Facilities.TotalCount)
	assert.True(t, resp.Facilities.Pagination.HasNextPage)    // More pages ahead
	assert.True(t, resp.Facilities.Pagination.HasPreviousPage) // Previous page exists
	assert.Equal(t, 2, resp.Facilities.Pagination.CurrentPage)
	assert.Equal(t, 4, resp.Facilities.Pagination.TotalPages) // 35 / 10 = 4 pages
	assert.Equal(t, 10, resp.Facilities.Pagination.Limit)
	assert.Equal(t, 10, resp.Facilities.Pagination.Offset)
}
