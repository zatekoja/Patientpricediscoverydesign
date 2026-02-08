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
import { MongoDocumentStore } from '../stores/MongoDocumentStore';
import { MongoProviderStateStore } from '../stores/MongoProviderStateStore';
import { LLMDocumentParser, DocumentSummaryCacheRecord } from '../ingestion/llmDocumentParser';
import { DataSyncScheduler, SyncIntervals } from '../config/DataSyncScheduler';
import { PriceData, GoogleSheetsConfig } from '../types/PriceData';
import { FacilityProfile } from '../types/FacilityProfile';
import { ProcedureProfile } from '../types/ProcedureProfile';
import { IProviderStateStore } from '../interfaces/IProviderStateStore';
import { IDocumentStore } from '../interfaces/IDocumentStore';
import { FacilityProfileService } from '../ingestion/facilityProfileService';
import { ProcedureProfileService } from '../ingestion/procedureProfileService';
import { loadVaultSecretsToEnv } from '../config/vaultSecrets';
import {
  CapacityRequestService,
  SesEmailSender,
  WhatsAppCloudSender,
  CapacityRequestToken,
} from '../ingestion/capacityRequestService';

/**
 * Initialize and start the API server
 */
async function startServer() {
  const vaultPath = process.env.VAULT_PROVIDER_PATH || process.env.VAULT_PATH;
  const vaultResult = await loadVaultSecretsToEnv({ path: vaultPath });
  if (vaultResult.enabled) {
    if (vaultResult.error) {
      console.warn(`⚠ Vault secrets not loaded: ${vaultResult.error}`);
    } else {
      console.log(`✓ Vault secrets loaded (${vaultResult.loaded} applied, ${vaultResult.skipped} skipped)`);
    }
  }

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

  // 2. Create document store
  const mongoURI = process.env.PROVIDER_MONGO_URI;
  const mongoDB = process.env.PROVIDER_MONGO_DB || 'provider_data';
  const mongoCollection = process.env.PROVIDER_MONGO_COLLECTION || 'price_records';
  const mongoDocSummaryCollection =
    process.env.PROVIDER_MONGO_DOC_SUMMARY_COLLECTION || 'document_summaries';
  const mongoStateCollection = process.env.PROVIDER_STATE_COLLECTION || 'provider_state';
  const mongoFacilityCollection = process.env.PROVIDER_FACILITY_COLLECTION || 'facility_profiles';
  const mongoCapacityCollection = process.env.PROVIDER_CAPACITY_TOKEN_COLLECTION || 'capacity_request_tokens';
  const mongoProcedureCollection = process.env.PROVIDER_PROCEDURE_COLLECTION || 'procedure_profiles';
  const mongoTTL = process.env.PROVIDER_MONGO_TTL_DAYS
    ? Number.parseInt(process.env.PROVIDER_MONGO_TTL_DAYS, 10)
    : 30;

  let documentStore: IDocumentStore<PriceData>;
  let docSummaryStore: IDocumentStore<DocumentSummaryCacheRecord>;
  let stateStore: IProviderStateStore | undefined;

  if (mongoURI) {
    documentStore = new MongoDocumentStore<PriceData>(mongoURI, mongoDB, mongoCollection, {
      ttlDays: Number.isFinite(mongoTTL) ? mongoTTL : 30,
    });
    docSummaryStore = new MongoDocumentStore<DocumentSummaryCacheRecord>(
      mongoURI,
      mongoDB,
      mongoDocSummaryCollection,
      { ttlDays: Number.isFinite(mongoTTL) ? mongoTTL : 30 }
    );
    stateStore = new MongoProviderStateStore(mongoURI, mongoDB, mongoStateCollection);
    console.log(`✓ Mongo provider store enabled (${mongoDB}.${mongoCollection})`);
  } else {
    documentStore = new InMemoryDocumentStore<PriceData>('price-data-store');
    docSummaryStore = new InMemoryDocumentStore<DocumentSummaryCacheRecord>('doc-summary-store');
    console.warn('⚠ Using in-memory provider store. Set PROVIDER_MONGO_URI for persistence.');
  }

  let facilityStore: IDocumentStore<FacilityProfile>;
  if (mongoURI) {
    facilityStore = new MongoDocumentStore<FacilityProfile>(mongoURI, mongoDB, mongoFacilityCollection);
    console.log(`✓ Facility profile store enabled (${mongoDB}.${mongoFacilityCollection})`);
  } else {
    facilityStore = new InMemoryDocumentStore<FacilityProfile>('facility-profiles');
  }

  const llmConfig = {
    apiEndpoint: process.env.PROVIDER_LLM_API_ENDPOINT,
    apiKey: process.env.PROVIDER_LLM_API_KEY,
    model: process.env.PROVIDER_LLM_MODEL,
    systemPrompt: process.env.PROVIDER_LLM_SYSTEM_PROMPT,
    temperature: process.env.PROVIDER_LLM_TEMPERATURE
      ? Number.parseFloat(process.env.PROVIDER_LLM_TEMPERATURE)
      : undefined,
    maxTags: process.env.PROVIDER_LLM_MAX_TAGS
      ? Number.parseInt(process.env.PROVIDER_LLM_MAX_TAGS, 10)
      : undefined,
  };
  const facilityProfileService = new FacilityProfileService(facilityStore, llmConfig);

  let procedureStore: IDocumentStore<ProcedureProfile>;
  if (mongoURI) {
    procedureStore = new MongoDocumentStore<ProcedureProfile>(mongoURI, mongoDB, mongoProcedureCollection);
    console.log(`✓ Procedure profile store enabled (${mongoDB}.${mongoProcedureCollection})`);
  } else {
    procedureStore = new InMemoryDocumentStore<ProcedureProfile>('procedure-profiles');
  }
  const procedureProfileService = new ProcedureProfileService(procedureStore, llmConfig);

  let capacityTokenStore: IDocumentStore<CapacityRequestToken>;
  if (mongoURI) {
    const ttlMinutes = process.env.PROVIDER_CAPACITY_TOKEN_TTL_MINUTES
      ? Number.parseInt(process.env.PROVIDER_CAPACITY_TOKEN_TTL_MINUTES, 10)
      : 120;
    const ttlDays = Math.max(1, Math.ceil(ttlMinutes / (60 * 24)));
    capacityTokenStore = new MongoDocumentStore<CapacityRequestToken>(mongoURI, mongoDB, mongoCapacityCollection, {
      ttlDays,
    });
  } else {
    capacityTokenStore = new InMemoryDocumentStore<CapacityRequestToken>('capacity-tokens');
  }

  const publicBaseUrl = process.env.PROVIDER_PUBLIC_BASE_URL || `http://localhost:${process.env.PORT || 3000}`;
  const sesRegion = process.env.AWS_REGION || 'us-east-1';
  const sesFrom = process.env.SES_FROM_ADDRESS || '';
  const emailSender = sesFrom ? new SesEmailSender(sesRegion, sesFrom) : undefined;

  const whatsappAccessToken = process.env.WHATSAPP_ACCESS_TOKEN || '';
  const whatsappPhoneId = process.env.WHATSAPP_PHONE_NUMBER_ID || '';
  const whatsappTemplate = process.env.WHATSAPP_TEMPLATE_NAME;
  const whatsappLang = process.env.WHATSAPP_TEMPLATE_LANG || 'en_US';
  const whatsappSender =
    whatsappAccessToken && whatsappPhoneId
      ? new WhatsAppCloudSender(whatsappAccessToken, whatsappPhoneId, whatsappTemplate, whatsappLang)
      : undefined;

  // Custom email template support
  const customEmailTemplate = process.env.PROVIDER_CAPACITY_EMAIL_TEMPLATE;
  const emailTemplate = customEmailTemplate
    ? (facilityName: string, link: string) => {
        // Replace placeholders in template
        return customEmailTemplate
          .replace(/\{facilityName\}/g, facilityName)
          .replace(/\{link\}/g, link);
      }
    : undefined;

  const capacityRequestService = new CapacityRequestService({
    facilityProfileService,
    tokenStore: capacityTokenStore,
    publicBaseUrl,
    tokenTTLMinutes: process.env.PROVIDER_CAPACITY_TOKEN_TTL_MINUTES
      ? Number.parseInt(process.env.PROVIDER_CAPACITY_TOKEN_TTL_MINUTES, 10)
      : 120,
    emailSender,
    whatsappSender,
    emailTemplate,
  });

  // 1. Create the API server
  const api = new DataProviderAPI({ facilityProfileService, capacityRequestService });

  // 3. Create and configure provider
  let providerInitialized = false;
  let scheduler: DataSyncScheduler | undefined;
  let providerId = 'megalek_ateru_helper';
  let provider: MegalekAteruHelper | FilePriceListProvider = new MegalekAteruHelper(
    documentStore,
    stateStore,
    procedureProfileService
  );

  if (useFileProvider) {
    providerId = 'file_price_list';
    const llmParserEnabled = process.env.LLM_DOC_PARSER_ENABLED === 'true';
    const llmApiKey =
      process.env.OPENAI_API_KEY || process.env.LLM_API_KEY || process.env.PROVIDER_LLM_API_KEY || '';
    const llmConfig = {
      enabled: llmParserEnabled,
      apiKey: llmApiKey,
      apiEndpoint: process.env.LLM_API_ENDPOINT || 'https://api.openai.com/v1/chat/completions',
      model: process.env.LLM_DOC_PARSER_MODEL || 'gpt-4o-mini',
      temperature: process.env.LLM_DOC_PARSER_TEMPERATURE
        ? Number.parseFloat(process.env.LLM_DOC_PARSER_TEMPERATURE)
        : 0.2,
      maxRows: process.env.LLM_DOC_PARSER_MAX_ROWS
        ? Number.parseInt(process.env.LLM_DOC_PARSER_MAX_ROWS, 10)
        : 5000,
      maxChars: process.env.LLM_DOC_PARSER_MAX_CHARS
        ? Number.parseInt(process.env.LLM_DOC_PARSER_MAX_CHARS, 10)
        : 50_000,
      maxBytes: process.env.LLM_DOC_PARSER_MAX_BYTES
        ? Number.parseInt(process.env.LLM_DOC_PARSER_MAX_BYTES, 10)
        : 5 * 1024 * 1024,
    };
    const llmDocumentParser = new LLMDocumentParser(llmConfig, docSummaryStore);
    provider = new FilePriceListProvider(documentStore, stateStore, procedureProfileService, llmDocumentParser);
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
    explicitFacilityMapping: (() => {
      const defaults: Record<string, string> = {
        'MEGALEK NEW PRICE LIST 2026.csv': 'Ijede General Hospital',
        'NEW LASUTH PRICE LIST (SERVICES).csv': 'Lagos State University Teaching Hospital (LASUTH)',
        'PRICE LIST FOR RANDLE GENERAL HOSPITAL JANUARY 2026.csv': 'Randle General Hospital',
      };
      const raw = process.env.PRICE_LIST_FACILITY_MAPPING;
      if (!raw) {
        return defaults;
      }
      try {
        const parsed = JSON.parse(raw) as Record<string, string>;
        return { ...defaults, ...parsed };
      } catch (error) {
        console.warn('Failed to parse PRICE_LIST_FACILITY_MAPPING JSON; using defaults only');
        return defaults;
      }
    })(),
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
    const runInitialSync = process.env.PROVIDER_RUN_INITIAL_SYNC
      ? process.env.PROVIDER_RUN_INITIAL_SYNC.toLowerCase() === 'true'
      : useFileProvider;

    scheduler.scheduleJob({
      name: `${providerId}_sync`,
      provider,
      intervalMs: SyncIntervals.THREE_DAYS,
      runImmediately: runInitialSync,
      onComplete: (result) => {
        console.log(`Scheduled sync completed: ${result.recordsProcessed} records, success: ${result.success}`);
        if (result.success) {
          facilityProfileService
            .ensureProfilesFromProvider(provider, { providerId })
            .catch((error) => console.error('Facility enrichment failed:', error.message));
        }
      },
      onError: (error) => {
        console.error('Scheduled sync error:', error.message);
      },
    });
    console.log('✓ Scheduled sync configured (every 3 days)');
  } else {
    console.warn('⚠ Skipping provider registration and sync scheduler due to initialization failure');
  }

  const capacityIntervalMinutes = process.env.PROVIDER_CAPACITY_REQUEST_INTERVAL_MINUTES
    ? Number.parseInt(process.env.PROVIDER_CAPACITY_REQUEST_INTERVAL_MINUTES, 10)
    : 45;
  if (capacityIntervalMinutes > 0) {
    capacityRequestService.start(capacityIntervalMinutes * 60 * 1000);
    console.log(`✓ Capacity request scheduler enabled (every ${capacityIntervalMinutes} minutes)`);
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
    capacityRequestService.stop();
    process.exit(0);
  });

  process.on('SIGINT', () => {
    console.log('Received SIGINT, shutting down gracefully...');
    if (scheduler) {
      scheduler.stopAll();
    }
    capacityRequestService.stop();
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
