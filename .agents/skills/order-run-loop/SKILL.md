---
name: order-run-loop
description: Coordinate long-running order goals through bounded OpenSpec change lanes, evidence-gated states, readiness scoring, failure routing, and deterministic stopping. Use when a main Agent must progress multiple order changes without session sprawl or false completion.
---

# Run the Order Goal Loop

## Use one method lineage

Use Addy Osmani's 2026-06-07 [Loop Engineering](https://addyosmani.com/blog/loop-engineering/) as the umbrella method. Treat [Ralph](https://ghuntley.com/ralph/) as an earlier simple execution loop and [ReAct](https://arxiv.org/abs/2210.03629) plus [Reflexion](https://arxiv.org/abs/2303.11366) as antecedent mechanisms, not aliases or parallel protocols. These sources were accessed on 2026-08-12.

## Keep the control plane thin

Read root `AGENTS.md` and use it as the governance source. Route change-local work to exactly one existing handler:

- Route planning and OpenSpec refinement to `$order-plan-change`.
- Route approved implementation through candidate creation to `$order-implement-tdd`.
- Route read-only verification of an exact candidate SHA to `$order-verify-change`.
- Route authorized integration and archive handling to `$order-integrate-change`.

Do not reproduce, replace, or weaken those skills' procedures. Keep only cross-change selection, ledger, evidence Gate, retry, score, and stop decisions here.

## Freeze runner evolution by module

Read the [self-evolution protocol](references/self-evolution.md) before allocating each module.

- Freeze the runner base before each module and persist its repository SHA, Skill blob, content SHA256, and explicit version state in the repository checkpoint.
- Keep runner evolution observation-only during the module. Append exactly classified observations without changing the active Skill, reference, Gates, or routes.
- Screen observation candidates only at a module boundary. Treat admission to a dedicated OpenSpec `DRAFT` as planning, never as promotion.
- Promote only an implemented exact candidate that passes every non-weakening, regression, clean-detached fresh-session, and independent-verification Gate in the protocol.
- Activate an integrated rule only for the next module after local main contains it. Never switch the runner of an active module.
- Queue admissible observations for a later Goal when the current Goal does not authorize another runner change.

## Maintain one persistent control ledger

Keep the main Goal session alive. Maintain one row per change lane and update it only from repository evidence or an active handoff:

| lane | threadId | hostId | worktree | change | state | SHA | cursor | error_fingerprint | repeat_count | dependency | owned_paths | blocker | next |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | ---: | --- | --- | --- | --- |

Use a full immutable SHA only in `SHA`; write `none` before a candidate exists. Store the latest consumed cursor so snapshots are incremental. Derive `error_fingerprint` from the failed command, exit code, first decisive error, SHA, and environment identifier.

Keep a separate Goal checkpoint with the readiness evidence, current score, OPEN P0/P1 list, blocking OpenSpec strict results, `BLOCKED_EXTERNAL` entries, and last scheduling decision.

## Enforce lane and module Gates

- Allow at most two active change lane slots in addition to the main Goal session. Allow only one writer per lane and at most two writer lanes.
- Reuse the same writer session and worktree from `DRAFT` through `CANDIDATE`; queue a third eligible change instead of opening another lane.
- Start a dependency module only after every declared upstream dependency is `INTEGRATED` in main. Treat a candidate SHA or independent PASS outside main as insufficient.
- Run two changes concurrently only when they have no dependency, no public-contract conflict, and no overlapping owned paths. Serialize them when any condition fails.
- Start an independent verifier only for a committed full candidate SHA. Reject verification for exploration, `DRAFT`, `APPROVED`, `IMPLEMENTING`, a dirty tree, or a moving ref.
- Let the verifier consume the same lane slot while the writer is idle. After a failed candidate, return to the existing writer session; reuse the verifier session for the new SHA but rebuild a clean detached worktree and rerun all checks.

## Advance only with evidence

Use this decision table; never advance from an Agent's unsupported status claim:

| Transition | Enter only when | Handler | Required exit evidence |
| --- | --- | --- | --- |
| queue → `DRAFT` | one highest-risk blocker is independently understandable, acceptable, and reversible | `$order-plan-change` | proposal, design, specs, tasks, declared ownership/dependencies, strict PASS |
| `DRAFT` → `APPROVED` | DRAFT exit evidence is current | main Agent Gate | explicit approval plus current strict PASS |
| `APPROVED` → `IMPLEMENTING` | dependencies are `INTEGRATED` in main; branch/worktree/owner/owned paths match | `$order-implement-tdd` | first observable Red evidence |
| `IMPLEMENTING` → `CANDIDATE` | change-local tasks, acceptance, strict, and owned-path audit PASS | `$order-implement-tdd` | committed full candidate SHA, decisive command results, clean status |
| `CANDIDATE` → `INDEPENDENT_VERIFIED` | exact full SHA is available | `$order-verify-change` | PASS attestation from another clean detached worktree for that exact SHA |
| `INDEPENDENT_VERIFIED` → `INTEGRATED` | dependencies/review are satisfied and integration is authorized | `$order-integrate-change` | actual main integrated SHA plus integration checks |
| `INTEGRATED` → `ARCHIVED` | delivery evidence is complete on main | `$order-integrate-change` | archive validation PASS and archive fact |

