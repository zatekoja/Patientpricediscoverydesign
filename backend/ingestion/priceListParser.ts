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
  return extractDocxTableRows(xml);
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

function rowsToPriceData(rows: string[][], context: PriceListParseContext): PriceData[] {
  const normalizedRows = rows.map((row) => row.map(normalizeCell));
  const headerIndex = findHeaderRow(normalizedRows);
  const headerRow = headerIndex >= 0 ? normalizedRows[headerIndex] : [];
  const headerMap = buildHeaderMap(headerRow);
  
  // Check for explicit facility mapping first
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
    return [];
  }

  const facilityId = buildFacilityId(context.providerId, normalizedFacilityName);
  const effectiveDate =
    context.defaultEffectiveDate ||
    inferEffectiveDate(normalizedRows.slice(0, headerIndex > 0 ? headerIndex : 10), context.sourceFile);

  const mergedRows = mergeContinuationRows(normalizedRows, headerIndex, headerMap);
  let currentArea: string | undefined;
  let currentCategory: string | undefined;
  const priceData: PriceData[] = [];

  const startIndex = headerIndex >= 0 ? headerIndex + 1 : 0;
  for (let i = startIndex; i < mergedRows.length; i += 1) {
    const row = mergedRows[i];
    if (!row || row.every((cell) => !cell)) {
      continue;
    }

    if (headerIndex < 0 && row.some((cell) => isHeaderLike(cell))) {
      continue;
    }

    if (headerMap.areaIndex !== undefined) {
      const areaCell = row[headerMap.areaIndex] || '';
      if (areaCell) {
        currentArea = areaCell;
      }
    }

    const description = findDescription(row, headerMap);
    const priceText = findPriceText(row, headerMap);

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
      const id = buildId(normalizedFacilityName, description, variant, i);
      priceData.push({
        id,
        facilityName: normalizedFacilityName,
        facilityId: facilityId || undefined,
        procedureCode,
        procedureDescription: description,
        price: variant.price,
        currency: context.currency || 'NGN',
        effectiveDate,
        lastUpdated: new Date(),
        source: 'file_price_list',
        metadata: {
          sourceFile: context.sourceFile,
          area: currentArea,
          category: currentCategory,
          unit: variant.unit,
          priceTier: variant.tier,
          rawPriceText: variant.rawText,
          rowNumber: i + 1,
        },
      });
    }
  }

  return priceData;
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
  const segments = priceText.split(/\n|;/).map((segment) => segment.trim()).filter(Boolean);
  const variants: PriceVariant[] = [];

  const addVariant = (price: number, tier: string | undefined, rawText: string) => {
    variants.push({ price, tier, unit, rawText });
  };

  const parts = segments.length > 0 ? segments : [priceText];
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

    const splitParts = part.split(/,/).map((p) => p.trim()).filter(Boolean);
    for (const splitPart of splitParts) {
      const splitNumbers = extractNumbers(splitPart);
      if (splitNumbers.length === 0) {
        continue;
      }
      addVariant(splitNumbers[0], detectTier(splitPart.toLowerCase()), splitPart);
    }
  }

  return variants.length > 0 ? variants : [];
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
  return slug.slice(0, 32);
}

function buildId(facilityName: string, description: string, variant: PriceVariant, index: number): string {
  const base = `${facilityName}-${description}-${variant.price}-${variant.tier || 'base'}-${index}`;
  return base.toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/^-+|-+$/g, '').slice(0, 80);
}
