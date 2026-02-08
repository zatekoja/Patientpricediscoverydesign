import assert from 'assert';
import http from 'http';
import { AddressInfo } from 'net';
import { DataProviderAPI } from '../../api/server';
import { FacilityProfileService } from '../../ingestion/facilityProfileService';
import {
  CapacityRequestService,
  CapacityRequestToken,
  EmailSender,
} from '../../ingestion/capacityRequestService';
import { InMemoryDocumentStore } from '../../stores/InMemoryDocumentStore';
import { FacilityProfile } from '../../types/FacilityProfile';

async function runTest(name: string, fn: () => void | Promise<void>): Promise<void> {
  try {
    await fn();
    console.log(`✓ ${name}`);
  } catch (err) {
    console.error(`✗ ${name}`);
    console.error(err);
    process.exitCode = 1;
  }
}

class CaptureEmailSender implements EmailSender {
  emails: Array<{ to: string; subject: string; html: string }> = [];
  lastHtml?: string;
  lastToken?: string;

  async send(to: string, subject: string, html: string): Promise<void> {
    this.emails.push({ to, subject, html });
    this.lastHtml = html;
    const match = html.match(/\/capacity\/form\/([^"']+)/);
    if (match) {
      this.lastToken = decodeURIComponent(match[1]);
    }
  }

  clear(): void {
    this.emails = [];
    this.lastHtml = undefined;
    this.lastToken = undefined;
  }
}

class FailingWebhookServer {
  private server: http.Server | null = null;
  private attempts: number = 0;
  private shouldFail: boolean = true;
  private failCount: number = 0;
  private statusCode: number = 500;
  private delayMs: number = 0;

  async start(options: {
    shouldFail?: boolean;
    failCount?: number;
    statusCode?: number;
    delayMs?: number;
  }): Promise<{ url: string; close: () => void; getAttempts: () => number }> {
    this.shouldFail = options.shouldFail ?? true;
    this.failCount = options.failCount ?? 3;
    this.statusCode = options.statusCode ?? 500;
    this.delayMs = options.delayMs ?? 0;
    this.attempts = 0;

    return new Promise((resolve) => {
      this.server = http.createServer((req, res) => {
        if (req.method !== 'POST') {
          res.statusCode = 405;
          res.end();
          return;
        }

        this.attempts++;
        let body = '';
        req.on('data', (chunk) => {
          body += chunk.toString();
        });
        req.on('end', () => {
          setTimeout(() => {
            if (this.shouldFail && this.attempts <= this.failCount) {
              res.statusCode = this.statusCode;
              res.end('error');
            } else {
              res.statusCode = 200;
              res.end('ok');
            }
          }, this.delayMs);
        });
      });

      this.server.listen(0, '127.0.0.1', () => {
        const address = this.server!.address() as AddressInfo;
        resolve({
          url: `http://localhost:${address.port}/webhook`,
          close: () => {
            if (this.server) {
              this.server.close();
              this.server = null;
            }
          },
          getAttempts: () => this.attempts,
        });
      });
    });
  }
}

function startServer(app: any): Promise<{ baseUrl: string; close: () => void }> {
  return new Promise((resolve) => {
    const server = app.listen(0, '127.0.0.1', () => {
      const address = server.address() as AddressInfo;
      const baseUrl = `http://localhost:${address.port}`;
      resolve({
        baseUrl,
        close: () => server.close(),
      });
    });
  });
}

function sleep(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

async function main() {
  // Test 1: Token TTL Configuration - Global
  await runTest('token TTL uses global configuration', async () => {
    process.env.PROVIDER_ADMIN_TOKEN = 'test-admin-token';
    
    const facilityStore = new InMemoryDocumentStore<FacilityProfile>('facility-profiles');
    const tokenStore = new InMemoryDocumentStore<CapacityRequestToken>('capacity-tokens');
    const facilityProfileService = new FacilityProfileService(facilityStore, {});
    const emailSender = new CaptureEmailSender();

    const globalTTL = 30; // 30 minutes
    const capacityRequestService = new CapacityRequestService({
      facilityProfileService,
      tokenStore,
      publicBaseUrl: 'http://localhost:0',
      tokenTTLMinutes: globalTTL,
      emailSender,
    });

    const api = new DataProviderAPI({ facilityProfileService, capacityRequestService });
    const server = await startServer(api.getApp());

    const facility: FacilityProfile = {
      id: 'facility-ttl-global',
      name: 'TTL Test Facility',
      email: 'ttl@example.com',
      source: 'file_price_list',
      lastUpdated: new Date(),
    };
    await facilityStore.put(facility.id, facility);
    const requestResponse = await fetch(`${server.baseUrl}/api/v1/capacity/request`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        Authorization: 'Bearer test-admin-token',
      },
      body: JSON.stringify({ facilityId: facility.id, channel: 'email' }),
    });

    const responseText = await requestResponse.text();
    if (requestResponse.status !== 200) {
      throw new Error(`Request failed with status ${requestResponse.status}: ${responseText}`);
    }
    assert.strictEqual(requestResponse.status, 200, 'Request should succeed');
    
    // Wait a bit for async email sending
    await sleep(50);
    assert.ok(emailSender.lastToken, `Token should be generated. Email sender state: ${JSON.stringify({ emails: emailSender.emails.length, lastHtml: emailSender.lastHtml?.substring(0, 100) })}`);
    const tokenRecord = await tokenStore.get(
      require('crypto').createHash('sha256').update(emailSender.lastToken).digest('hex')
    );
    assert.ok(tokenRecord, 'Token record should exist');
    
    const expiresAt = new Date(tokenRecord.expiresAt);
    const createdAt = new Date(tokenRecord.createdAt);
    const ttlMinutes = (expiresAt.getTime() - createdAt.getTime()) / (1000 * 60);
    
    // Allow 1 minute tolerance
    assert.ok(
      Math.abs(ttlMinutes - globalTTL) < 1,
      `TTL should be approximately ${globalTTL} minutes, got ${ttlMinutes}`
    );

    server.close();
  });

  // Test 2: Token TTL Configuration - Per-Facility Override
  await runTest('token TTL uses per-facility override', async () => {
    const facilityStore = new InMemoryDocumentStore<FacilityProfile>('facility-profiles');
    const tokenStore = new InMemoryDocumentStore<CapacityRequestToken>('capacity-tokens');
    const facilityProfileService = new FacilityProfileService(facilityStore, {});
    const emailSender = new CaptureEmailSender();

    const globalTTL = 60;
    const facilityTTL = 120; // Per-facility override
    const capacityRequestService = new CapacityRequestService({
      facilityProfileService,
      tokenStore,
      publicBaseUrl: 'http://localhost:0',
      tokenTTLMinutes: globalTTL,
      emailSender,
    });

    const api = new DataProviderAPI({ facilityProfileService, capacityRequestService });
    const server = await startServer(api.getApp());

    const facility: FacilityProfile = {
      id: 'facility-ttl-override',
      name: 'TTL Override Facility',
      email: 'ttl-override@example.com',
      source: 'file_price_list',
      lastUpdated: new Date(),
      metadata: {
        capacityTokenTTLMinutes: facilityTTL,
      },
    };
    await facilityStore.put(facility.id, facility);

    process.env.PROVIDER_ADMIN_TOKEN = 'test-admin-token';
    await fetch(`${server.baseUrl}/api/v1/capacity/request`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        Authorization: 'Bearer test-admin-token',
      },
      body: JSON.stringify({ facilityId: facility.id, channel: 'email' }),
    });

    assert.ok(emailSender.lastToken, 'Token should be generated');
    const tokenRecord = await tokenStore.get(
      require('crypto').createHash('sha256').update(emailSender.lastToken).digest('hex')
    );
    assert.ok(tokenRecord, 'Token record should exist');
    
    const expiresAt = new Date(tokenRecord.expiresAt);
    const createdAt = new Date(tokenRecord.createdAt);
    const ttlMinutes = (expiresAt.getTime() - createdAt.getTime()) / (1000 * 60);
    
    // Should use facility TTL, not global
    assert.ok(
      Math.abs(ttlMinutes - facilityTTL) < 1,
      `TTL should be approximately ${facilityTTL} minutes (facility override), got ${ttlMinutes}`
    );

    server.close();
  });

