## Context

The base is `510a84baddfb5b61200c159fd2041b7c512a92db`. The frozen runner is Skill blob `211498668419dcb66d11f5bfacf7457ed385aa05`, SHA256 `c347f88e3593ed80cea79e4b2d4885bb6d92c677d1e1012c12171fab34482d14`, version `unversioned`.

Candidate checkpoint/tasks must remain byte-identical after exact verification. A later archive cannot write its SHA back into the candidate, and a receipt cannot contain its own commit SHA. The existing partial implementation also exposed two P1 false greens: a forward validator accepted a nonexistent self-reported candidate, then the receipt checker accepted writer-authored PASS/provenance as verification. All prior task-2/task-3 Green evidence is invalid.

This is W1/UI0 workflow infrastructure. It changes no product behavior and adds no lifecycle state.

## Goals / Non-Goals

Goals are append-only archive recovery, exact-SHA mechanical reproducibility, truthful historical FAIL/supersession semantics, deterministic fail-closed execution, and preservation of mandatory fresh independent verification.

Non-goals are cryptographic identity, proof of which human/session ran a command, business/product changes, remote services, arbitrary receipt-provided commands, environment-dependent claims, or retroactive conversion of historical text into PASS.

## Decisions

### 1. Separate audit attestation from derived evidence

Receipt field `recorded_attestation_json` contains the historical handoff only for audit and is always interpreted as `UNTRUSTED_FOR_MECHANICAL_PASS`. Neither its verdict/provenance strings, Git author/committer identity, receipt ownership, nor chat/session labels can produce any checker PASS.

The checker derives `mechanical_verification` exclusively as:

- `MECHANICAL_PASS`: every fixed deterministic step for the exact target passes;
- `EXPECTED_MECHANICAL_FAIL`: the old-menu profile reproduces its one fixed semantic failure and deliberately leaves later Gates `NOT_RUN`;
- `UNVERIFIED`: profile missing/mismatched, any unexpected exit/output, timeout, unsafe environment, dirty tree, or unavailable deterministic tool.

These values are recovery evidence, not lifecycle states and not actor-independence evidence. `INDEPENDENT_VERIFIED` still requires a fresh independent `$order-verify-change` run at the exact candidate SHA.

The receipt persists only `profile_id`, its exact binding reference, expected historical verdict, `recorded_attestation_json`, and `mechanical_verification=REQUIRED_DERIVED`. `MECHANICAL_PASS`, `EXPECTED_MECHANICAL_FAIL`, and `UNVERIFIED` exist only in current checker output and are recomputed on every recovery.

### 2. Repository-controlled profile definitions and exact target bindings

`tools/lifecycle-receipts/mechanical-profiles-v1.json` is the only command source. Receipt fields cannot add, replace, or weaken commands. Each controlled binding contains:

- `registry_version`, `profile_id`, `change_name`, and exact `target_sha`;
- `tool_source_blob` plus the profile definition hash/version;
- ordered steps with argv arrays, repository-relative cwd, explicit environment allowlist, timeout, expected exit and bounded output assertions;
- `network=false`, `write_scope=temp-only`, and an output-size cap as profile contracts.

The executor uses `subprocess` argv with `shell=false`; checks out the exact target in a validated narrow system temp directory; requires detached HEAD and clean pre/post status; clears non-allowlisted environment variables; points `HOME`, `TMPDIR`, cache directories, `GOCACHE`, and npm cache into the validated temp root; applies offline tool flags; uses a bounded process-group timeout; caps output; and stops on the first mismatch. Standard-library execution does not claim OS-level prevention of raw sockets or arbitrary absolute-path writes by an untrusted child. A profile is eligible only when trusted repository wrappers and the available tool environment make its declared offline/temp-only behavior auditable; if stronger sandbox enforcement is required but unavailable, the result is `UNVERIFIED`. Missing profile, binding/blob/hash/step mismatch, process error, timeout, unsafe path/link, dirty repository, cleanup failure, or overflow also returns `UNVERIFIED`.

