import axios, { AxiosInstance } from 'axios';
import crypto from 'crypto';

export interface FlutterwaveTransaction {
  id: number;
  tx_ref: string;
  flw_ref: string;
  amount: number;
  currency: string;
  status: string;
  customer: {
    id?: number;
    name: string;
    email: string;
    phone_number?: string;
  };
  meta?: Record<string, any>;
  narration?: string;
  created_at?: string;
}

export interface FlutterwaveWebhookPayload {
  event: string;
  data: FlutterwaveTransaction;
  id?: string;
  timestamp?: number;
}

export class FlutterwaveService {
  private client: AxiosInstance;
  private webhookSecret: string;

  constructor(secretKey: string, webhookSecret: string) {
    this.webhookSecret = webhookSecret;
    this.client = axios.create({
      baseURL: 'https://api.flutterwave.com/v3',
      headers: {
        Authorization: `Bearer ${secretKey}`,
        'Content-Type': 'application/json',
      },
    });
  }

  /**
   * Verifies the webhook signature (verif-hash)
   */
  verifySignature(payload: any, signature: string): boolean {
    if (!signature || !this.webhookSecret) return false;
    return signature === this.webhookSecret;
  }

  /**
   * Calls Flutterwave API to verify transaction details by ID
   */
  async verifyTransaction(transactionId: number): Promise<FlutterwaveTransaction> {
    try {
      const response = await this.client.get(`/transactions/${transactionId}/verify`);
      if (response.data.status === 'success') {
        return response.data.data;
      }
      throw new Error(`Flutterwave verification failed: ${response.data.message}`);
    } catch (error: any) {
      console.error('Error verifying Flutterwave transaction:', error.response?.data || error.message);
      throw error;
    }
  }
}
