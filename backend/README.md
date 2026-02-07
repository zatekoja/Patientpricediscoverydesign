# Patient Price Discovery Backend

This backend includes two aligned subsystems:

1. **Core Backend (Go)** - Domain-driven API, repositories, adapters, infrastructure.
2. **External Data Provider System (TypeScript)** - Provider interface, document store, scheduler, REST API.

## Core Backend (Go)

### Architecture

This backend follows a clean architecture with strict separation of responsibilities:

```
API Layer → Application Layer → Domain Layer → Infrastructure Layer
     ↓           ↓                   ↓              ↓
 Handlers    Services           Entities        Clients
 Routes      Use Cases          Repositories    Adapters
 Middleware                     Providers
```

### Data Flow

```
HTTP Request → API Handler → Application Service → Adapter → Client → External System
```

### Project Structure

```
backend/
├── cmd/
│   └── api/                    # Main application entry point
├── internal/
│   ├── domain/                 # Domain layer (core business logic)
│   │   ├── entities/           # Domain entities
│   │   ├── repositories/       # Repository interfaces (ports)
│   │   └── providers/          # External service interfaces
│   ├── application/            # Application layer
│   │   ├── services/           # Business services (e.g., FacilityService)
│   │   └── usecases/           # Use case implementations
│   ├── adapters/               # Adapters layer (data access)
│   │   ├── database/           # Database adapters (PostgreSQL)
│   │   ├── search/             # Search adapters (Typesense)
│   │   ├── cache/              # Cache adapters (Redis)
│   │   └── providers/          # External provider implementations
│   ├── infrastructure/         # Infrastructure layer
│   │   ├── clients/            # Database and external service clients
│   │   │   ├── postgres/       # PostgreSQL client
│   │   │   └── redis/          # Redis client
│   │   └── observability/      # OTEL setup and metrics
│   └── api/                    # API layer
│       ├── handlers/           # HTTP handlers
│       ├── middleware/         # HTTP middleware
│       └── routes/             # Route configuration
├── pkg/
│   ├── config/                 # Configuration management
│   └── errors/                 # Custom error types
├── tests/
│   └── mocks/                  # Generated mocks (mockery)
├── migrations/                 # Database migrations
├── .mockery.yml               # Mockery configuration
└── go.mod
```

### Domain Entities

#### Core Entities
- **Facility**: Healthcare facilities with location, pricing, insurance acceptance
- **Procedure**: Medical procedures/services with CPT codes
- **Appointment**: Scheduled appointments
- **Insurance**: Insurance providers
- **User**: System users
- **Review**: User reviews for facilities

#### Relationships
- Facilities offer multiple Procedures with specific pricing
- Facilities accept multiple Insurance Providers
- Users can book Appointments at Facilities for Procedures
- Users can leave Reviews for Facilities

### Tech Stack

- **Language**: Go 1.22+
- **Database**: PostgreSQL (with spatial queries support)
- **Search Engine**: Typesense
- **Cache**: Redis
- **Observability**: OpenTelemetry (OTEL)
- **Testing**: testify, mockery
- **Geolocation**: Mock provider (can be replaced with Google Maps, Mapbox, etc.)
- **Query Builder**: goqu for type-safe SQL generation

### Getting Started

#### Prerequisites

- Go 1.22 or higher (required for ServeMux pattern matching)
- PostgreSQL 13+
- Redis 6+
- Docker (optional, for local development)

#### Environment Variables

Create a `.env` file or set the following environment variables:

```bash
# Server Configuration
SERVER_HOST=0.0.0.0
SERVER_PORT=8080

# Database Configuration
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=your_password
DB_NAME=patient_price_discovery
DB_SSLMODE=disable

# Redis Configuration
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=
REDIS_DB=0

# Geolocation Provider
GEOLOCATION_PROVIDER=mock
GEOLOCATION_API_KEY=

# Provider API (for ingesting price list facilities)
PROVIDER_API_BASE_URL=http://localhost:3002/api/v1
PROVIDER_INGEST_ON_START=false
PROVIDER_INGEST_PROVIDER_ID=file_price_list
PROVIDER_INGEST_PAGE_SIZE=500

# OpenTelemetry Configuration
OTEL_ENABLED=false
OTEL_SERVICE_NAME=patient-price-discovery
OTEL_SERVICE_VERSION=1.0.0
OTEL_ENDPOINT=
```

#### Installation

1. Install dependencies:
```bash
cd backend
go mod download
```

2. Set up the database:
```bash
# Create database
createdb patient_price_discovery

# Run migrations
psql -d patient_price_discovery -f migrations/001_initial_schema.sql
```

