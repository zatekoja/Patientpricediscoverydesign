import { Collection, MongoClient, MongoClientOptions } from 'mongodb';
import { IDocumentStore } from '../interfaces/IDocumentStore';

type StoredDocument<T> = {
  _id: string;
  data: T;
  metadata?: Record<string, any>;
  createdAt: Date;
  updatedAt: Date;
};

export class MongoDocumentStore<T = any> implements IDocumentStore<T> {
  private client: MongoClient;
  private collectionName: string;
  private dbName: string;
  private isConnected: boolean = false;
  private indexesReady: boolean = false;
  private ttlDays?: number;

  constructor(
    uri: string,
    dbName: string,
    collectionName: string = 'provider_documents',
    config?: { clientOptions?: MongoClientOptions; ttlDays?: number }
  ) {
    this.client = new MongoClient(uri, config?.clientOptions);
    this.dbName = dbName;
    this.collectionName = collectionName;
    this.ttlDays = config?.ttlDays;
  }

  async put(key: string, data: T, metadata?: Record<string, any>): Promise<void> {
    const collection = await this.getCollection();
    const now = new Date();
    await collection.updateOne(
      { _id: key },
      {
        $set: {
          data,
          metadata,
          updatedAt: now,
        },
        $setOnInsert: {
          createdAt: now,
        },
      },
      { upsert: true }
    );
  }

  async get(key: string): Promise<T | null> {
    const collection = await this.getCollection();
    const doc = await collection.findOne({ _id: key });
    return doc ? doc.data : null;
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
    const collection = await this.getCollection();
    const mongoFilter = this.mapFilter(filter);
    let cursor = collection.find(mongoFilter);

    if (options?.sortBy) {
      const sortField = `data.${options.sortBy}`;
      const sortOrder = options.sortOrder === 'desc' ? -1 : 1;
      cursor = cursor.sort({ [sortField]: sortOrder });
    }

    if (options?.offset) {
      cursor = cursor.skip(options.offset);
    }
    if (options?.limit) {
      cursor = cursor.limit(options.limit);
    }

    const docs = await cursor.toArray();
    return docs.map((doc) => doc.data);
  }

  async delete(key: string): Promise<void> {
    const collection = await this.getCollection();
    await collection.deleteOne({ _id: key });
  }

  async exists(key: string): Promise<boolean> {
    const collection = await this.getCollection();
    const count = await collection.countDocuments({ _id: key }, { limit: 1 });
    return count > 0;
  }

  async batchPut(items: Array<{ key: string; data: T; metadata?: Record<string, any> }>): Promise<void> {
    if (items.length === 0) {
      return;
    }
    const collection = await this.getCollection();
    const now = new Date();
    const operations = items.map((item) => ({
      updateOne: {
        filter: { _id: item.key },
        update: {
          $set: {
            data: item.data,
            metadata: item.metadata,
            updatedAt: now,
          },
          $setOnInsert: {
            createdAt: now,
          },
        },
        upsert: true,
      },
    }));
    await collection.bulkWrite(operations, { ordered: false });
  }

  getStoreName(): string {
    return `mongodb:${this.dbName}/${this.collectionName}`;
  }

  private async getCollection(): Promise<Collection<StoredDocument<T>>> {
    if (!this.isConnected) {
      await this.client.connect();
      this.isConnected = true;
    }
    const collection = this.client.db(this.dbName).collection<StoredDocument<T>>(this.collectionName);
    await this.ensureIndexes(collection);
    return collection;
  }

  private async ensureIndexes(collection: Collection<StoredDocument<T>>): Promise<void> {
    if (this.indexesReady) {
      return;
    }
    await collection.createIndex({ 'data.source': 1 });
    await collection.createIndex({ 'data.batchId': 1 });
    await collection.createIndex({ 'data.effectiveDate': -1 });
    await collection.createIndex({ 'data.syncTimestamp': -1 });
    if (this.ttlDays && this.ttlDays > 0) {
      await collection.createIndex(
        { updatedAt: 1 },
        { expireAfterSeconds: this.ttlDays * 24 * 60 * 60 }
      );
    } else {
      await collection.createIndex({ updatedAt: -1 });
    }
    this.indexesReady = true;
  }

  private mapFilter(filter: Record<string, any>): Record<string, any> {
    const mapped: Record<string, any> = {};
    for (const [key, value] of Object.entries(filter)) {
      if (key.startsWith('data.')) {
        mapped[key] = value;
      } else {
        mapped[`data.${key}`] = value;
      }
    }
    return mapped;
  }
}
