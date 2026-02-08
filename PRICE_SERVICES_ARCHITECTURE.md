## Price Aggregation & Service Availability Architecture

```
┌─────────────────────────────────────────────────────────────────────────┐
│                      EXTERNAL PROVIDERS                                 │
│  (Different pricing for same facility-procedure)                        │
└─────────────┬──────────────────────────┬────────────────────────────────┘
              │                          │
         Provider A                   Provider B
        (Hospital ABC)               (Hospital ABC)
        X-Ray: $100                  X-Ray: $200
              │                          │
              └──────────────┬───────────┘
                             │
        ┌────────────────────▼────────────────────┐
        │  Provider Ingestion Service             │
        │  ensureFacilityProcedure()              │
        │                                         │
        │  ✅ NEW: calculateAveragePrice()        │
        │     Price = ($100 + $200) / 2 = $150    │
        └────────────────────┬────────────────────┘
                             │
        ┌────────────────────▼────────────────────┐
        │  DATABASE                               │
        │  facility_procedures TABLE              │
        │  ┌────────────────────────────────┐    │
        │  │ facility_id    = hospital_abc  │    │
        │  │ procedure_id   = proc_xray     │    │
        │  │ price          = 150.00        │    │
        │  │ currency       = NGN           │    │
        │  │ is_available   = true          │    │
        │  │ est_duration   = 30 minutes    │    │
        │  └────────────────────────────────┘    │
        └────────────────────┬────────────────────┘
                             │
        ┌────────────────────▼──────────────────────┐
        │  PROCEDURE ADAPTER                       │
        │  ListByFacilityWithCount()               │
        │                                          │
        │  Step 1: JOIN procedures (filter active) │
        │  Step 2: Apply FILTERS                   │
        │    ├─ Category     (if specified)        │
        │    ├─ Price range  (if specified)        │
        │    ├─ Availability (if specified)        │
        │    └─ Search query (if specified)        │
        │  Step 3: Count total matches             │
        │  Step 4: Sort results                    │
        │  Step 5: ✅ Paginate (AFTER filtering)   │
        │  Step 6: ✅ Log audit trail              │
        │  Return: [FacilityProcedure] + count     │
        └────────────────────┬──────────────────────┘
                             │
        ┌────────────────────▼──────────────────────┐
        │  REST API HANDLER                        │
        │  GET /api/facilities/:id/services        │
        │                                          │
        │  Response:                               │
        │  {                                       │
        │    "services": [                         │
        │      {                                   │
        │        "id": "fp_123",                   │
        │        "price": 150.00,                  │
        │        "currency": "NGN",                │
        │        "is_available": true, ✅ NEW    │
        │        "estimated_duration": 30         │
        │      }                                   │
        │    ],                                    │
        │    "total_count": 847,    ← All matches │
        │    "current_page": 1,                    │
        │    "has_next": true                      │
        │  }                                       │
        └────────────────────┬──────────────────────┘
                             │
        ┌────────────────────▼──────────────────────┐
        │  FRONTEND MAPPER (mappers.ts)            │
        │  mapFacilitySearchResultToUI()           │
        │                                          │
        │  Maps API response to UIFacility:        │
        │  servicePrices: {                        │
        │    price: 150,                           │
        │    currency: "NGN",                      │
        │    isAvailable: true,  ✅ NEW           │
        │    estimatedDuration: 30                 │
        │  }                                       │
        └────────────────────┬──────────────────────┘
                             │
        ┌────────────────────▼──────────────────────┐
        │  FRONTEND UI                             │
        │                                          │
        │  isAvailable: true                       │
        │  ├─ Render normally                      │
        │  ├─ Full color                           │
        │  └─ Clickable/bookable                   │
        │                                          │
        │  isAvailable: false                      │
        │  ├─ Render grayed out ✅ NEW            │
        │  ├─ Stroke color                        │
        │  └─ Show "Temporarily unavailable"      │
        └────────────────────────────────────────────┘
```

## Filter Precedence & Data Flow

