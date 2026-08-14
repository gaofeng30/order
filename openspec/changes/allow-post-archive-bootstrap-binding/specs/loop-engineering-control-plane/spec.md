## ADDED Requirements

### Requirement: Bootstrap binding admission is stage-aware and exact
The lifecycle receipt control plane MUST accept exactly the three immutable historical bindings while the repair candidate is unbound. Candidate `C` MUST contain `checks/verify_archive.py` and the complete target bytes in `checks/expected-canonical-loop-engineering-control-plane-spec.md`, and both exact-`C` verifier recipes MUST exercise the archive-checker fixtures. Every archive-Gate judgment MUST execute the checker blob from exact `C`, and that checker MUST execute the runner-validator blob from exact `C`; the corresponding `A` files MUST be compared subjects only. It MUST accept one additional `lifecycle-receipt-control-v1` binding only when uniquely derived repair archive `A` has exactly one parent equal to `C`; `C→A` consists solely of a complete byte-identical `R100` move of every active change blob to one dated archive plus one canonical spec `M`; the canonical blob at `A` equals the expected fixture blob at `C` byte-for-byte; protected registry/wrapper/executor/judge blobs are identical at `C` and `A`; and a unique later binding-only commit `B` descends from `A`. The additional binding MUST target `d0b70a077bcaa64c401837eb0e9b6f27035210a0`; match the fixed controlled profile definition, hash, argv, environment, expected results, wrapper and repaired executor bytes at both `C` and `A`; and leave those controlled bytes identical at the current head. The loader MUST derive only Git/byte/mechanical results, MUST report actor independence as `NOT_PROVEN_BY_MECHANICAL_REPLAY`, and MUST NOT consume thread pointers or create a lifecycle state.

#### Scenario: Candidate remains valid and unbound
- **WHEN** the repair candidate contains exactly the three historical bindings and no bootstrap binding
- **THEN** registry loading and the existing historical profiles remain valid
- **AND** the candidate receives verification only from an ordinary fresh clean-detached exact-SHA verifier

#### Scenario: Exact post-archive bootstrap binding is admitted
- **WHEN** the dispatcher loads checker bytes from literal full `C`, that checker loads runner-validator bytes from the same `C`, those trusted bytes prove `HEAD=A` and clean, the sole parent and complete same-relative `R100` dated move, the sole canonical `M` blob equals the expected fixture at `C`, protected blobs `A=C`, and one later binding-only `B` appends the fixed `lifecycle-receipt-control-v1` target with every definition/source/blob/hash/execution field matching `C` and `A`
- **THEN** the loader accepts exactly four bindings and the binding-head verifier may execute the fixed bootstrap profile
- **AND** `--all-historical` still selects only the three historical profiles

#### Scenario: Archive Gate is exact and read-only
- **WHEN** a fixed direct-argv bootstrap at exact clean `HEAD=A` loads `checks/verify_archive.py` from the literal full `C` Git blob, passes `--repo . --candidate <literal-full-C-SHA> --archive <literal-full-A-SHA>`, and that checker loads its runner validator from the same `C`
- **THEN** success exits `0` with stdout exactly `archive-gate=PASS` plus one LF and no mutation
- **AND** any non-full SHA, wrong HEAD, dirty status, wrong parent/rename/path, canonical extra/missing/reordered byte, protected registry/wrapper/executor/judge drift, or other ambiguity exits `1` at the first decisive error

#### Scenario: Archive carries an unverified change
- **WHEN** `A` supplies the checker or runner used as judgment authority, `A` is a merge, its sole parent is not exact `C`, an active change blob is missing or not moved with same-relative `R100` byte identity, the canonical blob differs by any extra/missing/reordered byte from the expected fixture at `C`, or any other path including registry, wrapper, executor, judge, Skill, product, or test source changes in `C→A`
- **THEN** loading fails closed before the bootstrap profile executes
- **AND** archive presence, ancestry alone, author identity, or a later matching working-tree file cannot compensate

#### Scenario: Binding is premature or malformed
- **WHEN** bootstrap is bound before repair archive, at the same archive stage, more than once, with an extra binding, or with a wrong target, definition, hash, path, blob, argv, environment, expected result, timeout, network/write contract, or output cap
- **THEN** loading fails closed before profile execution
- **AND** receipt text, provenance, author identity, or a receipt-supplied command cannot compensate

