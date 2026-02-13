# GraphQL Query Service Implementation Plan

## Executive Summary
Build a scalable, independently deployable GraphQL query service to provide the best search experience using Typesense, following CQRS principles and TDD methodology. This service will handle all read operations while the existing REST API manages writes (commands).

## Current State Analysis

### ✅ Already Implemented
- Typesense adapter with basic search functionality
- Typesense client infrastructure
- Domain entities (Facility, Procedure, Insurance, Appointment, User)
- Repository interfaces (CQRS-ready separation)
- Docker compose setup with Typesense
- Integration tests for Typesense
- OpenTelemetry observability setup

### 🔨 To Be Implemented
- GraphQL schema and server
- Enhanced Typesense search capabilities
- Query-side optimizations
- GraphQL resolvers with Typesense integration
- Separate GraphQL service deployment
- GraphQL-specific tests

---

## Architecture Design

### CQRS Separation

```
┌─────────────────────────────────────────────────────────────┐
│                    Frontend (React)                          │
└────────────┬────────────────────────────────┬───────────────┘
             │                                │
    Queries  │                                │  Mutations
   (GraphQL) │                                │  (REST)
             ▼                                ▼
┌────────────────────────┐      ┌────────────────────────────┐
│  GraphQL Query Service │      │   REST Command Service     │
│  (Port 8081)          │      │   (Port 8080)              │
│  - Read-only          │      │   - Write operations       │
│  - Typesense queries  │      │   - Business validation    │
│  - Complex filtering  │      │   - PostgreSQL writes      │
│  - Geo search         │      │   - Event publishing       │
└───────────┬────────────┘      └─────────────┬──────────────┘
            │                                  │
            │ Query                            │ Write + Sync
            ▼                                  ▼
┌────────────────────────┐      ┌────────────────────────────┐
│      Typesense         │◄─────│      PostgreSQL            │
│   (Read Model)         │ Sync │   (Write Model)            │
│   - Optimized search   │      │   - Source of truth        │
│   - Denormalized data  │      │   - Normalized relations   │
└────────────────────────┘      └────────────────────────────┘
```

### Service Responsibilities

#### GraphQL Query Service (New)
- **Port**: 8081
- **Purpose**: Fast, flexible read operations
- **Data Source**: Primarily Typesense, fallback to PostgreSQL for detailed relational data
- **Operations**:
  - Search facilities by location
  - Filter by multiple criteria (price, insurance, rating, etc.)
  - Faceted search (aggregations)
  - Typeahead/autocomplete
  - Get detailed facility information
  - Get available appointments

#### REST Command Service (Existing)
- **Port**: 8080
- **Purpose**: Write operations and business logic
- **Data Source**: PostgreSQL
- **Operations**:
  - Create/Update/Delete facilities
  - Book appointments
  - Update insurance information
  - Add procedures
- **New Responsibility**: Sync to Typesense on successful writes

---

## Implementation Phases

### Phase 1: Enhanced Typesense Schema & Indexing (Week 1)

#### 1.1 Define Comprehensive Typesense Collections

**Collections to Create:**

##### `facilities` Collection (Enhanced)
```go
Fields:
- id (string) - Primary key
- name (string) - Searchable
- description (string) - Searchable, optional
- facility_type (string) - Faceted (hospital, clinic, lab, etc.)
- location (geopoint) - [lat, lon]
- address (string) - Searchable
- city (string) - Faceted
- state (string) - Faceted
- zip_code (string)
- phone (string)
- email (string)
- website (string, optional)
- is_active (bool)
- rating (float) - Faceted, sortable
- review_count (int32) - Sortable
- accepts_new_patients (bool) - Faceted
- has_emergency (bool) - Faceted
- has_parking (bool) - Faceted
- wheelchair_accessible (bool) - Faceted
- insurance_providers (string[]) - Faceted, multi-select
- specialties (string[]) - Faceted, searchable
- languages_spoken (string[]) - Faceted
- min_procedure_price (float) - Sortable
- max_procedure_price (float) - Sortable
- avg_wait_time_minutes (int32) - Sortable
- next_available_slot (int64) - Sortable (Unix timestamp)
- created_at (int64) - Sortable
- updated_at (int64) - Sortable
```

##### `procedures` Collection (New)
```go
Fields:
- id (string)
- name (string) - Searchable
- code (string) - CPT/ICD code
- category (string) - Faceted
- description (string) - Searchable
- facility_id (string) - For filtering
- facility_name (string) - Searchable
- price (float) - Sortable, faceted
- insurance_coverage (string[]) - Faceted
- duration_minutes (int32)
- requires_referral (bool) - Faceted
- preparation_required (bool) - Faceted
- is_active (bool)
- created_at (int64)
```

##### `appointments` Collection (New)
```go
Fields:
- id (string)
- facility_id (string)
- facility_name (string) - Searchable
- patient_id (string)
- patient_name (string) - Searchable
- procedure_id (string)
- procedure_name (string) - Searchable
- provider_name (string) - Searchable
- appointment_date (int64) - Sortable
- duration_minutes (int32)
- status (string) - Faceted (scheduled, completed, cancelled, no-show)
- price (float)
- insurance_provider (string) - Faceted
- notes (string, optional) - Searchable
- created_at (int64)
```

##### `insurance_providers` Collection (New)
```go
Fields:
- id (string)
- name (string) - Searchable
- provider_type (string) - Faceted (PPO, HMO, EPO, etc.)
- network_tier (string) - Faceted
- is_active (bool)
- coverage_states (string[]) - Faceted
- facilities_count (int32) - Sortable
- procedures_count (int32) - Sortable
```

#### 1.2 Create Enhanced Typesense Adapters

**Files to Create:**

```
backend/internal/adapters/search/
├── typesense_adapter.go (enhance existing)
├── typesense_facility_adapter.go (new)
├── typesense_procedure_adapter.go (new)
├── typesense_appointment_adapter.go (new)
├── typesense_insurance_adapter.go (new)
└── search_builder.go (new - query builder)
```

**Key Features:**
- Type-safe query builder
- Complex filtering support
- Faceted search capabilities
- Geo-radius search optimization
- Pagination helpers
- Sort/rank optimization

#### 1.3 Create Data Sync Service

**Files to Create:**
```
backend/internal/application/services/
└── sync_service.go
```

**Responsibilities:**
- Sync PostgreSQL → Typesense on writes
- Batch indexing for initial data load
- Error handling and retry logic
- Metrics for sync performance

**Tests to Write:**
```
backend/internal/application/services/
└── sync_service_test.go
```

#### 1.4 Create Indexer Command

**Files to Create:**
```
backend/cmd/indexer/
└── main.go (enhance existing)
```

