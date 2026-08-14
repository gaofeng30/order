## Why

The archived `persist-archived-lifecycle-receipt` contract requires a post-archive bootstrap binding, but the production loader still hard-codes exactly three historical bindings and rejects the required fourth entry as if it were premature candidate self-binding. This leaves the already archived control-plane change unable to complete its binding-head and receipt-head closure.

## What Changes

- Preserve the candidate-era rule: three historical bindings are valid and `lifecycle-receipt-control-v1` remains unbound before the repair is archived.
- Promote one exact repair candidate `C` only after both the executable minimal-context fresh-session Gate and ordinary clean-detached exact-SHA independent verification pass for `C`, and the main Gate has read current exact-SHA verifier handoffs for those results.
- Make candidate `C` own `checks/verify_archive.py` and `checks/expected-canonical-loop-engineering-control-plane-spec.md`. The latter is the complete expected canonical target blob, reviewed and exercised by both exact-`C` verifier recipes; it is never generated at `A` and never derived from mutable CLI output. Trust repair archive `A` only when the dispatcher executes the checker blob from exact `C`, that checker loads its validator runner blob from exact `C`, and those trusted bytes prove `A` has sole parent `C`, every active-change blob moved once to one dated archive at the same relative path with `R100`, the only other path is the canonical spec `M`, the canonical blob at `A` is byte-for-byte equal to the fixture blob at `C`, protected registry/wrapper/executor/judge blobs are identical at `C` and `A`, and the worktree is clean at `HEAD=A`. The checker and runner copies in `A` are compared subjects only and never judge authority. Any merge, omitted/mutated move, extra path/byte, source drift, wrong HEAD, or checker failure is `NO-GO`.
- After trusted `A`, allow at most one fourth binding only for `lifecycle-receipt-control-v1` targeting exact candidate `d0b70a077bcaa64c401837eb0e9b6f27035210a0`.
- Admit that binding only when its fixed definition/hash, argv/environment/expected results, wrapper blob, and repaired executor blob match the controlled sources at `C` and `A`, current controlled sources remain byte-identical, and Git ancestry/path history proves one later binding-only append commit.
- Fail closed for premature, duplicate, additional, wrong-target, wrong-hash/blob/source, altered-profile, or receipt-supplied-command cases. The loader derives only Git/byte/mechanical results; it never accepts repository text, an audit pointer, author identity, or mechanical replay as actor/session-independence proof.
- Require the main Gate to read the current final verifier handoff using exact `threadId` plus `hostId`. A stored `wait_cursor` is typed only as `wait_threads.afterCursor`; it is an optional wait/audit optimization, not a proof input, and MUST NOT be reused as `read_thread`'s distinct pagination cursor. A wait cursor that is unavailable across callers does not make a final unreadable. If `read_thread(threadId, hostId)` cannot read the final, or the final is stale or SHA/result-mismatched, the Gate is `NO-GO` and requires a new ordinary exact-SHA verification; it MUST NOT recover PASS from checkpoint/tasks/receipt text.
- Keep this change's lifecycle separate from Goal0 delivery: this repair may reach `ARCHIVED` at `A`, but Goal0 remains `NO-GO` until an authorized integrator appends the exact binding, a different clean-detached verifier passes exact binding-head `B`, the integrator later adds the current receipt, and another clean-detached verifier derives exact receipt-head `R`.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `loop-engineering-control-plane`: make the one-time bootstrap binding admission stage-aware while preserving all candidate, independent-verification, integration, archive, binding-head, and receipt-head Gates.

## Impact

Owner: independent planner/writer thread `019ffe92-0854-74f2-89b9-918c78f85efe` (`hostId=local`) is the sole DRAFT writer and, only after explicit approval, the sole implementation writer in this same branch/worktree.

Base: clean local `main@44aea3b3f892acf8a6e0537f65f28f33de22cf2f`.

Writer-owned paths:

- `openspec/changes/allow-post-archive-bootstrap-binding/**`
- `openspec/changes/allow-post-archive-bootstrap-binding/checks/verify_archive.py` (explicit candidate-owned read-only archive Gate)
- `openspec/changes/allow-post-archive-bootstrap-binding/checks/expected-canonical-loop-engineering-control-plane-spec.md` (explicit candidate-owned full expected canonical bytes)
- `tools/lifecycle-receipts/profile_runner.py`
- `tools/lifecycle-receipts/tests/test_post_archive_bootstrap_binding.py`

