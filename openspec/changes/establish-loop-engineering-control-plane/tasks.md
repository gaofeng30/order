## 1. Approval Gate and Red Evidence

- [ ] 1.1 Before editing `.agents/skills/order-run-loop/**`, record the main Agent's explicit approval, confirm the change is `APPROVED`, verify dependencies are satisfied, and capture `git branch --show-current`, `git rev-parse HEAD`, and `git status --short`; stop if the branch/worktree or owned paths do not match this change.
- [ ] 1.2 Establish structural Red evidence with `test -f .agents/skills/order-run-loop/SKILL.md && test -f .agents/skills/order-run-loop/agents/openai.yaml`; record the expected non-zero result before initialization rather than creating either file by hand.

## 2. Green: Generate the Thin Control Plane

- [ ] 2.1 Initialize the skill with `python3 /Users/vivix/.codex/skills/.system/skill-creator/scripts/init_skill.py order-run-loop --path .agents/skills --interface 'display_name=Order Run Loop' --interface 'short_description=有界调度 OpenSpec change 并以证据闭环长任务' --interface 'default_prompt=Use $order-run-loop to coordinate this goal through bounded OpenSpec change lanes.'`; do not pass `--resources`, and record the created paths.
- [ ] 2.2 Replace the generated `SKILL.md` template with an imperative thin router that names Addy Osmani's 2026-06-07 Loop Engineering as the umbrella, keeps Ralph/ReAct/Reflexion distinct, and delegates exactly to `$order-plan-change`, `$order-implement-tdd`, `$order-verify-change`, and `$order-integrate-change` without copying their procedures.
- [ ] 2.3 Add the decision-ready lifecycle table with entry condition, unique handler, and exit evidence for every `DRAFT → APPROVED → IMPLEMENTING → CANDIDATE → INDEPENDENT_VERIFIED → INTEGRATED → ARCHIVED` transition, including exact-SHA invalidation rules.
- [ ] 2.4 Add the fixed control policy: main Goal persistence; main plus two lane slots; one writer per lane; writer session/worktree reuse through CANDIDATE; verifier only for exact SHA and reused after repair with a rebuilt clean detached worktree; conflict serialization; highest-risk smallest P0/P1 scheduler; `BLOCKED_EXTERNAL`; third identical error fingerprint stop; compact proactive handoff; non-blocking snapshot; human-escalation boundary.
- [ ] 2.5 Add the seven-dimension 100-point scoring table and the exact stop formula `Score >= 85 AND OPEN(P0/P1) = 0 AND first_blocking_openspec_strict = PASS`; state that hard blockers override score and planning artifacts do not satisfy implementation acceptance.
- [ ] 2.6 Set `agents/openai.yaml` to quoted `interface.display_name`, `interface.short_description`, and `interface.default_prompt` values generated in task 2.1; confirm the default prompt explicitly contains `$order-run-loop`.

## 3. Refactor: Keep One Thin Source

- [ ] 3.1 Remove generated placeholders and any duplicated change-level instructions; verify the new skill does not contain the existing full headings `Plan an Order Change`, `Implement an Order Change with TDD`, `Verify an Order Change`, or `Integrate an Order Change`.
- [ ] 3.2 Confirm `.agents/skills/order-run-loop/` contains only `SKILL.md` and `agents/openai.yaml`, the SKILL frontmatter contains only `name` and a trigger-oriented `description`, and the full file remains imperative and under 500 lines.

## 4. Local Verification and Candidate

- [ ] 4.1 Run `python3 /Users/vivix/.codex/skills/.system/skill-creator/scripts/quick_validate.py .agents/skills/order-run-loop` and record PASS.
- [ ] 4.2 Verify all four literal skill references are present once or more, all seven lifecycle states and weights `25,15,15,15,15,10,5` are present and total 100, and the lifecycle/scoring decision tables, two-lane limit, exact-SHA verifier Gate, third-failure stop, `BLOCKED_EXTERNAL`, 10-line handoff, snapshot, escalation, and stop rules are machine-searchable in `SKILL.md`; record the commands and PASS results.
- [ ] 4.3 Run `openspec validate establish-loop-engineering-control-plane --strict`, `git diff --check`, and an owned-path audit from the approved baseline; fail if any changed path is outside `openspec/changes/establish-loop-engineering-control-plane/**` or `.agents/skills/order-run-loop/**`.
- [ ] 4.4 Commit only the owned paths, record the full candidate SHA and clean `git status --short`, move the change to `CANDIDATE`, and proactively return no more than 10 lines to the main Goal with conclusion, SHA, validation, self-assessment, blocker, and next action.

## 5. Independent Verification and Integration

- [ ] 5.1 Invoke `$order-verify-change` only after task 4.4 yields a full candidate SHA; verify that exact SHA in another clean detached worktree by rerunning tasks 4.1-4.3 and the four-skill/no-copy checks. On failure, return to the same writer session; after a new SHA, reuse the verifier session but rebuild its clean detached worktree and rerun all checks.
- [ ] 5.2 After exact-SHA PASS, dependencies and required review are satisfied, invoke `$order-integrate-change` only with explicit integration authorization; record candidate and integrated SHA, rerun required integration checks, and archive only after the change is actually integrated into main.