3. Generate mocks for testing:
```bash
go install github.com/vektra/mockery/v2@latest
mockery
```

4. Run the application:
```bash
go run cmd/api/main.go
```

#### Running Tests

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run tests with race detection
go test -race ./...
```

### API Endpoints

#### Health Check
- `GET /health` - Health check endpoint

#### Facilities
- `GET /api/facilities` - List all facilities
- `GET /api/facilities/:id` - Get facility by ID
- `GET /api/facilities/search` - Search facilities by location

#### Provider Data (REST)
- `GET /api/provider/prices/current` - Current provider price data
- `GET /api/provider/prices/previous` - Previous provider price batch
- `GET /api/provider/prices/historical` - Historical provider price data
- `GET /api/provider/health` - Provider health status
- `GET /api/provider/list` - List registered providers
- `POST /api/provider/sync/trigger` - Trigger provider sync
- `GET /api/provider/sync/status` - Provider sync status
- `POST /api/provider/ingest` - Ingest provider data into core backend tables

#### Future Endpoints (Phase 2+)
- `GET /api/procedures` - List procedures
- `GET /api/facilities/:id/availability` - Get facility availability
- `POST /api/appointments` - Book appointment
- `GET /api/facilities/:id/reviews` - Get facility reviews

### GraphQL + Provider API

The GraphQL server (`cmd/graphql`) exposes external provider data via:

- `providerPriceCurrent(providerId, limit, offset)` → `ProviderPriceResponse`

It calls the Provider API configured by `PROVIDER_API_BASE_URL`. Example:

```graphql
query {
  providerPriceCurrent(limit: 5) {
    data {
      id
      facilityName
      procedureDescription
      price
      tags
    }
    timestamp
  }
}
```

Run locally:

```bash
PROVIDER_API_BASE_URL=http://localhost:3002/api/v1 \
go run cmd/graphql/main.go
```

### OTEL Metrics

The application emits the following key metrics:

- `http.server.request.count` - Number of HTTP requests
- `http.server.request.duration` - HTTP request duration
- `db.query.duration` - Database query duration
- `cache.hit.count` - Cache hit count
- `cache.miss.count` - Cache miss count

### Development Phases

#### Phase 1: Foundation (Current)
- ✅ Domain entities and schemas
- ✅ Repository interfaces (ports)
- ✅ Database clients
- ✅ Basic adapters
- ✅ OTEL setup
- ✅ Configuration management
- ✅ API structure with handlers
- ✅ Middleware (logging, CORS, observability)
- ✅ Mock providers
- ✅ Test structure

#### Phase 2: Core Features
- [ ] Complete all repository adapters
- [ ] Implement use cases
- [ ] Add validation logic
- [ ] Implement remaining API endpoints
- [ ] Integration tests
- [ ] Real geolocation provider integration

#### Phase 3: Advanced Features
- [ ] Authentication & authorization
- [ ] Rate limiting
- [ ] Advanced search with filters
- [ ] Appointment booking workflow
- [ ] Email notifications
- [ ] Data seeding

### Testing Strategy

This project follows Test-Driven Development (TDD):

1. **Unit Tests**: Test individual components in isolation using mocks
2. **Integration Tests**: Test interactions between components
3. **End-to-End Tests**: Test complete user workflows

Use mockery to generate mocks from interfaces:
```bash
mockery
```

### Contributing

1. Follow the existing architecture patterns
2. Write tests before implementation (TDD)
3. Keep functions small and focused
4. Use dependency injection
5. Document complex logic
6. Follow Go conventions and best practices

### License

MIT License

## External Data Provider System (TypeScript)

This system provides a flexible interface for connecting external data providers to the Patient Price Discovery application. The primary implementation is the `megalek_ateru_helper` provider, which connects to Google Sheets to retrieve price data.

### Quick Links

- **[REST API Documentation](./api/README.md)** - HTTP API for accessing price data
- **[OpenAPI Specification](./api/openapi.yaml)** - Complete API specification
- **[Quick Reference](./QUICK_REFERENCE.md)** - Common use cases and examples
- **[Architecture Diagrams](./ARCHITECTURE.md)** - System architecture

### Architecture Overview

#### Core Components

1. **IExternalDataProvider Interface** - Contract that all external providers must implement
2. **BaseDataProvider** - Abstract base class providing common functionality
3. **MegalekAteruHelper** - Google Sheets implementation of the data provider
4. **IDocumentStore Interface** - Abstraction for data storage (S3, DynamoDB, MongoDB)
5. **DataSyncScheduler** - Scheduler for automatic data synchronization
6. **REST API** - HTTP endpoints for external services to access data

#### Directory Structure

```
backend/
├── api/
│   ├── openapi.yaml                 # OpenAPI 3.0 specification
│   ├── server.ts                    # Express REST API server
│   ├── example-server.ts            # Example server setup
│   └── README.md                    # API documentation
├── interfaces/
│   ├── IExternalDataProvider.ts    # Main provider interface
│   └── IDocumentStore.ts            # Document store interface
├── providers/
│   ├── BaseDataProvider.ts          # Base implementation
│   └── MegalekAteruHelper.ts        # Google Sheets provider
├── stores/
│   └── InMemoryDocumentStore.ts     # Example in-memory store
├── config/
│   └── DataSyncScheduler.ts         # Job scheduler
├── types/
│   └── PriceData.ts                 # Type definitions
└── example-usage.ts                 # Usage examples
```

### Features

#### Data Provider Interface

All external providers support:

- **Current Data** - Get the most recent data
- **Previous Data** - Get the last batch before current
- **Historical Data** - Query data within a time range
- **Configurable Options** - Time windows, pagination, custom parameters
- **Automatic Sync** - Scheduled data synchronization
- **Health Monitoring** - Check provider status

#### REST API

The system includes a production-ready REST API:

- **HTTP Endpoints** - RESTful API for data access
- **OpenAPI 3.0 Spec** - Complete API documentation
- **Error Handling** - Standardized error responses
- **CORS Support** - Cross-origin resource sharing
- **Pagination** - Support for large datasets

See `backend/api/README.md` for details.

#### Configurable Options

The `DataProviderOptions` interface supports:

```typescript
{
  timeWindow: "30d" | "7d" | "1y",  // e.g., "30d" = last 30 days
  startDate: Date,                   // Explicit date range
  endDate: Date,
  parameters: { /* custom */ },      // Provider-specific params
  limit: 100,                        // Pagination
  offset: 0
}
```

### Google Sheets Provider (megalek_ateru_helper)

#### Configuration

```typescript
const config: GoogleSheetsConfig = {
  credentials: {
    clientEmail: 'service-account@project.iam.gserviceaccount.com',
    privateKey: '-----BEGIN PRIVATE KEY-----
...
-----END PRIVATE KEY-----',
    projectId: 'my-project-id',
  },
  spreadsheetIds: [
    '1BxiMVs0XRA5nFMdKvBdBZjgmUUqptlbs74OgvE2upms',
  ],
  sheetNames: ['Price Data'],
  columnMapping: {
    facilityName: 'Facility Name',
    procedureCode: 'CPT Code',
    price: 'Cash Price',
    effectiveDate: 'Effective Date',
  },
};
```

#### Usage

```typescript
import { MegalekAteruHelper } from './providers/MegalekAteruHelper';
import { InMemoryDocumentStore } from './stores/InMemoryDocumentStore';