**Features:**
- Full reindex command
- Incremental sync based on timestamps
- Progress reporting
- Dry-run mode
- Collection management (create/delete/reset)

---

### Phase 2: GraphQL Service Foundation (Week 2)

#### 2.1 Project Structure

```
backend/cmd/
└── graphql/
    └── main.go

backend/internal/
├── graphql/
│   ├── schema.graphql
│   ├── gqlgen.yml
│   ├── generated/
│   │   └── generated.go (auto-generated)
│   ├── models/
│   │   └── models.go
│   ├── resolvers/
│   │   ├── resolver.go
│   │   ├── query.resolvers.go
│   │   ├── mutation.resolvers.go (minimal/none)
│   │   ├── facility.resolvers.go
│   │   ├── procedure.resolvers.go
│   │   ├── appointment.resolvers.go
│   │   └── search.resolvers.go
│   ├── loaders/
│   │   └── dataloader.go (for N+1 prevention)
│   └── middleware/
│       ├── auth.go
│       ├── observability.go
│       ├── error_handling.go
│       └── complexity.go
└── query/
    ├── services/
    │   ├── facility_query_service.go
    │   ├── procedure_query_service.go
    │   ├── appointment_query_service.go
    │   └── search_query_service.go
    └── repositories/
        └── query_repository.go (read-only interfaces)
```

#### 2.2 GraphQL Schema Design

**File:** `backend/internal/graphql/schema.graphql`

```graphql
# ============================================================================
# Core Types
# ============================================================================

type Facility {
  id: ID!
  name: String!
  description: String
  facilityType: FacilityType!
  location: Location!
  address: Address!
  contact: Contact!
  rating: Float!
  reviewCount: Int!
  isActive: Boolean!
  acceptsNewPatients: Boolean!
  
  # Amenities
  hasEmergency: Boolean!
  hasParking: Boolean!
  wheelchairAccessible: Boolean!
  
  # Financial
  priceRange: PriceRange
  
  # Relations (resolved via dataloaders)
  insuranceProviders: [InsuranceProvider!]!
  specialties: [Specialty!]!
  procedures(limit: Int = 10, offset: Int = 0): ProcedureConnection!
  availableSlots(
    startDate: DateTime!
    endDate: DateTime!
    procedureId: ID
  ): [TimeSlot!]!
  
  # Metadata
  languagesSpoken: [String!]!
  avgWaitTime: Int
  nextAvailableSlot: DateTime
  createdAt: DateTime!
  updatedAt: DateTime!
}

type Procedure {
  id: ID!
  name: String!
  code: String!
  category: ProcedureCategory!
  description: String
  price: Float!
  duration: Int!
  requiresReferral: Boolean!
  preparationRequired: Boolean!
  facility: Facility!
  insuranceCoverage: [InsuranceProvider!]!
  isActive: Boolean!
}

type Appointment {
  id: ID!
  facility: Facility!
  patient: Patient
  procedure: Procedure!
  providerName: String!
  appointmentDate: DateTime!
  duration: Int!
  status: AppointmentStatus!
  price: Float!
  insuranceProvider: InsuranceProvider
  notes: String
  createdAt: DateTime!
}

type InsuranceProvider {
  id: ID!
  name: String!
  providerType: InsuranceType!
  networkTier: String
  isActive: Boolean!
  coverageStates: [String!]!
  facilitiesCount: Int!
  proceduresCount: Int!
}

type Patient {
  id: ID!
  name: String!
  email: String!
  phone: String!
}

# ============================================================================
# Enums
# ============================================================================

enum FacilityType {
  HOSPITAL
  CLINIC
  URGENT_CARE
  DIAGNOSTIC_LAB
  IMAGING_CENTER
  OUTPATIENT_SURGERY
  SPECIALTY_CLINIC
}

enum ProcedureCategory {
  DIAGNOSTIC
  SURGICAL
  THERAPEUTIC
  PREVENTIVE
  IMAGING
  LABORATORY
}

enum AppointmentStatus {
  SCHEDULED
  CONFIRMED
  IN_PROGRESS
  COMPLETED
  CANCELLED
  NO_SHOW
  RESCHEDULED
}

enum InsuranceType {
  PPO
  HMO
  EPO
  POS
  HDHP
  MEDICARE
  MEDICAID
}

enum SortOrder {
  ASC
  DESC
}

enum FacilitySortField {
  NAME
  RATING
  DISTANCE
  PRICE
  NEXT_AVAILABLE
  WAIT_TIME
}

# ============================================================================
# Supporting Types
# ============================================================================

type Location {
  latitude: Float!
  longitude: Float!
}

type Address {
  street: String!
  city: String!
  state: String!
  zipCode: String!
  country: String!
}

type Contact {
  phone: String!
  email: String
  website: String
}

type PriceRange {
  min: Float!
  max: Float!
  avg: Float!
}

type TimeSlot {
  startTime: DateTime!
  endTime: DateTime!
  available: Boolean!
  providerId: ID
  providerName: String
}

type Specialty {
  id: ID!
  name: String!
  description: String
}

# ============================================================================
# Search & Filter Inputs
# ============================================================================

input FacilitySearchInput {
  # Text search
  query: String
  
  # Geo search (required for most searches)
  location: LocationInput!
  radiusKm: Float! # Default: 50
  
  # Filters
  facilityTypes: [FacilityType!]
  insuranceProviders: [ID!]
  specialties: [String!]
  languages: [String!]
  
  # Price filtering
  maxPrice: Float
  minPrice: Float
  
  # Amenities
  acceptsNewPatients: Boolean
  hasEmergency: Boolean
  hasParking: Boolean
  wheelchairAccessible: Boolean
  
  # Availability
  needsAppointmentBy: DateTime
  
  # Rating
  minRating: Float
  
  # Sorting
  sortBy: FacilitySortField
  sortOrder: SortOrder
  
  # Pagination
  limit: Int # Default: 20, Max: 100
  offset: Int # Default: 0
}

input LocationInput {
  latitude: Float!
  longitude: Float!
}

input ProcedureSearchInput {
  query: String
  facilityId: ID
  category: ProcedureCategory
  maxPrice: Float
  requiresReferral: Boolean
  insuranceProviders: [ID!]
  limit: Int
  offset: Int
}

input AppointmentSearchInput {
  facilityId: ID
  patientId: ID
  status: AppointmentStatus
  startDate: DateTime
  endDate: DateTime
  limit: Int
  offset: Int
}

# ============================================================================
# Search Results & Pagination
# ============================================================================

type FacilitySearchResult {
  facilities: [Facility!]!
  facets: FacilityFacets!
  pagination: PaginationInfo!
  totalCount: Int!
  searchTime: Float! # milliseconds
}

type ProcedureConnection {
  nodes: [Procedure!]!
  pageInfo: PageInfo!
  totalCount: Int!
}

type FacilityFacets {
  facilityTypes: [FacetCount!]!
  insuranceProviders: [FacetCount!]!
  specialties: [FacetCount!]!
  cities: [FacetCount!]!
  states: [FacetCount!]!
  priceRanges: [PriceRangeFacet!]!
  ratingDistribution: [RatingFacet!]!
}

type FacetCount {
  value: String!
  count: Int!
}

type PriceRangeFacet {
  min: Float!
  max: Float!
  count: Int!
}

type RatingFacet {
  rating: Float!
  count: Int!
}

type PaginationInfo {
  hasNextPage: Boolean!
  hasPreviousPage: Boolean!
  currentPage: Int!
  totalPages: Int!
  limit: Int!
  offset: Int!
}

type PageInfo {
  hasNextPage: Boolean!
  hasPreviousPage: Boolean!
  startCursor: String
  endCursor: String
}

# ============================================================================
# Queries (Read Operations)
# ============================================================================

type Query {
  # Facility Queries
  facility(id: ID!): Facility
  facilities(
    filter: FacilitySearchInput!
  ): FacilitySearchResult!
  
  # Advanced search with typo tolerance
  searchFacilities(
    query: String!
    location: LocationInput!
    radiusKm: Float
    filters: FacilitySearchInput
  ): FacilitySearchResult!
  
  # Autocomplete/Typeahead
  facilitySuggestions(
    query: String!
    location: LocationInput!
    limit: Int
  ): [FacilitySuggestion!]!
  
  # Procedure Queries
  procedure(id: ID!): Procedure
  procedures(filter: ProcedureSearchInput!): ProcedureConnection!
  
  # Appointment Queries (Patient-specific)
  appointment(id: ID!): Appointment
  appointments(filter: AppointmentSearchInput!): [Appointment!]!
  
  # Insurance Queries
  insuranceProvider(id: ID!): InsuranceProvider
  insuranceProviders(
    isActive: Boolean
    state: String
    limit: Int
    offset: Int
  ): [InsuranceProvider!]!
  
  # Availability
  checkAvailability(
    facilityId: ID!
    procedureId: ID
    startDate: DateTime!
    endDate: DateTime!
  ): [TimeSlot!]!
  
  # Aggregations/Stats
  facilityStats: FacilityStats!
  priceComparison(procedureCode: String!, location: LocationInput!, radiusKm: Float!): PriceComparisonResult!
}

type FacilitySuggestion {
  id: ID!
  name: String!
  facilityType: FacilityType!
  city: String!
  state: String!
  distance: Float # km
  rating: Float!
}

type FacilityStats {
  totalFacilities: Int!
  totalProcedures: Int!
  avgRating: Float!
  avgWaitTime: Int!
  facilitiesByType: [FacetCount!]!
  topInsuranceProviders: [FacetCount!]!
}

type PriceComparisonResult {
  procedureCode: String!
  procedureName: String!
  minPrice: Float!
  maxPrice: Float!
  avgPrice: Float!
  facilities: [FacilityPriceInfo!]!
}

type FacilityPriceInfo {
  facility: Facility!
  price: Float!
  distance: Float!
  acceptsInsurance: [String!]!
}

# ============================================================================
# Mutations (Minimal - most writes go through REST API)
# ============================================================================

type Mutation {
  # These might just proxy to the REST API or be removed entirely
  # depending on your architecture decisions
  
  # For now, keep mutations minimal or none
  # All writes should go through the REST Command Service
}

# ============================================================================
# Subscriptions (Future Phase)
# ============================================================================

type Subscription {
  # Real-time updates for appointment availability
  availabilityChanged(facilityId: ID!): TimeSlot!
  
  # Price updates
  priceUpdated(procedureId: ID!): Procedure!
}

# ============================================================================
# Custom Scalars
# ============================================================================

scalar DateTime
scalar Upload
```

