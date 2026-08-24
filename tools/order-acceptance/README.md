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

At integration base `247d84d8816346ab665174594d7f62cc7b41e6f3`, structural mapping finds 95/95 cases and 23 separate L4 blocks. `TestAcceptanceLocalThreeRoleOrderToRefund` adds one real L2 selector across the root HTTP router, workers, a fresh v1-v44 MySQL schema and deterministic local providers. It closes the exact L2 requirements for `BE-11`, `BE-15`, `INV-08`, `INV-10`, and `INV-13`; it does not claim UI1/L3 or broader boundary variants.

`TestAcceptanceImportBoundariesAreDurable` adds one root-composed OWNER PC multipart selector on a fresh v1-v44 schema. It closes L2 for `BE-27` through `BE-33`, including failure zero writes, durable batches/business rows, replay conflict shields and audit uniqueness. Their L3 requirement remains `MISSING`; the 11-check PC import runner stays supporting-only because it does not render every boundary.

Ten root-composed rendered selector profiles are now inventoried: Mini's four-scenario success profile plus pending-payment, merchant, broad user-boundary and exact BE-22/BE-26 profiles; PC's read, CRUD, import, catalog/image and transaction profiles. The Merchant Mini selector stays supporting-only because its one happy fulfillment path does not close the broader lane/search, manual/cross-date/replay/refunded, notification-failure and tomorrow-isolation requirements. Its integrated TDD receipt preserves the pre-fix real `400` for a missing scan idempotency key; the current selector proves the keyed atomic scan and requires exact HTTP cleanup of saved settings and sold-out state.

`TestAcceptanceUserBoundariesAreFailClosed` crosses the root HTTP seam on fresh v1-v44 MySQL and exactly closes the L2 requirements of `BE-04`, `BE-06` and `BE-23`. The locked-Chrome user-boundary selector closes L3 for `BE-01` through `BE-06` and `BE-23` through `BE-25`; its substituted facts remain supporting-only for `BE-22` and `BE-26`. The later exact selector candidate `f33139fd3a308cb00b2d221109773fb026ffa1b3` passed both writer and detached Chrome 151 runs and was rerun after integration as `8abdcd45b15000a1cd146149f19e7f28798f7aee`: each receipt used a fresh v44 schema and recorded 65 root-HTTP responses. It starts with a real unbound session, proves denied/failed phone authorization has zero bind/Quote/prepay/navigation side effects before accepted binding resumes the same checkout, and proves no-READY/READY/token/401/503/replay/completion shields. Therefore it closes local L3 for `BE-22` and `BE-26`; BE-22's real WeChat phone evidence remains L4 `BLOCKED_EXTERNAL`.

The 26-check PC catalog/image selector is `satisfies:true` only for `PAGE-PC06` and `BE-20`: it renders every category CRUD/order/state action with duplicate/FK shields, and it toggles today's sold-out state while reading tomorrow unchanged. It remains supporting-only for `PAGE-PC05`, `PAGE-PC08` and `AC-17` because their product-edit/upload-failure, unreadable-object display, or rendered Mini cross-client protections are not all exercised.

The PC transaction candidate `f998190ec508a0b5b7b6d6ed6019eb8a9e5a49d4` passed writer and detached Chrome 151 runs with all 23 checks and was integrated as the identical tree in `247d84d8816346ab665174594d7f62cc7b41e6f3`. It is inventoried for `PAGE-PC01` through `PAGE-PC04` but remains `satisfies:false`: PC01 does not independently assert query non-mutation or unclaimed-revenue exclusion; PC02 does not exercise date, pickup-number and phone searches; PC03 does not prove fake-bill single-sided/unavailable reconciliation or its required L2; PC04 does not render the corrupt-snapshot/no-abnormal-order shield. The existing SUBACCOUNT authorization selector is L2, not UI1, and real WeChat refund/bill evidence remains L4.

Seventeen cases have complete executable selectors for every local level assigned by the matrix: `PAGE-PC06`, `BE-01` through `BE-06`, `BE-11`, `BE-20`, `BE-22` through `BE-26`, `INV-03`, `INV-08`, and `INV-12`. They remain `AVAILABLE_NOT_RUN` until an exact-SHA run. The other 78 cases remain `MISSING`; most require composed MySQL-backed UI1, while the manifest preserves narrower L1/L2 gaps where present.
