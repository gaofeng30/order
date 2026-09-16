# miniprogram-gate-lifecycle-profile Specification

## Purpose
TBD - created by archiving change add-miniprogram-gate-receipt-profile. Update Purpose after archive.
## Requirements
### Requirement: Mini Program Gate profile definition is fixed and initially unbound

The repository MUST add exactly one profile `miniprogram-user-regression-gate-v1` for change `enforce-miniprogram-user-regression-gate` and target `f5719e98690d0b1301ed567c8b616e074846b445`. The definition MUST declare `network=false`, `write_scope=temp-only`, bounded time/output, exact required tool versions, a fixed wrapper path and exact success output. Candidate bytes MUST contain five fixed profile definitions while retaining exactly the current four bindings; candidate MUST NOT add the governance binding or receipt.

Existing four profile definitions, bindings, wrapper/executor source contracts and historical receipts MUST remain valid and unchanged. Registry text, profile score, task text or an unbound receipt MUST NOT authorize target execution.

#### Scenario: Candidate contains an unbound fifth definition
- **WHEN** the profile change candidate loads its five-profile registry with the unchanged current four bindings
- **THEN** loader validation succeeds and historical plus bootstrap bindings remain exact
- **AND** `miniprogram-user-regression-gate-v1` cannot execute because no binding exists

#### Scenario: Candidate self-binds the target
- **WHEN** candidate bytes append the governance binding, alter an existing binding or provide more than the exact current four bindings
- **THEN** loader validation fails before any profile command executes
- **AND** tasks, author identity, receipt text or mechanical output cannot compensate

### Requirement: Profile replay is exact, isolated and fail closed

The bound profile MUST clone/check out exact target `f5719e98690d0b1301ed567c8b616e074846b445` in a clean detached worktree and MUST execute only the fixed argv wrapper from the independently verified profile source. The wrapper MUST verify exact HEAD and clean state, then run Mini Program Gate/Harness control tests, Mini Program UI1 regression, Go format/test/race/vet/build/smoke and deterministic target scope/protected/sensitive checks. It MUST use no network or repository/external writes; all mutable HOME/cache/tmp output MUST remain within the validated profile temp root.

Success MUST exit zero with stdout exactly `miniprogram-user-regression-gate-v1=MECHANICAL_PASS` plus one LF. Any command failure, timeout, output overflow, target/source/tool mismatch, dirty state, symlink, write escape, protected/product drift or unexpected output MUST fail at the first decisive result. Mechanical PASS MUST explicitly exclude actor independence, mutable OpenSpec CLI behavior, user-directory Skill validation, UI2/UI3 and real WeChat results.

#### Scenario: Exact target passes the fixed matrix
- **WHEN** binding-head verifier executes the fixed profile with the required local tool versions and module cache against exact clean target
- **THEN** every declared control, Mini Program UI1, Go and deterministic non-weakening check passes and output is the exact success line
- **AND** target worktree remains detached and clean after execution

#### Scenario: A Gate or isolation boundary is weakened
- **WHEN** a fixture removes a required test, changes target/product/protected bytes, returns an unexpected result, writes outside profile temp or exceeds a bound
- **THEN** the profile exits nonzero with a sanitized first error
- **AND** another PASS, coverage number or receipt cannot upgrade the result

### Requirement: Governance and profile archives are proven from exact candidate authority

The profile candidate `C` MUST contain read-only checker code and complete expected canonical fixtures for both the historical governance transition `T=f5719e98690d0b1301ed567c8b616e074846b445 → TA=b53cb520c4cac0505803a9d1e6dcc1807ac34540` and the future profile transition `C → A`. Exact-C minimal-context and ordinary verifiers MUST execute positive and negative archive fixtures before candidate promotion.

