import crypto from 'crypto';
import fs from 'fs';
import path from 'path';
import { DocumentSummary, DocumentSummaryParser, parseDocumentSummaryResponse } from './documentSummary';
import { extractDocumentPreview } from './documentTextExtractor';
import { IDocumentStore } from '../interfaces/IDocumentStore';
import { PriceListParseContext } from './priceListParser';

export interface LLMDocumentParserConfig {
  enabled?: boolean;
  apiKey: string;
  apiEndpoint?: string;
  model: string;
  temperature?: number;
  maxRows?: number;
  maxChars?: number;
  maxBytes?: number;
}

export interface LLMDocumentSummaryClient {
  summarize: (prompt: string, config: LLMDocumentParserConfig) => Promise<{ content: string; tokensUsed?: number }>;
}

export interface DocumentSummaryCacheRecord {
  fileHash: string;
  sourceFile: string;
  model: string;
  extractedAt: string;
  summary: DocumentSummary;
}

export class LLMDocumentParser implements DocumentSummaryParser {
  private config: LLMDocumentParserConfig;
  private store?: IDocumentStore<DocumentSummaryCacheRecord>;
  private client: LLMDocumentSummaryClient;

  constructor(
    config: LLMDocumentParserConfig,
    store?: IDocumentStore<DocumentSummaryCacheRecord>,
    client?: LLMDocumentSummaryClient
  ) {
    this.config = config;
    this.store = store;
    this.client = client ?? new OpenAIChatClient();
  }

  async parse(filePath: string, context: PriceListParseContext): Promise<DocumentSummary | null> {
    if (!this.config.enabled) {
      return null;
    }
    if (!this.config.apiKey) {
      console.warn('LLM document parser is enabled but apiKey is missing.');
      return null;
    }

    const maxBytes = this.config.maxBytes ?? 5 * 1024 * 1024;
    const stat = await fs.promises.stat(filePath);
    if (stat.size > maxBytes) {
      console.warn(`Skipping LLM document parsing (file too large): ${filePath}`);
      return null;
    }

    const fileHash = await hashFile(filePath);
    const cacheKey = `doc_summary_${fileHash}`;
    if (this.store && (await this.store.exists(cacheKey))) {
      const cached = await this.store.get(cacheKey);
      return cached?.summary ?? null;
    }

    const preview = await extractDocumentPreview(filePath, {
      maxRows: this.config.maxRows,
      maxChars: this.config.maxChars,
      maxBytes: maxBytes,
    });
    if (!preview) {
      return null;
    }

    const prompt = buildDocumentSummaryPrompt(preview, context);
    const response = await this.client.summarize(prompt, this.config);
    const summary = parseDocumentSummaryResponse(response.content, preview.sourceFile);
    if (!summary) {
      return null;
    }

    summary.documentMetadata = {
      ...summary.documentMetadata,
      sourceFile: preview.sourceFile,
      extractedAt: summary.documentMetadata.extractedAt || new Date().toISOString(),
      model: this.config.model,
      tokensUsed: response.tokensUsed,
    };

    if (this.store) {
      const record: DocumentSummaryCacheRecord = {
        fileHash,
        sourceFile: preview.sourceFile,
        model: this.config.model,
        extractedAt: summary.documentMetadata.extractedAt,
        summary,
      };
      await this.store.put(cacheKey, record, {
        fileHash,
        sourceFile: preview.sourceFile,
        model: this.config.model,
      });
    }

    return summary;
  }
}

