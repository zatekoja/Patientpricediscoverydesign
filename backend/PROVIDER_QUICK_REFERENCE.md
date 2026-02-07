# External Data Provider Quick Reference

## Table of Contents
- [Quick Start](#quick-start)
- [Interface Methods](#interface-methods)
- [Configuration](#configuration)
- [Common Use Cases](#common-use-cases)

## Quick Start

### 1. Basic Setup
```typescript
import { MegalekAteruHelper, InMemoryDocumentStore, GoogleSheetsConfig } from './backend';

// Create store and provider
const store = new InMemoryDocumentStore<PriceData>();
const provider = new MegalekAteruHelper(store);

// Configure
const config: GoogleSheetsConfig = {
  credentials: { /* ... */ },
  spreadsheetIds: ['your-spreadsheet-id'],
};

await provider.initialize(config);
```

### 2. Get Current Data
```typescript
const data = await provider.getCurrentData({ limit: 100 });
console.log(data.data); // Array of PriceData
```

### 3. Schedule Auto-Sync
```typescript
import { DataSyncScheduler, SyncIntervals } from './backend';

const scheduler = new DataSyncScheduler();
scheduler.scheduleJob({
  name: 'price-sync',
  provider: provider,
  intervalMs: SyncIntervals.THREE_DAYS,
  runImmediately: true,
});
```

## Interface Methods

### getCurrentData(options?)
Get the most recent data from the provider.

**Parameters:**
- `options.limit` - Max records (optional)
- `options.offset` - Pagination offset (optional)
- `options.parameters` - Custom params (optional)

**Returns:** `DataProviderResponse<T>`

**Example:**
```typescript
const current = await provider.getCurrentData({ limit: 50 });
```

### getPreviousData(options?)
Get the last batch of data before current.

**Parameters:** Same as `getCurrentData`

**Returns:** `DataProviderResponse<T>`

**Example:**
```typescript
const previous = await provider.getPreviousData({ limit: 50 });
```

### getHistoricalData(options)
Query historical data within a time range.

**Parameters:**
- `options.timeWindow` - Time window string (e.g., "30d")
  - OR -
- `options.startDate` + `options.endDate` - Explicit date range
- `options.limit` - Max records (optional)
- `options.offset` - Pagination (optional)

**Returns:** `DataProviderResponse<T>`

**Examples:**
```typescript
// Last 30 days
const last30Days = await provider.getHistoricalData({ 
  timeWindow: '30d' 
});

// Specific date range
const yearData = await provider.getHistoricalData({
  startDate: new Date('2024-01-01'),
  endDate: new Date('2024-12-31'),
  limit: 1000,
});
```

### syncData()
Manually trigger data sync from external source to document store.

**Returns:** Sync result with status and metadata

**Example:**
```typescript
const result = await provider.syncData();
console.log(`Synced ${result.recordsProcessed} records`);
```

### getHealthStatus()
Check provider health and last sync time.

**Returns:** Health status object

**Example:**
```typescript
const health = await provider.getHealthStatus();
if (!health.healthy) {
  console.error('Provider unhealthy:', health.message);
}
```

## Configuration

### Google Sheets Config
```typescript
interface GoogleSheetsConfig {
  credentials: {
    clientEmail: string;      // Service account email
    privateKey: string;        // Private key (with \n)
    projectId: string;         // GCP project ID
  };
  spreadsheetIds: string[];    // Array of spreadsheet IDs
  sheetNames?: string[];       // Optional sheet names
  columnMapping?: {            // Map columns to fields
    facilityName?: string;
    procedureCode?: string;
    price?: string;
    // ... more mappings
  };
  syncSchedule?: string;       // Cron or interval
}
```

### Time Window Format
- `"7d"` - 7 days
- `"30d"` - 30 days
- `"3m"` - 3 months
- `"1y"` - 1 year

## Common Use Cases

### Use Case 1: Dashboard Display
Get current prices for display on a dashboard.

```typescript
const currentPrices = await provider.getCurrentData({ limit: 100 });

// Display in UI
currentPrices.data.forEach(price => {
  console.log(`${price.facilityName}: ${price.procedureDescription} - $${price.price}`);
});
```

### Use Case 2: Price Comparison
Compare current vs. previous prices.

```typescript
const current = await provider.getCurrentData();
const previous = await provider.getPreviousData();

// Compare prices
// ... comparison logic
```

### Use Case 3: Historical Trends
Analyze price trends over time.

```typescript
const last6Months = await provider.getHistoricalData({ 
  timeWindow: '6m',
  limit: 5000 
});

// Analyze trends
// ... trend analysis logic
```

### Use Case 4: Automated Updates
Set up automatic data updates every 3 days.

```typescript
const scheduler = new DataSyncScheduler();

scheduler.scheduleJob({
  name: 'auto-price-update',
  provider: provider,
  intervalMs: SyncIntervals.THREE_DAYS,
  runImmediately: true,
  onComplete: (result) => {
    if (result.success) {
      console.log(`Updated ${result.recordsProcessed} prices`);
      // Notify stakeholders
      // Send metrics
    }
  },
  onError: (error) => {
    // Alert on-call engineer
    console.error('Sync failed:', error);
  },
});
```

### Use Case 5: Data Quality Monitoring
Monitor data freshness and provider health.

```typescript
async function checkDataQuality() {
  const health = await provider.getHealthStatus();
  
  if (!health.healthy) {
    console.error('Provider unhealthy:', health.message);
    return;
  }
  
  const timeSinceLastSync = Date.now() - (health.lastSync?.getTime() || 0);
  const threeDaysMs = 3 * 24 * 60 * 60 * 1000;
  
  if (timeSinceLastSync > threeDaysMs) {
    console.warn('Data may be stale. Last sync:', health.lastSync);
  }
}
```

## Error Handling

### Standard Pattern
```typescript
try {
  const data = await provider.getCurrentData();
  // Use data
} catch (error) {
  console.error('Failed to fetch data:', error);
  // Fallback behavior
}
```

### Health Check Before Query
```typescript
const health = await provider.getHealthStatus();
if (health.healthy) {
  const data = await provider.getCurrentData();
} else {
  console.error('Provider unavailable');
}
```

## Best Practices

1. **Initialize Once** - Initialize providers at application startup
2. **Use Health Checks** - Monitor provider health regularly
3. **Handle Errors** - Always wrap provider calls in try-catch
4. **Use Pagination** - For large datasets, use limit/offset
5. **Monitor Sync Jobs** - Track sync success/failure rates
6. **Cache Results** - Cache current data to reduce API calls
7. **Validate Configs** - Always validate configs before initialization

## Integration with Frontend

### React Hook Example
```typescript
function usePriceData() {
  const [prices, setPrices] = useState<PriceData[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    async function fetchPrices() {
      try {
        const response = await provider.getCurrentData({ limit: 100 });
        setPrices(response.data);
      } catch (error) {
        console.error('Failed to fetch prices:', error);
      } finally {
        setLoading(false);
      }
    }
    
    fetchPrices();
  }, []);

  return { prices, loading };
}
```

## Support

For issues or questions:
1. Check the full [README](./backend/README.md)
2. Review [example-usage.ts](./backend/example-usage.ts)
3. Consult the interface documentation in code

## Next Steps

1. Set up Google Cloud credentials
2. Configure spreadsheet access
3. Choose production document store (S3/DynamoDB/MongoDB)
4. Deploy scheduler as background service
5. Set up monitoring and alerts
