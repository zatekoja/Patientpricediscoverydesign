import fs from 'fs';
import path from 'path';
import AdmZip from 'adm-zip';
import { XMLParser } from 'fast-xml-parser';
import { parse as csvParse } from 'csv-parse/sync';
import { parse as csvParseStream } from 'csv-parse';
import * as xlsx from 'xlsx';
import { PriceData } from '../types/PriceData';
import { buildFacilityId, normalizeFacilityName } from './facilityIds';

export interface PriceListParseContext {
  facilityName?: string;
  sourceFile?: string;
  currency?: string;
  defaultEffectiveDate?: Date;
  providerId?: string;
  /** Explicit facility name mapping, bypasses inference */
  explicitFacilityMapping?: Record<string, string>;
  /** Minimum confidence threshold for facility inference */
  facilityInferenceThreshold?: number;
}

interface HeaderMap {
  descriptionIndex?: number;
  priceIndex?: number;
  areaIndex?: number;
}

interface PriceVariant {
  price: number;
  tier?: string;
  unit?: string;
  rawText: string;
  qualifier?: string;
}

interface AreaHierarchy {
  parentArea: string;
  subCategory?: string;
}

function parseAreaHierarchy(areaCell: string): AreaHierarchy {
  // Pattern 1: Colon-separated (e.g., "DENTAL UNIT: ORAL AND MAXILLOFACIAL SURGERY")
  const colonIndex = areaCell.indexOf(':');
  if (colonIndex > 0) {
    const parent = areaCell.slice(0, colonIndex).trim();
    const child = areaCell.slice(colonIndex + 1).trim();
    if (parent && child) {
      return { parentArea: parent, subCategory: child };
    }
  }
  // Pattern 2: Parenthetical (e.g., "VIP SERVICES (ACCELERATED CARE)")
  const parenMatch = areaCell.match(/^(.+?)\s*\(([^)]+)\)\s*$/);
  if (parenMatch) {
    return { parentArea: parenMatch[1].trim(), subCategory: parenMatch[2].trim() };
  }
  return { parentArea: areaCell };
}

export async function parseCsvFile(filePath: string, context?: PriceListParseContext): Promise<PriceData[]> {
  const rows = await parseCsvFileRows(filePath);
  return rowsToPriceData(rows, buildContext(filePath, context));
}

export async function parseDocxFile(filePath: string, context?: PriceListParseContext): Promise<PriceData[]> {
  const buffer = await fs.promises.readFile(filePath);
  const rows = parseDocxBuffer(buffer);
  return rowsToPriceData(rows, buildContext(filePath, context));
}

export async function parseXlsxFile(filePath: string, context?: PriceListParseContext): Promise<PriceData[]> {
  const workbook = xlsx.readFile(filePath);
  const sheetName = workbook.SheetNames[0];
  if (!sheetName) {
    return [];
  }
  const sheet = workbook.Sheets[sheetName];
  const rows = xlsx.utils.sheet_to_json(sheet, { header: 1, raw: true }) as any[][];
  return rowsToPriceData(rows, buildContext(filePath, context));
}

export function parseCsvContent(content: string): string[][] {
  try {
    // Use csv-parse library for RFC-compliant CSV parsing
    // This handles quoted fields, escaped characters, and multi-line cells properly
    const records = csvParse(content, {
      relax_quotes: true,
      skip_empty_lines: false,
      trim: false,
      relax_column_count: true,
    });
    return records;
  } catch (error) {
    console.error('CSV parsing error:', error);
    // Fallback to simple parsing if csv-parse fails
    return content.split(/\r?\n/).map(line => parseCsvLine(line));
  }
}

export function parseDocxBuffer(buffer: Buffer): string[][] {
  const zip = new AdmZip(buffer);
  const entry = zip.getEntry('word/document.xml');
  if (!entry) {
    return [];
  }
  const xml = entry.getData().toString('utf-8');
  const paragraphRows = extractDocxParagraphTexts(xml);
  const tableRows = extractDocxTableRows(xml);
  return [...paragraphRows, ...tableRows];
}

function buildContext(filePath: string, context?: PriceListParseContext): PriceListParseContext {
  const sourceFile = context?.sourceFile || path.basename(filePath);
  return {
    facilityName: context?.facilityName,
    sourceFile,
    currency: context?.currency || 'NGN',
    defaultEffectiveDate: context?.defaultEffectiveDate,
    providerId: context?.providerId,
    explicitFacilityMapping: context?.explicitFacilityMapping,
    facilityInferenceThreshold: context?.facilityInferenceThreshold,
  };
}

interface PreparedContext {
  normalizedRows: string[][];
  rawRows: string[][];
  headerIndex: number;
  headerMap: HeaderMap;
  facilityName: string;
  facilityId: string | undefined;
  effectiveDate: Date;
  currency: string;
  sourceFile?: string;
}

