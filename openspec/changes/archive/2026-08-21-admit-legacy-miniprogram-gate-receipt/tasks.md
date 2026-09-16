## 1. Planning and admission

- [x] 1.1 Confirm exact-B base, protected blob IDs, immutable governance candidate/archive/checkpoint/tasks facts, owner paths, dependency, non-goals, frozen runner and all five self-evolution admission fields; run `openspec validate admit-legacy-miniprogram-gate-receipt --strict`.
  - Evidence: exact-B `07688c7cdcc957bd9fed48f8c408c929a930c23f`; protected judge/schema/runner/bindings blobs `f7b53f0…/c0c9f57…/4dcd6c8…/b323055…`; immutable candidate/archive/checkpoint/tasks facts are fixed in proposal/design/spec; `openspec validate admit-legacy-miniprogram-gate-receipt --strict` PASS and `openspec validate --all --strict` PASS `23/23`; no product path is owned.
- [x] 1.2 Record the direct production Red from a clean exact-B preview containing a schema-valid standard governance receipt: `verify_receipt.py --change enforce-miniprogram-user-regression-gate` MUST exit nonzero only with `checkpoint runtime section count must be 1, got 0`; separately prove a protected judge/runner edit is rejected by current binding trust.
  - Evidence: closure worktree exact-B plus the uncommitted schema-valid standard receipt ran `/usr/bin/python3 tools/lifecycle-receipts/verify_receipt.py --change enforce-miniprogram-user-regression-gate --json`, exit `1`, exact first error `checkpoint runtime section count must be 1, got 0`. An isolated clone at exact-B committed only a harmless judge comment, then the current Mini Program profile exited `1` with `UNVERIFIED: current protected blob differs from archive: tools/lifecycle-receipts/verify_receipt.py`; therefore direct judge repair is not admissible.

## 2. Red

- [x] 2.1 Add focused tests before implementation for exact legacy checkpoint interpretation, fixed legacy path, pinned protected blobs, duplicate/standard/wrong-change/wrong-SHA/wrong-hash/tampered-history rejection, and explicit recorded-state output.
  - Evidence: `test_legacy_miniprogram_receipt.py` was added before the production module and covers exact/changed/fallback checkpoint bytes, fixed/missing/standard/duplicate/wrong-change/symlink receipt inputs, pinned blob positive/negative and explicit result boundaries. Git archive/history integration negatives remain assigned to the disposable forward fixture in 4.3.
- [x] 2.2 Run the focused test module and record an expected Red caused by the missing compatibility implementation, not fixture/import/syntax failure; confirm existing protected v1 `--list` remains Green without a standard governance receipt.
  - Evidence: `PYTHONDONTWRITEBYTECODE=1 /usr/bin/python3 -m unittest tools/lifecycle-receipts/tests/test_legacy_miniprogram_receipt.py -v` exited `1` with four setup errors, all the same missing production file `verify_legacy_miniprogram_receipt.py`; the test file itself parsed/imported and no fixture assertion or syntax failed. A clean detached exact-B replay with required Python 3.14 then ran protected `verify_receipt.py --list --json`, exit `0`, preserving exactly the four existing receipt results; Python 3.9 was separately and correctly rejected as an environment mismatch.

## 3. Green

- [x] 3.1 Add only `tools/lifecycle-receipts/verify_legacy_miniprogram_receipt.py`: exact constants/blob preflight, one legacy receipt input, narrow checkpoint compatibility view, scoped restoration of v1 inputs, production mechanical derivation, and explicit original-state/compatibility output.
  - Evidence: the adapter pins six exact-B protected blobs, accepts only the fixed legacy file/change/checkpoint digest, restores both scoped v1 hooks in `finally`, reuses v1 mechanical derivation, validates repair C→A and A→R stage shapes, and augments output with recorded state/candidate plus a fixed compatibility rule ID.
- [x] 3.2 Make focused positive and negative tests Green; verify the candidate itself still rejects because it contains no legacy receipt and cannot claim closure.
  - Evidence: focused suite `13/13` PASS on Python 3.9 with no skip; immediately afterward the production CLI on the candidate exited `1` with exact error `missing exact legacy lifecycle receipt`, so candidate bytes cannot self-claim receipt closure.

## 4. Refactor and writer Gates

