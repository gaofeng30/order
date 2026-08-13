All implementation evidence recorded before this re-DRAFT is invalidated. Existing uncommitted files are inputs to inspect, not Green evidence; no historical receipt exists.

## 1. Re-approval and Exact Boundary

- [x] 1.1 Obtain explicit approval of this exact strict-valid DRAFT; re-read apply instructions and verify base `510a84baddfb5b61200c159fd2041b7c512a92db`, frozen runner identity, branch/worktree, sole writer, dependencies, and owned paths.
  - Evidence: `/root` explicitly approved this exact DRAFT and bound `/root/receipt_implementation_writer` to `codex/persist-archived-lifecycle-receipt-v2` in `/Users/vivix/.codex/worktrees/order-persist-archived-lifecycle-receipt-v2.Writer`; `HEAD=510a84b...`, base Skill blob `2114986...`, base SHA256 `c347f88...`, strict exit `0`.
- [x] 1.2 Reconfirm the three exact candidate/archive lineages and immutable app/catalog/httpapi tree objects without treating saved handoffs, Git authors, or provenance as mechanical PASS.
  - Evidence: all three `git merge-base --is-ancestor candidate archive` checks exited `0`; both menu candidates resolve identical app `80d1642...`, catalog `1867e1c...`, and httpapi `38f9f48...` trees. These are Git facts only, not verifier identity or PASS.
- [x] 1.3 Audit existing uncommitted implementation against the approved design; retain only conforming work and record the first real mismatch before editing.
  - Evidence: first mismatch is `receipt-schema-v1.json` plus `verify_receipt.py` treating receipt-authored `candidate_verification` and `verifier_provenance` as trusted verdict inputs. All earlier Green claims remain invalid; existing parser/Git/temp helpers are retained only when covered by the new negative suites.

## 2. Red: Prove False Attestation and Unsafe Replay

- [x] 2.1 Add receipt/schema tests proving writer self-attestation, fake provenance, commit-author identity, and receipt-supplied weak commands cannot produce `MECHANICAL_PASS` or `INDEPENDENT_VERIFIED`.
  - Red: `python3 -m unittest tools/lifecycle-receipts/tests/test_trust_boundary.py` failed because the schema still contains trusted `candidate_verification`/`verifier_provenance`, lacks profile/audit fields, and the registry does not exist. `checks/test_verify_forward.py -k writer_supplied...` also failed because a clean detached writer-created fake checker still returned `fresh-session forward result PASS`.
  - Green: the final receipt test suite passed `27/27`; schema and CLI expose no receipt-supplied command, provenance, actor, or independent-verdict input.
- [x] 2.2 Add registry/executor tests proving missing or altered target SHA, profile/tool blob/hash, argv, cwd, env allowlist, expected exit/output, timeout, network/write contract, output cap, or any of the four predeclared profiles fails closed as `UNVERIFIED`.
  - Red: focused profile-runner suite cannot load because `tools/lifecycle-receipts/profile_runner.py` and the controlled registry/bindings are missing; the suite already fixes mutation cases for target/hash/blob/argv/network/write/timeout/output/isolation.
  - Green: definition/binding mutations, literal shell text, timeout, output cap, unsafe link, Go environment injection and required-artifact negatives passed without replacing production checker logic.
- [x] 2.3 Add clean-detached/temp/process tests for attached HEAD, dirty tree, nonexistent SHA, timeout/process group, output overflow, unsafe link/write/cleanup, and fake forward JSON; capture the first missing-behavior Red.
  - Evidence: forward negatives passed `7/7`, including nonexistent/wrong/attached/dirty/fake-writer cases; executor tests cover process-group timeout, overflow and unsafe links.
- [x] 2.4 Add historical Red fixtures: trusted structural parsing must reproduce the exact old-menu semantic FAIL with later Gates `NOT_RUN`; any attempted old PASS or expected-fail laundering must fail.
  - Evidence: exact `6d77bdd...` replay returned only `EXPECTED_MECHANICAL_FAIL`, the fixed artifact fingerprint and `subsequent_gates=NOT_RUN`.

## 3. Green: Controlled Mechanical Replay

