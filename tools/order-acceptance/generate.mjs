#!/usr/bin/env node

import { readFile, writeFile } from 'node:fs/promises';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

import { coverageFor } from './coverage.mjs';
import { extractMatrixCases, sha256, stableJSON, validateManifest } from './lib.mjs';

const toolDir = path.dirname(fileURLToPath(import.meta.url));
const repositoryRoot = path.resolve(toolDir, '../..');
const matrixPath = 'docs/quality/order-mvp-prd-acceptance-matrix.md';
const matrixText = await readFile(path.join(repositoryRoot, matrixPath), 'utf8');
const extracted = extractMatrixCases(matrixText);

const manifest = {
  schema_version: 1,
  source: {
    product_baseline: 'docs/product/online-ordering-system-prd-0818.md',
    frozen_architecture: 'docs/architecture/order-mvp-domain-schema-interfaces.md',
    acceptance_matrix: matrixPath,
    matrix_revision: 'ORDER-MVP-R2.2',
    matrix_sha256: sha256(matrixText),
    inventory_base_sha: process.argv[2] || null,
  },
  policy: {
    exact_counts: extracted.counts,
    local_levels: ['L1', 'L2', 'L3'],
    external_level: 'L4',
    ui0_is_not_ui1: true,
    module_pass_is_not_case_pass: true,
    local_missing_skip_or_nonzero_fails: true,
  },
  cases: extracted.rows.map(row => buildCase(row)),
};

validateManifest(manifest);
manifest.manifest_sha256 = sha256(stableJSON(manifest));
await writeFile(path.join(toolDir, 'manifest.json'), `${JSON.stringify(manifest, null, 2)}\n`, 'utf8');
process.stdout.write(`generated ${manifest.cases.length} cases manifest_sha256=${manifest.manifest_sha256}\n`);

function buildCase(row) {
  const available = coverageFor(row.case_id).map(item => ({
    evidence_id: `${row.case_id}:${item.profile_id}`,
    level: item.level,
    status: item.status,
    satisfies: item.satisfies === true,
    claim: item.claim,
    ...(item.selector ? { selector: item.selector } : { selectors: item.selectors }),
  }));
  const missing = row.required_local_levels
    .filter(level => !available.some(item => item.level === level && item.satisfies === true))
    .map(level => ({
      evidence_id: `${row.case_id}:missing:${level}`,
      level,
      status: 'MISSING',
      satisfies: true,
      shortest_gap: shortestGap(row, level),
    }));
  return {
    case_id: row.case_id,
    group: row.group,
    scenario: row.scenario,
    matrix_evidence: row.matrix_evidence,
    required_local_levels: row.required_local_levels,
    local_evidence: [...available, ...missing],
    external_evidence: row.requires_l4 ? [{
      level: 'L4',
      status: 'BLOCKED_EXTERNAL',
      reason: externalReason(row.case_id),
    }] : [],
  };
}

function shortestGap(row, level) {
  if (level === 'L1') {
    return `Add one exact pure-rule selector for ${row.case_id}: assert "${plain(row.scenario.expected)}" and the fail-closed boundary "${plain(row.scenario.failure_protection)}".`;
  }
  if (level === 'L2') {
    return `Add one exact-SHA composed HTTP/worker + fresh MySQL scenario for ${row.case_id}: execute ${plain(row.scenario.http)}, assert ${plain(row.scenario.mysql_facts)}, expected "${plain(row.scenario.expected)}", and the failure shield.`;
  }
  return `Add one exact-SHA rendered UI1 scenario for ${row.case_id} against the composed local API, fresh MySQL and deterministic fake providers: perform "${plain(row.scenario.ui_operation)}" and assert HTTP, MySQL, visible result and zero false-success side effects.`;
}

function externalReason(caseID) {
  if (/^(?:AC|INV)-0?6$|^PAGE-U06$/u.test(caseID)) return 'Real WeChat Pay JSAPI, callback and funds require approved AppID/merchant assets.';
  if (/refund|14$/iu.test(caseID) || ['PAGE-PC02', 'PAGE-U07'].includes(caseID)) return 'Real refund funds/callback require approved WeChat Pay merchant assets.';
  if (['PAGE-M04'].includes(caseID)) return 'Real camera scan requires logged-in WeChat DevTools or device UI3.';
  if (/PC05|PC08/u.test(caseID)) return 'Real COS bucket/CAM/domain objects require customer Tencent Cloud assets.';
  if (/PC10|AC-16|INV-13/u.test(caseID)) return 'Real WeChat scan/phone authorization requires platform login and account data.';
  if (/AC-15|INV-15|BE-21/u.test(caseID)) return 'Real subscription delivery requires approved WeChat templates and user authorization.';
  return 'Real WeChat profile, phone, platform UI or customer asset evidence remains external to local acceptance.';
}

function plain(value) {
  return value.replace(/[`*_]/gu, '').replace(/\s+/gu, ' ').trim();
}
