import { google, sheets_v4 } from 'googleapis';
import { BaseDataProvider } from './BaseDataProvider';
import { DataProviderOptions, DataProviderResponse } from '../interfaces/IExternalDataProvider';
import { IDocumentStore } from '../interfaces/IDocumentStore';
import { IProviderStateStore } from '../interfaces/IProviderStateStore';
import { PriceData, GoogleSheetsConfig } from '../types/PriceData';
import { ProcedureProfileService } from '../ingestion/procedureProfileService';
import { buildFacilityId } from '../ingestion/facilityIds';
import { recordProviderDataMetrics, recordProviderSyncMetrics } from '../observability/metrics';
import { trace, SpanStatusCode } from '@opentelemetry/api';

/**
 * External data provider for Google Sheets
 * Connects to Google Sheets API to retrieve price data
 * 
 * This provider:
 * - Queries price data from specified Google Sheets
 * - Syncs data on a schedule (e.g., every 3 days)
 * - Stores retrieved data in a document store (S3, DynamoDB, MongoDB, etc.)
 */
export class MegalekAteruHelper extends BaseDataProvider<PriceData> {
  private sheetsConfig?: GoogleSheetsConfig;
  private googleSheetsClient?: sheets_v4.Sheets;
  private procedureProfileService?: ProcedureProfileService;
  
  constructor(
    documentStore?: IDocumentStore<PriceData>,
    stateStore?: IProviderStateStore,
    procedureProfileService?: ProcedureProfileService
  ) {
    super('megalek_ateru_helper', documentStore, stateStore);
    this.procedureProfileService = procedureProfileService;
  }
  
  validateConfig(config: Record<string, any>): boolean {
    const sheetsConfig = config as GoogleSheetsConfig;
    
    // Validate required fields
    if (!sheetsConfig.credentials) {
      console.error('Missing credentials in configuration');
      return false;
    }
    
    if (!sheetsConfig.credentials.clientEmail || 
        !sheetsConfig.credentials.privateKey || 
        !sheetsConfig.credentials.projectId) {
      console.error('Missing required credential fields');
      return false;
    }
    
    if (!sheetsConfig.spreadsheetIds || sheetsConfig.spreadsheetIds.length === 0) {
      console.error('No spreadsheet IDs provided');
      return false;
    }
    
    return true;
  }
  
  async initialize(config: Record<string, any>): Promise<void> {
    await super.initialize(config);
    this.sheetsConfig = this.normalizeConfig(config as GoogleSheetsConfig);
    
    const auth = new google.auth.GoogleAuth({
      credentials: {
        client_email: this.sheetsConfig.credentials.clientEmail,
        private_key: this.sheetsConfig.credentials.privateKey,
      },
      projectId: this.sheetsConfig.credentials.projectId,
      scopes: ['https://www.googleapis.com/auth/spreadsheets.readonly'],
    });

    this.googleSheetsClient = google.sheets({ version: 'v4', auth });
    
    console.log(`Initialized ${this.name} with ${this.sheetsConfig.spreadsheetIds.length} spreadsheet(s)`);
  }
  
  async getCurrentData(options?: DataProviderOptions): Promise<DataProviderResponse<PriceData>> {
    this.ensureInitialized();
    
    try {
      const fromStore = await this.getLatestSyncedData(options);
      if (fromStore) {
        return fromStore;
      }
      return await this.loadFromSource(options);
    } catch (error) {
      console.error(`Error fetching current data from ${this.name}:`, error);
      throw error;
    }
  }

