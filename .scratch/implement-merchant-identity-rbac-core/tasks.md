# implement-merchant-identity-rbac-core evidence

## Checklist

- [x] fixed base/worktree/branch、v12/v13/merchantidentity 缺失、规则/PRD/skills/冻结 reference provenance 已只读核验。
- [x] shared router/main ownership 无当前冲突；冻结 `T1 -> T4` 顺序已记录。
- [x] DRAFT 已声明 W3/UI0、依赖、owned/read-only/non-goals、HTTP/schema/transaction/PII、seams、RGR、三阶段命令、外部资产与恢复。
- [x] strict HTTP vertical slices 完成真实 Red→Green。
- [x] v12/v13 migration chain/schema 完成真实 Red→Green。
- [x] merchant-login/identity transaction、audit、concurrency/fault recovery 完成真实 Red→Green。
- [x] transaction-bound AuthorizeInTx 与实时账号变化完成真实 Red→Green。
- [x] Refactor 后 focused/race/fresh MySQL/full/vet/build/smoke/gofmt/diff/owned/PII Gate PASS。
- [ ] 中文功能提交后 fixed-base Standards/Spec 双轴 Review PASS；finding replacement SHA 从头重验。
- [ ] 主控独立审计批准后，fresh detached exact-SHA verifier PASS。

## Evidence

仅追加已实际发生的脱敏决定性证据；禁止记录请求原文、手机号、姓名、openid、code/token、provider body、认证材料或数据库凭据。

```yaml
change: implement-merchant-identity-rbac-core
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: 122913c6bcc8c22acb73e05385d54449f27c2465
candidate_sha: e89a7561382b486c4958cceb455eb698736d129c
phase: red
command_or_action: static evidence-format checker requiring every candidate_sha to be either one complete SHA or not-yet-created
exit_result: FAIL
sanitized_summary: evidence contained non-canonical invalid-suffix and replacement-pending values; WIP SHA e89a7561382b486c4958cceb455eb698736d129c and its reviews were invalidated, with invalidity retained only in summaries
artifact_or_environment: exact externally pinned WIP SHA e89a7561382b486c4958cceb455eb698736d129c
unverified_boundary: evidence normalization, corrected migration Red/Green, complete Writer Gate, replacement review and detached verification remained pending
external_asset:
  owner: Writer
  missing: N/A
  recovery: retain pure complete SHA for committed historical records and use only not-yet-created before a replacement is formed
```

```yaml
change: implement-merchant-identity-rbac-core
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: 122913c6bcc8c22acb73e05385d54449f27c2465
candidate_sha: not-yet-created
phase: green
command_or_action: same evidence-format checker after canonicalizing all candidate_sha fields
exit_result: PASS
sanitized_summary: every evidence candidate_sha is now either a pure 40-character committed SHA or not-yet-created; invalid status remains only in sanitized summaries
artifact_or_environment: replacement Writer evidence
unverified_boundary: complete Writer Gate, replacement WIP SHA, review and detached verification remained pending
external_asset:
  owner: N/A
  missing: N/A
  recovery: rerun the format checker after every evidence edit
```

```yaml
change: implement-merchant-identity-rbac-core
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: 122913c6bcc8c22acb73e05385d54449f27c2465
candidate_sha: not-yet-created
phase: writer
command_or_action: post-evidence-remediation focused and race; anchored migration Green; fresh MySQL 8.0.46; repository-wide test and race; vet; build; smoke; static gates
exit_result: PASS
sanitized_summary: complete Writer Gate passed after authenticating the migration Red and canonicalizing every evidence SHA; all product and failure scenarios stayed green and temporary assets were cleaned
artifact_or_environment: final replacement WIP tree before commit
unverified_boundary: replacement clean HEAD still required two fresh review axes and controller review before Candidate designation; verifier remained prohibited
external_asset:
  owner: controller
  missing: review approval and Candidate designation
  recovery: commit the unchanged product plus corrected evidence, pin reviews externally to clean HEAD, and do not modify files after approval
```

