# Architecture Overview

## System Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                          Client Applications                         │
│                     (React Frontend, Mobile Apps)                    │
└────────────────────────────┬────────────────────────────────────────┘
                             │ HTTP/REST
                             ▼
┌─────────────────────────────────────────────────────────────────────┐
│                          API Layer                                   │
│  ┌──────────────────────────────────────────────────────────────┐  │
│  │  Middleware: CORS, Logging, Observability (OTEL)            │  │
│  └──────────────────────────────────────────────────────────────┘  │
│  ┌──────────────────────────────────────────────────────────────┐  │
│  │  HTTP Handlers                                                │  │
│  │  - FacilityHandler                                            │  │
│  │  - ProcedureHandler (Phase 2)                                 │  │
│  │  - AppointmentHandler (Phase 2)                               │  │
│  └──────────────────────────────────────────────────────────────┘  │
└────────────────────────────┬────────────────────────────────────────┘
                             │ Domain Types
                             ▼
┌─────────────────────────────────────────────────────────────────────┐
│                      Application Layer (Phase 2)                     │
│  ┌──────────────────────────────────────────────────────────────┐  │
│  │  Use Cases / Services                                         │  │
│  │  - SearchFacilitiesUseCase                                    │  │
│  │  - BookAppointmentUseCase                                     │  │
│  │  - GetAvailabilityUseCase                                     │  │
│  └──────────────────────────────────────────────────────────────┘  │
└────────────────────────────┬────────────────────────────────────────┘
                             │ Repository Interfaces
                             ▼
┌─────────────────────────────────────────────────────────────────────┐
│                         Domain Layer                                 │
│  ┌──────────────────────────────────────────────────────────────┐  │
│  │  Entities                                                     │  │
│  │  - Facility, Procedure, Appointment, Insurance, User          │  │
│  └──────────────────────────────────────────────────────────────┘  │
│  ┌──────────────────────────────────────────────────────────────┐  │
│  │  Repository Interfaces (Ports)                                │  │
│  │  - FacilityRepository                                         │  │
│  │  - ProcedureRepository                                        │  │
│  │  - AppointmentRepository                                      │  │
│  └──────────────────────────────────────────────────────────────┘  │
│  ┌──────────────────────────────────────────────────────────────┐  │
│  │  Provider Interfaces                                          │  │
│  │  - GeolocationProvider                                        │  │
│  │  - CacheProvider                                              │  │
│  └──────────────────────────────────────────────────────────────┘  │
└────────────────┬───────────────────────────┬────────────────────────┘
                 │                           │
                 ▼                           ▼
┌────────────────────────────────┐  ┌──────────────────────────────┐
│      Adapters Layer            │  │   External Providers         │
│  ┌─────────────────────────┐   │  │  ┌──────────────────────┐   │
│  │ Database Adapters       │   │  │  │ Geolocation Provider │   │
│  │ - FacilityAdapter       │   │  │  │ - MockProvider       │   │
│  │ - ProcedureAdapter      │   │  │  │ - GoogleMapsProvider │   │
│  │ - AppointmentAdapter    │   │  │  │ - MapboxProvider     │   │
│  └─────────────────────────┘   │  │  └──────────────────────┘   │
│  ┌─────────────────────────┐   │  │  ┌──────────────────────┐   │
│  │ Cache Adapters          │   │  │  │ Data Providers       │   │
│  │ - RedisAdapter          │   │  │  │ (Future: CMS data)   │   │
│  └─────────────────────────┘   │  │  └──────────────────────┘   │
└───────────┬────────────────────┘  └──────────────┬───────────────┘
            │                                      │
            ▼                                      ▼
┌─────────────────────────────────────────────────────────────────────┐
│                     Infrastructure Layer                             │
│  ┌──────────────────────┐  ┌──────────────────┐  ┌──────────────┐ │
│  │  Database Clients    │  │  Cache Clients   │  │  Observability│ │
│  │  - PostgreSQL Client │  │  - Redis Client  │  │  - OTEL Setup │ │
│  │    (Connection Pool) │  │                  │  │  - Metrics    │ │
│  └──────────────────────┘  └──────────────────┘  │  - Tracing    │ │
│                                                   └──────────────┘ │
└────────────────────────────┬────────────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────────────┐
│                    External Systems                                  │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────────────────┐ │
│  │  PostgreSQL  │  │    Redis     │  │  OTEL Collector          │ │
│  │   Database   │  │    Cache     │  │  (Prometheus, Jaeger)    │ │
│  └──────────────┘  └──────────────┘  └──────────────────────────┘ │
└─────────────────────────────────────────────────────────────────────┘
```

## Data Flow

### Read Operation (e.g., Get Facility)
```
1. HTTP GET /api/facilities/{id}
   ↓
2. FacilityHandler.GetFacility()
   ↓
3. [Middleware: OTEL Tracing] Start span
   ↓
4. facilityRepo.GetByID(ctx, id) [Repository Interface]
   ↓
5. FacilityAdapter.GetByID() [Adapter Implementation]
   ↓
6. PostgreSQL Client.Query()
   ↓
7. [OTEL Metrics] Record DB query duration
   ↓
8. Return Facility Entity
   ↓
9. [Middleware: OTEL] Record request metrics
   ↓
10. HTTP Response (JSON)
```

### Write Operation (e.g., Book Appointment - Phase 2)
```
1. HTTP POST /api/appointments
   ↓
2. AppointmentHandler.CreateAppointment()
   ↓
3. [Validation] Validate request payload
   ↓
