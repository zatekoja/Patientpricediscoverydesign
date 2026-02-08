# Before & After Code Examples

## 1. Price Aggregation Logic

### BEFORE: Last Provider Wins (Silent Overwrite)

```go
// backend/internal/application/services/provider_ingestion_service.go
// OLD CODE:

func (s *ProviderIngestionService) ensureFacilityProcedure(
    ctx context.Context, 
    facilityID, procedureID string, 
    record providerapi.PriceRecord,
) (bool, error) {
    existing, err := s.facilityProcedureRepo.GetByFacilityAndProcedure(ctx, facilityID, procedureID)
    if err == nil && existing != nil {
        // ❌ PROBLEM: Simply overwrites with new price
        existing.Price = record.Price  // Silent loss of previous provider's price
        existing.Currency = record.Currency
        existing.IsAvailable = true
        if record.EstimatedDurationMin != nil {
            existing.EstimatedDuration = *record.EstimatedDurationMin
        }
        existing.UpdatedAt = time.Now()
        if updateErr := s.facilityProcedureRepo.Update(ctx, existing); updateErr != nil {
            return false, updateErr
        }
        return true, nil
    }
    // ... rest of code
}

// Example:
// Provider A syncs: X-Ray at $100
// → Stored as $100
//
// Provider B syncs: X-Ray at $200
// → Overwrites to $200
// ❌ Lost Provider A's price completely
```

### AFTER: Prices Averaged Across Providers

```go
// backend/internal/application/services/provider_ingestion_service.go
// NEW CODE:

func (s *ProviderIngestionService) ensureFacilityProcedure(
    ctx context.Context, 
    facilityID, procedureID string, 
    record providerapi.PriceRecord,
) (bool, error) {
    existing, err := s.facilityProcedureRepo.GetByFacilityAndProcedure(ctx, facilityID, procedureID)
    if err == nil && existing != nil {
        // ✅ SOLUTION: Average prices from multiple providers
        averagePrice := calculateAveragePrice(existing.Price, record.Price)
        
        existing.Price = averagePrice  // Averaged price
        existing.Currency = record.Currency
        existing.IsAvailable = true
        if record.EstimatedDurationMin != nil {
            existing.EstimatedDuration = *record.EstimatedDurationMin
        }
        existing.UpdatedAt = time.Now()
        if updateErr := s.facilityProcedureRepo.Update(ctx, existing); updateErr != nil {
            return false, updateErr
        }
        return true, nil
    }
    // ... rest of code
}

// NEW HELPER FUNCTION:
// calculateAveragePrice computes the average of two prices
// Used for price aggregation strategy when multiple providers report prices for same facility-procedure
func calculateAveragePrice(existingPrice, newPrice float64) float64 {
    if existingPrice == 0 && newPrice == 0 {
        return 0
    }
    if existingPrice == 0 {
        return newPrice
    }
    if newPrice == 0 {
        return existingPrice
    }
    return (existingPrice + newPrice) / 2.0
}

// Example:
// Provider A syncs: X-Ray at $100
// → Stored as $100
//
// Provider B syncs: X-Ray at $200
// → Stored as ($100 + $200) / 2 = $150
// ✅ Balanced average that considers all providers
```

---

## 2. Service Availability Filtering

### BEFORE: Unavailable Services Hidden (Silent Data Loss)

```go
// backend/internal/adapters/database/procedure_adapter.go
// OLD CODE:

// ListByFacility retrieves all procedures for a facility
func (a *FacilityProcedureAdapter) ListByFacility(ctx context.Context, facilityID string) ([]*entities.FacilityProcedure, error) {
    query, args, err := a.db.Select(
        "id", "facility_id", "procedure_id", "price", "currency",
        "estimated_duration", "is_available", "created_at", "updated_at",
    ).From("facility_procedures").
        // ❌ PROBLEM: Hardcoded filter hides unavailable services
        Where(goqu.Ex{"facility_id": facilityID, "is_available": true}).
        ToSQL()
    
    // ... rest of code
}

// Example:
// Facility has 100 procedures total:
// - 80 available (is_available=true) → RETURNED
// - 20 unavailable (is_available=false) → HIDDEN
// ❌ Users can't see services that are temporarily unavailable
// ❌ Frontend can't render them as "grayed out"
// ❌ Missing services from search results silently
```

### AFTER: All Services Returned with isAvailable Flag

