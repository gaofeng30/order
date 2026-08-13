## 1. Approval and Frozen Context

- [x] 1.1 Before any edit to `.agents/skills/order-run-loop/**`, obtain the main Agent's explicit `APPROVED` decision; reread the apply context; verify branch `codex/enable-run-loop-self-evolution`, worktree ownership, clean planning state, dependency integration and `openspec validate enable-run-loop-self-evolution --strict`; record the approval and decisive results in `goal-checkpoint.md`.
  - Evidence (2026-08-13):
    ```yaml
    change: enable-run-loop-self-evolution
    gate_type: W1
    ui_level_target: UI0
    ui_level_actual: UI0
    base_sha: 2209c071a21860231827b2a8c8c81d9b7745e6e1
    candidate_sha: not-yet-created
    phase: writer
    command_or_action: main Agent explicit APPROVED gate; read apply context; verify branch, HEAD/main, only planned owned paths, canonical/archive dependencies; run openspec validate enable-run-loop-self-evolution --strict
    exit_result: PASS
    sanitized_summary: approval context, ownership, dependency archives and strict validation all matched; Skill remained unchanged
    artifact_or_environment: codex/enable-run-loop-self-evolution in /Users/vivix/.codex/worktrees/order-run-loop-self-evolution.Writer
    unverified_boundary: no implementation, candidate SHA, independent verification, integration or archive proven
    external_asset: { owner: N/A, missing: N/A, recovery: N/A }
    ```
- [x] 1.2 Recompute and compare the module base against repo SHA `2209c071a21860231827b2a8c8c81d9b7745e6e1`, Skill blob `d529461de5af1bf7cc65562e59ec3c84f0750963`, Skill SHA256 `558b549a4410d72d4c22acad621ffae96af3aeccd26adc186ede76601097aa59` and the explicit unversioned marker; stop instead of rewriting the frozen base if any value differs.
  - Evidence (2026-08-13):
    ```yaml
    change: enable-run-loop-self-evolution
    gate_type: W1
    ui_level_target: UI0
    ui_level_actual: UI0
    base_sha: 2209c071a21860231827b2a8c8c81d9b7745e6e1
    candidate_sha: not-yet-created
    phase: writer
    command_or_action: git rev-parse HEAD and HEAD:.agents/skills/order-run-loop/SKILL.md; shasum -a 256 SKILL.md; inspect front matter
    exit_result: PASS
    sanitized_summary: repo SHA, Skill blob, SHA256 and explicit unversioned marker exactly matched the frozen module base
    artifact_or_environment: frozen runner identity in goal-checkpoint.md
    unverified_boundary: identity match does not prove the new behavior
    external_asset: { owner: N/A, missing: N/A, recovery: N/A }
    ```

## 2. Red: Prove the Missing Contracts

- [x] 2.1 Add the smallest `checks/verify_contract.py` assertions for the intended one-level reference, module-base identity, four observation classes, DRAFT-admission-versus-promotion separation, all-condition post-implementation promotion Gate and local-main/next-module activation; run them against the legacy Skill and record the first decisive non-zero Red caused by the missing self-evolution contract.
  - Evidence (Red, 2026-08-13):
    ```yaml
    change: enable-run-loop-self-evolution
    gate_type: W1
    ui_level_target: UI0
    ui_level_actual: UI0
    base_sha: 2209c071a21860231827b2a8c8c81d9b7745e6e1
    candidate_sha: not-yet-created
    phase: red
    command_or_action: /usr/bin/python3 openspec/changes/enable-run-loop-self-evolution/checks/verify_contract.py --repo .
    exit_result: exit 1
    sanitized_summary: legacy package had SKILL.md and agents/openai.yaml but lacked the required references/self-evolution.md
    artifact_or_environment: local writer worktree against frozen legacy runner
    unverified_boundary: Red proves the target contract is absent; it does not prove implementation or independent behavior
    external_asset: { owner: N/A, missing: N/A, recovery: N/A }
    ```
- [x] 2.2 Add `checks/checker_contract.py`, seven focused positive/negative fixtures and `checks/run_checker_regressions.py`; run the focused suite before production rules exist and record the first decisive Red without deleting a test, loosening an assertion, using `|| true` or changing expected production behavior.
  - Evidence (Red, 2026-08-13):
    ```yaml
    change: enable-run-loop-self-evolution
    gate_type: W1
    ui_level_target: UI0
    ui_level_actual: UI0
    base_sha: 2209c071a21860231827b2a8c8c81d9b7745e6e1
    candidate_sha: not-yet-created
    phase: red
    command_or_action: /usr/bin/python3 openspec/changes/enable-run-loop-self-evolution/checks/run_checker_regressions.py
    exit_result: exit 1
    sanitized_summary: first positive fixture failed because zero_match_count was not implemented; all seven positive/negative fixture pairs were already fixed
    artifact_or_environment: change-local standard-library checker surface
    unverified_boundary: Red does not prove any checker contract is implemented
    external_asset: { owner: N/A, missing: N/A, recovery: N/A }
    ```