#### 2.3 gqlgen Configuration

**File:** `backend/internal/graphql/gqlgen.yml`

```yaml
# GraphQL schema location
schema:
  - internal/graphql/schema.graphql

# Where to generate the server
exec:
  filename: internal/graphql/generated/generated.go
  package: generated

# Where to generate the models
model:
  filename: internal/graphql/models/models_gen.go
  package: models

# Where the resolver implementations go
resolver:
  layout: follow-schema
  dir: internal/graphql/resolvers
  package: resolvers
  filename_template: "{name}.resolvers.go"

# Model mapping to existing domain entities
models:
  ID:
    model:
      - github.com/99designs/gqlgen/graphql.ID
      - github.com/99designs/gqlgen/graphql.Int
      - github.com/99designs/gqlgen/graphql.Int64
      - github.com/99designs/gqlgen/graphql.Int32
  DateTime:
    model:
      - github.com/99designs/gqlgen/graphql.Time
  
  # Map to existing domain entities
  Facility:
    model: github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/entities.Facility
  Procedure:
    model: github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/entities.Procedure
  Appointment:
    model: github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/entities.Appointment
  InsuranceProvider:
    model: github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/entities.Insurance

# Autobind allows gqlgen to match GraphQL types to Go types automatically
autobind:
  - github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/entities

# Directives
directives:
  goField:
    skip_runtime: true

# Complexity limits (prevent DoS)
complexity:
  default: 1
  facilities: 10
  procedures: 5
  appointments: 5

# Features
omit_slice_element_pointers: true
struct_tag: json
```

#### 2.4 TDD: Write Tests First

**Test Structure:**
```
backend/internal/graphql/resolvers/
├── query_test.go
├── facility_test.go
├── procedure_test.go
├── search_test.go
└── integration_test.go
```

**Example Test:** `query_test.go`

