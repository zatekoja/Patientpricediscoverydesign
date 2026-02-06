/**
 * Example API Server Setup
 * Demonstrates how to start the HTTP API server with configured providers
 */

import { DataProviderAPI } from './server';
import { MegalekAteruHelper } from '../providers/MegalekAteruHelper';
import { InMemoryDocumentStore } from '../stores/InMemoryDocumentStore';
import { DataSyncScheduler, SyncIntervals } from '../config/DataSyncScheduler';
import { PriceData, GoogleSheetsConfig } from '../types/PriceData';

/**
 * Initialize and start the API server
 */
async function startServer() {
  // 1. Create the API server
  const api = new DataProviderAPI();

  // 2. Create document store
  const documentStore = new InMemoryDocumentStore<PriceData>('price-data-store');

  // 3. Create and configure the Google Sheets provider
  const megalekProvider = new MegalekAteruHelper(documentStore);

  // Example configuration (replace with actual credentials)
  const config: GoogleSheetsConfig = {
    credentials: {
      clientEmail: process.env.GOOGLE_CLIENT_EMAIL || 'service-account@project.iam.gserviceaccount.com',
      privateKey: process.env.GOOGLE_PRIVATE_KEY || '-----BEGIN PRIVATE KEY-----\n...\n-----END PRIVATE KEY-----',
      projectId: process.env.GOOGLE_PROJECT_ID || 'my-project-id',
    },
    spreadsheetIds: (process.env.SPREADSHEET_IDS || '').split(',').filter(Boolean) || [
      '1BxiMVs0XRA5nFMdKvBdBZjgmUUqptlbs74OgvE2upms',
    ],
    sheetNames: ['Price Data'],
    columnMapping: {
      facilityName: 'Facility Name',
      procedureCode: 'CPT Code',
      procedureDescription: 'Procedure',
      price: 'Cash Price',
      effectiveDate: 'Effective Date',
    },
  };

  // 4. Initialize the provider
  try {
    await megalekProvider.initialize(config);
    console.log('✓ Provider initialized successfully');
  } catch (error) {
    console.error('✗ Failed to initialize provider:', error);
    process.exit(1);
  }

  // 5. Register the provider with the API
  api.registerProvider('megalek_ateru_helper', megalekProvider, true);
  console.log('✓ Provider registered with API');

  // 6. Setup scheduled sync (optional)
  const scheduler = new DataSyncScheduler();
  scheduler.scheduleJob({
    name: 'megalek_sync',
    provider: megalekProvider,
    intervalMs: SyncIntervals.THREE_DAYS,
    runImmediately: false, // Set to true to sync on startup
    onComplete: (result) => {
      console.log(`Scheduled sync completed: ${result.recordsProcessed} records, success: ${result.success}`);
    },
    onError: (error) => {
      console.error('Scheduled sync error:', error.message);
    },
  });
  console.log('✓ Scheduled sync configured (every 3 days)');

  // 7. Start the API server
  const port = parseInt(process.env.PORT || '3000', 10);
  api.listen(port);

  // Handle graceful shutdown
  process.on('SIGTERM', () => {
    console.log('Received SIGTERM, shutting down gracefully...');
    scheduler.stopAll();
    process.exit(0);
  });

  process.on('SIGINT', () => {
    console.log('Received SIGINT, shutting down gracefully...');
    scheduler.stopAll();
    process.exit(0);
  });
}

// Start the server
if (require.main === module) {
  startServer().catch((error) => {
    console.error('Failed to start server:', error);
    process.exit(1);
  });
}

export { startServer };
