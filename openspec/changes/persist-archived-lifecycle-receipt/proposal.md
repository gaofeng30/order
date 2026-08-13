## Why

Archived changes need a repository-local recovery record, but the first implementation could falsely turn receipt-authored `candidate_verification=PASS` or provenance text into a verified conclusion. Git text, commit author, and writer self-attestation cannot prove actor/session independence, so archived recovery must separate untrusted recorded attestations from checker-derived mechanical replay.

## What Changes

- Add exactly one append-only `lifecycle-receipt.md` after archive without editing candidate-era checkpoint/tasks or adding a lifecycle state.
- Store only a controlled profile reference, expected historical verdict, audit-only `recorded_attestation_json` marked `UNTRUSTED_FOR_MECHANICAL_PASS`, and `mechanical_verification=REQUIRED_DERIVED`; no receipt persists its own mechanical result.
- Add a repository-controlled `mechanical-profiles-v1.json` registry and fail-closed executor. The checker derives only `MECHANICAL_PASS`, `EXPECTED_MECHANICAL_FAIL`, or `UNVERIFIED` by replaying fixed argv-only profiles at an exact candidate SHA.
- Predeclare four profiles: the three historical replay profiles plus `lifecycle-receipt-control-v1`, which targets this change's exact candidate only after archive and fixes the repository-contained parser/Git/registry/executor/profile/runner/forward, three-history chain, and non-weakening suites used to close the bootstrap.
- Backfill three honest histories whose results are re-derived on every recovery: self-evolution can reproduce deterministic checks but not actor independence; old menu stays permanently `FAILED` with expected artifact failure and later Gates `NOT_RUN`; supersession can reproduce its deterministic matrix with reciprocal links and exact product-tree identity.
- Recover the current menu delivery only as mechanically reproducible from the old FAIL plus replacement evidence. Never relabel the old candidate or any recorded attestation as `INDEPENDENT_VERIFIED`.
- Keep full fresh independent `$order-verify-change` exact-SHA verification mandatory for every real `INDEPENDENT_VERIFIED` transition. Mechanical replay is additive recovery evidence, not actor proof.
- Extend the runner only as a thin archive-closure route. For every future business module, a separate preceding control-plane change must define its profile/wrapper, pass independent verification, integrate/archive into local main, and be included in the business module's frozen runner base. A business module cannot define or edit its own judge profile.
- After a future business candidate independently verifies and pure-FF integrates/archives, only the integrator may append its exact-target binding. A separate clean-detached binding-head PASS must precede the receipt commit, and another clean-detached receipt-head derivation must close the module before the next module starts.
- Treat this change as the one-time bootstrap: its own candidate first receives normal independent exact-SHA verification; only after integration/archive can its fixed tool blob and exact candidate be bound, independently verified at the binding-head, and then referenced by its later receipt.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `loop-engineering-control-plane`: add append-only archive receipts, untrusted attestation recording, controlled exact-SHA mechanical replay, truthful supersession recovery, and fail-closed receipt closure without weakening any existing Gate.

## Impact

Owner: `/root/receipt_implementation_writer` is the sole implementation writer after this DRAFT is explicitly re-approved; the main Agent must bind that role to the dedicated branch/worktree before implementation.

Writer-owned paths:

- `openspec/changes/persist-archived-lifecycle-receipt/**`
- `.agents/skills/order-run-loop/SKILL.md`
- `.agents/skills/order-run-loop/references/self-evolution.md`
- `tools/lifecycle-receipts/**`
- `openspec/changes/archive/2026-08-13-enable-run-loop-self-evolution/lifecycle-receipt.md`
- `openspec/changes/archive/2026-08-13-connect-miniprogram-menu-catalog/lifecycle-receipt.md`
- `openspec/changes/archive/2026-08-13-supersede-miniprogram-catalog-evidence/lifecycle-receipt.md`

Integration-only deterministic outputs are the canonical `openspec/specs/loop-engineering-control-plane/spec.md` delta, this change's dated archive move, its post-archive exact-target profile binding, and its later `lifecycle-receipt.md`. They are not candidate-writer outputs.

Dependencies: local Git objects for the three histories; archived `supersede-miniprogram-catalog-evidence`; repository-contained Python checks; and local runtime/environment assets required by controlled profiles. DRAFT preflight observed Python `3.14.6`, Git `2.53.0`, Node `25.8.1`, npm `11.11.0`, local Go `1.26.5 darwin/arm64`, and a non-link `$GOMODCACHE/cache/download`; implementation must revalidate exact versions, safe paths, cache completeness, and module checksums. Any missing/mismatched/unsafe asset yields `UNVERIFIED`; no network fetch is allowed. User-directory quick validation, mutable OpenSpec CLI behavior, fresh-session actor identity, UI2/UI3, and real order/payment are excluded from mechanical replay.

`gate_type`: `W1`. `ui_level_target`: `UI0`; actual remains `NOT_RUN` until implementation evidence exists.

Non-goals: no business code, API, product spec, score change, root governance, other stage skill, history rewrite, cryptographic identity claim, push/PR/deploy, external write, or cleanup of legacy worktrees. UI2/UI3 and real order/payment remain outside this workflow change.

Minimum success: four consistent strict-valid artifacts; meaningful Red for all false-green paths; four controlled profiles and positive/negative replay; three truthful historical receipts; exact clean-detached candidate verification; then authorized pure-FF integration/archive, exact bootstrap binding, independent binding-head PASS, later receipt commit, and independent receipt-head derivation.
