#!/usr/bin/env node

import { spawnSync } from 'node:child_process';
import { mkdir, readFile, writeFile } from 'node:fs/promises';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

import {
  extractMatrixCases,
  parseGoTestJSON,
  sha256,
  stableJSON,
  summarizeManifest,
  validateManifest,
} from './lib.mjs';

const toolDir = path.dirname(fileURLToPath(import.meta.url));
const repositoryRoot = path.resolve(toolDir, '../..');
const command = process.argv[2] || 'validate';
const options = parseOptions(process.argv.slice(3));
const manifest = JSON.parse(await readFile(path.join(toolDir, 'manifest.json'), 'utf8'));
validateManifest(manifest);
await validateSourceBinding(manifest);

if (command === 'validate') {
  const summary = summarizeManifest(manifest.cases);
  process.stdout.write(`${JSON.stringify({ type: 'validation', valid: true, counts: manifest.policy.exact_counts, inventory: summary })}\n`);
  process.exit(0);
}

if (!['inventory', 'run'].includes(command)) usage();
const selectedCases = selectCases(manifest.cases, options.caseIDs);
const candidateSHA = command === 'run' ? exactCandidateSHA(options.candidateSHA) : null;
if (command === 'run') requireCleanExactTree();

const selectorResults = new Map();
if (command === 'run') {
  const selectors = new Map();
  for (const entry of selectedCases) {
    for (const evidence of entry.local_evidence) {
      if (evidence.status !== 'AVAILABLE' || evidence.satisfies !== true) continue;
      for (const selector of evidence.selector ? [evidence.selector] : evidence.selectors) {
        selectors.set(stableJSON(selector), selector);
      }
    }
  }
  for (const [key, selector] of selectors) selectorResults.set(key, executeSelector(selector));
}

const runtimeEvidence = new Map();
const records = [];
for (const entry of selectedCases) {
  for (const evidence of entry.local_evidence) {
    const runtime = evidenceRuntime(evidence, selectorResults, command);
    runtimeEvidence.set(evidence.evidence_id, runtime);
  }
  const local = entry.required_local_levels.map(level => {
    const evidence = entry.local_evidence.filter(item => item.level === level && item.satisfies === true);
    const statuses = evidence.map(item => runtimeEvidence.get(item.evidence_id).status);
    const status = evidence.length === 0 || statuses.includes('MISSING') ? 'MISSING' :
      statuses.every(value => value === 'PASS' || value === 'AVAILABLE_NOT_RUN') ?
        (command === 'run' ? 'PASS' : 'AVAILABLE_NOT_RUN') : 'FAIL';
    return { level, status, evidence_ids: evidence.map(item => item.evidence_id) };
  });
  records.push({
    type: 'case',
    candidate_sha: candidateSHA,
    case_id: entry.case_id,
    group: entry.group,
    required_local_levels: entry.required_local_levels,
    local,
    local_status: local.every(item => item.status === 'PASS' || item.status === 'AVAILABLE_NOT_RUN') ?
      (command === 'run' ? 'PASS' : 'AVAILABLE_NOT_RUN') :
      local.some(item => item.status === 'MISSING') ? 'MISSING' : 'FAIL',
    external: entry.external_evidence,
    supporting_available: entry.local_evidence
      .filter(item => item.status === 'AVAILABLE' && item.satisfies !== true)
      .map(item => ({ evidence_id: item.evidence_id, level: item.level, claim: item.claim })),
  });
}

const summary = summarizeManifest(selectedCases, runtimeEvidence);
const summaryRecord = {
  type: 'summary',
  candidate_sha: candidateSHA,
  manifest_sha256: manifest.manifest_sha256,
  matrix_sha256: manifest.source.matrix_sha256,
  selected_case_count: selectedCases.length,
  ...summary,
};
const output = `${records.map(record => JSON.stringify(record)).join('\n')}\n${JSON.stringify(summaryRecord)}\n`;
process.stdout.write(output);
if (options.output) {
  const outputPath = path.resolve(repositoryRoot, options.output);
  const ownedScratch = path.resolve(repositoryRoot, '.scratch/overnight-acceptance');
  if (outputPath !== ownedScratch && !outputPath.startsWith(`${ownedScratch}${path.sep}`)) {
    throw new Error('output must stay below .scratch/overnight-acceptance');
  }
  await mkdir(path.dirname(outputPath), { recursive: true });
  await writeFile(outputPath, output, 'utf8');
}
process.exitCode = summary.ok ? 0 : 1;

function executeSelector(selector) {
  const startedAt = new Date().toISOString();
  try {
    let result;
    if (selector.kind === 'go-test') {
      result = spawnSync('go', ['test', '-json', selector.package, '-run', `^${selector.test}$`, '-count=1'], executeOptions());
      parseGoTestJSON(result.stdout || '', result.status ?? 1, selector.test);
    } else if (selector.kind === 'node-test') {
      const args = ['--test', '--test-reporter=tap'];
      if (selector.name_pattern !== '.*') args.push('--test-name-pattern', selector.name_pattern);
      args.push(selector.file);
      result = spawnSync('node', args, executeOptions());
      parseNodeTestTAP(result.stdout || '', result.status ?? 1);
    } else {
      result = spawnSync(selector.argv[0], selector.argv.slice(1), executeOptions());
      if ((result.status ?? 1) !== 0) throw new Error(`command exit code ${result.status ?? 1}`);
    }
    return {
      status: 'PASS',
      started_at: startedAt,
      finished_at: new Date().toISOString(),
      output_sha256: sha256(`${result.stdout || ''}\n${result.stderr || ''}`),
    };
  } catch (error) {
    return {
      status: /skipped/iu.test(error.message) ? 'SKIP' : 'FAIL',
      started_at: startedAt,
      finished_at: new Date().toISOString(),
      error: sanitizeError(error.message),
    };
  }
}

