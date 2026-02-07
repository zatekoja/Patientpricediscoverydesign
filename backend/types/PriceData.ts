/**
 * Price data structure from external providers
 */
export interface PriceData {
  /** Unique identifier for the price record */
  id: string;
  
  /** Facility or provider name */
  facilityName: string;
  
  /** Facility identifier */
  facilityId?: string;
  
  /** Procedure or service code */
  procedureCode: string;
  
  /** Procedure description */
  procedureDescription: string;

  /** Procedure category (enriched) */
  procedureCategory?: string;

  /** Detailed procedure description (enriched) */
  procedureDetails?: string;
  
  /** Price amount */
  price: number;
  
  /** Currency code (e.g., "USD") */
  currency: string;
  
  /** Insurance information */
  insurance?: {
    provider?: string;
    plan?: string;
    coveredAmount?: number;
    outOfPocket?: number;
  };
  
  /** Location information */
  location?: {
    address?: string;
    city?: string;
    state?: string;
    zipCode?: string;
    coordinates?: {
      latitude: number;
      longitude: number;
    };
  };
  
  /** Availability information */
  availability?: {
    nextAvailable?: Date;
    timeSlots?: string[];
  };

  /** Estimated duration in minutes (enriched) */
  estimatedDurationMinutes?: number;
  
  /** Date when this price was effective */
  effectiveDate: Date;
  
  /** Date when this price record was last updated */
  lastUpdated: Date;

  /** Sync batch identifier for provider syncs */
  batchId?: string;

  /** Timestamp of the sync that ingested this record (ISO string) */
  syncTimestamp?: string;
  
  /** Source of the data */
  source: string;

  /** Search tags for this price record */
  tags?: string[];

  /** Metadata about tag generation */
  tagMetadata?: {
    generatedAt?: Date;
    model?: string;
    confidence?: number;
    curated?: {
      sources?: string[];
      facilityTags?: string[];
      ruleTags?: string[];
      metadataTags?: string[];
      matchedRules?: string[];
    };
  };
  
  /** Additional metadata */
  metadata?: Record<string, any>;
}

/**
 * Configuration for Google Sheets provider
 */
export interface GoogleSheetsConfig {
  /** Google Sheets API credentials */
  credentials: {
    clientEmail: string;
    privateKey: string;
    projectId: string;
  };
  
  /** Spreadsheet IDs to query */
  spreadsheetIds: string[];
  
  /** Sheet names to read from (optional, defaults to first sheet) */
  sheetNames?: string[];
  
  /** Column mapping for price data */
  columnMapping?: {
    facilityName?: string;
    procedureCode?: string;
    procedureDescription?: string;
    price?: string;
    effectiveDate?: string;
    [key: string]: string | undefined;
  };
  
  /** Sync schedule (cron expression or interval) */
  syncSchedule?: string;
}
