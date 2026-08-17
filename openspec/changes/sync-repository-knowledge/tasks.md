## 1. Boundary and approval

- [x] 1.1 Record W0/UI0, owner, owned paths, dependency, external assets, non-goals and the user's neat-freak authorization; run strict planning validation and checkpoint `DRAFT → APPROVED`.
  - Evidence: the artifacts fix the five-document owned-path boundary, `gate_type=W0`, `ui_level_target=UI0`, `ui_level_actual=UI0`, no external assets, and dependency `add-lightweight-harness-loop@79c4f1d8ebf300bf2f9f4c226fcd2c2aa2643963`; `openspec validate sync-repository-knowledge --strict` passed before implementation and the local lifecycle reached `APPROVED`.

## 2. Red

- [x] 2.1 Run the focused documentation check and record failure on the stale all-mock/client-not-connected/schema-not-implemented claims and missing Harness entry.
  - Red: the focused Python check returned `DOC_SYNC=FAIL` on six stale claims across the root README, mini-program README, Demo guide and technical document, plus the missing `./tools/harness status` entry.

## 3. Green

- [x] 3.1 Update the root and mini-program READMEs with one hybrid as-built boundary, current catalog client paths, runtime setup and the thin Harness entry.
  - Evidence: both READMEs now identify the API-backed home/menu/detail slice, the remaining local state, required local runtime and the two read-only Harness entry commands.
- [x] 3.2 Update the docs index, Demo guide and technical document with the same implemented/unimplemented and verification boundaries.
  - Evidence: the docs index routes developers through the root README; Demo and technical documents separate the implemented catalog slice from local prototype behavior and unimplemented production architecture.
- [x] 3.3 Rerun the focused content check and local relative-link audit to Green.
  - Green: the Python standard-library audit returned `DOC_SYNC=PASS files=5 links=local-relative-verified` with every stale phrase absent and required handoff fact present.

## 4. Refactor and writer gate

- [x] 4.1 Remove duplication, keep AGENTS/CLAUDE/customer/contracts/archive unchanged, and rerun size, stale-phrase, relative-time, link and owned-path checks.
  - Refactor: `git diff --check`, the five-document content/link audit, byte comparison against base and the owned-path script passed with 10/10 intended paths; root governance, Skills, customer material, contracts, operation guides and archives remain unchanged. New uses of “当前” describe the explicitly scoped as-built boundary; product uses such as “今日” remain business semantics rather than stale handoff dates.
- [x] 4.2 Run OpenSpec change/all strict, `./tools/harness check`, Markdown/diff checks and the workflow-quality repository regression required for this documentation handoff.
  - Writer regression: change strict and all strict passed `14/14`; Harness check and focused Harness tests passed (`14` tests); five stage Skills passed `quick_validate.py` with `/usr/bin/python3`; Go format, test, race, vet, build and smoke passed; all app JavaScript parsed, `44` JSON files parsed, and mini-program UI1 passed `13/13`.
- [x] 4.3 Record W0/UI0 evidence and C/T/V/R score, commit only owned paths, checkpoint `CANDIDATE` and return the exact clean SHA.
  - Writer verdict prepared: `{ gate_type: W0, ui_level_target: UI0, ui_level_actual: UI0, base_sha: 79c4f1d8ebf300bf2f9f4c226fcd2c2aa2643963, candidate_sha: external-post-commit, score: C9/T10/V8/R9=36, hard_blockers: 0, unverified_boundary: no external URL availability, real MySQL, WeChat tool/device, payment, UAT, deployment or production proof }`. The exact clean SHA and receipt replay are checked after the immutable commit, before the lifecycle checkpoint.

## 5. Independent verification

- [ ] 5.1 Verify the exact candidate read-only in a separate clean detached worktree; any content/base/SHA change invalidates the result.
- [ ] 5.2 Record PASS/FAIL and remaining external boundaries without modifying candidate bytes.
