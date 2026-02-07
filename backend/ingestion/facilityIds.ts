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