  // Test 3: Custom Email Template
  await runTest('custom email template with placeholders', async () => {
    const facilityStore = new InMemoryDocumentStore<FacilityProfile>('facility-profiles');
    const tokenStore = new InMemoryDocumentStore<CapacityRequestToken>('capacity-tokens');
    const facilityProfileService = new FacilityProfileService(facilityStore, {});
    const emailSender = new CaptureEmailSender();

    const customTemplate = (facilityName: string, link: string) => {
      return `<p>Custom template for ${facilityName}</p><p>Link: ${link}</p><p>Custom footer</p>`;
    };

    const capacityRequestService = new CapacityRequestService({
      facilityProfileService,
      tokenStore,
      publicBaseUrl: 'http://localhost:0',
      tokenTTLMinutes: 60,
      emailSender,
      emailTemplate: customTemplate,
    });

    const api = new DataProviderAPI({ facilityProfileService, capacityRequestService });
    const server = await startServer(api.getApp());

    const facility: FacilityProfile = {
      id: 'facility-custom-email',
      name: 'Custom Email Facility',
      email: 'custom@example.com',
      source: 'file_price_list',
      lastUpdated: new Date(),
    };
    await facilityStore.put(facility.id, facility);

    process.env.PROVIDER_ADMIN_TOKEN = 'test-admin-token';
    await fetch(`${server.baseUrl}/api/v1/capacity/request`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        Authorization: 'Bearer test-admin-token',
      },
      body: JSON.stringify({ facilityId: facility.id, channel: 'email' }),
    });

