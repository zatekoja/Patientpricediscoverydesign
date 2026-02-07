import assert from 'assert';
import path from 'path';
import { parseCsvFile, parseDocxFile } from '../../ingestion/priceListParser';
import { FilePriceListProvider } from '../../providers/FilePriceListProvider';
import { InMemoryDocumentStore } from '../../stores/InMemoryDocumentStore';
import { PriceData } from '../../types/PriceData';

const fixturesDir = path.resolve(__dirname, '..', '..', 'fixtures', 'price_lists');
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
  await runTest('parse MEGALEK CSV (category + price)', () => {
    const records = parseCsvFile(megalekCsv, { currency: 'NGN' });
    assert(records.length > 0, 'expected records from MEGALEK CSV');

    const padRecord = findRecord(records, 'pad');
    assert(padRecord, 'expected PAD record');
    assert.strictEqual(padRecord!.price, 1500, 'expected PAD price 1500');
    assert.strictEqual(padRecord!.metadata?.category, 'CONSUMABLES', 'expected category CONSUMABLES');
  });

  await runTest('parse LASUTH CSV (area + tier)', () => {
    const records = parseCsvFile(lasuthCsv, { currency: 'NGN' });
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

  await runTest('parse RANDLE CSV (unit detection)', () => {
    const records = parseCsvFile(randleCsv, { currency: 'NGN' });
    assert(records.length > 0, 'expected records from RANDLE CSV');

    const extraDay = findRecord(records, 'extral day');
    assert(extraDay, 'expected EXTRAL DAY record');
    assert.strictEqual(extraDay!.price, 4800, 'expected EXTRAL DAY price 4800');
    assert.strictEqual(extraDay!.metadata?.unit, 'per_day', 'expected unit per_day');
  });

  await runTest('parse DOCX price list', () => {
    const records = parseDocxFile(officeDocx, { currency: 'NGN' });
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

    await provider.syncData();
    await provider.syncData();

    const previous = await provider.getPreviousData();
    assert(previous.data.length > 0, 'expected previous data after two syncs');
  });

  if (process.exitCode) {
    process.exit(1);
  }
}

main().catch((err) => {
  console.error(err);
  process.exit(1);
});
