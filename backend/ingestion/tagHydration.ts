import { PriceData } from '../types/PriceData';

interface FacilityTagDefinition {
  key: string;
  aliases: string[];
  tags: string[];
  sources: string[];
}

interface TagRule {
  id: string;
  pattern: RegExp;
  tags: string[];
}

export interface CuratedTagMetadata {
  sources: string[];
  facilityTags: string[];
  ruleTags: string[];
  metadataTags: string[];
  matchedRules: string[];
}

const FACILITY_TAGS: FacilityTagDefinition[] = [
  {
    key: 'lasuth',
    aliases: [
      'lagos state university teaching hospital',
      'lagos state university teaching hospital price list',
      'lasuth',
    ],
    tags: [
      'teaching_hospital',
      'hospital',
      'lagos_state',
      'ikeja',
      'nigeria',
      'emergency',
    ],
    sources: [
      'services.gphas.org',
    ],
  },
  {
    key: 'randle',
    aliases: [
      'randle general hospital',
      'randle general hospital price list',
      'surulere general hospital',
    ],
    tags: [
      'general_hospital',
      'hospital',
      'lagos_state',
      'surulere',
      'nigeria',
      'emergency',
      'maternal_child_health',
      'dental',
      'medical',
      'surgical',
      'obstetrics_gynecology',
      'gynecology',
      'vct',
      'dots',
      'pediatrics',
      'pharmacy',
      'laboratory',
      'radiology',
      'blood_bank',
      'inpatient',
    ],
    sources: [
      'lshsc.com.ng',
      'panafrican-med-journal.com',
    ],
  },
  {
    key: 'ijede',
    aliases: [
      'ijede general hospital',
      'general hospital ijede',
    ],
    tags: [
      'general_hospital',
      'hospital',
      'lagos_state',
      'ikorodu',
      'ijede',
      'nigeria',
    ],
    sources: [
      'adekunlegoldfoundation.com',
    ],
  },
];

