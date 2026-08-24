const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const test = require('node:test');

const testsRoot = __dirname;

test('AC-17 owns one fresh private API and three rendered clients without production shortcuts', () => {
  const runnerPath = path.join(testsRoot, 'composed-ui1-ac17-three-client-source-runner.mjs');
  const karmaPath = path.join(testsRoot, 'ac17-three-client-source-ui1.spec.cjs');
  const configPath = path.join(testsRoot, 'ac17-three-client-source-karma.cjs');
  assert.equal(fs.existsSync(runnerPath), true, 'dedicated AC-17 orchestrator is required');
  assert.equal(fs.existsSync(karmaPath), true, 'dedicated rendered Mini contract is required');
  assert.equal(fs.existsSync(configPath), true, 'dedicated rendered Mini launcher is required');

  const runner = fs.readFileSync(runnerPath, 'utf8');
  for (const marker of [
    'order-bootstrap',
    'order-api',
    'pc-rendered-write',
    'customer-before',
    'merchant-soldout',
    'customer-after',
    'customer-http-failure',
    'customer-bad-object',
    'private API stopped and fresh schema cleanup completed',
  ]) assert.match(runner, new RegExp(marker), `runner must prove ${marker}`);
  assert.doesNotMatch(runner, /127\.0\.0\.1:8080|order_local_e2e|INSERT\s+INTO|UPDATE\s+\w+\s+SET/i);

  const rendered = fs.readFileSync(karmaPath, 'utf8');
  for (const marker of ['pages/launch/launch', 'pages/menu/menu', 'pages/detail/detail', 'pages/admin-products/admin-products']) {
    assert.match(rendered, new RegExp(marker.replaceAll('/', '\\/')), `rendered Mini contract must load ${marker}`);
  }
  assert.match(rendered, /listState[^\n]+error/);
  assert.match(rendered, /actionState[^\n]+error/);
});