  private async loadFromSource(options?: DataProviderOptions): Promise<DataProviderResponse<PriceData>> {
    const tracer = trace.getTracer('patient-price-discovery-provider');
    return tracer.startActiveSpan(
      'provider.load_source',
      {
        attributes: {
          provider: this.name,
          spreadsheet_count: this.sheetsConfig?.spreadsheetIds.length || 0,
        },
      },
      async (span) => {
        try {
          let data = await this.fetchFromGoogleSheets(options);
          if (this.procedureProfileService) {
            data = await this.procedureProfileService.enrichRecords(data, this.name);
          }
          span.setAttribute('records_total', data.length);
          span.setStatus({ code: SpanStatusCode.OK });
          return {
            data,
            timestamp: new Date(),
            metadata: {
              source: this.name,
              count: data.length,
              spreadsheets: this.sheetsConfig?.spreadsheetIds,
            },
          };
        } catch (error) {
          span.recordException(error as Error);
          span.setStatus({
            code: SpanStatusCode.ERROR,
            message: error instanceof Error ? error.message : 'Unknown error',
          });
          throw error;
        } finally {
          span.end();
        }
      }
    );
  }

  private async getLatestSyncedData(options?: DataProviderOptions): Promise<DataProviderResponse<PriceData> | null> {
    if (!this.documentStore) {
      return null;
    }

    const limit = options?.limit ?? 100;
    const offset = options?.offset ?? 0;
    const probe = await this.documentStore.query(
      { source: this.name },
      { limit: 1, offset: 0, sortBy: 'syncTimestamp', sortOrder: 'desc' }
    );
    if (probe.length === 0) {
      return null;
    }

    const latestBatchId = (probe[0] as PriceData).batchId;
    if (!latestBatchId) {
      const data = await this.documentStore.query(
        { source: this.name },
        { limit, offset, sortBy: 'syncTimestamp', sortOrder: 'desc' }
      );
      return {
        data,
        timestamp: new Date(),
        metadata: {
          source: this.name,
          count: data.length,
          total: data.length,
        },
      };
    }

    const data = await this.documentStore.query(
      { source: this.name, batchId: latestBatchId },
      { limit, offset, sortBy: 'effectiveDate', sortOrder: 'desc' }
    );
    const total = (await this.documentStore.query({ source: this.name, batchId: latestBatchId })).length;
    return {
      data,
      timestamp: new Date(),
      metadata: {
        source: this.name,
        count: data.length,
        total,
        batchId: latestBatchId,
      },
    };
  }
  
  async getPreviousData(options?: DataProviderOptions): Promise<DataProviderResponse<PriceData>> {
    this.ensureInitialized();
    
    // Query from document store for previous sync
    if (!this.documentStore) {
      throw new Error('Document store not configured');
    }

    if (!this.previousBatchId) {
      return {
        data: [],
        timestamp: new Date(),
        metadata: {
          source: this.name,
          count: 0,
          type: 'previous',
          message: 'No previous batch available',
        },
      };
    }
    
    try {
      // Query for data from the previous sync
      const previousData = await this.documentStore.query(
        { 
          source: this.name,
          batchId: this.previousBatchId,
        },
        {
          limit: options?.limit || 100,
          offset: options?.offset || 0,
          sortBy: 'effectiveDate',
          sortOrder: 'desc',
        }
      );
      
      return {
        data: previousData,
        timestamp: new Date(),
        metadata: {
          source: this.name,
          count: previousData.length,
          type: 'previous',
        },
      };
    } catch (error) {
      console.error(`Error fetching previous data from ${this.name}:`, error);
      throw error;
    }
  }
  
  async getHistoricalData(options: DataProviderOptions): Promise<DataProviderResponse<PriceData>> {
    this.ensureInitialized();
    
    if (!this.documentStore) {
      throw new Error('Document store not configured');
    }
    
    try {
      let startDate: Date;
      let endDate: Date;
      
      // Parse time window or use provided dates
      if (options.timeWindow) {
        const dateRange = this.parseTimeWindow(options.timeWindow);
        startDate = dateRange.startDate;
        endDate = dateRange.endDate;
      } else if (options.startDate && options.endDate) {
        startDate = options.startDate;
        endDate = options.endDate;
      } else {
        throw new Error('Either timeWindow or startDate/endDate must be provided');
      }
      
      // Query historical data from document store
      const historicalData = await this.documentStore.query(
        {
          source: this.name,
          effectiveDate: {
            $gte: startDate,
            $lte: endDate,
          },
        },
        {
          limit: options.limit || 1000,
          offset: options.offset || 0,
          sortBy: 'effectiveDate',
          sortOrder: 'desc',
        }
      );
      
      return {
        data: historicalData,
        timestamp: new Date(),
        metadata: {
          source: this.name,
          count: historicalData.length,
          type: 'historical',
          dateRange: { startDate, endDate },
        },
      };
    } catch (error) {
      console.error(`Error fetching historical data from ${this.name}:`, error);
      throw error;
    }
  }

