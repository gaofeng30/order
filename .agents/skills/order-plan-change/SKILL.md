---
name: order-plan-change
description: Create or refine one fine-grained OpenSpec change for the order repository. Use when planning a product capability, API, architecture change, workflow change, migration, or deployment change before implementation, especially when work must be split across parallel worktrees without overlapping ownership.
---

# Plan an Order Change

## Establish the boundary

1. Read the root `AGENTS.md` and the minimum relevant PRD, code, contract, and existing OpenSpec files.
2. State one primary outcome and one acceptance verdict. If parts can be accepted or rolled back independently, create separate changes.
3. Declare:
   - owner and branch/worktree;
   - owned paths and shared contracts;
   - dependencies on other changes;
   - non-goals;
   - executable acceptance checks.
4. Ask one concise question with a recommended answer only when ambiguity changes behavior, contract, data, authorization, or acceptance. Keep unresolved recommendations out of confirmed requirements.

## Build the OpenSpec artifacts

1. Use a verb-led kebab-case change name.
2. Run `openspec new change <name>`.
3. Follow `openspec status --change <name> --json` and `openspec instructions <artifact> --change <name>`.
4. Create artifacts in dependency order:
   - `proposal.md`: why, one outcome, non-goals, capability, impact, ownership and dependencies;
   - `specs/<capability>/spec.md`: testable requirements and success/failure scenarios;
   - `design.md`: one selected implementation, boundaries, trade-offs and invalidation conditions;
   - `tasks.md`: Red, Green, Refactor, local verification and independent verification.
5. Run `openspec validate <name> --strict`.

Do not enter implementation while a behavior-changing decision is unresolved. Mark the change APPROVED only when the four artifacts are complete, internally consistent and executable by a writer without guessing.

## Preserve parallelism

- Allow unrelated changes to proceed in other worktrees.
- Assign one writer to each owned path at a time.
- Express real prerequisites as dependencies; do not create a global phase gate.
- Split shared foundational work into its own change instead of duplicating it across feature changes.