```yaml
change: implement-merchant-identity-rbac-core
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: 122913c6bcc8c22acb73e05385d54449f27c2465
candidate_sha: 9cd2ffa330e40af05e91e5424b9ad252327b15f1
phase: red
command_or_action: static lifecycle assertion comparing the committed clean HEAD with ticket status and candidate declaration
exit_result: FAIL
sanitized_summary: ticket still claimed CANDIDATE_PENDING_COMMIT after the review target had already been committed; both review axes and SHA were invalidated before verifier
artifact_or_environment: exact WIP SHA 9cd2ffa330e40af05e91e5424b9ad252327b15f1
unverified_boundary: corrected WIP lifecycle, complete Writer Gate, replacement WIP SHA, both reviews and detached verification remained pending
external_asset:
  owner: Writer and controller
  missing: N/A
  recovery: declare IMPLEMENTING / REVIEWING_WIP with candidate not yet created; externally bind reviews to clean HEAD and designate that same HEAD only after both reviews and controller approval
```

```yaml
change: implement-merchant-identity-rbac-core
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: 122913c6bcc8c22acb73e05385d54449f27c2465
candidate_sha: not-yet-created
phase: green
command_or_action: same static lifecycle assertion after removing the premature Candidate claim and recording the commit self-reference boundary
exit_result: PASS
sanitized_summary: ticket now truthfully remains IMPLEMENTING / REVIEWING_WIP; review targets are externally pinned to clean HEAD, and no commit is asked to contain its own hash
artifact_or_environment: replacement Writer lifecycle artifacts
unverified_boundary: full Writer Gate, replacement WIP commit, two-axis review, controller designation and detached verification remained pending
external_asset:
  owner: controller
  missing: final Candidate designation after reviews
  recovery: if reviews pass, designate the unchanged clean HEAD as Candidate and only then authorize exact-SHA verifier
```

```yaml
change: implement-merchant-identity-rbac-core
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: 122913c6bcc8c22acb73e05385d54449f27c2465
candidate_sha: not-yet-created
phase: writer
command_or_action: post-lifecycle focused and race; fresh MySQL 8.0.46; repository-wide test and race; vet; build; smoke; format/diff/ownership/PII cleanup checks
exit_result: PASS
sanitized_summary: complete Writer Gate passed after correcting the WIP review lifecycle; product behavior remained unchanged and every disposable MySQL asset was cleaned
artifact_or_environment: final replacement WIP tree before commit
unverified_boundary: replacement clean HEAD still required fresh two-axis review and controller review before external Candidate designation; detached verifier did not start
external_asset:
  owner: controller
  missing: review approval and Candidate designation
  recovery: commit the WIP tree, pin reviews externally to that clean HEAD, and leave the tree unchanged if approved
```

```yaml
change: implement-merchant-identity-rbac-core
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: 122913c6bcc8c22acb73e05385d54449f27c2465
candidate_sha: dbdb508194a7e5524c3f5abdb98c60e5fb0b9878
phase: red
command_or_action: fixed-base zero-context shared-file ownership checker allowing only the v12/v13 wantNames additions in catalog/migrations_test.go
exit_result: FAIL
sanitized_summary: checker rejected an extra 78-line merchant schema and PII contract test in the shared catalog file; the old Candidate and both in-flight reviews were invalidated and stopped
artifact_or_environment: Writer tree and fixed-base catalog diff
unverified_boundary: replacement move, complete Writer Gate, replacement SHA, two-axis review and detached verification remained pending
external_asset:
  owner: Writer
  missing: N/A
  recovery: move the test unchanged into owned merchantidentity/migration_test.go, retain only two exact-list additions in catalog, then rerun every Writer Gate and both reviews
```

```yaml
change: implement-merchant-identity-rbac-core
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: 122913c6bcc8c22acb73e05385d54449f27c2465
candidate_sha: not-yet-created
phase: green
command_or_action: same fixed-base shared-file ownership checker plus focused TestMerchantIdentityMigrationContracts and TestCatalogMigrationSet after moving the contract test
exit_result: PASS
sanitized_summary: catalog/migrations_test.go now differs from fixed base only by the two permitted v12/v13 wantNames additions; the unchanged schema and PII assertions pass from owned merchantidentity/migration_test.go
artifact_or_environment: replacement Writer tree
unverified_boundary: complete fresh MySQL/full/race/vet/build/smoke/static Gate, replacement commit, review and detached verification remained pending
external_asset:
  owner: N/A
  missing: N/A
  recovery: rerun every declared Writer Gate before creating replacement SHA
```

