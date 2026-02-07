import { IExternalDataProvider, DataProviderOptions, DataProviderResponse } from '../interfaces/IExternalDataProvider';
import { IDocumentStore } from '../interfaces/IDocumentStore';
import { IProviderStateStore, ProviderState } from '../interfaces/IProviderStateStore';
import { recordProviderSyncMetrics } from '../observability/metrics';

/**
 * Base abstract class for external data providers
 * Provides common functionality for all providers
 */
export abstract class BaseDataProvider<T = any> implements IExternalDataProvider<T> {
  protected config: Record<string, any> = {};
  protected documentStore?: IDocumentStore<T>;
  protected stateStore?: IProviderStateStore;
  protected lastSyncDate?: Date;
  protected lastBatchId?: string;
  protected previousBatchId?: string;
  protected isInitialized: boolean = false;
  
  constructor(
    protected name: string,
    documentStore?: IDocumentStore<T>,
    stateStore?: IProviderStateStore
  ) {
    this.documentStore = documentStore;
    this.stateStore = stateStore;
  }
  
  getName(): string {
    return this.name;
  }
  
  async initialize(config: Record<string, any>): Promise<void> {
    if (!this.validateConfig(config)) {
      throw new Error(`Invalid configuration for provider: ${this.name}`);
    }
    this.config = config;
    this.isInitialized = true;
    await this.loadState();
  }
  
  abstract validateConfig(config: Record<string, any>): boolean;
  
  abstract getCurrentData(options?: DataProviderOptions): Promise<DataProviderResponse<T>>;
  
  abstract getPreviousData(options?: DataProviderOptions): Promise<DataProviderResponse<T>>;
  
  abstract getHistoricalData(options: DataProviderOptions): Promise<DataProviderResponse<T>>;
  
  async syncData(): Promise<{
    success: boolean;
    recordsProcessed: number;
    timestamp: Date;
    error?: string;
  }> {
    const startTime = Date.now();
    const timestamp = new Date();
    const batchId = timestamp.toISOString();
    
    try {
      // Get current data from external source
      const response = await this.getCurrentData();
      const dataWithSync = response.data.map((data) =>
        this.attachSyncMetadata(data, batchId, timestamp)
      );
      
      // Store in document store if available
      if (this.documentStore && dataWithSync.length > 0) {
        const items = dataWithSync.map((data, index) => ({
          key: this.generateKey(data, index),
          data,
          metadata: {
            syncTimestamp: timestamp,
            source: this.name,
            batchId,
          },
        }));
        
        await this.documentStore.batchPut(items);
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
      });
      
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
      });
      return {
        success: false,
        recordsProcessed: 0,
        timestamp,
        error: error instanceof Error ? error.message : 'Unknown error',
      };
    }
  }
  
  async getHealthStatus(): Promise<{
    healthy: boolean;
    lastSync?: Date;
    message?: string;
  }> {
    if (!this.isInitialized) {
      return {
        healthy: false,
        message: 'Provider not initialized',
      };
    }
    
    return {
      healthy: true,
      lastSync: this.lastSyncDate,
      message: 'Provider is operational',
    };
  }
  
  /**
   * Generate a unique key for storing data
   * Override this method for custom key generation
   */
  protected generateKey(data: T, index: number): string {
    return `${this.name}_${Date.now()}_${index}`;
  }

  protected async loadState(): Promise<void> {
    if (!this.stateStore) {
      return;
    }
    const state = await this.stateStore.getState(this.name);
    if (!state) {
      return;
    }
    this.applyState(state);
  }

  protected async persistState(): Promise<void> {
    if (!this.stateStore) {
      return;
    }
    const state: ProviderState = {
      lastSyncDate: this.lastSyncDate ? this.lastSyncDate.toISOString() : undefined,
      lastBatchId: this.lastBatchId,
      previousBatchId: this.previousBatchId,
    };
    await this.stateStore.saveState(this.name, state);
  }

  protected applyState(state: ProviderState): void {
    if (state.lastSyncDate) {
      const parsed = new Date(state.lastSyncDate);
      if (!Number.isNaN(parsed.getTime())) {
        this.lastSyncDate = parsed;
      }
    }
    if (state.lastBatchId) {
      this.lastBatchId = state.lastBatchId;
    }
    if (state.previousBatchId) {
      this.previousBatchId = state.previousBatchId;
    }
  }
  
  /**
   * Parse time window string to date range
   * Examples: "30d" = 30 days, "7d" = 7 days, "1y" = 1 year
   */
  protected parseTimeWindow(timeWindow: string): { startDate: Date; endDate: Date } {
    const now = new Date();
    const endDate = new Date(now);
    const startDate = new Date(now);
    
    const match = timeWindow.match(/^(\d+)([dmy])$/);
    if (!match) {
      throw new Error(`Invalid time window format: ${timeWindow}`);
    }
    
    const [, amount, unit] = match;
    const amountNum = parseInt(amount, 10);
    
    switch (unit) {
      case 'd':
        startDate.setDate(startDate.getDate() - amountNum);
        break;
      case 'm':
        startDate.setMonth(startDate.getMonth() - amountNum);
        break;
      case 'y':
        startDate.setFullYear(startDate.getFullYear() - amountNum);
        break;
    }
    
    return { startDate, endDate };
  }

  /**
   * Attach sync metadata to a record for storage/querying.
   * For non-object data types, returns the original value.
   */
  protected attachSyncMetadata(data: T, batchId: string, timestamp: Date): T {
    if (!data || typeof data !== 'object') {
      return data;
    }

    const record = data as Record<string, any>;
    return {
      ...record,
      batchId,
      syncTimestamp: timestamp.toISOString(),
      source: record.source ?? this.name,
      metadata: {
        ...(record.metadata ?? {}),
        batchId,
        syncTimestamp: timestamp.toISOString(),
        source: this.name,
      },
    } as T;
  }
}
