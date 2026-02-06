# Architecture Diagram

## System Overview

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

### 1. Initial Setup
```
User → Configure Provider → Initialize with Credentials → Ready
```

### 2. Scheduled Sync (Every 3 Days)
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

### 3. Query Current Data
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

### 4. Query Historical Data
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

## Component Relationships

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

## Configuration Flow

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

## Error Handling

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

## Deployment Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                         AWS/Cloud                               │
│                                                                 │
│  ┌──────────────┐    ┌──────────────┐    ┌──────────────┐     │
│  │   Lambda     │    │   S3 Bucket  │    │  DynamoDB    │     │
│  │   Function   │───▶│              │    │   Table      │     │
│  │              │    │  Price Data  │    │              │     │
│  │  - Scheduler │    │  Storage     │    │  Metadata    │     │
│  │  - Provider  │    └──────────────┘    └──────────────┘     │
│  └──────────────┘                                              │
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

## Security

```
┌─────────────────────────────────────────────────────────────────┐
│                      Security Layers                            │
│                                                                 │
│  1. Authentication                                              │
│     └─ Google Service Account (IAM)                            │
│        └─ Read-only access to spreadsheets                     │
│                                                                 │
│  2. Credentials Management                                      │
│     └─ AWS Secrets Manager / Environment Variables             │
│        └─ Never commit to version control                      │
│                                                                 │
│  3. Data Access                                                 │
│     └─ IAM roles for S3/DynamoDB access                        │
│        └─ Least privilege principle                            │
│                                                                 │
│  4. Validation                                                  │
│     └─ Config validation before initialization                 │
│        └─ Data validation before storage                       │
└─────────────────────────────────────────────────────────────────┘
```
