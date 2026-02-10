import axios, { AxiosInstance } from 'axios';

export interface BlnkLedger {
  ledger_id: string;
  name: string;
  created_at: string;
}

export interface BlnkBalance {
  balance_id: string;
  ledger_id: string;
  currency: string;
  currency_multiplier: number;
}

export interface BlnkTransaction {
  transaction_id: string;
  amount: number;
  status: string;
}

export interface BlnkMatchingCriteria {
  field: string;
  operator: string;
  value?: string;
  pattern?: string;
  allowable_drift?: number;
}

export interface BlnkMatchingRule {
  rule_id?: string;
  name: string;
  description: string;
  criteria: BlnkMatchingCriteria[];
}

export interface BlnkExternalTransaction {
  id: string;
  amount: number;
  reference: string;
  currency: string;
  description: string;
  date: string; // ISO Date
  source: string;
}

export interface BlnkReconciliationResult {
  reconciliation_id: string;
  status: string;
  matched_transactions: number;
  unmatched_transactions: number;
}

export class BlnkService {
  private client: AxiosInstance;

  constructor(baseUrl: string) {
    this.client = axios.create({
      baseURL: baseUrl,
      headers: {
        'Content-Type': 'application/json',
      },
    });
  }

  async createLedger(name: string): Promise<BlnkLedger> {
    try {
      const response = await this.client.post('/ledgers', { name });
      // console.log('Create Ledger Response:', JSON.stringify(response.data, null, 2));
      return response.data;
    } catch (error: any) {
      console.error('Error creating ledger:', error.response?.data || error.message);
      throw error;
    }
  }

  async createBalance(params: { ledgerId: string; currency: string; currency_multiplier: number }): Promise<BlnkBalance> {
    try {
      const response = await this.client.post('/balances', {
        ledger_id: params.ledgerId,
        currency: params.currency,
        currency_multiplier: params.currency_multiplier
      });
      return response.data;
    } catch (error: any) {
      console.error('Error creating balance:', error.response?.data || error.message);
      throw error;
    }
  }

  async recordTransaction(params: { 
    amount: number; 
    currency: string; 
    source: string; 
    destination: string; 
    reference: string; 
    description: string;
    allow_overdraft?: boolean;
    skip_queue?: boolean;
  }): Promise<BlnkTransaction> {
    try {
      const response = await this.client.post('/transactions', {
        amount: params.amount,
        currency: params.currency,
        source: params.source,
        destination: params.destination,
        reference: params.reference,
        description: params.description,
        allow_overdraft: params.allow_overdraft,
        skip_queue: params.skip_queue
      });
      return response.data;
    } catch (error: any) {
      console.error('Error recording transaction:', error.response?.data || error.message);
      throw error;
    }
  }

  async createMatchingRule(rule: BlnkMatchingRule): Promise<BlnkMatchingRule> {
    try {
      const response = await this.client.post('/reconciliation/matching-rules', rule);
      return response.data;
    } catch (error: any) {
      console.error('Error creating matching rule:', error.response?.data || error.message);
      throw error;
    }
  }

  async startInstantReconciliation(params: { 
    external_transactions: BlnkExternalTransaction[]; 
    strategy: string; 
    matching_rule_ids: string[] 
  }): Promise<{ reconciliation_id: string }> {
    try {
      const response = await this.client.post('/reconciliation/start-instant', params);
      return response.data;
    } catch (error: any) {
      console.error('Error starting instant reconciliation:', error.response?.data || error.message);
      throw error;
    }
  }

  async getReconciliation(id: string): Promise<BlnkReconciliationResult> {
    try {
      const response = await this.client.get(`/reconciliation/${id}`);
      return response.data;
    } catch (error: any) {
      console.error('Error getting reconciliation:', error.response?.data || error.message);
      throw error;
    }
  }

  async getTransaction(id: string): Promise<BlnkTransaction> {
    try {
      const response = await this.client.get(`/transactions/${id}`);
      return response.data;
    } catch (error: any) {
      console.error('Error getting transaction:', error.response?.data || error.message);
      throw error;
    }
  }
}
