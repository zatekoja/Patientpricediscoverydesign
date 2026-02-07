/**
 * Configuration options for data provider queries
 */
export interface DataProviderOptions {
  /** Time window for data retrieval (e.g., "30d", "7d", "1y") */
  timeWindow?: string;
  
  /** Start date for historical data queries */
  startDate?: Date;
  
  /** End date for historical data queries */
  endDate?: Date;
  
  /** Specific parameters for the provider */
  parameters?: Record<string, any>;
  
  /** Maximum number of records to retrieve */
  limit?: number;
  
  /** Offset for pagination */
  offset?: number;
}

/**
 * Generic data response from external providers
 */
export interface DataProviderResponse<T = any> {
  /** The actual data payload */
  data: T[];
  
  /** Timestamp when data was retrieved */
  timestamp: Date;
  
  /** Metadata about the response */
  metadata?: {
    source?: string;
    count?: number;
    hasMore?: boolean;
    [key: string]: any;
  };
}

/**
 * Interface for all external data providers
 * External providers must implement this interface to connect to our system
 */
export interface IExternalDataProvider<T = any> {
  /**
   * Get the provider name
   */
  getName(): string;
  
  /**
   * Initialize the provider with configuration
   */
  initialize(config: Record<string, any>): Promise<void>;
  
  /**
   * Get current/latest data from the provider
   * @param options - Configurable options for the query
   * @returns Current data response
   */
  getCurrentData(options?: DataProviderOptions): Promise<DataProviderResponse<T>>;
  
  /**
   * Get previous data (the last batch before current)
   * @param options - Configurable options for the query
   * @returns Previous data response
   */
  getPreviousData(options?: DataProviderOptions): Promise<DataProviderResponse<T>>;
  
  /**
   * Get historical data within a time range
   * @param options - Configurable options including time window or date range
   * @returns Historical data response
   */
  getHistoricalData(options: DataProviderOptions): Promise<DataProviderResponse<T>>;
  
  /**
   * Sync data from external source to internal data store
   * This is typically called by a scheduled job
   * @returns Success status and metadata
   */
  syncData(): Promise<{
    success: boolean;
    recordsProcessed: number;
    timestamp: Date;
    error?: string;
  }>;
  
  /**
   * Validate the provider configuration
   */
  validateConfig(config: Record<string, any>): boolean;
  
  /**
   * Get provider health status
   */
  getHealthStatus(): Promise<{
    healthy: boolean;
    lastSync?: Date;
    message?: string;
  }>;
}
