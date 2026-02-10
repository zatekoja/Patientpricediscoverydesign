import { BlnkService, BlnkExternalTransaction } from '../internal/adapters/blnk/BlnkService';

export class ReconciliationService {
  constructor(private blnk: BlnkService) {}

  /**
   * Performs an "Instant" reconciliation for a batch of external transactions
   * @param externalTransactions Transactions from Flutterwave/Bank
   * @param ruleIds IDs of matching rules to apply
   */
  async reconcileInstant(externalTransactions: BlnkExternalTransaction[], ruleIds: string[]) {
    return await this.blnk.startInstantReconciliation({
      external_transactions: externalTransactions,
      strategy: 'one_to_one', // Fixed: Blnk uses underscores
      matching_rule_ids: ruleIds
    });
  }

  /**
   * Helper to initialize standard Megalek matching rules
   */
  async ensureStandardRules() {
    return await this.blnk.createMatchingRule({
      name: 'Megalek Default Match',
      description: 'Match by amount and reference string',
      criteria: [
        { field: 'amount', operator: 'equals' },
        { field: 'reference', operator: 'equals' }
      ]
    });
  }
}