Candidate read-only contracts: `tools/lifecycle-receipts/mechanical-profiles-v1.json`, `mechanical-bindings-v1.json`, `receipt-schema-v1.json`, `profiles/lifecycle_receipt_control.py`, canonical `openspec/specs/loop-engineering-control-plane/spec.md`, and archived `persist-archived-lifecycle-receipt`. Schema and registries need no candidate edit because the definition already exists and the binding remains a later Goal0 artifact.

Authorized archive-only derived paths, never candidate-writer paths: the byte-identical dated move of `openspec/changes/allow-post-archive-bootstrap-binding/**`, including both candidate-owned check artifacts, and the sole canonical `openspec/specs/loop-engineering-control-plane/spec.md` modification whose resulting blob equals the expected fixture at `C`. Goal0-only paths after this change is archived: `tools/lifecycle-receipts/mechanical-bindings-v1.json` and `openspec/changes/archive/2026-08-13-persist-archived-lifecycle-receipt/lifecycle-receipt.md`.

Dependency: archived `persist-archived-lifecycle-receipt@44aea3b3f892acf8a6e0537f65f28f33de22cf2f`, its exact candidate target `d0b70a077bcaa64c401837eb0e9b6f27035210a0`, and unchanged controlled profile/wrapper bytes. Its archived checkpoint still says `CANDIDATE`/`independent_verification=NOT_RUN` and tasks 5.x are unchecked, so repository history cannot supply independent PASS. Attempt 1 remains `FAIL`. The main Gate read no-parallel attempt 2 final using thread `019ffeab-ca4c-71f3-aa90-d15283219361` and `hostId=local`; its audit-only `wait_cursor` (`wait_threads.afterCursor`) was `aef7a56e-afd6-4f74-b5a3-3a5a7e0f8440:80`. The final reports exact `d0b70a077bcaa64c401837eb0e9b6f27035210a0`, state `INDEPENDENT_VERIFIED`, all required Gates PASS, `C9/T10/V9/R9=37`, hard blockers `0`. This current handoff satisfies only the main dependency Gate; neither it nor its wait cursor is loader mechanical proof. Fresh reviewer `019ffed1-b59e-7883-867d-2c0316bfb7ef` later reported `APPROVABLE` with P0/P1/P2 zero, and Main Gate explicitly approved implementation; candidate independence remains a separate NOT_RUN Gate.

`gate_type`: `W1`. `ui_level_target`: `UI0`; `ui_level_actual`: `UI0`; UI1+ is N/A, not PASS. No UI, secret, account, or network asset is required. Historical dependency attempt 2 is a current readable main-Gate PASS pointer; attempt 1 FAIL remains immutable, and neither pointer changes loader/mechanical semantics.

Non-goals: no current bootstrap binding or receipt commit, no binding-head or receipt-head PASS, no Goal0 completion claim, no arbitrary future profile admission, no schema/registry/Skill/state/score/product change, no archive/canonical mutation by `verify_archive.py`, no A-time fixture generation or CLI-output trust, no push/PR/deploy/external write, and no touch or cleanup of legacy worktrees or temp residuals.

Minimum success: current exact loader yields a meaningful Red for an otherwise valid fourth post-archive binding; the same local-Git fixtures become Green only for exact `C→A→B`, including byte equality between `A` canonical and the expected fixture at `C`, and reject canonical extra/missing/reordered bytes, A-controlled checker/runner authority, smuggled judge/path changes, wrong parent/rename/HEAD, or any source violation while existing three-binding behavior remains Green. Both candidate verifier recipes run the archive-checker fixture tests. At exact `A`, the literal-SHA launcher executes checker bytes from `C`; that checker executes runner-validator bytes from `C`, exits `0`, and writes exactly `archive-gate=PASS` plus one LF only after HEAD/clean/postflight conditions hold. Loader output remains mechanical only. That completes this change only; Goal0 remains `NO-GO` until the separately tracked `B` and `R` recipes and live handoffs close in order.
