# Order 95-case local acceptance harness

This harness binds the canonical `ORDER-MVP-R2.2` matrix to executable evidence without changing the matrix. It does not treat module tests, UI0/static checks, loopback-only UI1, fake-provider availability, or `BLOCKED_EXTERNAL` as a passed end-to-end case.

## Files

- `manifest.json`: all 95 unique cases, their role/UI/HTTP/MySQL/expected/failure scenario, required local levels, executable selectors already present, and the shortest missing selector/scenario.
- `coverage.mjs`: reviewed mapping from existing selectors to cases. `satisfies:false` means useful supporting evidence only.
- `generate.mjs`: deterministic matrix-to-manifest generator. Regenerate only when the frozen matrix or reviewed coverage mapping changes.
- `run.mjs`: fail-closed validator/inventory/exact-SHA runner.

## Commands

```sh
node tools/order-acceptance/generate.mjs "$(git rev-parse HEAD)"
node tools/order-acceptance/run.mjs validate
node tools/order-acceptance/run.mjs inventory \
  --output .scratch/overnight-acceptance/inventory.jsonl
node tools/order-acceptance/run.mjs run \
  --candidate-sha "$(git rev-parse HEAD)" \
  --output .scratch/overnight-acceptance/evidence.jsonl
```

`validate` checks structure/source binding and exits zero when that structure is valid. Its `structure.local_available` count means selectors are bound, not executed; the separate `inventory` field therefore reports those rows as `local_not_run` with `ok:false`. `inventory` and `run` emit one JSON object per CaseID plus one summary. They exit nonzero unless all 95 local requirements pass. A run also fails when the worktree is dirty, an exact test selector disappears, a required test skips, or any required command exits nonzero. L4 rows stay separately visible as `BLOCKED_EXTERNAL` and do not satisfy or invalidate L1-L3.

The runner starts no Docker container and installs no dependency. It inherits the existing Go/Node/Chrome/MySQL environment and therefore reuses the machine's caches and the single already-running MySQL gate. Fresh-MySQL tests require the repository's complete `ORDER_TEST_MYSQL_*` environment; absence causes a skip, which the harness rejects.

## AC19 exact-integration inventory

At integration base `cefc290539b45abb0d7435dacae9345552f57157`, the reviewed mapping keeps all 95 canonical CaseIDs explicit and 23 L4 blocks separate. The refund/unclaimed dependency is pinned to independently verified candidate `74e558d74a0994600d5781ea0c2be99814a201dd` and its integration commit. Structural validation therefore reports 95 rows available and zero missing, but this is still `AVAILABLE_NOT_RUN`, not runtime PASS.

The historical refund Writer Gate remains recorded as provenance. Because that script's source-scope assertion is bound to its implementation base, the final ledger rebinds it to the integrated exact-HEAD runner `node tools/miniprogram-ui/run-ui1-refund-unclaimed-l3.mjs`. The same rule applies to earlier UI Writer Gates: `.scratch/overnight-acceptance/run-fresh-root-ui1.sh` provisions one fresh-v44 schema and one random private API for each non-self-contained integrated UI runner, then stops the API and drops the schema. `run-self-contained-ui1.sh` redirects mutable receipts to `/private/tmp` and removes only known untracked PC evidence paths, so exact-SHA execution cannot overwrite committed evidence or leave the candidate dirty.

The 95 available rows are backed by integrated exact selectors in four strict groups:

- Core L1/L2: fresh-v44 root HTTP selectors for authoritative identity/pricing/RBAC, payment/production/sequence/subscription, PC derived facts, imports and user boundaries.
- User L3: all nine rendered user pages plus staff/profile, detail-media, user boundaries, exact BE-22/BE-26 and transaction/order gates.
- Merchant L3: complete merchant pages, cross-date redemption and injected failure gates.
- PC L3: PAGE-PC01–PC12 closure runners plus three-client source-of-truth coverage.

Only a clean candidate command `node tools/order-acceptance/run.mjs run --candidate-sha <exact SHA> --output .scratch/overnight-acceptance/<receipt>.jsonl` can change that conclusion to local PASS. It executes every strict selector, rejects skips and failed commands, and keeps the 23 real-platform/funds rows separately `BLOCKED_EXTERNAL`. Until that SHA-bound run passes, this inventory is not local completion and not submission readiness.
