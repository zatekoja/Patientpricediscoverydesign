/**
 * Interface for document store operations
 * This abstraction allows switching between S3, DynamoDB, MongoDB, etc.
 */
export interface IDocumentStore<T = any> {
  /**
   * Store a document
   * @param key - Unique identifier for the document
   * @param data - Data to store
   * @param metadata - Optional metadata
   */
  put(key: string, data: T, metadata?: Record<string, any>): Promise<void>;
  
  /**
   * Retrieve a document
   * @param key - Unique identifier for the document
   */
  get(key: string): Promise<T | null>;
  
  /**
   * Query documents based on filters
   * @param filter - Query filter
   * @param options - Query options (limit, offset, etc.)
   */
  query(filter: Record<string, any>, options?: {
    limit?: number;
    offset?: number;
    sortBy?: string;
    sortOrder?: 'asc' | 'desc';
  }): Promise<T[]>;
  
  /**
   * Delete a document
   * @param key - Unique identifier for the document
   */
  delete(key: string): Promise<void>;
  
  /**
   * Check if document exists
   * @param key - Unique identifier for the document
   */
  exists(key: string): Promise<boolean>;
  
  /**
   * Batch put multiple documents
   * @param items - Array of key-data pairs
   */
  batchPut(items: Array<{ key: string; data: T; metadata?: Record<string, any> }>): Promise<void>;
  
  /**
   * Get the store name/type
   */
  getStoreName(): string;
}