function prepareParseContext(rows: string[][], context: PriceListParseContext): PreparedContext | null {
  const normalizedRows = rows.map((row) => row.map(normalizeCell));
  const rawRows = rows.map((row) => row.map(normalizeCellPreserveNewlines));
  const headerIndex = findHeaderRow(normalizedRows);
  const headerRow = headerIndex >= 0 ? normalizedRows[headerIndex] : [];
  const headerMap = buildHeaderMap(headerRow);

  let facilityName: string;
  if (context.explicitFacilityMapping && context.sourceFile) {
    const mappedName = context.explicitFacilityMapping[context.sourceFile];
    if (mappedName) {
      facilityName = mappedName;
      console.log(`Using explicit facility mapping: ${facilityName}`);
    } else {
      facilityName = context.facilityName ||
        inferFacilityName(
          normalizedRows.slice(0, headerIndex > 0 ? headerIndex : 10),
          context.sourceFile,
          context.facilityInferenceThreshold
        );
    }
  } else {
    facilityName = context.facilityName ||
      inferFacilityName(
        normalizedRows.slice(0, headerIndex > 0 ? headerIndex : 10),
        context.sourceFile,
        context.facilityInferenceThreshold
      );
  }

  const normalizedFacilityName = normalizeFacilityName(facilityName, context.sourceFile);
  if (!normalizedFacilityName) {
    console.warn(`Skipping price list; unable to normalize facility name from ${context.sourceFile || 'unknown source'}`);
    return null;
  }

  const facilityId = buildFacilityId(context.providerId, normalizedFacilityName);
  const effectiveDate =
    context.defaultEffectiveDate ||
    inferEffectiveDate(normalizedRows.slice(0, headerIndex > 0 ? headerIndex : 10), context.sourceFile);

  return {
    normalizedRows,
    rawRows,
    headerIndex,
    headerMap,
    facilityName: normalizedFacilityName,
    facilityId: facilityId || undefined,
    effectiveDate,
    currency: context.currency || 'NGN',
    sourceFile: context.sourceFile,
  };
}

type CsvFormat = 'columnar' | 'hierarchical' | 'flat';

function detectCsvFormat(headerIndex: number, headerMap: HeaderMap): CsvFormat {
  if (headerIndex < 0) {
    return 'hierarchical';
  }
  if (headerMap.areaIndex !== undefined) {
    return 'columnar';
  }
  return 'flat';
}

function rowsToPriceData(rows: string[][], context: PriceListParseContext): PriceData[] {
  const prepared = prepareParseContext(rows, context);
  if (!prepared) {
    return [];
  }

  const format = detectCsvFormat(prepared.headerIndex, prepared.headerMap);

  switch (format) {
    case 'hierarchical':
      return parseHierarchicalRows(prepared);
    case 'columnar':
    case 'flat':
      return parseHeaderBasedRows(prepared);
  }
}

function parseHeaderBasedRows(ctx: PreparedContext): PriceData[] {
  const mergedRows = mergeContinuationRows(ctx.normalizedRows, ctx.headerIndex, ctx.headerMap);
  const mergedRawRows = mergeContinuationRows(ctx.rawRows, ctx.headerIndex, ctx.headerMap);
  let currentArea: string | undefined;
  let currentCategory: string | undefined;
  const priceData: PriceData[] = [];

  const startIndex = ctx.headerIndex >= 0 ? ctx.headerIndex + 1 : 0;
  for (let i = startIndex; i < mergedRows.length; i += 1) {
    const row = mergedRows[i];
    if (!row || row.every((cell) => !cell)) {
      continue;
    }

    if (ctx.headerMap.areaIndex !== undefined) {
      const areaCell = row[ctx.headerMap.areaIndex] || '';
      if (areaCell) {
        const hierarchy = parseAreaHierarchy(areaCell);
        currentArea = hierarchy.parentArea;
        if (hierarchy.subCategory) {
          currentCategory = hierarchy.subCategory;
        }
      }
    }

    const description = findDescription(row, ctx.headerMap);
    const rawRow = mergedRawRows[i] || row;
    const priceText = findPriceText(rawRow, ctx.headerMap);

    if (description && description.endsWith(':') && !priceText) {
      currentCategory = description.replace(/:$/, '').trim();
      continue;
    }

    if (!description || !priceText) {
      continue;
    }

    const variants = expandPriceVariants(priceText, description);
    for (const variant of variants) {
      const procedureCode = buildProcedureCode(description, i);
      const id = buildId(ctx.facilityName, description, variant, i);
      priceData.push({
        id,
        facilityName: ctx.facilityName,
        facilityId: ctx.facilityId,
        procedureCode,
        procedureDescription: description,
        price: variant.price,
        currency: ctx.currency,
        effectiveDate: ctx.effectiveDate,
        lastUpdated: new Date(),
        source: 'file_price_list',
        metadata: {
          sourceFile: ctx.sourceFile,
          area: currentArea,
          category: currentCategory,
          unit: variant.unit,
          priceTier: variant.tier,
          rawPriceText: variant.rawText,
          priceQualifier: variant.qualifier,
          rowNumber: i + 1,
        },
      });
    }
  }

  return priceData;
}

// --- Hierarchical parser for MEGALEK/IJEDE-style CSVs ---
// These CSVs have no header row and use parent-child row groups:
//   16,MYMECTOMY, HYSTERECTOMY, TAH,,,       ← Parent (S/N, description, no price)
//   ,SURGICAL PACK," 130,000.00 ",,           ← Sub-item
//   ,OPERATION FEE," 85,000.00 ",,            ← Sub-item
//   ,ANAESTHESIA," 150,000.00 "," 450,000.00" ← Sub-item (col D = running total)

interface HierarchicalGroup {
  parentDescription: string;
  parentSn: string;
  subItems: Array<{ description: string; priceC: number; priceD: number; raw: string[] }>;
  rowIndex: number;
}

function isHierarchicalSectionHeader(row: string[]): string | null {
  // Use only colB for section detection — other columns may have stray symbols (#)
  const text = (row[1] || '').trim().toUpperCase();
  if (!text) return null;
  const sectionPatterns = [
    /NEW PRICE LIST FOR\s+(.+?)(?:\s+SECTION)?$/i,
    /^(PAEDIATRICS)\s+WARD/i,
    /^(PHYSIOTHERAPHY?)\s+DEPT/i,
    /^(SCAN)\s*:/i,
  ];
  for (const pat of sectionPatterns) {
    const match = text.match(pat);
    if (match) return match[1].trim();
  }
  return null;
}