For governance, exact-C authority MUST prove `TA` has sole parent `T`, every active governance change blob moved exactly once with same-relative `R100` into one dated archive, only the three declared canonical specs changed to the complete fixture bytes, and profile/binding/runner/judge/run-loop/product bytes stayed unchanged. For the profile change, exact-C authority MUST prove `A` has sole parent `C`, every active blob moved exactly once, only the new capability canonical spec changed to its complete fixture bytes, and all candidate-owned implementation plus protected bytes stayed unchanged. Archive worktree copies MUST never be judgment authority.

#### Scenario: Both exact archive histories match
- **WHEN** exact-C checker bytes inspect literal full `T`, `TA`, `C` and `A` and every parent/path/rename/blob/canonical/protected invariant matches
- **THEN** archive trust returns the exact declared PASS output without mutation
- **AND** the result proves only Git/byte mechanics, not actor independence

#### Scenario: Archive supplies or changes its own judge
- **WHEN** either archive has a wrong or merge parent, incomplete/non-R100 move, extra path, canonical byte mismatch, implementation/protected drift, dirty HEAD or uses checker/runner bytes from archive worktree as authority
- **THEN** archive trust fails before binding admission
- **AND** ancestry alone, current matching files, author identity or receipt text cannot compensate

### Requirement: Only one exact post-archive governance binding is admitted

After profile change `C` receives both required independent PASS records and exact archive `A` passes, the loader MUST allow only one later binding-only commit `B` that appends `miniprogram-user-regression-gate-v1` as the fifth binding. The binding MUST target exact `T`, match the profile definition SHA256, tool/executor paths and Git blobs from exact `C/A`, descend from exact `A`, change only the bindings registry, preserve the first four binding objects byte-for-byte and have no later edit.

Before exact `A`, at `A`, with a wrong parent/order/path/hash/blob/target/profile, as a merge, with another file change, as a duplicate/sixth binding or after later modification, the loader MUST return `UNVERIFIED` before execution. Existing bootstrap binding history MUST remain valid under the sole fifth append and MUST NOT be rewritten.

#### Scenario: Exact later fifth binding is admitted
- **WHEN** authorized integrator appends the sole exact fifth object after trusted `A` and current controlled sources equal exact `C/A`
- **THEN** loader admits binding `B` and a different clean-detached verifier may run the profile
- **AND** the first four bindings and all prior receipts retain their original meaning

#### Scenario: Fifth binding is premature or malformed
- **WHEN** binding appears before trusted `A`, changes another path/object, uses a stale source/target/hash, is duplicated or is later edited
- **THEN** loader fails closed with the first deterministic error
- **AND** no profile command or receipt derivation runs

### Requirement: Receipt closure remains ordered and externally verified

Profile candidate PASS, archive PASS and binding presence MUST NOT close the governance lifecycle receipt. A different clean-detached verifier MUST first execute exact binding-head `B` and report current `MECHANICAL_PASS`. Only afterward MAY an authorized integrator append exactly one governance `lifecycle-receipt.md` in a later receipt-only commit `R`; another clean-detached verifier MUST run schema/Git/profile/chain verification at exact `R` and derive receipt-head PASS without persisting that derived result.

The receipt MUST record mechanical and receipt-head fields as `REQUIRED_DERIVED`, preserve the exact governance candidate/archive facts and excluded claims, and MUST NOT assert actor independence. Missing, stale, reordered, dirty, wrong-SHA, same-verifier, later-edited or failed binding/receipt evidence MUST keep business-module allocation `NO-GO`.

#### Scenario: Binding exists but receipt closure is incomplete
- **WHEN** `B` exists but binding-head PASS, later receipt commit or separate receipt-head PASS is missing
- **THEN** governance closure remains `NO-GO`
- **AND** the next Mini Program business module is not allocated

#### Scenario: Binding and receipt heads pass in order
- **WHEN** different clean-detached verifiers pass exact `B` and later exact `R` with all declared mechanical and chain checks
- **THEN** governance lifecycle receipt closure may be reported complete
- **AND** the next module still captures a new runner base and treats mechanical replay as non-actor evidence
