const test = require('node:test');
const assert = require('node:assert/strict');
const runtime = require('../../apps/wechat-miniprogram/utils/runtimeEndpoint');
const config = require('../../apps/wechat-miniprogram/utils/runtimeEndpointConfig');
const templates = require('../../apps/wechat-miniprogram/utils/subscriptionTemplateConfig');
const wxFor = envVersion => ({ getAccountInfoSync: () => ({miniProgram:{envVersion}}) });
const production = 'https://order.suianwuye.com';
const stages = ['develop', 'trial', 'release'];

test('production package uses its production origin at every WeChat distribution stage', () => {
  for (const stage of stages) {
    const endpoint = runtime.resolveRuntimeEndpoint(wxFor(stage), config);
    assert.equal(endpoint.state, 'ready', stage);
    assert.equal(endpoint.origin, production, stage);
    assert.equal(runtime.isRuntimeEndpointReady(endpoint, production), true, stage);
  }
});
test('every production stage rejects UAT, foreign, insecure and malformed origins', () => {
  for (const stage of stages) for (const origin of [
    'https://order-uat-d1gtzpzj50615c300-1473835269.ap-shanghai.app.tcloudbase.com',
    'https://uat.suianwuye.com', 'https://example.com',
    'https://order.suianwuye.com:8443', 'http://order.suianwuye.com',
    'http://127.0.0.1:8080', 'https://order.suianwuye.com.evil.example',
    'https://order.suianwuye.com@evil.example', 'https://order.suianwuye.com/pc/',
  ]) {
    assert.equal(runtime.resolveRuntimeEndpoint(wxFor(stage), {[stage]: origin}).state, 'error', `${stage} ${origin}`);
    assert.equal(runtime.isRuntimeEndpointReady({state:'ready',envVersion:stage,origin}, origin), false);
  }
});
test('unknown distribution stages and missing configuration fail closed', () => {
  assert.equal(runtime.resolveRuntimeEndpoint(wxFor('unknown'), config).state, 'error');
  for (const stage of stages) assert.equal(runtime.resolveRuntimeEndpoint(wxFor(stage), {}).state, 'error');
});
test('production subscription template configuration is complete at all stages', () => {
  for (const stage of stages) {
    assert.equal(templates[stage].READY, 'QNnxKlUem_FA7PQO8FkOrcI2RdHm2PsTzMtPMfukQBE', stage);
    assert.equal(templates[stage].REFUND_RESULT, 'A3ETXV00c4TQEFB8SOK2mkOhrnv0x0uR9sNbpK1QgWU', stage);
  }
});
