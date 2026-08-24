const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const test = require('node:test');

test('PAGE-PC09 and PAGE-PC12 final L3 gate remains bounded to frozen matrix semantics', () => {
  const runnerPath = path.join(__dirname, 'composed-ui1-pc09-pc12-final-l3-runner.mjs');
  assert.equal(fs.existsSync(runnerPath), true, 'dedicated rendered final L3 runner is required');
  const source = fs.readFileSync(runnerPath, 'utf8');
  for (const marker of [
    'PAGE-PC09', 'PAGE-PC12', 'order-bootstrap', 'fresh-v44',
    'cumulative staff spend/order count', 'disabled staff current-fact revalidation',
    'same-phone import preserves disabled state', 'duplicate phone is visibly abnormal',
    'private API stopped and fresh schema cleanup completed',
  ]) assert.match(source, new RegExp(marker.replaceAll('-', '\\-')), `missing ${marker}`);
  assert.doesNotMatch(source, /127\.0\.0\.1:8080|order_local_e2e|INSERT\s+INTO|UPDATE\s+\w+\s+SET|DELETE\s+FROM/i);
});
