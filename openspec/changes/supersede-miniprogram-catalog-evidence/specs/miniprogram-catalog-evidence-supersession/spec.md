## ADDED Requirements

### Requirement: Historical verification failure remains immutable

The supersession evidence MUST preserve `connect-miniprogram-menu-catalog@6d77bdd6319722b7c71b4726c6159955da9a84b6` as verifier FAIL with fingerprint `artifact-consistency|semantic|proposal-stale-draft-not-run|6d77bdd|verifier`. It MUST distinguish the successful field-search process exit from the semantic FAIL, retain the clean-detached environment and first decisive proposal/checkpoint/tasks contradiction, and state that every later Gate was `NOT_RUN`. It MUST NOT edit the old candidate/archive or attribute a later PASS to that SHA.

#### Scenario: Historical search finds contradictory fields

- **WHEN** the old candidate contains proposal DRAFT/none/NOT_RUN fields and checkpoint/tasks CANDIDATE/UI1/completion fields
- **THEN** the historical verdict remains artifact-consistency FAIL even though the search command exits zero
- **AND** exact-base Red, UI1, provider, static and scope checks remain recorded as not run for `6d77bdd...`

#### Scenario: A later current-tree verification passes

- **WHEN** a different exact candidate based on current main passes every supersession Gate
- **THEN** only that new candidate receives the new attestation
- **AND** `6d77bdd...` remains FAIL without changed archive bytes or retroactive PASS wording

### Requirement: Superseding evidence binds a new exact candidate to the current product tree

The supersession candidate MUST have repository base `7d01fe22ded67aeded78cb7d03de87aa12416ada` and MUST change only `openspec/changes/supersede-miniprogram-catalog-evidence/**`. Its app and provider trees MUST remain equal to both the repository base and the historical implementation trees: app `80d16424aefa0d4b9d4e451a1ebe5e8013627a8b`, catalog provider `1867e1cb94fd38b718641d28022e1cf2e386c85b`, and httpapi provider `38f9f486156547cd547d2f3840566acfbbd4c0eb`. The full candidate SHA MUST be derived after commit from Git and external handoff, never written into its own candidate artifacts.

#### Scenario: Evidence candidate is formed

- **WHEN** the writer commits the completed supersession artifacts
- **THEN** Git proves repository base is the candidate parent, every changed path is under the one owned directory and every protected product/archive/canonical/runner/receipt path is unchanged
- **AND** the writer emits the full SHA and clean status externally without modifying the candidate

#### Scenario: Candidate or protected tree changes

- **WHEN** base, candidate SHA, product tree, old archive, spec, task, command or any protected path differs from the attested inputs
- **THEN** all writer and independent supersession evidence is invalid
- **AND** a new exact candidate MUST be created and verified from the beginning

### Requirement: Superseding candidate reruns the complete W2 UI1 matrix

The writer and independent verifier MUST each replay the three focused legacy failures on exact base `94e04bf26e37e93299c26ef2c9c8aa7552619444`, then run the current exact candidate's UI1 13/13 suite, focused catalog/httpapi provider tests, JavaScript syntax, JSON parsing, zero-dependency, OpenSpec strict, whitespace, forbidden/legacy/fake-ID, owned/protected, sensitive and ending-clean checks. The Red replay MUST fail only with list request zero, queued-network request zero and detail fallback `p001`, not missing modules. UI2 and UI3 MUST remain `BLOCKED_EXTERNAL`; promo/order/pay results MUST be non-regression only.

#### Scenario: Full current-tree Gate passes

- **WHEN** exact-base replay returns 3 tests/0 pass/3 expected behavior failures and every candidate Green/static/scope Gate passes in the same exact tree
- **THEN** actual UI evidence may be recorded as UI1 for that candidate
- **AND** real order, payment, availability, quote, UI2, UI3 and production behavior remain unverified

#### Scenario: Any required Gate is absent or fails

- **WHEN** either verifier skips a command, reuses old logs, gets a different Red fingerprint, reports fewer than 13 UI1 passes, crosses owned paths, exposes sensitive data or ends dirty
- **THEN** the supersession verdict is `NO-GO`
- **AND** another PASS, score or product-tree equality cannot compensate

### Requirement: Independent supersession precedes integration archive and receipt consumption

Only another session in a newly clean detached worktree MAY attest the committed full candidate SHA. The attestation MUST report the exact SHA and complete Gate results without repository modification. Integration MUST be separately authorized and pure fast-forward; deterministic archive MAY occur only after integrated-main verification. `persist-archived-lifecycle-receipt` MUST treat this change's actual main integration as a dependency and MUST NOT consume a DRAFT, branch, unintegrated candidate or old `6d77bdd...` claim as PASS evidence.

#### Scenario: Exact candidate independently passes

- **WHEN** the clean detached verifier reruns every declared Gate against the unchanged full candidate SHA and returns PASS
- **THEN** that SHA may enter `INDEPENDENT_VERIFIED` while the old FAIL remains unchanged
- **AND** integration/archive still require their own authorization and evidence

#### Scenario: Downstream receipt requests evidence too early

- **WHEN** the supersession change is not actually integrated in current local main or its exact-SHA verification is missing/stale
- **THEN** the receipt change remains blocked on this dependency
- **AND** it MUST NOT backfill a PASS from old chat, writer self-attestation, branch state or the failed historical candidate
