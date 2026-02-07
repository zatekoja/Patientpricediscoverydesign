import { IExternalDataProvider } from '../interfaces/IExternalDataProvider';
import { recordSchedulerRun, recordSchedulerSkip } from '../observability/metrics';

/**
 * Configuration for scheduled sync jobs
 */
export interface SyncJobConfig {
  /** Job name */
  name: string;
  
  /** Data provider to sync */
  provider: IExternalDataProvider;
  
  /** Sync interval in milliseconds (e.g., 3 days = 3 * 24 * 60 * 60 * 1000) */
  intervalMs: number;
  
  /** Whether to run immediately on start */
  runImmediately?: boolean;
  
  /** Callback for sync completion */
  onComplete?: (result: {
    success: boolean;
    recordsProcessed: number;
    timestamp: Date;
    error?: string;
  }) => void;
  
  /** Callback for sync errors */
  onError?: (error: Error) => void;
}

/**
 * Scheduler for running data sync jobs
 * Supports scheduling providers to sync data at regular intervals
 */
export class DataSyncScheduler {
  private jobs: Map<string, NodeJS.Timeout> = new Map();
  private jobConfigs: Map<string, SyncJobConfig> = new Map();
  private runningJobs: Set<string> = new Set();
  
  /**
   * Schedule a sync job
   */
  scheduleJob(config: SyncJobConfig): void {
    // Stop existing job if it exists
    this.stopJob(config.name);
    
    // Store config
    this.jobConfigs.set(config.name, config);
    
    // Run immediately if configured
    if (config.runImmediately) {
      this.runSyncJob(config).catch(error => {
        console.error(`Error in immediate sync for job ${config.name}:`, error);
        config.onError?.(error);
      });
    }
    
    // Schedule recurring job
    const intervalId = setInterval(() => {
      this.runSyncJob(config).catch(error => {
        console.error(`Error in scheduled sync for job ${config.name}:`, error);
        config.onError?.(error);
      });
    }, config.intervalMs);
    
    this.jobs.set(config.name, intervalId);
    
    console.log(`Scheduled job '${config.name}' to run every ${config.intervalMs}ms`);
  }
  
  /**
   * Stop a scheduled job
   */
  stopJob(name: string): void {
    const intervalId = this.jobs.get(name);
    if (intervalId) {
      clearInterval(intervalId);
      this.jobs.delete(name);
      console.log(`Stopped job '${name}'`);
    }
  }
  
  /**
   * Stop all scheduled jobs
   */
  stopAll(): void {
    for (const [name, intervalId] of this.jobs) {
      clearInterval(intervalId);
      console.log(`Stopped job '${name}'`);
    }
    this.jobs.clear();
  }
  
  /**
   * Get active job names
   */
  getActiveJobs(): string[] {
    return Array.from(this.jobs.keys());
  }
  
  /**
   * Manually trigger a job to run now
   */
  async triggerJob(name: string): Promise<void> {
    const config = this.jobConfigs.get(name);
    if (!config) {
      throw new Error(`Job '${name}' not found`);
    }
    
    await this.runSyncJob(config);
  }
  
  /**
   * Run a sync job
   */
  private async runSyncJob(config: SyncJobConfig): Promise<void> {
    if (this.runningJobs.has(config.name)) {
      console.warn(`Sync job '${config.name}' skipped (previous run still in progress).`);
      recordSchedulerSkip(config.name);
      return;
    }

    this.runningJobs.add(config.name);
    const startTime = Date.now();
    console.log(`Running sync job '${config.name}'...`);
    
    try {
      const result = await config.provider.syncData();
      
      console.log(`Sync job '${config.name}' completed:`, {
        success: result.success,
        recordsProcessed: result.recordsProcessed,
        timestamp: result.timestamp,
      });
      
      config.onComplete?.(result);
      recordSchedulerRun({
        job: config.name,
        success: result.success,
        durationMs: Date.now() - startTime,
      });
      
      if (!result.success) {
        console.error(`Sync job '${config.name}' failed:`, result.error);
      }
    } catch (error) {
      console.error(`Sync job '${config.name}' threw error:`, error);
      
      const errorObj = error instanceof Error ? error : new Error(String(error));
      config.onError?.(errorObj);
      
      // Call onComplete with error result
      config.onComplete?.({
        success: false,
        recordsProcessed: 0,
        timestamp: new Date(),
        error: errorObj.message,
      });
      recordSchedulerRun({
        job: config.name,
        success: false,
        durationMs: Date.now() - startTime,
      });
    } finally {
      this.runningJobs.delete(config.name);
    }
  }
  
  /**
   * Get job status
   */
  getJobStatus(name: string): {
    exists: boolean;
    active: boolean;
    config?: SyncJobConfig;
  } {
    const config = this.jobConfigs.get(name);
    const isActive = this.jobs.has(name);
    
    return {
      exists: config !== undefined,
      active: isActive,
      config,
    };
  }
}

/**
 * Time constants for common intervals
 */
export const SyncIntervals = {
  THREE_DAYS: 3 * 24 * 60 * 60 * 1000,
  ONE_DAY: 24 * 60 * 60 * 1000,
  TWELVE_HOURS: 12 * 60 * 60 * 1000,
  SIX_HOURS: 6 * 60 * 60 * 1000,
  ONE_HOUR: 60 * 60 * 1000,
} as const;
