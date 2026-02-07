import assert from 'assert';
import { parseCsvContent } from '../../ingestion/priceListParser';

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
  await runTest('parseCsvContent handles quoted fields with commas', () => {
    const csv = 'Name,Description,Price\n"Smith, John","Heart surgery, complex",15000\n';
    const rows = parseCsvContent(csv);
    assert.strictEqual(rows.length, 2);
    assert.strictEqual(rows[1][0], 'Smith, John');
    assert.strictEqual(rows[1][1], 'Heart surgery, complex');
    assert.strictEqual(rows[1][2], '15000');
  });

  await runTest('parseCsvContent handles multi-line cells', () => {
    const csv = 'Name,Description\n"John","Line 1\nLine 2\nLine 3"\n';
    const rows = parseCsvContent(csv);
    assert.strictEqual(rows.length, 2);
    assert(rows[1][1].includes('Line 1'));
    assert(rows[1][1].includes('Line 2'));
    assert(rows[1][1].includes('Line 3'));
  });

  await runTest('parseCsvContent handles escaped quotes', () => {
    const csv = 'Name,Quote\n"John","He said ""Hello"" to me"\n';
    const rows = parseCsvContent(csv);
    assert.strictEqual(rows.length, 2);
    assert(rows[1][1].includes('"'));
  });

  await runTest('parseCsvContent handles empty fields', () => {
    const csv = 'A,B,C\n1,,3\n,2,\n';
    const rows = parseCsvContent(csv);
    assert.strictEqual(rows.length, 3);
    assert.strictEqual(rows[1][0], '1');
    assert.strictEqual(rows[1][1], '');
    assert.strictEqual(rows[1][2], '3');
  });

  await runTest('parseCsvContent handles varying column counts', () => {
    const csv = 'A,B,C\n1,2\n4,5,6,7\n';
    const rows = parseCsvContent(csv);
    assert.strictEqual(rows.length, 3);
    // csv-parse with relax_column_count handles this gracefully
    assert(rows[1].length >= 2);
  });

  await runTest('parseCsvContent handles price numbers with commas', () => {
    const csv = 'Description,Price\n"Surgery","1,500.00"\n';
    const rows = parseCsvContent(csv);
    assert.strictEqual(rows.length, 2);
    assert.strictEqual(rows[1][1], '1,500.00');
  });

  await runTest('parseCsvContent handles Windows line endings', () => {
    const csv = 'A,B\r\n1,2\r\n3,4\r\n';
    const rows = parseCsvContent(csv);
    assert(rows.length >= 3);
  });

  await runTest('parseCsvContent handles Unix line endings', () => {
    const csv = 'A,B\n1,2\n3,4\n';
    const rows = parseCsvContent(csv);
    assert(rows.length >= 3);
  });

  if (process.exitCode) {
    process.exit(1);
  }
}

main().catch((err) => {
  console.error(err);
  process.exit(1);
});