```go
package resolvers_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/entities"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/graphql/models"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/graphql/resolvers"
)

// Mock repository
type MockFacilityQueryService struct {
	mock.Mock
}

func (m *MockFacilityQueryService) Search(ctx context.Context, params SearchParams) (*SearchResult, error) {
	args := m.Called(ctx, params)
	return args.Get(0).(*SearchResult), args.Error(1)
}

func TestQueryResolver_SearchFacilities(t *testing.T) {
	// Arrange
	mockService := new(MockFacilityQueryService)
	resolver := resolvers.NewResolver(mockService, nil, nil, nil)
	
	ctx := context.Background()
	input := models.FacilitySearchInput{
		Location: &models.LocationInput{
			Latitude:  37.7749,
			Longitude: -122.4194,
		},
		RadiusKm: ptr(10.0),
		Limit:    ptr(20),
	}
	
	expectedFacility := &entities.Facility{
		ID:   "fac-1",
		Name: "Test Hospital",
		// ... other fields
	}
	
	mockService.On("Search", ctx, mock.Anything).Return(&SearchResult{
		Facilities: []*entities.Facility{expectedFacility},
		TotalCount: 1,
		SearchTime: 45.2,
	}, nil)
	
	// Act
	result, err := resolver.Query().SearchFacilities(ctx, "hospital", input.Location, input.RadiusKm, &input)
	
	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result.Facilities, 1)
	assert.Equal(t, "fac-1", result.Facilities[0].ID)
	assert.Equal(t, "Test Hospital", result.Facilities[0].Name)
	mockService.AssertExpectations(t)
}

func TestQueryResolver_Facility_NotFound(t *testing.T) {
	// Arrange
	mockService := new(MockFacilityQueryService)
	resolver := resolvers.NewResolver(mockService, nil, nil, nil)
	
	ctx := context.Background()
	
	mockService.On("GetByID", ctx, "non-existent").Return(nil, errors.New("not found"))
	
	// Act
	result, err := resolver.Query().Facility(ctx, "non-existent")
	
	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
}

func ptr[T any](v T) *T {
	return &v
}
```

---

### Phase 3: Query Services Implementation (Week 3)

#### 3.1 Facility Query Service

**File:** `backend/internal/query/services/facility_query_service.go`

```go
package services

import (
	"context"
	"fmt"
	"time"

	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/adapters/search"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/entities"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/repositories"
)

type FacilityQueryService struct {
	searchRepo search.FacilitySearchRepository
	dbRepo     repositories.FacilityRepository // Fallback for detailed data
	cache      CacheProvider
}

func NewFacilityQueryService(
	searchRepo search.FacilitySearchRepository,
	dbRepo repositories.FacilityRepository,
	cache CacheProvider,
) *FacilityQueryService {
	return &FacilityQueryService{
		searchRepo: searchRepo,
		dbRepo:     dbRepo,
		cache:      cache,
	}
}

type SearchParams struct {
	Query                string
	Latitude             float64
	Longitude            float64
	RadiusKm             float64
	FacilityTypes        []string
	InsuranceProviders   []string
	Specialties          []string
	MinRating            *float64
	MaxPrice             *float64
	MinPrice             *float64
	AcceptsNewPatients   *bool
	HasEmergency         *bool
	HasParking           *bool
	WheelchairAccessible *bool
	SortBy               string
	SortOrder            string
	Limit                int
	Offset               int
}

type SearchResult struct {
	Facilities []*entities.Facility
	Facets     *Facets
	TotalCount int
	SearchTime float64 // milliseconds
}

type Facets struct {
	FacilityTypes       map[string]int
	InsuranceProviders  map[string]int
	Cities              map[string]int
	States              map[string]int
	Specialties         map[string]int
	PriceRanges         []PriceRangeFacet
	RatingDistribution  []RatingFacet
}

type PriceRangeFacet struct {
	Min   float64
	Max   float64
	Count int
}

type RatingFacet struct {
	Rating float64
	Count  int
}

func (s *FacilityQueryService) Search(ctx context.Context, params SearchParams) (*SearchResult, error) {
	start := time.Now()
	
	// Build Typesense query
	tsParams := s.buildTypesenseParams(params)
	
	// Execute search
	facilities, facets, total, err := s.searchRepo.SearchWithFacets(ctx, tsParams)
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}
	
	// Enrich with cached/DB data if needed
	// ...
	
	searchTime := float64(time.Since(start).Milliseconds())
	
	return &SearchResult{
		Facilities: facilities,
		Facets:     facets,
		TotalCount: total,
		SearchTime: searchTime,
	}, nil
}

func (s *FacilityQueryService) GetByID(ctx context.Context, id string) (*entities.Facility, error) {
	// Try cache first
	if s.cache != nil {
		cached, err := s.cache.Get(ctx, "facility:"+id)
		if err == nil && cached != nil {
			return cached.(*entities.Facility), nil
		}
	}
	
	// Fall back to database for complete data
	facility, err := s.dbRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	
	// Cache result
	if s.cache != nil {
		_ = s.cache.Set(ctx, "facility:"+id, facility, 5*time.Minute)
	}
	
	return facility, nil
}

func (s *FacilityQueryService) Suggest(ctx context.Context, query string, lat, lon float64, limit int) ([]*FacilitySuggestion, error) {
	// Use Typesense autocomplete/prefix search
	// ...
	return nil, nil
}

func (s *FacilityQueryService) buildTypesenseParams(params SearchParams) *search.TypesenseSearchParams {
	// Convert GraphQL params to Typesense query
	// ...
	return nil
}
```

**Test File:** `facility_query_service_test.go`

```go
package services_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/entities"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/query/services"
)

func TestFacilityQueryService_Search_Success(t *testing.T) {
	// Arrange
	mockSearchRepo := new(MockSearchRepo)
	mockDBRepo := new(MockFacilityRepo)
	mockCache := new(MockCache)
	
	service := services.NewFacilityQueryService(mockSearchRepo, mockDBRepo, mockCache)
	
	ctx := context.Background()
	params := services.SearchParams{
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
	
	mockSearchRepo.On("SearchWithFacets", ctx, mock.Anything).Return(
		expectedFacilities,
		&services.Facets{},
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
	assert.Equal(t, 1, result.TotalCount)
	assert.Greater(t, result.SearchTime, 0.0)
	mockSearchRepo.AssertExpectations(t)
}

func TestFacilityQueryService_GetByID_CacheHit(t *testing.T) {
	// Test cache hit scenario
	// ...
}

func TestFacilityQueryService_GetByID_CacheMiss_DBFallback(t *testing.T) {
	// Test cache miss, falls back to DB
	// ...
}
```

#### 3.2 Enhanced Typesense Search Adapter

**File:** `backend/internal/adapters/search/typesense_facility_adapter.go`

