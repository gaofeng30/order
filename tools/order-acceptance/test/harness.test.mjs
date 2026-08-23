import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import test from 'node:test';

import {
  extractMatrixCases,
  parseGoTestJSON,
  summarizeManifest,
  validateManifest,
} from '../lib.mjs';

const expectedCounts = Object.freeze({
  total: 95,
  page: 25,
  pageUser: 9,
  pageMerchant: 4,
  pagePC: 12,
  acceptance: 19,
  boundary: 35,
  invariant: 16,
});

test('canonical matrix yields exactly 95 unique cases in frozen groups', () => {
  const cases = extractMatrixCases(`# Matrix
| CaseID | role | UI | HTTP | MySQL | expected | shield | evidence | status |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
${[
    ...Array.from({ length: 9 }, (_, index) => `| PAGE-U${String(index + 1).padStart(2, '0')} | u | ui | GET /x | users | ok | closed | L3 | NOT_RUN |`),
    ...Array.from({ length: 4 }, (_, index) => `| PAGE-M${String(index + 2).padStart(2, '0')} | m | ui | GET /x | orders | ok | closed | L3 | NOT_RUN |`),
    ...Array.from({ length: 12 }, (_, index) => `| PAGE-PC${String(index + 1).padStart(2, '0')} | p | ui | GET /x | orders | ok | closed | L3 | NOT_RUN |`),
    ...Array.from({ length: 19 }, (_, index) => `| AC-${String(index + 1).padStart(2, '0')} | u | ui | GET /x | orders | ok | closed | L2+L3 | NOT_RUN |`),
    ...Array.from({ length: 35 }, (_, index) => `| BE-${String(index + 1).padStart(2, '0')} | u | ui | GET /x | orders | ok | closed | L3 | NOT_RUN |`),
    ...Array.from({ length: 16 }, (_, index) => `| INV-${String(index + 1).padStart(2, '0')} | u | ui | GET /x | orders | ok | closed | L1+L2 | NOT_RUN |`),
  ].join('\n')}`);

  assert.deepEqual(cases.counts, expectedCounts);
  assert.equal(new Set(cases.rows.map(row => row.case_id)).size, 95);
});

test('manifest validation rejects duplicate IDs and any count drift', () => {
  const manifest = {
    schema_version: 1,
    source: { matrix_revision: 'ORDER-MVP-R2.2' },
    cases: [minimalCase('AC-01'), minimalCase('AC-01')],
  };
  assert.throws(() => validateManifest(manifest), /duplicate CaseID AC-01/);
});

test('local MISSING fails while L4 BLOCKED_EXTERNAL stays independently visible', () => {
  const localMissing = minimalCase('AC-01');
  localMissing.local_evidence = [{
    level: 'L3',
    status: 'MISSING',
    shortest_gap: 'one rendered UI1 plus fake-provider and MySQL scenario',
  }];
  localMissing.external_evidence = [{ level: 'L4', status: 'BLOCKED_EXTERNAL', reason: 'real WeChat assets' }];

  const summary = summarizeManifest([localMissing]);
  assert.equal(summary.local_ready, 0);
  assert.equal(summary.local_missing, 1);
  assert.equal(summary.external_blocked, 1);
  assert.equal(summary.ok, false);
});

test('L4 external block does not fail an otherwise complete local case', () => {
  const localReady = minimalCase('AC-01');
  localReady.required_local_levels = ['L3'];
  localReady.local_evidence = [{
    level: 'L3',
    status: 'AVAILABLE',
    satisfies: true,
    selector: { kind: 'node-test', file: 'test/example.test.mjs', name_pattern: '^scenario$' },
  }];
  localReady.external_evidence = [{ level: 'L4', status: 'BLOCKED_EXTERNAL', reason: 'real WeChat assets' }];

  const summary = summarizeManifest(Array.from({ length: 95 }, () => structuredClone(localReady)));
  assert.equal(summary.local_ready, 95);
  assert.equal(summary.local_missing, 0);
  assert.equal(summary.external_blocked, 95);
  assert.equal(summary.ok, true);
});

test('go test JSON requires an actual selected pass and rejects skip/nonzero', () => {
  assert.deepEqual(parseGoTestJSON([
    JSON.stringify({ Action: 'run', Package: 'p', Test: 'TestExact' }),
    JSON.stringify({ Action: 'pass', Package: 'p', Test: 'TestExact' }),
    JSON.stringify({ Action: 'pass', Package: 'p' }),
  ].join('\n'), 0, 'TestExact'), { passed: true, test: 'TestExact' });

  assert.throws(() => parseGoTestJSON(JSON.stringify({ Action: 'skip', Package: 'p', Test: 'TestExact' }), 0, 'TestExact'), /skipped/);
  assert.throws(() => parseGoTestJSON(JSON.stringify({ Action: 'pass', Package: 'p' }), 0, 'TestExact'), /did not pass/);
  assert.throws(() => parseGoTestJSON('', 1, 'TestExact'), /exit code 1/);
});

test('committed inventory keeps every case explicit and preserves current gaps', async () => {
  const manifest = JSON.parse(await readFile(new URL('../manifest.json', import.meta.url), 'utf8'));
  validateManifest(manifest);
  const summary = summarizeManifest(manifest.cases);
  assert.deepEqual({
    total: summary.total,
    localReady: summary.local_ready,
    localMissing: summary.local_missing,
    externalBlocked: summary.external_blocked,
  }, { total: 95, localReady: 4, localMissing: 91, externalBlocked: 23 });
  assert.equal(manifest.cases.every(entry => entry.local_evidence.length > 0), true);
});

function minimalCase(caseID) {
  return {
    case_id: caseID,
    group: 'AC',
    scenario: {
      role: 'user',
      ui_operation: 'operate',
      http: 'GET /x',
      mysql_facts: 'orders',
      expected: 'ok',
      failure_protection: 'closed',
    },
    required_local_levels: ['L3'],
    local_evidence: [],
    external_evidence: [],
  };
}
