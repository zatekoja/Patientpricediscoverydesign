import { describe, it } from 'node:test';
import assert from 'node:assert';
import http from 'node:http';

describe('Blnk Infrastructure', () => {
  it('should be reachable at http://localhost:5001', async () => {
    const response = await new Promise<{ statusCode?: number }>((resolve, reject) => {
      const req = http.get('http://localhost:5001/', (res) => {
        resolve({ statusCode: res.statusCode });
      });
      req.on('error', (err) => {
        resolve({ statusCode: undefined }); // connection refused
      });
    });

    assert.strictEqual(response.statusCode, 200, 'Blnk service is not reachable');
  });
});
