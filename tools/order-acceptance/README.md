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

At integration base `4eceab6c33baf1d87b057edddce1e4b39b24e5fb`, structural mapping finds 95/95 cases and 23 separate L4 blocks. `TestAcceptanceLocalThreeRoleOrderToRefund` adds one real L2 selector across the root HTTP router, workers, a fresh v1-v44 MySQL schema and deterministic local providers. It closes the exact L2 requirements for `BE-11`, `BE-15`, `INV-08`, `INV-10`, and `INV-13`; it does not claim UI1/L3 or broader boundary variants.

`TestAcceptanceImportBoundariesAreDurable` adds one root-composed OWNER PC multipart selector on a fresh v1-v44 schema. It closes L2 for `BE-27` through `BE-33`, including failure zero writes, durable batches/business rows, replay conflict shields and audit uniqueness. Their L3 requirement remains `MISSING`; the 11-check PC import runner stays supporting-only because it does not render every boundary.

Seven root-composed rendered selector profiles are now inventoried: Mini's four success-mode user scenarios, separate pending-payment and merchant scenarios, plus PC's read, CRUD and import flows and the PC catalog/image flow. The Merchant Mini selector stays supporting-only because its one happy fulfillment path does not close the broader lane/search, manual/cross-date/replay/refunded, notification-failure and tomorrow-isolation requirements. Its integrated TDD receipt preserves the pre-fix real `400` for a missing scan idempotency key; the current selector proves the keyed atomic scan and requires exact HTTP cleanup of saved settings and sold-out state.

The 26-check PC catalog/image selector is `satisfies:true` only for `PAGE-PC06` and `BE-20`: it renders every category CRUD/order/state action with duplicate/FK shields, and it toggles today's sold-out state while reading tomorrow unchanged. It remains supporting-only for `PAGE-PC05`, `PAGE-PC08` and `AC-17` because their product-edit/upload-failure, unreadable-object display, or rendered Mini cross-client protections are not all exercised.

Only `PAGE-PC06`, `BE-11`, `BE-20`, `INV-03`, `INV-08`, and `INV-12` have complete executable selectors for every local level assigned by the matrix, and those six are still `AVAILABLE_NOT_RUN` until an exact-SHA run. The other 89 cases remain `MISSING`; most require composed MySQL-backed UI1, while the manifest preserves narrower L1/L2 gaps where present.
