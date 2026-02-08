import path from 'path';
import { loadVaultSecretsToEnv } from '../config/vaultSecrets';
import { LLMDocumentParser } from '../ingestion/llmDocumentParser';
import { PriceListParseContext } from '../ingestion/priceListParser';

type FixtureExpectation = {
  file: string;
  expectedFacility: string;
};

const fixtures: FixtureExpectation[] = [
  {
    file: 'MEGALEK NEW PRICE LIST 2026.csv',
    expectedFacility: 'Ijede General Hospital',
  },
  {
    file: 'NEW LASUTH PRICE LIST (SERVICES).csv',
    expectedFacility: 'Lagos State University Teaching Hospital (LASUTH)',
  },
  {
    file: 'PRICE LIST FOR RANDLE GENERAL HOSPITAL JANUARY 2026.csv',
    expectedFacility: 'Randle General Hospital',
  },
  {
    file: 'PRICE_LIST_FOR_OFFICE_USE[1].docx',
    expectedFacility: 'General Hospital Badagry',
  },
];

function resolveFixturePath(fileName: string): string {
  const candidates = [
    path.join(process.cwd(), 'backend', 'fixtures', 'price_lists', fileName),
    path.join(process.cwd(), 'fixtures', 'price_lists', fileName),
  ];
  for (const candidate of candidates) {
    try {
      if (candidate && require('fs').existsSync(candidate)) {
        return candidate;
      }
    } catch {
      // ignore
    }
  }
  return candidates[0];
}

function pct(value: number): string {
  return `${(value * 100).toFixed(1)}%`;
}

async function main() {
  const vaultPath = process.env.VAULT_PROVIDER_PATH || 'patient-price-discovery/provider';
  const vaultResult = await loadVaultSecretsToEnv({ path: vaultPath });
  if (vaultResult.enabled && vaultResult.error) {
    console.warn(`Vault not ready: ${vaultResult.error}`);
  }

  const apiKey =
    process.env.PROVIDER_LLM_API_KEY ||
    process.env.OPENAI_API_KEY ||
    '';
  if (!apiKey) {
    throw new Error('Missing LLM API key (PROVIDER_LLM_API_KEY / OPENAI_API_KEY).');
  }

  const parser = new LLMDocumentParser({
    enabled: true,
    apiKey,
    apiEndpoint: process.env.PROVIDER_LLM_API_ENDPOINT || process.env.OPENAI_API_ENDPOINT,
    model: process.env.OPENAI_MODEL || 'gpt-4o-mini',
    temperature: 0.1,
    maxRows: 80,
    maxChars: 60000,
  });

  let facilityTotal = 0;
  let facilityMatches = 0;
  let totalItems = 0;
  let itemsWithCategory = 0;

  for (const fixture of fixtures) {
    const filePath = resolveFixturePath(fixture.file);
    const sourceFile = path.basename(filePath);
    const context: PriceListParseContext = {
      currency: 'NGN',
      sourceFile,
      providerId: 'file_price_list',
    };

    const summary = await parser.parse(filePath, context);
    if (!summary) {
      console.log(`✗ ${sourceFile}: no summary returned`);
      facilityTotal += 1;
      continue;
    }

    facilityTotal += 1;
    const facilityOk = summary.facilityName === fixture.expectedFacility;
    if (facilityOk) {
      facilityMatches += 1;
    }

    const categoryCount = summary.items.filter((item: { category?: string | null }) =>
      (item.category || '').trim().length > 0
    ).length;
    totalItems += summary.items.length;
    itemsWithCategory += categoryCount;

    const coverage = summary.items.length > 0 ? categoryCount / summary.items.length : 0;
    console.log(
      `${facilityOk ? '✓' : '✗'} ${sourceFile}: ` +
        `facility="${summary.facilityName}" expected="${fixture.expectedFacility}" ` +
        `categories=${categoryCount}/${summary.items.length} (${pct(coverage)})`
    );
  }

  const facilityAccuracy = facilityTotal > 0 ? facilityMatches / facilityTotal : 0;
  const categoryCoverage = totalItems > 0 ? itemsWithCategory / totalItems : 0;

  console.log('');
  console.log(`Facility name accuracy: ${facilityMatches}/${facilityTotal} (${pct(facilityAccuracy)})`);
  console.log(`Category coverage: ${itemsWithCategory}/${totalItems} (${pct(categoryCoverage)})`);

  const target = 0.85;
  if (facilityAccuracy < target || categoryCoverage < target) {
    process.exitCode = 1;
  }
}

main().catch((err) => {
  console.error(err);
  process.exit(1);
});