- [x] 2.3 Add the minimal `forward-test.md` scenario and `checks/verify_forward_test.py` schema validator covering base capture, observation-only behavior, all four classes, rejection of an incomplete DRAFT screen, complete-screen admission without promotion, local-main-gated next-module adoption and unchanged stage routing; record the validator's expected Red on a legacy-runner result. Do not claim writer execution as independent evidence.
  - Evidence (Red, 2026-08-13):
    ```yaml
    change: enable-run-loop-self-evolution
    gate_type: W1
    ui_level_target: UI0
    ui_level_actual: UI0
    base_sha: 2209c071a21860231827b2a8c8c81d9b7745e6e1
    candidate_sha: not-yet-created
    phase: red
    command_or_action: /usr/bin/python3 openspec/changes/enable-run-loop-self-evolution/checks/verify_forward_test.py --candidate-sha 2209c071a21860231827b2a8c8c81d9b7745e6e1 checks/fixtures/legacy-forward-result.json
    exit_result: exit 1
    sanitized_summary: validator rejected the legacy result because it changed the current module rule
    artifact_or_environment: deterministic negative validator fixture; no fresh agent session was run
    unverified_boundary: writer validates only the schema's failure path; exact-SHA independent fresh-session evidence remains verifier-only
    external_asset: { owner: N/A, missing: N/A, recovery: N/A }
    ```

## 3. Green: Upgrade the Thin Runner

- [x] 3.1 Add only `.agents/skills/order-run-loop/references/self-evolution.md` with the identity/checkpoint fields, immutable class definitions, five-field DRAFT admission screen, distinct post-implementation promotion formula, local-main/next-module activation, seven checker contracts and retrospective/next-Goal queue; keep the detail one reference deep and do not copy stage procedures.
  - Evidence (Green, 2026-08-13):
    ```yaml
    change: enable-run-loop-self-evolution
    gate_type: W1
    ui_level_target: UI0
    ui_level_actual: UI0
    base_sha: 2209c071a21860231827b2a8c8c81d9b7745e6e1
    candidate_sha: not-yet-created
    phase: green
    command_or_action: /usr/bin/python3 openspec/changes/enable-run-loop-self-evolution/checks/verify_contract.py --repo .
    exit_result: PASS
    sanitized_summary: the only new Skill resource is one-level references/self-evolution.md and every frozen evolution token is present
    artifact_or_environment: .agents/skills/order-run-loop/references/self-evolution.md
    unverified_boundary: writer structural evidence is not the independent forward-test
    external_asset: { owner: N/A, missing: N/A, recovery: N/A }
    ```
- [x] 3.2 Make the minimum `SKILL.md` edit that captures the module base before work, keeps module execution observation-only, reads the single reference at the module boundary, applies integrated rules only to later modules and preserves the exact existing handler/lifecycle/lane/retry/readiness/hard-Gate behavior.
  - Evidence (Green, 2026-08-13):
    ```yaml
    change: enable-run-loop-self-evolution
    gate_type: W1
    ui_level_target: UI0
    ui_level_actual: UI0
    base_sha: 2209c071a21860231827b2a8c8c81d9b7745e6e1
    candidate_sha: not-yet-created
    phase: green
    command_or_action: verify_contract.py legacy transition, route, lane, retry, BLOCKED_EXTERNAL, score and hard-Gate assertions
    exit_result: PASS
    sanitized_summary: SKILL.md gained one 11-line thin entry and retained every legacy control invariant
    artifact_or_environment: .agents/skills/order-run-loop/SKILL.md at 132 lines
    unverified_boundary: activation remains inactive until exact-SHA verification and local-main integration
    external_asset: { owner: N/A, missing: N/A, recovery: N/A }
    ```
- [x] 3.3 Keep `agents/openai.yaml` unchanged unless the metadata checker proves an actual inconsistency; verify directory name, frontmatter `name`, display metadata and quoted `default_prompt` agree and the prompt still explicitly contains `$order-run-loop`.
  - Evidence (Green, 2026-08-13):
    ```yaml
    change: enable-run-loop-self-evolution
    gate_type: W1
    ui_level_target: UI0
    ui_level_actual: UI0
    base_sha: 2209c071a21860231827b2a8c8c81d9b7745e6e1
    candidate_sha: not-yet-created
    phase: green
    command_or_action: git diff --exit-code 2209c071a21860231827b2a8c8c81d9b7745e6e1 -- .agents/skills/order-run-loop/agents/openai.yaml; verify_contract.py --repo .
    exit_result: PASS
    sanitized_summary: agents/openai.yaml stayed byte-unchanged; directory/frontmatter name and quoted default prompt remained consistent
    artifact_or_environment: .agents/skills/order-run-loop/agents/openai.yaml
    unverified_boundary: quick_validate and full writer Gate are recorded later
    external_asset: { owner: N/A, missing: N/A, recovery: N/A }
    ```
