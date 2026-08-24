const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const test = require('node:test');

const root = path.resolve(__dirname, '../../..');
const runnerPath = path.join(root, 'tools/miniprogram-ui/run-ui1-composed-be22-be26.mjs');
const specPath = path.join(root, 'tools/miniprogram-ui/test/browser/ui1-composed-be22-be26.spec.cjs');

test('BE-22/BE-26 composed UI1 gate uses real facts instead of response projections', () => {
  assert.equal(fs.existsSync(runnerPath), true, 'missing composed BE-22/BE-26 runner');
  assert.equal(fs.existsSync(specPath), true, 'missing composed BE-22/BE-26 browser spec');
  const source = `${fs.readFileSync(runnerPath, 'utf8')}\n${fs.readFileSync(specPath, 'utf8')}`;
  for (const forbidden of ['forceUnboundPhone', 'filterReadyOrders']) {
    assert.equal(source.includes(forbidden), false, `forbidden response projection: ${forbidden}`);
  }
  for (const required of [
    'BOOTSTRAP_ROOT_OF_TRUST_FIXTURE',
    'getPhoneNumber:fail user deny',
    '/api/v1/me/primary-phone',
    '/api/v1/orders?active=true',
    '/api/v1/verify/scan',
    'READY_FOR_PICKUP',
    'COMPLETED',
  ]) {
    assert.equal(source.includes(required), true, `missing public-seam evidence: ${required}`);
  }
});