function isHierarchicalSkipRow(row: string[]): boolean {
  const joined = row.join(' ').trim().toUpperCase();
  if (!joined) return true;
  if (/^\s*YEAR\s+20\d{2}/i.test(joined)) return true;
  if (/SIGN\s*BY/i.test(joined)) return true;
  if (/NOTE\s*:/i.test(joined)) return true;
  if (/PRICE\s+LIST/i.test(joined) && /HOSPITAL/i.test(joined)) return true;
  if (/ADJUSTED\s+PRICE/i.test(joined)) return true;
  if (/MANAGEMENT\s*:/i.test(joined)) return true;
  if (/MEDICAL\s+DIRECTOR/i.test(joined)) return true;
  return false;
}

function isSerialNumber(value: string): boolean {
  return /^\d+[a-z]?$/i.test(value.trim());
}

function extractCellPrice(cell: string): number {
  if (!cell) return 0;
  const trimmed = cell.trim();
  // Tier/age labels should not be treated as prices
  if (/\d+\s*-?\s*\d*\s*yrs/i.test(trimmed)) return 0;  // "0-12YRS", "18YRS ABOVE"
  if (/^\d+\s*yrs/i.test(trimmed)) return 0;             // "18YRS ABOVE"
  if (/^(simple|complex|small|big|free)\s*$/i.test(trimmed)) return 0;
  if (/^\(?(out|in)\s*patient\)?/i.test(trimmed)) return 0;
  if (/^(each|per\s+(day|hour|session|visit|week))/i.test(trimmed)) return 0;
  const cleaned = cell.replace(/[#\t\s]/g, '').trim();
  const nums = extractNumbers(cleaned);
  return nums.length > 0 ? nums[nums.length - 1] : 0;
}

function parseHierarchicalRows(ctx: PreparedContext): PriceData[] {
  const rows = ctx.normalizedRows;
  const priceData: PriceData[] = [];
  let currentArea: string | undefined;
  let currentCategory: string | undefined;

  // First pass: classify rows and group parent-child relationships
  let i = 0;
  while (i < rows.length) {
    const row = rows[i];
    if (!row || row.every((cell) => !cell)) {
      i++;
      continue;
    }

    // Skip header/meta rows
    if (isHierarchicalSkipRow(row)) {
      i++;
      continue;
    }

    // Check for section headers
    const section = isHierarchicalSectionHeader(row);
    if (section) {
      currentArea = section.replace(/\s+SECTION$/i, '').trim();
      i++;
      continue;
    }

    const colA = (row[0] || '').trim();
    const colB = (row[1] || '').trim();
    const colC = (row[2] || '').trim();
    const colD = (row[3] || '').trim();

    // --- Numbered rows (serial number in colA) — checked BEFORE category ---
    const TIER_LABEL_RE = /\d+\s*-?\s*\d*\s*yrs|simple|complex|out\s*patient|in\s*patient|small|big/i;
    const isTierHeader = colC && containsLetters(colC) && TIER_LABEL_RE.test(colC);

    if (isTierHeader && isSerialNumber(colA)) {
      // Tiered item header (e.g., "14,MEDICAL CERTIFICATE (NYSC),0-12YRS,18YRS ABOVE")
      const descClean = colB.replace(/:$/, '').trim();
      const tierLabelC = colC;
      const tierLabelD = colD;
      const subItems: Array<{ desc: string; priceC: number; priceD: number }> = [];
      let j = i + 1;
      while (j < rows.length) {
        const subRow = rows[j];
        if (!subRow || subRow.every((c) => !c)) { j++; continue; }
        const subA = (subRow[0] || '').trim();
        const subB = (subRow[1] || '').trim();
        if (subA && isSerialNumber(subA)) break;
        if (isHierarchicalSectionHeader(subRow)) break;
        if (isHierarchicalSkipRow(subRow)) { j++; continue; }
        const pC = extractCellPrice(subRow[2] || '');
        const pD = extractCellPrice(subRow[3] || '');
        // Empty/short description with prices → implicit TOTAL row
        if ((!subB || subB.length <= 1) && (pC > 0 || pD > 0)) {
          subItems.push({ desc: '__TOTAL__', priceC: pC, priceD: pD });
          j++;
          continue;
        }
        if (!subB || subB.length <= 1) { j++; continue; }
        if (pC > 0 || pD > 0 || subB.toUpperCase() === 'TOTAL') {
          const d = subB.toUpperCase() === 'TOTAL' ? 'TOTAL' : subB;
          subItems.push({ desc: d, priceC: pC, priceD: pD });
        }
        j++;
      }

      // Find TOTAL (explicit "TOTAL" or implicit empty-desc total)
      const totalItem = subItems.find((s) => s.desc === 'TOTAL') || subItems.find((s) => s.desc === '__TOTAL__');
      const nonTotalItems = subItems.filter((s) => s.desc !== 'TOTAL' && s.desc !== '__TOTAL__');

      if (totalItem) {
        // Consolidated tier records from TOTAL prices
        const breakdown = nonTotalItems.map((s) => ({ item: s.desc, price: s.priceC }));
        if (totalItem.priceC > 0) {
          emitRecord(priceData, ctx, descClean, totalItem.priceC, tierLabelC, currentArea, currentCategory, i, undefined, undefined, undefined, breakdown);
        }
        if (totalItem.priceD > 0 && totalItem.priceD !== totalItem.priceC) {
          emitRecord(priceData, ctx, descClean, totalItem.priceD, tierLabelD, currentArea, currentCategory, i, undefined, undefined, undefined, breakdown);
        }
      } else if (nonTotalItems.length > 0) {
        // No TOTAL — emit individual sub-items per tier
        for (const sub of nonTotalItems) {
          const fullDesc = `${descClean} - ${sub.desc}`;
          if (sub.priceC > 0) {
            emitRecord(priceData, ctx, fullDesc, sub.priceC, tierLabelC, currentArea, currentCategory, i);
          }
          if (sub.priceD > 0) {
            emitRecord(priceData, ctx, fullDesc, sub.priceD, tierLabelD, currentArea, currentCategory, i);
          }
        }
      }

      i = j;
      continue;
    }

    // Numbered row with S/N (not a tier header)
    if (!isTierHeader && isSerialNumber(colA) && colB && colB.length > 1 && containsLetters(colB)) {
      const descCleanN = colB.replace(/:$/, '').trim();
      const hasPrice = extractCellPrice(colC) > 0 || extractCellPrice(colD) > 0;

      if (hasPrice) {
        const price = extractCellPrice(colD) || extractCellPrice(colC);
        const priceText = colD || colC;
        const variants = expandPriceVariants(priceText, descCleanN);
        if (variants.length > 0) {
          for (const v of variants) {
            emitRecord(priceData, ctx, descCleanN, v.price, v.tier, currentArea, currentCategory, i, v.unit, v.rawText, v.qualifier);
          }
        } else {
          emitRecord(priceData, ctx, descCleanN, price, undefined, currentArea, currentCategory, i);
        }
        i++;
        continue;
      }

      // No price — parent procedure or category, collect sub-items
      const SURGICAL_COMPONENTS = new Set(['surgical pack', 'operation fee', 'theatre pack', 'anaesthesia']);
      const groupItems: Array<{ description: string; priceC: number; priceD: number; raw: string[] }> = [];
      let j = i + 1;
      while (j < rows.length) {
        const subRow = rows[j];
        if (!subRow || subRow.every((c) => !c)) { j++; continue; }
        const subA = (subRow[0] || '').trim();
        const subB = (subRow[1] || '').trim();
        if (subA && isSerialNumber(subA)) break;
        if (isHierarchicalSectionHeader(subRow)) break;
        if (isHierarchicalSkipRow(subRow)) { j++; continue; }
        const pC = extractCellPrice(subRow[2] || '');
        const pD = extractCellPrice(subRow[3] || '');
        // Empty/short description with prices → implicit TOTAL row
        if ((!subB || subB.length <= 1) && (pC > 0 || pD > 0)) {
          groupItems.push({ description: '__TOTAL__', priceC: pC, priceD: pD, raw: subRow });
          j++;
          continue;
        }
        if (!subB || (subB.length <= 1 && !containsNumber(subRow[2] || '') && !containsNumber(subRow[3] || ''))) {
          j++;
          continue;
        }
        if (pC > 0 || pD > 0 || subB.toUpperCase() === 'TOTAL') {
          groupItems.push({ description: subB, priceC: pC, priceD: pD, raw: subRow });
        }
        j++;
      }

      if (groupItems.length > 0) {
        const totalItem = groupItems.find((s) => s.description.toUpperCase() === 'TOTAL') ||
          groupItems.find((s) => s.description === '__TOTAL__');
        const nonTotalItems = groupItems.filter(
          (s) => s.description.toUpperCase() !== 'TOTAL' && s.description !== '__TOTAL__',
        );
        const isSurgical = nonTotalItems.some((s) => SURGICAL_COMPONENTS.has(s.description.toLowerCase().trim()));
        const lastItem = nonTotalItems[nonTotalItems.length - 1];
        const hasRunningTotal = lastItem && lastItem.priceD > 0 && lastItem.priceD > lastItem.priceC;

        if (isSurgical || hasRunningTotal || totalItem) {
          // Consolidate: cost breakdown (surgical or delivery package)
          const breakdown = nonTotalItems.map((s) => ({ item: s.description, price: s.priceC }));
          let totalPrice: number;

          if (totalItem) {
            if (totalItem.priceC > 0 && totalItem.priceD > 0 && totalItem.priceC !== totalItem.priceD) {
              emitRecord(priceData, ctx, descCleanN, totalItem.priceC, 'simple', currentArea, currentCategory, i, undefined, undefined, undefined, breakdown);
              emitRecord(priceData, ctx, descCleanN, totalItem.priceD, 'complex', currentArea, currentCategory, i, undefined, undefined, undefined, breakdown);
              i = j;
              continue;
            }
            totalPrice = totalItem.priceD || totalItem.priceC;
          } else if (hasRunningTotal) {
            totalPrice = lastItem.priceD;
          } else {
            totalPrice = nonTotalItems.reduce((sum, s) => sum + s.priceC, 0);
          }

          emitRecord(priceData, ctx, descCleanN, totalPrice, undefined, currentArea, currentCategory, i, undefined, undefined, undefined, breakdown);
        } else {
          // Not surgical — emit sub-items individually, parent becomes category
          currentCategory = descCleanN;
          for (const sub of nonTotalItems) {
            const price = sub.priceC || sub.priceD;
            if (price > 0) {
              emitRecord(priceData, ctx, sub.description, price, undefined, currentArea, currentCategory, i);
            }
          }
        }
      }

      i = j;
      continue;
    }

    // --- Category headers (unnumbered, colB ends with :, no prices) ---
    if (!colA && colB && colB.endsWith(':') && !extractCellPrice(colC) && !extractCellPrice(colD)) {
      const catName = colB.replace(/:$/, '').trim();
      if (catName.length > 2 && containsLetters(catName)) {
        currentCategory = catName;
      }
      i++;
      continue;
    }

    // --- Unnumbered row with description and price (PAD, UNDERLAY, etc.) ---
    if (!colA && colB && colB.length > 1 && containsLetters(colB)) {
      const price = extractCellPrice(colC) || extractCellPrice(colD);
      if (price > 0) {
        const priceText = colC || colD;
        const variants = expandPriceVariants(priceText, colB);
        if (variants.length > 0) {
          for (const v of variants) {
            emitRecord(priceData, ctx, colB, v.price, v.tier, currentArea, currentCategory, i, v.unit, v.rawText, v.qualifier);
          }
        } else {
          emitRecord(priceData, ctx, colB, price, undefined, currentArea, currentCategory, i);
        }
        i++;
        continue;
      }
      // Description-only row with no price — treat as category
      if (colB.length > 2 && containsLetters(colB)) {
        currentCategory = colB;
      }
    }

    // Lettered sub-items (a, b, c, d) that are standalone with description in colB and price in colC/colD
    if (/^[a-z]$/i.test(colA) && colB && colB.length > 1 && containsLetters(colB)) {
      const price = extractCellPrice(colC) || extractCellPrice(colD);
      if (price > 0) {
        emitRecord(priceData, ctx, colB, price, undefined, currentArea, currentCategory, i);
        i++;
        continue;
      }
    }

    i++;
  }

  return priceData;
}

function emitRecord(
  priceData: PriceData[],
  ctx: PreparedContext,
  description: string,
  price: number,
  tier: string | undefined,
  area: string | undefined,
  category: string | undefined,
  rowIndex: number,
  unit?: string,
  rawPriceText?: string,
  qualifier?: string,
  breakdown?: Array<{ item: string; price: number }>,
): void {
  const procedureCode = buildProcedureCode(description, rowIndex);
  const variant: PriceVariant = { price, tier, unit, rawText: rawPriceText || String(price), qualifier };
  const id = buildId(ctx.facilityName, description, variant, rowIndex);
  const metadata: Record<string, any> = {
    sourceFile: ctx.sourceFile,
    area,
    category,
    unit: unit || extractUnit(`${description} ${price}`),
    priceTier: tier,
    rawPriceText: rawPriceText || String(price),
    priceQualifier: qualifier,
    rowNumber: rowIndex + 1,
  };
  if (breakdown && breakdown.length > 0) {
    metadata.breakdown = breakdown;
  }
  priceData.push({
    id,
    facilityName: ctx.facilityName,
    facilityId: ctx.facilityId,
    procedureCode,
    procedureDescription: description,
    price,
    currency: ctx.currency,
    effectiveDate: ctx.effectiveDate,
    lastUpdated: new Date(),
    source: 'file_price_list',
    metadata,
  });
}

/**
 * Legacy CSV line parser - kept as fallback for edge cases
 * Note: This parser has limitations with quoted fields containing commas.
 * Use parseCsvContent() which uses csv-parse library for RFC-compliant parsing.
 */
function parseCsvLine(line: string): string[] {
  const cells: string[] = [];
  let current = '';
  let inQuotes = false;

  for (let i = 0; i < line.length; i += 1) {
    const char = line[i];
    if (char === '\"') {
      if (inQuotes && line[i + 1] === '\"') {
        current += '\"';
        i += 1;
      } else {
        inQuotes = !inQuotes;
      }
      continue;
    }
    if (char === ',' && !inQuotes) {
      const prev = line[i - 1];
      const next = line[i + 1];
      const next3 = line.slice(i + 1, i + 4);
      if (prev && /\d/.test(prev) && next && /^\d{3}$/.test(next3)) {
        current += ',';
        continue;
      }
      cells.push(current);
      current = '';
      continue;
    }
    current += char;
  }
  cells.push(current);
  return cells;
}

async function parseCsvFileRows(filePath: string): Promise<string[][]> {
  return new Promise((resolve, reject) => {
    const records: string[][] = [];
    let settled = false;
    const parser = csvParseStream({
      relax_quotes: true,
      skip_empty_lines: false,
      trim: false,
      relax_column_count: true,
    });

    const stream = fs.createReadStream(filePath);
    stream.on('error', (error) => {
      if (settled) {
        return;
      }
      settled = true;
      reject(error);
    });
    parser.on('data', (row: string[]) => records.push(row));
    parser.on('error', async () => {
      if (settled) {
        return;
      }
      settled = true;
      stream.destroy();
      try {
        const content = await fs.promises.readFile(filePath, 'utf-8');
        resolve(parseCsvContent(content));
      } catch (fallbackError) {
        reject(fallbackError);
      }
    });
    parser.on('end', () => {
      if (settled) {
        return;
      }
      settled = true;
      resolve(records);
    });

    stream.pipe(parser);
  });
}

function extractDocxTableRows(xml: string): string[][] {
  const parser = new XMLParser({ ignoreAttributes: false });
  const doc = parser.parse(xml);
  const tables = collectNodes(doc, 'w:tbl');
  const rows: string[][] = [];

  for (const table of tables) {
    const tableRows = collectNodes(table, 'w:tr');
    for (const row of tableRows) {
      const cells = collectNodes(row, 'w:tc');
      const rowValues: string[] = [];
      for (const cell of cells) {
        const texts = collectNodes(cell, 'w:t');
        const text = texts.map((t) => (typeof t === 'string' ? t : t['#text'] || '')).join('');
        rowValues.push(normalizeCell(text));
      }
      if (rowValues.some((value) => value)) {
        rows.push(rowValues);
      }
    }
  }
  return rows;
}

function extractDocxParagraphTexts(xml: string): string[][] {
  const parser = new XMLParser({ ignoreAttributes: false });
  const doc = parser.parse(xml);
  const body = doc?.['w:document']?.['w:body'];
  if (!body) return [];

  const paragraphs: string[][] = [];
  const pNodes = body['w:p'];
  if (!pNodes) return [];

  const pList = Array.isArray(pNodes) ? pNodes : [pNodes];
  for (const p of pList) {
    const texts = collectNodes(p, 'w:t');
    const text = texts.map((t: any) => (typeof t === 'string' ? t : t['#text'] || '')).join('');
    const normalized = normalizeCell(text);
    if (normalized) {
      paragraphs.push([normalized]);
    }
  }
  return paragraphs;
}

function collectNodes(node: any, key: string): any[] {
  const results: any[] = [];
  if (!node) {
    return results;
  }
  if (Array.isArray(node)) {
    for (const item of node) {
      results.push(...collectNodes(item, key));
    }
    return results;
  }
  if (typeof node === 'object') {
    for (const [k, value] of Object.entries(node)) {
      if (k === key) {
        if (Array.isArray(value)) {
          results.push(...value);
        } else {
          results.push(value);
        }
      } else if (typeof value === 'object') {
        results.push(...collectNodes(value, key));
      }
    }
  }
  return results;
}

function normalizeCell(value: unknown): string {
  return String(value ?? '').replace(/\s+/g, ' ').trim();
}

function normalizeCellPreserveNewlines(value: unknown): string {
  return String(value ?? '').replace(/[^\S\n]+/g, ' ').replace(/\n\s*/g, '\n').trim();
}

function containsLetters(value: string): boolean {
  return /[A-Za-z]/.test(value);
}

function isHeaderLike(cell: string): boolean {
  const value = cell.toLowerCase();
  return value.includes('price') || value.includes('amount') || value.includes('description');
}

function findHeaderRow(rows: string[][]): number {
  return rows.findIndex((row) => {
    const lowerCells = row.map((cell) => cell.toLowerCase());
    const hasDesc = lowerCells.some((cell) =>
      ['description', 'procedures', 'procedure', 'service', 'revenue'].some((k) => cell.includes(k))
    );
    const hasPrice = lowerCells.some((cell) =>
      ['price', 'amount', 'rate', 'fee'].some((k) => cell.includes(k))
    );
    if (!hasDesc || !hasPrice) {
      return false;
    }
    const headerCells = lowerCells.filter((cell) =>
      ['description', 'procedures', 'procedure', 'service', 'revenue', 'price', 'amount', 'rate', 'fee', 'area', 'category']
        .some((k) => cell.includes(k))
    ).length;
    return headerCells >= 2;
  });
}

function buildHeaderMap(headerRow: string[]): HeaderMap {
  const headerLower = headerRow.map((cell) => cell.toLowerCase());
  const descriptionIndex = headerLower.findIndex((cell) =>
    ['description', 'procedures', 'procedure', 'service', 'revenue'].some((k) => cell.includes(k))
  );
  const priceIndex = headerLower.findIndex((cell) =>
    ['price', 'amount', 'rate', 'fee'].some((k) => cell.includes(k))
  );
  const areaIndex = headerLower.findIndex((cell) =>
    ['area', 'category', 'section', 'department'].some((k) => cell.includes(k))
  );
  return {
    descriptionIndex: descriptionIndex >= 0 ? descriptionIndex : undefined,
    priceIndex: priceIndex >= 0 ? priceIndex : undefined,
    areaIndex: areaIndex >= 0 ? areaIndex : undefined,
  };
}

function mergeContinuationRows(rows: string[][], headerIndex: number, headerMap: HeaderMap): string[][] {
  if (headerIndex < 0 || headerMap.descriptionIndex === undefined || headerMap.priceIndex === undefined) {
    return rows;
  }
  const merged: string[][] = rows.slice(0, headerIndex + 1);
  let current: string[] | null = null;

  for (let i = headerIndex + 1; i < rows.length; i += 1) {
    const row = rows[i];
    if (!row || row.every((cell) => !cell)) {
      if (current) {
        merged.push(current);
        current = null;
      }
      continue;
    }

    if (!current) {
      current = row.slice();
      continue;
    }

    if (isNewRecordRow(row, headerMap)) {
      merged.push(current);
      current = row.slice();
      continue;
    }

    current = mergeRowInto(current, row, headerMap);
  }

  if (current) {
    merged.push(current);
  }

  return merged;
}

function isNewRecordRow(row: string[], headerMap: HeaderMap): boolean {
  if (row[0] && isLikelyIndex(row[0])) {
    return true;
  }
  if (headerMap.descriptionIndex !== undefined) {
    const descCell = row[headerMap.descriptionIndex];
    if (descCell && containsLetters(descCell)) {
      return true;
    }
  }
  if (headerMap.priceIndex !== undefined) {
    const priceCell = row[headerMap.priceIndex];
    if (priceCell && (containsNumber(priceCell) || containsFree(priceCell))) {
      return true;
    }
  }
  return false;
}

function mergeRowInto(baseRow: string[], continuationRow: string[], headerMap: HeaderMap): string[] {
  const merged = baseRow.slice();
  const descriptionIndex = headerMap.descriptionIndex ?? 0;
  const priceIndex = headerMap.priceIndex ?? 0;
  const descriptionParts: string[] = [];
  const priceParts: string[] = [];

  for (const cell of continuationRow) {
    if (!cell) {
      continue;
    }
    if (isPriceLike(cell)) {
      priceParts.push(cell);
    } else {
      descriptionParts.push(cell);
    }
  }

  if (descriptionParts.length > 0) {
    merged[descriptionIndex] = appendCell(merged[descriptionIndex], descriptionParts.join(' '));
  }
  if (priceParts.length > 0) {
    merged[priceIndex] = appendCell(merged[priceIndex], priceParts.join(' '));
  }

  return merged;
}

function appendCell(existing: string | undefined, addition: string): string {
  if (!existing) {
    return addition;
  }
  return `${existing}\n${addition}`;
}

function findDescription(row: string[], headerMap: HeaderMap): string {
  if (headerMap.descriptionIndex !== undefined) {
    return row[headerMap.descriptionIndex] || '';
  }
  for (let i = 0; i < row.length; i += 1) {
    const cell = row[i];
    if (!cell) {
      continue;
    }
    if (isLikelyIndex(cell)) {
      continue;
    }
    if (isPriceLike(cell)) {
      continue;
    }
    return cell;
  }
  return '';
}

function findPriceText(row: string[], headerMap: HeaderMap): string {
  if (headerMap.priceIndex !== undefined) {
    const cell = row[headerMap.priceIndex];
    if (cell && (containsNumber(cell) || containsFree(cell))) {
      return cell;
    }
  }
  const hasDescriptiveText = row.some((cell) => cell && containsLetters(cell));
  for (const cell of row) {
    if (cell && isLikelyIndex(cell) && hasDescriptiveText) {
      continue;
    }
    if (cell && (containsNumber(cell) || containsFree(cell))) {
      return cell;
    }
  }
  return '';
}

function containsNumber(value: string): boolean {
  return /\d/.test(value);
}

function containsFree(value: string): boolean {
  return value.toLowerCase().includes('free');
}

function isLikelyIndex(value: string): boolean {
  return /^\d+$/.test(value.trim());
}

function isPriceLike(value: string): boolean {
  if (containsFree(value)) {
    return true;
  }
  return containsNumber(value) && !containsLetters(value);
}

function expandPriceVariants(priceText: string, description: string): PriceVariant[] {
  const unit = extractUnit(`${description} ${priceText}`);
  const qualifier = extractPriceQualifier(priceText);

  // Phase 2: Strip duplicate description prefix from price text
  const cleaned = cleanPriceText(priceText, description);

  // Phase 1: Check for TOTAL line in multi-line breakdowns
  const totalPrice = detectTotalPrice(cleaned);
  if (totalPrice !== null) {
    return [{ price: totalPrice, tier: detectTier(cleaned.toLowerCase()), unit, rawText: priceText, qualifier }];
  }

  const segments = cleaned.split(/\n|;/).map((segment) => segment.trim()).filter(Boolean);
  const variants: PriceVariant[] = [];

  const addVariant = (price: number, tier: string | undefined, rawText: string) => {
    variants.push({ price, tier, unit, rawText, qualifier });
  };

  const parts = segments.length > 0 ? segments : [cleaned];
  for (const part of parts) {
    const lower = part.toLowerCase();
    const numbers = extractNumbers(part);
    if (numbers.length === 0 && lower.includes('free')) {
      addVariant(0, detectTier(lower) || 'free', part);
      continue;
    }

    if (numbers.length === 1) {
      addVariant(numbers[0], detectTier(lower), part);
      continue;
    }

    // Multiple numbers: try splitting by tier indicators (Adult/Paed) first
    const tierSplit = splitByTier(part);
    if (tierSplit.length > 0) {
      for (const ts of tierSplit) {
        addVariant(ts.price, ts.tier, part);
      }
      continue;
    }

    // Fallback: take the first number found
    if (numbers.length > 0) {
      addVariant(numbers[0], detectTier(lower), part);
    }
  }

  return variants.length > 0 ? variants : [];
}

function detectTotalPrice(text: string): number | null {
  const totalMatch = text.match(/\bTOTAL\b\s*[=:\-#N]*\s*#?([\d,]+(?:\.\d+)?)/i);
  if (totalMatch) {
    const price = parseFloat(totalMatch[1].replace(/,/g, ''));
    if (!Number.isNaN(price) && price > 0) {
      return price;
    }
  }
  return null;
}

function cleanPriceText(priceText: string, description: string): string {
  if (!description || !priceText) {
    return priceText;
  }
  // If the price text starts with the same description text, strip it
  const descNorm = description.replace(/[^a-z0-9]/gi, '').toLowerCase();
  const priceNorm = priceText.replace(/[^a-z0-9]/gi, '').toLowerCase();
  if (descNorm.length > 10 && priceNorm.startsWith(descNorm) && priceNorm.length > descNorm.length) {
    // Find the first colon or newline after the description-like prefix
    const colonIndex = priceText.indexOf(':');
    const newlineIndex = priceText.indexOf('\n');
    const cutIndex = colonIndex >= 0 ? colonIndex : newlineIndex;
    if (cutIndex > 0 && cutIndex < priceText.length - 1) {
      return priceText.slice(cutIndex + 1).trim();
    }
  }
  return priceText;
}

function extractPriceQualifier(text: string): string | undefined {
  const qualifiers: string[] = [];
  const marketMatch = text.match(/\(depending on market value\)/i);
  if (marketMatch) {
    qualifiers.push('depending on market value');
  }
  const perSessionMatch = text.match(/\([^)]*per session[^)]*\)/i);
  if (perSessionMatch) {
    qualifiers.push(perSessionMatch[0].replace(/[()]/g, '').trim());
  }
  const outsideMatch = text.match(/\([^)]*outside[^)]*\)/i);
  if (outsideMatch) {
    qualifiers.push(outsideMatch[0].replace(/[()]/g, '').trim());
  }
  const additionalMatch = text.match(/for additional \w+/i);
  if (additionalMatch) {
    qualifiers.push(additionalMatch[0]);
  }
  return qualifiers.length > 0 ? qualifiers.join('; ') : undefined;
}