- [x] 3.4 Complete the change-local contract/checker implementation using only local standard tooling; run the same focused commands from 2.1-2.3 and record Green, including each checker's expected non-zero negative case.
  - Evidence (Green, 2026-08-13):
    ```yaml
    change: enable-run-loop-self-evolution
    gate_type: W1
    ui_level_target: UI0
    ui_level_actual: UI0
    base_sha: 2209c071a21860231827b2a8c8c81d9b7745e6e1
    candidate_sha: not-yet-created
    phase: green
    command_or_action: verify_contract.py; run_checker_regressions.py; verify_forward_test.py with valid fixture; rerun validator with legacy fixture and explicitly require exit 1
    exit_result: PASS
    sanitized_summary: contract PASS; checker positive=7 negative=7; forward validator positive exit 0 and negative exit 1
    artifact_or_environment: local standard-library tests; no independent fresh session executed
    unverified_boundary: valid JSON fixture proves validator behavior only; exact-SHA fresh-session evidence remains verifier-only
    external_asset: { owner: N/A, missing: N/A, recovery: N/A }
    ```

## 4. Refactor and Writer Gate

- [x] 4.1 Remove duplication and placeholders; verify `SKILL.md` stays thin, `references/` has only `self-evolution.md`, there are no nested references/scripts/assets/README in the Skill, and none of the full headings or procedures from the four `$order-*` stage skills is copied.
  - Evidence (Refactor, 2026-08-13):
    ```yaml
    change: enable-run-loop-self-evolution
    gate_type: W1
    ui_level_target: UI0
    ui_level_actual: UI0
    base_sha: 2209c071a21860231827b2a8c8c81d9b7745e6e1
    candidate_sha: not-yet-created
    phase: refactor
    command_or_action: enumerate exact Skill files; enforce SKILL<=160/reference<=200 lines; explicitly require no copied stage headings, shell=True, false-green construct or placeholder; rerun verify_contract.py
    exit_result: PASS
    sanitized_summary: files=3, SKILL=132 lines, reference=73 lines, copied headings=0, forbidden checker constructs=0, placeholders=0
    artifact_or_environment: .agents/skills/order-run-loop/** and change-local checks
    unverified_boundary: line/heading checks do not replace behavioral regression or independent verification
    external_asset: { owner: N/A, missing: N/A, recovery: N/A }
    ```
- [x] 4.2 Run and record PASS for `/usr/bin/python3 /Users/vivix/.codex/skills/.system/skill-creator/scripts/quick_validate.py .agents/skills/order-run-loop`, `/usr/bin/python3 openspec/changes/enable-run-loop-self-evolution/checks/verify_contract.py --repo .`, and `/usr/bin/python3 openspec/changes/enable-run-loop-self-evolution/checks/run_checker_regressions.py`.
  - Evidence (Writer, 2026-08-13):
    ```yaml
    change: enable-run-loop-self-evolution
    gate_type: W1
    ui_level_target: UI0
    ui_level_actual: UI0
    base_sha: 2209c071a21860231827b2a8c8c81d9b7745e6e1
    candidate_sha: not-yet-created
    phase: writer
    command_or_action: /usr/bin/python3 quick_validate.py .agents/skills/order-run-loop; verify_contract.py --repo .; run_checker_regressions.py
    exit_result: PASS
    sanitized_summary: Skill is valid; contract PASS; checker positive=7 negative=7
    artifact_or_environment: local writer worktree
    unverified_boundary: exact-SHA verifier and fresh-session forward-test have not run
    external_asset: { owner: N/A, missing: N/A, recovery: N/A }
    ```
- [x] 4.3 Run `openspec validate enable-run-loop-self-evolution --strict`, `git diff --check 2209c071a21860231827b2a8c8c81d9b7745e6e1...HEAD`, and an owned-path audit that fails for every path outside `openspec/changes/enable-run-loop-self-evolution/**` and `.agents/skills/order-run-loop/**`; separately verify root governance, four stage skills and business code are unchanged.
  - Evidence (Writer, 2026-08-13):
    ```yaml
    change: enable-run-loop-self-evolution
    gate_type: W1
    ui_level_target: UI0
    ui_level_actual: UI0
    base_sha: 2209c071a21860231827b2a8c8c81d9b7745e6e1
    candidate_sha: not-yet-created
    phase: writer
    command_or_action: strict; tracked/untracked whitespace and owned-path audit; unchanged root/stage/business check; gofmt; Go test/race/vet/build/smoke; app JS syntax and JSON parse
    exit_result: PASS
    sanitized_summary: strict valid; 16 working paths owned; root/stage/business unchanged; Go test/race/vet/build and smoke PASS; JS check PASS; JSON OK 42
    artifact_or_environment: local repository workflow/quality Gate
    unverified_boundary: precommit audit must be rerun on the staged diff and final committed SHA
    external_asset: { owner: N/A, missing: N/A, recovery: N/A }
    ```
