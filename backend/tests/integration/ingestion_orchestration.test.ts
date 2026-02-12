import { describe, it, before, after } from 'node:test';
import assert from 'node:assert';
import { createClient } from 'redis';
import { BlnkService } from '../../internal/adapters/blnk/BlnkService';
import { CapacityService } from '../../services/CapacityService';
import { TransactionIngestionService } from '../../ingestion/TransactionIngestionService';

describe('TransactionIngestionService Orchestration', () => {
  let redisClient: any;
  let capacityService: CapacityService;
  let blnkService: BlnkService;
  let ingestionService: TransactionIngestionService;
  
  // Mock FacilityProfileService
  const mockFacilityService = {
    updateStatus: async (id: string, update: any) => {
      // Store the update for verification
      (mockFacilityService as any).lastUpdate = { id, update };
      return {} as any;
    },
    lastUpdate: null as any
  };

  const WARD = 'maternity_ward_orch';
  const FACILITY = 'facility_123';
  const LEDGER_NAME = 'Ingestion Ledger';
  const CURRENCY = 'NGN';
  
  let sourceBalanceId: string;
  let destBalanceId: string;

  before(async () => {
    // Redis
    redisClient = createClient({ url: 'redis://localhost:6379' });
    await redisClient.connect();
    capacityService = new CapacityService(redisClient);

    // Blnk
    blnkService = new BlnkService('http://localhost:5001');
    try {
      // Clear Redis
      await redisClient.del(`capacity:${FACILITY}:${WARD}`);
      await redisClient.del(`capacity_history:${FACILITY}:${WARD}`);

      const ledger = await blnkService.createLedger(LEDGER_NAME);
      const s = await blnkService.createBalance({ ledgerId: ledger.ledger_id, currency: CURRENCY, currency_multiplier: 1 });
      const d = await blnkService.createBalance({ ledgerId: ledger.ledger_id, currency: CURRENCY, currency_multiplier: 1 });
      sourceBalanceId = s.balance_id;
      destBalanceId = d.balance_id;
    } catch (e) {
      console.warn("Ledger setup failed (maybe exists), continuing...", e);
    }

    // Ingestion Service (Low threshold for testing)
    ingestionService = new TransactionIngestionService(
      capacityService,
      blnkService,
      mockFacilityService as any,
      { capacityThreshold: 2, windowMinutes: 60 }
    );
  });

  after(async () => {
    await redisClient.quit();
  });

  it('should ingest an event and record in Blnk + Redis', async () => {
    // Clear Redis
    await redisClient.del(`capacity:${WARD}`);

    const event = {
      wardId: WARD,
      facilityId: FACILITY,
      transactionAmount: 1000,
      currency: CURRENCY,
      reference: `REF-${Date.now()}`,
      sourceAccount: sourceBalanceId,
      destinationAccount: destBalanceId,
      timestamp: new Date()
    };

    const result = await ingestionService.ingestEvent(event);

    assert.strictEqual(result.capacityCount, 1);
    assert.strictEqual(result.status, 'available'); 
    assert.ok(result.blnkTransactionId, 'Blnk Transaction ID should be present');
  });

  it('should trigger busy status when threshold reached', async () => {
    // Seed history so p75 threshold is low (e.g. 5)
    const historyKey = `capacity_history:${FACILITY}:${WARD}`;
    await redisClient.del(historyKey);
    const now = Date.now();
    for (let i = 1; i <= 5; i++) {
        await redisClient.zAdd(historyKey, { score: now-i, value: `5_${now-i}` });
    }

    // Record many events to exceed threshold 5
    await capacityService.recordEvent(FACILITY, WARD);
    await capacityService.recordEvent(FACILITY, WARD);
    await capacityService.recordEvent(FACILITY, WARD);
    await capacityService.recordEvent(FACILITY, WARD);

    const event = {
      wardId: WARD,
      facilityId: FACILITY,
      transactionAmount: 1000,
      currency: CURRENCY,
      reference: `REF-${Date.now()}-2`,
      sourceAccount: sourceBalanceId,
      destinationAccount: destBalanceId,
      timestamp: new Date()
    };

    const result = await ingestionService.ingestEvent(event);

    assert.ok(result.capacityCount >= 5);
    assert.notStrictEqual(result.status, 'available');
    
    // Check facility update
    assert.deepStrictEqual(mockFacilityService.lastUpdate.id, FACILITY);
    assert.ok(['busy', 'full'].includes(mockFacilityService.lastUpdate.update.wardUpdate.status));
  });
});