// 1. Create document store
const store = new InMemoryDocumentStore<PriceData>('price-data');

// 2. Create provider
const provider = new MegalekAteruHelper(store);

// 3. Initialize with config
await provider.initialize(config);

// 4. Fetch data
const currentData = await provider.getCurrentData({ limit: 100 });
const historicalData = await provider.getHistoricalData({
  timeWindow: '30d'
});
```

#### Scheduled Sync (Every 3 Days)

```typescript
import { DataSyncScheduler, SyncIntervals } from './config/DataSyncScheduler';

const scheduler = new DataSyncScheduler();

scheduler.scheduleJob({
  name: 'megalek_sync',
  provider: provider,
  intervalMs: SyncIntervals.THREE_DAYS,
  runImmediately: true,
  onComplete: (result) => {
    console.log('Sync completed:', result);
  },
});
```

### File Price List Provider (file_price_list)

For CSV/DOCX price lists (local or downloaded from Drive), you can use the file provider.

```bash
PRICE_LIST_FILES=/app/fixtures/price_lists/MEGALEK NEW PRICE LIST 2026.csv,/app/fixtures/price_lists/PRICE_LIST_FOR_OFFICE_USE[1].docx
PRICE_LIST_CURRENCY=NGN
PRICE_LIST_EFFECTIVE_DATE=2026-01-01
```

The provider normalizes rows into `PriceData` and stores:
- `metadata.area` and `metadata.category` (when detected)
- `metadata.priceTier` (adult/paediatric/etc.)
- `metadata.unit` (per_day, per_hour, etc.)

### Data Flow

1. **External Source (Google Sheets)**
   - Provider queries Google Sheets API
   - Maps spreadsheet rows to `PriceData` objects

2. **Data Transformation**
   - Validates and normalizes data
   - Applies column mappings from config

3. **Storage (Document Store)**
   - Stores data in S3, DynamoDB, MongoDB, or other store
   - Maintains metadata (timestamps, source info)

4. **Query Interface**
   - Applications query through the provider interface
   - Supports current, previous, and historical queries
   - Configurable time windows and filters

### Document Store Implementations

The system supports any document store that implements `IDocumentStore`:

#### In-Memory Store (Development)
```typescript
const store = new InMemoryDocumentStore<PriceData>('my-store');
```

#### Production Stores (To Implement)

**S3 Document Store**
- Store data as JSON files in S3 buckets
- Use prefixes for organization

**DynamoDB Document Store**
- Store as items in DynamoDB table
- Use GSI for efficient queries

**MongoDB Document Store**
- Store as documents in MongoDB collection
- Use indexes for query performance

### Adding a New Provider

To create a new external data provider:

1. **Extend BaseDataProvider**
```typescript
export class MyCustomProvider extends BaseDataProvider<MyDataType> {
  constructor(store?: IDocumentStore<MyDataType>) {
    super('my_custom_provider', store);
  }
}
```

2. **Implement Required Methods**
```typescript
validateConfig(config: Record<string, any>): boolean { /* ... */ }
getCurrentData(options?: DataProviderOptions): Promise<DataProviderResponse<MyDataType>> { /* ... */ }
getPreviousData(options?: DataProviderOptions): Promise<DataProviderResponse<MyDataType>> { /* ... */ }
getHistoricalData(options: DataProviderOptions): Promise<DataProviderResponse<MyDataType>> { /* ... */ }
```

3. **Use the Provider**
```typescript
const provider = new MyCustomProvider(documentStore);
await provider.initialize(config);
const data = await provider.getCurrentData();
```

### Time Window Format

Time windows use the format: `<number><unit>`

- `d` - Days (e.g., `"30d"` = 30 days)
- `m` - Months (e.g., `"6m"` = 6 months)
- `y` - Years (e.g., `"1y"` = 1 year)

Examples:
- `"7d"` - Last 7 days
- `"3m"` - Last 3 months
- `"1y"` - Last year

### Error Handling

All provider methods handle errors gracefully:

```typescript
try {
  const data = await provider.getCurrentData();
} catch (error) {
  console.error('Failed to fetch data:', error);
}

