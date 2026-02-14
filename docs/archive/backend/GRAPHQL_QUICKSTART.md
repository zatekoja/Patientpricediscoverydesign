# GraphQL Query Service - Quick Start Guide

## Overview

This GraphQL service is part of our CQRS architecture, handling all **read operations** for the Patient Price Discovery system. It provides a flexible, performant query API powered by Typesense for lightning-fast search capabilities.

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    Frontend (React)                          │
└────────────┬────────────────────────────────┬───────────────┘
             │                                │
    Queries  │                                │  Mutations
   (GraphQL) │                                │  (REST)
   Port 8081 │                                │  Port 8080
             ▼                                ▼
┌────────────────────────┐      ┌────────────────────────────┐
│  GraphQL Query Service │      │   REST Command Service     │
│  - Read-only           │      │   - Write operations       │
│  - Typesense queries   │      │   - Business validation    │
│  - Complex filtering   │      │   - PostgreSQL writes      │
│  - Geo search          │      │   - Sync to Typesense      │
└───────────┬────────────┘      └─────────────┬──────────────┘
            │                                  │
            │ Query                            │ Write + Sync
            ▼                                  ▼
┌────────────────────────┐      ┌────────────────────────────┐
│      Typesense         │◄─────│      PostgreSQL            │
│   (Read Model)         │ Sync │   (Write Model)            │
└────────────────────────┘      └────────────────────────────┘
```

## Quick Start

### 1. Install Dependencies

```bash
cd backend
make deps
make install-tools  # Installs gqlgen and other tools
```

### 2. Generate GraphQL Code

```bash
make graphql-generate
```

This will:
- Read `internal/graphql/schema.graphql`
- Generate resolvers, models, and server code
- Create files in `internal/graphql/generated/`

### 3. Run Services

#### Option A: Run Locally

```bash
# Terminal 1: Start infrastructure (Postgres, Redis, Typesense)
make docker-up

# Terminal 2: Run REST API (Command Service)
make run

# Terminal 3: Run GraphQL Server (Query Service)
make run-graphql

# Terminal 4: Index data to Typesense
make index-data
```

#### Option B: Run with Docker

```bash
# Start all services
docker-compose up -d

# View logs
docker-compose logs -f graphql
```

### 4. Access GraphQL Playground

Open your browser:
- **GraphQL Playground**: http://localhost:8081/playground
- **GraphQL Endpoint**: http://localhost:8081/graphql

### 5. Test Your First Query

```graphql
query SearchNearbyHospitals {
  searchFacilities(
    query: "hospital"
    location: { latitude: 37.7749, longitude: -122.4194 }
    radiusKm: 10
  ) {
    facilities {
      id
      name
      facilityType
      rating
      reviewCount
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
      }
      priceRange {
        min
        max
        avg
      }
    }
    facets {
      facilityTypes {
        value
        count
      }
      cities {
        value
        count
      }
    }
    totalCount
    searchTime
  }
}
```

## Development Workflow (TDD)

We follow **Test-Driven Development**. Here's the workflow:

### 1. Write Tests First

```bash
# Create a test file
touch internal/query/services/my_service_test.go

