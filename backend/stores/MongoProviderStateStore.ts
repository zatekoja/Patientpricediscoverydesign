import { Collection, MongoClient, MongoClientOptions } from 'mongodb';
import { IProviderStateStore, ProviderState } from '../interfaces/IProviderStateStore';

type ProviderStateDocument = ProviderState & {
  _id: string;
  updatedAt: Date;
};

export class MongoProviderStateStore implements IProviderStateStore {
  private client: MongoClient;
  private collectionName: string;
  private dbName: string;
  private isConnected: boolean = false;

  constructor(uri: string, dbName: string, collectionName: string = 'provider_state', options?: MongoClientOptions) {
    this.client = new MongoClient(uri, options);
    this.dbName = dbName;
    this.collectionName = collectionName;
  }

  async getState(providerName: string): Promise<ProviderState | null> {
    const collection = await this.getCollection();
    const doc = await collection.findOne({ _id: providerName });
    if (!doc) {
      return null;
    }
    const { lastSyncDate, lastBatchId, previousBatchId } = doc;
    return { lastSyncDate, lastBatchId, previousBatchId };
  }

  async saveState(providerName: string, state: ProviderState): Promise<void> {
    const collection = await this.getCollection();
    await collection.updateOne(
      { _id: providerName },
      {
        $set: {
          ...state,
          updatedAt: new Date(),
        },
      },
      { upsert: true }
    );
  }

  private async getCollection(): Promise<Collection<ProviderStateDocument>> {
    if (!this.isConnected) {
      await this.client.connect();
      this.isConnected = true;
    }
    return this.client.db(this.dbName).collection<ProviderStateDocument>(this.collectionName);
  }
}