```yaml
change: implement-merchant-identity-rbac-core
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: 122913c6bcc8c22acb73e05385d54449f27c2465
candidate_sha: not-yet-created
phase: writer
command_or_action: replacement focused and race; fresh MySQL 8.0.46; repository-wide test and race; vet; build; smoke; gofmt; fixed-base shared-file and owned-path checks
exit_result: PASS
sanitized_summary: moved migration contract retained behavior while catalog stayed within its exact two-line authorization; all focused/full/race/MySQL/tooling/smoke checks passed and disposable assets were cleaned
artifact_or_environment: replacement Writer tree after ownership correction
unverified_boundary: replacement commit, fresh fixed-base Standards/Spec review and approved detached exact-SHA verification remained pending
external_asset:
  owner: Writer/Verifier
  missing: none for local W3
  recovery: any further product/test/task/SHA change invalidates this run and requires the complete Gate again
```

```yaml
change: implement-merchant-identity-rbac-core
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: 122913c6bcc8c22acb73e05385d54449f27c2465
candidate_sha: not-yet-created
phase: writer
command_or_action: exact base, worktree branch, staging shared-path conflict, rules, PRD, skills, frozen exact-SHA T1 contract and missing-capability audit
exit_result: PASS
sanitized_summary: detached exact base was switched only in the visible worktree to the unique Writer branch; staging stayed clean; migration chain ended at v11 and neither v12/v13 nor merchantidentity existed; no current shared router/main delta conflicted
artifact_or_environment: /Users/vivix/.codex/worktrees/aa63/order
unverified_boundary: no product code, Red, Green, MySQL, review, candidate, detached verification, integration or external operation has run
external_asset:
  owner: Writer and workflow owner
  missing: fresh local MySQL execution and Matt tracker linkage
  recovery: run each TDD slice and a new isolated MySQL 8.0.46 container; configure tracker only after explicit user confirmation
```

```yaml
change: implement-merchant-identity-rbac-core
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: 122913c6bcc8c22acb73e05385d54449f27c2465
candidate_sha: not-yet-created
phase: green
command_or_action: same TestMerchantLoginRejectsInvalidUTF8BeforeAuthentication focused command after pre-decode UTF-8 validation
exit_result: PASS
sanitized_summary: invalid UTF-8 now returns exact INVALID_REQUEST before authentication or application access; accepted code bytes remain untrimmed
artifact_or_environment: Writer strict HTTP final slice
unverified_boundary: full Refactor and Writer gates were still required after this behavior fix
external_asset:
  owner: N/A
  missing: N/A
  recovery: rerun every focused, race, fresh-MySQL and repository-wide Writer check
```

```yaml
change: implement-merchant-identity-rbac-core
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: 122913c6bcc8c22acb73e05385d54449f27c2465
candidate_sha: not-yet-created
phase: refactor
command_or_action: gofmt; focused merchantidentity/httpapi/order-api/migrations/catalog tests; same focused race set; fresh verify-mysql.sh
exit_result: PASS
sanitized_summary: domain types and transaction authorizer were separated from HTTP and login persistence; strict HTTP, exact v1-v13, schema constraints, first/rejected/idempotent binding, same-code concurrency, live role/state, rollback, real deadlock, commit-unknown, caller-tx ordering and PII scenarios all stayed green
artifact_or_environment: final Writer product tree and disposable loopback MySQL 8.0.46 container; container and credential were cleaned
unverified_boundary: fixed-base commit, two-axis review and detached exact-SHA verification remained pending
external_asset:
  owner: Writer/Verifier
  missing: none for local W3
  recovery: rerun the same fresh-container suite for every product, test or candidate change
```

