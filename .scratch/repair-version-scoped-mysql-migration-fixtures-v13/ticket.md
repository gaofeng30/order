# Ticket: version-scoped MySQL migration fixtures

## Problem

Historical integration tests currently equate their frozen schema target with the repository-wide migration count. Once v12/v13 were added, fresh MySQL runs fail before exercising the historical v10/v11 contracts.

## Required behavior

1. Catalog, identity, and menu historical schema/upgrade scenarios run only the validated v1-v10 prefix.
2. Storefront's frozen schema contract runs only the validated v1-v11 prefix.
3. Prefix selection fails closed unless the loaded set contains the required index and the migration at that index has exactly the required version.
4. Later v12/v13/future migrations do not change historical `FromVersion`, `ToVersion`, `AppliedCount`, repeat, history, or checksum-drift assertions.
5. Current repository/HTTP helpers that need the latest schema still run the entire loaded migration set.
6. No code outside the declared owned paths changes; the final diff is test/evidence only.
7. `migrate/mysql_integration_test.go` is deliberately different from historical fixtures: its `wantNames` is the strict current-all ledger and must name v1-v13 exactly, including storefront settings, merchant accounts, and merchant action audits. Count, order, checksums, apply, repeat, and current-state assertions remain strict rather than becoming dynamic.

## Acceptance

- Exact-base Red is reproduced on a fresh loopback-only MySQL 8.0.46 container.
- The same focused suites and race pass after the minimal change.
- The migrate current-all ledger first reproduces `13 vs 10`, then passes with the exact v11-v13 names appended.
- Adjacent migrate/merchantidentity/storefront/wechatpay checks prove the full current v1-v13 chain still works.
- Mutants for v10 widening, v11 removal, required-version-guard deletion, and current-all v13 ledger removal are killed.
- Full `services/api` test/race/vet/build/smoke, format/diff/scope, and clean checks pass.
- One Chinese candidate commit receives two-axis review and a fresh detached exact-SHA verification.
