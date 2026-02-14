# Phase 1 Implementation Plan

## Overview
Phase 1 establishes the foundation for a performant, well-architected Go backend following Domain-Driven Design, Test-Driven Development, and OpenTelemetry standards.

## Architecture Principles

### 1. Domain-Driven Design (DDD)
- **Domain Layer**: Contains core business entities, repository interfaces, and provider interfaces
- **Application Layer**: Contains use cases and business services
- **Infrastructure Layer**: Contains concrete implementations (database clients, external service clients)
- **Adapters Layer**: Implements repository interfaces and connects domain to infrastructure
- **API Layer**: HTTP handlers, routes, and middleware

### 2. Test-Driven Development (TDD)
- Write tests before implementation
- Use mockery to generate mocks from interfaces
- Mock external dependencies (database, cache, external APIs)
- Focus on unit tests for business logic
- Integration tests for data flow

### 3. OpenTelemetry (OTEL) Standards
- Distributed tracing for all requests
- Key metrics emitted:
  - HTTP request count and duration
  - Database query duration
  - Cache hit/miss rates
- Span attributes for debugging
- Graceful degradation if OTEL is unavailable

## Data Flow Architecture

```
HTTP Request
    ↓
API Handler (converts HTTP to domain types)
    ↓
Internal Provider/Service (business logic)
    ↓
Adapter (implements repository interface)
    ↓
Client (PostgreSQL/Redis) OR External Provider (Maps API)
```

## Separation of Responsibilities

### API Layer
- **Responsibility**: Handle HTTP requests/responses
- **Dependencies**: Handlers depend on repositories (interfaces only)
- **No**: Business logic or direct database access

### Application Layer (Future)
- **Responsibility**: Orchestrate business operations
- **Dependencies**: Use domain repositories and providers
- **No**: HTTP concerns or database implementation details

### Domain Layer
- **Responsibility**: Define business entities and contracts
- **Dependencies**: None (pure Go types and interfaces)
- **No**: Framework dependencies or implementation details

### Adapters Layer
- **Responsibility**: Implement repository interfaces
- **Dependencies**: Database clients, domain entities
- **No**: Business logic or HTTP concerns

### Infrastructure Layer
- **Responsibility**: Provide connections to databases and external services
- **Dependencies**: External libraries (PostgreSQL, Redis drivers)
- **No**: Business logic

## Phase 1 Deliverables

### ✅ Completed

1. **Domain Entities** (`internal/domain/entities/`)
   - Facility
   - Procedure
   - Appointment
   - Insurance
   - User
   - Review

2. **Repository Interfaces** (`internal/domain/repositories/`)
   - FacilityRepository
   - ProcedureRepository
   - AppointmentRepository
   - InsuranceRepository
   - UserRepository

3. **Provider Interfaces** (`internal/domain/providers/`)
   - GeolocationProvider
   - CacheProvider

4. **Database Client** (`internal/infrastructure/clients/postgres/`)
   - PostgreSQL connection management
   - Connection pooling
   - Health checks

5. **Cache Client** (`internal/infrastructure/clients/redis/`)
   - Redis connection management
   - Health checks

6. **Adapters**
   - FacilityAdapter (implements FacilityRepository)
   - RedisAdapter (implements CacheProvider)
   - MockGeolocationProvider (for testing)

7. **Configuration** (`pkg/config/`)
   - Environment-based configuration
   - Server, Database, Redis, OTEL settings

8. **Error Handling** (`pkg/errors/`)
   - Custom error types
   - Error categorization

9. **Observability** (`internal/infrastructure/observability/`)
   - OTEL setup
   - Metrics initialization
   - Tracing helpers

10. **API Layer** (`internal/api/`)
    - Facility handler
    - Logging middleware
    - CORS middleware
    - Observability middleware
    - Router configuration

11. **Main Application** (`cmd/api/main.go`)
    - Dependency injection
    - Graceful shutdown
    - Component initialization

12. **Database Schema** (`migrations/001_initial_schema.sql`)
    - All tables with proper indexes
    - Foreign key relationships
    - Constraints

13. **Testing Infrastructure**
    - Mockery configuration (`.mockery.yml`)
    - Sample test structure
    - Test organization

14. **Development Tools**
    - Makefile with common commands
    - Docker Compose for local development
    - .gitignore
    - Documentation

## Mock Strategy

Using `mockery` for generating mocks:

```yaml
# .mockery.yml
with-expecter: true
dir: "tests/mocks"
packages:
  github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/repositories:
    interfaces:
      FacilityRepository:
      ProcedureRepository:
      # ... other repositories
  github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/providers:
    interfaces:
      GeolocationProvider:
      CacheProvider:
```

Generate mocks with:
```bash
mockery
```

## Key Metrics (OTEL)

### HTTP Metrics
- `http.server.request.count` - Total requests
  - Labels: method, route, status_code
- `http.server.request.duration` - Request duration (ms)
  - Labels: method, route, status_code

### Database Metrics
- `db.query.duration` - Query execution time (ms)
  - Labels: operation (select, insert, update, delete)

### Cache Metrics
- `cache.hit.count` - Cache hits
  - Labels: cache.key
- `cache.miss.count` - Cache misses
  - Labels: cache.key

## Next Steps (Phase 2)

1. **Complete Repository Adapters**
   - ProcedureAdapter
   - AppointmentAdapter
   - InsuranceAdapter
   - UserAdapter
   - ReviewAdapter

2. **Use Cases Layer**
   - SearchFacilitiesUseCase
   - BookAppointmentUseCase
   - GetAvailabilityUseCase

3. **Additional API Endpoints**
   - Procedures
   - Appointments
   - Availability
   - Reviews

4. **Integration Tests**
   - Test complete data flow
   - Use test containers for PostgreSQL/Redis

5. **External Provider Integration**
   - Real geolocation provider (Google Maps, Mapbox)
   - Provider adapter pattern

## Best Practices

### Testing
1. Write tests first (TDD)
2. Test one thing per test
3. Use table-driven tests for multiple scenarios
4. Mock external dependencies
5. Test error paths

### Code Organization
1. Keep packages focused and small
2. Use dependency injection
3. Depend on interfaces, not implementations
4. Keep business logic in domain layer
5. No circular dependencies

### Error Handling
1. Use custom error types
2. Wrap errors with context
3. Log errors at the boundary
4. Return domain errors from adapters

### Observability
1. Add spans for important operations
2. Record errors in spans
3. Emit metrics at boundaries
4. Use structured logging

## Performance Considerations

1. **Database**
   - Connection pooling configured
   - Indexes on frequently queried columns
   - Prepared statements where applicable
   - Spatial queries for location search

2. **Caching**
   - Redis for frequently accessed data
   - Cache invalidation strategy
   - TTL configuration

3. **Concurrency**
   - Context for cancellation
   - Timeouts on all external calls
   - Goroutines for async operations (future)

4. **Memory**
   - Limit result set sizes
   - Stream large datasets (future)
   - Pagination support

## Security Considerations

1. **Input Validation**
   - Validate all inputs at API boundary
   - Sanitize user input
   - Use parameterized queries (SQL injection prevention)

2. **Authentication** (Phase 2+)
   - JWT tokens
   - API key authentication

3. **Authorization** (Phase 2+)
   - Role-based access control
   - Resource-level permissions

4. **Data Protection**
   - No sensitive data in logs
   - Encrypted connections (TLS)
   - Secure configuration management