```yaml
change: implement-merchant-identity-rbac-core
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: 122913c6bcc8c22acb73e05385d54449f27c2465
candidate_sha: not-yet-created
phase: writer
command_or_action: go test ./services/api/...; go test -race ./services/api/...; go vet ./services/api/...; go build ./services/api/...; clean rerun of services/api/scripts/smoke.sh; gofmt; cached diff/owned/migration/PII/high-risk-material audits
exit_result: PASS
sanitized_summary: all Go packages passed normal and race tests; vet/build exited zero; clean smoke reported PASS; formatting/diff passed; v12/v13 remained one CREATE TABLE each with no v14 occupation; only declared owned paths changed; v13 audit had no sensitive fields and high-risk material count was zero
artifact_or_environment: final staged Writer tree; no Docker container, temporary credential, server process or smoke artifact remained
unverified_boundary: functional commit, fixed-base Standards/Spec review and fresh detached exact-SHA verification remained pending; no integration, push, PR, deploy, WeChat, Tencent Cloud or production action ran
external_asset:
  owner: workflow owner and external UAT owners
  missing: Matt tracker linkage and all T2/T5 platform assets, neither required for T1 Candidate
  recovery: configure tracker only after user confirmation; run external integration/UAT only under separate authorization
```

```yaml
change: implement-merchant-identity-rbac-core
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: 122913c6bcc8c22acb73e05385d54449f27c2465
candidate_sha: not-yet-created
phase: red
command_or_action: GOPROXY=off GOTOOLCHAIN=go1.26.5 go test ./services/api/internal/merchantidentity -run '^TestMerchantLoginRejectsInvalidUTF8BeforeAuthentication$' -count=1
exit_result: FAIL
sanitized_summary: invalid UTF-8 inside code was replacement-decoded and reached the application, returning 200 instead of exact INVALID_REQUEST
artifact_or_environment: Writer strict JSON self-review tracer
unverified_boundary: the defect changed provider input bytes even though all prior valid-UTF8 scenarios remained green
external_asset:
  owner: N/A
  missing: N/A
  recovery: reject non-UTF8 body bytes before JSON decoding and rerun the complete Refactor gate
```

```yaml
change: implement-merchant-identity-rbac-core
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: 122913c6bcc8c22acb73e05385d54449f27c2465
candidate_sha: not-yet-created
phase: red
command_or_action: GOPROXY=off GOTOOLCHAIN=go1.26.5 go test ./services/api/internal/httpapi -run '^TestRouterRegistersOnlyVersionedMerchantIdentityRoutes$' -count=1
exit_result: FAIL
sanitized_summary: root NewRouter accepted only the existing handlers and could not compile with the new merchant identity provider
artifact_or_environment: Writer router tracer at fixed base plus merchantidentity package
unverified_boundary: runtime dependency composition was still absent
external_asset:
  owner: N/A
  missing: N/A
  recovery: add exactly one merchant handler dependency and register only its two versioned routes
```

```yaml
change: implement-merchant-identity-rbac-core
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: 122913c6bcc8c22acb73e05385d54449f27c2465
candidate_sha: not-yet-created
phase: green
command_or_action: same TestRouterRegistersOnlyVersionedMerchantIdentityRoutes focused command after root router registration
exit_result: PASS
sanitized_summary: root router exposed exact GET identity and POST merchant-login while unversioned, broad and trailing paths remained 404
artifact_or_environment: Writer shared-router slice
unverified_boundary: order-api main had not yet composed repository, service and provider dependencies
external_asset:
  owner: N/A
  missing: N/A
  recovery: use the router signature change as the main composition Red
```

```yaml
change: implement-merchant-identity-rbac-core
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: 122913c6bcc8c22acb73e05385d54449f27c2465
candidate_sha: not-yet-created
phase: red
command_or_action: GOPROXY=off GOTOOLCHAIN=go1.26.5 go test ./services/api/cmd/order-api -count=1
exit_result: FAIL
sanitized_summary: cmd/order-api no longer compiled because main had not supplied the required merchant identity handler to NewRouter
artifact_or_environment: Writer runtime-composition tracer
unverified_boundary: no service startup or external listener was attempted
external_asset:
  owner: N/A
  missing: N/A
  recovery: compose merchant repository/service/handler with the existing session authenticator and primary-phone provider
```

