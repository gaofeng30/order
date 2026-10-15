const test = require('node:test');
const assert = require('node:assert/strict');
const runtime = require('../../apps/wechat-miniprogram/utils/runtimeEndpoint');
const config = require('../../apps/wechat-miniprogram/utils/runtimeEndpointConfig');
const wxFor = envVersion => ({ getAccountInfoSync: () => ({miniProgram:{envVersion}}) });

test('release uses the confirmed production origin', () => {
  const endpoint = runtime.resolveRuntimeEndpoint(wxFor('release'), config);
  assert.equal(endpoint.state, 'ready');
  assert.equal(endpoint.origin, 'https://order.suianwuye.com');
  assert.equal(runtime.isRuntimeEndpointReady(endpoint, endpoint.origin), true);
});
test('release rejects UAT and other origins even if config is replaced', () => {
  for (const origin of [config.trial, 'https://uat.suianwuye.com', 'https://example.com', 'https://order.suianwuye.com:8443']) {
    const endpoint = runtime.resolveRuntimeEndpoint(wxFor('release'), {release:origin});
    assert.equal(endpoint.state,'error', origin);
    assert.equal(runtime.isRuntimeEndpointReady({state:'ready',envVersion:'release',origin},origin),false,origin);
  }
});
test('develop and trial stay isolated from production', () => {
  for (const env of ['develop','trial']) {
    assert.equal(runtime.resolveRuntimeEndpoint(wxFor(env), config).origin, config.trial);
    assert.equal(runtime.resolveRuntimeEndpoint(wxFor(env), {[env]:'https://order.suianwuye.com'}).state,'error');
  }
});
