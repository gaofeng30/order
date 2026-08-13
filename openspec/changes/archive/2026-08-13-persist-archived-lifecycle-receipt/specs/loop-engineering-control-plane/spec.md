## ADDED Requirements

### Requirement: Recorded attestations never prove mechanical or independent verification
The receipt MUST store only its controlled profile/binding reference, expected historical verdict, `recorded_attestation_json` labeled `UNTRUSTED_FOR_MECHANICAL_PASS`, and `mechanical_verification=REQUIRED_DERIVED`. It MUST NOT persist a derived mechanical result or derive PASS from receipt text, provenance, Git author/committer, writer identity, or session labels. `INDEPENDENT_VERIFIED` MUST still require a fresh independent `$order-verify-change` run at the exact candidate SHA.

#### Scenario: Writer claims independent PASS
- **WHEN** a writer self-attestation, fake provenance, or commit-author identity claims PASS
- **THEN** mechanical verification remains `UNVERIFIED` unless the controlled exact-SHA profile independently replays successfully
- **AND** no lifecycle state becomes `INDEPENDENT_VERIFIED`

### Requirement: Mechanical verification uses only controlled exact-SHA profiles
The checker MUST derive only `MECHANICAL_PASS`, `EXPECTED_MECHANICAL_FAIL`, or `UNVERIFIED` from `tools/lifecycle-receipts/mechanical-profiles-v1.json`. Each binding MUST fix profile ID/version/hash, change, exact target SHA, tool-source blob, ordered argv/cwd/env/timeout/expected-output steps, `network=false`, `write_scope=temp-only`, and output cap; receipt-supplied commands MUST be ignored and rejected.

#### Scenario: Exact controlled replay passes
- **WHEN** the exact target and every registry/blob/hash/step/environment/expected result match in a clean detached temp worktree
- **THEN** the checker emits the profile's fixed mechanical result
- **AND** identifies the exact profile and target used

#### Scenario: Profile evidence is missing or altered
- **WHEN** target SHA, tool blob/hash, argv, cwd, environment allowlist, expected exit/output, timeout, write scope, or registry binding is missing or mismatched
- **THEN** the checker returns non-zero and `UNVERIFIED`
- **AND** no receipt field or weaker command can compensate

### Requirement: Mechanical execution is bounded and fail closed
The executor MUST invoke argv with `shell=false`, use only allowlisted environment values, run in a validated narrow system-temp detached worktree, route HOME/TMP/cache paths into that root, apply offline tool flags, enforce process-group timeout/output caps, require clean pre/post repository state, and fail safely on cleanup or link ambiguity. `network=false` and `write_scope=temp-only` are controlled profile contracts checked through trusted wrappers and environment constraints, not claims that the Python standard library blocks every raw socket or absolute-path write. A profile requiring stronger unavailable OS isolation MUST return `UNVERIFIED`. Go steps MUST prevalidate an installed matching version and use `GOTOOLCHAIN=local`, `GOPROXY=off`, and `GOSUMDB=off`; automatic toolchain download is forbidden.

#### Scenario: Step times out or escapes its boundary
- **WHEN** a subprocess times out, exceeds output limits, dirties repository state, violates an auditable offline/temp contract, requires unavailable stronger isolation, encounters unsafe links, or cleanup cannot be proven
- **THEN** execution stops at the first decisive error and returns `UNVERIFIED`
- **AND** no later step is reported PASS

### Requirement: Four controlled profiles preserve truthful scope
The registry MUST predeclare `self-evolution-v1`, `old-menu-artifact-fail-v1`, `menu-supersession-v1`, and `lifecycle-receipt-control-v1`. Self-evolution MUST replay only its repository contract and seven-positive/seven-negative checker suite. Old menu MUST structurally reproduce its exact semantic artifact failure with later Gates `NOT_RUN`. Supersession MUST replay its fixed structural, legacy Red, UI1 13/13, provider, static, strict-structure, scope, and product-tree matrix. The bootstrap control profile MUST target this exact candidate after archive and replay the repository-contained parser/Git/registry/executor/profile/runner/forward positive and negative suites, three historical profiles/chain, false-attestation rejection, and non-weakening checks.

#### Scenario: Old menu expected failure is replayed
- **WHEN** exact Git blobs reproduce fingerprint `artifact-consistency|semantic|proposal-stale-draft-not-run|6d77bdd|verifier`
- **THEN** the result is `EXPECTED_MECHANICAL_FAIL` and later Gates remain `NOT_RUN`
- **AND** the historical candidate remains `FAILED`

#### Scenario: Environment-dependent evidence is unavailable
- **WHEN** an excluded or required local toolchain is unavailable or incompatible
- **THEN** the affected claim is excluded as declared or becomes `UNVERIFIED`
- **AND** recorded provenance is not used as fallback

#### Scenario: Bootstrap control profile is bound after archive
- **WHEN** `lifecycle-receipt-control-v1` is exact-target-bound to this independently verified candidate after integration/archive
- **THEN** a different clean-detached binding-head verifier reruns its fixed repository-contained suites and three-history chain
- **AND** user-directory quick validation, mutable OpenSpec CLI behavior, Git author/session identity, network, UI2/UI3, and real order/payment remain excluded