```yaml
change: implement-merchant-identity-rbac-core
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: 122913c6bcc8c22acb73e05385d54449f27c2465
candidate_sha: not-yet-created
phase: green
command_or_action: fresh loopback MySQL 8.0.46 running the same race-enabled TestMerchantIdentityMySQL8Integration after the v13 FK/CHECK compatibility fix
exit_result: PASS
sanitized_summary: v1-v13 first/repeat migration and first merchant binding passed; primary-phone, account binding, record/auth version increments and full resolved success audit committed together; disposable assets were cleaned
artifact_or_environment: disposable mysql:8.0.46-oraclelinux9 schema and container
unverified_boundary: business rejection matrix, concurrency, authorization locks, recovery, router/main and complete gates remained pending
external_asset:
  owner: Writer/Verifier
  missing: none for local W3
  recovery: rerun the fresh-container script after every product or test change
```

```yaml
change: implement-merchant-identity-rbac-core
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: 122913c6bcc8c22acb73e05385d54449f27c2465
candidate_sha: not-yet-created
phase: green
command_or_action: same TestMerchantLoginBypassesPhoneProviderForExistingEnabledBinding focused command after minimal service/store/provider orchestration
exit_result: PASS
sanitized_summary: an existing enabled audited binding returned the current OWNER projection with zero provider, completion or recovery calls
artifact_or_environment: Writer application-service slice
unverified_boundary: concrete database transaction and fresh MySQL remained pending
external_asset:
  owner: N/A
  missing: N/A
  recovery: establish the repository behavior in a fresh MySQL tracer
```

```yaml
change: implement-merchant-identity-rbac-core
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: 122913c6bcc8c22acb73e05385d54449f27c2465
candidate_sha: not-yet-created
phase: red
command_or_action: fresh loopback mysql:8.0.46-oraclelinux9 running go test -race ./services/api/internal/merchantidentity -run '^TestMerchantIdentityMySQL8Integration$' -count=1
exit_result: FAIL
sanitized_summary: the W3 tracer stopped at compile because NewRepository was absent; the disposable container and credential were cleaned
artifact_or_environment: fresh Docker MySQL 8.0.46 and fixed-base Writer tree
unverified_boundary: migration application and first binding had not executed because persistence capability did not compile
external_asset:
  owner: Writer
  missing: none for local W3
  recovery: implement only the repository transaction required by first binding, then rerun the same fresh-container command
```

```yaml
change: implement-merchant-identity-rbac-core
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: 122913c6bcc8c22acb73e05385d54449f27c2465
candidate_sha: not-yet-created
phase: green
command_or_action: same TestMerchantLoginRejectsWhitespaceOnlyCodeBeforeAuthentication focused command after whitespace validation without trimming accepted codes
exit_result: PASS
sanitized_summary: whitespace-only code now returned exact INVALID_REQUEST before authentication while accepted codes remained byte-for-byte unchanged
artifact_or_environment: Writer strict HTTP slice
unverified_boundary: complete malformed-input matrix and durable login behavior remained pending
external_asset:
  owner: N/A
  missing: N/A
  recovery: proceed to the provider-bypass transaction orchestration tracer
```

```yaml
change: implement-merchant-identity-rbac-core
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: 122913c6bcc8c22acb73e05385d54449f27c2465
candidate_sha: not-yet-created
phase: red
command_or_action: GOPROXY=off GOTOOLCHAIN=go1.26.5 go test ./services/api/internal/merchantidentity -run '^TestMerchantLoginBypassesPhoneProviderForExistingEnabledBinding$' -count=1
exit_result: FAIL
sanitized_summary: LoginStart and Service did not exist, so provider-bypass orchestration did not compile
artifact_or_environment: Writer application-service tracer
unverified_boundary: repository transactions, audits, concurrency and failures remained absent
external_asset:
  owner: N/A
  missing: N/A
  recovery: add the minimal store/provider orchestration that returns an audited existing binding without provider access
```

