import { createHash } from 'node:crypto';

export const FROZEN_COUNTS = Object.freeze({
  total: 95,
  page: 25,
  pageUser: 9,
  pageMerchant: 4,
  pagePC: 12,
  acceptance: 19,
  boundary: 35,
  invariant: 16,
});

const localLevels = new Set(['L1', 'L2', 'L3']);
const scenarioFields = [
  'role',
  'ui_operation',
  'http',
  'mysql_facts',
  'expected',
  'failure_protection',
];

export function extractMatrixCases(markdown) {
  if (typeof markdown !== 'string') throw new Error('matrix must be text');
  const rows = [];
  const seen = new Set();
  for (const line of markdown.split(/\r?\n/u)) {
    if (!/^\|\s*(?:PAGE-(?:U|M|PC)\d{2}|AC-\d{2}|BE-\d{2}|INV-\d{2})\s*\|/u.test(line)) continue;
    const cells = line.slice(1, -1).split('|').map(cell => cell.trim());
    if (cells.length !== 9) throw new Error(`matrix row must have 9 columns: ${line}`);
    const [caseID, role, ui, http, mysql, expected, failureProtection, evidenceLevel, status] = cells;
    if (seen.has(caseID)) throw new Error(`duplicate matrix CaseID ${caseID}`);
    seen.add(caseID);
    const required = [...new Set((evidenceLevel.match(/L[123]/gu) || []))];
    if (required.length === 0) throw new Error(`matrix CaseID ${caseID} has no local evidence level`);
    rows.push({
      case_id: caseID,
      group: groupFor(caseID),
      scenario: {
        role,
        ui_operation: ui,
        http,
        mysql_facts: mysql,
        expected,
        failure_protection: failureProtection,
      },
      matrix_evidence: evidenceLevel,
      matrix_status: status,
      required_local_levels: required,
      requires_l4: /L4/u.test(evidenceLevel),
    });
  }
  const counts = countCases(rows);
  assertFrozenCounts(counts, 'matrix');
  return { rows, counts };
}

export function validateManifest(manifest) {
  if (!manifest || typeof manifest !== 'object' || Array.isArray(manifest)) throw new Error('manifest must be an object');
  if (manifest.schema_version !== 1) throw new Error('manifest schema_version must be 1');
  if (!Array.isArray(manifest.cases)) throw new Error('manifest cases must be an array');

  const seen = new Set();
  for (const entry of manifest.cases) {
    if (!entry || typeof entry !== 'object') throw new Error('manifest case must be an object');
    if (typeof entry.case_id !== 'string' || entry.case_id.length === 0) throw new Error('manifest case_id is required');
    if (seen.has(entry.case_id)) throw new Error(`duplicate CaseID ${entry.case_id}`);
    seen.add(entry.case_id);
  }
  assertFrozenCounts(countCases(manifest.cases), 'manifest');

  for (const entry of manifest.cases) {
    if (entry.group !== groupFor(entry.case_id)) throw new Error(`${entry.case_id} has wrong group ${entry.group}`);
    if (!entry.scenario || typeof entry.scenario !== 'object') throw new Error(`${entry.case_id} scenario is required`);
    for (const field of scenarioFields) {
      if (typeof entry.scenario[field] !== 'string' || entry.scenario[field].trim() === '') {
        throw new Error(`${entry.case_id} scenario.${field} is required`);
      }
    }
    if (!Array.isArray(entry.required_local_levels) || entry.required_local_levels.length === 0) {
      throw new Error(`${entry.case_id} required_local_levels is empty`);
    }
    if (new Set(entry.required_local_levels).size !== entry.required_local_levels.length ||
      entry.required_local_levels.some(level => !localLevels.has(level))) {
      throw new Error(`${entry.case_id} required_local_levels is invalid`);
    }
    if (!Array.isArray(entry.local_evidence)) throw new Error(`${entry.case_id} local_evidence must be an array`);
    const evidenceIDs = new Set();
    for (const evidence of entry.local_evidence) {
      validateLocalEvidence(entry.case_id, evidence);
      if (evidenceIDs.has(evidence.evidence_id)) throw new Error(`${entry.case_id} duplicate evidence_id ${evidence.evidence_id}`);
      evidenceIDs.add(evidence.evidence_id);
    }
    if (!Array.isArray(entry.external_evidence)) throw new Error(`${entry.case_id} external_evidence must be an array`);
    for (const evidence of entry.external_evidence) {
      if (evidence.level !== 'L4' || !['AVAILABLE', 'BLOCKED_EXTERNAL'].includes(evidence.status)) {
        throw new Error(`${entry.case_id} external evidence must be L4 AVAILABLE or BLOCKED_EXTERNAL`);
      }
      if (typeof evidence.reason !== 'string' || evidence.reason.trim() === '') {
        throw new Error(`${entry.case_id} external evidence reason is required`);
      }
    }
  }
  return manifest;
}

export function summarizeManifest(cases, runtimeResults = new Map()) {
  const result = {
    total: cases.length,
    local_ready: 0,
    local_missing: 0,
    local_not_run: 0,
    local_failed: 0,
    external_blocked: 0,
    ok: false,
  };
  for (const entry of cases) {
    result.external_blocked += entry.external_evidence.filter(evidence => evidence.status === 'BLOCKED_EXTERNAL').length;
    let missing = false;
    let notRun = false;
    let failed = false;
    for (const level of entry.required_local_levels) {
      const candidates = entry.local_evidence.filter(evidence => evidence.level === level && evidence.satisfies === true);
      if (candidates.length === 0 || candidates.some(evidence => evidence.status !== 'AVAILABLE')) {
        missing = true;
        continue;
      }
      for (const evidence of candidates) {
        const runtime = runtimeResults.get(evidence.evidence_id);
        if (runtime?.status === 'AVAILABLE_NOT_RUN') notRun = true;
        else if (runtime && runtime.status !== 'PASS') failed = true;
      }
    }
    if (missing) result.local_missing += 1;
    else if (failed) result.local_failed += 1;
    else if (notRun) result.local_not_run += 1;
    else result.local_ready += 1;
  }
  result.ok = result.total === FROZEN_COUNTS.total && result.local_missing === 0 && result.local_not_run === 0 && result.local_failed === 0;
  return result;
}