#### Scenario: Binding history or repaired source drifts
- **WHEN** the archive/binding commits are missing, ambiguous or ancestry-inconsistent, the binding commit changes another path, the binding is later edited, or current controlled sources differ from exact `C`/`A`
- **THEN** loading returns `UNVERIFIED` with the first decisive error
- **AND** no historical receipt is rewritten or upgraded

### Requirement: Repair promotion and Goal0 closure are separately gated
Before APPROVED/implementation, the main Gate MUST read a current independent thread final proving exact `d0b70a077bcaa64c401837eb0e9b6f27035210a0` PASS because its archived checkpoint/tasks explicitly do not. Before integration/archive, dispatcher-supplied literal full repair candidate `C` MUST pass both a no-secret minimal-context forward-test and ordinary exact-SHA independent verification in separate verifier-created clean detached worktrees; both MUST run the archive-checker fixtures against the exact expected canonical bytes in `C`; and the main Gate MUST read both current exact-`C` finals. A repository audit pointer or `wait_cursor` MUST NOT itself prove PASS. The repair MAY reach `ARCHIVED` at trusted `A`, but Goal0 MUST remain `NO-GO` until an authorized integrator appends exact binding-head `B`, a different verifier returns a current exact-`B` final after fixed profile `MECHANICAL_PASS`, the integrator adds later receipt-head `R`, and another verifier returns a current exact-`R` final after deriving `PASS_DERIVED` without persisting it.

#### Scenario: Historical dependency handoff is pending or unavailable
- **WHEN** exact `d0b70a...` handoff is active/PENDING, missing, unreadable, stale, non-final, wrong-SHA, or not PASS even though repository archive/checkpoint/receipt text exists
- **THEN** the main Gate returns `NO-GO` for APPROVED/implementation
- **AND** waits for the current verifier or dispatches a fresh exact-SHA verifier instead of inferring PASS from repository text

#### Scenario: Current exact handoff is readable
- **WHEN** the main Gate calls `read_thread` with recorded `threadId` and `hostId` and the current final contains the expected full SHA, commands, result, and limitations
- **THEN** the independent Gate may consume that current handoff for actor/session separation
- **AND** an optional stored `wait_cursor` is only `wait_threads.afterCursor`; it may be unavailable across callers without affecting final readability, MUST NOT be reused as `read_thread`'s separate pagination cursor, and never becomes loader input or mechanical proof

#### Scenario: Minimal-context forward-test passes independently
- **WHEN** the dispatcher supplies repository, change name, and literal full `C`; the verifier creates a clean detached exact-SHA worktree under the verification contract; and the fixed recipe proves exact/detached/clean pre/post state while running the loader and archive-checker local temporary-Git positive plus A-controlled checker/runner, canonical-byte/judge/path/parent/rename/HEAD negatives without network or secrets
- **THEN** the minimal-context Gate records PASS for exact `C`
- **AND** that PASS does not replace the separate ordinary independent-verification record

#### Scenario: Repair is archived before Goal0 closure
- **WHEN** exact `C` is independently verified and trusted `A` is integrated/archived but `B`, independent binding-head PASS, later `R`, or independent receipt-head derivation is missing, stale, dirty, non-detached, wrong-SHA, out of order, or performed by a forbidden role
- **THEN** this repair change may be `ARCHIVED` while Goal0 remains `NO-GO`
- **AND** the runner MUST NOT treat repair archive state as Goal0 completion

#### Scenario: Goal0 closure completes in order
- **WHEN** the integrator-only exact `B`, different verifier's current exact-`B` handoff, later receipt-only `R`, and another verifier's current exact-`R` handoff all match their dispatcher-supplied literal SHAs, declared commands, mechanical outputs, and clean-state results
- **THEN** Goal0 receipt closure may be reported complete
- **AND** mechanical replay still does not prove actor/session independence

#### Scenario: Repair archive precedes external receipt closure
- **WHEN** this repair itself has ordinary candidate PASS and is integrated and archived while the original control receipt remains unbound
- **THEN** the repair may reach `ARCHIVED` without using the bootstrap profile to attest itself
- **AND** the later exact `B`, independent binding-head PASS, receipt-only `R`, and separate receipt-head derivation remain required before Goal0 is closed
