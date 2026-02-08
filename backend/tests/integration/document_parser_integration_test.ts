import assert from 'assert';
import crypto from 'crypto';
import fs from 'fs';
import os from 'os';
import path from 'path';
import { MongoDocumentStore } from '../../stores/MongoDocumentStore';
import {
  LLMDocumentParser,
  LLMDocumentParserConfig,
  LLMDocumentSummaryClient,
  DocumentSummaryCacheRecord,
} from '../../ingestion/llmDocumentParser';
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
  const mongoUri = process.env.TEST_MONGO_URI || process.env.PROVIDER_MONGO_URI;
  if (!mongoUri) {
    console.log('Skipping integration test: TEST_MONGO_URI not set');
    return;
  }

  const tempDir = fs.mkdtempSync(path.join(os.tmpdir(), 'ppd-llm-int-'));
  const filePath = path.join(tempDir, 'sample.csv');
  fs.writeFileSync(filePath, 'Description,Price\nMRI Scan,1500\nCT Scan,2000\n', 'utf-8');

  let calls = 0;
  const fakeClient: LLMDocumentSummaryClient = {
    summarize: async () => {
      calls += 1;
      return {
        content: JSON.stringify({
          facilityName: 'Integration Hospital',
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

  const config: LLMDocumentParserConfig = {
    enabled: true,
    apiKey: 'test-key',
    model: 'gpt-4o-mini',
    maxRows: 100,
    maxChars: 2000,
    maxBytes: 1024 * 1024,
  };

  const store = new MongoDocumentStore<DocumentSummaryCacheRecord>(
    mongoUri,
    process.env.PROVIDER_MONGO_DB || 'provider_data',
    process.env.PROVIDER_MONGO_DOC_SUMMARY_COLLECTION || 'document_summaries'
  );
  const mongoReady = await verifyMongo(store);
  if (!mongoReady) {
    console.log('Skipping integration test: Mongo not reachable');
    fs.rmSync(tempDir, { recursive: true, force: true });
    return;
  }

  const parser = new LLMDocumentParser(config, store, fakeClient);
  const context: PriceListParseContext = {
    currency: 'NGN',
    defaultEffectiveDate: new Date('2026-01-01T00:00:00Z'),
    providerId: 'file_price_list',
  };

  await runTest('LLMDocumentParser stores summaries in Mongo', async () => {
    const summary = await parser.parse(filePath, context);
    assert(summary, 'expected summary to be returned');
    assert.strictEqual(summary!.facilityName, 'Integration Hospital');

    const hash = await hashFile(filePath);
    const cacheKey = `doc_summary_${hash}`;
    const exists = await store.exists(cacheKey);
    assert.strictEqual(exists, true, 'expected summary cache record to exist');

    const cached = await store.get(cacheKey);
    assert(cached, 'expected cached summary record');
    assert.strictEqual(cached!.summary.facilityName, 'Integration Hospital');
  });

  await runTest('LLMDocumentParser uses cache on repeated calls', async () => {
    const summary = await parser.parse(filePath, context);
    assert(summary, 'expected summary to be returned from cache');
    assert.strictEqual(calls, 1, 'expected fake client to be called once');
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

async function hashFile(filePath: string): Promise<string> {
  const data = await fs.promises.readFile(filePath);
  return crypto.createHash('sha256').update(data).digest('hex');
}

async function verifyMongo(store: MongoDocumentStore<DocumentSummaryCacheRecord>): Promise<boolean> {
  try {
    await store.exists('healthcheck');
    return true;
  } catch (error) {
    return false;
  }
}
