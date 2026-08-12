---
name: order-implement-tdd
description: Implement one approved OpenSpec change in the order repository with Red Green Refactor evidence. Use when starting or continuing implementation in the change owner's worktree after proposal, design, specs, tasks, ownership, dependencies, and acceptance checks are complete.
---

# Implement an Order Change with TDD

## Confirm the implementation context

1. Read root `AGENTS.md`, `docs/quality/change-quality-gates.md`, and every context file returned by `openspec instructions apply --change <name> --json`.
2. Confirm the current branch/worktree belongs to this change and edits stay within its owned paths.
3. Confirm the approved change declares its single highest `gate_type`, target `ui_level`, external assets, dependencies and acceptance commands.
4. Run `openspec validate <name> --strict`. Stop if artifacts are incomplete or a dependency blocks implementation.

## Execute Red → Green → Refactor

For each behavior-sized task:

1. **Red**: add or select the smallest acceptance check that demonstrates the missing behavior. Run it and record the decisive failure in `tasks.md`.
2. **Green**: implement only enough to satisfy that requirement. Run the focused check and record the pass.
3. **Refactor**: improve structure without changing the contract, then rerun the focused check and affected regression checks.
4. Mark the task `[x]` only after the recorded evidence passes.

For docs, config, migrations and file moves, use meaningful link, schema, migration, structure or content-integrity checks instead of artificial unit tests.

Use the minimum RGR and sanitized evidence template for the declared class in `docs/quality/change-quality-gates.md`. If a required database, WeChat, payment, device or other external asset is unavailable, record `BLOCKED_EXTERNAL`; never report the missing Gate as PASS.

## Handle discoveries

- If implementation requires behavior outside the spec, update the OpenSpec artifacts first; obtain confirmation when public behavior, data, authorization or acceptance changes.
- If another change owns a required path, stop writing that path and declare a dependency or reassign ownership.
- Preserve unrelated worktree changes and do not perform adjacent refactors.

## Produce a candidate

1. Run the writer Gate in `docs/quality/change-quality-gates.md`, all change-local acceptance checks and `openspec validate <name> --strict`.
2. Reject every hard blocker; require C/T/V/R total at least 36 and every dimension at least 8. The writer's V score cannot claim independent PASS.
3. Review the diff against owned paths and commit only this change.
4. Record the candidate SHA and hand it to `order-verify-change`.

Implementation self-test creates CANDIDATE, not INDEPENDENT_VERIFIED. Do not claim independent verification from the writer worktree.
