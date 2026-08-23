# Tasks: repair-version-scoped-mysql-migration-fixtures-v13

- [x] Pin exact base, prove target branch absent, create writer branch, and confirm clean worktree.
  - Evidence: `HEAD=b7f484f54decfa38bc36bbed1ada041828d87483`; branch `codex/repair-version-scoped-mysql-migration-fixtures-v13`; initial status clean.
- [x] Declare W3/UI0, owned/read-only paths, test seam, historical/latest mapping, RGR, writer/reviewer/verifier commands.
  - Evidence: `DRAFT.md` and `ticket.md`.
- [x] Record the approved ownership extension for `services/api/internal/migrate/mysql_integration_test.go` and its distinct strict current-all semantics.
  - Evidence: DRAFT mapping and ticket requirement 7; no production or migration ownership added.
- [x] Reproduce the first exact-base Red on fresh loopback MySQL 8.0.46.
  - Evidence: `TestCatalogSchemaIntegration`; MySQL `8.0.46`; loopback binding; `mysql_integration_test.go:32: migration count = 13, want 10`; `go test` exit `1`; disposable container and credential file cleaned.
- [x] Apply the smallest test-only version-prefix repair.
  - Green evidence: fresh MySQL `8.0.46`; four package focused and focused `-race` PASS; catalog/identity/menu historical v10, storefront historical v11, catalog/identity current repository setup latest.
- [x] Reproduce and repair the migrate current-all ledger Red without weakening exact count/order/checksum behavior.
  - Red evidence: `TestMySQL8Integration`; MySQL `8.0.46`; loopback binding; `mysql_integration_test.go:38: embedded migration count = 13, want 10`; `go test -race` exit `1`; disposable container and credential file cleaned.
  - Green evidence: exact names v11-v13 appended; fresh MySQL current-all suite PASS without changing its count/order/checksum/apply/repeat assertions.
- [x] Run focused Green and race on fresh MySQL.
  - Evidence: same disposable MySQL `8.0.46` container; focused non-race and race PASS; current-all migrate race PASS; container and credential file cleaned.
- [x] Run adjacent full-chain real-MySQL regression.
  - Evidence: same fresh MySQL; migrate/merchantidentity/storefront/wechatpay `go test -race` PASS, proving current v1-v13 chain remains executable.
- [x] Kill required v10/v11/guard/current-ledger mutants and clean temporary state.
  - Evidence: v10 expansion failed catalog v1→v10 assertion; v11 prefix removal produced 0→13/13 instead of 0→11/11; guard deletion was killed by missing-version guard test; v13 ledger removal failed `13, want 12`. Each used fresh loopback MySQL 8.0.46, exited nonzero, was restored immediately, and cleaned its container/credential file.
- [x] Run all `services/api` test/race/vet/build/smoke and formatting/scope checks.
  - Evidence: fresh MySQL full `go test` and `go test -race` PASS; `go vet`, `go build`, and controlled smoke PASS; `gofmt`, `git diff --check`, test-only owned paths, protected paths, and temporary-resource residue checks PASS.
- [x] Commit only owned files with a Chinese message and record the exact candidate SHA.
  - Evidence: this tasks snapshot is part of the single candidate commit; the resulting full SHA is recorded in the immutable review/verifier handoff.
- [ ] Run parallel Standards/Spec review against the exact base.
- [ ] Verify the exact candidate from a fresh clean detached worktree.

## Evidence records

All records are sanitized; the temporary password value and file contents were never emitted.

```yaml
change: repair-version-scoped-mysql-migration-fixtures-v13
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: b7f484f54decfa38bc36bbed1ada041828d87483
candidate_sha: not-yet-created
phase: writer
command_or_action: git rev-parse HEAD; git status --short --branch; git branch --list codex/repair-version-scoped-mysql-migration-fixtures-v13; git switch -c codex/repair-version-scoped-mysql-migration-fixtures-v13
exit_result: 0
sanitized_summary: exact base, clean detached start, absent target branch, and isolated writer branch confirmed
artifact_or_environment: /Users/vivix/.codex/worktrees/1979/order
unverified_boundary: no implementation or runtime behavior verified by setup
external_asset:
  owner: N/A
  missing: N/A
  recovery: N/A
```

```yaml
change: repair-version-scoped-mysql-migration-fixtures-v13
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: b7f484f54decfa38bc36bbed1ada041828d87483
candidate_sha: not-yet-created
phase: red
command_or_action: fresh loopback MySQL 8.0.46 with seven ORDER_TEST_MYSQL variables; go test -race ./services/api/internal/catalog -run '^TestCatalogSchemaIntegration$' -count=1 -timeout=3m
exit_result: 1
sanitized_summary: mysql_integration_test.go:32 migration count = 13, want 10
artifact_or_environment: disposable mysql:8.0.46-oraclelinux9 with random loopback port; container and credential file cleaned
unverified_boundary: this Red proves only the historical catalog fixture failure
external_asset:
  owner: writer
  missing: N/A
  recovery: start a new disposable loopback-only MySQL 8.0.46 container
```

