import { describe, it, before } from 'node:test';
import assert from 'node:assert';
import { BlnkService } from '../../internal/adapters/blnk/BlnkService';
import { ReconciliationService } from '../../services/ReconciliationService';

describe('Financial Reconciliation Orchestration', () => {
  let blnkService: BlnkService;
  let reconciliationService: ReconciliationService;
  
  const LEDGER_NAME = 'Reconciliation Test Ledger';
  const CURRENCY = 'NGN';
  let sourceBalanceId: string;
  let destBalanceId: string;
  let ruleId: string;

  before(async () => {
    blnkService = new BlnkService('http://localhost:5001');
    reconciliationService = new ReconciliationService(blnkService);

    // Setup Ledger and Balances
    const ledger = await blnkService.createLedger(LEDGER_NAME);
    const s = await blnkService.createBalance({ ledgerId: ledger.ledger_id, currency: CURRENCY, currency_multiplier: 1 });
    const d = await blnkService.createBalance({ ledgerId: ledger.ledger_id, currency: CURRENCY, currency_multiplier: 1 });
    sourceBalanceId = s.balance_id;
    destBalanceId = d.balance_id;

    // Setup Matching Rule (Match by amount and reference)
    const rule = await blnkService.createMatchingRule({
      name: 'Amount and Reference Match',
      description: 'Matches transactions where amount and reference are equal',
      criteria: [
        { field: 'amount', operator: 'equals' },
        { field: 'reference', operator: 'equals' }
      ]
    });
    ruleId = rule.rule_id!;
  });

  it('should reconcile internal transactions against external statements', async () => {
    const reference = `TX-REC-${Date.now()}`;
    const amount = 7500;

    // 1. Record internal transaction
    const tx = await blnkService.recordTransaction({
      amount,
      currency: CURRENCY,
      source: sourceBalanceId,
      destination: destBalanceId,
      reference,
      description: 'Internal payment',
      allow_overdraft: true,
      skip_queue: true
    });

    // 1.1 Wait for transaction to be APPLIED by workers
    let appliedTx: any;
    for(let i=0; i<15; i++) {
        await new Promise(r => setTimeout(r, 1000));
        appliedTx = await blnkService.getTransaction(tx.transaction_id);
        console.log(`Polling transaction ${tx.transaction_id} status: ${appliedTx.status}`);
        if (appliedTx.status === 'APPLIED') break;
        if (appliedTx.status === 'REJECTED') break;
    }
    assert.strictEqual(appliedTx.status, 'APPLIED', `Transaction status: ${appliedTx.status}`);

    // 2. Simulate Receiving External Statement
    const externalTx = {
      id: `EXT-${reference}`,
      amount,
      reference, // This is the key link
      currency: CURRENCY,
      description: 'External payment recorded by Flutterwave',
      date: new Date().toISOString(),
      source: 'flutterwave'
    };

    // 3. Start Reconciliation
    const result = await reconciliationService.reconcileInstant([externalTx], [ruleId]);

    assert.ok(result.reconciliation_id, 'Should have a reconciliation ID');
    
    // 4. Poll for results (Blnk reconciliation is async)
    let summary: any;
    for(let i=0; i<10; i++) {
        await new Promise(r => setTimeout(r, 1000));
        summary = await blnkService.getReconciliation(result.reconciliation_id);
        if (summary && (summary.status === 'completed' || summary.status === 'success')) break;
    }

    if (!summary) throw new Error('Reconciliation summary not found');
    assert.ok(['completed', 'success'].includes(summary.status), `Status should be completed, got ${summary.status}`);
    assert.strictEqual(summary.matched_transactions, 1, 'Should have matched exactly 1 transaction');
  });
});