  async syncData(): Promise<{
    success: boolean;
    recordsProcessed: number;
    timestamp: Date;
    error?: string;
  }> {
    const tracer = trace.getTracer('patient-price-discovery-provider');
    return tracer.startActiveSpan(
      'provider.sync',
      { attributes: { provider: this.name } },
      async (span) => {
        const startTime = Date.now();
        const timestamp = new Date();
        const batchId = timestamp.toISOString();
        span.setAttribute('batch_id', batchId);

        try {
          const response = await this.loadFromSource();
          const dataWithSync = response.data.map((data) =>
            this.attachSyncMetadata(data, batchId, timestamp)
          );

          recordProviderDataMetrics({ provider: this.name, records: dataWithSync });

          if (this.documentStore && dataWithSync.length > 0) {
            await tracer.startActiveSpan(
              'provider.store_batch',
              {
                attributes: {
                  provider: this.name,
                  batch_id: batchId,
                  records: dataWithSync.length,
                },
              },
              async (storeSpan) => {
                try {
                  const items = dataWithSync.map((data, index) => {
                    const stableKey = this.generateKey(data, index);
                    return {
                      key: `${batchId}_${stableKey}`,
                      data,
                      metadata: {
                        syncTimestamp: timestamp,
                        source: this.name,
                        batchId,
                        stableKey,
                      },
                    };
                  });
                  await this.documentStore!.batchPut(items);
                  storeSpan.setStatus({ code: SpanStatusCode.OK });
                } catch (error) {
                  storeSpan.recordException(error as Error);
                  storeSpan.setStatus({
                    code: SpanStatusCode.ERROR,
                    message: error instanceof Error ? error.message : 'Unknown error',
                  });
                  throw error;
                } finally {
                  storeSpan.end();
                }
              }
            );
          }

          this.previousBatchId = this.lastBatchId;
          this.lastBatchId = batchId;
          this.lastSyncDate = timestamp;
          await this.persistState();
          recordProviderSyncMetrics({
            provider: this.name,
            success: true,
            recordsProcessed: dataWithSync.length,
            durationMs: Date.now() - startTime,
            timestamp,
          });

          span.setAttribute('records_processed', dataWithSync.length);
          span.setStatus({ code: SpanStatusCode.OK });
          return {
            success: true,
            recordsProcessed: dataWithSync.length,
            timestamp,
          };
        } catch (error) {
          recordProviderSyncMetrics({
            provider: this.name,
            success: false,
            recordsProcessed: 0,
            durationMs: Date.now() - startTime,
            timestamp,
          });
          span.recordException(error as Error);
          span.setStatus({
            code: SpanStatusCode.ERROR,
            message: error instanceof Error ? error.message : 'Unknown error',
          });
          return {
            success: false,
            recordsProcessed: 0,
            timestamp,
            error: error instanceof Error ? error.message : 'Unknown error',
          };
        } finally {
          span.end();
        }
      }
    );
  }
  
