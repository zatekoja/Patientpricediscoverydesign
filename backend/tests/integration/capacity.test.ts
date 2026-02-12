import { describe, it, before, after } from 'node:test';
import assert from 'node:assert';
import { createClient } from 'redis';
import { CapacityService } from '../../services/CapacityService';

describe('CapacityService (Sliding Window)', () => {
  let redisClient: any;
  let service: CapacityService;
  const WARD = 'maternity_ward';
  const FACILITY = 'test_facility';

  before(async () => {
    redisClient = createClient({ url: 'redis://localhost:6379' });
    await redisClient.connect();
    service = new CapacityService(redisClient);
  });

  after(async () => {
    await redisClient.quit();
  });

  it('should record an event in the sliding window', async () => {
    await redisClient.del(`capacity:${FACILITY}:${WARD}`);
    
    await service.recordEvent(FACILITY, WARD);
    
    const count = await redisClient.zCard(`capacity:${FACILITY}:${WARD}`);
    assert.strictEqual(count, 1);
  });

  it('should count events within the window', async () => {
    await redisClient.del(`capacity:${FACILITY}:${WARD}`);
    
    // Add old event (5 hours ago)
    const oldTime = Date.now() - (5 * 60 * 60 * 1000);
    await redisClient.zAdd(`capacity:${FACILITY}:${WARD}`, { score: oldTime, value: `evt_${oldTime}` });

    // Add new event
    await service.recordEvent(FACILITY, WARD);

    // Get count (should exclude old event if we implement clean up)
    const count = await service.getWindowCount(FACILITY, WARD, 4 * 60); // 4 hours in minutes
    assert.strictEqual(count, 1);
  });

  it('should calculate p75 dynamic threshold', async () => {
    const HISTORY_KEY = `capacity_history:${FACILITY}:${WARD}`;
    await redisClient.del(HISTORY_KEY);

    // Seed history with counts: 10, 20, 30, 40, 50, 60, 70, 80, 90, 100
    // Total 10 items. p75 is the 7.5th item. (approx 80)
    for (let i = 1; i <= 10; i++) {
      const timestamp = Date.now() - (i * 1000);
      await redisClient.zAdd(HISTORY_KEY, { score: timestamp, value: `${i*10}_${timestamp}` });
    }

    const threshold = await service.calculateP75Threshold(FACILITY, WARD);
    // Counts: 10, 20, 30, 40, 50, 60, 70, 80, 90, 100
    // Index for p75: 8th item -> 80
    assert.strictEqual(threshold, 80);
  });

  it('should use default threshold when samples are insufficient (maturity)', async () => {
    const WARD_NEW = 'new_ward';
    const HISTORY_KEY = `capacity_history:${FACILITY}:${WARD_NEW}`;
    await redisClient.del(HISTORY_KEY);

    // Only 2 samples (less than maturity requirement of 5)
    await redisClient.zAdd(HISTORY_KEY, { score: Date.now(), value: `10_${Date.now()}` });
    await redisClient.zAdd(HISTORY_KEY, { score: Date.now() + 1, value: `20_${Date.now()}` });

    const analysis = await service.analyzeCapacity(FACILITY, WARD_NEW);
    // Should fallback to default (50) because it's not "mature"
    assert.strictEqual(analysis.thresholds.busy, 50);
    assert.strictEqual(analysis.status, 'available');
  });

  it('should distinguish between busy (p75) and full (p95)', async () => {
    const WARD_MATURE = 'mature_ward';
    const HISTORY_KEY = `capacity_history:${FACILITY}:${WARD_MATURE}`;
    await redisClient.del(HISTORY_KEY);
    await redisClient.del(`capacity:${FACILITY}:${WARD_MATURE}`);

    // Seed 10 samples: 10, 20, ... 100
    for (let i = 1; i <= 10; i++) {
      await redisClient.zAdd(HISTORY_KEY, { score: Date.now() - i, value: `${i * 10}_${Date.now() - i}` });
    }

    // Current count is 85 (above p75=80, below p95=100)
    for (let i = 0; i < 85; i++) {
      await redisClient.zAdd(`capacity:${FACILITY}:${WARD_MATURE}`, { score: Date.now(), value: `tx_${i}` });
    }

    const analysis = await service.analyzeCapacity(FACILITY, WARD_MATURE);
    assert.strictEqual(analysis.status, 'busy');

    // Add more to reach 100 (p95)
    for (let i = 85; i < 105; i++) {
      await redisClient.zAdd(`capacity:${FACILITY}:${WARD_MATURE}`, { score: Date.now(), value: `tx_${i}` });
    }

    const analysisFull = await service.analyzeCapacity(FACILITY, WARD_MATURE);
    assert.strictEqual(analysisFull.status, 'full');
  });

  it('should detect increasing trend', async () => {
    const WARD_TREND = 'trend_ward';
    const HISTORY_KEY = `capacity_history:${FACILITY}:${WARD_TREND}`;
    await redisClient.del(HISTORY_KEY);
    await redisClient.del(`capacity:${FACILITY}:${WARD_TREND}`);

    // History average is 10
    const now = Date.now();
    for (let i = 1; i <= 10; i++) {
      const ts = now - (i * 60 * 1000); // Minutes apart
      await redisClient.zAdd(HISTORY_KEY, { score: ts, value: `10_${ts}` });
    }

    // Current count is 20 (2x average)
    for (let i = 0; i < 20; i++) {
      await redisClient.zAdd(`capacity:${FACILITY}:${WARD_TREND}`, { score: now, value: `tx_${i}` });
    }

    const analysis = await service.analyzeCapacity(FACILITY, WARD_TREND);
    assert.strictEqual(analysis.trend, 'increasing');
  });

  it('should estimate wait time based on load and trend', async () => {
    const WARD_WAIT = 'emergency_ward';
    const HISTORY_KEY = `capacity_history:${FACILITY}:${WARD_WAIT}`;
    await redisClient.del(HISTORY_KEY);
    await redisClient.del(`capacity:${FACILITY}:${WARD_WAIT}`);

    // Emergency baseWait = 15, congestion = 90. p75 = 50 (default)
    // Low load: count = 10 (20% of busy mark)
    // Estimated = 15 + (0.2 * 90) = 15 + 18 = 33
    for(let i=0; i<10; i++) await service.recordEvent(FACILITY, WARD_WAIT);
    
    let analysis = await service.analyzeCapacity(FACILITY, WARD_WAIT);
    assert.strictEqual(analysis.estimatedWaitMinutes, 33);

    // High load + Spike: count = 100
    // Busy Threshold becomes 100 because of p75 of history or default?
    // Let's force it to 100 by seeding history.
    for (let i = 1; i <= 5; i++) {
        const ts = Date.now() - (i * 1000);
        await redisClient.zAdd(HISTORY_KEY, { score: ts, value: `100_${ts}` });
    }
    // Average becomes 100. Current count 100. Trend stable (multiplier 1.0)
    // Estimated = 15 + (100/100 * 90) = 105
    for(let i=10; i<100; i++) await service.recordEvent(FACILITY, WARD_WAIT);
    
    analysis = await service.analyzeCapacity(FACILITY, WARD_WAIT);
    assert.strictEqual(analysis.estimatedWaitMinutes, 105);
  });
});
