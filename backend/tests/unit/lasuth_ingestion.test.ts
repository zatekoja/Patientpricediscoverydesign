import assert from 'assert';
import path from 'path';
import { parseCsvFile, _testExports } from '../../ingestion/priceListParser';
import { applyCuratedTags, hydrateTags } from '../../ingestion/tagHydration';
import { PriceData } from '../../types/PriceData';

const {
  expandPriceVariants,
  detectTotalPrice,
  cleanPriceText,
  extractPriceQualifier,
  parseAreaHierarchy,
  buildProcedureCode,
  splitByTier,
} = _testExports;

const fixturesDir = path.resolve(__dirname, '..', '..', 'fixtures', 'price_lists');
const lasuthCsv = path.join(fixturesDir, 'NEW LASUTH PRICE LIST (SERVICES).csv');

let lasuthRecords: PriceData[] = [];

async function runTest(name: string, fn: () => void | Promise<void>): Promise<void> {
  try {
    await fn();
    console.log(`\u2713 ${name}`);
  } catch (err) {
    console.error(`\u2717 ${name}`);
    console.error(err);
    process.exitCode = 1;
  }
}

function findRecord(records: PriceData[], query: string) {
  return records.find((r) => r.procedureDescription.toLowerCase().includes(query.toLowerCase()));
}

function findAllRecords(records: PriceData[], query: string) {
  return records.filter((r) => r.procedureDescription.toLowerCase().includes(query.toLowerCase()));
}

