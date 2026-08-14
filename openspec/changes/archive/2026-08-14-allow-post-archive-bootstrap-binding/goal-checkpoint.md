# Goal Checkpoint

## Current Runtime

| field | value |
|---|---|
| change | `allow-post-archive-bootstrap-binding` |
| state | `CANDIDATE`; replacement exact SHA is returned in the writer handoff |
| owner | independent planner/writer thread `019ffe92-0854-74f2-89b9-918c78f85efe` (`hostId=local`) |
| lane | sole implementation writer in the approved same session/worktree |
| branch | `codex/allow-post-archive-bootstrap-binding` |
| worktree | `/Users/vivix/.codex/worktrees/order-allow-post-archive-bootstrap-binding.Plan` |
| repo_base_sha | `44aea3b3f892acf8a6e0537f65f28f33de22cf2f` |
| candidate_sha | `the single replacement commit containing this checkpoint; literal full SHA is returned in the writer handoff because a commit cannot contain its own object ID` |
| rejected_candidate_sha | `77bdf711575c390e5bc56b2d5eaf3a8f1e6caa64`; independent FAIL; object remains readable |
| integrated_sha | `none` |
| archive_sha | `none` |
| binding_head_sha | `none` |
| receipt_head_sha | `none` |
| dependency | `persist-archived-lifecycle-receipt archive 44aea3b3f892acf8a6e0537f65f28f33de22cf2f; exact candidate d0b70a077bcaa64c401837eb0e9b6f27035210a0; current attempt 2 handoff PASS` |
| blocker | `both independent exact-candidate verifier Gates are NOT_RUN for the replacement SHA` |
| next_action | `dispatch the replacement literal SHA to fresh minimal-context and ordinary clean-detached verifier runs; no old-SHA result is reusable` |
| approval | `APPROVED by Main Gate; fresh reviewer thread 019ffed1-b59e-7883-867d-2c0316bfb7ef concluded APPROVABLE with P0/P1/P2=0` |
| gate_type | `W1` |
| ui_level_target | `UI0` |
| ui_level_actual | `UI0`; no visual surface; UI1+ `N/A`, not PASS |

## Frozen Runner Base

| field | value |
|---|---|
| runner_repo_sha | `44aea3b3f892acf8a6e0537f65f28f33de22cf2f` |
| runner_skill_git_blob | `0f41f64ad87fc9fd410cb916b4d1562aee92e42f` |
| runner_skill_sha256 | `2781bdda1544106e30e7483c4b500d611df85c79753cc1bd3b717d91d1edaac8` |
| runner_version | `unversioned` |

## Boundary

Candidate writer ownership is only this change directory, including exactly two new Gate outputs `checks/verify_archive.py` and `checks/expected-canonical-loop-engineering-control-plane-spec.md`, plus `tools/lifecycle-receipts/profile_runner.py` and new `tools/lifecycle-receipts/tests/test_post_archive_bootstrap_binding.py`. Registry, bindings, schema, wrapper, Skill, canonical spec, archives, receipts, products, remote operations, dirty worktrees, and temp residuals are not candidate-writer paths.

The candidate keeps exactly three bindings. It changes only admission logic and tests, receives ordinary independent exact-SHA verification, and does not close the original binding/receipt chain.

The authorized archive integrator alone may create the byte-identical dated move and canonical modification after both candidate verifier Gates have reviewed/tested the complete expected canonical fixture at exact `C`. At exact `A`, the fixed direct-argv bootstrap must execute checker bytes from `C`; that checker must execute runner-validator bytes from `C`. The `A` copies are compared subjects only. These trusted bytes prove the resulting canonical blob equals the fixture byte-for-byte; they never generate/mutate canonical bytes or trust mutable CLI output. Later Goal0 binding/receipt writes are external delivery dependencies, not outputs of this change.

The loader/profile/receipt checker derives only repository Git/byte/mechanical results. Actor/session independence belongs only to the main Gate reading a current verifier thread handoff. Repository text and the audit pointer below are never mechanical proof; if the handoff cannot be read, the state is `NO-GO` and exact-SHA verification must rerun.

## Historical Dependency Independent Handoff

