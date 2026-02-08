import assert from 'assert';
import path from 'path';
import { FilePriceListProvider, FilePriceListConfig } from '../../providers/FilePriceListProvider';
import { DocumentSummaryParser } from '../../ingestion/documentSummary';

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
  await runTest('FilePriceListProvider uses LLM document summary when available', async () => {
    const parser: DocumentSummaryParser = {
      parse: async (filePath, context) => ({
        facilityName: 'LLM Facility',
        currency: 'NGN',
        effectiveDate: '2026-01-01',
        items: [{ description: 'LLM Procedure', price: 1234, category: 'Imaging' }],
        documentMetadata: {
          sourceFile: path.basename(filePath),
          extractedAt: new Date().toISOString(),
          model: 'gpt-4o-mini',
        },
      }),
    };

    const provider = new FilePriceListProvider(undefined, undefined, undefined, parser);
    const config: FilePriceListConfig = {
      files: [{ path: megalekCsv }],
      currency: 'NGN',
      defaultEffectiveDate: '2026-01-01',
    };

    await provider.initialize(config);
    const current = await provider.getCurrentData({ limit: 10 });
    assert.strictEqual(current.data.length, 1);
    assert.strictEqual(current.data[0].facilityName, 'LLM Facility');
    assert.strictEqual(current.data[0].procedureDescription, 'LLM Procedure');
  });

  await runTest('FilePriceListProvider falls back to parser when LLM returns null', async () => {
    const parser: DocumentSummaryParser = {
      parse: async () => null,
    };

    const provider = new FilePriceListProvider(undefined, undefined, undefined, parser);
    const config: FilePriceListConfig = {
      files: [{ path: megalekCsv }],
      currency: 'NGN',
      defaultEffectiveDate: '2026-01-01',
    };

    await provider.initialize(config);
    const current = await provider.getCurrentData({ limit: 5 });
    assert(current.data.length > 0, 'expected fallback parser to return data');
  });

  if (process.exitCode) {
    process.exit(1);
  }
}

main().catch((err) => {
  console.error(err);
  process.exit(1);
});
