# External Data Provider System

This system provides a flexible interface for connecting external data providers to our Patient Price Discovery application. The primary implementation is the `megalek_ateru_helper` provider, which connects to Google Sheets to retrieve price data.

## Quick Links

- **[REST API Documentation](./api/README.md)** - HTTP API for accessing price data
- **[OpenAPI Specification](./api/openapi.yaml)** - Complete API specification
- **[Quick Reference](./QUICK_REFERENCE.md)** - Common use cases and examples
- **[Architecture Diagrams](./ARCHITECTURE.md)** - System architecture

## Architecture Overview

### Core Components

1. **IExternalDataProvider Interface** - Contract that all external providers must implement
2. **BaseDataProvider** - Abstract base class providing common functionality
3. **MegalekAteruHelper** - Google Sheets implementation of the data provider
4. **IDocumentStore Interface** - Abstraction for data storage (S3, DynamoDB, MongoDB)
5. **DataSyncScheduler** - Scheduler for automatic data synchronization
6. **REST API** - HTTP endpoints for external services to access data

### Directory Structure

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

## Features

### Data Provider Interface

All external providers support:

- **Current Data** - Get the most recent data
- **Previous Data** - Get the last batch before current
- **Historical Data** - Query data within a time range
- **Configurable Options** - Time windows, pagination, custom parameters
- **Automatic Sync** - Scheduled data synchronization
- **Health Monitoring** - Check provider status

### REST API

The system includes a production-ready REST API:

- **HTTP Endpoints** - RESTful API for data access
- **OpenAPI 3.0 Spec** - Complete API documentation
- **Error Handling** - Standardized error responses
- **CORS Support** - Cross-origin resource sharing
- **Pagination** - Support for large datasets

See [API Documentation](./api/README.md) for details.

### Configurable Options

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

## Google Sheets Provider (megalek_ateru_helper)

### Configuration

```typescript
const config: GoogleSheetsConfig = {
  credentials: {
    clientEmail: 'service-account@project.iam.gserviceaccount.com',
    privateKey: '-----BEGIN PRIVATE KEY-----\n...\n-----END PRIVATE KEY-----',
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

### Usage

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

### Scheduled Sync (Every 3 Days)

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

## Data Flow

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

## Document Store Implementations

The system supports any document store that implements `IDocumentStore`:

### In-Memory Store (Development)
```typescript
const store = new InMemoryDocumentStore<PriceData>('my-store');
```

### Production Stores (To Implement)

**S3 Document Store**
- Store data as JSON files in S3 buckets
- Use prefixes for organization

**DynamoDB Document Store**
- Store as items in DynamoDB table
- Use GSI for efficient queries

**MongoDB Document Store**
- Store as documents in MongoDB collection
- Use indexes for query performance

## Adding a New Provider

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

## Time Window Format

Time windows use the format: `<number><unit>`

- `d` - Days (e.g., `"30d"` = 30 days)
- `m` - Months (e.g., `"6m"` = 6 months)
- `y` - Years (e.g., `"1y"` = 1 year)

Examples:
- `"7d"` - Last 7 days
- `"3m"` - Last 3 months
- `"1y"` - Last year

## Error Handling

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

## Testing

See `example-usage.ts` for comprehensive examples of:
- Setting up providers
- Fetching data
- Manual sync
- Scheduled sync
- Health checks

## Production Deployment

### Google Sheets API Setup

1. Create a Google Cloud Project
2. Enable Google Sheets API
3. Create a Service Account
4. Download credentials JSON
5. Share spreadsheets with service account email

### Environment Variables

```bash
GOOGLE_CLIENT_EMAIL=service-account@project.iam.gserviceaccount.com
GOOGLE_PRIVATE_KEY=-----BEGIN PRIVATE KEY-----\n...\n-----END PRIVATE KEY-----
GOOGLE_PROJECT_ID=my-project-id
SPREADSHEET_IDS=id1,id2,id3
SYNC_INTERVAL_MS=259200000  # 3 days
```

### Dependencies

```json
{
  "dependencies": {
    "googleapis": "^118.0.0",  // For Google Sheets API
    "aws-sdk": "^2.1400.0",     // For S3/DynamoDB (if used)
    "mongodb": "^5.6.0"         // For MongoDB (if used)
  }
}
```

## Next Steps

1. **Implement Production Document Store** - Choose S3, DynamoDB, or MongoDB
2. **Add Google Sheets API Integration** - Replace placeholder with actual API calls
3. **Set Up Authentication** - Configure service account credentials
4. **Deploy Scheduler** - Run as a background service or AWS Lambda
5. **Add Monitoring** - Track sync success/failure rates
6. **Add Data Validation** - Ensure data quality from spreadsheets

## License

See project LICENSE file.