| field | value |
|---|---|
| attempt | `1` |
| status | `FAIL`; do not upgrade partial command PASS |
| client_thread_id | `client-new-thread:affb30c1-6f26-4b3a-8f1c-f0bf3912112e` |
| thread_id | `019ffea5-5368-7970-9afb-1309a0cb6a4d` |
| host_id | `local` |
| wait_cursor (`wait_threads.afterCursor`) | `not supplied; FAIL remains decisive and cannot satisfy the Gate` |
| exact_sha | `d0b70a077bcaa64c401837eb0e9b6f27035210a0` |
| result | `FAIL`; five `__pycache__/*.pyc` made the worktree dirty, receipt `--list` exit `1`, later Gates `NOT_RUN` |
| failure_fingerprint | `dependency-verifier|dirty-worktree|pycache-created-by-parallel-process|d0b70a|cbf4`; count `1` |
| verifier_worktree | `/Users/vivix/.codex/worktrees/cbf4/order` |
| trust | `audit pointer only; this FAIL forces a new exact-SHA verification` |

| attempt 2 field | value |
|---|---|
| attempt | `2` |
| status | `PASS`; current readable handoff, audit pointer only |
| thread_id | `019ffeab-ca4c-71f3-aa90-d15283219361` |
| host_id | `local` |
| wait_cursor (`wait_threads.afterCursor`) | `aef7a56e-afd6-4f74-b5a3-3a5a7e0f8440:80` |
| exact_sha | `d0b70a077bcaa64c401837eb0e9b6f27035210a0` |
| result | `INDEPENDENT_VERIFIED`; all required Gates PASS; `C9/T10/V9/R9=37`; hard blockers `0` |
| verifier_worktree | `/Users/vivix/.codex/worktrees/7ebd/order` |
| trust | `main-Gate audit pointer only; read final with read_thread(threadId, hostId); wait cursor is not read_thread pagination and is not loader/mechanical proof` |

Repository contradiction requiring this rerun: archived dependency checkpoint records `state=CANDIDATE` and `independent_verification=NOT_RUN`; archived tasks 5.1-5.3 are unchecked. Neither archive presence nor current loader output may upgrade those bytes to actor-independent PASS.

## DRAFT Admission

| required field | status | evidence |
|---|---|---|
| reproducible evidence | `PRESENT` | `OBS-CHECK-001`: exact loader rejection at base; focused Red task 2.1 |
| generalizable or safety-critical rationale | `PRESENT` | `OBS-CAND-001`: post-archive binding must not admit an unverified archive payload or self-attestation |
| explicit non-weakening intent | `PRESENT` | three historical bindings remain immutable; loader remains mechanical-only; every candidate/integration/archive/binding/receipt Gate remains mandatory |
| executable regression plan | `PRESENT` | tasks 2.1-4.2 specify loader plus archive-checker positive/negative, exact canonical fixture, full regression, strict and scope checks |
| executable minimal-context forward-test plan | `PRESENT` | task 5.1 fixes literal full-SHA input, verifier-created worktree, archive-checker fixture cases, exits/stdout and ending-clean result |

This table is the immutable historical DRAFT admission record. Main Gate approval is recorded above; it does not itself prove candidate verification, integration, archive, binding, receipt, or Goal0 promotion.

## Observation Ledger