const TAG_RULES: TagRule[] = [
  {
    id: 'obgyn',
    pattern: /\b(antenatal|prenatal|postnatal|labou?r|delivery|c[-\s]?section|c\/s|obstet|gyn|o&g|o\/g)\b/i,
    tags: ['obstetrics_gynecology', 'maternity'],
  },
  {
    id: 'pediatrics',
    pattern: /\b(paed|pediatric|paediatric|child|neonate|newborn|infant)\b/i,
    tags: ['pediatrics'],
  },
  {
    id: 'dental',
    pattern: /\b(dental|tooth|dentist)\b/i,
    tags: ['dental'],
  },
  {
    id: 'imaging',
    pattern: /\b(x[-\s]?ray|radiograph|ultrasound|scan|ct|mri|echo)\b/i,
    tags: ['imaging', 'radiology'],
  },
  {
    id: 'laboratory',
    pattern: /\b(lab|laboratory|haematology|hematology|microbiology|pathology|urine|blood|culture)\b/i,
    tags: ['laboratory', 'diagnostics'],
  },
  {
    id: 'surgery',
    pattern: /\b(surgery|surgical|theatre|operation|operative|laparotomy|laparoscopy)\b/i,
    tags: ['surgery', 'operating_theatre'],
  },
  {
    id: 'anaesthesia',
    pattern: /\b(anaesth|anesth|sedation)\b/i,
    tags: ['anesthesia'],
  },
  {
    id: 'inpatient',
    pattern: /\b(ward|admission|accommodation|bed|inpatient)\b/i,
    tags: ['inpatient', 'accommodation'],
  },
  {
    id: 'emergency',
    pattern: /\b(emergency|casualty|accident|triage|observation)\b/i,
    tags: ['emergency'],
  },
  {
    id: 'ambulance',
    pattern: /\b(ambulance)\b/i,
    tags: ['ambulance', 'emergency'],
  },
  {
    id: 'oxygen',
    pattern: /\b(oxygen)\b/i,
    tags: ['oxygen_therapy'],
  },
  {
    id: 'outpatient',
    pattern: /\b(consultation|clinic|outpatient|opd)\b/i,
    tags: ['outpatient'],
  },
  {
    id: 'administrative',
    pattern: /\b(card|folder|registration|appointment|certificate|report|notification of birth|police report|sick leave|maternity leave)\b/i,
    tags: ['administrative', 'documentation'],
  },
  {
    id: 'consumables',
    pattern: /\b(consumable|pack|gown|pad|underlay|dressing)\b/i,
    tags: ['consumables'],
  },
  {
    id: 'physiotherapy',
    pattern: /\b(physio|physiotherapy|rehab)\b/i,
    tags: ['physiotherapy', 'rehabilitation'],
  },
  {
    id: 'pharmacy',
    pattern: /\b(pharmacy|drug|medication)\b/i,
    tags: ['pharmacy'],
  },
  {
    id: 'blood_bank',
    pattern: /\b(blood bank|transfusion)\b/i,
    tags: ['blood_bank'],
  },
  {
    id: 'icu',
    pattern: /\b(icu|intensive care)\b/i,
    tags: ['critical_care', 'icu'],
  },
  {
    id: 'ophthalmology',
    pattern: /\b(ophthal|ophthamol|cataract|glaucoma|retina|cornea|eyelid|lacrimal|orbital|pterygium|iop|tonometry|refraction|eye)\b/i,
    tags: ['ophthalmology', 'eye_care'],
  },
  {
    id: 'ent',
    pattern: /\b(ent|tonsil|adenoid|septoplasty|rhinoplasty|mastoid|myringotomy|turbinect|laryngoscopy|tracheostomy|ear\s*nose|otolaryngol)\b/i,
    tags: ['ent', 'ear_nose_throat'],
  },
  {
    id: 'urology',
    pattern: /\b(urol|catheter|endourol|lithotrip|cystoscopy|prostat|circumcision)\b/i,
    tags: ['urology'],
  },
  {
    id: 'oncology',
    pattern: /\b(oncol|chemo|chemotherapy|radiotherapy|cancer)\b/i,
    tags: ['oncology'],
  },
  {
    id: 'stroke',
    pattern: /\b(stroke|cerebrovascular)\b/i,
    tags: ['stroke', 'neurology'],
  },
  {
    id: 'dermatology',
    pattern: /\b(dermatol|skin biopsy|hyfrecation|cryotherapy|chemical peel)\b/i,
    tags: ['dermatology'],
  },
  {
    id: 'dietary',
    pattern: /\b(diet|dietary|nutrition|nutritional|feeding|meal)\b/i,
    tags: ['dietary', 'nutrition'],
  },
  {
    id: 'psychiatry',
    pattern: /\b(psychiatr|psycholog|mental|electroconvulsive|ect|hypnotherapy)\b/i,
    tags: ['psychiatry', 'mental_health'],
  },
  {
    id: 'orthopaedics',
    pattern: /\b(ortho|orthop|plaster of paris|fracture|pop)\b/i,
    tags: ['orthopaedics'],
  },
  {
    id: 'endoscopy',
    pattern: /\b(endoscop|colonoscop|polypectomy|peg tube|variceal band|phototherapy)\b/i,
    tags: ['endoscopy', 'gastroenterology'],
  },
  {
    id: 'eeg',
    pattern: /\b(eeg|electroencephalogr)\b/i,
    tags: ['eeg', 'neurology'],
  },
  {
    id: 'vip',
    pattern: /\b(vip|accelerated care)\b/i,
    tags: ['vip', 'premium_care'],
  },
  {
    id: 'reports',
    pattern: /\b(medical report|death certificate|police report|assault fee|adoption fee|notification of death)\b/i,
    tags: ['reports', 'administrative'],
  },
  {
    id: 'sti_testing',
    pattern: /\b(hiv|sti|std|vdrl|hepatitis|sexual\s*health|gonorrh|chlamydia|syphilis)\b/i,
    tags: ['sti_testing', 'diagnostics'],
  },
];

export function applyCuratedTags(items: PriceData[]): PriceData[] {
  return items.map((item) => attachCuratedTags(item));
}

