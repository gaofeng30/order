# Ticket: implement-staff-identity-snapshot-core

## Fixed point

- change: `implement-staff-identity-snapshot-core`
- status: `CANDIDATE_READY_FOR_EXTERNAL_SHA_HANDOFF`
- authoritative candidate comparison base/source: `19ca1e46e106f293070f0cdf820951e31107cba6` (authoritative staging fast-forward)
- historical development base: `f3c4efa4cd665652d93d5da76f92d18c4bdc59ac`; the 38 granular Red/Green tree receipts were genuinely produced there and are not rewritten after the base advance
- writer branch: `codex/implement-staff-identity-snapshot-core`
- writer worktree: `/Users/vivix/.codex/worktrees/9391/order`
- candidate SHA status: `not-yet-created` (machine evidence enum: `NOT_CREATED`); the next owned-only commit forms the candidate and the controller binds its immutable full SHA externally after commit
- gate: `W3`; UI target/actual: `UI0` / `UI0`
- governance: `GOVERNANCE_PENDING`; `docs/agents/issue-tracker.md` is absent. This change does not fabricate or initialize tracker state.

## Goal and success

Create a backend-only, pure in-process staff identity snapshot Module at the frozen public seam:

```go
func Resolve(input Input) (Snapshot, error)
```

The Module validates one complete, versioned candidate view and resolves `VISITOR` or `STAFF` without I/O, global state, time, configuration, persistence, routing, pricing, ordering, or payment behavior. Success requires the exact contract in `spec.md`, stable redacted errors, zero-Snapshot fail-closed behavior, Red -> Green -> Refactor evidence, at least twelve mutation kills with an infrastructure shield, all declared writer Gates, two-axis review, and clean detached exact-SHA verification.

## Ownership and dependencies

Owned paths only:

- `.scratch/implement-staff-identity-snapshot-core/**`
- `services/api/internal/staffidentity/**`
- `go.mod`, limited to moving existing `golang.org/x/text v0.34.0` from indirect to direct

`go.sum` and every other path are read-only, especially `services/api/internal/{identity,merchantidentity,quotepricing,httpapi}/**`, `cmd/**`, `migrations/**`, `apps/**`, router, and main wiring.

Runtime dependency: existing `golang.org/x/text v0.34.0` only. There is no predecessor change dependency. Future callers own obtaining a complete consistent whitelist view and deciding when to invoke the Module. Tracker initialization is a governance dependency, not a business implementation blocker.

## Non-goals and unverified boundary

No DB, migration, repository, HTTP/interface adapter, router, WeChat operation, user binding, whitelist administration/import, pricing, P5 discount activation, quote, order, payment, clock, environment lookup, global state, I/O, UI, push, PR, deployment, integration, or external-system write. Passing this change proves pure identity-decision semantics only; it does not prove whitelist persistence/API, price application, order/payment snapshots, or a production flow.

## Lifecycle

- The first uncommitted staged attempt is `INVALIDATED_PRE_CANDIDATE` by parallel writer Standards/Spec review; it has no candidate and all its receipts are void.
- The second uncommitted replacement is `INVALIDATED_PRE_CANDIDATE`; its grouped tracers, unsupported score, and non-replayable command records cannot support a candidate.
- Every broad Gate, mutation, review, or verification receipt created before the current implementation/spec/tasks/tests freeze is `INVALIDATED_NOT_CURRENT` after those artifacts changed. The later old-base post-freeze static/evidence/mutation-self-test observations are separately `INVALIDATED_BY_BASE_ADVANCE`.
- The 38 immutable granular Red/Green receipts honestly retain development base `f3c4efa4cd665652d93d5da76f92d18c4bdc59ac`. They prove only historical public-seam behavior of this owned pure Module. Pickup integration writes completely non-overlapping paths and are not claimed by those receipts.
- The current third replacement has 11 additional exact Red/Green tree pairs for primary validation, Extra validation, NFC, whitelist version, and entry validation; `SI-03` is complete. Unstaged intermediate state is never described as immutable evidence.
- NFC completed a real implementation Red/Green. The not-NFKC negative contract is only a post-Green reversible `norm.NFC` to `norm.NFKC` mutation sensitivity check; it is not retroactively labelled an implementation Red.
- Four later tests for primary-error priority, Extra-error priority, immutability/determinism, and same-primary-Extra behavior are post-implementation regression/mutation anchors. They passed focused on the current implementation but have no historical implementation Red.
- Historical old-base observation only, now `INVALIDATED_BY_BASE_ADVANCE`: the positional mutation self-test rejected missing/duplicate source with `79` and simulated Go exit `2` with `82`, with `real_mutations=NOT_RUN`; the isolated `extra_phone_only_authorizes` mutant exited `1` with the named `TestResolveExtraNameMismatchIsVisitor` FAIL marker. These earn no current Gate credit.
- Authoritative-base Writer behavior Gates passed on the receipt-before-docs staged source snapshot `b7ad5a82ae5aa5f1bdf1ef5eaab1759b04d392bc`. The initial mutation run is retained as FAIL, not PASS; after the minimum Gate-only mutant replacement correction, the self-test, isolated mutant, and full 17/17 shield passed. All business dynamic receipts bind that tree.
- First two-axis pre-review result is `INVALIDATED_AFTER_FINDING`: Standards found one P1 for the missing `phase: refactor` ledger evidence while Spec found zero issues. Neither result supports a candidate.
- The frozen implementation at staged tree `f5daa28f883317494842a2f236981557366ab4ec` then passed identical Refactor focused `count=1`, determinism `count=100`, and race `count=20` reruns. This fixed the finding's evidence gap but did not itself create a replacement pre-review result, so both Standards and Spec axes were required to rerun.
- Replacement writer pre-review restarted from zero on frozen staged tree `b0846d513d8d7ec181231e3c01cf654ff77879d6` and passed Standards=`0 findings`, `0/12 smells`, Spec=`0 findings`, with unchanged start/end HEAD/tree and zero unstaged/untracked paths. It is not formal exact-candidate review.
- Writer candidate-ready score is `C9/T10/V8/R9=36`. `V8` becomes applicable only when the next owned-only commit yields an exact SHA and the controller binds it to the complete clean-detached package awaiting verifier; it does not mean independent PASS. Formal exact-candidate dual review, fresh detached full-Gate verification, integration, and archive remain pending.
- This governance-only candidate-ready delta changes the tree after pre-review. The accepted external-SHA handoff permits the next commit to form the candidate without self-reference, but no old review or independent result transfers to that SHA; formal review and detached verification start from zero.
- Freeze order is implementation/spec/tasks/tests complete -> invalidate earlier broad Gates -> user stages the exact owned path set once -> staged/unstaged/untracked post-freeze Gate -> remaining Writer Gates in the order recorded in `tasks.md` -> Standards/Spec pre-review.
- Pre-review is forbidden until the latest freeze is fully staged and all post-freeze Writer Gates pass.
- The next owned-only commit containing this governance update creates the candidate; its full SHA is recorded only by immutable external handoff after commit, never guessed inside the commit.
- Standards and Spec reviews compare `19ca1e46e106f293070f0cdf820951e31107cba6...<candidate>` independently.
- Verifier uses another fresh clean detached worktree at the exact candidate and makes no source edits.
- Any implementation, ticket, spec, tasks, Gate, base, dependency, or SHA change invalidates all prior review/verification receipts.
- Integration is explicitly out of scope and requires separate authorization.
