/**
 * Example usage of the External Data Provider system
 * 
 * This file demonstrates:
 * 1. Setting up the Google Sheets provider (megalek_ateru_helper)
 * 2. Configuring a document store
 * 3. Scheduling automatic data syncs
 * 4. Querying current, previous, and historical data
 */

import { MegalekAteruHelper } from './providers/MegalekAteruHelper';
import { InMemoryDocumentStore } from './stores/InMemoryDocumentStore';
import { DataSyncScheduler, SyncIntervals } from './config/DataSyncScheduler';
import { PriceData, GoogleSheetsConfig } from './types/PriceData';

/**
 * Example 1: Basic setup and initialization
 */
async function basicSetup() {
  // 1. Create a document store (in production, use S3, DynamoDB, or MongoDB)
  const documentStore = new InMemoryDocumentStore<PriceData>('price-data-store');
  
  // 2. Create the Google Sheets provider
  const megalekProvider = new MegalekAteruHelper(documentStore);
  
  // 3. Configure the provider
  const config: GoogleSheetsConfig = {
    credentials: {
      clientEmail: 'service-account@project.iam.gserviceaccount.com',
      privateKey: '-----BEGIN PRIVATE KEY-----\n...\n-----END PRIVATE KEY-----',
      projectId: 'my-project-id',
    },
    spreadsheetIds: [
      '1BxiMVs0XRA5nFMdKvBdBZjgmUUqptlbs74OgvE2upms', // Example spreadsheet ID
      '1AbCdEfGhIjKlMnOpQrStUvWxYz', // Another spreadsheet
    ],
    sheetNames: ['Price Data', 'Historical Prices'],
    columnMapping: {
      facilityName: 'Facility Name',
      procedureCode: 'CPT Code',
      procedureDescription: 'Procedure',
      price: 'Cash Price',
      effectiveDate: 'Effective Date',
    },
    syncSchedule: '0 0 */3 * *', // Every 3 days at midnight (cron format)
  };
  
  // 4. Initialize the provider
  await megalekProvider.initialize(config);
  
  console.log('Provider initialized:', megalekProvider.getName());
  
  return { megalekProvider, documentStore };
}

/**
 * Example 2: Fetching current data
 */
async function fetchCurrentData(megalekProvider: MegalekAteruHelper) {
  console.log('\n--- Fetching Current Data ---');
  
  const currentData = await megalekProvider.getCurrentData({
    limit: 100,
  });
  
  console.log('Current data retrieved:');
  console.log('- Count:', currentData.metadata?.count);
  console.log('- Timestamp:', currentData.timestamp);
  console.log('- Source:', currentData.metadata?.source);
  
  // Display sample data
  if (currentData.data.length > 0) {
    console.log('- Sample record:', currentData.data[0]);
  }
  
  return currentData;
}

/**
 * Example 3: Fetching previous data
 */
async function fetchPreviousData(megalekProvider: MegalekAteruHelper) {
  console.log('\n--- Fetching Previous Data ---');
  
  const previousData = await megalekProvider.getPreviousData({
    limit: 50,
  });
  
  console.log('Previous data retrieved:');
  console.log('- Count:', previousData.metadata?.count);
  console.log('- Type:', previousData.metadata?.type);
  
  return previousData;
}

/**
 * Example 4: Fetching historical data with time window
 */
async function fetchHistoricalData(megalekProvider: MegalekAteruHelper) {
  console.log('\n--- Fetching Historical Data ---');
  
  // Fetch data from the last 30 days
  const historicalData = await megalekProvider.getHistoricalData({
    timeWindow: '30d',
    limit: 200,
  });
  
  console.log('Historical data retrieved:');
  console.log('- Count:', historicalData.metadata?.count);
  console.log('- Date range:', historicalData.metadata?.dateRange);
  
  return historicalData;
}

/**
 * Example 5: Fetching historical data with specific date range
 */
