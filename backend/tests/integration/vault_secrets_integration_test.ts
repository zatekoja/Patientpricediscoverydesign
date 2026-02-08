import assert from 'assert';
import crypto from 'crypto';
import { loadVaultSecretsToEnv } from '../../config/vaultSecrets';

async function runTest(name: string, fn: () => void | Promise<void>): Promise<void> {
  try {
    await fn();
    console.log(`✓ ${name}`);
  } catch (err) {
    console.error(`✗ ${name}`);
    console.error(err);
    process.exitCode = 1;
  }
}

async function main() {
  const addr = process.env.TEST_VAULT_ADDR || process.env.VAULT_ADDR || '';
  const token = process.env.TEST_VAULT_TOKEN || process.env.VAULT_TOKEN || '';
  const mount = process.env.TEST_VAULT_MOUNT || 'secret';

  if (!addr || !token) {
    console.log('Skipping integration test: TEST_VAULT_ADDR/TEST_VAULT_TOKEN not set');
    return;
  }

  const ready = await vaultReady(addr);
  if (!ready) {
    console.log('Skipping integration test: Vault not reachable');
    return;
  }

  const path = `patient-price-discovery/tests/${randomId()}`;
  const payload = {
    OPENAI_API_KEY: 'vault-test-openai',
    GEOLOCATION_API_KEY: 'vault-test-geo',
    PROVIDER_LLM_API_KEY: 'vault-test-provider',
  };

  await writeVaultSecret(addr, token, mount, path, payload);

  const prevOpenAI = process.env.OPENAI_API_KEY;
  const prevGeo = process.env.GEOLOCATION_API_KEY;
  const prevProvider = process.env.PROVIDER_LLM_API_KEY;

  await runTest('loadVaultSecretsToEnv loads KV v2 secrets into env', async () => {
    const result = await loadVaultSecretsToEnv({
      enabled: true,
      addr,
      token,
      mount,
      path,
      kvVersion: 2,
      overwrite: true,
      timeoutMs: 3000,
    });

    assert.strictEqual(result.enabled, true);
    assert.strictEqual(result.error, undefined);
    assert.ok(result.loaded >= 3, 'expected secrets to be loaded');
    assert.strictEqual(process.env.OPENAI_API_KEY, payload.OPENAI_API_KEY);
    assert.strictEqual(process.env.GEOLOCATION_API_KEY, payload.GEOLOCATION_API_KEY);
    assert.strictEqual(process.env.PROVIDER_LLM_API_KEY, payload.PROVIDER_LLM_API_KEY);
  });

  restoreEnv('OPENAI_API_KEY', prevOpenAI);
  restoreEnv('GEOLOCATION_API_KEY', prevGeo);
  restoreEnv('PROVIDER_LLM_API_KEY', prevProvider);

  if (process.exitCode) {
    process.exit(1);
  }
}

main().catch((err) => {
  console.error(err);
  process.exit(1);
});

function restoreEnv(key: string, value?: string) {
  if (value === undefined) {
    delete process.env[key];
  } else {
    process.env[key] = value;
  }
}

function randomId(): string {
  return crypto.randomBytes(6).toString('hex');
}

async function vaultReady(addr: string): Promise<boolean> {
  try {
    const resp = await fetch(`${addr.replace(/\/+$/, '')}/v1/sys/health`);
    return resp.ok || resp.status === 429;
  } catch {
    return false;
  }
}

async function writeVaultSecret(
  addr: string,
  token: string,
  mount: string,
  path: string,
  data: Record<string, string>
): Promise<void> {
  const base = addr.replace(/\/+$/, '');
  const cleanMount = mount.replace(/^\/+|\/+$/g, '');
  const cleanPath = path.replace(/^\/+/, '');
  const url = `${base}/v1/${cleanMount}/data/${cleanPath}`;
  const response = await fetch(url, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'X-Vault-Token': token,
    },
    body: JSON.stringify({ data }),
  });
  if (!response.ok) {
    const text = await response.text();
    throw new Error(`Vault write failed: ${response.status} ${response.statusText} ${text}`);
  }
}
