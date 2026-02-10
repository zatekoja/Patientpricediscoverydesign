import { describe, it, before } from 'node:test';
import assert from 'node:assert';
import { BlnkService } from '../../internal/adapters/blnk/BlnkService';

describe('BlnkService Integration', () => {
  let service: BlnkService;
  const LEDGER_NAME = 'Megalek General Ledger';
  const CURRENCY = 'NGN';

  before(() => {
    // Point to localhost:5001 as verified in Phase 1
    service = new BlnkService('http://localhost:5001');
  });

  it('should create a general ledger', async () => {
    const ledger = await service.createLedger(LEDGER_NAME);
    assert.ok(ledger.ledger_id, 'Ledger ID should be returned');
    assert.strictEqual(ledger.name, LEDGER_NAME);
  });

  it('should create a balance (account) in the ledger', async () => {
    // First ensure ledger exists (idempotent helper usually, but here we just flow)
    const ledger = await service.createLedger(LEDGER_NAME);
    
    const balance = await service.createBalance({
      ledgerId: ledger.ledger_id,
      currency: CURRENCY,
      currency_multiplier: 1 // Blnk specific
    });

    assert.ok(balance.balance_id, 'Balance ID should be returned');
    assert.strictEqual(balance.currency, CURRENCY);
  });

  it('should record a transaction', async () => {
    const ledger = await service.createLedger(LEDGER_NAME);
    const source = await service.createBalance({ ledgerId: ledger.ledger_id, currency: CURRENCY, currency_multiplier: 1 });
    const destination = await service.createBalance({ ledgerId: ledger.ledger_id, currency: CURRENCY, currency_multiplier: 1 });

    const transaction = await service.recordTransaction({
      amount: 5000,
      currency: CURRENCY,
      source: source.balance_id,
      destination: destination.balance_id,
      reference: `REF-${Date.now()}`,
      description: 'Test Transaction'
    });

    assert.ok(transaction.transaction_id, 'Transaction ID should be returned');
    assert.strictEqual(transaction.amount, 5000);
    assert.strictEqual(transaction.status, 'QUEUED'); // Or APPLIED, depending on Blnk speed
  });
});