```
Input Query:
  ?search=xray
  &category=imaging
  &min_price=100
  &max_price=500
  &available=true
  &limit=20
  &offset=0

          │
          ▼
┌─────────────────────────────────┐
│ Database: facility_procedures   │
│ Total records: 10,000           │
└─────────────────────────────────┘
          │
          ├─ Filter: is_active=true (procedures)
          │ Result: 9,500 records
          │
          ├─ Filter: category ILIKE 'imaging'
          │ Result: 1,200 records
          │
          ├─ Filter: price >= 100 AND price <= 500
          │ Result: 800 records
          │
          ├─ Filter: is_available=true
          │ Result: 720 records
          │
          ├─ Filter: name ILIKE 'xray' OR desc ILIKE 'xray'
          │ Result: 450 records ← totalCount = 450 (ALL matches)
          │
          ├─ Sort: ORDER BY price ASC
          │ Result: 450 records (sorted)
          │
          └─ Paginate: LIMIT 20 OFFSET 0
            Result: 20 records ← returned to client

Response:
  "services": [20 records],
  "total_count": 450,        ← ALL matching services
  "current_page": 1,
  "total_pages": 23,
  "has_next": true
```

## Price Averaging Logic

```
Multiple Providers Scenario:

              Provider A          Provider B          Provider C
              Hospital ABC        Hospital ABC        Hospital ABC
              X-Ray: $100         X-Ray: $200         X-Ray: $150

                    │                   │                   │
                    └───────────────────┼───────────────────┘
                                        │
                        ┌───────────────▼───────────────┐
                        │  calculateAveragePrice()      │
                        │                               │
                        │  1st sync (Provider A):       │
                        │    price = 100                │
                        │                               │
                        │  2nd sync (Provider B):       │
                        │    avg = (100 + 200) / 2      │
                        │    price = 150                │
                        │                               │
                        │  3rd sync (Provider C):       │
                        │    avg = (150 + 150) / 2      │
                        │    price = 150                │
                        └───────────────┬───────────────┘
                                        │
                                        ▼
                            Final Price: $150.00
                        (Average across all providers)
```

## Service Availability States

```
is_available=true (Database)
    ├─ Frontend isAvailable: true
    ├─ Render: Full color, clickable
    ├─ State: Available for booking
    └─ Action: User can select service

is_available=false (Database)
    ├─ Frontend isAvailable: false
    ├─ Render: Grayed out ✅ NEW
    ├─ State: Temporarily unavailable
    ├─ Action: Show tooltip "Temporarily unavailable"
    └─ Note: Service details still visible for comparison

Not in response (Filtered out):
    ├─ Procedure is_active=false
    │   └─ Reason: Procedure definition inactive (admin decision)
    │
    ├─ Filter not matching
    │   └─ Reason: User applied specific filter (expected)
    │
    └─ Example: 
        User searches "imaging" but facility has no imaging
        → Service not returned (expected, not an error)
```

## Key Changes Summary

```
╔════════════════════════════════════════════════════════════════╗
║              BEFORE                    AFTER                    ║
╠════════════════════════════════════════════════════════════════╣
║ Price Aggregation:                                              ║
║ • Last provider wins               • Prices averaged            ║
║ • Silent overwrite                 • Transparent calculation    ║
║                                                                 ║
║ Service Visibility:                                             ║
║ • is_available=false → Hidden      • is_available=false        ║
║ • Silent data loss                 • Returned, grayed out      ║
║ • Incomplete view                  • Complete view             ║
║                                                                 ║
║ Filter Auditing:                                                ║
║ • No logging                       • Every operation logged    ║
║ • Hard to debug                    • Easy to troubleshoot      ║
║                                                                 ║
║ Pagination:                                                     ║
║ • Search after pagination ❌       • Search before pagination ✅║
║ • Incomplete results               • Complete result set       ║
║ • Wrong page count                 • Accurate totalCount       ║
╚════════════════════════════════════════════════════════════════╝
```

## Test Coverage

```
Price Averaging (14 tests):
  ✅ Both prices zero
  ✅ One price zero (use other)
  ✅ Same prices
  ✅ Different prices
  ✅ Decimal precision
  ✅ Large differences
  ✅ Multiple provider averaging
  ✅ Edge cases
  ✅ Benchmark performance

Service Filtering (12 documented scenarios):
  ✅ No filters → all returned
  ✅ Category filter
  ✅ Price range filter
  ✅ Search filter
  ✅ Availability filter
  ✅ Combined filters
  ✅ Sorting
  ✅ Pagination
  ✅ Edge cases (missing duration, zero price, etc.)
  ✅ Inactive procedures (filtered by design)
  ✅ Empty facilities
  ✅ Audit logging
```