- [x] 3.1 Replace trusted `candidate_verification`/provenance semantics with audit-only `recorded_attestation_json` marked `UNTRUSTED_FOR_MECHANICAL_PASS`, while retaining deterministic Markdown parsing and append-only Git checks.
  - Evidence: parser/Git/trust suites passed; public `verify_change()` always derives a controlled profile result and exposes no bypass option.
- [x] 3.2 Implement `mechanical-profiles-v1.json`, exact binding validation, and an argv-only standard-library executor with shell disabled, environment allowlist, temp-routed HOME/TMP/cache, offline flags, bounded timeout/output, pre/post clean checks, and fail-safe cleanup; test that unavailable OS-level isolation required by a profile returns `UNVERIFIED` rather than making a false enforcement claim.
  - Evidence: four definitions and three historical bindings validate; all successful final replays ended clean and removed their new validated temp trees. One earlier cleanup residue is frozen under the main-Agent safety decision and was never retried or altered.
- [x] 3.3 Implement `self-evolution-v1`: exact `7a5e8bb...` repository contract plus seven-positive/seven-negative checker replay only; explicitly exclude quick_validate/OpenSpec/Go/Node/fresh-actor claims.
  - Evidence: exact replay returned `MECHANICAL_PASS` and retained actor independence as `NOT_PROVEN_BY_MECHANICAL_REPLAY`.
- [x] 3.4 Implement `old-menu-artifact-fail-v1`: trusted Python parsing of exact `6d77bdd...` Git blobs must emit only the fixed `EXPECTED_MECHANICAL_FAIL` fingerprint and later `NOT_RUN`, without shell or historical `rg` transport.
  - Evidence: exact replay passed the structural wrapper with no missing-module substitute and no shell transport.
- [x] 3.5 Implement `menu-supersession-v1`: exact `109c8e8...` structural checker, trusted exact-base overlay Red wrapper, UI1 13/13, provider two-package test, JS51/JSON43/deps0, strict-structure/scope/sensitive/tree checks. Validate exact local Go and safe `$GOMODCACHE/cache/download`; populate a temporary GOMODCACHE through `GOPROXY=file://...` with `GOTOOLCHAIN=local GOSUMDB=off`, then set `GOPROXY=off`, run `go mod verify` and the two tests; any absence/mismatch/link/checksum error is `UNVERIFIED` with no network fallback.
  - Evidence: exact Go `1.26.5` replay passed A/MVS build-list through validated `file://`, B/two-package package closure with required `.mod/.zip/.ziphash`, and C/`GOPROXY=off` verify plus catalog/httpapi tests; UI1 `13/13`, JS `51`, JSON `43`, dependencies `0`. Ambient proxy and GOENV values cannot enter the child environment.
- [x] 3.6 Predeclare `lifecycle-receipt-control-v1` with fixed repository-contained parser/Git/registry/executor/profile/runner/forward positive/negative suites, all three historical profiles/chain, false-attestation rejection, and non-weakening checks; mark quick_validate/OpenSpec CLI/actor identity/network/UI2/UI3/order/payment excluded and required local runtime/cache mismatches `UNVERIFIED`.
  - Evidence: registry requires exactly four definitions and exactly three candidate bindings; the bootstrap profile remains deliberately unbound and cannot judge this candidate.
- [x] 3.7 Implement layered `--list`, `--change`, and `--chain` JSON: per-change recorded attestation plus derived mechanical result; old verdict `FAILED`; delivery only `mechanically_reproducible=true`; never infer actor independence.
  - Evidence: pre-candidate chain replay returned `mechanically_reproducible=true`, historical `FAILED`, exact reciprocal candidates and actor independence `NOT_PROVEN`; receipt-head Git checks remain mandatory at the committed candidate.
- [x] 3.8 Make only thin runner/reference changes that forbid a future business module from defining its judge and require a separately verified/integrated/archived preceding control-plane profile before its runner-base freeze, followed by candidate, integrator-only binding, binding-head, receipt, and receipt-head Gates before the next module; preserve all existing handlers/states/Gates.
  - Evidence: runner contract passed; Skill remains `138` lines and one-level reference `83` lines, with four routes, seven states, scoring, retry and authorization tokens unchanged.

## 4. Refactor, Backfills, and Writer Gate

