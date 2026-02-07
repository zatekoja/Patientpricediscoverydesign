# Architecture Overview

This backend includes two aligned subsystems:

1. **Core Backend (Go)** - Domain-driven API, repositories, adapters, infrastructure.
2. **External Data Provider System (TypeScript)** - Provider interface, document store, scheduler, REST API.

## Core Backend Architecture

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

## External Data Provider System

```
┌─────────────────────────────────────────────────────────────────────┐
│                     External Data Provider System                    │
└─────────────────────────────────────────────────────────────────────┘

┌──────────────────────┐
│  External Sources    │
│                      │
│  ┌────────────────┐  │
│  │ Google Sheets  │  │     ┌──────────────────────────────────┐
│  │  Spreadsheet 1 │  │────▶│    MegalekAteruHelper Provider   │
│  │  Spreadsheet 2 │  │     │                                  │
│  │  Spreadsheet 3 │  │     │  - Fetch data from sheets       │
│  └────────────────┘  │     │  - Transform to PriceData       │
│                      │     │  - Apply column mapping          │
└──────────────────────┘     │  - Validate data                 │
                             └──────────────────────────────────┘
                                            │
                                            ▼
                             ┌──────────────────────────────────┐
                             │   IExternalDataProvider          │
                             │   Interface                      │
                             │                                  │
                             │  + getCurrentData()              │
                             │  + getPreviousData()             │
                             │  + getHistoricalData()           │
                             │  + syncData()                    │
                             │  + getHealthStatus()             │
                             └──────────────────────────────────┘
                                            │
                    ┌───────────────────────┼───────────────────────┐
                    ▼                       ▼                       ▼
        ┌────────────────────┐  ┌────────────────────┐  ┌────────────────────┐
        │   Current Data     │  │   Previous Data    │  │  Historical Data   │
        │                    │  │                    │  │                    │
        │  Latest sync       │  │  Last sync before  │  │  Time range query  │
        │  Real-time view    │  │  current           │  │  30d, 6m, 1y, etc. │
        └────────────────────┘  └────────────────────┘  └────────────────────┘
                    │                       │                       │
                    └───────────────────────┼───────────────────────┘
                                            ▼
                             ┌──────────────────────────────────┐
                             │   IDocumentStore Interface       │
                             │                                  │
                             │  + put(key, data)                │
                             │  + get(key)                      │
                             │  + query(filter, options)        │
                             │  + batchPut(items)               │
                             └──────────────────────────────────┘
                                            │
                    ┌───────────────────────┼───────────────────────┐
                    ▼                       ▼                       ▼
        ┌────────────────────┐  ┌────────────────────┐  ┌────────────────────┐
        │   S3 Document      │  │  DynamoDB Store    │  │   MongoDB Store    │
        │   Store            │  │                    │  │                    │
        │                    │  │  - Fast queries    │  │  - Rich queries    │
        │  - File-based      │  │  - Scalable        │  │  - Flexible schema │
        │  - Cost effective  │  │  - Serverless      │  │  - Aggregations    │
        └────────────────────┘  └────────────────────┘  └────────────────────┘

┌─────────────────────────────────────────────────────────────────────┐
│                       Data Sync Scheduler                            │
│                                                                      │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │  Schedule: Every 3 Days (Configurable)                      │   │
│  │                                                              │   │
│  │  1. Trigger sync job                                        │   │
│  │  2. Provider fetches data from external source              │   │
│  │  3. Transform and validate data                             │   │
│  │  4. Store in document store                                 │   │
│  │  5. Update metadata (timestamp, record count)               │   │
│  │  6. Report success/failure                                  │   │
│  └─────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────┐
│                       Application Integration                        │
└─────────────────────────────────────────────────────────────────────┘

        ┌────────────────────┐         ┌────────────────────┐
        │   React Frontend   │         │   Backend API      │
        │                    │         │                    │
        │  - Display prices  │◀────────│  - Query provider  │
        │  - Search/filter   │         │  - Cache results   │
        │  - Compare prices  │         │  - Handle errors   │
        └────────────────────┘         └────────────────────┘
                                                 │
                                                 ▼
                             ┌──────────────────────────────────┐
                             │   Data Provider System           │
                             │                                  │
                             │  - MegalekAteruHelper            │
                             │  - Document Store                │
                             │  - Scheduler                     │
                             └──────────────────────────────────┘
```