function splitByTier(text: string): Array<{ price: number; tier: string }> {
  const results: Array<{ price: number; tier: string }> = [];
  const lower = text.toLowerCase();

  // Pattern: "5,000 Adult only\nFree for Paed." or "5,000 Adult 2,000 Paed."
  // Also handles: "10,000 Adult\n\n5,000 Paed."
  const adultMatch = text.match(/([\d,]+(?:\.\d+)?)\s*(?:adult)/i);
  const paedMatch = text.match(/([\d,]+(?:\.\d+)?)\s*(?:paed|pediatric|child)/i);
  const freeForPaed = lower.includes('free') && (lower.includes('paed') || lower.includes('child'));

  if (adultMatch) {
    const price = parseFloat(adultMatch[1].replace(/,/g, ''));
    if (!Number.isNaN(price)) {
      results.push({ price, tier: 'adult' });
    }
  }
  if (paedMatch) {
    const price = parseFloat(paedMatch[1].replace(/,/g, ''));
    if (!Number.isNaN(price)) {
      results.push({ price, tier: 'paediatric' });
    }
  } else if (freeForPaed) {
    results.push({ price: 0, tier: 'paediatric' });
  }

  // Only return tier results if we found at least one tier indicator
  if (results.length > 0) {
    return results;
  }
  return [];
}

