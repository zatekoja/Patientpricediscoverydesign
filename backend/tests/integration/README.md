# Integration Tests

This directory contains integration tests that connect to live database instances.

## Overview

Integration tests verify the correct interaction between adapters and real databases. Unlike unit tests that use mocks, integration tests:

- Connect to actual PostgreSQL and Redis instances
- Execute real SQL queries
- Verify data persistence and retrieval
- Test transaction behavior
- Validate NULL handling and edge cases

## Running Integration Tests

### Quick Start

```bash
# Start test database, run tests, and cleanup
make test-integration-full
```

### Manual Steps

```bash
# 1. Start test database
make test-db-up

# 2. Run integration tests
make test-integration

# 3. Stop test database
make test-db-down
```

### Provider API Integration Test (Docker)

This test hits the **external provider API** running in Docker and validates all interface-backed endpoints.

```bash
# Start test services (includes provider API + OTEL collector)
docker-compose -f docker-compose.test.yml up -d

# Run provider API integration test
PROVIDER_API_BASE_URL=http://localhost:3002/api/v1 \
node tests/integration/provider_api_integration_test.mjs

# Cleanup
docker-compose -f docker-compose.test.yml down -v
```

### Provider GraphQL Integration Test (Go)

This test runs the **GraphQL resolver** in-process and verifies it can call the live provider API.

```bash
# Ensure provider API is running
docker-compose -f docker-compose.test.yml up -d provider-api-test

# Run GraphQL integration test
PROVIDER_API_BASE_URL=http://localhost:3002/api/v1 \
PROVIDER_ID=file_price_list \
go test -v -tags=integration ./tests/integration -run TestProviderPriceCurrentGraphQLIntegration

# Cleanup
docker-compose -f docker-compose.test.yml down -v
```

### Provider API Client Integration Test (Go)

This test uses the Go provider API client against a live provider container.

```bash
docker-compose -f docker-compose.test.yml up -d provider-api-test

PROVIDER_API_BASE_URL=http://localhost:3002/api/v1 \
PROVIDER_ID=file_price_list \
go test -v -tags=integration ./tests/integration -run TestProviderAPIClientCurrentData

docker-compose -f docker-compose.test.yml down -v
```

### Individual Test Execution

```bash
# Run specific test suite
TEST_DB_HOST=localhost TEST_DB_PORT=5433 \
TEST_DB_USER=postgres TEST_DB_PASSWORD=postgres \
TEST_DB_NAME=patient_price_discovery_test \
TEST_DB_SSLMODE=disable \
go test -v -tags=integration ./tests/integration -run TestFacilityAdapter
```

## Test Database

Integration tests use a separate database (`patient_price_discovery_test`) to avoid interfering with development data.

### Configuration

Test database is configured via environment variables:

- `TEST_DB_HOST` - Database host (default: localhost)
- `TEST_DB_PORT` - Database port (default: 5433)
- `TEST_DB_USER` - Database user (default: postgres)
- `TEST_DB_PASSWORD` - Database password (default: postgres)
- `TEST_DB_NAME` - Database name (default: patient_price_discovery_test)
- `TEST_DB_SSLMODE` - SSL mode (default: disable)
- `TEST_REDIS_HOST` - Redis host (default: localhost)
- `TEST_REDIS_PORT` - Redis port (default: 6379)

### Test Database Setup

The test database runs in Docker using `docker-compose.test.yml`:

- Uses tmpfs for faster performance
- Runs on port 5433 to avoid conflicts
- Automatically cleaned up after tests
- Migrations are applied during test suite setup
- Redis runs on port 6380 for event bus integration tests

## Test Structure

### Test Suites

Tests are organized using testify suites:

```go
type FacilityAdapterIntegrationTestSuite struct {
    suite.Suite
    client  *postgres.Client
    adapter repositories.FacilityRepository
    db      *sql.DB
}
```

### Lifecycle Hooks

