/**
 * MEGALEK / IJEDE GENERAL HOSPITAL ingestion tests.
 *
 * CSV format: hierarchical parent-child rows, no header row, mixed
 * categories, tier headers, surgical breakdowns with running totals.
 */
import assert from 'assert';
import path from 'path';
import { parseCsvFile } from '../../ingestion/priceListParser';
import { PriceData } from '../../types/PriceData';

const CSV_PATH = path.resolve(__dirname, '../../../MEGALEK NEW PRICE LIST 2026.csv');

let records: PriceData[] = [];

async function loadOnce(): Promise<PriceData[]> {
  if (records.length === 0) {
    records = await parseCsvFile(CSV_PATH, {
      sourceFile: 'MEGALEK NEW PRICE LIST 2026.csv',
      currency: 'NGN',
    });
  }
  return records;
}

function find(desc: string, tier?: string): PriceData | undefined {
  return records.find(
    (r) =>
      r.procedureDescription.toUpperCase().includes(desc.toUpperCase()) &&
      (!tier || (r.metadata?.priceTier || '').toLowerCase() === tier.toLowerCase()),
  );
}

function findAll(desc: string): PriceData[] {
  return records.filter((r) =>
    r.procedureDescription.toUpperCase().includes(desc.toUpperCase()),
  );
}

// -----------------------------------------------------------
// Test runner
// -----------------------------------------------------------
const tests: Array<{ name: string; fn: () => void | Promise<void> }> = [];
function test(name: string, fn: () => void | Promise<void>) {
  tests.push({ name, fn });
}

async function runTest(t: { name: string; fn: () => void | Promise<void> }) {
  try {
    await t.fn();
    console.log(`\u2713 ${t.name}`);
  } catch (err: any) {
    console.log(`\u2717 ${t.name}`);
    console.error(err);
    process.exitCode = 1;
  }
}

// ============================================================
//  Phase 1: Section / Area detection
// ============================================================

test('REVENUE section area is clean (no # artifacts)', async () => {
  await loadOnce();
  const folder = find('FOLDER');
  assert.ok(folder, 'FOLDER record expected');
  const area = (folder.metadata?.area || '').toString();
  assert.ok(!area.includes('#'), `Area should not contain #: "${area}"`);
  assert.ok(/revenue/i.test(area), `Area should be REVENUE, got "${area}"`);
});

test('PAEDIATRICS section items have area PAEDIATRICS', async () => {
  await loadOnce();
  // Lettered items a-i in PAEDIATRICS section
  const incubator = find('INCUBATOR');
  assert.ok(incubator, 'INCUBATOR record expected');
  assert.ok(
    /paediatric/i.test(incubator.metadata?.area || ''),
    `Expected area PAEDIATRICS, got "${incubator.metadata?.area}"`,
  );
});

test('PHYSIOTHERAPY section items have area PHYSIOTHERAPY', async () => {
  await loadOnce();
  const physio = records.find(
    (r) =>
      /yrs.*session|ultra\s*sound|laser|wax\s*bath|hemi\s*tilt/i.test(r.procedureDescription) &&
      /physiotherap/i.test(r.metadata?.area || ''),
  );
  assert.ok(physio, 'Expected at least one PHYSIOTHERAPY area record');
});

// ============================================================
//  Phase 2: Surgical parent-child consolidation
// ============================================================

test('MYMECTOMY consolidated with total 450,000 and breakdown', async () => {
  await loadOnce();
  const rec = find('MYMECTOMY');
  assert.ok(rec, 'MYMECTOMY record expected');
  assert.strictEqual(rec.price, 450000, `Expected 450000, got ${rec.price}`);
  assert.ok(rec.metadata?.breakdown?.length > 0, 'Expected breakdown');
});

test('CEASEREAN SECTION (BOOKED) consolidated with total 200,000', async () => {
  await loadOnce();
  const rec = find('CEASEREAN SECTION (BOOKED)');
  assert.ok(rec, 'CEASEREAN SECTION (BOOKED) record expected');
  assert.strictEqual(rec.price, 200000, `Expected 200000, got ${rec.price}`);
  assert.ok(rec.metadata?.breakdown, 'Expected breakdown');
});

test('BOOKED NORMAL DELIVERY consolidated with running total', async () => {
  await loadOnce();
  const rec = find('BOOKED NORMAL DELIVERY');
  assert.ok(rec, 'BOOKED NORMAL DELIVERY record expected');
  assert.ok(rec.price >= 100000, `Expected consolidated price >= 100000, got ${rec.price}`);
  assert.ok(rec.metadata?.breakdown, 'Expected breakdown in metadata');
});

test('No orphaned SURGICAL PACK standalone records', async () => {
  await loadOnce();
  const orphans = records.filter(
    (r) =>
      /^(surgical pack|operation fee|theatre pack|anaesthesia)$/i.test(
        r.procedureDescription.trim(),
      ),
  );
  assert.strictEqual(
    orphans.length,
    0,
    `Expected 0 orphaned sub-items, got ${orphans.length}: ${orphans.map((r) => `"${r.procedureDescription}" row=${r.metadata?.rowNumber}`).join(', ')}`,
  );
});

// ============================================================
//  Phase 3: Standalone items
// ============================================================

test('PAD = 1,500 (standalone consumable)', async () => {
  await loadOnce();
  const rec = find('PAD');
  assert.ok(rec, 'PAD record expected');
  assert.strictEqual(rec.price, 1500);
});

