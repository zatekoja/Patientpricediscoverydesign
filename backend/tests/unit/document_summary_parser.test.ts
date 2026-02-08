import assert from 'assert';
import { parseDocumentSummaryResponse, summaryToPriceData } from '../../ingestion/documentSummary';

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
  await runTest('parseDocumentSummaryResponse normalizes price strings', () => {
    const raw = JSON.stringify({
      facilityName: 'General Hospital Lagos',
      currency: 'NGN',
      effectiveDate: '2026-01-01',
      items: [
        { description: 'MRI Scan', price: '1,500.00', category: 'Imaging' },
      ],
      documentMetadata: {
        sourceFile: 'sample.csv',
        extractedAt: '2026-02-07T00:00:00Z',
        model: 'gpt-4o-mini',
      },
    });

    const summary = parseDocumentSummaryResponse(raw, 'sample.csv');
    assert(summary, 'expected parsed summary');
    assert.strictEqual(summary!.items[0].price, 1500);
  });

  await runTest('parseDocumentSummaryResponse rejects missing items', () => {
    const raw = JSON.stringify({
      facilityName: 'Missing Items Hospital',
      documentMetadata: {
        sourceFile: 'sample.csv',
        extractedAt: '2026-02-07T00:00:00Z',
        model: 'gpt-4o-mini',
      },
    });
    const summary = parseDocumentSummaryResponse(raw, 'sample.csv');
    assert.strictEqual(summary, null);
  });

  await runTest('summaryToPriceData falls back to context currency', () => {
    const raw = JSON.stringify({
      facilityName: 'Summary Hospital',
      items: [{ description: 'Ultrasound', price: 2000 }],
      documentMetadata: {
        sourceFile: 'sample.csv',
        extractedAt: '2026-02-07T00:00:00Z',
        model: 'gpt-4o-mini',
      },
    });
    const summary = parseDocumentSummaryResponse(raw, 'sample.csv');
    assert(summary, 'expected parsed summary');
    const priceData = summaryToPriceData(summary!, {
      currency: 'NGN',
      defaultEffectiveDate: new Date('2026-01-01T00:00:00Z'),
      providerId: 'file_price_list',
    });
    assert.strictEqual(priceData.length, 1);
    assert.strictEqual(priceData[0].currency, 'NGN');
  });

  await runTest('parseDocumentSummaryResponse normalizes facility aliases', () => {
    const raw = JSON.stringify({
      facilityName: 'LAGOS STATE UNIVERSITY TEACHING HOSPITAL PRICE LIST',
      currency: 'NGN',
      items: [{ description: 'CT Scan', price: 20000 }],
      documentMetadata: {
        sourceFile: 'NEW LASUTH PRICE LIST (SERVICES).csv',
        extractedAt: '2026-02-07T00:00:00Z',
        model: 'gpt-4o-mini',
      },
    });

    const summary = parseDocumentSummaryResponse(raw, 'NEW LASUTH PRICE LIST (SERVICES).csv');
    assert(summary, 'expected parsed summary');
    assert.strictEqual(summary!.facilityName, 'Lagos State University Teaching Hospital (LASUTH)');
  });

  await runTest('parseDocumentSummaryResponse rejects non-facility names', () => {
    const raw = JSON.stringify({
      facilityName: 'PRICE LIST FOR OFFICE USE[1]',
      items: [{ description: 'Test Item', price: 1000 }],
      documentMetadata: {
        sourceFile: 'PRICE_LIST_FOR_OFFICE_USE[1].docx',
        extractedAt: '2026-02-07T00:00:00Z',
        model: 'gpt-4o-mini',
      },
    });
    const summary = parseDocumentSummaryResponse(raw, 'PRICE_LIST_FOR_OFFICE_USE[1].docx');
    assert.strictEqual(summary, null);
  });

  if (process.exitCode) {
    process.exit(1);
  }
}

main().catch((err) => {
  console.error(err);
  process.exit(1);
});