```yaml
change: repair-version-scoped-mysql-migration-fixtures-v13
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: b7f484f54decfa38bc36bbed1ada041828d87483
candidate_sha: not-yet-created
phase: red
command_or_action: fresh loopback MySQL 8.0.46 with seven ORDER_TEST_MYSQL variables; go test -race ./services/api/internal/migrate -run '^TestMySQL8Integration$' -count=1 -timeout=8m
exit_result: 1
sanitized_summary: mysql_integration_test.go:38 embedded migration count = 13, want 10
artifact_or_environment: disposable mysql:8.0.46-oraclelinux9 with random loopback port; container and credential file cleaned
unverified_boundary: this Red proves only the strict current-all ledger omission
external_asset:
  owner: writer
  missing: N/A
  recovery: start a new disposable loopback-only MySQL 8.0.46 container
```

```yaml
change: repair-version-scoped-mysql-migration-fixtures-v13
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: b7f484f54decfa38bc36bbed1ada041828d87483
candidate_sha: not-yet-created
phase: green
command_or_action: .scratch/repair-version-scoped-mysql-migration-fixtures-v13/verify-mysql.sh w3; .scratch/repair-version-scoped-mysql-migration-fixtures-v13/verify-mysql.sh adjacent
exit_result: 0
sanitized_summary: historical v10/v11 focused and race PASS; migrate strict current-all plus migrate/merchantidentity/storefront/wechatpay race PASS
artifact_or_environment: fresh loopback-only MySQL 8.0.46 containers; all containers and credential files cleaned
unverified_boundary: local W3 does not prove production deployment or platform UI behavior
external_asset:
  owner: writer
  missing: N/A
  recovery: rerun both profiles with Docker and the pinned image available
```

```yaml
change: repair-version-scoped-mysql-migration-fixtures-v13
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: b7f484f54decfa38bc36bbed1ada041828d87483
candidate_sha: not-yet-created
phase: refactor
command_or_action: temporarily widen catalog v10, remove storefront v11 prefix, delete catalog required-version guards, and remove migrate v13 ledger name; run verify-mysql.sh catalog-focused/storefront-focused/migrate-focused; restore each mutant immediately
exit_result: 1 for every mutant, expected
sanitized_summary: catalog v1-to-v10 assertion, storefront 0-to-11 assertion, missing-version guard test, and current-all 13-vs-12 ledger each killed its mutant
artifact_or_environment: four fresh loopback-only MySQL 8.0.46 containers; test files restored; all containers and credential files cleaned
unverified_boundary: mutation proves sensitivity to the four named faults only
external_asset:
  owner: writer
  missing: N/A
  recovery: reapply one named mutant and rerun its focused profile
```

```yaml
change: repair-version-scoped-mysql-migration-fixtures-v13
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: b7f484f54decfa38bc36bbed1ada041828d87483
candidate_sha: not-yet-created
phase: writer
command_or_action: verify-mysql.sh full; GOPROXY=off GOTOOLCHAIN=go1.26.5 go vet ./services/api/...; GOPROXY=off GOTOOLCHAIN=go1.26.5 go build ./services/api/...; GOPROXY=off GOTOOLCHAIN=go1.26.5 bash services/api/scripts/smoke.sh
exit_result: 0
sanitized_summary: full services/api test and race, vet, build, and controlled smoke PASS
artifact_or_environment: fresh loopback-only MySQL 8.0.46 plus local Go 1.26.5 toolchain; disposable database resources cleaned
unverified_boundary: no external UI, push, PR, integration, deployment, or production write performed
external_asset:
  owner: writer
  missing: N/A
  recovery: rerun full profile and local commands from the exact candidate
```

```yaml
change: repair-version-scoped-mysql-migration-fixtures-v13
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: b7f484f54decfa38bc36bbed1ada041828d87483
candidate_sha: not-yet-created
phase: writer
command_or_action: gofmt -l on five exact owned Go files; bash -n verify-mysql.sh; git diff --check; explicit allowed-path and protected-path audit; Docker and credential residue audit; git status --short
exit_result: 0
sanitized_summary: formatting, shell syntax, diff, test-only ownership, protected paths, residue, and pre-commit scope PASS
artifact_or_environment: writer worktree and local Docker inventory
unverified_boundary: clean status is asserted after the candidate commit, not by this pre-commit record
external_asset:
  owner: N/A
  missing: N/A
  recovery: rerun the same read-only audit
```