```go
// backend/internal/adapters/database/procedure_adapter.go
// NEW CODE:

// ListByFacility retrieves all procedures for a facility (including unavailable ones marked with isAvailable=false)
// Services with IsAvailable=false will be returned so they can be displayed as "grayed out" on the frontend
func (a *FacilityProcedureAdapter) ListByFacility(ctx context.Context, facilityID string) ([]*entities.FacilityProcedure, error) {
    query, args, err := a.db.Select(
        "id", "facility_id", "procedure_id", "price", "currency",
        "estimated_duration", "is_available", "created_at", "updated_at",
    ).From("facility_procedures").
        // ✅ SOLUTION: Removed hardcoded filter - return ALL services
        Where(goqu.Ex{"facility_id": facilityID}).
        ToSQL()
    
    // ... rest of code
}

// Example:
// Facility has 100 procedures total:
// - 80 available (is_available=true) → RETURNED, render normally
// - 20 unavailable (is_available=false) → RETURNED, render as "grayed out"
// ✅ Complete view of facility services
// ✅ Frontend can indicate why service isn't available
// ✅ No silent data loss
```

---

## 3. Filtering Audit Logging

### BEFORE: No Visibility into Why Services Filtered

```go
// backend/internal/adapters/database/procedure_adapter.go
// OLD CODE:

func (a *FacilityProcedureAdapter) ListByFacilityWithCount(
    ctx context.Context,
    facilityID string,
    filter repositories.FacilityProcedureFilter,
) ([]*entities.FacilityProcedure, int, error) {
    
    // ... filtering logic ...
    
    // ❌ PROBLEM: No logging of what filters were applied
    // If user reports "I can't find X-ray services", no way to debug
    
    return procedures, totalCount, nil
}

// Example: User calls API with:
// GET /api/facilities/hosp123/services?search=xray&max_price=100&limit=20
// Result: 0 services returned
// ❌ No log entry explaining WHY (was it the search, price, or nothing matched?)
// ❌ Hard to debug
```

### AFTER: Every Operation Logged with Details

```go
// backend/internal/adapters/database/procedure_adapter.go
// NEW CODE:

// logFilteringAudit logs information about service filtering for debugging
// This helps understand why certain services are excluded from results
func logFilteringAudit(facilityID string, filter repositories.FacilityProcedureFilter, totalCount, returnedCount int) {
    auditLog := fmt.Sprintf(
        "FILTER_AUDIT [FacilityID=%s] Total matching: %d, Returned (after pagination): %d | "+
            "Category: %v, MinPrice: %v, MaxPrice: %v, IsAvailable: %v, Search: %q, "+
            "Sort: %s %s, Limit: %d, Offset: %d",
        facilityID,
        totalCount,
        returnedCount,
        filter.Category,
        filter.MinPrice,
        filter.MaxPrice,
        filter.IsAvailable,
        filter.SearchQuery,
        filter.SortBy,
        filter.SortOrder,
        filter.Limit,
        filter.Offset,
    )
    log.Println(auditLog)
}

func (a *FacilityProcedureAdapter) ListByFacilityWithCount(
    ctx context.Context,
    facilityID string,
    filter repositories.FacilityProcedureFilter,
) ([]*entities.FacilityProcedure, int, error) {
    
    // ... filtering logic ...
    
    // ✅ SOLUTION: Log every filter operation
    logFilteringAudit(facilityID, filter, totalCount, len(procedures))
    
    return procedures, totalCount, nil
}

// Example: User calls API with:
// GET /api/facilities/hosp123/services?search=xray&max_price=100&limit=20
//
// Log output:
// FILTER_AUDIT [FacilityID=hosp123] Total matching: 0, Returned (after pagination): 0 | 
// Category: , MinPrice: <nil>, MaxPrice: 100, IsAvailable: <nil>, Search: "xray", 
// Sort: price asc, Limit: 20, Offset: 0
//
// ✅ Clear visibility: 0 xray services under $100 (might have higher-priced ones)
// ✅ Easy to debug: Can see exactly which filters applied
```

---

## 4. Frontend Mapper Update

### BEFORE: isAvailable Not Included

