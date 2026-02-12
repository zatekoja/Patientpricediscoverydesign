import { describe, it, before, after } from 'node:test';
import assert from 'node:assert';
import http from 'http';
import { createClient } from 'redis';
import { FlutterwaveService } from '../../services/FlutterwaveService';
import { DataProviderAPI } from '../../api/server';
import { CapacityService } from '../../services/CapacityService';
import { TransactionIngestionService } from '../../ingestion/TransactionIngestionService';

describe('Flutterwave Webhook Integration', () => {
  let mockFlwServer: http.Server;
  let apiServer: http.Server;
  let redisClient: any;
  const PORT = 3005;
  const MOCK_FLW_PORT = 3006;
  const WEBHOOK_SECRET = 'test_webhook_secret';

  before(async () => {
    // 1. Mock Flutterwave API
    mockFlwServer = http.createServer((req, res) => {
      if (req.url?.includes('/verify')) {
        res.writeHead(200, { 'Content-Type': 'application/json' });
        res.end(JSON.stringify({
          status: 'success',
          data: {
            id: 12345,
            tx_ref: 'TX-VERIFIED-123',
            flw_ref: 'FLW-MOCK-123',
            amount: 5000,
            currency: 'NGN',
            status: 'successful',
            customer: {
              name: 'Test Patient',
              email: 'test@example.com'
            },
            meta: { ward_id: 'maternity', facility_id: 'gh_ijede' }
          }
        }));
      }
    });
    mockFlwServer.listen(MOCK_FLW_PORT);

    // 2. Setup App
    redisClient = createClient({ url: 'redis://localhost:6379' });
    await redisClient.connect();
    const capacityService = new CapacityService(redisClient);
    
    // Override Flutterwave base URL for testing
    const flwService = new FlutterwaveService('sk_test_123', WEBHOOK_SECRET);
    (flwService as any).client.defaults.baseURL = `http://localhost:${MOCK_FLW_PORT}`;

    const ingestionService = new TransactionIngestionService(capacityService, null, null);

    const api = new DataProviderAPI({
      transactionIngestionService: ingestionService,
      flutterwaveService: flwService
    });

    apiServer = api.getApp().listen(PORT);
  });

  after(async () => {
    apiServer.close();
    mockFlwServer.close();
    await redisClient.quit();
  });

  it('should verify signature and ingest verified transaction', async () => {
    // Clear Redis for this ward
    await redisClient.del('capacity:gh_ijede:maternity');

    const payload = {
      event: 'charge.completed',
      data: {
        id: 12345,
        tx_ref: 'TX-INIT-123',
        flw_ref: 'FLW-MOCK-123',
        amount: 5000,
        currency: 'NGN',
        status: 'successful',
        customer: {
          name: 'Test Patient',
          email: 'test@example.com'
        },
        meta: {
          ward_id: 'maternity',
          facility_id: 'gh_ijede'
        }
      }
    };

    const response = await fetch(`http://localhost:${PORT}/api/v1/webhooks/transaction`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'verif-hash': WEBHOOK_SECRET
      },
      body: JSON.stringify(payload)
    });

    const result = await response.json() as any;
    assert.strictEqual(response.status, 200);
    assert.strictEqual(result.success, true);
    assert.ok(result.capacityCount >= 1);
    assert.strictEqual(result.status, 'available');
    assert.ok(result.thresholds.busy > 0);
    
    // Check Redis directly to be sure
    const redisCount = await redisClient.zCard('capacity:gh_ijede:maternity');
    assert.strictEqual(redisCount, 1);
  });

  it('should reject unauthorized webhooks', async () => {
    const response = await fetch(`http://localhost:${PORT}/api/v1/webhooks/transaction`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'verif-hash': 'wrong_secret'
      },
      body: JSON.stringify({})
    });

    assert.strictEqual(response.status, 401);
  });
});
