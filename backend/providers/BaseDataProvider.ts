import { IExternalDataProvider, DataProviderOptions, DataProviderResponse } from '../interfaces/IExternalDataProvider';
import { IDocumentStore } from '../interfaces/IDocumentStore';

/**
 * Base abstract class for external data providers
 * Provides common functionality for all providers
 */
export abstract class BaseDataProvider<T = any> implements IExternalDataProvider<T> {
  protected config: Record<string, any> = {};
  protected documentStore?: IDocumentStore<T>;
  protected lastSyncDate?: Date;
  protected isInitialized: boolean = false;
  
  constructor(
    protected name: string,
    documentStore?: IDocumentStore<T>
  ) {
    this.documentStore = documentStore;
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
    const timestamp = new Date();
    
    try {
      // Get current data from external source
      const response = await this.getCurrentData();
      
      // Store in document store if available
      if (this.documentStore && response.data.length > 0) {
        const items = response.data.map((data, index) => ({
          key: this.generateKey(data, index),
          data,
          metadata: {
            syncTimestamp: timestamp,
            source: this.name,
          },
        }));
        
        await this.documentStore.batchPut(items);
      }
      
      this.lastSyncDate = timestamp;
      
      return {
        success: true,
        recordsProcessed: response.data.length,
        timestamp,
      };
    } catch (error) {
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
}