```go
package search

import (
	"context"
	"fmt"

	"github.com/typesense/typesense-go/v2/typesense/api"
	"github.com/typesense/typesense-go/v2/typesense/api/pointer"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/entities"
	tsclient "github.com/zatekoja/Patientpricediscoverydesign/backend/internal/infrastructure/clients/typesense"
)

type TypesenseSearchParams struct {
	Query                string
	Latitude             float64
	Longitude            float64
	RadiusKm             float64
	Filters              []string
	FacetBy              []string
	SortBy               string
	Page                 int
	PerPage              int
	IncludeFields        []string
	ExcludeFields        []string
	HighlightFields      []string
	TypoTolerance        bool
	PrefixSearch         bool
}

type FacilitySearchAdapter struct {
	client *tsclient.Client
}

func NewFacilitySearchAdapter(client *tsclient.Client) *FacilitySearchAdapter {
	return &FacilitySearchAdapter{client: client}
}

func (a *FacilitySearchAdapter) SearchWithFacets(
	ctx context.Context,
	params *TypesenseSearchParams,
) ([]*entities.Facility, *Facets, int, error) {
	// Build query
	query := a.buildQuery(params)
	
	// Execute search
	result, err := a.client.Client().Collection("facilities").Documents().Search(ctx, query)
	if err != nil {
		return nil, nil, 0, fmt.Errorf("typesense search failed: %w", err)
	}
	
	// Parse results
	facilities := a.parseFacilities(result)
	facets := a.parseFacets(result)
	totalCount := int(*result.Found)
	
	return facilities, facets, totalCount, nil
}

func (a *FacilitySearchAdapter) buildQuery(params *TypesenseSearchParams) *api.SearchCollectionParams {
	// Build filter string
	filterStr := a.buildFilterString(params)
	
	// Build facet string
	facetStr := a.buildFacetString(params.FacetBy)
	
	query := &api.SearchCollectionParams{
		Q:       pointer.String(params.Query),
		QueryBy: pointer.String("name,description,specialties,city,state"),
		
		// Filters
		FilterBy: pointer.String(filterStr),
		
		// Facets for aggregations
		FacetBy: pointer.String(facetStr),
		MaxFacetValues: pointer.Int(100),
		
		// Sorting
		SortBy: pointer.String(params.SortBy),
		
		// Pagination
		Page:    pointer.Int(params.Page),
		PerPage: pointer.Int(params.PerPage),
		
		// Typo tolerance
		NumTypos: pointer.String("2"),
		
		// Geo search
		// Note: Typesense uses filter_by for geo queries
		// Already included in FilterBy above
		
		// Fields
		IncludeFields: pointer.String("*"),
		
		// Highlighting for search terms
		HighlightFullFields: pointer.String("name,description"),
	}
	
	return query
}

func (a *FacilitySearchAdapter) buildFilterString(params *TypesenseSearchParams) string {
	filters := []string{
		"is_active:=true",
	}
	
	// Geo filter
	if params.Latitude != 0 && params.Longitude != 0 {
		geoFilter := fmt.Sprintf(
			"location:(%f, %f, %f km)",
			params.Latitude,
			params.Longitude,
			params.RadiusKm,
		)
		filters = append(filters, geoFilter)
	}
	
	// Append additional filters
	filters = append(filters, params.Filters...)
	
	// Join with AND
	return joinFilters(filters, "&&")
}

func (a *FacilitySearchAdapter) buildFacetString(facetBy []string) string {
	if len(facetBy) == 0 {
		// Default facets
		return "facility_type,insurance_providers,city,state,rating"
	}
	return joinStrings(facetBy, ",")
}

func (a *FacilitySearchAdapter) parseFacilities(result *api.SearchResult) []*entities.Facility {
	facilities := make([]*entities.Facility, 0, len(*result.Hits))
	
	for _, hit := range *result.Hits {
		doc := *hit.Document
		
		facility := &entities.Facility{
			ID:           getString(doc, "id"),
			Name:         getString(doc, "name"),
			Description:  getString(doc, "description"),
			FacilityType: getString(doc, "facility_type"),
			IsActive:     getBool(doc, "is_active"),
			Rating:       getFloat(doc, "rating"),
			ReviewCount:  getInt(doc, "review_count"),
			// Parse location
			Location: a.parseLocation(doc),
			// ... parse other fields
		}
		
		facilities = append(facilities, facility)
	}
	
	return facilities
}

func (a *FacilitySearchAdapter) parseFacets(result *api.SearchResult) *Facets {
	if result.FacetCounts == nil {
		return &Facets{}
	}
	
	facets := &Facets{
		FacilityTypes:      make(map[string]int),
		InsuranceProviders: make(map[string]int),
		Cities:             make(map[string]int),
		States:             make(map[string]int),
	}
	
	for _, facetResult := range *result.FacetCounts {
		switch facetResult.FieldName {
		case "facility_type":
			for _, count := range facetResult.Counts {
				facets.FacilityTypes[count.Value] = int(count.Count)
			}
		case "insurance_providers":
			for _, count := range facetResult.Counts {
				facets.InsuranceProviders[count.Value] = int(count.Count)
			}
		case "city":
			for _, count := range facetResult.Counts {
				facets.Cities[count.Value] = int(count.Count)
			}
		case "state":
			for _, count := range facetResult.Counts {
				facets.States[count.Value] = int(count.Count)
			}
		}
	}
	
	return facets
}

// Helper functions
func getString(doc map[string]interface{}, key string) string {
	if val, ok := doc[key].(string); ok {
		return val
	}
	return ""
}

func getFloat(doc map[string]interface{}, key string) float64 {
	if val, ok := doc[key].(float64); ok {
		return val
	}
	return 0
}

func getInt(doc map[string]interface{}, key string) int {
	if val, ok := doc[key].(float64); ok {
		return int(val)
	}
	return 0
}

func getBool(doc map[string]interface{}, key string) bool {
	if val, ok := doc[key].(bool); ok {
		return val
	}
	return false
}

func (a *FacilitySearchAdapter) parseLocation(doc map[string]interface{}) entities.Location {
	if loc, ok := doc["location"].([]interface{}); ok && len(loc) == 2 {
		return entities.Location{
			Latitude:  loc[0].(float64),
			Longitude: loc[1].(float64),
		}
	}
	return entities.Location{}
}

func joinFilters(filters []string, sep string) string {
	// Join non-empty filters
	result := ""
	for i, f := range filters {
		if f != "" {
			if i > 0 {
				result += " " + sep + " "
			}
			result += f
		}
	}
	return result
}

func joinStrings(strs []string, sep string) string {
	result := ""
	for i, s := range strs {
		if i > 0 {
			result += sep
		}
		result += s
	}
	return result
}
```

---

### Phase 4: GraphQL Server Implementation (Week 4)

#### 4.1 Main GraphQL Server

**File:** `backend/cmd/graphql/main.go`

