const assert = require('node:assert/strict');
const test = require('node:test');

const { createHarness } = require('./page-harness.js');

test('subscription template deployment config is empty until real assets exist', () => {
  assert.deepEqual(require('../utils/subscriptionTemplateConfig.js'), {
    develop: { READY: '', REFUND_RESULT: '' },
    trial: { READY: '', REFUND_RESULT: '' },
    release: { READY: '', REFUND_RESULT: '' },
  });
});

test('template resolver accepts only bounded exact public template IDs', () => {
  const { resolveSubscriptionTemplateIds } = require('../utils/subscriptionTemplate.js');
  assert.deepEqual(resolveSubscriptionTemplateIds('trial', {
    trial: { READY: 'ready-template-01', REFUND_RESULT: 'refund-template-02' },
  }), { READY: 'ready-template-01', REFUND_RESULT: 'refund-template-02' });

  for (const [envVersion, config] of [
    ['unknown', { unknown: { READY: 'template' } }],
    ['trial', { trial: { READY: ' ready-template' } }],
    ['release', { release: { READY: 'ready template' } }],
    ['develop', { develop: { READY: '模板' } }],
    ['develop', { develop: { READY: 'x'.repeat(129) } }],
  ]) {
    assert.deepEqual(resolveSubscriptionTemplateIds(envVersion, config), {});
  }
  assert.deepEqual(resolveSubscriptionTemplateIds('develop', {
    develop: { READY: 'ready-only', REFUND_RESULT: '' },
  }), { READY: 'ready-only' });
});

test('cold launch exposes only templates configured for the detected environment', () => {
  const harness = createHarness();
  global.wx.getAccountInfoSync = () => ({ miniProgram: { envVersion: 'develop' } });
  const config = require('../utils/subscriptionTemplateConfig.js');
  config.develop.READY = 'develop-ready-template';
  config.trial.READY = 'trial-ready-template';

  const app = harness.loadApp();
  assert.deepEqual(app.globalData.subscriptionTemplateIds, { READY: 'develop-ready-template' });
});
