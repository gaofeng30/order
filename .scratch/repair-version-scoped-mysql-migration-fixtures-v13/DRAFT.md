# DRAFT: repair-version-scoped-mysql-migration-fixtures-v13

## Fixed point

- `base_sha`: `b7f484f54decfa38bc36bbed1ada041828d87483`
- source branch containing the base: `codex/order-delivery-integration`
- writer branch: `codex/repair-version-scoped-mysql-migration-fixtures-v13`
- gate: `W3`
- UI target / actual: `UI0` / `UI0`
- owner: independent writer for this change
- dependency: the exact base already contains ordered migrations v1-v13
- external asset: disposable loopback-only `mysql:8.0.46-oraclelinux9`; owner is the writer/verifier; recovery is to start a fresh isolated container and rerun the declared command

## Goal and seam

The seam is the real MySQL integration-test boundary in the five owned package test files. Historical migration scenarios must run against the schema version they specify, while current repository/HTTP setup and the migrate ledger continue to load every embedded migration.

## Owned paths

- `.scratch/repair-version-scoped-mysql-migration-fixtures-v13/**`
- `services/api/internal/catalog/mysql_integration_test.go`
- `services/api/internal/identity/mysql_integration_test.go`
- `services/api/internal/menu/mysql_integration_test.go`
- `services/api/internal/storefront/mysql_integration_test.go`
- `services/api/internal/migrate/mysql_integration_test.go`

Everything else is read-only. In particular: production Go, `services/api/migrations/**`, migrate implementation and tests other than the one named owned file, merchantidentity, wechatpay, paymentobservation, router/main/apps, `go.mod`, and `go.sum`.

## Historical slice versus latest mapping

| Package / scenario | Migration set | Frozen assertions |
| --- | --- | --- |
| catalog `TestCatalogSchemaIntegration` | exact v1-v10 prefix | v1→v10, applied 9, repeat 10→10/0, history/checksum drift |
| catalog `applyCatalogMigrations` used by repository/HTTP/snapshot tests | all loaded migrations | latest schema, no frozen total |
| identity session schema and phone v9→v10 upgrade | exact v1-v10 prefix | 0/1→10 and 9→10 counts/repeat |
| identity phone-status repository/HTTP setup | all loaded migrations | latest schema, no frozen repository total |
| menu historical v3→v10 upgrade | exact v1-v10 prefix | 3→10/applied 7 and repeat 10→10/0 |
| storefront v11 schema/repository/HTTP contract | exact v1-v11 prefix | 0→11/applied 11 and repeat 11→11/0 |
| migrate `loadEmbeddedMigrations` and `TestMySQL8Integration` | strict current-all v1-v13 ledger | exact count, ordered names, checksum-backed apply/repeat/current-state behavior |

Every prefix loader first loads the repository set, verifies `len(set) >= requiredVersion`, verifies `set[requiredVersion-1].Version == requiredVersion`, and only then slices. It never asserts that the repository total is 13.

## Non-goals

- No migration count/order/content change.
- No production abstraction, compatibility path, skip, mock, or environment bypass.
- No push, PR, integration, deployment, or external-system write.

## Red → Green → Refactor

- Red: on fresh loopback MySQL 8.0.46 at the exact base, run the historical focused suites and current-all migrate suite; capture the first `migration count=13, want 10/11` and `embedded migration count=13, want 10` failures and exit codes.
- Green: introduce only test-local fixed-prefix helpers and route historical scenarios through v10/v11 while keeping current repository/HTTP setup on all migrations; rerun the identical focused command.
- Refactor: retain the smallest readable helper shape, `gofmt`, and rerun focused `-race`, adjacent real-MySQL suites, mutation checks, full API gates, protected-path checks, and diff checks.

## Commands

- Writer Red/Green/focused: fresh loopback MySQL with the four historical/latest package suites plus `go test -race ./services/api/internal/migrate -run '^TestMySQL8Integration$' -count=1 -timeout=8m`. The migrate suite remains an exact current-all ledger, unlike the frozen historical prefixes.
- Adjacent real MySQL: `go test -race ./services/api/internal/migrate ./services/api/internal/merchantidentity ./services/api/internal/storefront ./services/api/internal/wechatpay -count=1 -timeout=8m` using the same fresh container and required environment.
- Full Gate: `go test ./services/api/...`, `go test -race ./services/api/...`, `go vet ./services/api/...`, `go build ./services/api/...`, and `bash services/api/scripts/smoke.sh`, with repository-pinned `GOPROXY=off GOTOOLCHAIN=go1.26.5`.
- Formatting/scope: `test -z "$(gofmt -l services/api/internal/catalog/mysql_integration_test.go services/api/internal/identity/mysql_integration_test.go services/api/internal/menu/mysql_integration_test.go services/api/internal/storefront/mysql_integration_test.go services/api/internal/migrate/mysql_integration_test.go)"`, `git diff --check`, `git diff --name-only`, protected path audit, and clean status.
- Mutation: independently widen v10 prefix, remove v11 prefix, delete the required-version guard, and remove v13 from the current-all ledger; the focused W3 command must fail for each mutant, then restore and clean all temporary state.

## Writer score

- `C=10`: historical and current-all semantics are explicitly separated and all original assertions remain.
- `T=10`: exact-base Red, real-MySQL Green/race, current v1-v13 adjacency, four killed mutants, and full regressions are executable.
- `V=8`: writer Gate is complete; exact-SHA detached verification is required before independent PASS.
- `R=8`: the change is test-only, revertable as one commit, and every disposable database resource has bounded cleanup.
- Total: `36/40`; hard blockers: `0` before candidate creation.
- Reviewer: `git diff b7f484f54decfa38bc36bbed1ada041828d87483...HEAD` and `git log b7f484f54decfa38bc36bbed1ada041828d87483..HEAD --oneline`, with Standards and Spec agents in parallel.
- Verifier: create a new detached worktree at the exact candidate SHA and rerun the same W3, full Gate, formatting/scope/protected/clean checks from scratch.