```go
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/linter"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/adapters/cache"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/adapters/database"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/adapters/search"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/graphql/generated"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/graphql/resolvers"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/infrastructure/clients/postgres"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/infrastructure/clients/redis"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/infrastructure/clients/typesense"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/infrastructure/observability"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/query/services"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/pkg/config"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Set up context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize OpenTelemetry
	var shutdown func(context.Context) error
	if cfg.OTEL.Enabled && cfg.OTEL.Endpoint != "" {
		shutdown, err = observability.Setup(
			ctx,
			cfg.OTEL.ServiceName+"-graphql",
			cfg.OTEL.ServiceVersion,
			cfg.OTEL.Endpoint,
		)
		if err != nil {
			log.Printf("Warning: Failed to set up OpenTelemetry: %v", err)
		} else {
			defer func() {
				shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				if err := shutdown(shutdownCtx); err != nil {
					log.Printf("Error shutting down OpenTelemetry: %v", err)
				}
			}()
			log.Println("OpenTelemetry initialized successfully")
		}
	}

	// Initialize clients
	pgClient, err := postgres.NewClient(&cfg.Database)
	if err != nil {
		log.Fatalf("Failed to initialize PostgreSQL client: %v", err)
	}
	defer pgClient.Close()

	redisClient, err := redis.NewClient(&cfg.Redis)
	if err != nil {
		log.Printf("Warning: Failed to initialize Redis client: %v", err)
	} else {
		defer redisClient.Close()
		log.Println("Redis client initialized successfully")
	}

	typesenseClient, err := typesense.NewClient(&cfg.Typesense)
	if err != nil {
		log.Fatalf("Failed to initialize Typesense client: %v", err)
	}
	log.Println("Typesense client initialized successfully")

	// Initialize adapters
	facilityDBAdapter := database.NewFacilityAdapter(pgClient)
	facilitySearchAdapter := search.NewFacilitySearchAdapter(typesenseClient)
	procedureSearchAdapter := search.NewProcedureSearchAdapter(typesenseClient)
	appointmentDBAdapter := database.NewAppointmentAdapter(pgClient)
	
	var cacheAdapter *cache.RedisAdapter
	if redisClient != nil {
		cacheAdapter = cache.NewRedisAdapter(redisClient)
	}

	// Initialize query services
	facilityQueryService := services.NewFacilityQueryService(
		facilitySearchAdapter,
		facilityDBAdapter,
		cacheAdapter,
	)
	
	procedureQueryService := services.NewProcedureQueryService(
		procedureSearchAdapter,
		// ... other dependencies
	)
	
	appointmentQueryService := services.NewAppointmentQueryService(
		appointmentDBAdapter,
		// ... other dependencies
	)

	// Initialize GraphQL resolver
	resolver := resolvers.NewResolver(
		facilityQueryService,
		procedureQueryService,
		appointmentQueryService,
		cacheAdapter,
	)

	// Create GraphQL server
	srv := handler.NewDefaultServer(generated.NewExecutableSchema(generated.Config{
		Resolvers: resolver,
	}))

	// Add extensions
	srv.AddTransport(transport.POST{})
	srv.AddTransport(transport.GET{})
	srv.Use(extension.Introspection{})
	srv.Use(extension.AutomaticPersistedQuery{
		Cache: &InMemoryCache{}, // or Redis cache
	})
	
	// Complexity limiting (prevent DoS)
	srv.Use(extension.FixedComplexityLimit(300))
	
	// Linter
	srv.Use(linter.Linter{})

	// Set up HTTP routes
	mux := http.NewServeMux()
	
	// GraphQL endpoint
	mux.Handle("/graphql", srv)
	
	// GraphQL Playground (dev only)
	if os.Getenv("ENV") != "production" {
		mux.Handle("/playground", playground.Handler("GraphQL Playground", "/graphql"))
		log.Println("GraphQL Playground available at http://localhost:8081/playground")
	}
	
	// Health check
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Create HTTP server
	serverAddr := fmt.Sprintf(":%d", 8081) // Different port from REST API
	server := &http.Server{
		Addr:         serverAddr,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	go func() {
		log.Printf("GraphQL server starting on %s", serverAddr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("GraphQL server shutting down...")

	// Graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("Error during server shutdown: %v", err)
	}

	log.Println("GraphQL server stopped")
}

// InMemoryCache for APQ (Automatic Persisted Queries)
type InMemoryCache struct {
	data map[string]interface{}
}

func (c *InMemoryCache) Get(ctx context.Context, key string) (interface{}, bool) {
	val, ok := c.data[key]
	return val, ok
}

func (c *InMemoryCache) Set(ctx context.Context, key string, value interface{}) {
	if c.data == nil {
		c.data = make(map[string]interface{})
	}
	c.data[key] = value
}
```

#### 4.2 GraphQL Resolvers

**File:** `backend/internal/graphql/resolvers/resolver.go`

```go
package resolvers

import (
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/query/services"
)

// Resolver is the root resolver
type Resolver struct {
	facilityQueryService    *services.FacilityQueryService
	procedureQueryService   *services.ProcedureQueryService
	appointmentQueryService *services.AppointmentQueryService
	cache                   CacheProvider
}

func NewResolver(
	facilityQueryService *services.FacilityQueryService,
	procedureQueryService *services.ProcedureQueryService,
	appointmentQueryService *services.AppointmentQueryService,
	cache CacheProvider,
) *Resolver {
	return &Resolver{
		facilityQueryService:    facilityQueryService,
		procedureQueryService:   procedureQueryService,
		appointmentQueryService: appointmentQueryService,
		cache:                   cache,
	}
}
```

**File:** `backend/internal/graphql/resolvers/query.resolvers.go`

