const test = require('node:test');
const assert = require('node:assert/strict');
const { createHandler } = require('../../deploy/functions/order-compensation-wake/index.js');

test('forwards one durable wake only to the approved production endpoint', async () => {
  const calls = [];
  const handler = createHandler(async (url, options) => {
    calls.push({url, options});
    return { status: 204 };
  });
  assert.deepEqual(await handler({ token: 'a'.repeat(64) }), { accepted: true });
  assert.equal(calls.length, 1);
  assert.equal(calls[0].url, 'https://order.suianwuye.com/internal/compensation/wake');
  assert.equal(calls[0].options.method, 'POST');
  assert.equal(calls[0].options.redirect, 'error');
  assert.equal(calls[0].options.headers.Authorization, `Bearer ${'a'.repeat(64)}`);
  assert.equal(calls[0].options.body, undefined);
  assert.ok(calls[0].options.signal instanceof AbortSignal);
});

test('rejects malformed or expanded instructions before issuing any request', async () => {
  let calls = 0;
  const handler = createHandler(async () => { calls++; return {status:204}; });
  for (const event of [null, [], {}, {token:'x'}, {token:'A'.repeat(64)},
    {token:'a'.repeat(64), url:'https://example.invalid'},
    {token:'a'.repeat(64), kind:'refund'},
    {token:'a'.repeat(63)+'\n'}]) {
    assert.deepEqual(await handler(event), {accepted:false});
  }
  assert.equal(calls, 0);
});

test('transport failures and ambiguous replies do not retry or expose credentials', async () => {
  for (const behavior of [async () => ({status:200}), async () => ({status:503}),
    async () => { throw Error(`network error Bearer ${'a'.repeat(64)}`); }]) {
    let calls = 0;
    const handler = createHandler(async (...args) => { calls++; return behavior(...args); });
    assert.deepEqual(await handler({token:'a'.repeat(64)}), {accepted:false});
    assert.equal(calls, 1);
  }
});