| source_id | class | observation | source_link | state |
|---|---|---|---|---|
| `OBS-CHECK-001` | `checker` | valid post-archive fourth binding is rejected as candidate self-binding | `44aea3b3f892acf8a6e0537f65f28f33de22cf2f:tools/lifecycle-receipts/profile_runner.py#L254-L256` | reproducible |
| `OBS-EXTERNAL-001` | `external` | archived dependency contains no repository-independent PASS evidence | `44aea3b3f892acf8a6e0537f65f28f33de22cf2f:openspec/changes/archive/2026-08-13-persist-archived-lifecycle-receipt/goal-checkpoint.md#L10-L19` and `tasks.md#L57-L61` | open; rerun required |
| `OBS-EXTERNAL-002` | `external` | exact-`d0b70a...` verifier ended FAIL; focused PASS cannot compensate for dirty ending state | `codex-thread:019ffea5-5368-7970-9afb-1309a0cb6a4d?hostId=local` | FAIL; fresh rerun required |
| `OBS-ENV-001` | `environment` | parallel validation processes wrote five `__pycache__/*.pyc`; receipt `--list` failed closed exit `1`; later Gates `NOT_RUN` | `codex-thread:019ffea5-5368-7970-9afb-1309a0cb6a4d?hostId=local` | immutable source; do not reclassify |
| `OBS-EXTERNAL-003` | `external` | no-parallel attempt 2 independently verified exact `d0b70a...` with all required Gates PASS | `codex-thread:019ffeab-ca4c-71f3-aa90-d15283219361?hostId=local&afterCursor=aef7a56e-afd6-4f74-b5a3-3a5a7e0f8440:80` | PASS audit pointer; `afterCursor` is wait-only, not loader/mechanical proof |
| `OBS-CHECK-002` | `checker` | stored pointers and symbolic shell SHA recipes could create false recoverability | `codex-thread:019ffe87-8810-7441-ab7e-b155ec5ce19f; fresh rereview P1` | addressed and approved; exact-candidate verification remains NOT_RUN |
| `OBS-CHECK-003` | `checker` | parent/R100/name-status/strict could not prove `A` canonical bytes equal an independently reviewed exact target at `C` | `codex-thread:019ffe87-8810-7441-ab7e-b155ec5ce19f; archive Gate rereview P1` | addressed by candidate fixture plus read-only archived checker; exact-candidate verification remains NOT_RUN |
| `OBS-CHECK-004` | `checker` | untyped cursor text conflated `wait_threads.afterCursor` with `read_thread` pagination | `codex-thread:019ffe87-8810-7441-ab7e-b155ec5ce19f; archive Gate rereview P2` | typed, separated, and approved |
| `OBS-EXTERNAL-004` | `external` | fresh independent DRAFT review found P0/P1/P2 zero and all declared planning audits PASS | `codex-thread:019ffed1-b59e-7883-867d-2c0316bfb7ef?hostId=local` | Main Gate APPROVED; implementation authorized |
| `OBS-CAND-001` | `candidate` | derive only Git/byte/mechanical admission, pin complete expected canonical bytes at `C`, and require current finals in main Gate | `openspec/changes/allow-post-archive-bootstrap-binding/specs/loop-engineering-control-plane/spec.md` | implemented and writer-verified; independence NOT_RUN |
| `OBS-CAND-002` | `candidate` | exact post-archive fourth binding and archive-byte Gate became Green while candidate bindings stayed three/unbound | `tools/lifecycle-receipts/tests/test_post_archive_bootstrap_binding.py` | writer focused `23/23` and full receipt `50/50` PASS; exact-SHA verification NOT_RUN |
| `OBS-EXTERNAL-005` | `external` | ordinary verifier proved the A worktree could replace the runner validator yet make the archived checker return exact PASS | `codex-thread:019ffef3-ac44-7c22-9c4a-062a6f2dbcc4?hostId=local` | exact `77bdf711575c390e5bc56b2d5eaf3a8f1e6caa64` FAIL; all prior declarative PASS cannot compensate |
| `OBS-CAND-003` | `candidate` | trusted-C loader executes the checker and runner validator from candidate Git blobs while A copies remain comparison subjects | `tools/lifecycle-receipts/tests/test_post_archive_bootstrap_binding.py` and `checks/verify_archive.py` | corrective authority Red reproduced; focused `25/25` and full `52/52` writer PASS; independent replacement verification NOT_RUN |

## Immutable SHA and Archive Trust Gate

| symbol | meaning | current value |
|---|---|---|
| `C` | exact repair candidate awaiting minimal-context and ordinary independent PASS | `the single replacement commit containing this checkpoint; literal full SHA in writer handoff` |
| `A` | unique repair archive commit | `none` |
| `B` | later exact bootstrap binding-head | `none` |
| `R` | later current receipt-head | `none` |

Candidate `C` owns exactly two new Gate outputs: `checks/verify_archive.py` and the complete target bytes at `checks/expected-canonical-loop-engineering-control-plane-spec.md`. Both exact-`C` verifier recipes run the existing bootstrap test file's archive-checker fixture cases, including malicious `A` replacements of checker and runner authority; the fixture is not generated at `A` and mutable CLI output is never an expected-byte source.