async function fetchHistoricalDataByDateRange(megalekProvider: MegalekAteruHelper) {
  console.log('\n--- Fetching Historical Data (Date Range) ---');
  
  const startDate = new Date('2024-01-01');
  const endDate = new Date('2024-12-31');
  
  const historicalData = await megalekProvider.getHistoricalData({
    startDate,
    endDate,
    limit: 500,
  });
  
  console.log('Historical data retrieved:');
  console.log('- Count:', historicalData.metadata?.count);
  console.log('- Period:', `${startDate.toISOString()} to ${endDate.toISOString()}`);
  
  return historicalData;
}

/**
 * Example 6: Manual data sync
 */
async function manualSync(megalekProvider: MegalekAteruHelper) {
  console.log('\n--- Manual Data Sync ---');
  
  const syncResult = await megalekProvider.syncData();
  
  console.log('Sync completed:');
  console.log('- Success:', syncResult.success);
  console.log('- Records processed:', syncResult.recordsProcessed);
  console.log('- Timestamp:', syncResult.timestamp);
  
  if (!syncResult.success) {
    console.error('- Error:', syncResult.error);
  }
  
  return syncResult;
}

/**
 * Example 7: Scheduled automatic sync (every 3 days)
 */
async function scheduleAutomaticSync(megalekProvider: MegalekAteruHelper) {
  console.log('\n--- Setting up Scheduled Sync ---');
  
  const scheduler = new DataSyncScheduler();
  
  // Schedule to sync every 3 days
  scheduler.scheduleJob({
    name: 'megalek_ateru_sync',
    provider: megalekProvider,
    intervalMs: SyncIntervals.THREE_DAYS,
    runImmediately: true, // Run first sync immediately
    onComplete: (result) => {
      console.log('Scheduled sync completed:', {
        success: result.success,
        records: result.recordsProcessed,
        time: result.timestamp.toISOString(),
      });
    },
    onError: (error) => {
      console.error('Scheduled sync error:', error.message);
    },
  });
  
  console.log('Scheduled job created: megalek_ateru_sync');
  console.log('Will run every 3 days');
  
  return scheduler;
}

/**
 * Example 8: Check provider health
 */
async function checkHealth(megalekProvider: MegalekAteruHelper) {
  console.log('\n--- Checking Provider Health ---');
  
  const health = await megalekProvider.getHealthStatus();
  
  console.log('Health status:');
  console.log('- Healthy:', health.healthy);
  console.log('- Last sync:', health.lastSync?.toISOString() || 'Never');
  console.log('- Message:', health.message);
  
  return health;
}

/**
 * Main example runner
 */
async function main() {
  try {
    // Setup
    const { megalekProvider } = await basicSetup();
    
    // Check health
    await checkHealth(megalekProvider);
    
    // Fetch current data
    await fetchCurrentData(megalekProvider);
    
    // Manual sync
    await manualSync(megalekProvider);
    
    // Fetch previous and historical data
    await fetchPreviousData(megalekProvider);
    await fetchHistoricalData(megalekProvider);
    await fetchHistoricalDataByDateRange(megalekProvider);
    
    // Setup scheduled sync
    const scheduler = await scheduleAutomaticSync(megalekProvider);
    
    console.log('\n--- Example Complete ---');
    console.log('The provider is now set up and scheduled to sync every 3 days.');
    console.log('Active jobs:', scheduler.getActiveJobs());
    
    // In a real application, the scheduler would keep running
    // For this example, we'll stop it after a short delay
    setTimeout(() => {
      scheduler.stopAll();
      console.log('\nScheduler stopped.');
    }, 5000);
    
  } catch (error) {
    console.error('Error in example:', error);
  }
}

// Export for use in other modules
export {
  basicSetup,
  fetchCurrentData,
  fetchPreviousData,
  fetchHistoricalData,
  fetchHistoricalDataByDateRange,
  manualSync,
  scheduleAutomaticSync,
  checkHealth,
};

// Run if executed directly
if (require.main === module) {
  main();
}
