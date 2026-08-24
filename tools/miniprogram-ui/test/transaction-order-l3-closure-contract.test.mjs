import assert from 'node:assert/strict';
import { existsSync, readFileSync } from 'node:fs';
import path from 'node:path';
import test from 'node:test';
import { fileURLToPath } from 'node:url';

const here = path.dirname(fileURLToPath(import.meta.url));
const toolRoot = path.resolve(here, '..');
const repoRoot = path.resolve(toolRoot, '../..');
const runnerPath = path.join(toolRoot, 'run-ui1-transaction-order-l3.mjs');
const specPath = path.join(toolRoot, 'test/browser/ui1-transaction-order-l3.spec.cjs');
const configPath = path.join(toolRoot, 'karma.transaction-order-l3.conf.cjs');
const gatePath = path.join(repoRoot, '.scratch/transaction-order-l3-closure/verify-writer.sh');

test('transaction and order closure has one fresh-v44 rendered gate', () => {
  assert.equal(existsSync(runnerPath), true, 'missing transaction/order L3 runner');
  assert.equal(existsSync(specPath), true, 'missing transaction/order rendered spec');
  assert.equal(existsSync(configPath), true, 'missing transaction/order Karma config');
  assert.equal(existsSync(gatePath), true, 'missing transaction/order fresh-v44 writer gate');

  const runner = readFileSync(runnerPath, 'utf8');
  const spec = readFileSync(specPath, 'utf8');
  const gate = readFileSync(gatePath, 'utf8');
  const implementation = `${runner}\n${spec}`;
  for (const marker of [
    'TestTransactionOrderL3Server',
    'provider_create_count',
    'provider_query_count',
    'apply_sql_failure',
    'notification_provider_failure',
  ]) assert.match(implementation, new RegExp(marker), `gate implementation missing ${marker}`);
  for (const marker of [
    'requestPayment:fail cancel',
    'payment retry reuses one provider Create',
    'lost callback is recovered only by Query',
    'exact 30-minute cancellation stays unavailable',
    'rejected READY consent has a supplemental entry',
  ]) assert.match(spec, new RegExp(marker), `rendered spec missing ${marker}`);
  assert.match(gate, /order-mysql-w3/);
  assert.match(gate, /run-ui1-transaction-order-l3\.mjs/);
});
