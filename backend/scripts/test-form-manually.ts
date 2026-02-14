#!/usr/bin/env ts-node
/**
 * Script to generate a test token and open the capacity form in browser
 * 
 * Usage:
 *   ts-node scripts/test-form-manually.ts [facilityId]
 * 
 * Example:
 *   ts-node scripts/test-form-manually.ts facility-test-1
 */

import { DataProviderAPI } from '../api/server';
import { FacilityProfileService } from '../ingestion/facilityProfileService';
import { CapacityRequestService, EmailSender } from '../ingestion/capacityRequestService';
import { InMemoryDocumentStore } from '../stores/InMemoryDocumentStore';
import { FacilityProfile } from '../types/FacilityProfile';
import { CapacityRequestToken } from '../ingestion/capacityRequestService';
import http from 'http';
import { AddressInfo } from 'net';

class CaptureEmailSender implements EmailSender {
  lastToken?: string;
  lastHtml?: string;

  async send(_to: string, _subject: string, html: string): Promise<void> {
    this.lastHtml = html;
    const match = html.match(/\/capacity\/form\/([^"']+)/);
    if (match) {
      this.lastToken = decodeURIComponent(match[1]);
    }
    console.log('📧 Email would be sent with form link');
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

async function main() {
  const facilityId = process.argv[2] || 'facility-test-1';
  
  console.log('🚀 Setting up test environment...');
  console.log(`   Facility ID: ${facilityId}`);
  console.log('');

  // Setup stores
  const facilityStore = new InMemoryDocumentStore<FacilityProfile>('facility-profiles');
  const tokenStore = new InMemoryDocumentStore<CapacityRequestToken>('capacity-tokens');
  const facilityProfileService = new FacilityProfileService(facilityStore, {});
  const emailSender = new CaptureEmailSender();

  // Create test facility if it doesn't exist
  let facility = await facilityStore.get(facilityId);
  if (!facility) {
    console.log(`📝 Creating test facility: ${facilityId}`);
    facility = {
      id: facilityId,
      name: 'Test Facility',
      email: 'test@example.com',
      source: 'file_price_list',
      lastUpdated: new Date(),
    };
    await facilityStore.put(facilityId, facility);
  }

  // Setup capacity request service
  const capacityRequestService = new CapacityRequestService({
    facilityProfileService,
    tokenStore,
    publicBaseUrl: 'http://localhost:3001', // Default provider API port
    tokenTTLMinutes: 120,
    emailSender,
  });

  // Setup API
  process.env.PROVIDER_ADMIN_TOKEN = 'test-admin-token';
  const api = new DataProviderAPI({ facilityProfileService, capacityRequestService });
  const server = await startServer(api.getApp());

  console.log(`✅ Server started on ${server.baseUrl}`);
  console.log('');

  // Generate token
  console.log('🔑 Generating capacity request token...');
  try {
    const response = await fetch(`${server.baseUrl}/api/v1/capacity/request`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        Authorization: 'Bearer test-admin-token',
      },
      body: JSON.stringify({ facilityId, channel: 'email' }),
    });

    if (!response.ok) {
      const text = await response.text();
      throw new Error(`Failed to generate token: ${response.status} - ${text}`);
    }

    // Wait a bit for async email sending
    await new Promise(resolve => setTimeout(resolve, 100));

    if (!emailSender.lastToken) {
      throw new Error('Token was not generated. Check server logs.');
    }

    const formUrl = `${server.baseUrl}/api/v1/capacity/form/${encodeURIComponent(emailSender.lastToken)}`;
    
    console.log('✅ Token generated successfully!');
    console.log('');
    console.log('━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━');
    console.log('📋 CAPACITY UPDATE FORM URL:');
    console.log('');
    console.log(`   ${formUrl}`);
    console.log('');
    console.log('━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━');
    console.log('');
    console.log('🌐 To test the form:');
    console.log('   1. Copy the URL above');
    console.log('   2. Open it in your browser');
    console.log('   3. Fill out and submit the form');
    console.log('');
    console.log('💡 Server will keep running. Press Ctrl+C to stop.');
    console.log('');

    // Try to open in browser using platform-specific command (no dependencies needed)
    const platform = process.platform;
    let openCommand: string;
    
    if (platform === 'darwin') {
      openCommand = 'open';
    } else if (platform === 'win32') {
      openCommand = 'start';
    } else {
      openCommand = 'xdg-open';
    }
    
    try {
      const { exec } = await import('child_process');
      const { promisify } = await import('util');
      const execAsync = promisify(exec);
      console.log('🔗 Opening form in browser...');
      await execAsync(`${openCommand} "${formUrl}"`);
    } catch (error) {
      // Browser open failed, just show URL (already shown above)
    }

    // Keep server running
    process.on('SIGINT', () => {
      console.log('\n👋 Shutting down server...');
      server.close();
      process.exit(0);
    });

    // Keep process alive
    await new Promise(() => {});

  } catch (error) {
    console.error('❌ Error:', error);
    server.close();
    process.exit(1);
  }
}

main().catch((error) => {
  console.error('Fatal error:', error);
  process.exit(1);
});