```yaml
change: implement-merchant-identity-rbac-core
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: 122913c6bcc8c22acb73e05385d54449f27c2465
candidate_sha: not-yet-created
phase: green
command_or_action: same TestMerchantLoginEndpointReturnsExactBoundSubaccountProjection focused command after registering the strict POST route
exit_result: PASS
sanitized_summary: merchant-login accepted exact JSON and one bearer, passed the unmodified code and internal request ID, and returned only the SUBACCOUNT/auth_version projection with no-store
artifact_or_environment: Writer second HTTP vertical slice
unverified_boundary: strict invalid matrix and persistence remained pending
external_asset:
  owner: N/A
  missing: N/A
  recovery: drive malformed inputs independently before adding database behavior
```

```yaml
change: implement-merchant-identity-rbac-core
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: 122913c6bcc8c22acb73e05385d54449f27c2465
candidate_sha: not-yet-created
phase: red
command_or_action: GOPROXY=off GOTOOLCHAIN=go1.26.5 go test ./services/api/internal/merchantidentity -run '^TestMerchantLoginRejectsWhitespaceOnlyCodeBeforeAuthentication$' -count=1
exit_result: FAIL
sanitized_summary: a whitespace-only code reached authentication and returned 200 instead of exact INVALID_REQUEST
artifact_or_environment: Writer strict-request tracer
unverified_boundary: other malformed JSON, bearer failures and stable business errors were not covered by this tracer
external_asset:
  owner: N/A
  missing: N/A
  recovery: reject whitespace-only input without modifying any accepted nonempty code
```

```yaml
change: implement-merchant-identity-rbac-core
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: 122913c6bcc8c22acb73e05385d54449f27c2465
candidate_sha: not-yet-created
phase: red
command_or_action: GOPROXY=off GOTOOLCHAIN=go1.26.5 go test ./services/api/internal/merchantidentity -run '^TestMerchantLoginEndpointReturnsExactBoundSubaccountProjection$' -count=1
exit_result: FAIL
sanitized_summary: POST /api/v1/me/merchant-login returned 404 because the route and operation were absent
artifact_or_environment: Writer tree after identity GET Green
unverified_boundary: request strictness, error mapping, provider and database transactions remained unimplemented
external_asset:
  owner: N/A
  missing: N/A
  recovery: register only the frozen POST route with exact decoder, bearer and response mapping
```

```yaml
change: implement-merchant-identity-rbac-core
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: 122913c6bcc8c22acb73e05385d54449f27c2465
candidate_sha: not-yet-created
phase: green
command_or_action: same TestIdentityEndpointReturnsExactBoundOwnerProjection focused command after minimal protected handler implementation
exit_result: PASS
sanitized_summary: one strict bearer resolved the internal user and returned only exact primary_phone_bound plus OWNER/auth_version with no-store
artifact_or_environment: Writer first HTTP vertical slice
unverified_boundary: merchant-login, strict invalid matrices, persistence and W3 remained pending
external_asset:
  owner: N/A
  missing: N/A
  recovery: add the missing merchant-login route as the next independent tracer
```

```yaml
change: implement-merchant-identity-rbac-core
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: 122913c6bcc8c22acb73e05385d54449f27c2465
candidate_sha: not-yet-created
phase: red
command_or_action: GOPROXY=off GOTOOLCHAIN=go1.26.5 go test ./services/api/internal/merchantidentity -run '^TestIdentityEndpointReturnsExactBoundOwnerProjection$' -count=1
exit_result: FAIL
sanitized_summary: the merchantidentity package had no production Go files, so the public identity endpoint capability did not compile on the fixed base
artifact_or_environment: Writer tree with one public HTTP tracer
unverified_boundary: strict error cases, merchant-login, persistence, audit, authorization and MySQL remained absent
external_asset:
  owner: N/A
  missing: N/A
  recovery: add the minimal handler/types for exact identity success and rerun the identical tracer
```

```yaml
change: implement-merchant-identity-rbac-core
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: 122913c6bcc8c22acb73e05385d54449f27c2465
candidate_sha: not-yet-created
phase: green
command_or_action: same TestMerchantIdentityMigrationContracts focused command after adding frozen v12/v13 columns, uniqueness, FK, pair/version and snapshot-retention constraints
exit_result: PASS
sanitized_summary: the schema files matched the frozen account, binding, version, audit snapshot, target and PII-exclusion contract while remaining one CREATE TABLE each
artifact_or_environment: Writer migration contract slice
unverified_boundary: MySQL execution and runtime behavior remained pending
external_asset:
  owner: Writer
  missing: fresh MySQL run not yet started
  recovery: proceed to HTTP/runtime slices, then exercise all DDL constraints on disposable MySQL 8.0.46
```