- `SetupSuite()` - Runs once before all tests (creates client, runs migrations)
- `TearDownSuite()` - Runs once after all tests (closes connections)
- `SetupTest()` - Runs before each test (cleans up test data)
- `TearDownTest()` - Runs after each test (additional cleanup)

## Test Coverage

### Facility Adapter Integration Tests

- ✅ Create facility with all fields
- ✅ Create facility with nullable fields
- ✅ Get facility by ID
- ✅ Get non-existent facility (error handling)
- ✅ Update facility
- ✅ Delete facility (soft delete)
- ✅ List facilities with filters
- ✅ Search facilities by location (distance calculation)

### Future Integration Tests

- [ ] Procedure adapter tests
- [ ] Appointment adapter tests
- [ ] Insurance adapter tests
- [ ] Transaction tests
- [ ] Concurrent access tests
- [ ] Cache integration tests

## Best Practices

### 1. Test Isolation

Each test:
- Cleans up its data in SetupTest and TearDownTest
- Uses unique IDs to avoid conflicts
- Does not depend on other tests' state

### 2. External Service Mocking

Integration tests:
- Connect to real databases (PostgreSQL, Redis)
- Mock external APIs (geolocation, email, SMS)
- Use test containers when possible

### 3. Cleanup

Always clean up test data:
```go
func (suite *TestSuite) cleanupTestData() {
    // Delete in reverse order of dependencies
    tables := []string{
        "reviews", "appointments", "facilities", "users",
    }
    for _, table := range tables {
        suite.db.Exec("DELETE FROM " + table)
    }
}
```

### 4. Assertions

Use testify assertions:
```go
require.NoError(t, err)  // Fail immediately on error
assert.Equal(t, expected, actual)  // Continue on failure
assert.NotNil(t, value)  // Check for nil
```

## Continuous Integration

### GitHub Actions

```yaml
name: Integration Tests
on: [push, pull_request]
jobs:
  test:
    runs-on: ubuntu-latest
    services:
      postgres:
        image: postgres:15
        env:
          POSTGRES_PASSWORD: postgres
          POSTGRES_DB: patient_price_discovery_test
        options: >-
          --health-cmd pg_isready
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
      - run: make test-integration
```

## Troubleshooting

### Database Connection Errors

```bash
# Verify database is running
docker-compose -f docker-compose.test.yml ps

# Check logs
docker-compose -f docker-compose.test.yml logs postgres-test

# Restart database
make test-db-down
make test-db-up
```

### Migration Errors

```bash
# Manually apply migrations
psql -h localhost -p 5433 -U postgres -d patient_price_discovery_test \
  -f migrations/001_initial_schema.sql
```

### Port Conflicts

If port 5433 is in use:
1. Update `docker-compose.test.yml` to use a different port
2. Update `Makefile` TEST_DB_PORT accordingly
3. Restart test database

## Performance

Integration tests are slower than unit tests due to database I/O:

- **Unit tests**: ~10-50ms per test
- **Integration tests**: ~100-500ms per test

Tips for faster tests:
- Use tmpfs for test database (already configured)
- Run tests in parallel where possible
- Minimize database round trips
- Use transactions for cleanup when possible

## Adding New Integration Tests

1. Create test file: `tests/integration/[adapter]_integration_test.go`
2. Add build tag: `// +build integration`
3. Create test suite struct
4. Implement lifecycle hooks
5. Write test methods
6. Add helper functions
7. Update this README

Example:

```go
// +build integration

package integration

import (
    "testing"
    "github.com/stretchr/testify/suite"
)

type MyAdapterTestSuite struct {
    suite.Suite
    // Add fields
}

func (suite *MyAdapterTestSuite) SetupSuite() {
    // Initialize
}

func (suite *MyAdapterTestSuite) TestSomething() {
    // Test implementation
}

func TestMyAdapter(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test")
    }
    suite.Run(t, new(MyAdapterTestSuite))
}
```
