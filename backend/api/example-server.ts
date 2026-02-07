/**
 * Example API Server Setup
 * Demonstrates how to start the HTTP API server with configured providers
 * 
 * NOTE: This is an example for development/testing.
 * For production, ensure all environment variables are properly configured.
 */

import '../observability/otel';
import { DataProviderAPI } from './server';
import { MegalekAteruHelper } from '../providers/MegalekAteruHelper';
import { FilePriceListProvider, FilePriceListConfig } from '../providers/FilePriceListProvider';
import { InMemoryDocumentStore } from '../stores/InMemoryDocumentStore';
import { DataSyncScheduler, SyncIntervals } from '../config/DataSyncScheduler';
import { PriceData, GoogleSheetsConfig } from '../types/PriceData';

/**
 * Initialize and start the API server
 */
async function startServer() {
  const priceListFiles = (process.env.PRICE_LIST_FILES || '')
    .split(',')
    .map((value) => value.trim())
    .filter(Boolean);
  const useFileProvider = priceListFiles.length > 0;

  // Validate required environment variables for Google Sheets
  if (!useFileProvider) {
    const requiredEnvVars = ['GOOGLE_CLIENT_EMAIL', 'GOOGLE_PRIVATE_KEY', 'GOOGLE_PROJECT_ID', 'SPREADSHEET_IDS'];
    const missingVars = requiredEnvVars.filter(v => !process.env[v]);
    
    if (missingVars.length > 0 && process.env.NODE_ENV === 'production') {
      console.error(`Missing required environment variables: ${missingVars.join(', ')}`);
      console.error('Please configure these variables before starting the server in production.');
      process.exit(1);
    }
  }

  // 1. Create the API server
  const api = new DataProviderAPI();

  // 2. Create document store
  const documentStore = new InMemoryDocumentStore<PriceData>('price-data-store');

  // 3. Create and configure provider
  let providerInitialized = false;
  let scheduler: DataSyncScheduler | undefined;
  let providerId = 'megalek_ateru_helper';
  let provider: MegalekAteruHelper | FilePriceListProvider = new MegalekAteruHelper(documentStore);

  if (useFileProvider) {
    providerId = 'file_price_list';
    provider = new FilePriceListProvider(documentStore);
  }

  // Configuration (use environment variables or provide defaults for development)
  const config: GoogleSheetsConfig = {
    credentials: {
      clientEmail: process.env.GOOGLE_CLIENT_EMAIL || '',
      privateKey: process.env.GOOGLE_PRIVATE_KEY || '',
      projectId: process.env.GOOGLE_PROJECT_ID || '',
    },
    spreadsheetIds: (process.env.SPREADSHEET_IDS || '').split(',').filter(Boolean),
    sheetNames: ['Price Data'],
    columnMapping: {
      facilityName: 'Facility Name',
      procedureCode: 'CPT Code',
      procedureDescription: 'Procedure',
      price: 'Cash Price',
      effectiveDate: 'Effective Date',
    },
  };

  const fileConfig: FilePriceListConfig = {
    files: priceListFiles.map((filePath) => ({ path: filePath })),
    currency: process.env.PRICE_LIST_CURRENCY || 'NGN',
    defaultEffectiveDate: process.env.PRICE_LIST_EFFECTIVE_DATE,
  };

  // 4. Initialize the provider
  try {
    if (useFileProvider) {
      await (provider as FilePriceListProvider).initialize(fileConfig);
    } else {
      await (provider as MegalekAteruHelper).initialize(config);
    }
    providerInitialized = true;
    console.log(`✓ Provider initialized successfully (${providerId})`);
  } catch (error) {
    console.error('✗ Failed to initialize provider:', error);
    if (process.env.NODE_ENV === 'production') {
      process.exit(1);
    } else {
      console.warn('⚠ Continuing in development mode without provider (API will return errors)');
    }
  }

  // 5. Register the provider with the API only if initialized
  if (providerInitialized) {
    api.registerProvider(providerId, provider, true);
    console.log(`✓ Provider registered with API (${providerId})`);

    // 6. Setup scheduled sync (optional)
    scheduler = new DataSyncScheduler();
    scheduler.scheduleJob({
      name: `${providerId}_sync`,
      provider,
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
  } else {
    console.warn('⚠ Skipping provider registration and sync scheduler due to initialization failure');
  }

  // 7. Start the API server
  const port = parseInt(process.env.PORT || '3000', 10);
  api.listen(port);

  // Handle graceful shutdown
  process.on('SIGTERM', () => {
    console.log('Received SIGTERM, shutting down gracefully...');
    if (scheduler) {
      scheduler.stopAll();
    }
    process.exit(0);
  });

  process.on('SIGINT', () => {
    console.log('Received SIGINT, shutting down gracefully...');
    if (scheduler) {
      scheduler.stopAll();
    }
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
