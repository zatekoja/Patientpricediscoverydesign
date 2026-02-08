import { trace, SpanStatusCode } from '@opentelemetry/api';
import { IExternalDataProvider, DataProviderOptions } from '../interfaces/IExternalDataProvider';
import { IDocumentStore } from '../interfaces/IDocumentStore';
import { PriceData } from '../types/PriceData';
import { FacilityProfile } from '../types/FacilityProfile';
import { hydrateTags } from './tagHydration';
import { buildFacilityId, normalizeIdentifier } from './facilityIds';
import {
  recordCapacityUpdate,
  recordFacilityProfileEnrichment,
  recordFacilityProfileLLM,
} from '../observability/metrics';

export interface FacilityLLMConfig {
  apiEndpoint?: string;
  apiKey?: string;
  model?: string;
  systemPrompt?: string;
  temperature?: number;
  maxTags?: number;
}

export interface FacilityEnrichmentSummary {
  created: number;
  skipped: number;
  failed: number;
}

export interface FacilityStatusUpdate {
  capacityStatus?: string | null;
  avgWaitMinutes?: number | null;
  urgentCareAvailable?: boolean | null;
  wardUpdate?: {
    wardId: string;
    status: string;
    count: number;
    trend: string;
    estimatedWaitMinutes?: number;
  };
  wardName?: string; // Optional: if provided, update ward-specific capacity instead of facility-wide
}

export class FacilityProfileService {
  private tracer = trace.getTracer('patient-price-discovery-provider');

  constructor(
    private store: IDocumentStore<FacilityProfile>,
    private llmConfig?: FacilityLLMConfig,
    private redisClient?: any
  ) {}

  getStore(): IDocumentStore<FacilityProfile> {
    return this.store;
  }

  async getProfile(id: string): Promise<FacilityProfile | null> {
    return this.store.get(id);
  }

  async listProfiles(limit: number, offset: number): Promise<FacilityProfile[]> {
    return this.store.query({}, { limit, offset, sortBy: 'lastUpdated', sortOrder: 'desc' });
  }

