import assert from 'assert';
import fs from 'fs';
import os from 'os';
import path from 'path';
import * as xlsx from 'xlsx';
import { extractDocumentPreview } from '../../ingestion/documentTextExtractor';

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
  await runTest('extractDocumentPreview handles CSV files', async () => {
    const tempDir = fs.mkdtempSync(path.join(os.tmpdir(), 'ppd-csv-'));
    const filePath = path.join(tempDir, 'sample.csv');
    fs.writeFileSync(filePath, 'Description,Price\nMRI Scan,1500\nCT Scan,2000\n', 'utf-8');

    const preview = await extractDocumentPreview(filePath, { maxRows: 10, maxChars: 2000 });
    assert(preview, 'expected preview to be returned');
    assert.strictEqual(preview!.rowCount, 3);
    assert(preview!.preview.includes('Description | Price'));

    fs.rmSync(tempDir, { recursive: true, force: true });
  });

  await runTest('extractDocumentPreview handles XLSX files', async () => {
    const tempDir = fs.mkdtempSync(path.join(os.tmpdir(), 'ppd-xlsx-'));
    const filePath = path.join(tempDir, 'sample.xlsx');

    const workbook = xlsx.utils.book_new();
    const sheet = xlsx.utils.aoa_to_sheet([
      ['Description', 'Price'],
      ['Ultrasound', 2500],
      ['X-ray', 1800],
    ]);
    xlsx.utils.book_append_sheet(workbook, sheet, 'Sheet1');
    xlsx.writeFile(workbook, filePath);

    const preview = await extractDocumentPreview(filePath, { maxRows: 10, maxChars: 2000 });
    assert(preview, 'expected preview to be returned');
    assert.strictEqual(preview!.rowCount, 3);
    assert(preview!.preview.includes('Ultrasound | 2500'));

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