Invalidate independent verification after any code, spec, task, rebase, merge, or SHA change.

## Close archived recovery with one receipt

For modules whose frozen runner base includes this rule, keep candidate checkpoint/tasks immutable after exact verification. After the existing authorized integration and archive Gates pass, append exactly one `lifecycle-receipt.md` in the actual dated archive directory and verify it with `tools/lifecycle-receipts/verify_receipt.py`. A receipt stores audit-only attestation plus `mechanical_verification=REQUIRED_DERIVED` and `receipt_head_verification=REQUIRED_DERIVED`; only the repository-controlled exact-SHA profile registry may derive current mechanical evidence, and it never proves actor independence or an `INDEPENDENT_VERIFIED` state. Treat a missing, invalid, unbound, later-edited or falsely upgraded receipt as `NO-GO`.

Do not let a business module define or edit its own judge. Its separately owned control-plane profile must already be independently verified, integrated and archived in the frozen runner base. After the ordinary candidate/integration/archive Gates, require integrator-only exact binding, a different clean-detached binding-head PASS, the later receipt commit, and another clean-detached receipt-head derivation before allocating the next module. Read detailed trust, replay, bootstrap and supersession rules in the self-evolution reference; receipt closure adds no lifecycle state and replaces none of the existing verifier, integration, archive, quality or authorization Gates.

## Select the next smallest blocker

Run this scheduling algorithm whenever a slot becomes free:

1. Collect unresolved P0/P1 items; exclude active items, unmet dependencies, and conflicts with active owned paths or public contracts.
2. Mark an item `BLOCKED_EXTERNAL` when it requires unavailable real-world facts, qualifications, accounts, secrets, or authority. Record its owner, missing evidence, and recovery condition; never record it as PASS or let it block unrelated local work.
3. Sort eligible items by severity descending, downstream dependencies unlocked descending, independently acceptable boundary size ascending, then stable change name ascending.
4. Select the first item and reduce it to the smallest change that remains independently understandable, acceptable, and reversible. Plan an upstream change first if this is impossible.
5. Allocate only an empty lane. Serialize any dependency, public-contract conflict, or owned-path overlap.

Do not let P2 work or ordinary optimization displace an executable P0/P1.

## Run bounded checkpoints

1. Consume proactive lane handoffs and take cursor-based non-blocking snapshots with `timeoutMs: 0`.
2. Refresh the lane ledger, error counts, readiness evidence, hard blockers, and eligible queue.
3. Apply the scheduling algorithm and route each lane by its evidence-backed state.
4. Continue local selection, planning, or evidence review while useful work exists. Use one bounded wait only when no other planning is executable; do not poll unchanged state.
5. Persist the checkpoint after every transition, block, failure, approval, candidate SHA, verification result, and integration result.
6. Evaluate the relevant stop condition; otherwise begin the next smallest blocker.

## Stop repeated failures and escalate narrowly

On a real failure, record its fingerprint and apply only a minimal targeted correction. Reset the consecutive count when the fingerprint changes. On the third consecutive occurrence of the same fingerprint, stop the lane, retain its checkpoint, return all three observations plus one recommended decision, and do not make a fourth blind attempt.

Escalate to a human only for unavailable real-world facts/qualifications/secrets, irreversible authorization, or three consecutive identical fingerprints. Let the main Agent decide ordinary product or technical questions from confirmed PRD, OpenSpec, public contracts, and repository governance.

## Score readiness without overriding blockers

Award points only for traceable decision, artifact, or command evidence. Cap each category at its fixed weight and do not double-count evidence:

| Readiness category | Weight |
| --- | ---: |
| Product decisions and scope | 25 |
| Cross-client truth source and state | 15 |
| Money, inventory, and authorization | 15 |
| API and executable acceptance | 15 |
| Architecture, data, and recovery | 15 |
| Quality Gates and independent verification | 10 |
| External dependency governance | 5 |

Calculate `Score = sum(evidence-backed category points)`, capped at 100. Declare the readiness Goal ready only when:

`Score >= 85 AND OPEN(P0/P1) = 0 AND first_blocking_openspec_strict = PASS`

Return `NO-GO` when any hard Gate fails, regardless of score. Keep baseline/readiness Goals separate from implementation Goals: completed documents or strict PASS improve readiness but never prove business code complete. Stop an implementation Goal only from that change's acceptance, candidate, independent verification, and integration evidence.

## Hand off proactively

At completion, blockage, or a decision boundary, call `send_message_to_thread` using the main Goal's `threadId` and `hostId`. Send no more than 10 lines in this shape:

```text
change: <name>
state: <evidence-backed state>
conclusion: <single verdict>
SHA: <full SHA or none>
validation: <decisive PASS/FAIL/BLOCKED evidence>
self_assessment: <score and reason>
blocker: <none or one blocker>
next: <one action or decision>
```

Do not wait for the main Goal to poll the lane.