  async updateStatus(
    id: string,
    update: FacilityStatusUpdate,
    options?: { source?: string }
  ): Promise<FacilityProfile> {
    const profile = await this.store.get(id);
    if (!profile) {
      recordCapacityUpdate({ source: options?.source || 'unknown', success: false });
      throw new Error('Facility not found');
    }
    const now = new Date();
    let eventPayload: any = null;

    // Transaction-based ward update (wardStatuses)
    if (update.wardUpdate) {
      if (!profile.wardStatuses) {
        profile.wardStatuses = {};
      }
      profile.wardStatuses[update.wardUpdate.wardId] = {
        status: update.wardUpdate.status,
        count: update.wardUpdate.count,
        trend: update.wardUpdate.trend,
        estimatedWaitMinutes: (update as any).wardUpdate.estimatedWaitMinutes,
        lastUpdated: now
      };
      const wardInfos = Object.values(profile.wardStatuses);
      const statuses = wardInfos.map(s => s.status);
      if (statuses.includes('full')) profile.capacityStatus = 'full';
      else if (statuses.includes('busy')) profile.capacityStatus = 'busy';
      else profile.capacityStatus = 'available';
      const waits = wardInfos.map(s => (s as any).estimatedWaitMinutes).filter(w => typeof w === 'number');
      if (waits.length > 0) profile.avgWaitMinutes = Math.max(...waits);
      eventPayload = {
        id: `ev_${Date.now()}`,
        facility_id: id,
        event_type: 'ward_capacity_update',
        timestamp: now.toISOString(),
        location: profile.location || { latitude: 0, longitude: 0 },
        changed_fields: {
          ward_id: update.wardUpdate.wardId,
          status: update.wardUpdate.status,
          count: update.wardUpdate.count,
          trend: update.wardUpdate.trend,
          estimated_wait_minutes: (update as any).wardUpdate.estimatedWaitMinutes,
          capacity_status: profile.capacityStatus,
          avg_wait_minutes: profile.avgWaitMinutes
        }
      };
    } else if (update.wardName) {
      // Form-based ward capacity update (wards array)
      if (!profile.wards) profile.wards = [];
      const wardName = update.wardName!;
      let ward = profile.wards.find(w => w.wardName.toLowerCase() === wardName.toLowerCase());
      if (!ward) {
        ward = { wardName: wardName, lastUpdated: now };
        profile.wards.push(ward);
      }
      if (update.capacityStatus !== undefined) ward.capacityStatus = update.capacityStatus ?? undefined;
      if (update.avgWaitMinutes !== undefined) ward.avgWaitMinutes = update.avgWaitMinutes ?? undefined;
      if (update.urgentCareAvailable !== undefined) ward.urgentCareAvailable = update.urgentCareAvailable ?? undefined;
      ward.lastUpdated = now;
      eventPayload = {
        id: `ev_${Date.now()}`,
        facility_id: id,
        event_type: 'capacity_update',
        timestamp: now.toISOString(),
        location: profile.location || { latitude: 0, longitude: 0 },
        changed_fields: { ward_name: wardName, capacity_status: ward.capacityStatus, avg_wait_minutes: ward.avgWaitMinutes }
      };
    } else {
      // Facility-wide capacity (legacy)
      if (update.capacityStatus !== undefined) profile.capacityStatus = update.capacityStatus ?? undefined;
      if (update.avgWaitMinutes !== undefined) profile.avgWaitMinutes = update.avgWaitMinutes ?? undefined;
      if (update.urgentCareAvailable !== undefined) profile.urgentCareAvailable = update.urgentCareAvailable ?? undefined;
      eventPayload = {
        id: `ev_${Date.now()}`,
        facility_id: id,
        event_type: 'capacity_update',
        timestamp: now.toISOString(),
        location: profile.location || { latitude: 0, longitude: 0 },
        changed_fields: { capacity_status: profile.capacityStatus, avg_wait_minutes: profile.avgWaitMinutes }
      };
    }

    profile.lastUpdated = now;
    await this.store.put(id, profile, {
      source: profile.source,
      updatedAt: now.toISOString(),
    });

    // Publish to Redis for real-time SSE
    if (this.redisClient && eventPayload) {
      try {
        const channel = `facility:${id}`;
        const updatesChannel = 'facility:updates'; 
        await this.redisClient.publish(channel, JSON.stringify(eventPayload));
        await this.redisClient.publish(updatesChannel, JSON.stringify(eventPayload));
      } catch (e) {
        console.error('Failed to publish real-time update to Redis', e);
      }
    }

    recordCapacityUpdate({ source: options?.source || 'unknown', success: true });
    return profile;
  }

  async ensureProfiles(
    data: PriceData[],
    options?: { providerId?: string; force?: boolean }
  ): Promise<FacilityEnrichmentSummary> {
    const summary: FacilityEnrichmentSummary = { created: 0, skipped: 0, failed: 0 };
    if (!data || data.length === 0) {
      return summary;
    }

    const facilityMap = new Map<string, { name: string; records: PriceData[] }>();
    for (const item of data) {
      const facilityName = (item.facilityName || '').trim();
      const facilityId =
        item.facilityId ||
        buildFacilityId(options?.providerId || item.source, facilityName);
      if (!facilityId) {
        continue;
      }
      const existing = facilityMap.get(facilityId);
      if (existing) {
        existing.records.push(item);
      } else {
        facilityMap.set(facilityId, { name: facilityName || facilityId, records: [item] });
      }
    }

    for (const [facilityId, entry] of facilityMap.entries()) {
      const { name, records } = entry;
      try {
        if (!options?.force && (await this.store.exists(facilityId))) {
          summary.skipped += 1;
          continue;
        }

        const baseTags = collectFacilityTags(records);
        const curatedSources = collectCuratedSources(records);
        const sampleProcedures = collectSampleProcedures(records);
        const llmProfile = await this.generateLLMProfile(name, baseTags, sampleProcedures);
        const mergedTags = mergeTags(baseTags, llmProfile?.tags || []);
        const facilityType = llmProfile?.facilityType || inferFacilityType(name, mergedTags);

        const profile: FacilityProfile = {
          id: facilityId,
          name: llmProfile?.name || name || facilityId,
          facilityType,
          description: llmProfile?.description,
          tags: mergedTags,
          address: llmProfile?.address,
          location: llmProfile?.location,
          phoneNumber: llmProfile?.phoneNumber,
          email: llmProfile?.email,
          website: llmProfile?.website,
          lastUpdated: new Date(),
          source: options?.providerId || records[0]?.source || 'provider',
          metadata: {
            curatedSources,
            matchedRules: collectMatchedRules(records),
            sampleProcedures,
            llm: llmProfile?.metadata?.llm,
          },
        };

        await this.store.put(facilityId, profile, {
          source: profile.source,
          generatedAt: new Date().toISOString(),
        });
        summary.created += 1;
      } catch (error) {
        summary.failed += 1;
        console.error(`Failed to enrich facility ${facilityId}:`, error);
      }
    }

    recordFacilityProfileEnrichment({
      provider: options?.providerId || 'unknown',
      created: summary.created,
      skipped: summary.skipped,
      failed: summary.failed,
    });
    return summary;
  }

