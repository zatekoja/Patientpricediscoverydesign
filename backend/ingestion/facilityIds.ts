export function normalizeIdentifier(value?: string): string {
  const trimmed = (value || '').trim().toLowerCase();
  if (!trimmed) {
    return '';
  }
  let out = '';
  let lastUnderscore = false;
  for (const ch of trimmed) {
    const isAlphaNum =
      (ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9');
    if (isAlphaNum) {
      out += ch;
      lastUnderscore = false;
      continue;
    }
    if (!lastUnderscore) {
      out += '_';
      lastUnderscore = true;
    }
  }
  return out.replace(/^_+|_+$/g, '');
}

const FACILITY_ALIAS_MAP: Array<{ canonical: string; aliases: string[] }> = [
  {
    canonical: 'Lagos State University Teaching Hospital (LASUTH)',
    aliases: [
      'LASUTH',
      'Lagos State University Teaching Hospital',
      'Lagos State University Teaching Hospital (LASUTH)',
      'Lagos State University Teaching Hospital Price List',
      'Lagos State University Teaching Hospital Price List (LASUTH Services Only)',
      'New LASUTH Price List',
      'New LASUTH Price List (Services)',
      'LASUTH Services Only',
      'LASUTH Price List',
      'LASUTH Services',
    ],
  },
  {
    canonical: 'General Hospital Badagry',
    aliases: [
      'General Hospital Badagry',
      'Badagry General Hospital',
      'General Hospital Badagry Price List',
      'General Hospital Badagry (Docx)',
    ],
  },
  {
    canonical: 'Randle General Hospital',
    aliases: [
      'Randle General Hospital',
      'Price List For Randle General Hospital January 2026',
      'Randle General Hospital January 2026',
    ],
  },
];

const NON_FACILITY_KEYS = new Set([
  'pricelist',
  'pricelistforofficeuse',
  'officeuse',
  'officeuse1',
  'pricelistforofficeuse1',
  'healthcarefacility',
  'medicalfacility',
  'hospital',
  'clinic',
]);

const MONTH_TOKENS = [
  'jan', 'january',
  'feb', 'february',
  'mar', 'march',
  'apr', 'april',
  'may',
  'jun', 'june',
  'jul', 'july',
  'aug', 'august',
  'sep', 'sept', 'september',
  'oct', 'october',
  'nov', 'november',
  'dec', 'december',
];

const GENERIC_FACILITY_TOKENS = new Set([
  'healthcare',
  'medical',
  'facility',
  'hospital',
  'clinic',
  'general',
  'center',
  'centre',
]);

export function normalizeFacilityName(value?: string, sourceFile?: string): string {
  let candidate = sanitizeFacilityName(value);
  if (!candidate && sourceFile) {
    candidate = sanitizeFacilityName(sourceFile);
  }
  if (!candidate) {
    return '';
  }

  const aliasMatch = matchFacilityAlias(candidate);
  if (aliasMatch) {
    return aliasMatch;
  }

  candidate = stripFacilityNoise(candidate);
  if (!candidate) {
    return '';
  }

  if (isGenericFacilityName(candidate) && sourceFile) {
    const fromFile = stripFacilityNoise(sanitizeFacilityName(sourceFile));
    if (fromFile && !isGenericFacilityName(fromFile)) {
      candidate = fromFile;
    }
  }

  const aliasMatchAfter = matchFacilityAlias(candidate);
  if (aliasMatchAfter) {
    return aliasMatchAfter;
  }

  const normalizedKey = normalizeFacilityKey(candidate);
  if (!normalizedKey || NON_FACILITY_KEYS.has(normalizedKey) || isGenericFacilityName(candidate)) {
    return '';
  }

  return formatFacilityName(candidate);
}

function sanitizeFacilityName(value?: string): string {
  if (!value) {
    return '';
  }
  const trimmed = value.trim();
  if (!trimmed) {
    return '';
  }
  const withoutExt = trimmed.replace(/\.[^.]+$/, '');
  return withoutExt
    .replace(/[_]+/g, ' ')
    .replace(/\s+/g, ' ')
    .trim();
}

function stripFacilityNoise(value: string): string {
  let cleaned = value;
  cleaned = cleaned.replace(/\[[^\]]*\]/g, ' ');
  cleaned = cleaned.replace(/\(([^)]+)\)/g, (match, inner) => {
    const innerTrimmed = inner.trim();
    if (/services?\s*only/i.test(innerTrimmed) || /office\s*use/i.test(innerTrimmed)) {
      return ' ';
    }
    return match;
  });
  cleaned = cleaned.replace(/\bprice\s*list\b/gi, ' ');
  cleaned = cleaned.replace(/\bpricelist\b/gi, ' ');
  cleaned = cleaned.replace(/\bprice\s*sheet\b/gi, ' ');
  cleaned = cleaned.replace(/\bprice\s*schedule\b/gi, ' ');
  cleaned = cleaned.replace(/\btariff\s*list\b/gi, ' ');
  cleaned = cleaned.replace(/\bservices?\s*only\b/gi, ' ');
  cleaned = cleaned.replace(/\bservices?\s*list\b/gi, ' ');
  cleaned = cleaned.replace(/\bfor\s*office\s*use\b/gi, ' ');
  cleaned = cleaned.replace(/\boffice\s*use\b/gi, ' ');
  cleaned = cleaned.replace(new RegExp(`\\b(${MONTH_TOKENS.join('|')})\\b`, 'gi'), ' ');
  cleaned = cleaned.replace(/\b20\d{2}\b/g, ' ');
  cleaned = cleaned.replace(/[^\w\s&().'-]/g, ' ');
  cleaned = cleaned.replace(/\s+/g, ' ').trim();
  return cleaned;
}

function matchFacilityAlias(value: string): string | null {
  const key = normalizeFacilityKey(value);
  if (!key) {
    return null;
  }
  for (const entry of FACILITY_ALIAS_MAP) {
    for (const alias of entry.aliases) {
      if (normalizeFacilityKey(alias) === key) {
        return entry.canonical;
      }
    }
  }
  return null;
}

function normalizeFacilityKey(value: string): string {
  return (value || '').toLowerCase().replace(/[^a-z0-9]/g, '');
}

function isGenericFacilityName(value: string): boolean {
  const tokens = value
    .toLowerCase()
    .split(/[^a-z0-9]+/)
    .filter(Boolean);
  if (tokens.length === 0) {
    return true;
  }
  const nonGeneric = tokens.filter((token) => !GENERIC_FACILITY_TOKENS.has(token));
  return nonGeneric.length === 0;
}

function formatFacilityName(value: string): string {
  const words = value.split(/\s+/).filter(Boolean);
  const formatted = words.map((word) => {
    const cleaned = word.replace(/[^a-zA-Z0-9&().'-]/g, '');
    if (!cleaned) {
      return word;
    }
    const acronymCandidate = cleaned.replace(/^[()]+|[()]+$/g, '');
    const acronymLetters = acronymCandidate.replace(/[^A-Za-z0-9]/g, '');
    if (
      acronymLetters &&
      acronymLetters === acronymLetters.toUpperCase() &&
      acronymLetters.length <= 5
    ) {
      return cleaned.toUpperCase();
    }
    const lower = cleaned.toLowerCase();
    return lower.charAt(0).toUpperCase() + lower.slice(1);
  });
  return formatted.join(' ').trim();
}

export function buildFacilityId(providerId: string | undefined, facilityName: string | undefined): string {
  const normalized = normalizeIdentifier(facilityName);
  if (!normalized) {
    return '';
  }
  const prefix = (providerId || 'provider').trim();
  if (!prefix) {
    return normalized;
  }
  return `${prefix}_${normalized}`;
}
