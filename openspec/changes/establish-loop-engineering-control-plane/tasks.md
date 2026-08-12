## 1. Approval Gate and Red Evidence

- [x] 1.1 Before editing `.agents/skills/order-run-loop/**`, record the main Agent's explicit approval, confirm the change is `APPROVED`, verify dependencies are satisfied, and capture `git branch --show-current`, `git rev-parse HEAD`, and `git status --short`; stop if the branch/worktree or owned paths do not match this change.
  - Evidence (2026-08-12): main Goal `019ff620-bb1e-7702-b305-b1dd7c6651ca` explicitly approved; branch `codex/establish-loop-engineering-control-plane`, HEAD `a698443bbfc885be9bcfd50b6e95cacd718ef01b`, clean status, dependency baseline reachable, and strict validation PASS.
- [x] 1.2 Establish structural Red evidence with `test -f .agents/skills/order-run-loop/SKILL.md && test -f .agents/skills/order-run-loop/agents/openai.yaml`; record the expected non-zero result before initialization rather than creating either file by hand.
  - Red (2026-08-12): the exact command returned exit 1 because the approved skill package did not yet exist; the first observable Red moved the change to `IMPLEMENTING`.

## 2. Green: Generate the Thin Control Plane

- [x] 2.1 Initialize the skill with `python3 /Users/vivix/.codex/skills/.system/skill-creator/scripts/init_skill.py order-run-loop --path .agents/skills --interface 'display_name=Order Run Loop' --interface 'short_description=有界调度 OpenSpec change 并以证据闭环长任务' --interface 'default_prompt=Use $order-run-loop to coordinate this goal through bounded OpenSpec change lanes.'`; do not pass `--resources`, and record the created paths.
  - Green (2026-08-12): `init_skill.py` returned success and created only `.agents/skills/order-run-loop/SKILL.md` plus `agents/openai.yaml`; no resource directory was requested.
- [x] 2.2 Replace the generated `SKILL.md` template with an imperative thin router that names Addy Osmani's 2026-06-07 Loop Engineering as the umbrella, keeps Ralph/ReAct/Reflexion distinct, and delegates exactly to `$order-plan-change`, `$order-implement-tdd`, `$order-verify-change`, and `$order-integrate-change` without copying their procedures.
  - Green (2026-08-12): source-lineage and four literal handler checks PASS; the skill states that change-local procedures remain in the four existing skills.
- [x] 2.3 Add the decision-ready lifecycle table with entry condition, unique handler, and exit evidence for every `DRAFT → APPROVED → IMPLEMENTING → CANDIDATE → INDEPENDENT_VERIFIED → INTEGRATED → ARCHIVED` transition, including exact-SHA invalidation rules.
  - Green (2026-08-12): all seven states, seven transitions, handler column, required exit evidence, and post-change attestation invalidation are present and machine-searchable.
- [x] 2.4 Add the fixed control policy: main Goal persistence; main plus two lane slots; one writer per lane; writer session/worktree reuse through CANDIDATE; verifier only for exact SHA and reused after repair with a rebuilt clean detached worktree; conflict serialization; highest-risk smallest P0/P1 scheduler; `BLOCKED_EXTERNAL`; third identical error fingerprint stop; compact proactive handoff; non-blocking snapshot; human-escalation boundary.
  - Green (2026-08-12): focused checks PASS for the lane ledger fields, upstream `INTEGRATED`-main Gate, conflict serialization, two-writer limit, exact-SHA verifier, third-failure stop, external block, snapshot, handoff, and escalation terms.
- [x] 2.5 Add the seven-dimension 100-point scoring table and the exact stop formula `Score >= 85 AND OPEN(P0/P1) = 0 AND first_blocking_openspec_strict = PASS`; state that hard blockers override score and planning artifacts do not satisfy implementation acceptance.
  - Green (2026-08-12): parsed rubric weight sum is 100; exact stop formula and baseline/readiness-versus-implementation separation checks PASS.