async function main() {
  // Load LASUTH records once for all integration tests
  lasuthRecords = await parseCsvFile(lasuthCsv, { currency: 'NGN' });

  // =============================================
  // Phase 1: TOTAL-line Price Breakdown Detection
  // =============================================

  await runTest('Phase 1: detectTotalPrice extracts TOTAL from breakdown', () => {
    assert.strictEqual(detectTotalPrice('Adm pack -#28,000\nTOTAL -#203,000'), 203000);
    assert.strictEqual(detectTotalPrice('TOTAL =#615,000'), 615000);
    assert.strictEqual(detectTotalPrice('TOTAL =#105,000'), 105000);
    assert.strictEqual(detectTotalPrice('TOTAL -#203,000'), 203000);
    assert.strictEqual(detectTotalPrice('TOTAL -#150,000'), 150000);
  });

  await runTest('Phase 1: detectTotalPrice returns null for simple prices', () => {
    assert.strictEqual(detectTotalPrice('5,000'), null);
    assert.strictEqual(detectTotalPrice('Free'), null);
    assert.strictEqual(detectTotalPrice('10,000 Adult'), null);
  });

  await runTest('Phase 1: expandPriceVariants returns single variant for TOTAL breakdowns', () => {
    const text = 'Adm pack -#28,000\nDiet -#10,000 (daily)\nUtility -#20,000\nTOTAL -#203,000';
    const variants = expandPriceVariants(text, 'EMERGENCY ADMISSION FEE AT MESD');
    assert.strictEqual(variants.length, 1, `expected 1 variant, got ${variants.length}`);
    assert.strictEqual(variants[0].price, 203000);
  });

  await runTest('Phase 1: expandPriceVariants handles ICU ventilator breakdown', () => {
    const text = 'Admission fee for 3 days =N225,000 (N75,000/ day)\nVentilator for 3 days =N240,000 (N80,000/ day)\nVentilator Circuit Consumable (N60,000 one-time payment)\nOxygen for 3 days =N90,000 (N30,000/ day)\nTOTAL =#615,000';
    const variants = expandPriceVariants(text, 'ADMISSION DEPOSIT FOR PATIENT ON VENTILATOR');
    assert.strictEqual(variants.length, 1, `expected 1 variant, got ${variants.length}`);
    assert.strictEqual(variants[0].price, 615000);
  });

  await runTest('Phase 1: expandPriceVariants still works for simple prices', () => {
    const variants = expandPriceVariants('5,000', 'GENERAL WARD ACCOMMODATION');
    assert.strictEqual(variants.length, 1);
    assert.strictEqual(variants[0].price, 5000);
  });

  await runTest('Phase 1: expandPriceVariants handles adult/paediatric tiers', () => {
    const text = '5,000 Adult only\nFree for Paed.';
    const variants = expandPriceVariants(text, 'EMERGENCY WARDS ACCOMODATION');
    const adultVariant = variants.find((v) => v.tier === 'adult');
    const paedVariant = variants.find((v) => v.tier === 'paediatric');
    assert(adultVariant, 'expected adult variant');
    assert.strictEqual(adultVariant!.price, 5000);
    assert(paedVariant, 'expected paediatric variant');
    assert.strictEqual(paedVariant!.price, 0);
  });

  await runTest('Phase 1: expandPriceVariants handles numeric adult/paed tiers', () => {
    const text = '5,000 Adult\n\n3,000 Paed.';
    const variants = expandPriceVariants(text, 'AUTOMATED IOP');
    const adultVariant = variants.find((v) => v.tier === 'adult');
    const paedVariant = variants.find((v) => v.tier === 'paediatric');
    assert(adultVariant, 'expected adult variant');
    assert.strictEqual(adultVariant!.price, 5000);
    assert(paedVariant, 'expected paediatric variant');
    assert.strictEqual(paedVariant!.price, 3000);
  });

  await runTest('Phase 1: splitByTier parses adult/paed patterns', () => {
    const result = splitByTier('5,000 Adult only Free for Paed.');
    assert.strictEqual(result.length, 2);
    assert.strictEqual(result[0].price, 5000);
    assert.strictEqual(result[0].tier, 'adult');
    assert.strictEqual(result[1].price, 0);
    assert.strictEqual(result[1].tier, 'paediatric');
  });

  // =========================================
  // Phase 1 Integration: LASUTH TOTAL records
  // =========================================

  await runTest('Phase 1 Integration: MESD admission produces correct total', () => {
    const records = findAllRecords(lasuthRecords, 'EMERGENCY ADMISSION FEE AT MESD');
    assert(records.length >= 1, `expected at least 1 MESD record, got ${records.length}`);
    const primary = records.find((r) => r.price === 203000);
    assert(primary, `expected MESD record with price 203000, got prices: ${records.map((r) => r.price)}`);
  });

  await runTest('Phase 1 Integration: Ward from clinic admission produces correct total', () => {
    const record = findRecord(lasuthRecords, 'ADMISSION TO THE WARD FROM THE CLINIC');
    assert(record, 'expected ADMISSION TO THE WARD FROM THE CLINIC record');
    assert.strictEqual(record!.price, 198000, `expected price 198000, got ${record!.price}`);
  });

  await runTest('Phase 1 Integration: ICU ventilator produces correct total', () => {
    const records = findAllRecords(lasuthRecords, 'ADMISSION DEPOSIT FOR PATIENT ON VENTILATOR');
    assert(records.length >= 1, `expected at least 1 ICU ventilator record, got ${records.length}`);
    const primary = records.find((r) => r.price === 615000);
    assert(primary, `expected ICU ventilator record with price 615000, got prices: ${records.map((r) => r.price)}`);
  });

  // ================================================
  // Phase 2: Description/Price Column Overlap
  // ================================================

  await runTest('Phase 2: cleanPriceText strips duplicate description prefix', () => {
    const price = 'EMERGENCY ADMISSION FEE AT MESD (first 10 days):\nAdm pack -#28,000\nTOTAL -#203,000';
    const desc = 'EMERGENCY ADMISSION FEE AT MESD (first 10 days)';
    const cleaned = cleanPriceText(price, desc);
    assert(!cleaned.startsWith('EMERGENCY'), `cleaned text should not start with description: ${cleaned.slice(0, 50)}`);
  });

  await runTest('Phase 2: cleanPriceText leaves normal prices unchanged', () => {
    const result = cleanPriceText('5,000', 'GENERAL WARD ACCOMMODATION');
    assert.strictEqual(result, '5,000');
  });

  // =============================================
  // Phase 3: Category Hierarchy Parsing
  // =============================================

  await runTest('Phase 3: parseAreaHierarchy splits colon-separated categories', () => {
    const result = parseAreaHierarchy('DENTAL UNIT: ORAL AND MAXILLOFACIAL SURGERY');
    assert.strictEqual(result.parentArea, 'DENTAL UNIT');
    assert.strictEqual(result.subCategory, 'ORAL AND MAXILLOFACIAL SURGERY');
  });

  await runTest('Phase 3: parseAreaHierarchy splits parenthetical categories', () => {
    const result = parseAreaHierarchy('VIP SERVICES (ACCELERATED CARE)');
    assert.strictEqual(result.parentArea, 'VIP SERVICES');
    assert.strictEqual(result.subCategory, 'ACCELERATED CARE');
  });

  await runTest('Phase 3: parseAreaHierarchy handles ENDOSCOPY with parenthetical', () => {
    const result = parseAreaHierarchy('ENDOSCOPY (NEONATAL PHOTOTHERAPY SERVICES)');
    assert.strictEqual(result.parentArea, 'ENDOSCOPY');
    assert.strictEqual(result.subCategory, 'NEONATAL PHOTOTHERAPY SERVICES');
  });

  await runTest('Phase 3: parseAreaHierarchy keeps simple areas flat', () => {
    const result = parseAreaHierarchy('AMBULANCE RATE');
    assert.strictEqual(result.parentArea, 'AMBULANCE RATE');
    assert.strictEqual(result.subCategory, undefined);
  });

  await runTest('Phase 3 Integration: dental items have parent area DENTAL UNIT', () => {
    const extraction = findRecord(lasuthRecords, 'EXTRACTION OF ALL TEETH');
    assert(extraction, 'expected EXTRACTION OF ALL TEETH record');
    assert.strictEqual(extraction!.metadata?.area, 'DENTAL UNIT', `got area: ${extraction!.metadata?.area}`);
    assert.strictEqual(extraction!.metadata?.category, 'ORAL AND MAXILLOFACIAL SURGERY', `got category: ${extraction!.metadata?.category}`);
  });

  await runTest('Phase 3 Integration: orthodontic items have correct sub-category', () => {
    const record = findRecord(lasuthRecords, 'UPPER AND LOWER FIXED APPLIANCE');
    assert(record, 'expected UPPER AND LOWER FIXED APPLIANCE record');
    assert.strictEqual(record!.metadata?.area, 'DENTAL UNIT');
    assert.strictEqual(record!.metadata?.category, 'ORTHODONTIC UNIT');
  });

  await runTest('Phase 3 Integration: VIP consultation has correct hierarchy', () => {
    const vip = lasuthRecords.find(
      (r) => r.procedureDescription === 'CONSULTATION' && r.price === 70000
    );
    assert(vip, 'expected VIP CONSULTATION record at price 70000');
    assert.strictEqual(vip!.metadata?.area, 'VIP SERVICES');
    assert.strictEqual(vip!.metadata?.category, 'ACCELERATED CARE');
  });

  // =============================================
  // Phase 6: Procedure Code Collisions
  // =============================================

  await runTest('Phase 6: buildProcedureCode produces unique codes for similar long descriptions', () => {
    const code1 = buildProcedureCode('ADMISSION TO THE WARD FROM THE EMERGENCY (first 10 days)', 20);
    const code2 = buildProcedureCode('ADMISSION TO THE WARD FROM THE CLINIC (first 10 days)', 21);
    assert.notStrictEqual(code1, code2, `codes should differ: ${code1} vs ${code2}`);
  });

  await runTest('Phase 6: buildProcedureCode is deterministic', () => {
    const code1 = buildProcedureCode('EXTRACTION OF ALL TEETH', 40);
    const code2 = buildProcedureCode('EXTRACTION OF ALL TEETH', 40);
    assert.strictEqual(code1, code2);
  });

  await runTest('Phase 6: buildProcedureCode keeps short codes readable', () => {
    const code = buildProcedureCode('PAD', 0);
    assert.strictEqual(code, 'PAD');
  });

  await runTest('Phase 6 Integration: LASUTH has fewer code collisions', () => {
    const codes = lasuthRecords.map((r) => r.procedureCode);
    const uniqueCodes = new Set(codes);
    const collisions = codes.length - uniqueCodes.size;
    // We expect some collisions from tier variants (adult/paed) sharing the same description
    // but should be much fewer than the pre-fix 156
    assert(collisions < 60, `expected fewer than 60 collisions, got ${collisions}`);
  });

  // =============================================
  // Phase 7: Price Qualifiers
  // =============================================

  await runTest('Phase 7: extractPriceQualifier detects market value', () => {
    const q = extractPriceQualifier('130,000 (depending on market value)');
    assert(q?.includes('depending on market value'), `expected market value qualifier, got: ${q}`);
  });

  await runTest('Phase 7: extractPriceQualifier detects per session', () => {
    const q = extractPriceQualifier('16,000 (#2,000 per session)');
    assert(q?.includes('per session'), `expected per session qualifier, got: ${q}`);
  });

  await runTest('Phase 7: extractPriceQualifier detects outside comparison', () => {
    const q = extractPriceQualifier('2,522,550 (#4M outside)');
    assert(q?.includes('outside'), `expected outside qualifier, got: ${q}`);
  });

  await runTest('Phase 7: extractPriceQualifier returns undefined for plain prices', () => {
    assert.strictEqual(extractPriceQualifier('5,000'), undefined);
  });

  await runTest('Phase 7 Integration: YELLOW GOLD CROWN has qualifier', () => {
    const record = findRecord(lasuthRecords, 'YELLOW GOLD CROWN');
    assert(record, 'expected YELLOW GOLD CROWN record');
    assert.strictEqual(record!.price, 91000, `expected price 91000, got ${record!.price}`);
    assert(
      record!.metadata?.priceQualifier?.includes('depending on market value') ||
      record!.metadata?.rawPriceText?.includes('depending on market value'),
      'expected market value qualifier in metadata'
    );
  });

  // =============================================
  // Phase 4: Tag Hydration (inline tests)
  // =============================================

  const makePriceData = (overrides: Partial<PriceData>): PriceData => ({
    id: 'test',
    facilityName: 'Lagos State University Teaching Hospital',
    procedureCode: 'TEST',
    procedureDescription: '',
    price: 1000,
    currency: 'NGN',
    effectiveDate: new Date(),
    lastUpdated: new Date(),
    source: 'test',
    ...overrides,
  });

  await runTest('Phase 4: ophthalmology procedures get ophthalmology tag', () => {
    const item = makePriceData({ procedureDescription: 'CATARACT SURGERY', metadata: { area: 'OPHTHAMOLOGY' } });
    const { tags } = hydrateTags(item);
    assert(tags.includes('ophthalmology') || tags.includes('eye_care'), `expected ophthalmology tag, got: ${tags}`);
  });

  await runTest('Phase 4: ENT procedures get ent tag', () => {
    const item = makePriceData({ procedureDescription: 'TONSILLECTOMY', metadata: { area: 'ENT' } });
    const { tags } = hydrateTags(item);
    assert(tags.includes('ent') || tags.includes('ear_nose_throat'), `expected ent tag, got: ${tags}`);
  });

  await runTest('Phase 4: dermatology procedures get dermatology tag', () => {
    const item = makePriceData({ procedureDescription: 'SKIN BIOPSY', metadata: { area: 'DERMATOLOGY' } });
    const { tags } = hydrateTags(item);
    assert(tags.includes('dermatology'), `expected dermatology tag, got: ${tags}`);
  });

  await runTest('Phase 4: psychiatry procedures get psychiatry tag', () => {
    const item = makePriceData({ procedureDescription: 'ELECTROCONVULSIVE (ECT)', metadata: { area: 'PSYCHIATRY/ PSYCHOLOGY' } });
    const { tags } = hydrateTags(item);
    assert(tags.includes('psychiatry') || tags.includes('mental_health'), `expected psychiatry tag, got: ${tags}`);
  });

  await runTest('Phase 4: endoscopy procedures get endoscopy tag', () => {
    const item = makePriceData({ procedureDescription: 'DIAGNOSTIC UPPER G.I ENDOSCOPY', metadata: { area: 'ENDOSCOPY' } });
    const { tags } = hydrateTags(item);
    assert(tags.includes('endoscopy') || tags.includes('gastroenterology'), `expected endoscopy tag, got: ${tags}`);
  });

  await runTest('Phase 4: urology procedures get urology tag', () => {
    const item = makePriceData({ procedureDescription: 'CHANGE OF CATHETER', metadata: { area: 'UROLOGY' } });
    const { tags } = hydrateTags(item);
    assert(tags.includes('urology'), `expected urology tag, got: ${tags}`);
  });

  await runTest('Phase 4: oncology procedures get oncology tag', () => {
    const item = makePriceData({ procedureDescription: 'CHEMO. ADM. FEE', metadata: { area: 'ONCOLOGY' } });
    const { tags } = hydrateTags(item);
    assert(tags.includes('oncology'), `expected oncology tag, got: ${tags}`);
  });

  await runTest('Phase 4: stroke procedures get stroke/neurology tag', () => {
    const item = makePriceData({ procedureDescription: 'STROKE WARD (FIRST 10 DAYS)', metadata: { area: 'STROKE WARD' } });
    const { tags } = hydrateTags(item);
    assert(tags.includes('stroke') || tags.includes('neurology'), `expected stroke/neurology tag, got: ${tags}`);
  });

  // =============================================
  // Phase 8: Integration Validation
  // =============================================

  await runTest('Phase 8: LASUTH CSV produces reasonable record count', () => {
    assert(lasuthRecords.length >= 400, `expected >= 400 records, got ${lasuthRecords.length}`);
    assert(lasuthRecords.length <= 700, `expected <= 700 records, got ${lasuthRecords.length}`);
  });

  await runTest('Phase 8: all expected areas are represented', () => {
    const areas = new Set(lasuthRecords.map((r) => r.metadata?.area).filter(Boolean));
    const expectedAreas = [
      'ACCOMMODATION', 'ADMISSION AND EMERGENCY TREATMENT', 'ALL DIET FEES',
      'AMBULANCE RATE', 'CSSD', 'DENTAL UNIT', 'DERMATOLOGY',
      'DIETARY CLINIC', 'EEG SERVICES', 'ENDOSCOPY',
      'ENT', 'FAMILY MEDICINE', 'ICU', 'ONCOLOGY',
      'ORTHOPAEDICS', 'PHYSIOTHERAPY', 'REPORTS',
      'STROKE WARD', 'UROLOGY', 'VIP SERVICES',
    ];
    const areaArr = Array.from(areas);
    for (const expected of expectedAreas) {
      const found = areaArr.some((a) => a?.toUpperCase().includes(expected));
      assert(found, `expected area "${expected}" to be represented, areas: ${areaArr.join(', ')}`);
    }
  });

  await runTest('Phase 8: no unexpected zero-price records', () => {
    const zeroPrices = lasuthRecords.filter((r) => r.price === 0);
    for (const record of zeroPrices) {
      const rawText = (record.metadata?.rawPriceText || '').toLowerCase();
      const tier = record.metadata?.priceTier || '';
      const isFreeExpected = rawText.includes('free') || tier === 'free' || tier === 'paediatric';
      assert(isFreeExpected,
        `unexpected zero price for "${record.procedureDescription}" (raw: "${rawText}", tier: "${tier}")`
      );
    }
  });

  await runTest('Phase 8: simple prices still parse correctly', () => {
    const generalWard = findRecord(lasuthRecords, 'GENERAL WARD ACCOMMODATION');
    assert(generalWard, 'expected GENERAL WARD ACCOMMODATION');
    assert.strictEqual(generalWard!.price, 5000, `expected 5000, got ${generalWard!.price}`);

    const ambulanceOut = findRecord(lasuthRecords, 'OUTSIDE LASUTH');
    assert(ambulanceOut, 'expected OUTSIDE LASUTH');
    assert.strictEqual(ambulanceOut!.price, 15000, `expected 15000, got ${ambulanceOut!.price}`);
  });

  await runTest('Phase 8: high-value items parse correctly', () => {
    const orthodontic = findRecord(lasuthRecords, 'UPPER AND LOWER FIXED APPLIANCE');
    assert(orthodontic, 'expected UPPER AND LOWER FIXED APPLIANCE');
    assert.strictEqual(orthodontic!.price, 600000, `expected 600000, got ${orthodontic!.price}`);
  });

  // Summary
  const uniqueAreas = new Set(lasuthRecords.map((r) => r.metadata?.area).filter(Boolean));
  const uniqueCategories = new Set(lasuthRecords.map((r) => r.metadata?.category).filter(Boolean));
  const codes = lasuthRecords.map((r) => r.procedureCode);
  const uniqueCodes = new Set(codes);
  const zeroPrices = lasuthRecords.filter((r) => r.price === 0);

  console.log(`\n--- LASUTH Summary ---`);
  console.log(`Total records: ${lasuthRecords.length}`);
  console.log(`Unique areas: ${uniqueAreas.size} (${Array.from(uniqueAreas).slice(0, 5).join(', ')}...)`);
  console.log(`Unique categories: ${uniqueCategories.size}`);
  console.log(`Unique procedure codes: ${uniqueCodes.size} / ${codes.length} (${codes.length - uniqueCodes.size} collisions)`);
  console.log(`Zero-price records: ${zeroPrices.length}`);

  if (process.exitCode) {
    process.exit(1);
  }
}

main().catch((err) => {
  console.error(err);
  process.exit(1);
});
