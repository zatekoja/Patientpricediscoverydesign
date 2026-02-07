# Phase 2: CQRS & Advanced Search Architecture Plan

## Objective
Transition the architecture to a CQRS (Command Query Responsibility Segregation) pattern to separate write operations from high-performance search/read operations. Establish a dedicated GraphQL service for the frontend to consume, powered by Typesense.

## Architecture Overview

### 1. Command Service (Existing REST API)
*   **Role**: Handles all Write operations (Create, Update, Delete).
*   **Database**: PostgreSQL (Primary Source of Truth).
*   **Responsibility**:
    *   Validates business logic.
    *   Persists data to Postgres.
    *   **Sync**: Pushes changes to Typesense (the Read Model) immediately upon success (Synchronous for now, potentially Async via Events later).

### 2. Read Model (Typesense)
*   **Role**: High-performance, typo-tolerant search engine.
*   **Data**: Optimized, flattened JSON representation of Facilities suitable for filtering and geo-querying.
*   **Schema**:
    *   `id` (string)
    *   `name` (string)
    *   `location` (geopoint: [lat, lon])
    *   `price` (float)
    *   `specialties` (string[])
    *   `insurance_providers` (string[])

### 3. Query Service (New GraphQL Server)
*   **Role**: Dedicated Read-only API for the Frontend.
*   **Tech Stack**: Go + gqlgen.
*   **Scaling**: Deployed as a separate container, allowing independent scaling from the REST API.
*   **Responsibility**:
    *   Exposes a flexible GraphQL Schema.
    *   Resolves search queries by calling Typesense.
    *   Resolves detail queries by calling Typesense (or Postgres if deep relational data is needed).

## Implementation Steps

### Step 1: Typesense Infrastructure (Backend)
*   [ ] **Client Setup**: Configure the Typesense Go client in `backend/internal/infrastructure/clients/typesense`.
*   [ ] **Schema Definition**: Define the `facilities` collection schema programmatically.
*   [ ] **Migration/Seed**: Create a utility to index existing Postgres data into Typesense.
*   [ ] **Sync Logic**: Update `FacilityService` (or Handler) to index data on Create/Update.

### Step 2: GraphQL Service Setup
*   [ ] **Scaffold**: Create `graphql-service` directory.
*   [ ] **Schema**: Define `schema.graphql` with types `Facility`, `SearchInput`, etc.
*   [ ] **Generate**: Run `gqlgen` to generate Go code.
*   [ ] **Resolvers**: Implement `searchFacilities` resolver to query Typesense.

### Step 3: Deployment & Integration
*   [ ] **Docker**: Add `graphql-service` to `docker-compose.yml`.
*   [ ] **Gateway**: Ensure the Frontend can talk to the GraphQL port (e.g., 8081).
*   [ ] **Frontend**: Update React app to use Apollo Client.

## Future Scaling (Phase 3+)
*   **Async Sync**: Use a message queue (RabbitMQ/Kafka) between Command Service and Typesense to decouple writes from indexing.
