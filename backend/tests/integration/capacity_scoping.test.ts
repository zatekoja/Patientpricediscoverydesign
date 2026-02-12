import { describe, it, before, after } from 'node:test';
import assert from 'node:assert';
import { createClient } from 'redis';
import { CapacityService } from '../../services/CapacityService';

describe('Capacity Scoping (Facility-Ward Isolation)', () => {
  let redisClient: any;
  let service: CapacityService;
  
  const WARD = 'maternity';
  const FACILITY_A = 'hospital_ijede';
  const FACILITY_B = 'hospital_randle';

  before(async () => {
    redisClient = createClient({ url: 'redis://localhost:6379' });
    await redisClient.connect();
    service = new CapacityService(redisClient);
  });

  after(async () => {
    await redisClient.quit();
  });

  it('should isolate capacity counts between different facilities for the same ward', async () => {
    // Clear previous state
    await redisClient.del(`capacity:${FACILITY_A}:${WARD}`);
    await redisClient.del(`capacity:${FACILITY_B}:${WARD}`);

    // Record 3 events for Facility A
    await (service as any).recordEvent(FACILITY_A, WARD);
    await (service as any).recordEvent(FACILITY_A, WARD);
    await (service as any).recordEvent(FACILITY_A, WARD);

    // Record 1 event for Facility B
    await (service as any).recordEvent(FACILITY_B, WARD);

    const countA = await (service as any).getWindowCount(FACILITY_A, WARD);
    const countB = await (service as any).getWindowCount(FACILITY_B, WARD);

    assert.strictEqual(countA, 3, 'Facility A count should be 3');
    assert.strictEqual(countB, 1, 'Facility B count should be 1');
  });

  it('should isolate history and thresholds between facilities', async () => {
    await redisClient.del(`capacity_history:${FACILITY_A}:${WARD}`);
    await redisClient.del(`capacity_history:${FACILITY_B}:${WARD}`);

    // Seed high history for A (Avg = 100)
    for (let i = 1; i <= 10; i++) {
      const ts = Date.now() - (i * 1000);
      await redisClient.zAdd(`capacity_history:${FACILITY_A}:${WARD}`, { score: ts, value: `100_${ts}` });
    }

    // Seed low history for B (Avg = 10)
    for (let i = 1; i <= 10; i++) {
      const ts = Date.now() - (i * 1000);
      await redisClient.zAdd(`capacity_history:${FACILITY_B}:${WARD}`, { score: ts, value: `10_${ts}` });
    }

    const analysisA = await (service as any).analyzeCapacity(FACILITY_A, WARD);
    const analysisB = await (service as any).analyzeCapacity(FACILITY_B, WARD);

    assert.strictEqual(analysisA.thresholds.busy, 100);
    assert.strictEqual(analysisB.thresholds.busy, 10);
  });
});
