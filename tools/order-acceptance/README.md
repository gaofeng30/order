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

`validate` checks structure/source binding and may exit zero while local gaps remain. `inventory` and `run` emit one JSON object per CaseID plus one summary. They exit nonzero unless all 95 local requirements pass. A run also fails when the worktree is dirty, an exact test selector disappears, a required test skips, or any required command exits nonzero. L4 rows stay separately visible as `BLOCKED_EXTERNAL` and do not satisfy or invalidate L1-L3.

The runner starts no Docker container and installs no dependency. It inherits the existing Go/Node/Chrome/MySQL environment and therefore reuses the machine's caches and the single already-running MySQL gate. Fresh-MySQL tests require the repository's complete `ORDER_TEST_MYSQL_*` environment; absence causes a skip, which the harness rejects.

## Base inventory

At integration base `3b83d0515a03769ae2ab034fcde6b9eea453aa07`, structural mapping finds 95/95 cases and 23 separate L4 blocks. `TestAcceptanceLocalThreeRoleOrderToRefund` adds one real L2 selector across the root HTTP router, workers, a fresh v1-v44 MySQL schema and deterministic local providers. It closes the exact L2 requirements for `BE-11`, `BE-15`, `INV-08`, `INV-10`, and `INV-13`; it does not claim UI1/L3 or broader boundary variants.

Three root-composed rendered selectors are now inventoried as L3 supporting evidence: Mini's four user scenarios, PC's 25-check read smoke, and PC's 21-check CRUD flow. None is marked `satisfies:true`: the Mini selector omits several row-specific boundary/failure paths, the PC read selector mostly proves navigation and GET availability, and the PC write selector does not assert every invalid-input, ordering, upload, FK, RBAC, session or cross-client shield required by any one mapped row.

Only `BE-11`, `INV-03`, `INV-08`, and `INV-12` have complete executable selectors for every local level assigned by the matrix, and those four are still `AVAILABLE_NOT_RUN` until an exact-SHA run. The other 91 cases remain `MISSING`; most require composed MySQL-backed UI1, while the manifest preserves narrower L1/L2 gaps where present.
