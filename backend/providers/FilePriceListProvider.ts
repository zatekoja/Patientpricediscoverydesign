import fs from 'fs';
import path from 'path';
import { BaseDataProvider } from './BaseDataProvider';
import { DataProviderOptions, DataProviderResponse } from '../interfaces/IExternalDataProvider';
import { IDocumentStore } from '../interfaces/IDocumentStore';
import { IProviderStateStore } from '../interfaces/IProviderStateStore';
import { PriceData } from '../types/PriceData';
import { parseCsvFile, parseDocxFile, PriceListParseContext } from '../ingestion/priceListParser';
import { applyCuratedTags } from '../ingestion/tagHydration';
import { recordProviderDataMetrics, recordProviderSyncMetrics } from '../observability/metrics';
import { trace, SpanStatusCode } from '@opentelemetry/api';

export interface FilePriceListConfig {
  files: Array<{
    path: string;
    facilityName?: string;
  }>;
  currency?: string;
  defaultEffectiveDate?: string;
}

export class FilePriceListProvider extends BaseDataProvider<PriceData> {
  private fileConfig?: FilePriceListConfig;

  constructor(documentStore?: IDocumentStore<PriceData>, stateStore?: IProviderStateStore) {
    super('file_price_list', documentStore, stateStore);
  }

  validateConfig(config: Record<string, any>): boolean {
    const parsed = config as FilePriceListConfig;
    if (!parsed.files || parsed.files.length === 0) {
      console.error('No files configured for FilePriceListProvider');
      return false;
    }
    for (const file of parsed.files) {
      if (!file.path) {
        console.error('File entry missing path');
        return false;
      }
    }
    return true;
  }

  async initialize(config: Record<string, any>): Promise<void> {
    await super.initialize(config);
    this.fileConfig = config as FilePriceListConfig;
  }

  async getCurrentData(options?: DataProviderOptions): Promise<DataProviderResponse<PriceData>> {
    this.ensureInitialized();

    const fromStore = await this.getLatestSyncedData(options);
    if (fromStore) {
      return fromStore;
    }

    return this.loadFromSource(options);
  }

  private async loadFromSource(options?: DataProviderOptions): Promise<DataProviderResponse<PriceData>> {
    const tracer = trace.getTracer('patient-price-discovery-provider');
    return tracer.startActiveSpan(
      'provider.load_source',
      {
        attributes: {
          provider: this.name,
          file_count: this.fileConfig?.files.length || 0,
        },
      },
      async (span) => {
        try {
          const allData: PriceData[] = [];
          const currency = this.fileConfig?.currency || 'NGN';
          const defaultEffectiveDate = this.fileConfig?.defaultEffectiveDate
            ? new Date(this.fileConfig.defaultEffectiveDate)
            : undefined;

          for (const file of this.fileConfig!.files) {
            if (!fs.existsSync(file.path)) {
              console.warn(`File not found: ${file.path}`);
              continue;
            }

            const context: PriceListParseContext = {
              facilityName: file.facilityName,
              currency,
              defaultEffectiveDate,
              sourceFile: path.basename(file.path),
            };

            const ext = path.extname(file.path).toLowerCase();
            const fileName = path.basename(file.path);
            const parsed = await tracer.startActiveSpan(
              'provider.parse_file',
              {
                attributes: {
                  provider: this.name,
                  file: fileName,
                  extension: ext,
                },
              },
              (fileSpan) => {
                try {
                  if (ext === '.csv') {
                    const rows = parseCsvFile(file.path, context);
                    fileSpan.setAttribute('records', rows.length);
                    return rows;
                  }
                  if (ext === '.docx') {
                    const rows = parseDocxFile(file.path, context);
                    fileSpan.setAttribute('records', rows.length);
                    return rows;
                  }
                  console.warn(`Unsupported file type: ${file.path}`);
                  return [];
                } finally {
                  fileSpan.end();
                }
              }
            );
            allData.push(...parsed);
          }

          const hydrated = applyCuratedTags(allData);
          const offset = options?.offset || 0;
          const limit = options?.limit || hydrated.length;
          const sliced = hydrated.slice(offset, offset + limit);
          span.setAttribute('records_total', hydrated.length);

          span.setStatus({ code: SpanStatusCode.OK });
          return {
            data: sliced,
            timestamp: new Date(),
            metadata: {
              source: this.name,
              count: sliced.length,
              total: hydrated.length,
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

    const previousData = await this.documentStore.query(
      { source: this.name, batchId: this.previousBatchId },
      {
        limit: options?.limit || 100,
        sortBy: 'effectiveDate',
        sortOrder: 'desc',
        offset: options?.offset || 0,
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
  }

  async getHistoricalData(options: DataProviderOptions): Promise<DataProviderResponse<PriceData>> {
    this.ensureInitialized();

    if (!this.documentStore) {
      throw new Error('Document store not configured');
    }

    let startDate: Date;
    let endDate: Date;

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

  private ensureInitialized(): void {
    if (!this.isInitialized || !this.fileConfig) {
      throw new Error(`Provider ${this.name} is not initialized. Call initialize() first.`);
    }
  }

  protected generateKey(data: PriceData, index: number): string {
    const facilityKey = (data.facilityId || data.facilityName || 'unknown')
      .toString()
      .trim()
      .toLowerCase()
      .replace(/[^a-z0-9]+/g, '_');
    const procedureKey = (data.procedureCode || data.procedureDescription || `item_${index}`)
      .toString()
      .trim()
      .toLowerCase()
      .replace(/[^a-z0-9]+/g, '_');
    const effectiveDate = data.effectiveDate instanceof Date
      ? data.effectiveDate.toISOString().split('T')[0]
      : new Date(data.effectiveDate).toISOString().split('T')[0];
    const tier = typeof data.metadata?.priceTier === 'string'
      ? data.metadata?.priceTier.toString().trim().toLowerCase().replace(/[^a-z0-9]+/g, '_')
      : 'base';
    const priceKey = Number.isFinite(data.price) ? data.price.toFixed(2) : '0';
    return `${this.name}_${facilityKey}_${procedureKey}_${tier}_${priceKey}_${effectiveDate}`;
  }
}
