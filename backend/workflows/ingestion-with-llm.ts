/**
 * Ingestion Workflow with LLM Tag Generation
 * 
 * This example demonstrates how to integrate the LLM Tag Generator
 * into the data ingestion pipeline:
 * 
 * 1. Google Sheets Provider fetches price data
 * 2. LLM Tag Generator enriches data with contextual tags
 * 3. Tagged data is stored in document store
 * 4. Tagged data is indexed in Typesense for search
 */

import '../observability/otel';
import { MegalekAteruHelper } from '../providers/MegalekAteruHelper';
import { LLMTagGeneratorProvider } from '../providers/LLMTagGeneratorProvider';
import { InMemoryDocumentStore } from '../stores/InMemoryDocumentStore';
import { DataSyncScheduler, SyncIntervals } from '../config/DataSyncScheduler';
import { PriceData, GoogleSheetsConfig } from '../types/PriceData';
import { LLMTagGeneratorConfig, TaggedPriceData } from '../providers/LLMTagGeneratorProvider';

/**
 * Setup the complete ingestion workflow
 */
async function setupIngestionWorkflow() {
  console.log('=== Setting up Data Ingestion Workflow ===\n');
  
  // Step 1: Create document stores
  const rawDataStore = new InMemoryDocumentStore<PriceData>('raw-price-data');
  const taggedDataStore = new InMemoryDocumentStore<TaggedPriceData>('tagged-price-data');
  
  // Step 2: Configure and initialize Google Sheets provider (source)
  const googleSheetsProvider = new MegalekAteruHelper(rawDataStore);
  
  const sheetsConfig: GoogleSheetsConfig = {
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
  
  await googleSheetsProvider.initialize(sheetsConfig);
  console.log('✓ Google Sheets provider initialized');
  
  // Step 3: Configure and initialize LLM Tag Generator provider
  const llmProvider = new LLMTagGeneratorProvider(taggedDataStore, googleSheetsProvider);
  
  const llmConfig: LLMTagGeneratorConfig = {
    apiEndpoint: process.env.LLM_API_ENDPOINT || 'https://api.openai.com/v1/chat/completions',
    apiKey: process.env.LLM_API_KEY || '',
    model: process.env.LLM_MODEL || 'gpt-4',
    maxTags: 10,
    temperature: 0.3,
  };
  
  await llmProvider.initialize(llmConfig);
  console.log('✓ LLM Tag Generator provider initialized');
  
  // Step 4: Set up the ingestion pipeline
  const scheduler = new DataSyncScheduler();
  
  // Schedule Google Sheets sync every 3 days
  scheduler.scheduleJob({
    name: 'google-sheets-sync',
    provider: googleSheetsProvider,
    intervalMs: SyncIntervals.THREE_DAYS,
    runImmediately: false,
    onComplete: async (result) => {
      console.log(`\n[Google Sheets Sync] Completed:`);
      console.log(`  - Records: ${result.recordsProcessed}`);
      console.log(`  - Success: ${result.success}`);
      
      if (result.success && result.recordsProcessed > 0) {
        // Trigger LLM tagging after successful Google Sheets sync
        console.log('\n[Triggering LLM Tag Generation]');
        const tagResult = await llmProvider.syncData();
        console.log(`  - Tagged Records: ${tagResult.recordsProcessed}`);
        console.log(`  - Success: ${tagResult.success}`);
        
        if (tagResult.success) {
          // After tagging, index in Typesense
          await indexInTypesense(taggedDataStore);
        }
      }
    },
    onError: (error) => {
      console.error('[Google Sheets Sync] Error:', error.message);
    },
  });
  
  console.log('✓ Ingestion workflow scheduled');
  console.log('\nWorkflow steps:');
  console.log('  1. Google Sheets Provider → Fetch price data');
  console.log('  2. LLM Tag Generator → Add contextual tags');
  console.log('  3. Typesense Indexer → Index for search');
  
  return { googleSheetsProvider, llmProvider, scheduler };
}

/**
 * Index tagged data in Typesense for contextual search
 */
async function indexInTypesense(taggedDataStore: InMemoryDocumentStore<TaggedPriceData>) {
  console.log('\n[Typesense Indexing] Starting...');
  
  // Get all tagged data
  const taggedData = await taggedDataStore.getAll();
  
  if (taggedData.length === 0) {
    console.log('  - No data to index');
    return;
  }
  
  // Placeholder for Typesense integration
  // In production, this would use the Typesense client:
  /*
  const Typesense = require('typesense');
  
  const client = new Typesense.Client({
    nodes: [{
      host: process.env.TYPESENSE_HOST || 'localhost',
      port: process.env.TYPESENSE_PORT || '8108',
      protocol: 'http'
    }],
    apiKey: process.env.TYPESENSE_API_KEY || '',
  });
  
  // Define schema
  const schema = {
    name: 'healthcare_prices',
    fields: [
      { name: 'id', type: 'string' },
      { name: 'facilityName', type: 'string', facet: true },
      { name: 'procedureCode', type: 'string', facet: true },
      { name: 'procedureDescription', type: 'string' },
      { name: 'price', type: 'float', facet: true },
      { name: 'tags', type: 'string[]', facet: true }, // LLM-generated tags
      { name: 'location', type: 'object' },
      { name: 'effectiveDate', type: 'int64' },
    ],
  };
  
  // Create or update collection
  try {
    await client.collections('healthcare_prices').delete();
  } catch (e) {
    // Collection might not exist
  }
  
  await client.collections().create(schema);
  
  // Index documents
  const documents = taggedData.map(item => ({
    id: item.id,
    facilityName: item.facilityName,
    procedureCode: item.procedureCode,
    procedureDescription: item.procedureDescription,
    price: item.price,
    tags: item.tags || [],
    location: item.location,
    effectiveDate: new Date(item.effectiveDate).getTime(),
  }));
  
  await client.collections('healthcare_prices').documents().import(documents);
  
  console.log(`  - Indexed ${documents.length} documents in Typesense`);
  */
  
  console.log(`  - Would index ${taggedData.length} documents (placeholder)`);
  console.log('  - Sample tags from first item:', taggedData[0]?.tags || []);
  console.log('✓ Typesense indexing complete');
}

/**
 * Manual trigger for the complete workflow
 */
async function runIngestionWorkflowManually() {
  const { googleSheetsProvider, llmProvider } = await setupIngestionWorkflow();
  
  console.log('\n=== Running Manual Ingestion ===\n');
  
  // Step 1: Sync from Google Sheets
  console.log('[Step 1] Syncing from Google Sheets...');
  const sheetsResult = await googleSheetsProvider.syncData();
  console.log(`  - Records processed: ${sheetsResult.recordsProcessed}`);
  console.log(`  - Success: ${sheetsResult.success}`);
  
  if (!sheetsResult.success) {
    console.error('  - Error:', sheetsResult.error);
    return;
  }
  
  // Step 2: Generate tags with LLM
  console.log('\n[Step 2] Generating tags with LLM...');
  const tagResult = await llmProvider.syncData();
  console.log(`  - Records processed: ${tagResult.recordsProcessed}`);
  console.log(`  - Success: ${tagResult.success}`);
  
  if (!tagResult.success) {
    console.error('  - Error:', tagResult.error);
    return;
  }
  
  // Step 3: Index in Typesense
  console.log('\n[Step 3] Indexing in Typesense...');
  const taggedStore = llmProvider.getDocumentStore();
  if (taggedStore) {
    await indexInTypesense(taggedStore as InMemoryDocumentStore<TaggedPriceData>);
  }
  
  console.log('\n✓ Complete ingestion workflow finished successfully');
}

/**
 * Example: Query tagged data for contextual search
 */
async function exampleContextualSearch() {
  console.log('\n=== Example: Contextual Search ===\n');
  
  // This would use Typesense in production
  // Example queries:
  
  console.log('Example searches powered by LLM-generated tags:');
  console.log('  1. "brain imaging" → finds CT/MRI scans of head/brain');
  console.log('  2. "knee pain" → finds procedures related to knee injuries');
  console.log('  3. "diagnostic" → finds all diagnostic procedures');
  console.log('  4. "cardiology" → finds heart-related procedures');
  console.log('  5. "outpatient" → finds procedures typically done outpatient');
  
  // Typesense query example:
  /*
  const searchResults = await client.collections('healthcare_prices')
    .documents()
    .search({
      q: 'brain imaging',
      query_by: 'procedureDescription,tags',
      filter_by: 'price:<2000',
      sort_by: 'price:asc',
    });
  
  console.log('Search results:', searchResults.hits);
  */
}

// Export functions
export {
  setupIngestionWorkflow,
  runIngestionWorkflowManually,
  indexInTypesense,
  exampleContextualSearch,
};

// Run if executed directly
if (require.main === module) {
  setupIngestionWorkflow()
    .then(() => {
      console.log('\n✓ Workflow setup complete');
      console.log('  To run manual ingestion: runIngestionWorkflowManually()');
    })
    .catch((error) => {
      console.error('Failed to setup workflow:', error);
      process.exit(1);
    });
}
