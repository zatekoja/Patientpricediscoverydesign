import { describe, it, before, after } from 'node:test';
import assert from 'node:assert';
import { createClient } from 'redis';
import { CapacityService } from '../../services/CapacityService';
import { TransactionIngestionService, IngestionEvent } from '../../ingestion/TransactionIngestionService';

describe('Capacity Dynamic Behavior E2E', () => {
  let redisClient: any;
  let service: CapacityService;
  let ingestion: TransactionIngestionService;
  const WARD = 'e2e_test_ward';
  const FACILITY = 'e2e_facility';

  before(async () => {
    redisClient = createClient({ url: 'redis://localhost:6379' });
    await redisClient.connect();
    service = new CapacityService(redisClient);
    ingestion = new TransactionIngestionService(service, null, null);
  });

  after(async () => {
    await redisClient.quit();
  });

  it('should transition through lifecycle: Immature -> Mature -> Busy -> Full', async () => {
    await redisClient.del(`capacity:${FACILITY}:${WARD}`);
    await redisClient.del(`capacity_history:${FACILITY}:${WARD}`);

    const baseEvent: IngestionEvent = {
      wardId: WARD,
      facilityId: FACILITY,
      transactionAmount: 1000,
      currency: 'NGN',
      reference: 'REF-1',
      timestamp: new Date()
    };

    // 1. New Ward (Immature) - Should stay 'available' with default thresholds (50/100)
    let result = await ingestion.ingestEvent({ ...baseEvent, reference: 'REF-START' });
    assert.strictEqual(result.status, 'available');
    assert.strictEqual(result.thresholds.busy, 50);

    // 2. Make it Mature with varied history (Avg ~ 6)
    for (let i = 1; i <= 4; i++) {
      const now = Date.now() - (i * 1000);
      await redisClient.zAdd(`capacity_history:${FACILITY}:${WARD}`, { score: now, value: `5_${now}` });
    }
    const now95 = Date.now() - 5000;
    await redisClient.zAdd(`capacity_history:${FACILITY}:${WARD}`, { score: now95, value: `10_${now95}` });

    // 3. Now ingest - p75 of [5,5,5,5,10] is 5. p95 is 10.
    result = await ingestion.ingestEvent({ ...baseEvent, reference: 'REF-MATURE' });
    // Current count is 2 (REF-START and REF-MATURE).
    assert.strictEqual(result.status, 'available');
    assert.strictEqual(result.thresholds.busy, 5);
    assert.strictEqual(result.thresholds.full, 10);

    // 4. Trigger Busy (Count >= 5)
    await service.recordEvent(FACILITY, WARD);
    await service.recordEvent(FACILITY, WARD);
    result = await ingestion.ingestEvent({ ...baseEvent, reference: 'REF-BUSY' });
    
    assert.strictEqual(result.capacityCount, 5);
    assert.strictEqual(result.status, 'busy');

    // 5. Trigger Full (Count >= 10)
    for(let i=0; i<5; i++) await service.recordEvent(FACILITY, WARD);
    result = await ingestion.ingestEvent({ ...baseEvent, reference: 'REF-FULL' });
    assert.strictEqual(result.status, 'full');
    
    // 6. Check Trend (Avg=5, Current=16 -> > 1.5x)
    // Add many events to spike trend
    for(let i=0; i<10; i++) await service.recordEvent(FACILITY, WARD);
    result = await ingestion.ingestEvent({ ...baseEvent, reference: 'REF-TREND' });
    assert.strictEqual(result.isBusy, true);
    assert.strictEqual(result.trend, 'increasing');
  });
});
