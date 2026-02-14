import assert from 'assert';
import http from 'http';
import { AddressInfo } from 'net';
import { DataProviderAPI } from '../../api/server';
import { FacilityProfileService } from '../../ingestion/facilityProfileService';
import {
  CapacityRequestService,
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

function sleep(ms: number): Promise<void> {
  return new Promise(resolve => setTimeout(resolve, ms));
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

function startServer(app: any): Promise<{ server: http.Server; baseUrl: string; close: () => void }> {
  return new Promise((resolve, reject) => {
    const server = app.listen(0, '127.0.0.1', () => {
      const address = server.address() as AddressInfo;
      const baseUrl = `http://localhost:${address.port}`;
      resolve({
        server,
        baseUrl,
        close: () => {
          server.close();
        },
      });
    });
    server.on('error', reject);
  });
}

async function main() {
  console.log('\n🧪 Testing Ward-Level Capacity Update Flow\n');

  await runTest('ward capacity update - complete flow', async () => {
    // Setup
    process.env.PROVIDER_ADMIN_TOKEN = 'test-admin-token';

    const facilityStore = new InMemoryDocumentStore<FacilityProfile>('facility-profiles');
    const tokenStore = new InMemoryDocumentStore<any>('capacity-tokens');
    const facilityProfileService = new FacilityProfileService(facilityStore, {});
    const emailSender = new CaptureEmailSender();

    const capacityRequestService = new CapacityRequestService({
      facilityProfileService,
      tokenStore,
      publicBaseUrl: 'http://localhost:0',
      tokenTTLMinutes: 120,
      emailSender,
    });

    const api = new DataProviderAPI({ facilityProfileService, capacityRequestService });
    const server = await startServer(api.getApp());

    // Create test facility
    const facility: FacilityProfile = {
      id: 'facility-ward-test',
      name: 'Ward Test Hospital',
      email: 'ward-test@example.com',
      source: 'file_price_list',
      lastUpdated: new Date(),
      metadata: {
        availableWards: ['maternity', 'pharmacy', 'inpatient'],
      },
    };
    await facilityStore.put(facility.id, facility);

    // Step 1: Request capacity update for specific ward
    const requestResponse = await fetch(`${server.baseUrl}/api/v1/capacity/request`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        Authorization: 'Bearer test-admin-token',
      },
      body: JSON.stringify({
        facilityId: facility.id,
        channel: 'email',
        wardName: 'maternity', // Request ward-specific update
      }),
    });

    assert.strictEqual(requestResponse.status, 200, 'Request should succeed');
    await sleep(50); // Wait for async email sending

    assert.ok(emailSender.lastToken, 'Token should be generated');
    const token = emailSender.lastToken!;

    // Step 2: Access form with ward pre-selected
    const formResponse = await fetch(`${server.baseUrl}/api/v1/capacity/form/${token}`);
    assert.strictEqual(formResponse.status, 200, 'Form should be accessible');
    const formHtml = await formResponse.text();
    assert.ok(formHtml.includes('maternity'), 'Form should show maternity ward');

    // Step 3: Submit ward-specific capacity update
    const formData = new URLSearchParams();
    formData.append('token', token);
    formData.append('wardName', 'maternity');
    formData.append('capacityStatus', 'busy');
    formData.append('avgWaitMinutes', '45');
    formData.append('urgentCareAvailable', 'true');

    const submitResponse = await fetch(`${server.baseUrl}/api/v1/capacity/submit`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/x-www-form-urlencoded',
      },
      body: formData.toString(),
    });

    assert.strictEqual(submitResponse.status, 200, 'Submission should succeed');
    const submitHtml = await submitResponse.text();
    assert.ok(submitHtml.includes('success') || submitHtml.includes('Success'), 'Should show success message');

    // Step 4: Verify ward capacity was stored
    const updatedFacility = await facilityProfileService.getProfile(facility.id);
    assert.ok(updatedFacility, 'Facility should exist');
    if (!updatedFacility) throw new Error('Facility not found');
    
    assert.ok(updatedFacility.wards, 'Facility should have wards array');
    assert.strictEqual(updatedFacility.wards!.length, 1, 'Should have one ward');
    
    const maternityWard = updatedFacility.wards!.find(w => w.wardName.toLowerCase() === 'maternity');
    assert.ok(maternityWard, 'Maternity ward should exist');
    assert.strictEqual(maternityWard.capacityStatus, 'busy', 'Capacity status should be busy');
    assert.strictEqual(maternityWard.avgWaitMinutes, 45, 'Wait time should be 45 minutes');
    assert.strictEqual(maternityWard.urgentCareAvailable, true, 'Urgent care should be available');

    server.close();
  });

  await runTest('ward capacity update - multiple wards', async () => {
    process.env.PROVIDER_ADMIN_TOKEN = 'test-admin-token';

    const facilityStore = new InMemoryDocumentStore<FacilityProfile>('facility-profiles');
    const tokenStore = new InMemoryDocumentStore<any>('capacity-tokens');
    const facilityProfileService = new FacilityProfileService(facilityStore, {});
    const emailSender = new CaptureEmailSender();

    const capacityRequestService = new CapacityRequestService({
      facilityProfileService,
      tokenStore,
      publicBaseUrl: 'http://localhost:0',
      tokenTTLMinutes: 120,
      emailSender,
    });

    const api = new DataProviderAPI({ facilityProfileService, capacityRequestService });
    const server = await startServer(api.getApp());

    const facility: FacilityProfile = {
      id: 'facility-multi-ward',
      name: 'Multi-Ward Hospital',
      email: 'multi-ward@example.com',
      source: 'file_price_list',
      lastUpdated: new Date(),
    };
    await facilityStore.put(facility.id, facility);

    // Update maternity ward
    emailSender.clear();
    await fetch(`${server.baseUrl}/api/v1/capacity/request`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        Authorization: 'Bearer test-admin-token',
      },
      body: JSON.stringify({
        facilityId: facility.id,
        channel: 'email',
        wardName: 'maternity',
      }),
    });
    await sleep(50);

    const token1 = emailSender.lastToken!;
    const formData1 = new URLSearchParams();
    formData1.append('token', token1);
    formData1.append('wardName', 'maternity');
    formData1.append('capacityStatus', 'available');
    formData1.append('avgWaitMinutes', '30');

    await fetch(`${server.baseUrl}/api/v1/capacity/submit`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
      body: formData1.toString(),
    });

    // Update pharmacy ward
    emailSender.clear();
    await fetch(`${server.baseUrl}/api/v1/capacity/request`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        Authorization: 'Bearer test-admin-token',
      },
      body: JSON.stringify({
        facilityId: facility.id,
        channel: 'email',
        wardName: 'pharmacy',
      }),
    });
    await sleep(50);

    const token2 = emailSender.lastToken!;
    const formData2 = new URLSearchParams();
    formData2.append('token', token2);
    formData2.append('wardName', 'pharmacy');
    formData2.append('capacityStatus', 'busy');
    formData2.append('avgWaitMinutes', '15');

    await fetch(`${server.baseUrl}/api/v1/capacity/submit`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
      body: formData2.toString(),
    });

    // Verify both wards exist
    const updatedFacility = await facilityProfileService.getProfile(facility.id);
    assert.ok(updatedFacility, 'Facility should exist');
    if (!updatedFacility) throw new Error('Facility not found');
    
    assert.ok(updatedFacility.wards, 'Facility should have wards');
    assert.strictEqual(updatedFacility.wards!.length, 2, 'Should have two wards');

    const maternityWard = updatedFacility.wards!.find(w => w.wardName.toLowerCase() === 'maternity');
    const pharmacyWard = updatedFacility.wards!.find(w => w.wardName.toLowerCase() === 'pharmacy');

    assert.ok(maternityWard, 'Maternity ward should exist');
    assert.ok(pharmacyWard, 'Pharmacy ward should exist');
    assert.strictEqual(maternityWard.capacityStatus, 'available', 'Maternity should be available');
    assert.strictEqual(pharmacyWard.capacityStatus, 'busy', 'Pharmacy should be busy');

    server.close();
  });

  await runTest('ward capacity update - facility-wide fallback', async () => {
    process.env.PROVIDER_ADMIN_TOKEN = 'test-admin-token';

    const facilityStore = new InMemoryDocumentStore<FacilityProfile>('facility-profiles');
    const tokenStore = new InMemoryDocumentStore<any>('capacity-tokens');
    const facilityProfileService = new FacilityProfileService(facilityStore, {});
    const emailSender = new CaptureEmailSender();

    const capacityRequestService = new CapacityRequestService({
      facilityProfileService,
      tokenStore,
      publicBaseUrl: 'http://localhost:0',
      tokenTTLMinutes: 120,
      emailSender,
    });

    const api = new DataProviderAPI({ facilityProfileService, capacityRequestService });
    const server = await startServer(api.getApp());

    const facility: FacilityProfile = {
      id: 'facility-fallback',
      name: 'Fallback Hospital',
      email: 'fallback@example.com',
      source: 'file_price_list',
      lastUpdated: new Date(),
    };
    await facilityStore.put(facility.id, facility);

    // Request without ward name (facility-wide)
    await fetch(`${server.baseUrl}/api/v1/capacity/request`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        Authorization: 'Bearer test-admin-token',
      },
      body: JSON.stringify({
        facilityId: facility.id,
        channel: 'email',
      }),
    });
    await sleep(50);

    const token = emailSender.lastToken!;

    // Submit without ward name
    const formData = new URLSearchParams();
    formData.append('token', token);
    formData.append('capacityStatus', 'available');
    formData.append('avgWaitMinutes', '20');

    await fetch(`${server.baseUrl}/api/v1/capacity/submit`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
      body: formData.toString(),
    });

    // Verify facility-wide capacity was updated (not ward-specific)
    const updatedFacility = await facilityProfileService.getProfile(facility.id);
    assert.ok(updatedFacility, 'Facility should exist');
    if (!updatedFacility) throw new Error('Facility not found');
    
    assert.strictEqual(updatedFacility.capacityStatus, 'available', 'Facility-wide capacity should be updated');
    assert.strictEqual(updatedFacility.avgWaitMinutes, 20, 'Facility-wide wait time should be updated');
    
    // Should not have wards array or it should be empty
    if (updatedFacility.wards) {
      assert.strictEqual(updatedFacility.wards.length, 0, 'Should not have ward-specific updates');
    }

    server.close();
  });

  await runTest('ward capacity update - custom ward name', async () => {
    process.env.PROVIDER_ADMIN_TOKEN = 'test-admin-token';

    const facilityStore = new InMemoryDocumentStore<FacilityProfile>('facility-profiles');
    const tokenStore = new InMemoryDocumentStore<any>('capacity-tokens');
    const facilityProfileService = new FacilityProfileService(facilityStore, {});
    const emailSender = new CaptureEmailSender();

    const capacityRequestService = new CapacityRequestService({
      facilityProfileService,
      tokenStore,
      publicBaseUrl: 'http://localhost:0',
      tokenTTLMinutes: 120,
      emailSender,
    });

    const api = new DataProviderAPI({ facilityProfileService, capacityRequestService });
    const server = await startServer(api.getApp());

    const facility: FacilityProfile = {
      id: 'facility-custom-ward',
      name: 'Custom Ward Hospital',
      email: 'custom@example.com',
      source: 'file_price_list',
      lastUpdated: new Date(),
    };
    await facilityStore.put(facility.id, facility);

    await fetch(`${server.baseUrl}/api/v1/capacity/request`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        Authorization: 'Bearer test-admin-token',
      },
      body: JSON.stringify({
        facilityId: facility.id,
        channel: 'email',
      }),
    });
    await sleep(50);

    const token = emailSender.lastToken!;

    // Submit with custom ward name
    const formData = new URLSearchParams();
    formData.append('token', token);
    formData.append('wardName', 'Cardiac Care Unit');
    formData.append('capacityStatus', 'full');
    formData.append('avgWaitMinutes', '60');

    await fetch(`${server.baseUrl}/api/v1/capacity/submit`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
      body: formData.toString(),
    });

    // Verify custom ward was created
    const updatedFacility = await facilityProfileService.getProfile(facility.id);
    assert.ok(updatedFacility, 'Facility should exist');
    if (!updatedFacility) throw new Error('Facility not found');
    
    assert.ok(updatedFacility.wards, 'Facility should have wards');
    const customWard = updatedFacility.wards!.find(w => w.wardName === 'Cardiac Care Unit');
    assert.ok(customWard, 'Custom ward should exist');
    assert.strictEqual(customWard.capacityStatus, 'full', 'Custom ward should be full');

    server.close();
  });

  await runTest('ward capacity update - invalid ward status validation', async () => {
    process.env.PROVIDER_ADMIN_TOKEN = 'test-admin-token';

    const facilityStore = new InMemoryDocumentStore<FacilityProfile>('facility-profiles');
    const tokenStore = new InMemoryDocumentStore<any>('capacity-tokens');
    const facilityProfileService = new FacilityProfileService(facilityStore, {});
    const emailSender = new CaptureEmailSender();

    const capacityRequestService = new CapacityRequestService({
      facilityProfileService,
      tokenStore,
      publicBaseUrl: 'http://localhost:0',
      tokenTTLMinutes: 120,
      emailSender,
    });

    const api = new DataProviderAPI({ facilityProfileService, capacityRequestService });
    const server = await startServer(api.getApp());

    const facility: FacilityProfile = {
      id: 'facility-validation',
      name: 'Validation Hospital',
      email: 'validation@example.com',
      source: 'file_price_list',
      lastUpdated: new Date(),
    };
    await facilityStore.put(facility.id, facility);

    await fetch(`${server.baseUrl}/api/v1/capacity/request`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        Authorization: 'Bearer test-admin-token',
      },
      body: JSON.stringify({
        facilityId: facility.id,
        channel: 'email',
      }),
    });
    await sleep(50);

    const token = emailSender.lastToken!;

    // Try to submit with invalid capacity status
    const formData = new URLSearchParams();
    formData.append('token', token);
    formData.append('wardName', 'maternity');
    formData.append('capacityStatus', 'invalid-status');

    const submitResponse = await fetch(`${server.baseUrl}/api/v1/capacity/submit`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
      body: formData.toString(),
    });

    assert.strictEqual(submitResponse.status, 400, 'Should reject invalid status');
    const errorHtml = await submitResponse.text();
    assert.ok(errorHtml.includes('Invalid') || errorHtml.includes('invalid'), 'Should show error message');

    server.close();
  });

  console.log('\n✅ All ward-level capacity tests completed!\n');
}

main().catch((err) => {
  console.error('Test suite failed:', err);
  process.exit(1);
});