export function parseGoTestJSON(output, exitCode, expectedTest) {
  if (exitCode !== 0) throw new Error(`go test exit code ${exitCode}`);
  let passed = false;
  for (const line of output.split(/\r?\n/u)) {
    if (line.trim() === '') continue;
    let event;
    try {
      event = JSON.parse(line);
    } catch {
      continue;
    }
    if (event.Test !== expectedTest) continue;
    if (event.Action === 'skip') throw new Error(`selected test ${expectedTest} skipped`);
    if (event.Action === 'fail') throw new Error(`selected test ${expectedTest} failed`);
    if (event.Action === 'pass') passed = true;
  }
  if (!passed) throw new Error(`selected test ${expectedTest} did not pass`);
  return { passed: true, test: expectedTest };
}

export function stableJSON(value) {
  if (Array.isArray(value)) return `[${value.map(stableJSON).join(',')}]`;
  if (value && typeof value === 'object') {
    return `{${Object.keys(value).sort().map(key => `${JSON.stringify(key)}:${stableJSON(value[key])}`).join(',')}}`;
  }
  return JSON.stringify(value);
}

export function sha256(value) {
  return createHash('sha256').update(value).digest('hex');
}

function validateLocalEvidence(caseID, evidence) {
  if (!evidence || typeof evidence !== 'object') throw new Error(`${caseID} local evidence must be an object`);
  if (typeof evidence.evidence_id !== 'string' || evidence.evidence_id.trim() === '') {
    throw new Error(`${caseID} local evidence_id is required`);
  }
  if (![...localLevels, 'UI0', 'UI1'].includes(evidence.level)) throw new Error(`${caseID} invalid evidence level ${evidence.level}`);
  if (!['AVAILABLE', 'MISSING'].includes(evidence.status)) throw new Error(`${caseID} invalid local evidence status`);
  if (evidence.status === 'MISSING') {
    if (evidence.satisfies !== true || !localLevels.has(evidence.level)) {
      throw new Error(`${caseID} MISSING evidence must identify one required local level`);
    }
    if (typeof evidence.shortest_gap !== 'string' || evidence.shortest_gap.trim() === '') {
      throw new Error(`${caseID} MISSING evidence requires shortest_gap`);
    }
    return;
  }
  if (typeof evidence.claim !== 'string' || evidence.claim.trim() === '') {
    throw new Error(`${caseID} AVAILABLE evidence requires claim`);
  }
  if (!evidence.selector && !Array.isArray(evidence.selectors)) {
    throw new Error(`${caseID} AVAILABLE evidence requires selector(s)`);
  }
  const selectors = evidence.selector ? [evidence.selector] : evidence.selectors;
  if (selectors.length === 0) throw new Error(`${caseID} AVAILABLE evidence selectors is empty`);
  selectors.forEach(selector => validateSelector(caseID, selector));
}

function validateSelector(caseID, selector) {
  if (!selector || typeof selector !== 'object') throw new Error(`${caseID} selector must be an object`);
  if (selector.kind === 'go-test') {
    if (!/^\.\/services\/api\//u.test(selector.package) || !/^Test[A-Za-z0-9_]+$/u.test(selector.test)) {
      throw new Error(`${caseID} invalid go-test selector`);
    }
    return;
  }
  if (selector.kind === 'node-test') {
    if (typeof selector.file !== 'string' || selector.file.length === 0 || typeof selector.name_pattern !== 'string' || selector.name_pattern.length === 0) {
      throw new Error(`${caseID} invalid node-test selector`);
    }
    return;
  }
  if (selector.kind === 'command') {
    if (!Array.isArray(selector.argv) || selector.argv.length === 0 || selector.argv.some(value => typeof value !== 'string' || value.length === 0)) {
      throw new Error(`${caseID} invalid command selector`);
    }
    return;
  }
  throw new Error(`${caseID} unsupported selector kind ${selector.kind}`);
}

function groupFor(caseID) {
  if (/^PAGE-/u.test(caseID)) return 'PAGE';
  if (/^AC-/u.test(caseID)) return 'AC';
  if (/^BE-/u.test(caseID)) return 'BE';
  if (/^INV-/u.test(caseID)) return 'INV';
  throw new Error(`unsupported CaseID ${caseID}`);
}

function countCases(rows) {
  return {
    total: rows.length,
    page: rows.filter(row => /^PAGE-/u.test(row.case_id)).length,
    pageUser: rows.filter(row => /^PAGE-U/u.test(row.case_id)).length,
    pageMerchant: rows.filter(row => /^PAGE-M/u.test(row.case_id)).length,
    pagePC: rows.filter(row => /^PAGE-PC/u.test(row.case_id)).length,
    acceptance: rows.filter(row => /^AC-/u.test(row.case_id)).length,
    boundary: rows.filter(row => /^BE-/u.test(row.case_id)).length,
    invariant: rows.filter(row => /^INV-/u.test(row.case_id)).length,
  };
}

function assertFrozenCounts(actual, source) {
  for (const [key, expected] of Object.entries(FROZEN_COUNTS)) {
    if (actual[key] !== expected) throw new Error(`${source} ${key} count ${actual[key]} != ${expected}`);
  }
}
