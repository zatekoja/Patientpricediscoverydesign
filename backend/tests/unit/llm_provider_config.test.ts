import assert from 'assert';
import { LLMTagGeneratorProvider } from '../../providers/LLMTagGeneratorProvider';

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
  await runTest('LLM provider rejects missing apiEndpoint', async () => {
    const provider = new LLMTagGeneratorProvider();
    const valid = provider.validateConfig({
      apiKey: 'test-key',
      model: 'gpt-4',
    });
    assert.strictEqual(valid, false);
  });

  await runTest('LLM provider rejects missing apiKey', async () => {
    const provider = new LLMTagGeneratorProvider();
    const valid = provider.validateConfig({
      apiEndpoint: 'https://api.example.com',
      model: 'gpt-4',
    });
    assert.strictEqual(valid, false);
  });

  await runTest('LLM provider rejects missing model', async () => {
    const provider = new LLMTagGeneratorProvider();
    const valid = provider.validateConfig({
      apiEndpoint: 'https://api.example.com',
      apiKey: 'test-key',
    });
    assert.strictEqual(valid, false);
  });

  await runTest('LLM provider accepts valid config', async () => {
    const provider = new LLMTagGeneratorProvider();
    const valid = provider.validateConfig({
      apiEndpoint: 'https://api.openai.com/v1/chat/completions',
      apiKey: 'sk-test-key',
      model: 'gpt-4',
    });
    assert.strictEqual(valid, true);
  });

  await runTest('LLM provider warns about placeholder config', async () => {
    const provider = new LLMTagGeneratorProvider();
    let warningLogged = false;
    const originalWarn = console.warn;
    console.warn = (...args: any[]) => {
      const message = args.join(' ');
      if (message.includes('placeholder')) {
        warningLogged = true;
      }
      originalWarn.apply(console, args);
    };

    const valid = provider.validateConfig({
      apiEndpoint: 'placeholder',
      apiKey: 'placeholder',
      model: 'gpt-4',
    });

    console.warn = originalWarn;
    assert.strictEqual(valid, true); // Config is valid but should warn
    assert.strictEqual(warningLogged, true);
  });

  await runTest('LLM provider returns empty tags for placeholder config', async () => {
    const provider = new LLMTagGeneratorProvider();
    await provider.initialize({
      apiEndpoint: 'placeholder',
      apiKey: 'placeholder',
      model: 'gpt-4',
    });

    // Call the private callLLMAPI method to test placeholder detection
    const tags = await (provider as any).callLLMAPI('test prompt');
    assert(Array.isArray(tags));
    assert.strictEqual(tags.length, 0);
  });

  await runTest('LLM provider detects non-http endpoints as placeholder', async () => {
    const provider = new LLMTagGeneratorProvider();
    await provider.initialize({
      apiEndpoint: 'not-a-url',
      apiKey: 'test-key',
      model: 'gpt-4',
    });

    const tags = await (provider as any).callLLMAPI('test prompt');
    assert(Array.isArray(tags));
    assert.strictEqual(tags.length, 0);
  });

  if (process.exitCode) {
    process.exit(1);
  }
}

main().catch((err) => {
  console.error(err);
  process.exit(1);
});