# Write your tests
# See facility_query_service_test.go for examples
```

### 2. Run Tests (They Should Fail)

```bash
make test-query  # Run query service tests
make test-graphql  # Run GraphQL tests
```

### 3. Implement Code to Pass Tests

```bash
# Implement the service
vim internal/query/services/my_service.go
```

### 4. Verify Tests Pass

```bash
make test-query
```

### 5. Generate GraphQL Code

```bash
make graphql-generate
```

### 6. Implement Resolvers

```bash
vim internal/graphql/resolvers/query.resolvers.go
```

## Project Structure

```
backend/
├── cmd/
│   ├── api/              # REST API server (Command Service)
│   ├── graphql/          # GraphQL server (Query Service) - TO BE CREATED
│   └── indexer/          # Data sync utility
├── internal/
│   ├── adapters/
│   │   ├── database/     # PostgreSQL adapters
│   │   ├── search/       # Typesense adapters
│   │   └── cache/        # Redis adapters
│   ├── query/            # Query services (NEW)
│   │   └── services/
│   │       ├── facility_query_service.go
│   │       └── facility_query_service_test.go
│   ├── graphql/          # GraphQL layer (NEW)
│   │   ├── schema.graphql        # GraphQL schema definition
│   │   ├── generated/            # Auto-generated code
│   │   ├── models/               # GraphQL models
│   │   ├── resolvers/            # Resolver implementations
│   │   ├── loaders/              # DataLoaders (N+1 prevention)
│   │   └── middleware/           # GraphQL middleware
│   ├── application/      # Application services
│   ├── domain/           # Domain entities & interfaces
│   └── infrastructure/   # Infrastructure clients
├── gqlgen.yml            # gqlgen configuration
└── Makefile              # Build commands
```

## Available Make Commands

### Development
```bash
make help                  # Show all commands
make deps                  # Install dependencies
make install-tools         # Install dev tools (gqlgen, etc.)
```

### GraphQL
```bash
make graphql-generate      # Generate GraphQL code
make graphql-init          # Initialize GraphQL (first time)
make run-graphql           # Run GraphQL server
make test-graphql          # Test GraphQL resolvers
make build-graphql         # Build GraphQL binary
```

### Query Services
```bash
make test-query            # Test query services
```

### REST API
```bash
make run                   # Run REST API server
make build                 # Build REST API binary
```

### Testing
```bash
make test                  # Run all tests
make test-unit             # Unit tests only
make test-integration      # Integration tests
make test-coverage         # Generate coverage report
make test-race             # Run with race detection
```

### Docker
```bash
make docker-up             # Start all services
make docker-down           # Stop all services
make docker-logs           # View logs
make docker-logs-graphql   # View GraphQL logs only
```

### Data Sync
```bash
make index-data            # Sync PostgreSQL → Typesense
make index-data-dry-run    # Dry run
```

## Testing

### Run All Tests

```bash
make test
```

### Run Specific Tests

```bash
# Query service tests
go test -v ./internal/query/services/... -run TestFacilityQueryService_Search

# GraphQL tests (once implemented)
go test -v ./internal/graphql/resolvers/... -run TestQuery

# Integration tests
make test-integration
```

### Test Coverage

```bash
make test-coverage
# Opens coverage.html in browser
```

## GraphQL Schema

The schema is defined in `internal/graphql/schema.graphql`. Key features:

### Queries
- `facility(id)` - Get single facility
- `facilities(filter)` - Search with filters
- `searchFacilities(query, location)` - Full-text search
- `facilitySuggestions(query)` - Autocomplete
- `procedure(id)` - Get procedure
- `procedures(filter)` - Search procedures
- `insuranceProviders()` - List insurance
- `facilityStats` - Aggregations

### Features
- **Geo-search**: Distance-based facility search
- **Faceted search**: Aggregations by type, city, insurance, etc.
- **Typo tolerance**: Typesense handles typos automatically
- **Flexible filtering**: Price, rating, amenities, etc.
- **Pagination**: Offset/limit with metadata
- **Sorting**: By distance, rating, price, etc.

### Example Queries

#### Get Facility Details
```graphql
query GetFacility {
  facility(id: "fac-123") {
    id
    name
    description
    rating
    procedures(limit: 5) {
      nodes {
        id
        name
        price
      }
    }
  }
}
```

#### Search with Filters
```graphql
query SearchWithFilters {
  facilities(
    filter: {
      location: { latitude: 37.7749, longitude: -122.4194 }
      radiusKm: 50
      facilityTypes: [HOSPITAL, CLINIC]
      minRating: 4.0
      hasParking: true
      acceptsNewPatients: true
      limit: 20
      offset: 0
    }
  ) {
    facilities {
      id
      name
      rating
      distance
    }
    facets {
      facilityTypes {
        value
        count
      }
    }
    pagination {
      hasNextPage
      totalPages
    }
    totalCount
  }
}
```

#### Autocomplete Suggestions
```graphql
query AutoComplete {
  facilitySuggestions(
    query: "st mary"
    location: { latitude: 37.7749, longitude: -122.4194 }
    limit: 5
  ) {
    id
    name
    city
    state
    distance
    rating
  }
}
```

## Configuration

### Environment Variables

```bash
# GraphQL Server
SERVER_PORT=8081                    # Default: 8081
ENV=development                     # development | production

