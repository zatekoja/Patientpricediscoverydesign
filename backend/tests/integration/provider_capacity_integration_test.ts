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
  lastHtml?: string;
  lastToken?: string;

  async send(_to: string, _subject: string, html: string): Promise<void> {
    this.lastHtml = html;
    const match = html.match(/\/capacity\/form\/([^"']+)/);
    if (match) {
      this.lastToken = decodeURIComponent(match[1]);
    }
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

function startWebhookServer(): Promise<{
  url: string;
  close: () => void;
  waitFor: Promise<any>;
}> {
  return new Promise((resolve) => {
    let resolvePayload: (value: any) => void;
    const waitFor = new Promise((res) => {
      resolvePayload = res;
    });
    const server = http.createServer((req, res) => {
      if (req.method !== 'POST') {
        res.statusCode = 405;
        res.end();
        return;
      }
      let body = '';
      req.on('data', (chunk) => {
        body += chunk.toString();
      });
      req.on('end', () => {
        try {
          const payload = JSON.parse(body || '{}');
          resolvePayload(payload);
          res.statusCode = 200;
          res.end('ok');
        } catch (err) {
          res.statusCode = 400;
          res.end('bad');
        }
      });
    });
    server.listen(0, '127.0.0.1', () => {
      const address = server.address() as AddressInfo;
      resolve({
        url: `http://localhost:${address.port}/webhook`,
        close: () => server.close(),
        waitFor,
      });
    });
  });
}

async function main() {
  await runTest('capacity request flow triggers email + webhook', async () => {
    process.env.PROVIDER_ADMIN_TOKEN = 'test-admin-token';

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

    const webhook = await startWebhookServer();
    process.env.PROVIDER_INGESTION_WEBHOOK_URL = webhook.url;

    const facility: FacilityProfile = {
      id: 'facility-test-1',
      name: 'Test Facility',
      email: 'test@example.com',
      source: 'file_price_list',
      lastUpdated: new Date(),
    };
    await facilityStore.put(facility.id, facility);

    const adminResponse = await fetch(`${server.baseUrl}/api/v1/capacity/request`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${process.env.PROVIDER_ADMIN_TOKEN}`,
      },
      body: JSON.stringify({ facilityId: facility.id, channel: 'email' }),
    });
    assert.strictEqual(adminResponse.status, 200);
    assert.ok(emailSender.lastToken, 'expected email token');

    const submitResponse = await fetch(`${server.baseUrl}/api/v1/capacity/submit`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        token: emailSender.lastToken,
        capacityStatus: 'busy',
        avgWaitMinutes: 45,
        urgentCareAvailable: true,
      }),
    });
    assert.strictEqual(submitResponse.status, 200);

    const updated = await facilityStore.get(facility.id);
    assert(updated, 'expected facility update');
    assert.strictEqual(updated!.capacityStatus, 'busy');
    assert.strictEqual(updated!.avgWaitMinutes, 45);
    assert.strictEqual(updated!.urgentCareAvailable, true);

    const webhookPayload = await webhook.waitFor;
    assert.strictEqual(webhookPayload.facilityId, facility.id);
    assert.ok(webhookPayload.eventId, 'expected eventId in webhook payload');

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
