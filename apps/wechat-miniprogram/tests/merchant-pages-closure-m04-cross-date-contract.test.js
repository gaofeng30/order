const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const test = require('node:test');

const root = path.resolve(__dirname, '../../..');
const runnerPath = path.join(__dirname, 'run-merchant-pages-closure-ui1.mjs');
const specPath = path.join(__dirname, 'merchant-pages-closure-ui1.spec.cjs');
const gatePath = path.join(root, '.scratch/page-m04-cross-date-redeem/verify-writer.sh');

test('PAGE-M04 cross-date direct redemption has one exact rendered gate', () => {
  assert.equal(fs.existsSync(runnerPath), true, 'missing merchant closure runner');
  assert.equal(fs.existsSync(specPath), true, 'missing merchant rendered spec');
  assert.equal(fs.existsSync(gatePath), true, 'missing PAGE-M04 fresh-v44 writer gate');

  const runner = fs.readFileSync(runnerPath, 'utf8');
  const spec = fs.readFileSync(specPath, 'utf8');
  const gate = fs.readFileSync(gatePath, 'utf8');
  for (const marker of [
    '/api/v1/verify/code',
    'crossDateOrders',
    'advanceTomorrowFixtureToPreparing',
    'same pickup code is scoped to the current service date',
  ]) assert.match(runner, new RegExp(marker.replaceAll('/', '\\/')), `runner missing ${marker}`);
  assert.match(spec, /renders and directly redeems today and tomorrow orders with the same four-digit code/);
  assert.match(gate, /order-mysql-w3/);
  assert.match(gate, /run-merchant-pages-closure-ui1\.mjs/);
});
