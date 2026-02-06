# Patient Price Discovery Backend

A performant Go backend for the Patient Price Discovery healthcare transparency application, built following Domain-Driven Design (DDD), Test-Driven Development (TDD), and OpenTelemetry (OTEL) standards.

## Architecture

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

## Project Structure

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

## Domain Entities

### Core Entities
- **Facility**: Healthcare facilities with location, pricing, insurance acceptance
- **Procedure**: Medical procedures/services with CPT codes
- **Appointment**: Scheduled appointments
- **Insurance**: Insurance providers
- **User**: System users
- **Review**: User reviews for facilities

### Relationships
- Facilities offer multiple Procedures with specific pricing
- Facilities accept multiple Insurance Providers
- Users can book Appointments at Facilities for Procedures
- Users can leave Reviews for Facilities

## Tech Stack

- **Language**: Go 1.22+
- **Database**: PostgreSQL (with spatial queries support)
- **Search Engine**: Typesense
- **Cache**: Redis
- **Observability**: OpenTelemetry (OTEL)
- **Testing**: testify, mockery
- **Geolocation**: Mock provider (can be replaced with Google Maps, Mapbox, etc.)
- **Query Builder**: goqu for type-safe SQL generation

## Getting Started

### Prerequisites

- Go 1.22 or higher (required for ServeMux pattern matching)
- PostgreSQL 13+
- Redis 6+
- Docker (optional, for local development)

### Environment Variables

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

# OpenTelemetry Configuration
OTEL_ENABLED=false
OTEL_SERVICE_NAME=patient-price-discovery
OTEL_SERVICE_VERSION=1.0.0
OTEL_ENDPOINT=
```

### Installation

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

### Running Tests

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run tests with race detection
go test -race ./...
```

## API Endpoints

### Health Check
- `GET /health` - Health check endpoint

### Facilities
- `GET /api/facilities` - List all facilities
- `GET /api/facilities/:id` - Get facility by ID
- `GET /api/facilities/search` - Search facilities by location

### Future Endpoints (Phase 2+)
- `GET /api/procedures` - List procedures
- `GET /api/facilities/:id/availability` - Get facility availability
- `POST /api/appointments` - Book appointment
- `GET /api/facilities/:id/reviews` - Get facility reviews

## OTEL Metrics

The application emits the following key metrics:

- `http.server.request.count` - Number of HTTP requests
- `http.server.request.duration` - HTTP request duration
- `db.query.duration` - Database query duration
- `cache.hit.count` - Cache hit count
- `cache.miss.count` - Cache miss count

## Development Phases

### Phase 1: Foundation (Current)
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

### Phase 2: Core Features
- [ ] Complete all repository adapters
- [ ] Implement use cases
- [ ] Add validation logic
- [ ] Implement remaining API endpoints
- [ ] Integration tests
- [ ] Real geolocation provider integration

### Phase 3: Advanced Features
- [ ] Authentication & authorization
- [ ] Rate limiting
- [ ] Advanced search with filters
- [ ] Appointment booking workflow
- [ ] Email notifications
- [ ] Data seeding

## Testing Strategy

This project follows Test-Driven Development (TDD):

1. **Unit Tests**: Test individual components in isolation using mocks
2. **Integration Tests**: Test interactions between components
3. **End-to-End Tests**: Test complete user workflows

Use mockery to generate mocks from interfaces:
```bash
mockery
```

## Contributing

1. Follow the existing architecture patterns
2. Write tests before implementation (TDD)
3. Keep functions small and focused
4. Use dependency injection
5. Document complex logic
6. Follow Go conventions and best practices

## License

MIT License