test('CIRCUMCISION = 7,500 (standalone surgical)', async () => {
  await loadOnce();
  const rec = find('CIRCUMCISION');
  assert.ok(rec, 'CIRCUMCISION record expected');
  assert.strictEqual(rec.price, 7500);
});

test('EVACUTION = 50,000 (standalone)', async () => {
  await loadOnce();
  const recs = findAll('EVACUTION');
  const match = recs.find((r) => r.price === 50000);
  assert.ok(match, 'EVACUTION at 50000 expected');
});

test('ECG = 6,000', async () => {
  await loadOnce();
  const rec = find('ECG');
  assert.ok(rec, 'ECG record expected');
  assert.strictEqual(rec.price, 6000);
});

// ============================================================
//  Phase 4: Tier headers (MEDICAL CERTIFICATE, FIBROADENOMA)
// ============================================================

test('MEDICAL CERTIFICATE (NYSC) produces tiered records with correct prices', async () => {
  await loadOnce();
  const recs = findAll('MEDICAL CERTIFICATE (NYSC)');
  assert.ok(recs.length >= 1, `Expected MEDICAL CERTIFICATE records, got ${recs.length}`);
  // Should have child-age and adult tiers, total prices 13000 / 15000
  const prices = recs.map((r) => r.price).sort((a, b) => a - b);
  assert.ok(
    prices.some((p) => p >= 10000),
    `Expected at least one price >= 10000, got [${prices.join(', ')}]`,
  );
  // Should NOT have price=18 (which is "18YRS" parsed as number)
  assert.ok(
    !prices.includes(18),
    `Should not have price=18 (tier label parsed as price), got [${prices.join(', ')}]`,
  );
});

test('FIBROADENOMA produces SIMPLE/COMPLEX tier records with TOTAL prices', async () => {
  await loadOnce();
  const recs = findAll('FIBROADENOMA');
  assert.ok(recs.length >= 2, `Expected >= 2 FIBROADENOMA records, got ${recs.length}`);
  const prices = recs.map((r) => r.price).sort((a, b) => a - b);
  // TOTAL row: SIMPLE=80000, COMPLEX=100000
  assert.ok(
    prices.includes(80000) || prices.includes(100000),
    `Expected 80000 or 100000 in prices, got [${prices.join(', ')}]`,
  );
});

// ============================================================
//  Phase 5: Garbage filtering
// ============================================================

test('No single-letter description records', async () => {
  await loadOnce();
  const singles = records.filter((r) => /^[a-z]$/i.test(r.procedureDescription.trim()));
  assert.strictEqual(singles.length, 0, `Expected 0 single-letter records, got ${singles.length}`);
});

test('No footer/signature records', async () => {
  await loadOnce();
  const footers = records.filter((r) =>
    /SIGN\s*BY|MEDICAL\s+DIRECTOR|NOTE\s*:/i.test(r.procedureDescription),
  );
  assert.strictEqual(footers.length, 0, `Expected 0 footer records, got ${footers.length}`);
});

test('No zero-price records (all items should have positive prices)', async () => {
  await loadOnce();
  const zeroPriced = records.filter((r) => r.price === 0);
  assert.strictEqual(
    zeroPriced.length,
    0,
    `Expected 0 zero-price records, got ${zeroPriced.length}: ${zeroPriced.slice(0, 5).map((r) => `"${r.procedureDescription}"`).join(', ')}`,
  );
});

// ============================================================
//  Phase 6: Integration validation
// ============================================================

test('Total record count is between 80-160 (consolidated, not 262)', async () => {
  await loadOnce();
  assert.ok(
    records.length >= 80 && records.length <= 160,
    `Expected 80-160 records, got ${records.length}`,
  );
});

test('All records have facility IJEDE GENERAL HOSPITAL', async () => {
  await loadOnce();
  for (const r of records) {
    assert.ok(
      /ijede/i.test(r.facilityName),
      `Expected IJEDE in facility name, got "${r.facilityName}"`,
    );
  }
});

test('All records have NGN currency', async () => {
  await loadOnce();
  for (const r of records) {
    assert.strictEqual(r.currency, 'NGN', `Expected NGN, got "${r.currency}"`);
  }
});

// ============================================================
//  Run
// ============================================================

async function main() {
  await loadOnce();

  console.log(`Inferred facility name from content: ${records[0]?.facilityName || 'UNKNOWN'}`);
  console.log(`Total records loaded: ${records.length}\n`);

  for (const t of tests) {
    await runTest(t);
  }

  // Summary
  const areas = new Set(records.map((r) => r.metadata?.area).filter(Boolean));
  const categories = new Set(records.map((r) => r.metadata?.category).filter(Boolean));
  const codeCounts = new Map<string, number>();
  for (const r of records) {
    codeCounts.set(r.procedureCode, (codeCounts.get(r.procedureCode) || 0) + 1);
  }
  const collisions = [...codeCounts.entries()].filter(([, c]) => c > 1).length;
  console.log(`\n--- MEGALEK Summary ---`);
  console.log(`Total records: ${records.length}`);
  console.log(`Unique areas: ${areas.size} (${[...areas].sort().join(', ')})`);
  console.log(`Unique categories: ${categories.size}`);
  console.log(`Procedure code collisions: ${collisions} / ${codeCounts.size}`);
  console.log(
    `Zero-price records: ${records.filter((r) => r.price === 0).length}`,
  );
}

main().catch(console.error);
