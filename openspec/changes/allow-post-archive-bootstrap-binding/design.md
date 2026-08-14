## Context

At base `44aea3b3f892acf8a6e0537f65f28f33de22cf2f`, `mechanical-profiles-v1.json` already contains the fixed `lifecycle-receipt-control-v1` definition, while `mechanical-bindings-v1.json` correctly contains only the three historical candidate-era bindings. `profile_runner.py` rejects every fourth binding through `EXPECTED_PROFILE_TARGETS` and an exact-three set check, so the contractually required post-archive binding for target `d0b70a077bcaa64c401837eb0e9b6f27035210a0` cannot pass.

The archived dependency is not independent-verification evidence: its frozen checkpoint says `CANDIDATE` and `independent_verification=NOT_RUN`, and its tasks 5.1-5.3 remain unchecked. Attempt 1 thread `019ffea5-5368-7970-9afb-1309a0cb6a4d` remains `FAIL`. The main Gate read no-parallel attempt 2 final using thread `019ffeab-ca4c-71f3-aa90-d15283219361` and `hostId=local`; its audit-only `wait_cursor` (`wait_threads.afterCursor`) was `aef7a56e-afd6-4f74-b5a3-3a5a7e0f8440:80`. The final reports exact `d0b70a077bcaa64c401837eb0e9b6f27035210a0`, `INDEPENDENT_VERIFIED`, all required Gates PASS, `C9/T10/V9/R9=37`, hard blockers `0`. That live handoff supplies current main-Gate actor/session evidence only; it neither rewrites the frozen archive nor becomes loader mechanical proof.

Changing the executor also changes its Git blob. The three historical binding objects and their already appended receipt hashes must remain immutable, so the repair cannot rewrite them to point at the new executor. The loader must instead distinguish their frozen original-source evidence from the one repaired bootstrap source, without allowing a generic fourth profile.

This is W1/UI0 control-plane logic. No UI or external asset is involved.

## Goals / Non-Goals

**Goals:** preserve valid candidate-era three-binding behavior; make `C` own one read-only archive checker and one complete expected canonical-byte fixture; have both exact-`C` verifiers review/test those bytes; admit exactly one fixed bootstrap binding only after checker and runner-validator authority loaded from exact `C` prove `A` cannot carry unverified canonical or judge bytes; pin its complete controlled definition and repaired source; provide literal full-SHA verifier recipes; require the main Gate to read current exact-SHA thread finals for actor/session independence; and keep this change's `ARCHIVED` verdict distinct from Goal0 binding/receipt closure.

**Non-Goals:** arbitrary profile admission, rewriting historical bindings/receipts, creating the current binding or receipt, changing schema/registry/Skill/lifecycle/score/product behavior, generating or mutating canonical bytes at `A`, trusting mutable CLI output as archive evidence, actor identity proof, or touching legacy worktrees/residuals.

## Decisions

### 1. Keep two explicit allowed stages, never an arbitrary count

The loader accepts either exactly the existing three historical bindings, or those same three plus one `lifecycle-receipt-control-v1` binding. The bootstrap target is the fixed `d0b70a...` candidate and the profile object's canonical SHA256 is pinned to the already controlled definition. Duplicate, fifth, unknown, or mutated bindings fail before execution. `--all-historical` continues to execute only the three historical profiles.

This is selected over adding bootstrap to `EXPECTED_PROFILE_TARGETS`, which would make candidate self-binding legal and would silently change `--all-historical` semantics.

### 2. Derive an exact candidate-to-archive trust anchor from Git

Use four full immutable SHAs: repair candidate `C`, repair archive `A`, later bootstrap binding-head `B`, and later receipt-head `R`. The main Gate requires current independent handoffs for `C` before archive, but the loader does not and cannot prove that actor fact. For the four-binding form, the loader resolves exactly one dated archive directory ending `-allow-post-archive-bootstrap-binding` and exactly one `A`; `git rev-list --parents -n 1 A` MUST contain exactly `A C`, so merge archives, an intervening commit, or any other parent fail. `C` MUST be an ancestor of `A`, and `A` MUST be an ancestor of the current head.