    assert.ok(emailSender.lastHtml, 'Email should be sent');
    assert.ok(
      emailSender.lastHtml.includes('Custom template for Custom Email Facility'),
      'Email should contain custom template content'
    );
    assert.ok(
      emailSender.lastHtml.includes('Custom footer'),
      'Email should contain custom footer'
    );
    assert.ok(emailSender.lastHtml.includes('Link:'), 'Email should contain link');

    server.close();
  });

  // Test 4: Capacity Status Validation - Invalid Value
  await runTest('capacity status validation rejects invalid values', async () => {
    const facilityStore = new InMemoryDocumentStore<FacilityProfile>('facility-profiles');
    const tokenStore = new InMemoryDocumentStore<CapacityRequestToken>('capacity-tokens');
    const facilityProfileService = new FacilityProfileService(facilityStore, {});
    const emailSender = new CaptureEmailSender();

    const capacityRequestService = new CapacityRequestService({
      facilityProfileService,
      tokenStore,
      publicBaseUrl: 'http://localhost:0',
      tokenTTLMinutes: 60,
      emailSender,
    });

    const api = new DataProviderAPI({ facilityProfileService, capacityRequestService });
    const server = await startServer(api.getApp());

    const facility: FacilityProfile = {
      id: 'facility-validation',
      name: 'Validation Test Facility',
      email: 'validation@example.com',
      source: 'file_price_list',
      lastUpdated: new Date(),
    };
    await facilityStore.put(facility.id, facility);

    // Generate token
    process.env.PROVIDER_ADMIN_TOKEN = 'test-admin-token';
    await fetch(`${server.baseUrl}/api/v1/capacity/request`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        Authorization: 'Bearer test-admin-token',
      },
      body: JSON.stringify({ facilityId: facility.id, channel: 'email' }),
    });

    assert.ok(emailSender.lastToken, 'Token should be generated');

    // Try to submit with invalid capacity status
    const submitResponse = await fetch(`${server.baseUrl}/api/v1/capacity/submit`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        token: emailSender.lastToken,
        capacityStatus: 'invalid-status',
        avgWaitMinutes: 30,
      }),
    });

    assert.strictEqual(submitResponse.status, 400, 'Should reject invalid capacity status');
    const responseText = await submitResponse.text();
    assert.ok(
      responseText.includes('Invalid Capacity Status'),
      'Error message should mention invalid status'
    );
    assert.ok(
      responseText.includes('available, busy, full, closed'),
      'Error message should list valid values'
    );

    // Verify facility was NOT updated
    const facilityAfter = await facilityStore.get(facility.id);
    assert.ok(facilityAfter, 'Facility should still exist');
    assert.strictEqual(
      facilityAfter!.capacityStatus,
      undefined,
      'Facility capacity status should not be updated'
    );

    server.close();
  });

  // Test 4.5: Token NOT consumed when payload validation fails
  await runTest('token is not consumed when capacity status validation fails', async () => {
    const facilityStore = new InMemoryDocumentStore<FacilityProfile>('facility-profiles');
    const tokenStore = new InMemoryDocumentStore<CapacityRequestToken>('capacity-tokens');
    const facilityProfileService = new FacilityProfileService(facilityStore, {});
    const emailSender = new CaptureEmailSender();

    const capacityRequestService = new CapacityRequestService({
      facilityProfileService,
      tokenStore,
      publicBaseUrl: 'http://localhost:0',
      tokenTTLMinutes: 60,
      emailSender,
    });

    const api = new DataProviderAPI({ facilityProfileService, capacityRequestService });
    const server = await startServer(api.getApp());

    const facility: FacilityProfile = {
      id: 'facility-token-not-consumed',
      name: 'Token Not Consumed Test',
      email: 'tokentest@example.com',
      source: 'file_price_list',
      lastUpdated: new Date(),
    };
    await facilityStore.put(facility.id, facility);

    // Generate token
    process.env.PROVIDER_ADMIN_TOKEN = 'test-admin-token';
    await fetch(`${server.baseUrl}/api/v1/capacity/request`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        Authorization: 'Bearer test-admin-token',
      },
      body: JSON.stringify({ facilityId: facility.id, channel: 'email' }),
    });

    assert.ok(emailSender.lastToken, 'Token should be generated');
    const token = emailSender.lastToken;

    // Step 1: Submit with INVALID capacity status - should fail without consuming token
    const invalidSubmit = await fetch(`${server.baseUrl}/api/v1/capacity/submit`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        token,
        capacityStatus: 'invalid-status',
        avgWaitMinutes: 30,
      }),
    });

    assert.strictEqual(invalidSubmit.status, 400, 'Should reject invalid capacity status');
    
    // Verify facility was NOT updated
    const facilityAfterInvalid = await facilityStore.get(facility.id);
    assert.strictEqual(
      facilityAfterInvalid!.capacityStatus,
      undefined,
      'Facility should not be updated after invalid submission'
    );

    // Step 2: Submit again with VALID capacity status - should succeed with same token
    const validSubmit = await fetch(`${server.baseUrl}/api/v1/capacity/submit`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        token, // Same token as before
        capacityStatus: 'busy',
        avgWaitMinutes: 45,
      }),
    });

    assert.strictEqual(validSubmit.status, 200, 'Should accept valid submission with same token');

    // Verify facility WAS updated
    const facilityAfterValid = await facilityStore.get(facility.id);
    assert.strictEqual(
      facilityAfterValid!.capacityStatus,
      'busy',
      'Facility should be updated after valid submission'
    );
    assert.strictEqual(
      facilityAfterValid!.avgWaitMinutes,
      45,
      'Wait time should be updated'
    );

    // Step 3: Try to submit AGAIN with the same token - should fail because token is now consumed
    const secondValidSubmit = await fetch(`${server.baseUrl}/api/v1/capacity/submit`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        token,
        capacityStatus: 'available',
        avgWaitMinutes: 10,
      }),
    });

    assert.strictEqual(secondValidSubmit.status, 400, 'Should reject already-used token');
    const responseText = await secondValidSubmit.text();
    assert.ok(
      responseText.includes('Token already used'),
      'Error message should indicate token was already used'
    );

    server.close();
  });

  // Test 5: Capacity Status Validation - Valid Values (Case Insensitive)
  await runTest('capacity status validation accepts valid values (case insensitive)', async () => {
    const facilityStore = new InMemoryDocumentStore<FacilityProfile>('facility-profiles');
    const tokenStore = new InMemoryDocumentStore<CapacityRequestToken>('capacity-tokens');
    const facilityProfileService = new FacilityProfileService(facilityStore, {});
    const emailSender = new CaptureEmailSender();

    const capacityRequestService = new CapacityRequestService({
      facilityProfileService,
      tokenStore,
      publicBaseUrl: 'http://localhost:0',
      tokenTTLMinutes: 60,
      emailSender,
    });

    const api = new DataProviderAPI({ facilityProfileService, capacityRequestService });
    const server = await startServer(api.getApp());

    const validStatuses = ['AVAILABLE', 'BUSY', 'Full', 'closed', 'Available'];
    
    for (const status of validStatuses) {
      emailSender.clear();
      
      const facility: FacilityProfile = {
        id: `facility-${status.toLowerCase()}`,
        name: `Test Facility ${status}`,
        email: `test-${status.toLowerCase()}@example.com`,
        source: 'file_price_list',
        lastUpdated: new Date(),
      };
      await facilityStore.put(facility.id, facility);

      // Generate token
      process.env.PROVIDER_ADMIN_TOKEN = 'test-admin-token';
      await fetch(`${server.baseUrl}/api/v1/capacity/request`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: 'Bearer test-admin-token',
        },
        body: JSON.stringify({ facilityId: facility.id, channel: 'email' }),
      });

      // Submit with uppercase/mixed case status
      const submitResponse = await fetch(`${server.baseUrl}/api/v1/capacity/submit`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          token: emailSender.lastToken,
          capacityStatus: status,
        }),
      });

      assert.strictEqual(submitResponse.status, 200, `Should accept ${status}`);
      
      const facilityAfter = await facilityStore.get(facility.id);
      assert.strictEqual(
        facilityAfter!.capacityStatus?.toLowerCase(),
        status.toLowerCase(),
        `Facility should be updated with ${status} (normalized to lowercase)`
      );
    }

    server.close();
  });

  // Test 6: Expired Token Rejection
  await runTest('expired token is rejected', async () => {
    const facilityStore = new InMemoryDocumentStore<FacilityProfile>('facility-profiles');
    const tokenStore = new InMemoryDocumentStore<CapacityRequestToken>('capacity-tokens');
    const facilityProfileService = new FacilityProfileService(facilityStore, {});
    const emailSender = new CaptureEmailSender();

    // Create token with very short TTL (1 minute)
    const capacityRequestService = new CapacityRequestService({
      facilityProfileService,
      tokenStore,
      publicBaseUrl: 'http://localhost:0',
      tokenTTLMinutes: 0.01, // 0.6 seconds
      emailSender,
    });

    const api = new DataProviderAPI({ facilityProfileService, capacityRequestService });
    const server = await startServer(api.getApp());

    const facility: FacilityProfile = {
      id: 'facility-expired',
      name: 'Expired Token Facility',
      email: 'expired@example.com',
      source: 'file_price_list',
      lastUpdated: new Date(),
    };
    await facilityStore.put(facility.id, facility);

    process.env.PROVIDER_ADMIN_TOKEN = 'test-admin-token';
    await fetch(`${server.baseUrl}/api/v1/capacity/request`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        Authorization: 'Bearer test-admin-token',
      },
      body: JSON.stringify({ facilityId: facility.id, channel: 'email' }),
    });

    assert.ok(emailSender.lastToken, 'Token should be generated');

    // Wait for token to expire
    await sleep(1000); // Wait 1 second (token expires in 0.6 seconds)

    // Try to use expired token
    const submitResponse = await fetch(`${server.baseUrl}/api/v1/capacity/submit`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        token: emailSender.lastToken,
        capacityStatus: 'available',
      }),
    });

    assert.strictEqual(submitResponse.status, 400, 'Should reject expired token');
    const responseText = await submitResponse.text();
    assert.ok(
      responseText.includes('Token expired') || responseText.includes('Invalid token'),
      'Error should mention token expiration'
    );

    server.close();
  });

  // Test 7: Token Reuse Prevention
  await runTest('token can only be used once', async () => {
    const facilityStore = new InMemoryDocumentStore<FacilityProfile>('facility-profiles');
    const tokenStore = new InMemoryDocumentStore<CapacityRequestToken>('capacity-tokens');
    const facilityProfileService = new FacilityProfileService(facilityStore, {});
    const emailSender = new CaptureEmailSender();

    const capacityRequestService = new CapacityRequestService({
      facilityProfileService,
      tokenStore,
      publicBaseUrl: 'http://localhost:0',
      tokenTTLMinutes: 60,
      emailSender,
    });

    const api = new DataProviderAPI({ facilityProfileService, capacityRequestService });
    const server = await startServer(api.getApp());

    const facility: FacilityProfile = {
      id: 'facility-reuse',
      name: 'Reuse Test Facility',
      email: 'reuse@example.com',
      source: 'file_price_list',
      lastUpdated: new Date(),
    };
    await facilityStore.put(facility.id, facility);

    process.env.PROVIDER_ADMIN_TOKEN = 'test-admin-token';
    await fetch(`${server.baseUrl}/api/v1/capacity/request`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        Authorization: 'Bearer test-admin-token',
      },
      body: JSON.stringify({ facilityId: facility.id, channel: 'email' }),
    });

    assert.ok(emailSender.lastToken, 'Token should be generated');

    // First use - should succeed
    const firstSubmit = await fetch(`${server.baseUrl}/api/v1/capacity/submit`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        token: emailSender.lastToken,
        capacityStatus: 'busy',
      }),
    });

    assert.strictEqual(firstSubmit.status, 200, 'First submission should succeed');

    // Second use - should fail
    const secondSubmit = await fetch(`${server.baseUrl}/api/v1/capacity/submit`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        token: emailSender.lastToken,
        capacityStatus: 'available',
      }),
    });

    assert.strictEqual(secondSubmit.status, 400, 'Second submission should fail');
    const responseText = await secondSubmit.text();
    assert.ok(
      responseText.includes('Token already used') || responseText.includes('Invalid token'),
      'Error should mention token already used'
    );

    // Verify facility was only updated once
    const facilityAfter = await facilityStore.get(facility.id);
    assert.strictEqual(
      facilityAfter!.capacityStatus,
      'busy',
      'Facility should still have first update value'
    );

    server.close();
  });

  // Test 8: Webhook Retry Mechanism
  await runTest('webhook retries on failure with exponential backoff', async () => {
    const facilityStore = new InMemoryDocumentStore<FacilityProfile>('facility-profiles');
    const tokenStore = new InMemoryDocumentStore<CapacityRequestToken>('capacity-tokens');
    const facilityProfileService = new FacilityProfileService(facilityStore, {});
    const emailSender = new CaptureEmailSender();

    const capacityRequestService = new CapacityRequestService({
      facilityProfileService,
      tokenStore,
      publicBaseUrl: 'http://localhost:0',
      tokenTTLMinutes: 60,
      emailSender,
    });

    const api = new DataProviderAPI({ facilityProfileService, capacityRequestService });
    const server = await startServer(api.getApp());

    // Configure webhook to fail 2 times then succeed
    const webhookServer = new FailingWebhookServer();
    const webhook = await webhookServer.start({
      shouldFail: true,
      failCount: 2,
      statusCode: 500,
      delayMs: 10, // Small delay for testing
    });

    process.env.PROVIDER_INGESTION_WEBHOOK_URL = webhook.url;
    process.env.PROVIDER_WEBHOOK_MAX_RETRIES = '3';
    process.env.PROVIDER_WEBHOOK_RETRY_DELAY_MS = '50'; // Short delay for testing
    process.env.PROVIDER_WEBHOOK_EXPONENTIAL_BACKOFF = 'true';

    const facility: FacilityProfile = {
      id: 'facility-webhook-retry',
      name: 'Webhook Retry Facility',
      email: 'webhook@example.com',
      source: 'file_price_list',
      lastUpdated: new Date(),
    };
    await facilityStore.put(facility.id, facility);

    // Generate token and submit
    process.env.PROVIDER_ADMIN_TOKEN = 'test-admin-token';
    await fetch(`${server.baseUrl}/api/v1/capacity/request`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        Authorization: 'Bearer test-admin-token',
      },
      body: JSON.stringify({ facilityId: facility.id, channel: 'email' }),
    });

    const submitResponse = await fetch(`${server.baseUrl}/api/v1/capacity/submit`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        token: emailSender.lastToken,
        capacityStatus: 'available',
      }),
    });

    assert.strictEqual(submitResponse.status, 200, 'Submission should succeed');

    // Wait for retries to complete (with some buffer)
    await sleep(500);

    // Should have attempted 3 times (initial + 2 retries)
    const attempts = webhook.getAttempts();
    assert.ok(attempts >= 3, `Should have retried at least 3 times, got ${attempts}`);

    webhook.close();
    server.close();
  });

  // Test 9: Webhook Doesn't Retry 4xx Errors
  await runTest('webhook does not retry 4xx client errors', async () => {
    const facilityStore = new InMemoryDocumentStore<FacilityProfile>('facility-profiles');
    const tokenStore = new InMemoryDocumentStore<CapacityRequestToken>('capacity-tokens');
    const facilityProfileService = new FacilityProfileService(facilityStore, {});
    const emailSender = new CaptureEmailSender();

    const capacityRequestService = new CapacityRequestService({
      facilityProfileService,
      tokenStore,
      publicBaseUrl: 'http://localhost:0',
      tokenTTLMinutes: 60,
      emailSender,
    });

    const api = new DataProviderAPI({ facilityProfileService, capacityRequestService });
    const server = await startServer(api.getApp());

    // Configure webhook to return 400 (client error)
    const webhookServer = new FailingWebhookServer();
    const webhook = await webhookServer.start({
      shouldFail: true,
      failCount: 10, // Many failures, but should not retry
      statusCode: 400, // Client error
      delayMs: 10,
    });

    process.env.PROVIDER_INGESTION_WEBHOOK_URL = webhook.url;
    process.env.PROVIDER_WEBHOOK_MAX_RETRIES = '3';

    const facility: FacilityProfile = {
      id: 'facility-webhook-4xx',
      name: 'Webhook 4xx Facility',
      email: 'webhook4xx@example.com',
      source: 'file_price_list',
      lastUpdated: new Date(),
    };
    await facilityStore.put(facility.id, facility);

    process.env.PROVIDER_ADMIN_TOKEN = 'test-admin-token';
    await fetch(`${server.baseUrl}/api/v1/capacity/request`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        Authorization: 'Bearer test-admin-token',
      },
      body: JSON.stringify({ facilityId: facility.id, channel: 'email' }),
    });

    const submitResponse = await fetch(`${server.baseUrl}/api/v1/capacity/submit`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        token: emailSender.lastToken,
        capacityStatus: 'available',
      }),
    });

    assert.strictEqual(submitResponse.status, 200, 'Submission should succeed');

    // Wait a bit
    await sleep(200);

    // Should only attempt once (no retries for 4xx)
    const attempts = webhook.getAttempts();
    assert.strictEqual(attempts, 1, `Should not retry 4xx errors, got ${attempts} attempts`);

    webhook.close();
    server.close();
  });

  // Test 10: Webhook Retries 429 Rate Limit
  await runTest('webhook retries 429 rate limit errors', async () => {
    const facilityStore = new InMemoryDocumentStore<FacilityProfile>('facility-profiles');
    const tokenStore = new InMemoryDocumentStore<CapacityRequestToken>('capacity-tokens');
    const facilityProfileService = new FacilityProfileService(facilityStore, {});
    const emailSender = new CaptureEmailSender();

    const capacityRequestService = new CapacityRequestService({
      facilityProfileService,
      tokenStore,
      publicBaseUrl: 'http://localhost:0',
      tokenTTLMinutes: 60,
      emailSender,
    });

    const api = new DataProviderAPI({ facilityProfileService, capacityRequestService });
    const server = await startServer(api.getApp());

    // Configure webhook to return 429 (rate limit) - should retry
    const webhookServer = new FailingWebhookServer();
    const webhook = await webhookServer.start({
      shouldFail: true,
      failCount: 2,
      statusCode: 429, // Rate limit - should retry
      delayMs: 10,
    });

    process.env.PROVIDER_INGESTION_WEBHOOK_URL = webhook.url;
    process.env.PROVIDER_WEBHOOK_MAX_RETRIES = '3';
    process.env.PROVIDER_WEBHOOK_RETRY_DELAY_MS = '50';

    const facility: FacilityProfile = {
      id: 'facility-webhook-429',
      name: 'Webhook 429 Facility',
      email: 'webhook429@example.com',
      source: 'file_price_list',
      lastUpdated: new Date(),
    };
    await facilityStore.put(facility.id, facility);

    process.env.PROVIDER_ADMIN_TOKEN = 'test-admin-token';
    await fetch(`${server.baseUrl}/api/v1/capacity/request`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        Authorization: 'Bearer test-admin-token',
      },
      body: JSON.stringify({ facilityId: facility.id, channel: 'email' }),
    });

    const submitResponse = await fetch(`${server.baseUrl}/api/v1/capacity/submit`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        token: emailSender.lastToken,
        capacityStatus: 'available',
      }),
    });

    assert.strictEqual(submitResponse.status, 200, 'Submission should succeed');

    // Wait for retries
    await sleep(500);

    // Should have retried (429 is exception to 4xx no-retry rule)
    const attempts = webhook.getAttempts();
    assert.ok(attempts >= 2, `Should retry 429 errors, got ${attempts} attempts`);

    webhook.close();
    server.close();
  });

  if (process.exitCode) {
    process.exit(1);
  }
}

main().catch((err) => {
  console.error(err);
  process.exit(1);
});
