import fs from 'fs';
import path from 'path';
import AdmZip from 'adm-zip';
import { XMLParser } from 'fast-xml-parser';
import { parse as csvParseStream } from 'csv-parse';
import * as xlsx from 'xlsx';

export interface DocumentPreview {
  sourceFile: string;
  fileType: 'csv' | 'docx' | 'xlsx' | 'unknown';
  rowCount: number;
  preview: string;
  truncated: boolean;
}

export interface DocumentPreviewOptions {
  maxRows?: number;
  maxChars?: number;
  maxBytes?: number;
}

export async function extractDocumentPreview(
  filePath: string,
  options?: DocumentPreviewOptions
): Promise<DocumentPreview | null> {
  const ext = path.extname(filePath).toLowerCase();
  const fileType = ext === '.csv' ? 'csv' : ext === '.docx' ? 'docx' : ext === '.xlsx' ? 'xlsx' : 'unknown';
  if (fileType === 'unknown') {
    return null;
  }

  const maxBytes = options?.maxBytes ?? 5 * 1024 * 1024;
  const stat = await fs.promises.stat(filePath);
  if (stat.size > maxBytes) {
    return null;
  }

  const maxRows = options?.maxRows ?? 5000;
  const maxChars = options?.maxChars ?? 50_000;
  let rows: string[][] = [];

  if (fileType === 'csv') {
    rows = await extractCsvRows(filePath, maxRows);
  } else if (fileType === 'docx') {
    const buffer = await fs.promises.readFile(filePath);
    rows = extractDocxRows(buffer, maxRows);
  } else if (fileType === 'xlsx') {
    rows = extractXlsxRows(filePath, maxRows);
  }

  const { preview, truncated } = formatPreview(rows, maxChars);
  return {
    sourceFile: path.basename(filePath),
    fileType,
    rowCount: rows.length,
    preview,
    truncated,
  };
}

async function extractCsvRows(filePath: string, maxRows: number): Promise<string[][]> {
  return new Promise((resolve, reject) => {
    const records: string[][] = [];
    const parser = csvParseStream({
      relax_quotes: true,
      skip_empty_lines: false,
      trim: false,
      relax_column_count: true,
    });
    const stream = fs.createReadStream(filePath);

    stream.on('error', reject);
    parser.on('error', reject);
    parser.on('data', (row: string[]) => {
      if (records.length < maxRows) {
        records.push(row.map((cell) => (cell ?? '').toString()));
      }
    });
    parser.on('end', () => resolve(records));
    stream.pipe(parser);
  });
}

function extractDocxRows(buffer: Buffer, maxRows: number): string[][] {
  const zip = new AdmZip(buffer);
  const entry = zip.getEntry('word/document.xml');
  if (!entry) {
    return [];
  }
  const xml = entry.getData().toString('utf-8');
  const parser = new XMLParser({ ignoreAttributes: false });
  const doc = parser.parse(xml);
  const tables = collectNodes(doc, 'w:tbl');
  const rows: string[][] = [];

  for (const table of tables) {
    const tableRows = collectNodes(table, 'w:tr');
    for (const row of tableRows) {
      if (rows.length >= maxRows) {
        return rows;
      }
      const cells = collectNodes(row, 'w:tc');
      const rowValues: string[] = [];
      for (const cell of cells) {
        const texts = collectNodes(cell, 'w:t');
        const text = texts.map((t) => (typeof t === 'string' ? t : t['#text'] || '')).join('');
        rowValues.push(text);
      }
      if (rowValues.some((value) => value)) {
        rows.push(rowValues);
      }
    }
  }
  return rows;
}

function extractXlsxRows(filePath: string, maxRows: number): string[][] {
  const workbook = xlsx.readFile(filePath);
  const sheetName = workbook.SheetNames[0];
  if (!sheetName) {
    return [];
  }
  const sheet = workbook.Sheets[sheetName];
  const rows = xlsx.utils.sheet_to_json(sheet, { header: 1, raw: true }) as any[][];
  const limited = rows.slice(0, maxRows);
  return limited.map((row) => row.map((cell) => (cell ?? '').toString()));
}

function formatPreview(rows: string[][], maxChars: number): { preview: string; truncated: boolean } {
  const lines = rows.map((row) => row.map((cell) => (cell ?? '').toString().trim()).join(' | '));
  let preview = lines.join('\n');
  if (preview.length > maxChars) {
    preview = `${preview.slice(0, maxChars)}\n[TRUNCATED]`;
    return { preview, truncated: true };
  }
  return { preview, truncated: false };
}

function collectNodes(node: any, key: string): any[] {
  const results: any[] = [];
  if (!node || typeof node !== 'object') {
    return results;
  }
  for (const [entryKey, entryValue] of Object.entries(node)) {
    if (entryKey === key) {
      if (Array.isArray(entryValue)) {
        results.push(...entryValue);
      } else {
        results.push(entryValue);
      }
    } else if (typeof entryValue === 'object') {
      results.push(...collectNodes(entryValue, key));
    }
  }
  return results;
}