For every future business module, profile definition and trusted wrapper are owned by a separate preceding control-plane change. That control-plane candidate must receive independent exact-SHA verification, integrate/archive into local main, and be present when the business module freezes its runner base. The business module is forbidden to define or modify its own judge profile. After the immutable business candidate receives independent verification and pure-FF integration/archive, only the integrator may append its exact-target binding; a different clean-detached verifier must validate that exact binding-head before a later receipt commit. A second clean-detached verifier derives the receipt-head result, and only then may the next module freeze a base. Binding and receipt commits never edit the candidate. This avoids a self-judging module and self-hashing profile.

### 3. Use only deterministic profile replay

The four predeclared profiles are:

1. `self-evolution-v1`: at `7a5e8bb261b994d68ce9af5eada347df6700c490`, run only repository-contained `verify_contract.py` and `run_checker_regressions.py`, requiring contract PASS plus all seven positive/seven negative checker cases. User-directory `quick_validate`, OpenSpec CLI, Go/Node toolchains, and fresh-session actor independence are excluded from `MECHANICAL_PASS` because they are environment/actor dependent.
2. `old-menu-artifact-fail-v1`: at `6d77bdd6319722b7c71b4726c6159955da9a84b6`, run a trusted Python structural parser over exact Git blobs. It must reproduce fingerprint `artifact-consistency|semantic|proposal-stale-draft-not-run|6d77bdd|verifier` from conflicting proposal/checkpoint/tasks fields, emit `EXPECTED_MECHANICAL_FAIL`, and mark all later Gates `NOT_RUN`. It must not transport the historical `rg` expression, backticks, or receipt text through a shell.
3. `menu-supersession-v1`: at `109c8e828f6f5a10adff33ccdb73d4fd784b2f3d`, run repository `verify_supersession.py`, `npm test --prefix apps/wechat-miniprogram` with exact 13/13 assertions, and the two provider Go packages. Trusted repository wrappers additionally perform the exact-base two-file-overlay legacy Red, JS51/JSON43/dependencies0 static audit, strict artifact structure without depending on mutable OpenSpec CLI behavior, owned/protected/sensitive scope checks, and fixed tree identities. Every subcheck is argv-only and bounded.
4. `lifecycle-receipt-control-v1`: predeclared in this candidate but exact-target-bound only after this change archives. It targets this exact candidate and fixes all repository-contained parser/Git/registry/executor/profile/runner/forward positive and negative suites, the three historical profile replays and chain, false-attestation rejection, exact receipt markers, and non-weakening route/Gate checks. It excludes user-directory `quick_validate`, mutable OpenSpec CLI behavior, Git author/session identity, network, UI2/UI3, and real order/payment. Python/Git/Node/npm version mismatch, unavailable complete local Go cache, or a profile requiring unavailable stronger OS isolation returns `UNVERIFIED`.

Tool availability is explicit: an absent/incompatible Python, Git, Node, npm, or Go toolchain makes its profile `UNVERIFIED`, never PASS. For Go steps the trusted wrapper first validates the exact already-installed Go version. It obtains `go env GOMODCACHE`, resolves and validates the local `cache/download` directory and every Go-requested cache path against unsafe links, and constructs the Go child environment from an empty map: temp-only HOME/TMP/cache, exact tool PATH, `GOENV=off`, `GOTOOLCHAIN=local`, `GOSUMDB=off`, and no inherited proxy, private-module, VCS or credential variables. In the validated `file://` phase exact Go derives the MVS/pruned build list, then derives and materializes the actual two-target test package/module closure; the wrapper validates required package-closure `.mod`, `.zip`, and `.ziphash` artifacts but does not invent metadata, require unrelated cached `.info`, recursively interpret dependency requirements, or explicitly download non-package-closure zips. It then sets `GOPROXY=off`, runs `go mod verify`, and runs the two package tests from the temporary cache. Any Go-native lookup failure, required artifact absence, unsafe link, version mismatch, checksum failure, or attempted fallback yields `UNVERIFIED`; it never downloads a toolchain or contacts a network proxy.

### 4. Receipt and Git history remain append-only

Each archive has exactly one `lifecycle-receipt.md`. It stores `receipt_head_verification=REQUIRED_DERIVED`; the checker derives the unique Git add commit, requires the archive ancestor, rejects later edits, and never writes the derived head/PASS back.

