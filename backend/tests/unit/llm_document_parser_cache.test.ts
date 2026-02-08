import assert from 'assert';
import fs from 'fs';
import os from 'os';
import path from 'path';
import { InMemoryDocumentStore } from '../../stores/InMemoryDocumentStore';
import { LLMDocumentParser, LLMDocumentParserConfig, LLMDocumentSummaryClient } from '../../ingestion/llmDocumentParser';
import { PriceListParseContext } from '../../ingestion/priceListParser';

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
  await runTest('LLMDocumentParser caches summaries by file hash', async () => {
    const tempDir = fs.mkdtempSync(path.join(os.tmpdir(), 'ppd-llm-'));
    const filePath = path.join(tempDir, 'sample.csv');
    fs.writeFileSync(filePath, 'Description,Price\nMRI Scan,1500\n', 'utf-8');

    let calls = 0;
    const fakeClient: LLMDocumentSummaryClient = {
      summarize: async () => {
        calls += 1;
        return {
          content: JSON.stringify({
            facilityName: 'Cached Hospital',
            currency: 'NGN',
            items: [{ description: 'MRI Scan', price: 1500 }],
            documentMetadata: {
              sourceFile: 'sample.csv',
              extractedAt: new Date().toISOString(),
              model: 'gpt-4o-mini',
            },
          }),
        };
      },
    };

    const store = new InMemoryDocumentStore('doc-summary-store');
    const config: LLMDocumentParserConfig = {
      enabled: true,
      apiKey: 'test-key',
      model: 'gpt-4o-mini',
      maxChars: 2000,
      maxRows: 100,
      maxBytes: 1024 * 1024,
    };

    const parser = new LLMDocumentParser(config, store, fakeClient);
    const context: PriceListParseContext = {
      currency: 'NGN',
      defaultEffectiveDate: new Date('2026-01-01T00:00:00Z'),
      providerId: 'file_price_list',
    };

    const summary1 = await parser.parse(filePath, context);
    const summary2 = await parser.parse(filePath, context);

    assert(summary1, 'expected summary to be returned');
    assert(summary2, 'expected cached summary to be returned');
    assert.strictEqual(calls, 1, 'expected LLM to be called only once');

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