- [x] 4.1 Refactor only duplicated adapter-local parsing/guard code, rerun the same focused suite, then all currently applicable stable `tools/lifecycle-receipts/tests` plus archived forward/runner contract suites; preserve the separately reproduced post-archive stale-fixture baseline instead of editing protected history tests.
  - Evidence: focused adapter suite `18/18` PASS including synthetic C→A→R and archive/parent/path/source-drift negatives; stable parser/receipt-Git/menu-profile suites `17/17` PASS; archived forward `7/7` PASS; runner contract PASS. Full discovery's remaining failures are only the reproduced pre-existing active-path/four-profile stale fixtures and were recorded as checker observation `obs-20260821T130908297461Z-e4a16d`, promotion `NOT_RUN`.
- [x] 4.2 Run protected v1 `--list`/`--chain`, all five profile/binding loader checks including exact Mini Program profile replay, OpenSpec all strict, owned/protected/sensitive/diff/ending-clean checks; prove no product/Admin/API/Harness/Skill/profile/binding/history byte changed.
  - Evidence: on clean implementation head `ac7a35a…`, exact Mini Program profile returned `MECHANICAL_PASS`, binding hash `878dbeac…`; protected v1 `--list` returned exactly four unchanged receipts and `--chain` preserved historical FAIL/supersession PASS; OpenSpec all strict `23/23`; diff from exact-B contains only ten declared new owned files, protected byte diff is empty, no product/Admin/API/Harness/Skill/profile/binding/history path changed, and status was clean before the final evidence-only artifact edit.
- [x] 4.3 Build a disposable local exact receipt-head preview after the candidate, run the production compatibility CLI and require `PASS_DERIVED` plus recorded `IMPLEMENTING`/`none`; mutate each exact boundary to demonstrate fail closed, then discard only the disposable clone.
  - Evidence: disposable clone formed `ac7a35a… → A=d24eda86… → R=5a6bbce0…`; production CLI exited `0` with exact governance C/A, `MECHANICAL_PASS`, `receipt_head_verification=PASS_DERIVED`, persisted=false, compatibility rule exact, recorded checkpoint `IMPLEMENTING`/`none`, actor independence not proven. Old v1 list/chain remained PASS at R. Focused 18-test matrix rejects checkpoint bytes, protected blobs, legacy/standard/duplicate paths, wrong change, symlink, archive smuggling, wrong R parent, multi-path R and post-archive adapter drift.

## 5. Candidate and independent verification

- [x] 5.1 Commit only owned candidate paths, record literal full candidate `C`, set Harness/checkpoint to `CANDIDATE`, and rerun the complete writer Gate at exact clean `C`.
  - Evidence: this final evidence commit changes only the active OpenSpec checkpoint/tasks after implementation commit `ac7a35a…`; its full SHA is the sole candidate `C` returned by Git and then bound into Harness. No candidate claim is valid until the post-commit exact clean rerun succeeds.
- [ ] 5.2 Independent Agent A checks out exact `C` in a new clean detached worktree and runs focused/full receipt, v1 list/chain, Mini Program profile, strict, scope/protected/sensitive and ending-clean Gates; no repository modification.
- [ ] 5.3 Independent Agent B independently repeats exact `C` in a different clean detached worktree, including the disposable exact receipt-head forward-test and all negative compatibility boundaries; both PASS records MUST name the same literal `C` and limitations.

## 6. Local integration and archive

- [ ] 6.1 Only after both exact-C PASS records, pure fast-forward unchanged local `main` to `C`, rerun the main Gate, and do not push/create PR/deploy.
- [ ] 6.2 Deterministically archive the complete change to one dated path and apply only the three declared canonical spec deltas; archive commit `A` MUST have sole parent `C`, preserve tool/test bytes and pass archive/full regressions.

## 7. Original governance receipt closure

- [ ] 7.1 Only after trusted `A`, update observation counts and add exactly one `legacy-lifecycle-receipt.md` for `enforce-miniprogram-user-regression-gate` in a later receipt-only commit `R`; recorded attestation remains untrusted and both derived fields remain `REQUIRED_DERIVED`.
- [ ] 7.2 A different independent Agent checks out exact `R` clean detached, runs old v1 `--list`/`--chain`, exact Mini Program profile and the production compatibility verifier, confirms `receipt_head_verification=PASS_DERIVED`, original state `IMPLEMENTING`/`none`, no persisted PASS and ending clean.
- [ ] 7.3 Main Gate verifies literal ancestry `f571… → b53… → 91f… → 894… → 076… → C → A → R`, exact path-only stage diffs and all current handoffs; then close only the governance receipt TODO and retain actor/UI2/UI3/real-WeChat/push/PR/deploy limits.