function extractNumbers(value: string): number[] {
  const matches = value.match(/\d{1,3}(?:,\d{3})+(?:\.\d+)?|\d+(?:\.\d+)?/g);
  if (!matches) {
    return [];
  }
  return matches.map((match) => parseFloat(match.replace(/,/g, ''))).filter((num) => !Number.isNaN(num));
}

function detectTier(value: string): string | undefined {
  if (value.includes('adult')) {
    return 'adult';
  }
  if (value.includes('paed') || value.includes('pediatric') || value.includes('child')) {
    return 'paediatric';
  }
  if (value.includes('executive')) {
    return 'executive';
  }
  if (value.includes('private')) {
    return 'private';
  }
  if (value.includes('general')) {
    return 'general';
  }
  return undefined;
}

function extractUnit(value: string): string | undefined {
  const lower = value.toLowerCase();
  if (lower.includes('per day') || lower.includes('daily')) {
    return 'per_day';
  }
  if (lower.includes('per hour') || lower.includes('hourly')) {
    return 'per_hour';
  }
  if (lower.includes('per week') || lower.includes('weekly')) {
    return 'per_week';
  }
  if (lower.includes('per month') || lower.includes('monthly')) {
    return 'per_month';
  }
  return undefined;
}

function inferFacilityName(
  rows: string[][],
  sourceFile?: string,
  minimumConfidence: number = 0
): string {
  // Try to find facility name in first few rows
  for (const row of rows) {
    const line = normalizeCell(row.join(' '));
    if (!line) {
      continue;
    }
    const lower = line.toLowerCase();
    
    // Look for strong indicators of facility names
    if (lower.includes('hospital') || lower.includes('clinic') || lower.includes('medical center')) {
      // Additional validation: ensure it's not just a generic header
      if (!lower.includes('price list') && !lower.includes('rate') && !lower.includes('charges')) {
        // Extra safeguard: check that the line has reasonable length (not just "hospital")
        if (line.length > 10 && line.length < 200) {
          const confidence = scoreFacilityNameCandidate(line);
          if (confidence >= minimumConfidence) {
            console.log(`Inferred facility name from content: ${line}`);
            return line;
          }
        }
      }
    }
    
    // Special case for specific well-known facilities
    if (lower.includes('lasuth')) {
      const confidence = scoreFacilityNameCandidate(line);
      if (confidence >= minimumConfidence) {
        console.log(`Inferred facility name from content: ${line}`);
        return line;
      }
    }
  }
  
  // Fallback to filename if available
  if (sourceFile) {
    const facilityFromFile = path.basename(sourceFile).replace(/\.[^.]+$/, '').replace(/_/g, ' ');
    console.log(`Inferred facility name from filename: ${facilityFromFile}`);
    return facilityFromFile;
  }
  
  // Last resort
  console.warn('Could not reliably infer facility name, using default');
  return 'Unknown Facility';
}

