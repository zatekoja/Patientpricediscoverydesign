# Quick Reference: Price Aggregation & Service Availability Implementation

## What Changed?

### 1. Prices from Multiple Providers are Now **Averaged**
- **Old**: Last provider wins (silently overwrite)
- **New**: Average price when multiple providers report same service
- **File**: `backend/internal/application/services/provider_ingestion_service.go`
- **Function**: `calculateAveragePrice(existingPrice, newPrice float64) float64`

```go
// Example:
// Provider A: $100
// Provider B: $200
// Stored Price: $150 (average)
```

### 2. Unavailable Services are Now **Returned, Not Hidden**
- **Old**: Only returned services with `is_available=true`
- **New**: Returns all services with `isAvailable` flag (can be false)
- **File**: `backend/internal/adapters/database/procedure_adapter.go`
- **Method**: `ListByFacility()` - Removed hardcoded availability filter

**Frontend Impact**: Services with `isAvailable=false` should be rendered as "grayed out"

### 3. Service Filtering is Now **Logged**
- **What**: Every filter operation logged with details
- **Where**: Log output shows facility ID, filters applied, total vs returned count
- **Why**: Debug why services appear missing
- **File**: `backend/internal/adapters/database/procedure_adapter.go`
- **Function**: `logFilteringAudit()`

## API Behavior

### GET /api/facilities/{id}/services

**Query Parameters**:
```
?search=X-ray           # Search procedure name/description
&category=imaging       # Filter by category
&min_price=100          # Price range (inclusive)
&max_price=500
&available=true         # true/false (omit = return all)
&limit=20               # Pagination
&offset=0
&sort=price             # price, name, category, updated_at
&order=asc              # asc, desc
```

**Response**: 
```json
{
  "services": [
    {
      "id": "fp_123",
      "facility_id": "hospital_456",
      "procedure_id": "proc_789",
      "price": 150.00,
      "currency": "NGN",
      "estimated_duration": 30,
      "is_available": true,      // ← NEW: Can be false
      "created_at": "2026-02-07T...",
      "updated_at": "2026-02-07T..."
    }
  ],
  "total_count": 847,            // All matching services
  "current_page": 1,
  "total_pages": 43,
  "page_size": 20,
  "has_next": true,
  "has_prev": false,
  "filters_applied": {
    "search": "X-ray",
    "category": "imaging",
    "min_price": 100,
    "max_price": 500,
    "available": null,           // null = not filtered by availability
    "sort_by": "price",
    "sort_order": "asc"
  }
}
```

## Frontend Integration

### Update Type Definition

```typescript
// Before
servicePrices: Array<{
  procedureId?: string;
  name: string;
  price: number;
  currency: string;
  estimatedDuration?: number;
}>

// After
servicePrices: Array<{
  procedureId?: string;
  name: string;
  price: number;
  currency: string;
  estimatedDuration?: number;
  isAvailable?: boolean;  // ← NEW
}>
```

### Rendering Services

```typescript
// Render with visual distinction
services.forEach(service => {
  if (service.isAvailable) {
    // Render normally - full color, clickable
    renderService(service);
  } else {
    // Render as grayed out - stroke color, disabled/info message
    renderServiceGrayedOut(service, "Temporarily unavailable");
  }
});
```

## Testing

### Run Price Averaging Tests
```bash
cd backend
go test ./internal/application/services -v -run TestCalculateAveragePrice
```

### Check Filter Behavior
```bash
# Review documented test specifications
cat internal/adapters/database/procedure_adapter_test.go | grep "func Test"
```

### Verify Compilation
```bash
go build -o /tmp/test ./cmd/api
```

## Key Points to Remember

### ✅ Do This
- ✅ Request services with explicit `available=true` if you only want available services
- ✅ Check `total_count` for accurate pagination info
- ✅ Show `isAvailable: false` services as grayed out (don't hide them)
- ✅ Expect prices from multiple providers to be averaged
- ✅ Review logs if services seem missing

### ❌ Don't Do This
- ❌ Assume service won't be returned if `is_available=false`
- ❌ Ignore `total_count` - it reflects all matches, not page size
- ❌ Assume prices are from a single provider source
- ❌ Hide services with `isAvailable: false` - users need to see them
- ❌ Hardcode availability filtering on frontend

## Troubleshooting

### Q: Some services disappeared!
**A**: Check the audit log. Services are still there but may be:
- Filtered by category/price range
- Not matching search query
- Associated with inactive procedure
- Had `isAvailable` filtered

Run with `?limit=1000&available=null&category=null` to see all.

### Q: Price seems wrong?
**A**: If multiple providers report for same facility-procedure, it's the **average**. 
Check audit logs for which providers are synced.

### Q: Frontend shows "no results"
**A**: 
1. Check `total_count` in API response (0 = truly no matches)
2. Try removing filters one by one
3. Check logs: `FILTER_AUDIT [FacilityID=...]`

### Q: Performance degradation with many services?
**A**: Pagination works after filtering. If filtering many unavailable services:
1. Add explicit `available=true` filter to skip them early
2. Check database indexes on `(facility_id, is_available, category, price)`

## Related Files

**Backend**:
- `internal/application/services/provider_ingestion_service.go` - Price averaging
- `internal/application/services/provider_ingestion_service_test.go` - Tests
- `internal/adapters/database/procedure_adapter.go` - Service filtering & logging
- `internal/adapters/database/procedure_adapter_test.go` - Test specifications
- `internal/api/handlers/facility_handler.go` - REST API handler

**Frontend**:
- `Frontend/src/lib/mappers.ts` - Data mapping with isAvailable

**Schema**:
- `internal/graphql/schema.graphql` - GraphQL type definitions

**Documentation**:
- `IMPLEMENTATION_SUMMARY_PRICE_SERVICES_REVIEW.md` - Full details
- This file - Quick reference
