# Quick Start Guide

Get the Patient Price Discovery backend up and running in minutes.

## Prerequisites

- Go 1.21 or higher
- Docker and Docker Compose
- Make (optional, but recommended)

## Option 1: Quick Start with Docker (Recommended)

This will start PostgreSQL, Redis, and the API server.

```bash
# 1. Navigate to backend directory
cd backend

# 2. Copy environment variables
cp .env.example .env

# 3. Start infrastructure services
docker-compose up -d

# 4. Wait for services to be ready (about 10 seconds)
sleep 10

# 5. Install dependencies
go mod download

# 6. Run the application
go run cmd/api/main.go
```

The API will be available at `http://localhost:8080`

### Verify Installation

```bash
# Health check
curl http://localhost:8080/health

# Expected: OK
```

## Option 2: Manual Setup

### 1. Install PostgreSQL

```bash
# macOS
brew install postgresql@15
brew services start postgresql@15

# Ubuntu/Debian
sudo apt-get install postgresql-15
sudo systemctl start postgresql

# Create database
createdb patient_price_discovery
```

### 2. Install Redis

```bash
# macOS
brew install redis
brew services start redis

# Ubuntu/Debian
sudo apt-get install redis-server
sudo systemctl start redis
```

### 3. Configure Environment

```bash
cd backend
cp .env.example .env

# Edit .env with your database credentials
nano .env
```

### 4. Run Migrations

```bash
# Apply database schema
psql -d patient_price_discovery -f migrations/001_initial_schema.sql
```

### 5. Start the Server

```bash
# Install dependencies
go mod download

# Run the application
go run cmd/api/main.go
```

## Using Make Commands

If you have Make installed:

```bash
# Install dependencies
make deps

# Run migrations
make migrate-up DB_NAME=patient_price_discovery

# Build the application
make build

# Run tests
make test

# Run the application
make run
```

## Testing the API

### Health Check

```bash
curl http://localhost:8080/health
```

### List Facilities

```bash
curl http://localhost:8080/api/facilities
```

### Get Facility by ID

```bash
curl http://localhost:8080/api/facilities/facility-1
```

### Search Facilities

```bash
curl "http://localhost:8080/api/facilities/search?lat=37.7749&lon=-122.4194&radius=10"
```

## Sample Data (Coming in Phase 2)

To populate the database with sample data:

```bash
# This will be available in Phase 2
go run scripts/seed.go
```

## Development Workflow

### 1. Generate Mocks

When you change interfaces, regenerate mocks:

```bash
# Install mockery
go install github.com/vektra/mockery/v2@latest

# Generate mocks
mockery
# or
make mocks
```

### 2. Run Tests

```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Generate coverage report
make test-coverage
# Opens coverage.html in browser
```

### 3. Format Code

```bash
# Format code
go fmt ./...

# Or use make
make fmt
```

### 4. Lint Code

```bash
# Install golangci-lint
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Run linter
golangci-lint run ./...

# Or use make
make lint
```

## IDE Setup

### VS Code

Install recommended extensions:
- Go (golang.go)
- Go Test Explorer (premparihar.gotestexplorer)

### GoLand

GoLand works out of the box. Just open the backend directory.

## Troubleshooting

### Database Connection Error

```
Error: failed to connect to database
```

**Solution:**
- Ensure PostgreSQL is running: `pg_isready`
- Check `.env` database credentials
- Verify database exists: `psql -l | grep patient_price_discovery`

### Redis Connection Error

```
Warning: Failed to initialize Redis client
```

**Solution:**
- Ensure Redis is running: `redis-cli ping`
- Check `.env` Redis configuration
- Note: The app can run without Redis (caching is optional)

### Port Already in Use

```
Error: bind: address already in use
```

**Solution:**
- Change `SERVER_PORT` in `.env`
- Or stop the process using port 8080:
  ```bash
  lsof -ti:8080 | xargs kill
  ```

### Module Dependencies Error

```
Error: missing go.sum entry
```

**Solution:**
```bash
go mod tidy
go mod download
```

## Docker Compose Services

The `docker-compose.yml` includes:

- **PostgreSQL** (port 5432)
  - Database: patient_price_discovery
  - User: postgres
  - Password: postgres

- **Redis** (port 6379)
  - No authentication by default

- **OTEL Collector** (optional, commented out)
  - Ports: 4317 (gRPC), 4318 (HTTP)

### Accessing Services

```bash
# PostgreSQL
psql -h localhost -U postgres -d patient_price_discovery

# Redis
redis-cli

# View logs
docker-compose logs -f

# Stop services
docker-compose down

# Stop and remove volumes
docker-compose down -v
```

## Next Steps

1. **Explore the code**: Start with `cmd/api/main.go`
2. **Read the architecture**: See `ARCHITECTURE.md`
3. **Review the plan**: See `PHASE1_PLAN.md`
4. **Add sample data**: Wait for Phase 2 or create your own
5. **Build features**: Follow TDD approach

## Common Development Tasks

### Add a New Entity

1. Create entity in `internal/domain/entities/`
2. Define repository interface in `internal/domain/repositories/`
3. Implement adapter in `internal/adapters/database/`
4. Create handler in `internal/api/handlers/`
5. Add routes in `internal/api/routes/`
6. Write tests
7. Update mockery config
8. Generate mocks

### Add a New Endpoint

1. Add handler method in appropriate handler
2. Register route in `internal/api/routes/router.go`
3. Write tests
4. Update documentation

### Enable OpenTelemetry

1. Start OTEL Collector (uncomment in docker-compose.yml)
2. Update `.env`:
   ```
   OTEL_ENABLED=true
   OTEL_ENDPOINT=localhost:4317
   ```
3. Restart the server
4. View traces in Jaeger (http://localhost:16686)

## Production Checklist

Before deploying to production:

- [ ] Set strong database password
- [ ] Enable HTTPS/TLS
- [ ] Configure proper CORS origins
- [ ] Enable OTEL for monitoring
- [ ] Set up proper logging
- [ ] Configure connection pool sizes
- [ ] Set appropriate timeouts
- [ ] Enable rate limiting (Phase 2)
- [ ] Set up database backups
- [ ] Configure health checks
- [ ] Set up CI/CD pipelines
- [ ] Review security settings

## Getting Help

- Check the [README](README.md) for detailed documentation
- Review [ARCHITECTURE.md](ARCHITECTURE.md) for system design
- Read [PHASE1_PLAN.md](PHASE1_PLAN.md) for implementation details
- Open an issue for bugs or questions

## Resources

- [Go Documentation](https://go.dev/doc/)
- [OpenTelemetry Go](https://opentelemetry.io/docs/instrumentation/go/)
- [PostgreSQL Documentation](https://www.postgresql.org/docs/)
- [Redis Documentation](https://redis.io/documentation)