- [x] 4.4 Bind C/T/V/R to actual evidence: C to delta/metadata/legacy-route/non-weakening checks, T to Red-Green-Refactor plus seven positive/negative checker regressions, V to a committed exact-SHA clean verification package (writer maximum 8), and R to frozen-base recovery, bounded failure, next-module activation and rollback checks. Require every dimension `>= 8`, total `>= 36`, all mandatory Gates run and hard blockers `= 0`; do not pre-award planned evidence.
  - Evidence (Writer score, 2026-08-13):
    ```yaml
    change: enable-run-loop-self-evolution
    gate_type: W1
    ui_level_target: UI0
    ui_level_actual: UI0
    base_sha: 2209c071a21860231827b2a8c8c81d9b7745e6e1
    candidate_sha: emitted by git rev-parse HEAD immediately after the commit containing this record
    phase: writer
    command_or_action: bind strict/contract/metadata/routes to C; three real Red plus checker and validator positive/negative regressions to T; final owned clean commit to V; frozen identity, rejection paths, bounded recovery and next-module activation checks to R
    exit_result: PASS after candidate commit and clean-status evidence
    sanitized_summary: C=9, T=10, V=8, R=9, total=36; every dimension >=8; hard blockers=0
    artifact_or_environment: complete candidate tree plus post-commit exact-SHA/clean handoff
    unverified_boundary: V=8 is the writer ceiling; independent forward-test, V>=9, integration and activation remain unproven
    external_asset: { owner: N/A, missing: N/A, recovery: N/A }
    ```
- [x] 4.5 Commit only the two owned path sets, capture the full candidate SHA and clean status in the proactive handoff without editing candidate artifacts afterward, and move only to `CANDIDATE`; do not push, create/update PR, deploy or write any external system.
  - Candidate evidence (2026-08-13): this record is part of the single owned-path candidate commit. Immediately after commit, the writer runs `git rev-parse HEAD` and `git status --porcelain`, sends the exact SHA and clean result to the main Goal, and makes no further artifact edit; a failure in commit or clean-status check invalidates this checked task and the CANDIDATE claim.

## 5. Exact-SHA Independent Verification

- [ ] 5.1 Route the full candidate SHA to a different verifier session; create a newly clean detached worktree at exactly that SHA, confirm exact HEAD and clean status, and rerun tasks 4.1-4.4 plus every declared acceptance command without modifying files.
- [ ] 5.2 In that verifier session, execute `forward-test.md` with only root `AGENTS.md`, `SKILL.md`, `references/self-evolution.md` and the minimal fixture as context; store output only in a validated narrow temporary directory, run `verify_forward_test.py`, confirm the detached repository remains clean, and report PASS/FAIL for the exact SHA with unverified boundaries.
- [ ] 5.3 On the first verifier failure, return the decisive command/error to this same writer; after any repair, artifact edit or SHA change, create a new candidate and make the verifier rebuild a clean detached worktree and rerun all checks. Stop the lane on the third consecutive identical fingerprint and do not attempt a fourth blind retry.

## 6. Authorized Pure-FF Integration and Archive

- [ ] 6.1 Only after exact-SHA PASS and explicit integration authorization, verify current main contains all dependencies and can reach the candidate by pure fast-forward; if main moved such that update/rebase/merge is needed, return to the writer for a new candidate and full re-verification rather than integrating the old attestation.
- [ ] 6.2 Fast-forward local main without push/PR/deploy/external write, record candidate and integrated SHA separately, and rerun strict, quick validation, metadata/route/non-weakening and checker regressions on integrated main.
- [ ] 6.3 Only after integrated checks PASS, run the authorized OpenSpec archive flow; audit the deterministic lifecycle output so it contains only the date-stamped archive plus the expected canonical `loop-engineering-control-plane` delta, validate the archive strictly, commit the archive result, and record archive SHA and clean main.
- [ ] 6.4 Produce a module retrospective containing all four observation classes, DRAFT-screen decisions and evidence gaps; queue DRAFT-admissible lessons for a later Goal/module only, do not call them promoted, and do not reopen or modify the Skill again within this Goal.