4. BookAppointmentUseCase.Execute()
   ↓
5. Check availability: availabilityRepo.GetByFacility()
   ↓
6. Create appointment: appointmentRepo.Create()
   ↓
7. Update slot: availabilityRepo.Update()
   ↓
8. [Optional] Send notification
   ↓
9. Return created appointment
```

### Search Operation with Caching
```
1. HTTP GET /api/facilities/search?lat=37.7&lon=-122.4
   ↓
2. FacilityHandler.SearchFacilities()
   ↓
3. Generate cache key: "search:37.7:-122.4:10km"
   ↓
4. Check cache: cacheProvider.Get(key)
   ├─ Cache Hit: Return cached results
   │  └─ [OTEL] Record cache hit metric
   └─ Cache Miss: 
      ├─ [OTEL] Record cache miss metric
      ├─ facilityRepo.Search(params)
      ├─ cacheProvider.Set(key, results, ttl)
      └─ Return results
```

## Layer Responsibilities

### API Layer
- **Purpose**: HTTP interface
- **Responsibilities**:
  - Parse HTTP requests
  - Validate input
  - Call domain services/repositories
  - Format HTTP responses
  - Apply middleware
- **Dependencies**: Handlers → Repository Interfaces

### Application Layer (Phase 2)
- **Purpose**: Business workflows
- **Responsibilities**:
  - Orchestrate multiple domain operations
  - Implement complex business logic
  - Transaction management
- **Dependencies**: Use Cases → Repository Interfaces

### Domain Layer
- **Purpose**: Core business logic
- **Responsibilities**:
  - Define entities and value objects
  - Define repository interfaces (contracts)
  - Define provider interfaces
  - Business rules and validations
- **Dependencies**: None (pure Go)

### Adapters Layer
- **Purpose**: Connect domain to infrastructure
- **Responsibilities**:
  - Implement repository interfaces
  - Transform domain entities to/from database models
  - Handle database-specific logic
- **Dependencies**: Adapters → Database Clients, Domain Entities

### Infrastructure Layer
- **Purpose**: External system connections
- **Responsibilities**:
  - Manage database connections
  - Manage cache connections
  - OTEL setup and metrics
  - Configuration management
- **Dependencies**: External libraries only

## Key Design Patterns

### 1. Repository Pattern
- Abstracts data access
- Domain layer defines interfaces
- Adapters implement interfaces
- Allows easy swapping of data sources

### 2. Adapter Pattern
- Converts between different interfaces
- FacilityAdapter implements FacilityRepository
- RedisAdapter implements CacheProvider

### 3. Dependency Injection
- Dependencies passed via constructors
- Makes testing easier
- Reduces coupling

### 4. Middleware Pattern
- Cross-cutting concerns (logging, tracing)
- Applied to all HTTP requests
- Clean separation from business logic

## Testing Strategy

### Unit Tests
```
Test: FacilityAdapter.Create()
Mock: PostgreSQL Client
Verify: Correct SQL executed, entity saved
```

### Integration Tests
```
Test: Complete search flow
Setup: Test database with sample data
Execute: Search API request
Verify: Correct results returned
```

### End-to-End Tests
```
Test: Book appointment workflow
Setup: Full system with test containers
Execute: Search → Select → Book → Verify
Verify: Appointment created, email sent
```

## Observability

### Metrics Emitted
- `http.server.request.count` - Request volume by endpoint
- `http.server.request.duration` - Request latency
- `db.query.duration` - Database performance
- `cache.hit.count` / `cache.miss.count` - Cache efficiency

### Traces
- Each HTTP request creates a span
- Nested spans for DB queries, cache operations
- Error recording in spans
- Distributed tracing support

### Logging
- Structured logging
- Request ID correlation
- Error context

## Security Considerations

### Current (Phase 1)
- SQL injection prevention (parameterized queries)
- CORS configuration
- Input validation at API boundary

### Future (Phase 2+)
- JWT authentication
- Role-based authorization
- Rate limiting
- API key management
- TLS/HTTPS
- Secrets management

## Performance Optimizations

### Database
- Connection pooling
- Indexes on frequently queried columns
- Spatial queries for location search

### Caching
- Redis for frequently accessed data
- TTL configuration
- Cache invalidation strategy

### Concurrency
- Context for cancellation
- Timeouts on all operations
- Goroutines for async work (future)

## Scalability Considerations

### Horizontal Scaling
- Stateless API servers
- Load balancer friendly
- Session in cache/database

### Database Scaling
- Read replicas for queries
- Write to primary
- Connection pooling

### Caching
- Redis cluster for high availability
- Cache warming strategies

## Deployment Architecture (Future)

```
┌─────────────────────────────────────────────────────────────┐
│                      Load Balancer                          │
└──────────┬──────────────────────────────┬───────────────────┘
           │                              │
           ▼                              ▼
    ┌────────────┐                 ┌────────────┐
    │ API Server │                 │ API Server │
    │ Instance 1 │                 │ Instance 2 │
    └─────┬──────┘                 └─────┬──────┘
          │                              │
          └──────────────┬───────────────┘
                         │
         ┌───────────────┼───────────────┐
         ▼               ▼               ▼
    ┌─────────┐    ┌─────────┐    ┌─────────┐
    │PostgreSQL│    │  Redis  │    │  OTEL   │
    │ Primary  │    │ Cluster │    │Collector│
    └─────────┘    └─────────┘    └─────────┘
         │
         ▼
    ┌─────────┐
    │PostgreSQL│
    │ Replica  │
    └─────────┘
```
