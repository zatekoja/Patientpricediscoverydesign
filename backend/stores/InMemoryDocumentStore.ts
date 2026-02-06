import { IDocumentStore } from '../interfaces/IDocumentStore';

/**
 * In-memory document store implementation
 * For production, replace with actual store implementations (S3, DynamoDB, MongoDB)
 */
export class InMemoryDocumentStore<T = any> implements IDocumentStore<T> {
  private store: Map<string, { data: T; metadata?: Record<string, any> }> = new Map();
  private name: string;
  
  constructor(name: string = 'in-memory-store') {
    this.name = name;
  }
  
  async put(key: string, data: T, metadata?: Record<string, any>): Promise<void> {
    this.store.set(key, { data, metadata });
  }
  
  async get(key: string): Promise<T | null> {
    const item = this.store.get(key);
    return item ? item.data : null;
  }
  
  async query(
    filter: Record<string, any>,
    options?: {
      limit?: number;
      offset?: number;
      sortBy?: string;
      sortOrder?: 'asc' | 'desc';
    }
  ): Promise<T[]> {
    // Simple filtering implementation
    let results = Array.from(this.store.values())
      .map(item => item.data)
      .filter(data => this.matchesFilter(data, filter));
    
    // Sort if specified
    if (options?.sortBy) {
      results.sort((a: any, b: any) => {
        const aVal = a[options.sortBy!];
        const bVal = b[options.sortBy!];
        const comparison = aVal < bVal ? -1 : aVal > bVal ? 1 : 0;
        return options.sortOrder === 'desc' ? -comparison : comparison;
      });
    }
    
    // Apply pagination
    const offset = options?.offset || 0;
    const limit = options?.limit || results.length;
    
    return results.slice(offset, offset + limit);
  }
  
  async delete(key: string): Promise<void> {
    this.store.delete(key);
  }
  
  async exists(key: string): Promise<boolean> {
    return this.store.has(key);
  }
  
  async batchPut(items: Array<{ key: string; data: T; metadata?: Record<string, any> }>): Promise<void> {
    for (const item of items) {
      await this.put(item.key, item.data, item.metadata);
    }
  }
  
  getStoreName(): string {
    return this.name;
  }
  
  /**
   * Simple filter matching
   * For production, implement more sophisticated query logic
   */
  private matchesFilter(data: any, filter: Record<string, any>): boolean {
    for (const [key, value] of Object.entries(filter)) {
      if (typeof value === 'object' && value !== null && !Array.isArray(value)) {
        // Handle operators like $gte, $lte
        if ('$gte' in value && data[key] < value.$gte) {
          return false;
        }
        if ('$lte' in value && data[key] > value.$lte) {
          return false;
        }
        if ('$gt' in value && data[key] <= value.$gt) {
          return false;
        }
        if ('$lt' in value && data[key] >= value.$lt) {
          return false;
        }
      } else {
        // Simple equality check
        if (data[key] !== value) {
          return false;
        }
      }
    }
    return true;
  }
  
  /**
   * Get all items (for debugging/testing)
   */
  async getAll(): Promise<T[]> {
    return Array.from(this.store.values()).map(item => item.data);
  }
  
  /**
   * Clear all data (for testing)
   */
  async clear(): Promise<void> {
    this.store.clear();
  }
  
  /**
   * Get store size
   */
  size(): number {
    return this.store.size;
  }
}
