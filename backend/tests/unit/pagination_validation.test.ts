import assert from 'assert';
import { DataProviderAPI } from '../../api/server';

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
  await runTest('validatePagination rejects NaN limit', () => {
    const api = new DataProviderAPI();
    const result = (api as any).validatePagination('abc', '0');
    assert.strictEqual(result.valid, false);
    assert(result.error.includes('limit must be a valid integer'));
  });

  await runTest('validatePagination rejects negative limit', () => {
    const api = new DataProviderAPI();
    const result = (api as any).validatePagination('-10', '0');
    assert.strictEqual(result.valid, false);
    assert(result.error.includes('limit must be non-negative'));
  });

  await runTest('validatePagination rejects limit exceeding max', () => {
    const api = new DataProviderAPI();
    const result = (api as any).validatePagination('10000', '0');
    assert.strictEqual(result.valid, false);
    assert(result.error.includes('limit must not exceed 5000'));
  });

  await runTest('validatePagination rejects NaN offset', () => {
    const api = new DataProviderAPI();
    const result = (api as any).validatePagination('100', 'xyz');
    assert.strictEqual(result.valid, false);
    assert(result.error.includes('offset must be a valid integer'));
  });

  await runTest('validatePagination rejects negative offset', () => {
    const api = new DataProviderAPI();
    const result = (api as any).validatePagination('100', '-5');
    assert.strictEqual(result.valid, false);
    assert(result.error.includes('offset must be non-negative'));
  });

  await runTest('validatePagination accepts valid limit and offset', () => {
    const api = new DataProviderAPI();
    const result = (api as any).validatePagination('100', '50');
    assert.strictEqual(result.valid, true);
    assert.strictEqual(result.limit, 100);
    assert.strictEqual(result.offset, 50);
  });

  await runTest('validatePagination uses defaults when undefined', () => {
    const api = new DataProviderAPI();
    const result = (api as any).validatePagination(undefined, undefined);
    assert.strictEqual(result.valid, true);
    assert.strictEqual(result.limit, 100);
    assert.strictEqual(result.offset, 0);
  });

  await runTest('validatePagination accepts max limit of 5000', () => {
    const api = new DataProviderAPI();
    const result = (api as any).validatePagination('5000', '0');
    assert.strictEqual(result.valid, true);
    assert.strictEqual(result.limit, 5000);
  });

  await runTest('validatePagination honors default limit override', () => {
    const api = new DataProviderAPI();
    const result = (api as any).validatePagination(undefined, undefined, { defaultLimit: 1000 });
    assert.strictEqual(result.valid, true);
    assert.strictEqual(result.limit, 1000);
    assert.strictEqual(result.offset, 0);
  });

  await runTest('validatePagination honors max limit override', () => {
    const api = new DataProviderAPI();
    const result = (api as any).validatePagination('6000', '0', { maxLimit: 6000 });
    assert.strictEqual(result.valid, true);
    assert.strictEqual(result.limit, 6000);
  });

  if (process.exitCode) {
    process.exit(1);
  }
}

main().catch((err) => {
  console.error(err);
  process.exit(1);
});