Trusted `A` MUST satisfy the trusted-C direct-argv read-only Gate: the bootstrap loads `C:openspec/changes/allow-post-archive-bootstrap-binding/checks/verify_archive.py`, and that checker loads `C:tools/lifecycle-receipts/profile_runner.py`; no `A` checker/runner byte is executed. With literal full `C`/`A`, those bytes require `HEAD=A`; clean status; exactly one parent equal to `C`; one same-relative-path `R100` dated move for every active change blob including both Gate outputs and `.openspec.yaml`; only one additional changed path, `M openspec/specs/loop-engineering-control-plane/spec.md`; canonical blob at `A` byte-for-byte equal to the expected fixture blob at `C`; and profile/binding registry, wrapper, executor, receipt judge, plus frozen runner Skill/reference blobs identical at `C` and `A`. Success stdout is exactly `archive-gate=PASS` plus one LF. A-controlled authority, merge, intervening parent, incomplete/non-identical move, canonical extra/missing/reordered byte, extra path, judge/source drift, wrong HEAD, dirty status, or any other output is `NO-GO`.

The candidate minimal-context Gate uses a fresh session with only repository path and full `C`, exact detached/clean pre/post checks, and `python3 -m unittest tools/lifecycle-receipts/tests/test_post_archive_bootstrap_binding.py -v`, including the archive-checker positive plus A-controlled checker/runner, canonical-byte/judge/path/parent/rename/HEAD negatives. It uses local temporary Git fixtures only and no network or secret. A separate ordinary verifier reruns the same fixture cases plus all writer Gates at exact `C`; neither record substitutes for the other.

## Reproduction and Failure Ledger

An in-memory fourth binding with exact profile ID, target `d0b70a...`, current controlled definition hash, wrapper blob and executor blob reproduces `ProfileError: bootstrap profile must remain unbound in the candidate` against `44aea3b...`; no repository file was changed.

| fingerprint | count | status |
|---|---:|---|
| `bootstrap-binding-admission|semantic|valid-post-archive-fourth-treated-as-candidate-unbound|44aea3b|binding-head` | 2 | real focused Red reproduced after fixture precondition correction; recurrence at count 3 stops |
| `archive-gate|false-green|canonical-extra-byte-accepted|44aea3b|writer` | 1 | real focused Red; parent/R100/path-only scaffold returned `archive-gate=PASS` for extra canonical byte |
| `bootstrap-binding-admission|implementation|binding-registry-incorrectly-pinned-after-B|44aea3b|writer` | 1 | first Green attempt rejected the one authorized B-path change; fixed by excluding only bindings registry from A-to-HEAD protected comparison while retaining C-to-A equality |
| `archive-gate|content|fixture-extra-trailing-lf|44aea3b|writer` | 1 | content-integrity check found one extra terminal LF; removed only that record terminator |
| `writer-gate|environment|receipt-chain-requires-clean-precommit|44aea3b|writer` | 1 | expected fail-closed result in the dirty pre-commit writer worktree; the chain Gate must run in an equivalent clean preview and again at exact candidate |
| `writer-gate|environment|py-compile-created-three-pyc|44aea3b|writer` | 1 | direct syntax compilation created three enumerated pyc files; removed exactly those files and empty cache directories, then prohibited further py_compile use |
| `dependency-verifier|dirty-worktree|pycache-created-by-parallel-process|d0b70a|cbf4` | 1 | FAIL; five pyc files; receipt `--list` exit `1`; later Gates `NOT_RUN`; fresh no-parallel verifier required |
| `draft-rereview|P1|live-handoff-literal-sha-recipes-complete-checkpoint|44aea3b|planner` | 2 | second and final P1 revision; recurrence at count 3 triggers same-fingerprint stop |
| `draft-rereview|P1|archive-gate-missing-exact-canonical-byte-check|44aea3b|planner` | 1 | candidate fixture and archived checker added to DRAFT; fresh rereview required |
| `archive-gate|false-green|A-controlled-runner-overrides-validator|77bdf71|verifier` | 2 | independent task 5.2 plus writer focused Red; malicious sole-parent A replaced runner authority yet checker exited `0` with exact PASS stdout; recurrence at count 3 stops |
| `archive-gate|false-green|A-controlled-checker-overrides-validator|77bdf71|writer` | 1 | writer focused Red; malicious sole-parent A replaced the checker itself and returned exact PASS without judging any archive bytes |