### Requirement: Go replay uses a validated local module proxy cache
Go profile steps MUST require an already-installed exact Go version and a safe local `$GOMODCACHE/cache/download`. A trusted wrapper MUST resolve the source cache against unsafe links; construct an explicit temp-only environment with `GOENV=off` and no inherited proxy, private-module, VCS or credential variables; let exact Go obtain the MVS/pruned build list and actual two-target package closure through a `file://` GOPROXY with `GOTOOLCHAIN=local` and `GOSUMDB=off`; and validate the package closure's required `.mod`, `.zip`, and `.ziphash`. It MUST NOT invent metadata, require unrelated cached `.info`, recursively interpret module requirements, or explicitly download non-package-closure zips. It then MUST set `GOPROXY=off`, run `go mod verify`, and run the fixed package tests. It MUST NOT use a network proxy or automatic toolchain download.

#### Scenario: Local Go and cache are complete
- **WHEN** exact local Go, safe cached module artifacts, temporary population, and `go mod verify` all pass
- **THEN** the fixed provider package tests may contribute to the derived mechanical result
- **AND** all Go writes and caches are routed to the validated temporary root

#### Scenario: Local Go cache prerequisite fails
- **WHEN** Go version, cache archive, path/link safety, module checksum, or offline population is missing or mismatched
- **THEN** the profile returns `UNVERIFIED`
- **AND** it does not contact the network, weaken verification, or use recorded attestation as fallback

### Requirement: Archived recovery uses one append-only receipt
After existing candidate verification, integration, and archive Gates, the control plane MUST add exactly one `lifecycle-receipt.md` in the dated archive, keep candidate checkpoint/tasks byte-identical, and store `receipt_head_verification=REQUIRED_DERIVED`. Recovery MUST verify exact Git objects/ancestry, archive path/diff, ownership, candidate artifact digests/tasks, retrospective, unique receipt add, and no later receipt touch.

#### Scenario: Receipt history is valid
- **WHEN** structural/Git checks and the required controlled mechanical replay all pass
- **THEN** recovery reports the immutable receipt head and layered evidence result
- **AND** writes neither the derived head nor PASS back to repository evidence

#### Scenario: Receipt is stale or tampered
- **WHEN** the receipt is missing, duplicated, edited later, ancestry-inconsistent, task-inconsistent, or structurally ambiguous
- **THEN** recovery returns non-zero `NO-GO`
- **AND** archive presence alone does not close recovery

### Requirement: Supersession recovery never launders the old failure
The old-menu receipt MUST store expected verdict `FAILED`, later Gates `NOT_RUN`, and `mechanical_verification=REQUIRED_DERIVED`; the checker MUST re-derive `EXPECTED_MECHANICAL_FAIL` on every recovery. A reciprocal replacement receipt may support only `mechanically_reproducible=true` for the current delivery when its own re-derived mechanical result, integration/archive facts, receipts, links, and exact app/catalog/httpapi tree identities all agree.

#### Scenario: Complete replacement chain is recovered
- **WHEN** old FAIL, reciprocal links, replacement `MECHANICAL_PASS`, receipt histories, exact stage facts, and product trees all match
- **THEN** current delivery is reported mechanically reproducible
- **AND** neither old candidate nor recorded attestation is reported PASS or `INDEPENDENT_VERIFIED`

#### Scenario: Old candidate is upgraded
- **WHEN** any field or chain attempts to assign PASS to the old candidate
- **THEN** validation returns non-zero
- **AND** product-tree equality, score, provenance, or archive presence cannot override the FAIL

### Requirement: Profile admission and bootstrap do not weaken lifecycle Gates
For every future business module, a separate preceding control-plane change MUST define its profile and trusted wrapper, receive independent exact-SHA verification, integrate/archive into local main, and be included in the business module's frozen runner base. The business module MUST NOT define or modify its own judge. After the business candidate independently verifies and pure-FF integrates/archives, only the integrator MAY append the exact-target binding; a different clean-detached verifier MUST pass that binding-head before a later receipt commit, and another clean-detached verifier MUST derive the receipt-head before the next module starts. Binding and receipt commits MUST NOT edit the candidate. This change alone MAY bootstrap with its predeclared `lifecycle-receipt-control-v1`, but MUST still pass candidate, binding-head, and receipt-head verifiers. Receipts and mechanical replay MUST NOT replace any existing Gate.

#### Scenario: Business module tries to define its own judge
- **WHEN** a business module defines or edits its own profile/wrapper, or freezes a base before the preceding control-plane change is independently verified and archived in local main
- **THEN** planning or candidate validation returns `NO-GO`
- **AND** that profile cannot judge the business module

#### Scenario: Future candidate lacks a closed evidence chain
- **WHEN** a future module lacks independent candidate PASS, pure-FF integration/archive, integrator-only exact binding, independent binding-head PASS, receipt commit, or independent receipt-head derivation
- **THEN** receipt closure returns `NO-GO`
- **AND** the runner does not allocate the next module from that archive

#### Scenario: Bootstrap change closes its own receipt
- **WHEN** this change first passes a different clean-detached exact-SHA verifier, then integrates/archives, binds the fixed integrated tool blob, and a different verifier passes the exact binding-head
- **THEN** its later receipt may be added with `REQUIRED_DERIVED` and undergo a separate clean-detached receipt-head check
- **AND** the exception cannot be reused or cited as actor-independence proof

### Requirement: Fresh-session recovery is repository bound
The current candidate verifier MUST use a clean detached exact-SHA worktree and minimal repository context, rerun the controlled receipt/profile/chain checks, preserve the old FAIL and stage routes, reject invalid fixtures, and leave the repository clean.

#### Scenario: Fake forward result is presented
- **WHEN** a result names a nonexistent or wrong candidate, attached/dirty repository, fake receipt output, or writer-authored PASS
- **THEN** validation returns non-zero
- **AND** only repo-local checker results at the exact detached SHA are accepted