function evidenceRuntime(evidence, selectorResults, mode) {
  if (evidence.status === 'MISSING') return { status: 'MISSING', shortest_gap: evidence.shortest_gap };
  if (mode !== 'run' || evidence.satisfies !== true) return { status: 'AVAILABLE_NOT_RUN' };
  const selectors = evidence.selector ? [evidence.selector] : evidence.selectors;
  const results = selectors.map(selector => selectorResults.get(stableJSON(selector)));
  if (results.some(result => !result || result.status === 'FAIL')) return { status: 'FAIL' };
  if (results.some(result => result.status === 'SKIP')) return { status: 'SKIP' };
  return { status: 'PASS', selector_output_sha256: results.map(result => result.output_sha256) };
}

function parseNodeTestTAP(output, exitCode) {
  if (exitCode !== 0) throw new Error(`node test exit code ${exitCode}`);
  const skipped = output.match(/^# skipped (\d+)$/mu);
  if (skipped && Number(skipped[1]) > 0) throw new Error(`node test skipped ${skipped[1]}`);
  const passed = output.match(/^# pass (\d+)$/mu);
  if (!passed || Number(passed[1]) < 1) throw new Error('node test did not report a pass');
}

async function validateSourceBinding(currentManifest) {
  const declaredManifestSHA = currentManifest.manifest_sha256;
  const unhashedManifest = structuredClone(currentManifest);
  delete unhashedManifest.manifest_sha256;
  if (!/^[0-9a-f]{64}$/u.test(declaredManifestSHA || '') || sha256(stableJSON(unhashedManifest)) !== declaredManifestSHA) {
    throw new Error('manifest SHA is missing or does not match its content');
  }
  const matrixPath = path.join(repositoryRoot, currentManifest.source.acceptance_matrix);
  const matrixText = await readFile(matrixPath, 'utf8');
  if (sha256(matrixText) !== currentManifest.source.matrix_sha256) throw new Error('acceptance matrix SHA drifted; regenerate manifest');
  const extracted = extractMatrixCases(matrixText);
  const manifestByID = new Map(currentManifest.cases.map(entry => [entry.case_id, entry]));
  for (const row of extracted.rows) {
    const entry = manifestByID.get(row.case_id);
    if (!entry || stableJSON(entry.scenario) !== stableJSON(row.scenario) ||
      stableJSON(entry.required_local_levels) !== stableJSON(row.required_local_levels) ||
      (entry.external_evidence.length > 0) !== row.requires_l4) {
      throw new Error(`manifest mapping drifted for ${row.case_id}`);
    }
  }
}

function exactCandidateSHA(presented) {
  const value = presented || git(['rev-parse', 'HEAD']).trim();
  if (!/^[0-9a-f]{40}$/u.test(value)) throw new Error('candidate SHA must be exactly 40 lowercase hex characters');
  const resolved = git(['rev-parse', `${value}^{commit}`]).trim();
  if (resolved !== value) throw new Error('candidate SHA does not resolve exactly');
  const head = git(['rev-parse', 'HEAD']).trim();
  if (head !== value) throw new Error(`checked-out HEAD ${head} is not candidate ${value}`);
  return value;
}

function requireCleanExactTree() {
  const status = git(['status', '--porcelain=v1', '--untracked-files=all']);
  if (status.trim() !== '') throw new Error('exact-SHA run requires a clean worktree');
}

function git(args) {
  const result = spawnSync('git', args, executeOptions());
  if ((result.status ?? 1) !== 0) throw new Error(`git ${args[0]} failed`);
  return result.stdout;
}

function executeOptions() {
  return {
    cwd: repositoryRoot,
    env: process.env,
    encoding: 'utf8',
    timeout: 5 * 60 * 1000,
    maxBuffer: 16 * 1024 * 1024,
  };
}

function selectCases(cases, caseIDs) {
  if (caseIDs.length === 0) return cases;
  const requested = new Set(caseIDs);
  const selected = cases.filter(entry => requested.has(entry.case_id));
  if (selected.length !== requested.size) {
    const found = new Set(selected.map(entry => entry.case_id));
    throw new Error(`unknown CaseID: ${[...requested].filter(caseID => !found.has(caseID)).join(',')}`);
  }
  return selected;
}

function parseOptions(args) {
  const parsed = { candidateSHA: null, caseIDs: [], output: null };
  for (let index = 0; index < args.length; index += 1) {
    const name = args[index];
    if (name === '--candidate-sha') parsed.candidateSHA = requireValue(args, ++index, name);
    else if (name === '--case') parsed.caseIDs.push(requireValue(args, ++index, name));
    else if (name === '--output') parsed.output = requireValue(args, ++index, name);
    else usage();
  }
  return parsed;
}

function requireValue(args, index, name) {
  if (!args[index]) throw new Error(`${name} requires a value`);
  return args[index];
}

function sanitizeError(message) {
  return String(message).replace(/[\r\n\t]+/gu, ' ').slice(0, 200);
}

function usage() {
  process.stderr.write('usage: node tools/order-acceptance/run.mjs validate|inventory|run [--case CASE] [--candidate-sha SHA] [--output .scratch/overnight-acceptance/file.jsonl]\n');
  process.exit(2);
}
