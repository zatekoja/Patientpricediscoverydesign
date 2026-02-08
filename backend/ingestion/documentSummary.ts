import path from 'path';
import { PriceData } from '../types/PriceData';
import { buildFacilityId, normalizeFacilityName } from './facilityIds';
import { PriceListParseContext } from './priceListParser';

export interface DocumentSummaryItem {
  description: string;
  price: number;
  currency?: string;
  unit?: string;
  tier?: string;
  category?: string;
  notes?: string;
  rawRow?: string;
}

export interface DocumentSummaryMetadata {
  sourceFile: string;
  extractedAt: string;
  model: string;
  tokensUsed?: number;
  confidence?: number;
  warnings?: string[];
}

export interface DocumentSummary {
  facilityName: string;
  currency?: string;
  effectiveDate?: string;
  items: DocumentSummaryItem[];
  documentMetadata: DocumentSummaryMetadata;
}

export interface DocumentSummaryParser {
  parse: (filePath: string, context: PriceListParseContext) => Promise<DocumentSummary | null>;
}

export function parseDocumentSummaryResponse(raw: string, sourceFile: string): DocumentSummary | null {
  let parsed: any;
  try {
    parsed = JSON.parse(raw);
  } catch {
    return null;
  }

  const facilityName = parsed.facilityName || parsed.facility_name;
  const items = Array.isArray(parsed.items) ? parsed.items : [];
  if (!facilityName || items.length === 0) {
    return null;
  }

  const normalizedItems: DocumentSummaryItem[] = items
    .map((item: any): DocumentSummaryItem => {
      const description = typeof item.description === 'string' ? item.description.trim() : '';
      const category = typeof item.category === 'string' ? item.category.trim() : undefined;
      const fallbackCategory = category || inferCategoryFromDescription(description);
      return {
        description,
        price: normalizePrice(item.price),
        currency: typeof item.currency === 'string' ? item.currency.trim() : undefined,
        unit: typeof item.unit === 'string' ? item.unit.trim() : undefined,
        tier: typeof item.tier === 'string' ? item.tier.trim() : undefined,
        category: fallbackCategory || undefined,
        notes: typeof item.notes === 'string' ? item.notes.trim() : undefined,
        rawRow: typeof item.rawRow === 'string' ? item.rawRow : undefined,
      };
    })
    .filter((item: DocumentSummaryItem) => item.description && Number.isFinite(item.price));

  if (normalizedItems.length === 0) {
    return null;
  }

  const metadata: DocumentSummaryMetadata = {
    sourceFile: parsed.documentMetadata?.sourceFile || sourceFile || 'unknown',
    extractedAt: parsed.documentMetadata?.extractedAt || new Date().toISOString(),
    model: parsed.documentMetadata?.model || 'unknown',
    tokensUsed: parsed.documentMetadata?.tokensUsed,
    confidence: parsed.documentMetadata?.confidence,
    warnings: parsed.documentMetadata?.warnings,
  };

  const normalizedFacilityName = normalizeFacilityName(facilityName.toString().trim(), sourceFile);
  if (!normalizedFacilityName) {
    return null;
  }

  return {
    facilityName: normalizedFacilityName,
    currency: parsed.currency ? parsed.currency.toString().trim() : undefined,
    effectiveDate: parsed.effectiveDate ? parsed.effectiveDate.toString().trim() : undefined,
    items: normalizedItems,
    documentMetadata: metadata,
  };
}

export function summaryToPriceData(summary: DocumentSummary, context: PriceListParseContext): PriceData[] {
  const mappedFacilityName =
    (context.sourceFile && context.explicitFacilityMapping?.[context.sourceFile]) || undefined;
  const facilityName = normalizeFacilityName(
    mappedFacilityName || summary.facilityName || context.facilityName || 'Unknown Facility',
    context.sourceFile,
  );
  if (!facilityName) {
    return [];
  }
  const facilityId = buildFacilityId(context.providerId, facilityName);
  const effectiveDate =
    parseEffectiveDate(summary.effectiveDate) ||
    context.defaultEffectiveDate ||
    new Date();

  const defaultCurrency = summary.currency || context.currency || 'NGN';
  const sourceFile = summary.documentMetadata?.sourceFile || context.sourceFile || path.basename('unknown');

  return summary.items.map((item, index) => {
    const description = item.description.trim();
    const procedureCode = buildProcedureCode(description, index);
    const price = Number.isFinite(item.price) ? item.price : 0;
    const currency = item.currency || defaultCurrency;
    const id = buildSummaryId(facilityName, description, price, item.tier || 'base', index);

    return {
      id,
      facilityName,
      facilityId: facilityId || undefined,
      procedureCode,
      procedureDescription: description,
      procedureCategory: item.category,
      procedureDetails: item.notes,
      price,
      currency,
      effectiveDate,
      lastUpdated: new Date(),
      source: context.providerId || 'file_price_list',
      metadata: {
        sourceFile,
        area: undefined,
        category: item.category,
        unit: item.unit,
        priceTier: item.tier,
        rawPriceText: undefined,
        rawRow: item.rawRow,
        documentMetadata: summary.documentMetadata,
      },
    };
  });
}

function normalizePrice(value: unknown): number {
  if (typeof value === 'number' && Number.isFinite(value)) {
    return value;
  }
  if (typeof value === 'string') {
    const cleaned = value.replace(/[,₦$]/g, '').trim();
    const parsed = Number.parseFloat(cleaned);
    if (Number.isFinite(parsed)) {
      return parsed;
    }
  }
  return Number.NaN;
}

function inferCategoryFromDescription(description: string): string {
  const lower = (description || '').toLowerCase();
  if (!lower) {
    return '';
  }

  const rules: Array<{ category: string; keywords: string[] }> = [
    { category: 'Imaging', keywords: ['mri', 'ct', 'scan', 'x-ray', 'xray', 'ultrasound', 'radiology'] },
    { category: 'Laboratory', keywords: ['lab', 'test', 'blood', 'urine', 'specimen', 'hematology', 'haematology'] },
    { category: 'Pharmacy', keywords: ['drug', 'medication', 'pharmacy', 'injection', 'infusion'] },
    { category: 'Dental', keywords: ['dental', 'tooth', 'teeth', 'oral'] },
    { category: 'Maternity', keywords: ['antenatal', 'pregnancy', 'delivery', 'maternity', 'obstetric'] },
    { category: 'Surgery', keywords: ['surgery', 'surgical', 'operation', 'theatre'] },
    { category: 'Consultation', keywords: ['consultation', 'clinic', 'appointment', 'review'] },
    { category: 'Emergency', keywords: ['emergency', 'casualty', 'trauma'] },
    { category: 'Administrative', keywords: ['registration', 'card', 'folder', 'report', 'certificate', 'leave'] },
  ];

  for (const rule of rules) {
    if (rule.keywords.some((keyword) => lower.includes(keyword))) {
      return rule.category;
    }
  }

  return 'General';
}

function parseEffectiveDate(value?: string): Date | null {
  if (!value) {
    return null;
  }
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return null;
  }
  return date;
}

function buildProcedureCode(description: string, index: number): string {
  const slug = description.toUpperCase().replace(/[^A-Z0-9]+/g, '_').replace(/^_+|_+$/g, '');
  if (!slug) {
    return `ITEM_${index + 1}`;
  }
  return slug.slice(0, 32);
}

function buildSummaryId(
  facilityName: string,
  description: string,
  price: number,
  tier: string,
  index: number
): string {
  const base = `${facilityName}-${description}-${price}-${tier}-${index}`;
  return base.toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/^-+|-+$/g, '').slice(0, 80);
}