  async ensureProfilesFromProvider(
    provider: IExternalDataProvider<PriceData>,
    options?: { providerId?: string; pageSize?: number }
  ): Promise<FacilityEnrichmentSummary> {
    const summary: FacilityEnrichmentSummary = { created: 0, skipped: 0, failed: 0 };
    const limit = options?.pageSize && options.pageSize > 0 ? options.pageSize : 500;
    let offset = 0;
    let total = 0;

    for (;;) {
      const response = await provider.getCurrentData({ limit, offset } as DataProviderOptions);
      if (!response.data || response.data.length === 0) {
        break;
      }
      const pageSummary = await this.ensureProfiles(response.data, {
        providerId: options?.providerId || provider.getName(),
      });
      summary.created += pageSummary.created;
      summary.skipped += pageSummary.skipped;
      summary.failed += pageSummary.failed;
      offset += response.data.length;
      total = response.metadata?.total || total;
      if (response.data.length < limit) {
        break;
      }
      if (total > 0 && offset >= total) {
        break;
      }
      if (response.metadata?.hasMore === false) {
        break;
      }
    }

    return summary;
  }

  private async generateLLMProfile(
    facilityName: string,
    existingTags: string[],
    sampleProcedures: string[]
  ): Promise<Partial<FacilityProfile> | null> {
    if (!isLLMConfigured(this.llmConfig)) {
      return null;
    }

    const prompt = buildFacilityPrompt(facilityName, existingTags, sampleProcedures);
    const startTime = Date.now();
    return this.tracer.startActiveSpan(
      'provider.facility_enrichment',
      { attributes: { facility: facilityName, tags: existingTags.length } },
      async (span) => {
        try {
          const responseText = await callLLMAPI(prompt, this.llmConfig!);
          const parsed = parseLLMProfile(responseText);
          recordFacilityProfileLLM({
            provider: this.llmConfig?.model || 'unknown',
            success: true,
            durationMs: Date.now() - startTime,
            tags: parsed.tags ? parsed.tags.length : 0,
          });
          span.setStatus({ code: SpanStatusCode.OK });
          return {
            ...parsed,
            metadata: {
              llm: {
                model: this.llmConfig?.model,
                generatedAt: new Date(),
              },
            },
          };
        } catch (error) {
          recordFacilityProfileLLM({
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

function isLLMConfigured(config?: FacilityLLMConfig): boolean {
  return Boolean(config?.apiEndpoint && config?.apiKey && config?.model);
}

function buildFacilityPrompt(
  facilityName: string,
  tags: string[],
  sampleProcedures: string[]
): string {
  const samples = sampleProcedures.slice(0, 12).join('; ');
  return `Facility name: ${facilityName}
Known tags: ${tags.join(', ') || 'none'}
Sample procedures: ${samples || 'none'}

Return a JSON object with:
{
  "name": string,
  "facilityType": string,
  "description": string,
  "tags": string[],
  "address": { "street": string, "city": string, "state": string, "zipCode": string, "country": string },
  "location": { "latitude": number, "longitude": number },
  "phoneNumber": string,
  "email": string,
  "website": string
}
Only include fields you are confident about.`;
}

async function callLLMAPI(prompt: string, config: FacilityLLMConfig): Promise<string> {
  const payload = {
    model: config.model,
    messages: [
      { role: 'system', content: config.systemPrompt || defaultSystemPrompt() },
      { role: 'user', content: prompt },
    ],
    temperature: config.temperature ?? 0.3,
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

function parseLLMProfile(content: string): Partial<FacilityProfile> {
  const json = extractJson(content);
  if (!json) {
    throw new Error('Unable to parse LLM response');
  }
  const parsed = JSON.parse(json);
  return {
    name: parsed.name,
    facilityType: parsed.facilityType,
    description: parsed.description,
    tags: Array.isArray(parsed.tags) ? parsed.tags.map(normalizeTag).filter(Boolean) : undefined,
    address: parsed.address,
    location: parsed.location,
    phoneNumber: parsed.phoneNumber,
    email: parsed.email,
    website: parsed.website,
  };
}

function extractJson(content: string): string | null {
  const start = content.indexOf('{');
  const end = content.lastIndexOf('}');
  if (start === -1 || end === -1 || end <= start) {
    return null;
  }
  return content.slice(start, end + 1);
}

function normalizeTag(value: string): string {
  return normalizeIdentifier(value);
}

function mergeTags(base: string[], incoming: string[]): string[] {
  const merged = new Set<string>();
  for (const tag of base) {
    if (tag) {
      merged.add(tag);
    }
  }
  for (const tag of incoming) {
    if (tag) {
      merged.add(tag);
    }
  }
  return Array.from(merged).sort();
}

function collectFacilityTags(records: PriceData[]): string[] {
  const tags = new Set<string>();
  for (const record of records) {
    if (record.tags) {
      for (const tag of record.tags) {
        if (tag) {
          tags.add(normalizeTag(tag));
        }
      }
    }
    const hydrated = hydrateTags(record);
    for (const tag of hydrated.tags) {
      if (tag) {
        tags.add(normalizeTag(tag));
      }
    }
  }
  return Array.from(tags).sort();
}

function collectCuratedSources(records: PriceData[]): string[] {
  const sources = new Set<string>();
  for (const record of records) {
    const curated = record.tagMetadata?.curated;
    if (curated?.sources) {
      for (const source of curated.sources) {
        if (source) {
          sources.add(source);
        }
      }
    }
  }
  return Array.from(sources).sort();
}

function collectMatchedRules(records: PriceData[]): string[] {
  const rules = new Set<string>();
  for (const record of records) {
    const curated = record.tagMetadata?.curated;
    if (curated?.matchedRules) {
      for (const rule of curated.matchedRules) {
        if (rule) {
          rules.add(rule);
        }
      }
    }
  }
  return Array.from(rules).sort();
}

function collectSampleProcedures(records: PriceData[]): string[] {
  const samples = new Set<string>();
  for (const record of records) {
    if (record.procedureDescription) {
      samples.add(record.procedureDescription.trim());
    }
    if (samples.size >= 12) {
      break;
    }
  }
  return Array.from(samples);
}

function inferFacilityType(name: string, tags: string[]): string {
  const candidates = [name.toLowerCase(), ...tags.map((tag) => tag.toLowerCase())];
  for (const value of candidates) {
    if (value.includes('urgent')) {
      return 'urgent_care';
    }
    if (value.includes('imaging') || value.includes('radiology')) {
      return 'imaging_center';
    }
    if (value.includes('lab') || value.includes('laboratory') || value.includes('diagnostic')) {
      return 'diagnostic_lab';
    }
    if (value.includes('specialty')) {
      return 'specialty_clinic';
    }
    if (value.includes('surgery')) {
      return 'outpatient_surgery';
    }
    if (value.includes('clinic')) {
      return 'clinic';
    }
  }
  return 'hospital';
}

function defaultSystemPrompt(): string {
  return `You are a healthcare data assistant.
Generate concise facility metadata that helps patients search for care.
Return JSON only, with fields you are confident about.`;
}