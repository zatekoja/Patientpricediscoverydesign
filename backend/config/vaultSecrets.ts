export interface VaultSecretsOptions {
  enabled?: boolean;
  addr?: string;
  token?: string;
  namespace?: string;
  mount?: string;
  path?: string;
  kvVersion?: 1 | 2;
  timeoutMs?: number;
  overwrite?: boolean;
}

export interface VaultSecretsResult {
  enabled: boolean;
  loaded: number;
  skipped: number;
  path?: string;
  error?: string;
}

const DEFAULT_TIMEOUT_MS = 5000;

export async function loadVaultSecretsToEnv(options?: VaultSecretsOptions): Promise<VaultSecretsResult> {
  const enabled = options?.enabled ?? process.env.VAULT_ENABLED === 'true';
  if (!enabled) {
    return { enabled: false, loaded: 0, skipped: 0 };
  }

  const addr = options?.addr ?? process.env.VAULT_ADDR;
  const token = options?.token ?? process.env.VAULT_TOKEN;
  const namespace = options?.namespace ?? process.env.VAULT_NAMESPACE;
  const mount = options?.mount ?? process.env.VAULT_MOUNT ?? 'secret';
  const path = options?.path ?? process.env.VAULT_PATH;
  const kvVersion = normalizeKVVersion(options?.kvVersion ?? process.env.VAULT_KV_VERSION);
  const timeoutEnv = parseInt(process.env.VAULT_TIMEOUT_MS || '', 10);
  const timeoutMs = options?.timeoutMs ?? (Number.isFinite(timeoutEnv) ? timeoutEnv : DEFAULT_TIMEOUT_MS);
  const overwrite = options?.overwrite ?? process.env.VAULT_OVERWRITE === 'true';

  if (!addr || !token || !path) {
    return {
      enabled: true,
      loaded: 0,
      skipped: 0,
      path,
      error: 'Vault configuration incomplete (VAULT_ADDR, VAULT_TOKEN, VAULT_PATH).',
    };
  }

  const url = buildVaultUrl(addr, mount, path, kvVersion);
  const controller = new AbortController();
  const timeout = setTimeout(() => controller.abort(), timeoutMs);

  try {
    const response = await fetch(url, {
      method: 'GET',
      headers: buildHeaders(token, namespace),
      signal: controller.signal,
    });

    if (!response.ok) {
      const text = await response.text();
      return {
        enabled: true,
        loaded: 0,
        skipped: 0,
        path,
        error: `Vault fetch failed: ${response.status} ${response.statusText} ${text}`,
      };
    }

    const payload = await response.json();
    const secretData = extractSecretData(payload, kvVersion);
    if (!secretData) {
      return {
        enabled: true,
        loaded: 0,
        skipped: 0,
        path,
        error: 'Vault response did not include secret data.',
      };
    }

    let loaded = 0;
    let skipped = 0;
    for (const [key, value] of Object.entries(secretData)) {
      if (!overwrite && process.env[key]) {
        skipped += 1;
        continue;
      }
      process.env[key] = stringifyVaultValue(value);
      loaded += 1;
    }

    return { enabled: true, loaded, skipped, path };
  } catch (error: any) {
    return {
      enabled: true,
      loaded: 0,
      skipped: 0,
      path,
      error: error?.message || 'Vault fetch failed.',
    };
  } finally {
    clearTimeout(timeout);
  }
}

function buildHeaders(token: string, namespace?: string): Record<string, string> {
  const headers: Record<string, string> = {
    'X-Vault-Token': token,
  };
  if (namespace) {
    headers['X-Vault-Namespace'] = namespace;
  }
  return headers;
}

function buildVaultUrl(addr: string, mount: string, path: string, kvVersion: 1 | 2): string {
  const cleanAddr = addr.replace(/\/+$/, '');
  const cleanMount = mount.replace(/^\/+|\/+$/g, '');
  const cleanPath = path.replace(/^\/+/, '');
  if (kvVersion === 2) {
    return `${cleanAddr}/v1/${cleanMount}/data/${cleanPath}`;
  }
  return `${cleanAddr}/v1/${cleanMount}/${cleanPath}`;
}

function normalizeKVVersion(value?: string | number | null): 1 | 2 {
  if (value === 1 || value === '1') {
    return 1;
  }
  return 2;
}

function extractSecretData(payload: any, kvVersion: 1 | 2): Record<string, unknown> | null {
  if (!payload || typeof payload !== 'object') {
    return null;
  }
  if (kvVersion === 2) {
    const data = payload?.data?.data;
    if (data && typeof data === 'object') {
      return data as Record<string, unknown>;
    }
    return null;
  }
  const data = payload?.data;
  if (data && typeof data === 'object') {
    return data as Record<string, unknown>;
  }
  return null;
}

function stringifyVaultValue(value: unknown): string {
  if (typeof value === 'string') {
    return value;
  }
  if (value === null || value === undefined) {
    return '';
  }
  if (typeof value === 'number' || typeof value === 'boolean') {
    return String(value);
  }
  try {
    return JSON.stringify(value);
  } catch {
    return String(value);
  }
}
