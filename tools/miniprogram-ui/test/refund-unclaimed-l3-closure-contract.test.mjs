import assert from 'node:assert/strict';
import { existsSync, readFileSync } from 'node:fs';
import path from 'node:path';
import test from 'node:test';
import { fileURLToPath } from 'node:url';

const here = path.dirname(fileURLToPath(import.meta.url));
const toolRoot = path.resolve(here, '..');
const repoRoot = path.resolve(toolRoot, '../..');

test('refund and unclaimed closure has one fresh-v44 rendered gate', () => {
  const runnerPath = path.join(toolRoot, 'run-ui1-refund-unclaimed-l3.mjs');
  const miniSpecPath = path.join(toolRoot, 'test/browser/ui1-refund-unclaimed-l3.spec.cjs');
  const gatePath = path.join(repoRoot, '.scratch/refund-unclaimed-l3-closure/verify-writer.sh');
  assert.equal(existsSync(runnerPath), true, 'missing refund/unclaimed L3 runner');
  assert.equal(existsSync(miniSpecPath), true, 'missing refund/unclaimed rendered Mini/Merchant spec');
  assert.equal(existsSync(gatePath), true, 'missing refund/unclaimed fresh-v44 writer gate');

  const implementation = `${readFileSync(runnerPath, 'utf8')}\n${readFileSync(miniSpecPath, 'utf8')}`;
  for (const marker of [
    'TestRefundUnclaimedL3Server', 'provider_create_count', 'provider_query_count',
    'UNKNOWN', 'PROCESSING', 'SUCCESS', 'unclaimed', 'month_revenue_cents', 'product_sales',
  ]) assert.match(implementation, new RegExp(marker), `gate implementation missing ${marker}`);
});