class OpenAIChatClient implements LLMDocumentSummaryClient {
  async summarize(prompt: string, config: LLMDocumentParserConfig): Promise<{ content: string; tokensUsed?: number }> {
    const endpoint = config.apiEndpoint || 'https://api.openai.com/v1/chat/completions';
    const response = await fetch(endpoint, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${config.apiKey}`,
      },
      body: JSON.stringify({
        model: config.model,
        temperature: config.temperature ?? 0.2,
        response_format: buildResponseFormat(config),
        messages: [
          {
            role: 'system',
            content:
              'You are a data extraction assistant. Return ONLY valid JSON matching the provided schema.',
          },
          { role: 'user', content: prompt },
        ],
      }),
    });

    if (!response.ok) {
      const text = await response.text();
      throw new Error(`LLM API error: ${response.status} ${response.statusText}: ${text}`);
    }

    const data = (await response.json()) as {
      choices?: Array<{ message?: { content?: string } }>;
      usage?: { total_tokens?: number };
    };
    const content = data?.choices?.[0]?.message?.content;
    if (typeof content !== 'string') {
      throw new Error('LLM API returned empty response content');
    }
    const tokensUsed = data?.usage?.total_tokens;
    return { content, tokensUsed };
  }
}

function buildDocumentSummaryPrompt(preview: { preview: string; sourceFile: string }, context: PriceListParseContext): string {
  const currency = context.currency || 'NGN';
  return `Extract a structured summary of this healthcare price list document in JSON.

Return JSON with this schema:
{
  "facilityName": "string",
  "currency": "string",
  "effectiveDate": "YYYY-MM-DD or ISO date string (optional)",
  "items": [
    {
      "description": "string",
      "price": number,
      "unit": "string (optional)",
      "tier": "string (optional)",
      "category": "string (optional)",
      "notes": "string (optional)",
      "rawRow": "string (optional)"
    }
  ],
  "documentMetadata": {
    "sourceFile": "${preview.sourceFile}",
    "extractedAt": "${new Date().toISOString()}",
    "model": "${context.providerId || 'unknown'}"
  }
}

Guidelines:
- Use currency "${currency}" unless specified otherwise.
- If a price is missing or non-numeric, omit that item.
- Keep descriptions concise and human-readable.
- Facility name must be the real facility (not the provider, not "price list", not "services only").
- Avoid generic names like "Healthcare Facility", "Medical Facility", "Hospital", or "Clinic" without a location.
- If the facility name is not explicit, infer it from the document title, header rows, or filename.
- Preserve location qualifiers (e.g., "General Hospital Badagry").
- If the document mentions LASUTH, use "Lagos State University Teaching Hospital (LASUTH)".

Document Preview:
${preview.preview}`;
}

function buildResponseFormat(config: LLMDocumentParserConfig): Record<string, any> {
  const model = config.model || '';
  const supportsJsonSchema =
    model.includes('gpt-4o-mini') || model.includes('gpt-4o-2024-08-06') || model.includes('gpt-4o');

  if (!supportsJsonSchema) {
    return { type: 'json_object' };
  }

  return {
    type: 'json_schema',
    json_schema: {
      name: 'facility_price_list_summary',
      strict: true,
      schema: {
        type: 'object',
        additionalProperties: false,
        required: ['facilityName', 'currency', 'effectiveDate', 'items', 'documentMetadata'],
        properties: {
          facilityName: { type: 'string' },
          currency: { type: ['string', 'null'] },
          effectiveDate: { type: ['string', 'null'] },
          items: {
            type: 'array',
            items: {
              type: 'object',
              additionalProperties: false,
              required: ['description', 'price', 'unit', 'tier', 'category', 'notes', 'rawRow'],
              properties: {
                description: { type: 'string' },
                price: { type: 'number' },
                unit: { type: ['string', 'null'] },
                tier: { type: ['string', 'null'] },
                category: { type: ['string', 'null'] },
                notes: { type: ['string', 'null'] },
                rawRow: { type: ['string', 'null'] },
              },
            },
          },
          documentMetadata: {
            type: 'object',
            additionalProperties: false,
            required: ['sourceFile', 'extractedAt', 'model', 'tokensUsed', 'confidence', 'warnings'],
            properties: {
              sourceFile: { type: 'string' },
              extractedAt: { type: 'string' },
              model: { type: 'string' },
              tokensUsed: { type: ['number', 'null'] },
              confidence: { type: ['number', 'null'] },
              warnings: {
                type: ['array', 'null'],
                items: { type: 'string' },
              },
            },
          },
        },
      },
    },
  };
}

async function hashFile(filePath: string): Promise<string> {
  const data = await fs.promises.readFile(filePath);
  return crypto.createHash('sha256').update(data).digest('hex');
}