- [x] 4.1 Rerun all identical negative and positive suites after refactor; do not delete tests, loosen assertions, use shell evaluation/`|| true`, or replace production checker logic with mocks.
  - Evidence: receipt tests `27/27`, forward negatives `7/7`, runner contract and all three historical profiles passed after refactor; no test was removed or skipped.
- [x] 4.2 Write exactly three receipts containing only profile/binding reference, expected historical verdict/link, audit-only recorded attestation, and `mechanical_verification=REQUIRED_DERIVED`; persist no `MECHANICAL_PASS` or `EXPECTED_MECHANICAL_FAIL`. Historical checkpoint/tasks remain byte-identical.
  - Evidence: schema parser accepts exactly three backfills; receipt fields persist only `REQUIRED_DERIVED`, preserve old `FAILED`/later `NOT_RUN`, and do not edit historical checkpoint/tasks.
- [x] 4.3 Run the three exact historical profiles and chain recovery; validate the unbound bootstrap profile definition/suites separately, require fixed product trees, old FAIL unchanged, current delivery mechanically reproducible, and actor independence explicitly unverified by replay.
  - Evidence: all-historical returned self/replacement `MECHANICAL_PASS` and old `EXPECTED_MECHANICAL_FAIL`; chain returned `mechanically_reproducible=true`, old `FAILED`, exact reciprocal candidates and actor `NOT_PROVEN`.
- [x] 4.4 Run parser/Git/executor/profile/runner/forward suites, OpenSpec strict, whitespace, sensitive/no-shell, protected/owned-path, and applicable workflow-infrastructure Gates; N/A requires an explicit boundary.
  - Evidence: Python `27/27`, forward `7/7`, runner contract, strict, exact Go `1.26.5` format/test/race/vet/build/smoke, all-app JS syntax and JSON `44` passed. UI actual is `UI0`; UI1+ is N/A because this W1 workflow change has no user interface. Final owned/sensitive/whitespace audit is repeated after this evidence freeze.
- [x] 4.5 Finalize checkpoint with actual evidence, sections 5-6 unchecked, writer score target `C9/T10/V8/R9=36`, every dimension at least 8, hard blockers zero; commit only owned paths once and hand off the exact clean candidate SHA externally.
  - Evidence: candidate artifacts are frozen at `C9/T10/V8/R9=36`, hard blockers `0`; the sole Chinese candidate commit and externally derived exact SHA are final writer actions and are never self-written into these bytes.

## 5. Independent Exact-SHA Verification

- [ ] 5.1 A different verifier builds a fresh clean detached worktree at the exact candidate and reruns all declared writer, four-profile-definition/wrapper, three-receipt, historical-profile, chain, non-weakening, local-environment, and repository-clean Gates from the beginning; the bootstrap profile remains intentionally unbound until after archive.
- [ ] 5.2 A minimal-context fresh session reruns repo-local checks, rejects fake/self-authored results, preserves old FAIL/routes/Gates, reports only layered mechanical evidence, and ends clean.
- [ ] 5.3 Enter `INDEPENDENT_VERIFIED` only for unchanged full PASS with `C9/T10/V>=9/R9`, hard blockers zero; any change invalidates evidence and returns to this writer.

## 6. Authorized Integration, Archive, and Bootstrap Receipt

- [ ] 6.1 With separate authorization, pure-FF the unchanged verified candidate into clean local main, rerun integrated Gates, and perform no push/PR/deploy/external write in this stage.
- [ ] 6.2 Archive deterministically; audit only the dated move and canonical delta, strict validate, and commit the exact archive SHA.
- [ ] 6.3 Only the integrator makes a separate append commit binding `lifecycle-receipt-control-v1` to this exact candidate and fixed integrated tool/profile blobs; candidate bytes remain unchanged and the bootstrap cannot prove actor independence.
- [ ] 6.4 A different verifier checks the exact binding-head in a new clean detached worktree, runs the fixed bootstrap profile including three-history chain/non-weakening suites, validates local environment boundaries and ending clean status; no receipt may be added before PASS.
- [ ] 6.5 In a later commit, add this change's ordinary receipt with profile reference, expected history, audit-only attestation, and `mechanical_verification=REQUIRED_DERIVED`.
- [ ] 6.6 In another clean detached worktree at the exact receipt-head, rerun the narrow structural/Git/mechanical checker and ending-clean audit; consume but never persist the derived result.