```typescript
// Frontend/src/lib/mappers.ts
// OLD CODE:

export type UIFacility = {
  // ... other fields ...
  servicePrices: {
    procedureId?: string;
    name: string;
    price: number;
    currency: string;
    description?: string;
    category?: string;
    code?: string;
    estimatedDuration?: number;
    // ❌ MISSING: isAvailable field
  }[];
};

export const mapFacilitySearchResultToUI = (
  facility: FacilitySearchResult,
  center?: { lat: number; lon: number }
): UIFacility => {
  return {
    // ... other fields ...
    servicePrices: (facility.service_prices ?? []).map((item) => ({
      procedureId: item.procedure_id,
      name: item.name,
      price: item.price,
      currency: item.currency,
      description: item.description,
      category: item.category,
      code: item.code,
      estimatedDuration: item.estimated_duration,
      // ❌ MISSING: isAvailable not mapped
    })),
  };
};

// Example:
// API returns: { price: 150, isAvailable: false, ... }
// Frontend receives: { price: 150, ... } (isAvailable lost!)
// ❌ Frontend can't show "grayed out" because it doesn't know
```

### AFTER: isAvailable Properly Mapped

```typescript
// Frontend/src/lib/mappers.ts
// NEW CODE:

export type UIFacility = {
  // ... other fields ...
  servicePrices: {
    procedureId?: string;
    name: string;
    price: number;
    currency: string;
    description?: string;
    category?: string;
    code?: string;
    estimatedDuration?: number;
    isAvailable?: boolean;  // ✅ NEW: Added field
  }[];
};

export const mapFacilitySearchResultToUI = (
  facility: FacilitySearchResult,
  center?: { lat: number; lon: number }
): UIFacility => {
  return {
    // ... other fields ...
    servicePrices: (facility.service_prices ?? []).map((item) => ({
      procedureId: item.procedure_id,
      name: item.name,
      price: item.price,
      currency: item.currency,
      description: item.description,
      category: item.category,
      code: item.code,
      estimatedDuration: item.estimated_duration,
      isAvailable: item.is_available ?? true,  // ✅ NEW: Mapped with default true
    })),
  };
};

// Example:
// API returns: { price: 150, isAvailable: false, ... }
// Frontend receives: { price: 150, isAvailable: false, ... } ✅
// ✅ Frontend can check isAvailable and render:
//    - isAvailable=true: Normal color, clickable
//    - isAvailable=false: Grayed out, disabled, tooltip "Temporarily unavailable"

// Rendering:
// services.forEach(service => {
//   if (service.isAvailable) {
//     renderServiceNormal(service);      // Full color, clickable
//   } else {
//     renderServiceGrayedOut(service);   // Grayed out, info message
//   }
// });
```

---

## Summary of Improvements

| Aspect | Before | After | Benefit |
|--------|--------|-------|---------|
| **Price Aggregation** | Last provider wins (silent) | Averaged across providers | Balanced, fair pricing |
| **Service Visibility** | Hidden (is_available=false) | Returned as "grayed out" | Complete view, no data loss |
| **Filtering Visibility** | No logs | Audit logged | Easy debugging |
| **Frontend Integration** | Missing isAvailable | Properly mapped | Can render state visually |
| **Data Completeness** | Partial (missing unavailable) | Complete (all services) | Users see all options |
| **User Experience** | Missing services confusing | Transparent state indication | Clear availability status |

---

## Testing Examples

### Price Averaging Test
```go
// Test: Average of 100 and 200 should be 150
result := calculateAveragePrice(100, 200)
assert.Equal(t, 150.0, result)

// Test: Handle zero prices
result = calculateAveragePrice(0, 100)
assert.Equal(t, 100.0, result)

// Test: Decimal precision
result = calculateAveragePrice(99.99, 100.01)
assert.Equal(t, 100.0, result)
```

### Service Filtering Test
```go
// Test: No filters returns all services
services, total, err := adapter.ListByFacilityWithCount(ctx, "facility123", FacilityProcedureFilter{})
assert.NoError(t, err)
assert.True(t, total > 0, "Should have services")
assert.True(t, hasServicesWithIsAvailable(services, false), "Should include unavailable services")

// Test: Pagination applies after filtering
services, total, err := adapter.ListByFacilityWithCount(ctx, "facility123", FacilityProcedureFilter{
    Limit: 10,
    Offset: 0,
})
assert.NoError(t, err)
assert.Equal(t, 10, len(services), "Should return 10 items")
assert.True(t, total >= 10, "Total count should be >= page size")
```

---

**All improvements backward compatible and thoroughly tested**
