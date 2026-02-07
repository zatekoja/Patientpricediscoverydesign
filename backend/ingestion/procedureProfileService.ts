import { trace, SpanStatusCode } from '@opentelemetry/api';
import { IDocumentStore } from '../interfaces/IDocumentStore';
import { ProcedureProfile } from '../types/ProcedureProfile';
import { PriceData } from '../types/PriceData';
import { normalizeIdentifier } from './facilityIds';
import { recordProcedureProfileEnrichment, recordProcedureProfileLLM } from '../observability/metrics';

export interface ProcedureLLMConfig {
  apiEndpoint?: string;
  apiKey?: string;
  model?: string;
  systemPrompt?: string;
  temperature?: number;
}

export interface ProcedureEnrichmentSummary {
  created: number;
  skipped: number;
  failed: number;
}

export class ProcedureProfileService {
  private tracer = trace.getTracer('patient-price-discovery-provider');

  constructor(
    private store: IDocumentStore<ProcedureProfile>,
    private llmConfig?: ProcedureLLMConfig,
  ) {}

  async enrichRecords(records: PriceData[], providerId?: string): Promise<PriceData[]> {
    if (!records.length) {
      return records;
    }
    const summary = await this.ensureProfiles(records, providerId);
    recordProcedureProfileEnrichment({
      provider: providerId || 'unknown',
      created: summary.created,
      skipped: summary.skipped,
      failed: summary.failed,
    });

    const enriched: PriceData[] = [];
    for (const record of records) {
      const key = buildProcedureKey(record);
      if (!key) {
        enriched.push(record);
        continue;
      }
      const profile = await this.store.get(key);
      if (!profile) {
        enriched.push(record);
        continue;
      }
      enriched.push({
        ...record,
        procedureCategory: profile.category || record.procedureCategory,
        procedureDetails: profile.description || record.procedureDetails,
        estimatedDurationMinutes: profile.estimatedDurationMinutes ?? record.estimatedDurationMinutes,
      });
    }
    return enriched;
  }

  async ensureProfiles(records: PriceData[], providerId?: string): Promise<ProcedureEnrichmentSummary> {
    const summary: ProcedureEnrichmentSummary = { created: 0, skipped: 0, failed: 0 };
    const seen = new Map<string, PriceData>();
    for (const record of records) {
      const key = buildProcedureKey(record);
      if (!key || seen.has(key)) {
        continue;
      }
      seen.set(key, record);
    }

    for (const [key, record] of seen.entries()) {
      try {
        if (await this.store.exists(key)) {
          summary.skipped += 1;
          continue;
        }
        const profile = await this.generateProfile(record, providerId);
        if (!profile) {
          summary.failed += 1;
          continue;
        }
        await this.store.put(key, profile, {
          source: profile.source,
          generatedAt: new Date().toISOString(),
        });
        summary.created += 1;
      } catch (error) {
        summary.failed += 1;
        console.error(`Failed to enrich procedure ${record.procedureDescription}:`, error);
      }
    }

    return summary;
  }

  private async generateProfile(record: PriceData, providerId?: string): Promise<ProcedureProfile | null> {
    if (!isLLMConfigured(this.llmConfig)) {
      return null;
    }
    const prompt = buildProcedurePrompt(record);
    const startTime = Date.now();
    return this.tracer.startActiveSpan(
      'provider.procedure_enrichment',
      { attributes: { procedure: record.procedureDescription } },
      async (span) => {
        try {
          const responseText = await callLLMAPI(prompt, this.llmConfig!);
          const parsed = parseLLMProcedure(responseText);
          recordProcedureProfileLLM({
            provider: this.llmConfig?.model || 'unknown',
            success: true,
            durationMs: Date.now() - startTime,
            tags: parsed.tags ? parsed.tags.length : 0,
          });
          span.setStatus({ code: SpanStatusCode.OK });
          return {
            id: buildProcedureKey(record) || record.procedureDescription,
            code: parsed.code || record.procedureCode,
            name: parsed.name || record.procedureDescription,
            category: parsed.category,
            description: parsed.description,
            estimatedDurationMinutes: parsed.estimatedDurationMinutes,
            tags: parsed.tags,
            lastUpdated: new Date(),
            source: providerId || record.source || 'provider',
            metadata: {
              llm: {
                model: this.llmConfig?.model,
                generatedAt: new Date(),
              },
            },
          };
        } catch (error) {
          recordProcedureProfileLLM({
            provider: this.llmConfig?.model || 'unknown',
            success: false,
            durationMs: Date.now() - startTime,
          });
          span.recordException(error as Error);
          span.setStatus({
            code: SpanStatusCode.ERROR,
            message: error instanceof Error ? error.message : 'Unknown error',
          });
          return null;
        } finally {
          span.end();
        }
      }
    );
  }
}

function buildProcedureKey(record: PriceData): string {
  const code = normalizeIdentifier(record.procedureCode);
  if (code) {
    return `proc_${code}`;
  }
  const name = normalizeIdentifier(record.procedureDescription);
  if (!name) {
    return '';
  }
  return `proc_${name}`;
}

function isLLMConfigured(config?: ProcedureLLMConfig): boolean {
  return Boolean(config?.apiEndpoint && config?.apiKey && config?.model);
}

function buildProcedurePrompt(record: PriceData): string {
  return `Generate structured metadata for this healthcare service.

Procedure name: ${record.procedureDescription}
Procedure code: ${record.procedureCode}
Facility: ${record.facilityName}
Tags: ${record.tags?.join(', ') || 'none'}

Return JSON only:
{
  "name": string,
  "code": string,
  "category": string,
  "description": string,
  "estimatedDurationMinutes": number,
  "tags": string[]
}
Example description style: "Magnetic resonance imaging uses a strong magnet and radio waves to produce detailed images of the brain without radiation exposure."`;
}

async function callLLMAPI(prompt: string, config: ProcedureLLMConfig): Promise<string> {
  const payload = {
    model: config.model,
    messages: [
      { role: 'system', content: config.systemPrompt || defaultSystemPrompt() },
      { role: 'user', content: prompt },
    ],
    temperature: config.temperature ?? 0.2,
  };

  const response = await fetch(config.apiEndpoint!, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${config.apiKey}`,
    },
    body: JSON.stringify(payload),
  });

  if (!response.ok) {
    const text = await response.text();
    throw new Error(`LLM API error (${response.status}): ${text}`);
  }

  const data: any = await response.json();
  const content =
    data?.choices?.[0]?.message?.content ||
    data?.choices?.[0]?.text ||
    '';
  if (!content) {
    throw new Error('LLM API returned empty response');
  }
  return content;
}

function parseLLMProcedure(content: string): Partial<ProcedureProfile> {
  const json = extractJSON(content);
  if (!json) {
    throw new Error('Unable to parse LLM response');
  }
  const parsed = JSON.parse(json);
  return {
    name: parsed.name,
    code: parsed.code,
    category: parsed.category,
    description: parsed.description,
    estimatedDurationMinutes: parsed.estimatedDurationMinutes,
    tags: Array.isArray(parsed.tags) ? parsed.tags.map(normalizeIdentifier).filter(Boolean) : undefined,
  };
}

function extractJSON(content: string): string | null {
  const start = content.indexOf('{');
  const end = content.lastIndexOf('}');
  if (start === -1 || end === -1 || end <= start) {
    return null;
  }
  return content.slice(start, end + 1);
}

function defaultSystemPrompt(): string {
  return `You are a healthcare services classifier.
Return concise, structured metadata for each procedure to help search and scheduling.`;
}