  /**
   * Fetch data from Google Sheets
   * This is a placeholder implementation
   * In production, this would use the Google Sheets API
   */
  private async fetchFromGoogleSheets(options?: DataProviderOptions): Promise<PriceData[]> {
    if (!this.sheetsConfig) {
      throw new Error('Sheets config not initialized');
    }
    if (!this.googleSheetsClient) {
      throw new Error('Google Sheets client not initialized');
    }
    
    const allData: PriceData[] = [];
    const sheetNames = this.sheetsConfig.sheetNames && this.sheetsConfig.sheetNames.length > 0
      ? this.sheetsConfig.sheetNames
      : ['Sheet1'];

    const tracer = trace.getTracer('patient-price-discovery-provider');
    for (const spreadsheetId of this.sheetsConfig.spreadsheetIds) {
      for (const sheetName of sheetNames) {
        await tracer.startActiveSpan(
          'provider.fetch_sheet',
          {
            attributes: {
              provider: this.name,
              spreadsheet_id: spreadsheetId,
              sheet_name: sheetName,
            },
          },
          async (span) => {
            try {
              const range = `${sheetName}!A:ZZ`;
              const response = await this.googleSheetsClient!.spreadsheets.values.get({
                spreadsheetId,
                range,
              });
              const rows = (response.data.values || []) as string[][];
              span.setAttribute('rows', rows.length);
              const mappedData = this.mapRowsToPriceData(rows);
              span.setAttribute('records', mappedData.length);
              allData.push(...mappedData);
              span.setStatus({ code: SpanStatusCode.OK });
            } catch (error) {
              span.recordException(error as Error);
              span.setStatus({
                code: SpanStatusCode.ERROR,
                message: error instanceof Error ? error.message : 'Unknown error',
              });
              throw error;
            } finally {
              span.end();
            }
          }
        );
      }
    }

    const offset = options?.offset || 0;
    const limit = options?.limit || allData.length;
    return allData.slice(offset, offset + limit);
  }
  
  /**
   * Map spreadsheet rows to PriceData objects
   * Uses column mapping from configuration
   */
  private mapRowsToPriceData(rows: any[][]): PriceData[] {
    if (rows.length === 0) {
      return [];
    }
    
    const columnMapping = this.sheetsConfig?.columnMapping || {};
    const headers = rows[0]; // Assume first row is headers
    const dataRows = rows.slice(1);
    
    return dataRows.map((row, index) => {
      const facilityName = this.getCellValue(row, headers, columnMapping.facilityName || 'Facility Name');
      const facilityId = buildFacilityId(this.name, facilityName);
      const priceData: PriceData = {
        id: `${this.name}_${Date.now()}_${index}`,
        facilityName,
        facilityId: facilityId || undefined,
        procedureCode: this.getCellValue(row, headers, columnMapping.procedureCode || 'Procedure Code'),
        procedureDescription: this.getCellValue(row, headers, columnMapping.procedureDescription || 'Procedure Description'),
        price: parseFloat(this.getCellValue(row, headers, columnMapping.price || 'Price')) || 0,
        currency: 'USD',
        effectiveDate: new Date(this.getCellValue(row, headers, columnMapping.effectiveDate || 'Effective Date')),
        lastUpdated: new Date(),
        source: this.name,
      };
      
      return priceData;
    });
  }
  
  /**
   * Get cell value by column name
   */
  private getCellValue(row: any[], headers: any[], columnName: string): string {
    const columnIndex = headers.indexOf(columnName);
    if (columnIndex === -1) {
      return '';
    }
    return row[columnIndex] || '';
  }

  private normalizeConfig(config: GoogleSheetsConfig): GoogleSheetsConfig {
    const privateKey = config.credentials.privateKey?.replace(/\\n/g, '\n') || '';
    return {
      ...config,
      credentials: {
        ...config.credentials,
        privateKey,
      },
    };
  }
  
  /**
   * Generate unique key for price data
   * Uses stable business identifiers to enable deduplication and upserts
   */
  protected generateKey(data: PriceData, index: number): string {
    const facilityKey = data.facilityId || data.facilityName.replace(/\s+/g, '_').toLowerCase();
    const effectiveDate = data.effectiveDate instanceof Date 
      ? data.effectiveDate.toISOString().split('T')[0]
      : new Date(data.effectiveDate).toISOString().split('T')[0];
    return `${this.name}_${facilityKey}_${data.procedureCode}_${effectiveDate}`;
  }
  
  /**
   * Ensure provider is initialized before operations
   */
  private ensureInitialized(): void {
    if (!this.isInitialized || !this.sheetsConfig) {
      throw new Error(`Provider ${this.name} is not initialized. Call initialize() first.`);
    }
  }
}