```go
package resolvers

import (
	"context"

	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/entities"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/graphql/generated"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/graphql/models"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/query/services"
)

// Query returns the query resolver
func (r *Resolver) Query() generated.QueryResolver {
	return &queryResolver{r}
}

type queryResolver struct{ *Resolver }

// Facility resolver
func (r *queryResolver) Facility(ctx context.Context, id string) (*entities.Facility, error) {
	return r.facilityQueryService.GetByID(ctx, id)
}

// Facilities resolver
func (r *queryResolver) Facilities(
	ctx context.Context,
	filter models.FacilitySearchInput,
) (*models.FacilitySearchResult, error) {
	// Convert GraphQL input to service params
	params := r.convertToServiceParams(filter)
	
	// Execute search
	result, err := r.facilityQueryService.Search(ctx, params)
	if err != nil {
		return nil, err
	}
	
	// Convert to GraphQL result
	return r.convertToGraphQLResult(result), nil
}

// SearchFacilities resolver (with query string)
func (r *queryResolver) SearchFacilities(
	ctx context.Context,
	query string,
	location *models.LocationInput,
	radiusKm *float64,
	filters *models.FacilitySearchInput,
) (*models.FacilitySearchResult, error) {
	// Build params from query and filters
	params := services.SearchParams{
		Query:     query,
		Latitude:  location.Latitude,
		Longitude: location.Longitude,
		RadiusKm:  *radiusKm,
		Limit:     20,
		Offset:    0,
	}
	
	if filters != nil {
		// Apply additional filters
		params.FacilityTypes = filters.FacilityTypes
		params.InsuranceProviders = filters.InsuranceProviders
		params.Specialties = filters.Specialties
		// ... other fields
	}
	
	// Execute search
	result, err := r.facilityQueryService.Search(ctx, params)
	if err != nil {
		return nil, err
	}
	
	return r.convertToGraphQLResult(result), nil
}

// FacilitySuggestions resolver (autocomplete)
func (r *queryResolver) FacilitySuggestions(
	ctx context.Context,
	query string,
	location *models.LocationInput,
	limit *int,
) ([]*models.FacilitySuggestion, error) {
	l := 10
	if limit != nil {
		l = *limit
	}
	
	suggestions, err := r.facilityQueryService.Suggest(ctx, query, location.Latitude, location.Longitude, l)
	if err != nil {
		return nil, err
	}
	
	// Convert to GraphQL type
	result := make([]*models.FacilitySuggestion, len(suggestions))
	for i, s := range suggestions {
		result[i] = &models.FacilitySuggestion{
			ID:           s.ID,
			Name:         s.Name,
			FacilityType: models.FacilityType(s.FacilityType),
			City:         s.City,
			State:        s.State,
			Distance:     &s.Distance,
			Rating:       s.Rating,
		}
	}
	
	return result, nil
}

// Procedure resolver
func (r *queryResolver) Procedure(ctx context.Context, id string) (*entities.Procedure, error) {
	return r.procedureQueryService.GetByID(ctx, id)
}

// ... other resolvers

// Helper functions
func (r *queryResolver) convertToServiceParams(filter models.FacilitySearchInput) services.SearchParams {
	params := services.SearchParams{
		Latitude:  filter.Location.Latitude,
		Longitude: filter.Location.Longitude,
		RadiusKm:  *filter.RadiusKm,
		Limit:     20,
		Offset:    0,
	}
	
	if filter.Query != nil {
		params.Query = *filter.Query
	}
	
	if filter.FacilityTypes != nil {
		params.FacilityTypes = filter.FacilityTypes
	}
	
	// ... map other fields
	
	return params
}

func (r *queryResolver) convertToGraphQLResult(result *services.SearchResult) *models.FacilitySearchResult {
	return &models.FacilitySearchResult{
		Facilities: result.Facilities,
		Facets:     r.convertFacets(result.Facets),
		Pagination: r.buildPagination(result),
		TotalCount: result.TotalCount,
		SearchTime: result.SearchTime,
	}
}

func (r *queryResolver) convertFacets(facets *services.Facets) *models.FacilityFacets {
	// Convert facets to GraphQL format
	// ...
	return &models.FacilityFacets{}
}

func (r *queryResolver) buildPagination(result *services.SearchResult) *models.PaginationInfo {
	// Build pagination info
	// ...
	return &models.PaginationInfo{}
}
```

---

### Phase 5: Deployment & Integration (Week 5)

#### 5.1 Docker Setup

**File:** `backend/graphql.Dockerfile`

```dockerfile
# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build GraphQL server
RUN CGO_ENABLED=0 GOOS=linux go build -o /graphql-server ./cmd/graphql

# Run stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

COPY --from=builder /graphql-server .

EXPOSE 8081

CMD ["./graphql-server"]
```

**Update:** `backend/docker-compose.yml`

```yaml
services:
  # ... existing services (postgres, redis, typesense, etc.)
  
  # REST API (Command Service)
  api:
    build:
      context: .
      dockerfile: Dockerfile
    container_name: ppd_api
    ports:
      - "8080:8080"
    environment:
      - SERVER_PORT=8080
      - DB_HOST=postgres
      - DB_PORT=5432
      - DB_USER=postgres
      - DB_PASSWORD=postgres
      - DB_NAME=patient_price_discovery
      - REDIS_HOST=redis
      - REDIS_PORT=6379
      - TYPESENSE_URL=http://typesense:8108
      - TYPESENSE_API_KEY=xyz
      - OTEL_ENDPOINT=http://signoz-otel-collector:4317
      - OTEL_ENABLED=true
    depends_on:
      - postgres
      - redis
      - typesense
      - signoz-otel-collector
    networks:
      - ppd_network

  # GraphQL Query Service (NEW)
  graphql:
    build:
      context: .
      dockerfile: graphql.Dockerfile
    container_name: ppd_graphql
    ports:
      - "8081:8081"
    environment:
      - SERVER_PORT=8081
      - DB_HOST=postgres
      - DB_PORT=5432
      - DB_USER=postgres
      - DB_PASSWORD=postgres
      - DB_NAME=patient_price_discovery
      - REDIS_HOST=redis
      - REDIS_PORT=6379
      - TYPESENSE_URL=http://typesense:8108
      - TYPESENSE_API_KEY=xyz
      - OTEL_ENDPOINT=http://signoz-otel-collector:4317
      - OTEL_ENABLED=true
      - ENV=development
    depends_on:
      - postgres
      - redis
      - typesense
      - signoz-otel-collector
    networks:
      - ppd_network

networks:
  ppd_network:
    driver: bridge
```

#### 5.2 Makefile Updates

**File:** `backend/Makefile`

```makefile
# ... existing targets ...

.PHONY: graphql-generate
graphql-generate:
	cd internal/graphql && go run github.com/99designs/gqlgen generate

.PHONY: run-graphql
run-graphql:
	go run cmd/graphql/main.go

.PHONY: test-graphql
test-graphql:
	go test -v -race ./internal/graphql/... ./internal/query/...

.PHONY: docker-build-graphql
docker-build-graphql:
	docker build -t ppd-graphql:latest -f graphql.Dockerfile .

.PHONY: docker-up-graphql
docker-up-graphql:
	docker-compose up -d graphql

.PHONY: docker-logs-graphql
docker-logs-graphql:
	docker-compose logs -f graphql

.PHONY: index-data
index-data:
	go run cmd/indexer/main.go --action=reindex

.PHONY: index-data-dry-run
index-data-dry-run:
	go run cmd/indexer/main.go --action=reindex --dry-run
```

---

### Phase 6: Frontend Integration (Week 6)

#### 6.1 Install Apollo Client

```bash
cd frontend
npm install @apollo/client graphql
```

#### 6.2 Apollo Client Setup

**File:** `src/lib/apollo-client.ts`

```typescript
import { ApolloClient, InMemoryCache, HttpLink } from '@apollo/client';

const httpLink = new HttpLink({
  uri: import.meta.env.VITE_GRAPHQL_URL || 'http://localhost:8081/graphql',
});

export const apolloClient = new ApolloClient({
  link: httpLink,
  cache: new InMemoryCache({
    typePolicies: {
      Query: {
        fields: {
          facilities: {
            keyArgs: ['filter'],
            merge(existing, incoming, { args }) {
              // Handle pagination
              if (args?.filter?.offset === 0) {
                return incoming;
              }
              return {
                ...incoming,
                facilities: [
                  ...(existing?.facilities || []),
                  ...incoming.facilities,
                ],
              };
            },
          },
        },
      },
    },
  }),
  defaultOptions: {
    watchQuery: {
      fetchPolicy: 'cache-and-network',
    },
  },
});
```