## Replacement Writer Gate Evidence and Self-Assessment

| Gate | result |
|---|---|
| focused Red/Green | rejected candidate history preserved; corrective A-runner and A-checker authority Reds reproduced exact false PASS; trusted-C focused suite `25/25` PASS with no skip; materialized task 6.2 one-line bootstrap returned exact PASS on valid local `C→A` |
| receipt regression | full lifecycle receipt discovery `52/52` PASS; archived forward `7/7` PASS; runner contract PASS |
| historical derivation | exactly three historical profiles; two `MECHANICAL_PASS`, one `EXPECTED_MECHANICAL_FAIL`; every actor result `NOT_PROVEN_BY_MECHANICAL_REPLAY` |
| receipt chain | dirty pre-commit writer tree failed closed as required; equivalent local clean preview exited `0` and preserved historical `FAILED`, derived results, and non-persisted actor independence |
| repository/static | strict, tracked whitespace, owned/protected, bare-SHA-variable, sensitive/no-shell, gofmt, offline Go test/race/vet/build/smoke, JS syntax, and `JSON OK 44` all PASS |
| ending candidate content | only declared owned paths; exactly three bindings; bootstrap receipt absent; no registry/canonical/archive/Skill edit; no pyc/temp residual |

| dimension | writer score | basis |
|---|---:|---|
| C | 9 | approved exact stage contract now fixes both validator authority sources to candidate Git bytes without registry/archive/binding/receipt writes |
| T | 10 | real authority false-Green Reds, 25 focused cases, 52 receipt tests, archive/runner/historical/chain and offline repository Gates |
| V | 8 | complete replacement writer package only; both fresh exact-SHA independent finals remain NOT_RUN |
| R | 9 | trusted-C direct-argv bootstrap, exact byte/protected checks, safe single-parent replacement, no shell/network/secret/temp residue |
| total | 36 | every dimension at least 8; hard blockers `0` inside writer scope |

## Change Gate and Goal0 External Boundary

Four core artifacts plus checkpoint and strict validation received explicit Main Gate approval after fresh independent review. No UI, secret, account, or network asset is required. Historical attempt 1 remains `FAIL`; current readable attempt 2 final is main-Gate PASS. Its `wait_cursor` is only `wait_threads.afterCursor`; final readability comes from `read_thread(threadId, hostId)`, whose pagination cursor is separate. Ordinary verifier thread `019ffef3-ac44-7c22-9c4a-062a6f2dbcc4` rejected exact `77bdf711575c390e5bc56b2d5eaf3a8f1e6caa64`; that object and evidence remain immutable. The replacement returns to `CANDIDATE` only on the new single commit with sole parent `44aea3b3f892acf8a6e0537f65f28f33de22cf2f`; both old minimal-context PASS and old ordinary FAIL are invalid for the new SHA. Integration and archive remain `NOT_RUN`. The handoff never becomes loader/mechanical proof. UI0 is actual and UI1+ is N/A, not PASS. Push/PR/deploy and creation of the current binding or receipt are outside this change.

| Goal0 outer Gate | required role and exact input/output | current state |
|---|---|---|
| exact binding `B` | authorized integrator; descendant of trusted `A`; binding-only commit | `NOT_RUN` |
| binding-head verification | different clean-detached verifier at exact `B`; fixed profile JSON returns `MECHANICAL_PASS` and `NOT_PROVEN_BY_MECHANICAL_REPLAY` | `NOT_RUN` |
| later receipt `R` | integrator after binding PASS; one receipt-only add with both derived markers `REQUIRED_DERIVED` | `NOT_RUN` |
| receipt-head derivation | another clean-detached verifier at exact `R`; fixed receipt JSON returns `PASS_DERIVED`, persisted=false | `NOT_RUN` |

This repair may reach `ARCHIVED` at trusted `A`; that state never implies Goal0 completion. Until all four outer Gates pass in order with the required role separation, Goal0 verdict is `NO-GO`.
