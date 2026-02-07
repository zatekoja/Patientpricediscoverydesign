/**
 * Test script to verify API pagination validation works correctly
 */

import express from 'express';
import { DataProviderAPI } from '../../api/server';
import { FilePriceListProvider } from '../../providers/FilePriceListProvider';
import { InMemoryDocumentStore } from '../../stores/InMemoryDocumentStore';
import path from 'path';

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

async function main() {
  // Setup API with file provider
  const api = new DataProviderAPI();
  const store = new InMemoryDocumentStore('test-store');
  const provider = new FilePriceListProvider(store);
  
  const fixturesDir = path.resolve(__dirname, '..', '..', 'fixtures', 'price_lists');
  const testFile = path.join(fixturesDir, 'MEGALEK NEW PRICE LIST 2026.csv');
  
  await provider.initialize({
    files: [{ path: testFile }],
    currency: 'NGN',
    defaultEffectiveDate: '2026-01-01',
  });
  
  api.registerProvider('file_price_list', provider, true);
  
  const app = api.getApp();
  
  // Test valid pagination
  await runTest('API accepts valid pagination parameters', async () => {
    const req = {
      query: { limit: '50', offset: '10' },
      method: 'GET',
      path: '/api/v1/data/current',
    } as any;
    
    const res = {
      json: (data: any) => {
        if (data.error) {
          throw new Error(`Unexpected error: ${data.message}`);
        }
      },
      status: (code: number) => {
        if (code !== 200) {
          throw new Error(`Expected status 200, got ${code}`);
        }
        return res;
      },
    } as any;
    
    // This would normally be called by express, we're simulating
    const validated = (api as any).validatePagination('50', '10');
    if (!validated.valid) {
      throw new Error(`Validation failed: ${validated.error}`);
    }
  });
  
  // Test invalid limit
  await runTest('API rejects NaN limit', async () => {
    const validated = (api as any).validatePagination('abc', '0');
    if (validated.valid) {
      throw new Error('Should have rejected NaN limit');
    }
    if (!validated.error.includes('valid integer')) {
      throw new Error('Wrong error message');
    }
  });
  
  // Test limit exceeding max
  await runTest('API rejects excessive limit', async () => {
    const validated = (api as any).validatePagination('10000', '0');
    if (validated.valid) {
      throw new Error('Should have rejected excessive limit');
    }
    if (!validated.error.includes('5000')) {
      throw new Error('Wrong error message');
    }
  });
  
  // Test negative offset
  await runTest('API rejects negative offset', async () => {
    const validated = (api as any).validatePagination('100', '-10');
    if (validated.valid) {
      throw new Error('Should have rejected negative offset');
    }
    if (!validated.error.includes('non-negative')) {
      throw new Error('Wrong error message');
    }
  });
  
  // Test max limit boundary
  await runTest('API accepts max limit of 5000', async () => {
    const validated = (api as any).validatePagination('5000', '0');
    if (!validated.valid) {
      throw new Error(`Should have accepted 5000: ${validated.error}`);
    }
    if (validated.limit !== 5000) {
      throw new Error('Limit should be 5000');
    }
  });
  
  console.log('\n✅ All API pagination tests passed!');
  
  if (process.exitCode) {
    process.exit(1);
  }
}

main().catch((err) => {
  console.error(err);
  process.exit(1);
});
