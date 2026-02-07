export interface ProviderState {
  lastSyncDate?: string;
  lastBatchId?: string;
  previousBatchId?: string;
}

export interface IProviderStateStore {
  getState(providerName: string): Promise<ProviderState | null>;
  saveState(providerName: string, state: ProviderState): Promise<void>;
}