Candidate `C` contains exactly two new Gate outputs below its change directory: `checks/verify_archive.py` and `checks/expected-canonical-loop-engineering-control-plane-spec.md`. The second file is the complete target bytes for `openspec/specs/loop-engineering-control-plane/spec.md` after archive. It is part of `C`, and both the minimal-context and ordinary exact-`C` verifiers run its fixture cases from the existing `tools/lifecycle-receipts/tests/test_post_archive_bootstrap_binding.py`; `A` never generates it and the Gate never trusts OpenSpec CLI output as the expected bytes.

At exact `A`, a fixed direct-argv bootstrap reads `checks/verify_archive.py` from the literal full `C` Git blob and executes it in memory; that checker accepts only `--repo`, literal full `--candidate`, and literal full `--archive` argv, then reads `profile_runner.py` from the same exact `C` Git object and executes its validator in memory. No checker or runner byte from the `A` worktree participates in judgment. The trusted candidate bytes check, in deterministic order: both argv are full object SHAs; repository `HEAD` equals `A` and status is clean; `git rev-list --parents -n 1 A` is exactly `A C`; every blob below the active change at `C`, including both Gate outputs and `.openspec.yaml`, appears exactly once as a same-relative-path `R100` into one dated archive; the only remaining row is `M openspec/specs/loop-engineering-control-plane/spec.md`; the canonical blob at `A` equals the expected fixture blob at `C` byte-for-byte; and the profile/binding registries, `profiles/lifecycle_receipt_control.py` wrapper, `profile_runner.py` executor, `verify_receipt.py` judge, plus the frozen run-loop Skill/reference blobs are identical at `C` and `A`. Thus the `A` checker and runner copies are only compared archive subjects. Success writes exactly `archive-gate=PASS\n`; failure exits `1` after the first decisive error.

Only after that Gate does the loader derive the bootstrap binding's unique first appearance at `B`. `B` MUST descend from `A`; its parent MUST contain the exact three-binding document; its own diff MUST change only `tools/lifecycle-receipts/mechanical-bindings-v1.json`; and that path MUST have no later edit. Current controlled registry, wrapper, executor, and judge blobs MUST equal their trusted `C`/`A` blobs. Missing, premature, same-stage, wrong-parent, merge, incomplete/non-identical move, fixture mismatch, extra canonical content, extra path, duplicated, ambiguous, source-drifted, or later-edited history fails at the first deterministic `ProfileError`.

This is selected over trusting a receipt stage string or current file presence, neither of which proves append order.

### 3. Pin source by its independently integrated generation

The three historical binding objects remain byte-identical and retain their original executor/tool/blob hashes. Candidate verification reviews the repaired executor through the ordinary exact-SHA verifier, not through those old receipts.

The fourth binding must match the fixed registry definition and target, and its `profile_definition_sha256`, tool path/blob, executor path/blob, argv, environment allowlist, expected exit/output, timeouts, network/write contract, and output cap must equal the bytes at both `C` and trusted `A`. The current HEAD copies of registry, wrapper, and executor must remain blob-identical to that source. Thus later replay uses the exact independently verified repaired executor without laundering or rewriting historical receipts.

This is selected over updating all historical binding hashes, which would invalidate append-only receipts, and over trusting whatever source happens to exist at HEAD.

### 4. Candidate promotion uses live handoffs outside the loader

During DRAFT through candidate verification, the bootstrap binding remains absent. Red/Green tests construct an isolated local Git history from the real base to exercise the valid `C→A→B` form and malicious archive variants, including A-controlled checker and runner replacements; production or historical files are not mutated. The fixture uses only repository bytes, `tempfile.TemporaryDirectory`, local Git argv with `shell=false`, and no network, credential, or external secret.

The dispatcher supplies repository, change name, and a literal full `C` as mandatory task input. Each verifier follows the `order-verify-change` isolation contract to create its own clean detached worktree; recipes never infer a SHA from a branch or set an unbound shell SHA variable. One fresh minimal-context session runs the focused loader and archive-checker fixtures, while a separate ordinary verifier reruns every writer Gate including the same archive-checker fixtures. Each handoff reports exact thread, host, optional typed `wait_cursor` (`wait_threads.afterCursor`), SHA, commands, result, and limitations. Any candidate/spec/task/command change invalidates both.