function scoreFacilityNameCandidate(value: string): number {
  const lower = value.toLowerCase();
  let score = 0;

  if (lower.includes('hospital') || lower.includes('clinic') || lower.includes('medical center')) {
    score += 0.6;
  }
  if (lower.includes('teaching') || lower.includes('university') || lower.includes('general')) {
    score += 0.2;
  }
  if (lower.includes('price list') || lower.includes('rate') || lower.includes('charges')) {
    score -= 0.5;
  }
  if (value.length > 10 && value.length < 200) {
    score += 0.1;
  }

  if (score < 0) {
    return 0;
  }
  if (score > 1) {
    return 1;
  }
  return score;
}

function inferEffectiveDate(rows: string[][], sourceFile?: string): Date {
  const candidates = rows.map((row) => row.join(' ')).join(' ');
  const matches = candidates.match(/(20\d{2})/g) || [];
  const fileMatches = sourceFile ? sourceFile.match(/(20\d{2})/g) || [] : [];
  const year = (matches[0] || fileMatches[0]) ?? '';
  if (year) {
    return new Date(`${year}-01-01T00:00:00Z`);
  }
  return new Date();
}

function buildProcedureCode(description: string, index: number): string {
  const slug = description.toUpperCase().replace(/[^A-Z0-9]+/g, '_').replace(/^_+|_+$/g, '');
  if (!slug) {
    return `ITEM_${index + 1}`;
  }
  if (slug.length <= 32) {
    return slug;
  }
  // For long descriptions, use truncated prefix + hash suffix for uniqueness
  const hash = simpleHash(slug);
  const prefix = slug.slice(0, 24);
  return `${prefix}_${hash}`;
}

function simpleHash(value: string): string {
  // FNV-1a 32-bit hash for deterministic, collision-resistant codes
  let hash = 0x811c9dc5;
  for (let i = 0; i < value.length; i++) {
    hash ^= value.charCodeAt(i);
    hash = Math.imul(hash, 0x01000193) >>> 0;
  }
  return hash.toString(16).padStart(8, '0');
}

function buildId(facilityName: string, description: string, variant: PriceVariant, index: number): string {
  const base = `${facilityName}-${description}-${variant.price}-${variant.tier || 'base'}-${index}`;
  return base.toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/^-+|-+$/g, '').slice(0, 80);
}

// Exported for testing
export const _testExports = {
  expandPriceVariants,
  detectTotalPrice,
  cleanPriceText,
  extractPriceQualifier,
  parseAreaHierarchy,
  buildProcedureCode,
  splitByTier,
};
