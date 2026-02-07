package entities

// GraphQLFacilitySearchResult is the container for GraphQL search query results
// It matches the FacilitySearchResult type in the GraphQL schema
type GraphQLFacilitySearchResult struct {
	// Facilities contains the list of matching facilities
	FacilitiesData []*Facility

	// Facets contains aggregated search facets
	FacetsData *SearchFacets

	// Pagination contains pagination metadata
	PaginationData *PaginationInfo

	// TotalCount is the total number of results (before pagination)
	TotalCountValue int

	// SearchTimeMs is the time taken for the search in milliseconds
	SearchTimeMs float64
}

// SearchFacets contains aggregated facet data from search results
type SearchFacets struct {
	FacilityTypes      []FacetCount
	InsuranceProviders []FacetCount
	Specialties        []FacetCount
	Cities             []FacetCount
	States             []FacetCount
	PriceRanges        []PriceRangeFacet
	RatingDistribution []RatingFacet
}

// FacetCount represents a facet value and its count
type FacetCount struct {
	Value string
	Count int
}

// PriceRangeFacet represents a price range facet
type PriceRangeFacet struct {
	Min   float64
	Max   float64
	Count int
}

// RatingFacet represents a rating facet
type RatingFacet struct {
	Rating float64
	Count  int
}

// PaginationInfo contains pagination metadata
type PaginationInfo struct {
	HasNextPage     bool
	HasPreviousPage bool
	CurrentPage     int
	TotalPages      int
	Limit           int
	Offset          int
}