export function hydrateTags(item: PriceData): { tags: string[]; tagMetadata: { curated: CuratedTagMetadata } } {
  const tags = new Set<string>();
  const sources = new Set<string>();
  const facilityTags: string[] = [];
  const ruleTags: string[] = [];
  const metadataTags: string[] = [];
  const matchedRules: string[] = [];

  const facilityMatch = matchFacility(item.facilityName || '');
  if (facilityMatch) {
    for (const tag of facilityMatch.tags) {
      const normalized = normalizeTag(tag);
      if (normalized) {
        tags.add(normalized);
        facilityTags.push(normalized);
      }
    }
    for (const source of facilityMatch.sources) {
      sources.add(source);
    }
  }

  for (const tag of tagsFromMetadata(item)) {
    const normalized = normalizeTag(tag);
    if (normalized) {
      tags.add(normalized);
      metadataTags.push(normalized);
    }
  }

  const text = [
    item.procedureDescription,
    item.procedureCode,
    item.metadata?.category,
    item.metadata?.area,
  ]
    .filter(Boolean)
    .join(' ');

  for (const rule of TAG_RULES) {
    if (rule.pattern.test(text)) {
      matchedRules.push(rule.id);
      for (const tag of rule.tags) {
        const normalized = normalizeTag(tag);
        if (normalized) {
          tags.add(normalized);
          ruleTags.push(normalized);
        }
      }
    }
  }

  if (item.price === 0) {
    tags.add('free');
    metadataTags.push('free');
  }

  const finalTags = Array.from(new Set(tags)).sort();

  return {
    tags: finalTags,
    tagMetadata: {
      curated: {
        sources: Array.from(sources).sort(),
        facilityTags: Array.from(new Set(facilityTags)).sort(),
        ruleTags: Array.from(new Set(ruleTags)).sort(),
        metadataTags: Array.from(new Set(metadataTags)).sort(),
        matchedRules: Array.from(new Set(matchedRules)).sort(),
      },
    },
  };
}

function attachCuratedTags(item: PriceData): PriceData {
  const hydrated = hydrateTags(item);
  if (hydrated.tags.length === 0) {
    return item;
  }

  const mergedTags = mergeTags(item.tags, hydrated.tags);
  const mergedMetadata = mergeTagMetadata(item.tagMetadata, hydrated.tagMetadata);

  return {
    ...item,
    tags: mergedTags,
    tagMetadata: mergedMetadata,
  };
}

function tagsFromMetadata(item: PriceData): string[] {
  const tags: string[] = [];
  const category = typeof item.metadata?.category === 'string' ? item.metadata?.category : undefined;
  const area = typeof item.metadata?.area === 'string' ? item.metadata?.area : undefined;
  const unit = typeof item.metadata?.unit === 'string' ? item.metadata?.unit : undefined;
  const priceTier = typeof item.metadata?.priceTier === 'string' ? item.metadata?.priceTier : undefined;

  if (category) {
    tags.push(category);
  }
  if (area) {
    tags.push(area);
  }
  if (unit) {
    tags.push(unit);
  }
  if (priceTier) {
    tags.push(priceTier);
  }

  return tags;
}

function mergeTags(existing: string[] | undefined, incoming: string[]): string[] {
  const merged = new Set<string>();
  if (existing) {
    for (const tag of existing) {
      const normalized = normalizeTag(tag);
      if (normalized) {
        merged.add(normalized);
      }
    }
  }
  for (const tag of incoming) {
    const normalized = normalizeTag(tag);
    if (normalized) {
      merged.add(normalized);
    }
  }
  return Array.from(merged).sort();
}

function mergeTagMetadata(
  existing: PriceData['tagMetadata'] | undefined,
  incoming: { curated: CuratedTagMetadata }
): PriceData['tagMetadata'] {
  if (!existing) {
    return incoming;
  }
  const existingCurated = existing.curated;
  if (!existingCurated) {
    return {
      ...existing,
      ...incoming,
    };
  }

  return {
    ...existing,
    curated: {
      sources: mergeUnique(existingCurated.sources, incoming.curated.sources),
      facilityTags: mergeUnique(existingCurated.facilityTags, incoming.curated.facilityTags),
      ruleTags: mergeUnique(existingCurated.ruleTags, incoming.curated.ruleTags),
      metadataTags: mergeUnique(existingCurated.metadataTags, incoming.curated.metadataTags),
      matchedRules: mergeUnique(existingCurated.matchedRules, incoming.curated.matchedRules),
    },
  };
}

function mergeUnique(existing?: string[], incoming?: string[]): string[] {
  const merged = new Set<string>();
  if (existing) {
    for (const tag of existing) {
      merged.add(tag);
    }
  }
  if (incoming) {
    for (const tag of incoming) {
      merged.add(tag);
    }
  }
  return Array.from(merged).sort();
}

function matchFacility(facilityName: string): FacilityTagDefinition | undefined {
  const normalized = normalizeFacilityKey(facilityName);
  if (!normalized) {
    return undefined;
  }
  return FACILITY_TAGS.find((facility) =>
    facility.aliases.some((alias) => normalized.includes(normalizeFacilityKey(alias)))
  );
}

function normalizeFacilityKey(value: string): string {
  return (value || '').toLowerCase().replace(/[^a-z0-9]/g, '');
}

function normalizeTag(value: string | undefined): string {
  if (!value) {
    return '';
  }
  return value
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, '_')
    .replace(/^_+|_+$/g, '')
    .replace(/_+/g, '_');
}