// Check health status
const health = await provider.getHealthStatus();
if (!health.healthy) {
  console.error('Provider unhealthy:', health.message);
}
```

### Observability (OTEL + SigNoz)

The provider system emits OpenTelemetry traces and metrics when `OTEL_ENABLED=true`.
It exports via OTLP/HTTP so SigNoz can ingest both traces and metrics.

```bash
# Enable OTEL
OTEL_ENABLED=true

# Service identity
OTEL_SERVICE_NAME=patient-price-discovery-provider
OTEL_SERVICE_VERSION=1.0.0

# OTLP endpoint for SigNoz (HTTP)
OTEL_ENDPOINT=http://localhost:4318
```

Custom metrics include:
- `provider.sync.count`, `provider.sync.duration_ms`, `provider.records.processed`
- `scheduler.job.run`, `scheduler.job.duration_ms`, `scheduler.job.skipped`
- `provider.tag_generation.count`, `provider.tag_generation.duration_ms`, `provider.tags.generated`
- `api.request.count`, `api.request.duration_ms`

### Production Deployment

#### Google Sheets API Setup

1. Create a Google Cloud Project
2. Enable Google Sheets API
3. Create a Service Account
4. Download credentials JSON
5. Share spreadsheets with service account email

#### Environment Variables

```bash
GOOGLE_CLIENT_EMAIL=service-account@project.iam.gserviceaccount.com
GOOGLE_PRIVATE_KEY=-----BEGIN PRIVATE KEY-----
...
-----END PRIVATE KEY-----
GOOGLE_PROJECT_ID=my-project-id
SPREADSHEET_IDS=id1,id2,id3
SYNC_INTERVAL_MS=259200000  # 3 days
```

#### Dependencies

```json
{
  "dependencies": {
    "googleapis": "^118.0.0",  // For Google Sheets API
    "aws-sdk": "^2.1400.0",     // For S3/DynamoDB (if used)
    "mongodb": "^5.6.0"         // For MongoDB (if used)
  }
}
```

### Next Steps

1. **Implement Production Document Store** - Choose S3, DynamoDB, or MongoDB
2. **Add Google Sheets API Integration** - Replace placeholder with actual API calls
3. **Set Up Authentication** - Configure service account credentials
4. **Deploy Scheduler** - Run as a background service or AWS Lambda
5. **Add Monitoring** - Track sync success/failure rates
6. **Add Data Validation** - Ensure data quality from spreadsheets