The structural schema still verifies exact candidate/integrated/archive objects, ancestry, archive paths/diffs, owned paths, candidate checkpoint/tasks bytes/digests and open tasks, reciprocal links, product-tree identity, retrospective, and receipt history. Structural validity alone cannot produce mechanical PASS.

### 5. Preserve three historical conclusions and one layered delivery result

| change | exact candidate | archive | receipt-stored expected history | checker-derived recovery |
|---|---|---|---|---|
| `enable-run-loop-self-evolution` | `7a5e8bb261b994d68ce9af5eada347df6700c490` | `94e04bf26e37e93299c26ef2c9c8aa7552619444` | saved independent PASS, audit only | `MECHANICAL_PASS`; actor independence not re-proved |
| `connect-miniprogram-menu-catalog` | `6d77bdd6319722b7c71b4726c6159955da9a84b6` | `7d01fe22ded67aeded78cb7d03de87aa12416ada` | exact semantic FAIL | `EXPECTED_MECHANICAL_FAIL`; later Gates `NOT_RUN` |
| `supersede-miniprogram-catalog-evidence` | `109c8e828f6f5a10adff33ccdb73d4fd784b2f3d` | `510a84baddfb5b61200c159fd2041b7c512a92db` | saved independent PASS, audit only | `MECHANICAL_PASS`; actor independence not re-proved |

Menu delivery recovery may say only `mechanically_reproducible=true` when the old immutable FAIL, reciprocal links, replacement mechanical PASS, exact integration/archive facts, three receipt histories, and equal app/catalog/httpapi trees all agree. The old verdict remains `FAILED`; no output says it or the recorded attestations are `INDEPENDENT_VERIFIED`.

### 6. Bootstrap this change without circular proof

This change creates the registry/executor and is the sole bootstrap exception to the preceding-control-plane rule. It predeclares `lifecycle-receipt-control-v1` but cannot use its not-yet-integrated binding to prove its candidate. It must first pass the existing full writer Gate and a different clean-detached exact-SHA `$order-verify-change` session. Only after authorized pure-FF integration and archive may the integrator append the exact target/tool binding. A different clean-detached verifier executes `lifecycle-receipt-control-v1` and validates that binding-head before a later receipt commit; another clean-detached verifier then derives the receipt-head result. This does not waive candidate, binding-head, or receipt-head independence and cannot be reused by future business modules.

### 7. Keep the runner thin and non-weakening

`SKILL.md` only routes archive closure to the receipt checker/profile registry. Detailed trust, replay, supersession, and bootstrap rules remain one level down in `references/self-evolution.md`. Existing lifecycle states, four stage handlers, lane/retry/score/hard Gates, authorization, and next-module activation remain unchanged.

## Risks / Trade-offs

- [Mechanical replay is not actor identity] -> output names and docs state this explicitly; fresh independent verifier remains mandatory.
- [Historical tools may be unavailable] -> fail closed as `UNVERIFIED`; never substitute provenance or a weaker command.
- [Profiles can become a new judge surface] -> predeclare definitions, exact-bind targets, hash tool sources, use fixed wrappers, and independently review every registry change.
- [Self-bootstrap is circular] -> use one existing independent verifier first; activate replay only from the integrated fixed blob.
- [Standard library cannot prove OS-level network/filesystem isolation] -> use trusted wrappers, offline/temp-routed environment, bounded execution and fail `UNVERIFIED` whenever a profile requires stronger unavailable isolation.

## Migration Plan

After explicit re-approval: discard prior Green claims; add false-green Red tests; implement the registry/executor/schema/checker and all four profile definitions/wrappers; replay the three historical profiles; write the three historical receipts with derived markers only; run full writer Gate and commit one candidate. A different verifier runs the exact candidate. After authorized pure-FF integration/archive, the integrator appends the `lifecycle-receipt-control-v1` exact binding, an independent verifier passes the binding-head, a later commit adds the receipt, and another independent verifier derives the receipt-head. Rollback uses ordinary revert commits and never rewrites history.

## Open Questions

None.