## Data Flow

### Core Backend Read Operation (Example: Get Facility)
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

### Core Backend Write Operation (Example: Book Appointment - Phase 2)
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

### Core Backend Search Operation with Caching
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

### External Provider Setup
```
User → Configure Provider → Initialize with Credentials → Ready
```

### External Provider Scheduled Sync (Every 3 Days)
```
Scheduler Triggers
    ↓
Provider.syncData()
    ↓
Fetch from Google Sheets
    ↓
Transform to PriceData
    ↓
Validate Data
    ↓
Store in Document Store
    ↓
Update Metadata
    ↓
Report Success/Failure
```

### External Provider Query Current Data
```
Application Request
    ↓
Provider.getCurrentData()
    ↓
Check Document Store
    ↓
Return Latest Sync
    ↓
Display to User
```

### External Provider Query Historical Data
```
User Request (time window: "30d")
    ↓
Provider.getHistoricalData({ timeWindow: "30d" })
    ↓
Parse Time Window (30 days ago → today)
    ↓
Query Document Store (date range filter)
    ↓
Return Filtered Results
    ↓
Display Trends/Charts
```

## Layer Responsibilities (Core Backend)

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

## Provider System Components

```
IExternalDataProvider (Interface)
    ↑
    │ implements
    │
BaseDataProvider (Abstract Class)
    ↑
    │ extends
    │
MegalekAteruHelper (Concrete Implementation)
    │
    │ uses
    ↓
IDocumentStore (Interface)
    ↑
    │ implements
    │
InMemoryDocumentStore / S3Store / DynamoDBStore / MongoDBStore
```

## Configuration Flow (Provider System)

```
Environment Variables / Config File
    ↓
GoogleSheetsConfig
    │
    ├─ credentials (service account)
    ├─ spreadsheetIds (array)
    ├─ sheetNames (optional)
    ├─ columnMapping (field mapping)
    └─ syncSchedule (cron/interval)
    ↓
Provider.initialize(config)
    ↓
Ready to Use
```

## Error Handling (Provider System)

```
Provider Operation
    │
    ├─ Success → Return Data
    │
    └─ Error → Try
              │
              ├─ Network Error → Retry (with backoff)
              ├─ Auth Error → Log + Alert
              ├─ Validation Error → Skip record + Continue
              └─ Unknown Error → Log + Report
```

## Deployment Architecture (Provider System)

```
┌─────────────────────────────────────────────────────────────────┐
│                         AWS/Cloud                               │
│                                                                 │
│  ┌──────────────┐    ┌──────────────┐    ┌──────────────┐     │
│  │   Lambda     │    │   S3 Bucket  │    │  DynamoDB    │     │
│  │   Function   │───▶│              │    │   Table      │     │
│  │              │    │  Price Data  │    │              │     │
│  │  - Scheduler │    │  Storage     │    │  Metadata    │     │
│  └──────────────┘    └──────────────┘    └──────────────┘     │
│         │                                                       │
│         │ Calls                                                │
│         ▼                                                       │
│  ┌──────────────┐                                              │
│  │  EventBridge │                                              │
│  │  Rule        │                                              │
│  │  (3 days)    │                                              │
│  └──────────────┘                                              │
└─────────────────────────────────────────────────────────────────┘
                        │
                        │ Reads from
                        ▼
            ┌────────────────────┐
            │  Google Sheets     │
            │  API               │
            │                    │
            │  - Spreadsheet 1   │
            │  - Spreadsheet 2   │
            │  - Spreadsheet N   │
            └────────────────────┘
```

## Observability (Core Backend)

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

### Provider System Security
- Google service account authentication (IAM)
- Credentials stored in secrets manager or env vars
- Least-privilege IAM roles for S3/DynamoDB
- Config validation before initialization
- Data validation before storage

## Performance Optimizations (Core Backend)

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

## Scalability Considerations (Core Backend)

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

## Deployment Architecture (Core Backend - Future)

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
