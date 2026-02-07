import fs from 'fs';
import path from 'path';
import { BaseDataProvider } from './BaseDataProvider';
import { DataProviderOptions, DataProviderResponse } from '../interfaces/IExternalDataProvider';
import { IDocumentStore } from '../interfaces/IDocumentStore';
import { PriceData } from '../types/PriceData';
import { parseCsvFile, parseDocxFile, PriceListParseContext } from '../ingestion/priceListParser';
import { applyCuratedTags } from '../ingestion/tagHydration';

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

  constructor(documentStore?: IDocumentStore<PriceData>) {
    super('file_price_list', documentStore);
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
      if (ext === '.csv') {
        allData.push(...parseCsvFile(file.path, context));
      } else if (ext === '.docx') {
        allData.push(...parseDocxFile(file.path, context));
      } else {
        console.warn(`Unsupported file type: ${file.path}`);
      }
    }

    const hydrated = applyCuratedTags(allData);
    const offset = options?.offset || 0;
    const limit = options?.limit || hydrated.length;
    const sliced = hydrated.slice(offset, offset + limit);

    return {
      data: sliced,
      timestamp: new Date(),
      metadata: {
        source: this.name,
        count: sliced.length,
        total: hydrated.length,
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

  private ensureInitialized(): void {
    if (!this.isInitialized || !this.fileConfig) {
      throw new Error(`Provider ${this.name} is not initialized. Call initialize() first.`);
    }
  }
}
