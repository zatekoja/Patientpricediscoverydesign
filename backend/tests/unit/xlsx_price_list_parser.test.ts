import assert from 'assert';
import fs from 'fs';
import os from 'os';
import path from 'path';
import * as xlsx from 'xlsx';
import { parseXlsxFile } from '../../ingestion/priceListParser';

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
  await runTest('parse XLSX price list', async () => {
    const tempDir = fs.mkdtempSync(path.join(os.tmpdir(), 'ppd-xlsx-parse-'));
    const filePath = path.join(tempDir, 'prices.xlsx');

    const workbook = xlsx.utils.book_new();
    const sheet = xlsx.utils.aoa_to_sheet([
      ['Description', 'Price'],
      ['X-ray', 1200],
      ['MRI Scan', 5000],
    ]);
    xlsx.utils.book_append_sheet(workbook, sheet, 'Sheet1');
    xlsx.writeFile(workbook, filePath);

    const records = await parseXlsxFile(filePath, { currency: 'NGN' });
    assert(records.length >= 2, 'expected records from XLSX');
    const mri = records.find((record) => record.procedureDescription.toLowerCase().includes('mri'));
    assert(mri, 'expected MRI record');
    assert.strictEqual(mri!.price, 5000);

    fs.rmSync(tempDir, { recursive: true, force: true });
  });

  if (process.exitCode) {
    process.exit(1);
  }
}

main().catch((err) => {
  console.error(err);
  process.exit(1);
});
