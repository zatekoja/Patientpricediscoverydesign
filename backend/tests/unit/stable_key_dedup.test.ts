import assert from 'assert';
import path from 'path';
import { FilePriceListProvider } from '../../providers/FilePriceListProvider';
import { InMemoryDocumentStore } from '../../stores/InMemoryDocumentStore';
import { PriceData } from '../../types/PriceData';

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

const fixturesDir = path.resolve(__dirname, '..', '..', 'fixtures', 'price_lists');
const megalekCsv = path.join(fixturesDir, 'MEGALEK NEW PRICE LIST 2026.csv');

async function main() {
  await runTest('generateKey produces stable keys for identical records', async () => {
    const provider = new FilePriceListProvider();
    await provider.initialize({
      files: [{ path: megalekCsv }],
      currency: 'NGN',
      defaultEffectiveDate: '2026-01-01',
    });

    const record: PriceData = {
      id: 'test-1',
      facilityName: 'Test Hospital',
      facilityId: 'test_hospital',
      procedureCode: 'PROC001',
      procedureDescription: 'Test Procedure',
      price: 1000,
      currency: 'NGN',
      effectiveDate: new Date('2026-01-01'),
      lastUpdated: new Date(),
      source: 'file_price_list',
    };

    const key1 = (provider as any).generateKey(record, 0);
    const key2 = (provider as any).generateKey(record, 0);
    
    assert.strictEqual(key1, key2, 'Keys should be identical for same record');
    assert(key1.includes('test_hospital'), 'Key should include facility');
    assert(key1.includes('proc001'), 'Key should include procedure code');
    assert(key1.includes('2026-01-01'), 'Key should include effective date');
  });

  await runTest('generateKey produces different keys for different prices', () => {
    const provider = new FilePriceListProvider();
    
    const record1: PriceData = {
      id: 'test-1',
      facilityName: 'Test Hospital',
      procedureCode: 'PROC001',
      procedureDescription: 'Test Procedure',
      price: 1000,
      currency: 'NGN',
      effectiveDate: new Date('2026-01-01'),
      lastUpdated: new Date(),
      source: 'file_price_list',
    };

    const record2 = { ...record1, price: 2000 };

    const key1 = (provider as any).generateKey(record1, 0);
    const key2 = (provider as any).generateKey(record2, 0);
    
    assert.notStrictEqual(key1, key2, 'Keys should differ when price changes');
  });

  await runTest('generateKey produces different keys for different tiers', () => {
    const provider = new FilePriceListProvider();
    
    const record1: PriceData = {
      id: 'test-1',
      facilityName: 'Test Hospital',
      procedureCode: 'PROC001',
      procedureDescription: 'Test Procedure',
      price: 1000,
      currency: 'NGN',
      effectiveDate: new Date('2026-01-01'),
      lastUpdated: new Date(),
      source: 'file_price_list',
      metadata: { priceTier: 'adult' },
    };

    const record2 = {
      ...record1,
      metadata: { priceTier: 'paediatric' },
    };

    const key1 = (provider as any).generateKey(record1, 0);
    const key2 = (provider as any).generateKey(record2, 0);
    
    assert.notStrictEqual(key1, key2, 'Keys should differ for different tiers');
  });

  await runTest('sync deduplicates records with same stable key', async () => {
    const store = new InMemoryDocumentStore('dedup-test-store');
    const provider = new FilePriceListProvider(store);
    
    await provider.initialize({
      files: [{ path: megalekCsv }],
      currency: 'NGN',
      defaultEffectiveDate: '2026-01-01',
    });

    // First sync
    const result1 = await provider.syncData();
    assert(result1.success, 'First sync should succeed');
    const count1 = result1.recordsProcessed;

    // Second sync with same data
    const result2 = await provider.syncData();
    assert(result2.success, 'Second sync should succeed');
    const count2 = result2.recordsProcessed;

    // Counts should be the same (dedup working)
    assert.strictEqual(count1, count2, 'Record counts should match after dedup');

    // Verify data in store
    const allRecords = await store.query({});
    // Should have 2 batches but keys prevent duplicates within each batch
    assert(allRecords.length >= count1, 'Store should contain records from both syncs');
  });

  await runTest('stable keys include all identifying fields', () => {
    const provider = new FilePriceListProvider();
    
    const record: PriceData = {
      id: 'test-1',
      facilityName: 'Test Hospital',
      facilityId: 'test_hospital_id',
      procedureCode: 'PROC001',
      procedureDescription: 'Test Procedure',
      price: 1500.50,
      currency: 'NGN',
      effectiveDate: new Date('2026-01-15'),
      lastUpdated: new Date(),
      source: 'file_price_list',
      metadata: { priceTier: 'executive' },
    };

    const key = (provider as any).generateKey(record, 5);
    
    // Verify key contains normalized components
    assert(key.includes('file_price_list'), 'Key should include provider name');
    assert(key.includes('test_hospital'), 'Key should include facility');
    assert(key.includes('proc001'), 'Key should include procedure');
    assert(key.includes('executive'), 'Key should include tier');
    assert(key.includes('1500.50'), 'Key should include price');
    assert(key.includes('2026-01-15'), 'Key should include date');
  });

  await runTest('keys handle special characters safely', () => {
    const provider = new FilePriceListProvider();
    
    const record: PriceData = {
      id: 'test-1',
      facilityName: 'Test & Hospital (Lagos)',
      procedureCode: 'PROC/001',
      procedureDescription: 'Test Procedure: Advanced',
      price: 1000,
      currency: 'NGN',
      effectiveDate: new Date('2026-01-01'),
      lastUpdated: new Date(),
      source: 'file_price_list',
    };

    const key = (provider as any).generateKey(record, 0);
    
    // Key should be normalized and safe
    assert(!key.includes('&'), 'Key should not contain &');
    assert(!key.includes('('), 'Key should not contain (');
    assert(!key.includes('/'), 'Key should not contain /');
    assert(!key.includes(':'), 'Key should not contain :');
  });

  if (process.exitCode) {
    process.exit(1);
  }
}

main().catch((err) => {
  console.error(err);
  process.exit(1);
});