# Database (for fallback queries)
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=patient_price_discovery

# Typesense (primary query source)
TYPESENSE_URL=http://localhost:8108
TYPESENSE_API_KEY=xyz

# Redis (caching)
REDIS_HOST=localhost
REDIS_PORT=6379

# Observability
OTEL_ENDPOINT=http://localhost:4317
OTEL_ENABLED=true
OTEL_SERVICE_NAME=patient-price-discovery-graphql
```

### gqlgen Configuration

Edit `gqlgen.yml` to:
- Change schema location
- Configure model mappings
- Set resolver layout
- Enable/disable features

After changes, regenerate:
```bash
make graphql-generate
```

## Performance Optimization

### 1. DataLoaders (Prevent N+1)

```go
// internal/graphql/loaders/dataloader.go
type Loaders struct {
    FacilityLoader *FacilityLoader
    ProcedureLoader *ProcedureLoader
}

// Use in resolvers to batch load related data
```

### 2. Caching Strategy

```
┌─────────────────────┐
│   Redis Cache       │  5-minute TTL
│   - Facility details│
│   - Static data     │
└─────────────────────┘
          ↓ Cache Miss
┌─────────────────────┐
│   Typesense         │  Primary query source
│   - Search results  │
│   - Facets          │
└─────────────────────┘
          ↓ Detailed data needed
┌─────────────────────┐
│   PostgreSQL        │  Fallback for relations
│   - Full entity     │
└─────────────────────┘
```

### 3. Complexity Limiting

GraphQL queries are limited by complexity to prevent DoS:

```yaml
# gqlgen.yml
complexity:
  default: 1
  facilities: 10
  procedures: 5
```

### 4. Pagination

Use offset/limit pagination for large result sets:

```graphql
query PaginatedSearch {
  facilities(
    filter: {
      location: { latitude: 37.7749, longitude: -122.4194 }
      radiusKm: 50
      limit: 20
      offset: 0
    }
  ) {
    facilities { ... }
    pagination {
      hasNextPage
      currentPage
      totalPages
    }
  }
}
```

## Monitoring

### Metrics

The GraphQL server exposes metrics via OpenTelemetry:

- Query latency (p50, p95, p99)
- Query complexity
- Resolver execution time
- Cache hit/miss rates
- Error rates

### Dashboards

Access observability:
- **SigNoz**: http://localhost:3301
- **Health Check**: http://localhost:8081/health

## Troubleshooting

### GraphQL Generation Fails

```bash
# Clear generated files
rm -rf internal/graphql/generated/*

# Regenerate
make graphql-generate
```

### Tests Failing

```bash
# Ensure dependencies are up to date
make deps

# Run tests with verbose output
go test -v ./internal/query/services/... -run TestFacilityQueryService
```

### Typesense Connection Issues

```bash
# Check if Typesense is running
curl http://localhost:8108/health

# Restart Typesense
docker-compose restart typesense

# Check logs
docker-compose logs typesense
```

### Port Already in Use

```bash
# Find process using port 8081
lsof -i :8081

# Kill process
kill -9 <PID>
```

## Next Steps

1. **Implement GraphQL Server** (`cmd/graphql/main.go`)
2. **Create Resolvers** (`internal/graphql/resolvers/`)
3. **Enhanced Typesense Adapter** (faceted search, suggestions)
4. **DataLoaders** (N+1 prevention)
5. **Integration Tests** (end-to-end GraphQL tests)
6. **Frontend Integration** (Apollo Client)

## Resources

- [GraphQL Implementation Plan](./GRAPHQL_IMPLEMENTATION_PLAN.md)
- [Phase 2 CQRS Plan](../PHASE2_PLAN_CQRS_GRAPHQL.md)
- [gqlgen Documentation](https://gqlgen.com/)
- [Typesense Documentation](https://typesense.org/docs/)
- [Apollo Client](https://www.apollographql.com/docs/react/)

## Contributing

1. Write tests first (TDD)
2. Implement code to pass tests
3. Generate GraphQL code: `make graphql-generate`
4. Run all tests: `make test`
5. Check coverage: `make test-coverage`
6. Format code: `make fmt`
7. Submit PR

## License

See [LICENSE](../LICENSE)
