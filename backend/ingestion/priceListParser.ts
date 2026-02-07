import fs from 'fs';
import path from 'path';
import AdmZip from 'adm-zip';
import { XMLParser } from 'fast-xml-parser';
import { PriceData } from '../types/PriceData';

export interface PriceListParseContext {
  facilityName?: string;
  sourceFile?: string;
  currency?: string;
  defaultEffectiveDate?: Date;
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

export function parseCsvFile(filePath: string, context?: PriceListParseContext): PriceData[] {
  const content = fs.readFileSync(filePath, 'utf-8');
  const rows = parseCsvContent(content);
  return rowsToPriceData(rows, buildContext(filePath, context));
}

export function parseDocxFile(filePath: string, context?: PriceListParseContext): PriceData[] {
  const buffer = fs.readFileSync(filePath);
  const rows = parseDocxBuffer(buffer);
  return rowsToPriceData(rows, buildContext(filePath, context));
}

export function parseCsvContent(content: string): string[][] {
  const rows: string[][] = [];
  const lines = content.split(/\r?\n/);
  for (const line of lines) {
    const cells = parseCsvLine(line);
    rows.push(cells);
  }
  return rows;
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
  };
}

function rowsToPriceData(rows: string[][], context: PriceListParseContext): PriceData[] {
  const normalizedRows = rows.map((row) => row.map(normalizeCell));
  const headerIndex = findHeaderRow(normalizedRows);
  const headerRow = headerIndex >= 0 ? normalizedRows[headerIndex] : [];
  const headerMap = buildHeaderMap(headerRow);
  const facilityName =
    context.facilityName ||
    inferFacilityName(normalizedRows.slice(0, headerIndex > 0 ? headerIndex : 10), context.sourceFile);
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
      const id = buildId(facilityName, description, variant, i);
      priceData.push({
        id,
        facilityName,
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

function normalizeCell(value: string): string {
  return (value || '').replace(/\s+/g, ' ').trim();
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

function inferFacilityName(rows: string[][], sourceFile?: string): string {
  for (const row of rows) {
    const line = normalizeCell(row.join(' '));
    if (!line) {
      continue;
    }
    if (line.toLowerCase().includes('hospital') || line.toLowerCase().includes('lasuth')) {
      return line;
    }
  }
  if (sourceFile) {
    return path.basename(sourceFile).replace(/\.[^.]+$/, '').replace(/_/g, ' ');
  }
  return 'Unknown Facility';
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
