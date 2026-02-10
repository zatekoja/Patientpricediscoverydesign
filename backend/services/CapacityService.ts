import { RedisClientType } from 'redis';

export interface WardConfig {
  baseWaitMinutes: number;
  congestionFactor: number; // Max additional minutes when at p75
}

export const DEFAULT_WARD_CONFIGS: Record<string, WardConfig> = {
  maternity: { baseWaitMinutes: 30, congestionFactor: 45 },
  emergency: { baseWaitMinutes: 15, congestionFactor: 90 },
  pharmacy: { baseWaitMinutes: 10, congestionFactor: 20 },
  laboratory: { baseWaitMinutes: 15, congestionFactor: 30 },
  radiology: { baseWaitMinutes: 20, congestionFactor: 40 },
  general: { baseWaitMinutes: 20, congestionFactor: 30 },
};

export class CapacityService {
  private redis: RedisClientType;
  private readonly KEY_PREFIX = 'capacity';

  constructor(redisClient: any) {
    this.redis = redisClient;
  }

  private getWardConfig(wardId: string): WardConfig {
    const key = wardId.toLowerCase();
    for (const [configKey, config] of Object.entries(DEFAULT_WARD_CONFIGS)) {
      if (key.includes(configKey)) return config;
    }
    return DEFAULT_WARD_CONFIGS.general;
  }

  private buildKey(facilityId: string, wardId: string): string {
    return `${this.KEY_PREFIX}:${facilityId}:${wardId}`;
  }

  private buildHistoryKey(facilityId: string, wardId: string): string {
    return `capacity_history:${facilityId}:${wardId}`;
  }

  async recordEvent(facilityId: string, wardId: string): Promise<number> {
    const key = this.buildKey(facilityId, wardId);
    const now = Date.now();
    const eventId = `${now}-${Math.random().toString(36).substring(7)}`;
    
    // Add event
    await this.redis.zAdd(key, { score: now, value: eventId });
    
    // Cleanup old events (default 4 hours)
    const fourHoursAgo = now - (4 * 60 * 60 * 1000);
    await this.redis.zRemRangeByScore(key, '-inf', fourHoursAgo);
    
    return await this.redis.zCard(key);
  }

  async getWindowCount(facilityId: string, wardId: string, windowMinutes: number = 240): Promise<number> {
    const key = this.buildKey(facilityId, wardId);
    const now = Date.now();
    const windowStart = now - (windowMinutes * 60 * 1000);
    
    await this.redis.zRemRangeByScore(key, '-inf', windowStart);
    return await this.redis.zCard(key);
  }

  async recordSnapshot(facilityId: string, wardId: string): Promise<void> {
    const count = await this.getWindowCount(facilityId, wardId);
    const historyKey = this.buildHistoryKey(facilityId, wardId);
    const now = Date.now();
    await this.redis.zAdd(historyKey, { score: now, value: `${count}_${now}` });
    
    // Keep 7 days of history
    const sevenDaysAgo = now - (7 * 24 * 60 * 60 * 1000);
    await this.redis.zRemRangeByScore(historyKey, '-inf', sevenDaysAgo);
  }

  async calculateP75Threshold(facilityId: string, wardId: string): Promise<number> {
    const historyKey = this.buildHistoryKey(facilityId, wardId);
    const samples = await this.redis.zRange(historyKey, 0, -1);
    
    if (samples.length === 0) return 50; // Default fallback

    const counts = samples
      .map(s => parseInt(s.split('_')[0], 10))
      .sort((a, b) => a - b);

    const index = Math.ceil(0.75 * counts.length) - 1;
    return counts[Math.max(0, index)];
  }

  async analyzeCapacity(facilityId: string, wardId: string): Promise<{
    count: number;
    thresholds: { busy: number; full: number };
    status: 'available' | 'busy' | 'full';
    trend: 'stable' | 'increasing' | 'decreasing';
    estimatedWaitMinutes: number;
    isMature: boolean;
  }> {
    const currentCount = await this.getWindowCount(facilityId, wardId);
    const historyKey = this.buildHistoryKey(facilityId, wardId);
    const samples = await this.redis.zRange(historyKey, 0, -1);
    
    const MATURITY_THRESHOLD = 5;
    const isMature = samples.length >= MATURITY_THRESHOLD;

    let busyThreshold = 50; // Default
    let fullThreshold = 100; // Default
    let trend: 'stable' | 'increasing' | 'decreasing' = 'stable';
    let avgHistory = 25;

    if (isMature) {
      const counts = samples
        .map(s => parseInt(s.split('_')[0], 10))
        .sort((a, b) => a - b);

      const p75Idx = Math.ceil(0.75 * counts.length) - 1;
      const p95Idx = Math.ceil(0.95 * counts.length) - 1;
      
      busyThreshold = counts[Math.max(0, p75Idx)];
      fullThreshold = counts[Math.max(0, p95Idx)];

      // Simple trend: current count vs moving average
      const sum = counts.reduce((a, b) => a + b, 0);
      avgHistory = sum / counts.length;
      
      if (currentCount > avgHistory * 1.5) {
        trend = 'increasing';
      } else if (currentCount < avgHistory * 0.5) {
        trend = 'decreasing';
      }
    } else {
        // For immature samples, use a conservative default 
        busyThreshold = 50; 
    }

    let status: 'available' | 'busy' | 'full' = 'available';
    if (currentCount >= fullThreshold) {
      status = 'full';
    } else if (currentCount >= busyThreshold) {
      status = 'busy';
    }

    // Estimate Wait Time
    const config = this.getWardConfig(wardId);
    let waitTime = config.baseWaitMinutes;
    
    if (currentCount > 0) {
        // Ratio of current load vs current dynamic or default "Busy" mark
        const loadFactor = currentCount / Math.max(1, busyThreshold); 
        waitTime += (loadFactor * config.congestionFactor);
    }

    if (trend === 'increasing') waitTime *= 1.25;
    if (trend === 'decreasing') waitTime *= 0.85;

    return {
      count: currentCount,
      thresholds: { busy: busyThreshold, full: fullThreshold },
      status,
      trend,
      estimatedWaitMinutes: Math.round(waitTime),
      isMature
    };
  }
}
