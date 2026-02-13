import assert from 'assert';
import path from 'path';
import { parseCsvFile, parseDocxFile } from '../../ingestion/priceListParser';
import { FilePriceListProvider } from '../../providers/FilePriceListProvider';
import { InMemoryDocumentStore } from '../../stores/InMemoryDocumentStore';
import { PriceData } from '../../types/PriceData';

const fixturesDir = path.resolve(__dirname, '..', '..', 'internal', 'providers', 'data');
const megalekCsv = path.join(fixturesDir, 'MEGALEK NEW PRICE LIST 2026.csv');
const lasuthCsv = path.join(fixturesDir, 'NEW LASUTH PRICE LIST (SERVICES).csv');
const randleCsv = path.join(fixturesDir, 'PRICE LIST FOR RANDLE GENERAL HOSPITAL JANUARY 2026.csv');
const officeDocx = path.join(fixturesDir, 'PRICE_LIST_FOR_OFFICE_USE[1].docx');

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

function findRecord(records: PriceData[], query: string) {
  return records.find((record) => record.procedureDescription.toLowerCase().includes(query.toLowerCase()));
}

async function main() {
  await runTest('parse MEGALEK CSV (category + price)', async () => {
    const records = await parseCsvFile(megalekCsv, { currency: 'NGN' });
    assert(records.length > 0, 'expected records from MEGALEK CSV');

    const padRecord = findRecord(records, 'pad');
    assert(padRecord, 'expected PAD record');
    assert.strictEqual(padRecord!.price, 1500, 'expected PAD price 1500');
    assert.strictEqual(padRecord!.metadata?.category, 'CONSUMABLES', 'expected category CONSUMABLES');
  });

  await runTest('parse LASUTH CSV (area + tier)', async () => {
    const records = await parseCsvFile(lasuthCsv, { currency: 'NGN' });
    assert(records.length > 0, 'expected records from LASUTH CSV');

    const emergencyRecord = findRecord(records, 'emergency wards');
    assert(emergencyRecord, 'expected emergency wards accommodation record');
    assert.strictEqual(emergencyRecord!.metadata?.area, 'ACCOMMODATION', 'expected area ACCOMMODATION');

    const adultVariant = records.find(
      (record) =>
        record.procedureDescription.toLowerCase().includes('emergency wards') &&
        record.metadata?.priceTier === 'adult'
    );
    assert(adultVariant, 'expected adult tier variant');
    assert.strictEqual(adultVariant!.price, 5000, 'expected adult price 5000');
  });

  await runTest('parse RANDLE CSV (unit detection)', async () => {
    const records = await parseCsvFile(randleCsv, { currency: 'NGN' });
    assert(records.length > 0, 'expected records from RANDLE CSV');

    const extraDay = findRecord(records, 'extral day');
    assert(extraDay, 'expected EXTRAL DAY record');
    assert.strictEqual(extraDay!.price, 4800, 'expected EXTRAL DAY price 4800');
    assert.strictEqual(extraDay!.metadata?.unit, 'per_day', 'expected unit per_day');
  });

  await runTest('parse DOCX price list', async () => {
    const records = await parseDocxFile(officeDocx, { currency: 'NGN' });
    assert(records.length > 0, 'expected records from DOCX');

    const pelvicScan = findRecord(records, 'pelvic scan');
    assert(pelvicScan, 'expected pelvic scan record');
    assert.strictEqual(pelvicScan!.price, 4000, 'expected pelvic scan price 4000');
  });

  await runTest('file provider syncs and returns previous batch', async () => {
    const store = new InMemoryDocumentStore('test-store');
    const provider = new FilePriceListProvider(store);
    await provider.initialize({
      files: [
        { path: megalekCsv },
        { path: lasuthCsv },
        { path: randleCsv },
        { path: officeDocx },
      ],
      currency: 'NGN',
      defaultEffectiveDate: '2026-01-01',
    });

    const current = await provider.getCurrentData();
    assert(current.data.length > 0, 'expected current data to be non-empty');

    const padTagged = current.data.find((record) => record.procedureDescription.toLowerCase() === 'pad');
    assert(padTagged, 'expected PAD record in current data');
    assert(padTagged!.tags?.includes('consumables'), 'expected PAD to include consumables tag');

    const lasuthTagged = current.data.find((record) =>
      record.facilityName.toLowerCase().includes('lagos state university teaching hospital')
    );
    assert(lasuthTagged, 'expected LASUTH record in current data');
    assert(lasuthTagged!.tags?.includes('teaching_hospital'), 'expected LASUTH to include teaching_hospital tag');

    await provider.syncData();
    await provider.syncData();

    const previous = await provider.getPreviousData();
    assert(previous.data.length > 0, 'expected previous data after two syncs');
  });

  await runTest('file provider generates stable keys', async () => {
    const store = new InMemoryDocumentStore('key-test-store');
    const provider = new FilePriceListProvider(store);
    await provider.initialize({
      files: [{ path: megalekCsv }],
      currency: 'NGN',
      defaultEffectiveDate: '2026-01-01',
    });

    const current = await provider.getCurrentData({ limit: 1 });
    assert(current.data.length === 1, 'expected at least one record');
    const record = current.data[0];

    const key1 = (provider as any).generateKey(record, 0);
    const key2 = (provider as any).generateKey(record, 0);
    assert.strictEqual(key1, key2, 'expected stable key for identical record');

    const mutated = { ...record, price: record.price + 1 };
    const key3 = (provider as any).generateKey(mutated, 0);
    assert.notStrictEqual(key1, key3, 'expected key to change when price changes');
  });

  if (process.exitCode) {
    process.exit(1);
  }
}

main().catch((err) => {
  console.error(err);
  process.exit(1);
});
