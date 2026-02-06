/**
 * External Data Provider System
 * Main export file for the backend system
 */

// Interfaces
export { IExternalDataProvider, DataProviderOptions, DataProviderResponse } from './interfaces/IExternalDataProvider';
export { IDocumentStore } from './interfaces/IDocumentStore';

// Base Provider
export { BaseDataProvider } from './providers/BaseDataProvider';

// Providers
export { MegalekAteruHelper } from './providers/MegalekAteruHelper';

// Stores
export { InMemoryDocumentStore } from './stores/InMemoryDocumentStore';

// Scheduler
export { DataSyncScheduler, SyncJobConfig, SyncIntervals } from './config/DataSyncScheduler';

// Types
export { PriceData, GoogleSheetsConfig } from './types/PriceData';