#### 6.3 GraphQL Queries

**File:** `src/lib/graphql/queries.ts`

```typescript
import { gql } from '@apollo/client';

export const SEARCH_FACILITIES = gql`
  query SearchFacilities(
    $query: String!
    $location: LocationInput!
    $radiusKm: Float
    $filters: FacilitySearchInput
  ) {
    searchFacilities(
      query: $query
      location: $location
      radiusKm: $radiusKm
      filters: $filters
    ) {
      facilities {
        id
        name
        facilityType
        location {
          latitude
          longitude
        }
        address {
          street
          city
          state
          zipCode
        }
        contact {
          phone
          email
          website
        }
        rating
        reviewCount
        priceRange {
          min
          max
          avg
        }
        insuranceProviders {
          id
          name
        }
        nextAvailableSlot
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
        cities {
          value
          count
        }
      }
      pagination {
        hasNextPage
        currentPage
        totalPages
        totalCount
      }
      totalCount
      searchTime
    }
  }
`;

export const GET_FACILITY = gql`
  query GetFacility($id: ID!) {
    facility(id: $id) {
      id
      name
      description
      facilityType
      location {
        latitude
        longitude
      }
      address {
        street
        city
        state
        zipCode
      }
      contact {
        phone
        email
        website
      }
      rating
      reviewCount
      acceptsNewPatients
      hasEmergency
      hasParking
      wheelchairAccessible
      insuranceProviders {
        id
        name
        providerType
      }
      specialties {
        id
        name
      }
      procedures(limit: 10) {
        nodes {
          id
          name
          price
          duration
        }
      }
      availableSlots(
        startDate: $startDate
        endDate: $endDate
      ) {
        startTime
        endTime
        available
      }
    }
  }
`;

export const FACILITY_SUGGESTIONS = gql`
  query FacilitySuggestions(
    $query: String!
    $location: LocationInput!
    $limit: Int
  ) {
    facilitySuggestions(
      query: $query
      location: $location
      limit: $limit
    ) {
      id
      name
      facilityType
      city
      state
      distance
      rating
    }
  }
`;
```

#### 6.4 React Hook

**File:** `src/hooks/useFacilitySearch.ts`

```typescript
import { useQuery } from '@apollo/client';
import { SEARCH_FACILITIES } from '@/lib/graphql/queries';

export interface FacilitySearchParams {
  query: string;
  latitude: number;
  longitude: number;
  radiusKm?: number;
  filters?: {
    facilityTypes?: string[];
    insuranceProviders?: string[];
    maxPrice?: number;
    minRating?: number;
  };
}

export const useFacilitySearch = (params: FacilitySearchParams) => {
  const { data, loading, error, fetchMore } = useQuery(SEARCH_FACILITIES, {
    variables: {
      query: params.query,
      location: {
        latitude: params.latitude,
        longitude: params.longitude,
      },
      radiusKm: params.radiusKm || 50,
      filters: params.filters,
    },
    skip: !params.query || !params.latitude || !params.longitude,
  });

  const loadMore = () => {
    if (!data?.searchFacilities?.pagination?.hasNextPage) return;

    fetchMore({
      variables: {
        filters: {
          ...params.filters,
          offset: data.searchFacilities.facilities.length,
        },
      },
    });
  };

  return {
    facilities: data?.searchFacilities?.facilities || [],
    facets: data?.searchFacilities?.facets,
    pagination: data?.searchFacilities?.pagination,
    totalCount: data?.searchFacilities?.totalCount || 0,
    searchTime: data?.searchFacilities?.searchTime,
    loading,
    error,
    loadMore,
  };
};
```

---

## Testing Strategy

### Unit Tests
- [ ] Typesense adapter tests
- [ ] Query service tests
- [ ] GraphQL resolver tests
- [ ] Search builder tests

### Integration Tests
- [ ] Typesense integration tests
- [ ] GraphQL server integration tests
- [ ] End-to-end search flow tests

### Performance Tests
- [ ] Search latency benchmarks
- [ ] Concurrent query tests
- [ ] Load testing (100+ concurrent users)
- [ ] Complexity limit tests

### Test Coverage Goals
- Unit tests: >80%
- Integration tests: >60%
- Critical paths: 100%

---

## Monitoring & Observability

### Metrics to Track
- Search query latency (p50, p95, p99)
- Typesense response times
- Cache hit rates
- GraphQL query complexity
- Concurrent connections
- Error rates by resolver
- Facet computation time

### Dashboards
- GraphQL performance dashboard
- Search analytics dashboard
- CQRS sync lag monitoring
- Service health dashboard

---

## Rollout Plan

### Week 1-2: Foundation
- Enhanced Typesense schema
- Data sync service
- Initial indexing

### Week 3-4: GraphQL Development
- Schema implementation
- Resolver development
- Query services

### Week 5: Integration & Testing
- Docker setup
- Integration tests
- Performance tuning

### Week 6: Frontend Integration
- Apollo Client setup
- Component migration
- A/B testing

### Week 7: Production Deployment
- Gradual rollout (10% → 50% → 100%)
- Monitoring
- Performance validation

---

## Success Metrics

### Performance
- Search latency < 100ms (p95)
- GraphQL query latency < 200ms (p95)
- Support 1000+ concurrent users
- Cache hit rate > 70%

### User Experience
- Typo-tolerant search
- Real-time filtering
- Accurate facet counts
- Smooth pagination

### Scalability
- Independent scaling of query/command services
- Horizontal scaling support
- No single point of failure

---

## Future Enhancements (Phase 7+)

1. **Event-Driven Sync**: Replace synchronous sync with message queue (RabbitMQ/Kafka)
2. **GraphQL Subscriptions**: Real-time updates for availability
3. **Personalization**: User-specific search ranking
4. **AI-Powered Search**: Natural language queries, semantic search
5. **Multi-language Support**: Internationalization
6. **Advanced Analytics**: Search patterns, popular procedures
7. **Federated GraphQL**: Combine multiple data sources
8. **Rate Limiting & Throttling**: Per-user query limits

---

## References & Resources

- [Typesense Documentation](https://typesense.org/docs/)
- [gqlgen Documentation](https://gqlgen.com/)
- [CQRS Pattern](https://martinfowler.com/bliki/CQRS.html)
- [GraphQL Best Practices](https://graphql.org/learn/best-practices/)
- [Apollo Client Documentation](https://www.apollographql.com/docs/react/)