```yaml
change: implement-merchant-identity-rbac-core
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: 122913c6bcc8c22acb73e05385d54449f27c2465
candidate_sha: not-yet-created
phase: red
command_or_action: GOPROXY=off GOTOOLCHAIN=go1.26.5 go test ./services/api/internal/catalog -run '^TestMerchantIdentityMigrationContracts$' -count=1
exit_result: FAIL
sanitized_summary: the minimal v12 table lacked the first frozen account field, proving the schema contract was not implemented
artifact_or_environment: Writer tree after exact-chain Green and before schema expansion
unverified_boundary: DDL constraints and all real MySQL/runtime behavior remained unverified
external_asset:
  owner: N/A
  missing: N/A
  recovery: add only frozen account/audit columns, constraints and retention foreign key, then rerun the identical contract test
```

```yaml
change: implement-merchant-identity-rbac-core
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: 122913c6bcc8c22acb73e05385d54449f27c2465
candidate_sha: not-yet-created
phase: green
command_or_action: GOPROXY=off GOTOOLCHAIN=go1.26.5 go test ./services/api/migrations ./services/api/internal/catalog -run '^(TestEmbeddedMigrationChainIsExactAndRecoverable|TestCatalogMigrationSet)$' -count=1 after adding one minimal forward CREATE TABLE in each assigned v12/v13 file
exit_result: PASS
sanitized_summary: embedded and loaded migration sets were exact contiguous v1-v13 and each new file was one LF-terminated CREATE TABLE statement
artifact_or_environment: Writer first migration vertical slice
unverified_boundary: merchant columns, constraints and real MySQL behavior were intentionally still absent
external_asset:
  owner: N/A
  missing: N/A
  recovery: drive the frozen schema with a separate contract Red before expanding either table
```

```yaml
change: implement-merchant-identity-rbac-core
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: 122913c6bcc8c22acb73e05385d54449f27c2465
candidate_sha: not-yet-created
phase: red
command_or_action: fixed-base temporary export with only the v12/v13 expectations overlaid, then GOPROXY=off GOTOOLCHAIN=go1.26.5 go test ./services/api/migrations ./services/api/internal/catalog -run '^(TestEmbeddedMigrationChainIsExactAndRecoverable|TestCatalogMigrationSet)$' -count=1
exit_result: FAIL
sanitized_summary: both matching tests executed and failed because embedded and loaded migration sets each contained 11 files while the test-only expectations required 13; the temporary export was moved to Trash after capture
artifact_or_environment: disposable archive export of exact fixed base with only the two test expectation additions
unverified_boundary: schema columns, constraints, real MySQL application and all runtime behavior remained absent
external_asset:
  owner: N/A
  missing: N/A
  recovery: add only one forward CREATE TABLE file for each assigned v12/v13 version and rerun the identical anchored-regex focused command
```

## Writer C/T/V/R

- C = 9：HTTP/schema/action/transaction/PII/非目标与现有三接口不变均有可追溯契约；T2/T3/T4 consumer 与外部 UI 不属于本 Candidate。
- T = 10：真实 vertical RGR、严格错误矩阵、fresh MySQL 8.0.46 v1-v13、约束/FK/unique、并发、真实 deadlock、rollback、commit-unknown、审计不可用、锁顺序与 PII 全部闭环。
- V = 8：Writer Gate 完整，可形成提交；fixed-base 双轴 Review 与主控批准后的 exact-SHA detached verifier 尚待执行。
- R = 10：失败零部分写、同 code/commit-unknown 受限恢复、deadlock 单次 provider、caller rollback、临时 schema/container/credential/process 清理均有可复现证据。
- 总分：37；当前 Writer 产品 Gate 硬阻断为 0，Review/Verifier 生命周期步骤仍待执行。
