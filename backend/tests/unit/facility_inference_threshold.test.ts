import assert from 'assert';
import fs from 'fs';
import os from 'os';
import path from 'path';
import { parseCsvFile } from '../../ingestion/priceListParser';

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

const tempDir = fs.mkdtempSync(path.join(os.tmpdir(), 'ppd-facility-test-'));
const fileName = 'My_Facility_Test.csv';
const filePath = path.join(tempDir, fileName);
const officeUseFile = 'PRICE_LIST_FOR_OFFICE_USE[1].csv';
const officeUsePath = path.join(tempDir, officeUseFile);

const csvContent = [
  'General Hospital Lagos',
  'Description,Price',
  'CT Scan,15000',
].join('\n');

async function main() {
  fs.writeFileSync(filePath, csvContent, 'utf-8');
  fs.writeFileSync(
    officeUsePath,
    ['PRICE LIST FOR OFFICE USE[1]', 'Description,Price', 'CT Scan,15000'].join('\n'),
    'utf-8'
  );

  await runTest('facility inference uses threshold to fallback to filename', async () => {
    const records = await parseCsvFile(filePath, { facilityInferenceThreshold: 0.95 });
    assert(records.length > 0, 'expected records from CSV');
    const facilityName = records[0].facilityName;
    assert.strictEqual(facilityName, 'My Facility Test');
  });

  await runTest('facility inference accepts lower threshold', async () => {
    const records = await parseCsvFile(filePath, { facilityInferenceThreshold: 0.8 });
    assert(records.length > 0, 'expected records from CSV');
    const facilityName = records[0].facilityName;
    assert.strictEqual(facilityName, 'General Hospital Lagos');
  });

  await runTest('explicit facility mapping overrides inference', async () => {
    const records = await parseCsvFile(filePath, {
      explicitFacilityMapping: { [fileName]: 'Mapped Facility' },
    });
    assert(records.length > 0, 'expected records from CSV');
    const facilityName = records[0].facilityName;
    assert.strictEqual(facilityName, 'Mapped Facility');
  });

  await runTest('non-facility names are rejected', async () => {
    const records = await parseCsvFile(officeUsePath);
    assert.strictEqual(records.length, 0);
  });

  fs.rmSync(tempDir, { recursive: true, force: true });

  if (process.exitCode) {
    process.exit(1);
  }
}

main().catch((err) => {
  console.error(err);
  process.exit(1);
});
