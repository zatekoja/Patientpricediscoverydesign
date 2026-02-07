const baseUrl = process.env.PROVIDER_API_BASE_URL || 'http://localhost:3002/api/v1';
const providerId = process.env.PROVIDER_ID || 'megalek_ateru_helper';

function assert(condition, message) {
  if (!condition) {
    throw new Error(message);
  }
}

async function fetchJson(path, options = {}) {
  const res = await fetch(`${baseUrl}${path}`, options);
  const text = await res.text();
  let json;
  try {
    json = text ? JSON.parse(text) : {};
  } catch (err) {
    throw new Error(`Invalid JSON response from ${path}: ${text}`);
  }
  if (!res.ok) {
    throw new Error(`Request failed ${res.status} ${res.statusText}: ${JSON.stringify(json)}`);
  }
  return json;
}

async function waitForHealthy() {
  const maxAttempts = 30;
  for (let attempt = 1; attempt <= maxAttempts; attempt += 1) {
    try {
      const health = await fetchJson('/health');
      if (health.status === 'ok') {
        return;
      }
    } catch (err) {
      // ignore and retry
    }
    await new Promise((resolve) => setTimeout(resolve, 1000));
  }
  throw new Error(`API did not become healthy at ${baseUrl}/health`);
}

async function run() {
  console.log(`[provider-api] baseUrl=${baseUrl}`);
  await waitForHealthy();

  console.log('✓ /health');
  const health = await fetchJson('/health');
  assert(health.status === 'ok', 'health.status should be ok');

  const providerList = await fetchJson('/provider/list');
  assert(Array.isArray(providerList.providers), 'provider list should be an array');
  const registered = providerList.providers.find((p) => p.id === providerId);
  assert(registered, `expected provider ${providerId} to be registered`);

  const providerHealth = await fetchJson(`/provider/health?providerId=${providerId}`);
  assert(typeof providerHealth.healthy === 'boolean', 'provider health should be boolean');
  assert(providerHealth.healthy, `provider ${providerId} should be healthy`);

  // Trigger sync twice to ensure previous batch tracking
  const sync1 = await fetchJson(`/sync/trigger?providerId=${providerId}`, { method: 'POST' });
  assert(sync1.success === true, 'sync trigger 1 should succeed');
  const sync2 = await fetchJson(`/sync/trigger?providerId=${providerId}`, { method: 'POST' });
  assert(sync2.success === true, 'sync trigger 2 should succeed');

  const syncStatus = await fetchJson(`/sync/status?providerId=${providerId}`);
  assert(typeof syncStatus.success === 'boolean', 'sync status should include success boolean');

  const current = await fetchJson(`/data/current?providerId=${providerId}`);
  assert(Array.isArray(current.data), 'current data should be an array');
  assert(current.data.length > 0, 'current data should not be empty');

  const previous = await fetchJson(`/data/previous?providerId=${providerId}`);
  assert(Array.isArray(previous.data), 'previous data should be an array');
  assert(previous.data.length > 0, 'previous data should not be empty after two syncs');

  const historical = await fetchJson(`/data/historical?providerId=${providerId}&timeWindow=1y`);
  assert(Array.isArray(historical.data), 'historical data should be an array');
  assert(historical.data.length > 0, 'historical data should not be empty');

  console.log('✓ provider API integration tests passed');
}

run().catch((err) => {
  console.error('provider API integration tests failed');
  console.error(err);
  process.exit(1);
});