- [x] 2.6 Set `agents/openai.yaml` to quoted `interface.display_name`, `interface.short_description`, and `interface.default_prompt` values generated in task 2.1; confirm the default prompt explicitly contains `$order-run-loop`.
  - Green (2026-08-12): all three YAML strings are quoted, short description length is 30 characters, and the default prompt contains `$order-run-loop`.

## 3. Refactor: Keep One Thin Source

- [x] 3.1 Remove generated placeholders and any duplicated change-level instructions; verify the new skill does not contain the existing full headings `Plan an Order Change`, `Implement an Order Change with TDD`, `Verify an Order Change`, or `Integrate an Order Change`.
  - Refactor (2026-08-12): no TODO/template resource text or existing full skill heading remains; routing stays at handler and evidence-Gate level.
- [x] 3.2 Confirm `.agents/skills/order-run-loop/` contains only `SKILL.md` and `agents/openai.yaml`, the SKILL frontmatter contains only `name` and a trigger-oriented `description`, and the full file remains imperative and under 500 lines.
  - Refactor (2026-08-12): package file count is 2, frontmatter fields are exactly `name,description`, trigger phrase check PASS, and `SKILL.md` is 121 lines.

## 4. Local Verification and Candidate

- [x] 4.1 Run `python3 /Users/vivix/.codex/skills/.system/skill-creator/scripts/quick_validate.py .agents/skills/order-run-loop` and record PASS.
  - Validation (2026-08-12): the default `python3` and bundled runtime first exposed `ModuleNotFoundError: yaml`; without installing anything, `/usr/bin/python3 .../quick_validate.py .agents/skills/order-run-loop` returned `Skill is valid!` using the existing user-site PyYAML.
- [x] 4.2 Verify all four literal skill references are present once or more, all seven lifecycle states and weights `25,15,15,15,15,10,5` are present and total 100, and the lifecycle/scoring decision tables, two-lane limit, exact-SHA verifier Gate, third-failure stop, `BLOCKED_EXTERNAL`, 10-line handoff, snapshot, escalation, and stop rules are machine-searchable in `SKILL.md`; record the commands and PASS results.
  - Validation (2026-08-12): structural shell checks returned `package_files=2`, `line_count=121`, `short_description_chars=30`, `weights_sum=100`, and `focused_contracts=PASS`.
- [x] 4.3 Run `openspec validate establish-loop-engineering-control-plane --strict`, `git diff --check`, and an owned-path audit from the approved baseline; fail if any changed path is outside `openspec/changes/establish-loop-engineering-control-plane/**` or `.agents/skills/order-run-loop/**`.
  - Validation (2026-08-12): strict, quick validation, working-tree owned-path audit, diff check, and four-handler/no-full-heading-copy checks all returned PASS.
- [x] 4.4 Commit only the owned paths, record the full candidate SHA and clean `git status --short`, move the change to `CANDIDATE`, and proactively return no more than 10 lines to the main Goal with conclusion, SHA, validation, self-assessment, blocker, and next action.
  - Candidate evidence (2026-08-12): this task record is part of the candidate commit; its self-referential full SHA and clean status are emitted by `git rev-parse HEAD` plus `git status --short` after commit and sent proactively to the main Goal.

## 5. Independent Verification and Integration

- [ ] 5.1 Invoke `$order-verify-change` only after task 4.4 yields a full candidate SHA; verify that exact SHA in another clean detached worktree by rerunning tasks 4.1-4.3 and the four-skill/no-copy checks. On failure, return to the same writer session; after a new SHA, reuse the verifier session but rebuild its clean detached worktree and rerun all checks.
- [ ] 5.2 After exact-SHA PASS, dependencies and required review are satisfied, invoke `$order-integrate-change` only with explicit integration authorization; record candidate and integrated SHA, rerun required integration checks, and archive only after the change is actually integrated into main.