Only the main Gate may establish actor/session independence by calling `read_thread` with exact `threadId` and `hostId`, reading the current final, and matching its full SHA/result. A stored `wait_cursor` is only the `wait_threads.afterCursor` value; it may be unavailable across callers without affecting final readability, and it MUST NOT be passed as or conflated with `read_thread`'s separate pagination cursor. A checkpoint pointer is audit routing, not proof. If the final cannot be read, is stale, or has a different SHA/result, the main Gate returns `NO-GO` and dispatches fresh exact-SHA verification. It never infers PASS from archived checkpoint/tasks, receipt text, Git author, writer output, or loader/mechanical output. The same rule applies now to `d0b70a...` and later to `C`, `B`, and `R`.

This is selected over letting the writer's fixture result, a stored pointer, or one broad test label imply fresh-context recovery, lifecycle verification, or actor independence.

### 5. This change archives before, but never closes, Goal0

Authorized pure-FF integration and exact archive `A` may move this change to `ARCHIVED`. That verdict means only that the main Gate read current exact-`C` verifier handoffs and the loader repair/archive trust anchor passed. Goal0 remains `NO-GO` through four later delivery dependencies: the authorized integrator appends the exact fourth binding at `B`; a different verifier-created clean detached worktree at literal full `B` derives `MECHANICAL_PASS` and returns a live handoff; only then the integrator adds the current receipt in a later receipt-only commit `R`; another verifier at literal full `R` derives `PASS_DERIVED` and returns a separate live handoff without persisting it. Loader/profile/receipt output proves only the mechanical layer; the main Gate reads current handoffs for role/session separation.

The exact verifier inputs, commands, expected outputs, clean-state checks, and `NO-GO` conditions live in `tasks.md` and the checkpoint. These outer Goal0 Gates are delivery dependencies, not unfinished tasks that prevent this repair change from archiving.

## Risks / Trade-offs

- [History inspection becomes stricter] -> accept only the two exact stages and return the first deterministic `ProfileError` for ambiguity.
- [Archive could replace the checker/runner that judges itself] -> execute both authority sources from exact `C` Git bytes and treat their `A` copies only as protected comparison subjects.
- [Archive could smuggle unverified executor or canonical bytes] -> have both exact-`C` verifiers test the complete expected canonical fixture, then require the trusted-C checker/validator to prove sole parent `C`, complete `R100` move, exact fixture byte equality, protected blob equality, exact `HEAD=A`, clean status, and no other path.
- [Executor bytes change while old bindings are immutable] -> keep old objects frozen, independently verify this repair, and pin all post-archive execution to the identical `C`/`A` source.
- [A later legitimate executor change would fail] -> require its own separately planned control-plane change; do not add a generic escape hatch here.
- [Mechanical replay is mistaken for independent verification] -> preserve `NOT_PROVEN_BY_MECHANICAL_REPLAY` and the ordinary independent verifier Gate.
- [A stored wait cursor is stale or caller-local] -> type it only as `wait_threads.afterCursor`; read the final with `read_thread(threadId, hostId)` and rerun exact-SHA verification only if that final is unreadable or mismatched.

## Migration Plan

First require a currently readable exact-`d0b70a...` verifier PASS final. Implement with only three bindings plus the two candidate-owned Gate outputs; dispatch literal full `C` to both verifier recipes and require each to run the archive-checker fixtures; require the main Gate to read both current finals; pure-FF integrate; then archive only as single-parent `A` after a literal-SHA bootstrap executes checker bytes from exact `C`, that checker executes runner-validator bytes from exact `C`, and the result is exactly `archive-gate=PASS\n` for the supplied `C`/`A`. Goal0 remains `NO-GO` until the later integrator-only `B`, live independent exact-`B` final, later receipt-only `R`, and separate live exact-`R` final complete in order. Rollback is an ordinary revert before `B`; no history rewrite or receipt mutation is allowed.

## Open Questions

None.
