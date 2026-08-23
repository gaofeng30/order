# Tasks: implement-order-lifecycle-transition-policy-core

`candidate_sha` 使用 `external-post-commit`：产生 replacement SHA 的 commit 内不自指 SHA；exact SHA、双审与 detached receipt 只由外部 handoff 绑定，不回写 candidate。首个 candidate `2cfc0e4954ea80c07680ae56b1d055ff6c627731` 已由 Standards finding 作废，以下 writer evidence 已按 replacement 内容从头重跑后才可形成新 candidate。

- [x] 核验 exact base/clean detached start/branch 不存在，读取适用规则与 canonical PRD，冻结 W3/UI0、owner、owned/read-only、非目标、Interface、完整矩阵、错误/时间边界、RGR、命令、外部资产和 governance pending。

```yaml
change: implement-order-lifecycle-transition-policy-core
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: 8bcdf3d6b1ea41529adaa54f463cc118c69e0e25
candidate_sha: external-post-commit
phase: writer
command_or_action: git rev-parse HEAD; git status --short --branch; git show-ref --verify refs/heads/codex/implement-order-lifecycle-transition-policy-core; read AGENTS.md CONTEXT.md canonical PRD quality Gate and three required Skills
exit_result: PASS
sanitized_summary: exact base and clean detached start confirmed; target branch was absent; scratch DRAFT/spec froze the only public Transition seam and 6x5 matrix; issue tracker remains governance pending
artifact_or_environment: writer worktree and repository documents
unverified_boundary: document inspection does not prove implementation behavior
external_asset:
  owner: N/A
  missing: docs/agents/issue-tracker.md
  recovery: future governance setup only; not required for this user-provided scratch Spec
```

- [x] 只经公共 `Transition` Interface 完成 public-seam 编译 Red、逐 tracer 五类合法边、取消边界、完整矩阵、typed fail-closed、时间独立与确定性的 Red -> Green -> Refactor；existing `InitialState`/`Advance` 实现和测试未改。

```yaml
change: implement-order-lifecycle-transition-policy-core
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: 8bcdf3d6b1ea41529adaa54f463cc118c69e0e25
candidate_sha: external-post-commit
phase: red
command_or_action: >-
  GOPROXY=off GOTOOLCHAIN=go1.26.5 go test ./services/api/internal/orderproduction -run '^$' -count=1;
  the same exact command was run separately with -run '^TestTransitionMerchantCanMarkPreparingOrderReady$',
  '^TestTransitionRedeemCompletesReadyOrder$',
  '^TestTransitionUserCanCancelReservedOrderMoreThanThirtyMinutesBeforePickup$',
  '^TestTransitionOwnerCanRequestRefundFromEligibleStates$',
  '^TestTransitionVerifiedRefundCompletesRefundingOrder$',
  '^TestTransitionRejectsUserCancelAtOrInsideThirtyMinutes$',
  '^TestTransitionRejectsZeroTimesForUserCancel$',
  '^TestTransitionRejectsInvalidAndDeprecatedStates$',
  '^TestTransitionRejectsUnknownTriggers$' and
  '^TestTransitionMatchesCompleteStateTriggerMatrix$'
exit_result: FAIL
sanitized_summary: missing Transition Interface, zero Decisions for legal edges, nil errors on all 22 illegal matrix edges, and zero observed time incorrectly allowing REFUNDING exposed the frozen behavior before implementation
artifact_or_environment: writer worktree external-package tests
unverified_boundary: Red evidence does not prove Green or external integration
external_asset:
  owner: N/A
  missing: N/A
  recovery: N/A
```

```yaml
change: implement-order-lifecycle-transition-policy-core
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: 8bcdf3d6b1ea41529adaa54f463cc118c69e0e25
candidate_sha: external-post-commit
phase: green
command_or_action: GOPROXY=off GOTOOLCHAIN=go1.26.5 go test ./services/api/internal/orderproduction -count=1
exit_result: PASS
sanitized_summary: five trigger classes, 8 legal edges, 22 illegal edges, strict cancel boundary, invalid/deprecated state, unknown trigger, zero times, zero Decisions and stable typed errors passed through Transition
artifact_or_environment: writer worktree Go 1.26.5
unverified_boundary: pure Module test does not prove persistence or caller prerequisites
external_asset:
  owner: N/A
  missing: N/A
  recovery: rerun the same focused package command
```

```yaml
change: implement-order-lifecycle-transition-policy-core
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: 8bcdf3d6b1ea41529adaa54f463cc118c69e0e25
candidate_sha: external-post-commit
phase: refactor
command_or_action: GOPROXY=off GOTOOLCHAIN=go1.26.5 go test -race ./services/api/internal/orderproduction -count=20
exit_result: PASS
sanitized_summary: refactored single trigger switch remained deterministic and race clean; 32-worker repeated literal expectations passed; public seam compile-time checks enforce time.Time value fields
artifact_or_environment: writer worktree Go race detector
unverified_boundary: race-clean pure computation does not prove DB concurrency, locking or idempotency
external_asset:
  owner: N/A
  missing: N/A
  recovery: rerun the same race command
```

- [x] mutation harness 的 infrastructure failure shield 与 8 个指定可逆 mutant 通过，且只接受 source pattern 恰好一次、目标测试 exit 1 和指定 FAIL marker。

```yaml
change: implement-order-lifecycle-transition-policy-core
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: 8bcdf3d6b1ea41529adaa54f463cc118c69e0e25
candidate_sha: external-post-commit
phase: writer
command_or_action: bash .scratch/implement-order-lifecycle-transition-policy-core/verify-mutation-gate.sh
exit_result: PASS
sanitized_summary: simulated go exit 2 produced shield exit 82; all eight mutants printed MUTATION_KILLED with exit 1 and exact named FAIL marker; temporary copies were cleaned
artifact_or_environment: isolated temporary mutation copies under /private/tmp
unverified_boundary: mutation sensitivity does not prove DB transactions or external provider facts
external_asset:
  owner: writer/verifier
  missing: N/A
  recovery: rerun the mutation Gate from repository root
```

- [x] fresh loopback-only MySQL 8.0.46 全 `services/api` test/race 邻接回归，以及 vet/build/smoke/gofmt/diff/owned/protected/sensitive/phase/exit_result Gate 通过。

```yaml
change: implement-order-lifecycle-transition-policy-core
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: 8bcdf3d6b1ea41529adaa54f463cc118c69e0e25
candidate_sha: external-post-commit
phase: writer
command_or_action: >-
  bash .scratch/repair-version-scoped-mysql-migration-fixtures-v13/verify-mysql.sh full;
  GOPROXY=off GOTOOLCHAIN=go1.26.5 go vet ./services/api/...;
  GOPROXY=off GOTOOLCHAIN=go1.26.5 go build ./services/api/...;
  GOPROXY=off GOTOOLCHAIN=go1.26.5 bash services/api/scripts/smoke.sh;
  git diff --check;
  test -z "$(gofmt -l services/api/internal/orderproduction)";
  bash -n .scratch/implement-order-lifecycle-transition-policy-core/verify-mutations.sh .scratch/implement-order-lifecycle-transition-policy-core/verify-mutation-gate.sh;
  git diff --name-only plus protected sensitive phase and exit_result audits
exit_result: PASS
sanitized_summary: MySQL version 8.0.46 bound to 127.0.0.1 and all services/api normal/race suites passed; vet/build passed; standalone smoke printed smoke PASS; static audits passed
artifact_or_environment: writer worktree plus disposable mysql:8.0.46-oraclelinux9 container
unverified_boundary: adjacent MySQL regression does not prove Transition DB transaction, CAS, lock, idempotency, real refund, redemption, notification, QR/token, integration or production
external_asset:
  owner: writer/verifier
  missing: writer N/A; verifier must recreate fresh container
  recovery: Docker daemon and pinned MySQL image available, then rerun full profile from exact SHA
```

- [x] 首个 candidate 的两项 Standards finding 已修复；replacement 内容只含 owned paths，中文 commit 将形成 exact candidate，worktree clean 与 SHA 由 external post-commit receipt 证明。

```yaml
change: implement-order-lifecycle-transition-policy-core
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: 8bcdf3d6b1ea41529adaa54f463cc118c69e0e25
candidate_sha: external-post-commit
phase: writer
command_or_action: remove tautological runtime input-copy test; add compile-time time.Time value-field checks; replace free-text task evidence with complete per-task Red Green Refactor and writer receipts; run git diff --cached --check and owned/protected/sensitive audits
exit_result: PASS
sanitized_summary: invalidated candidate is recorded; both Standards findings are remediated without changing production Transition behavior or owned paths
artifact_or_environment: writer worktree staged replacement content
unverified_boundary: exact replacement SHA and post-commit reviews do not exist until the external receipt
external_asset:
  owner: writer/reviewers/verifier
  missing: replacement post-commit SHA, fresh dual review and fresh detached verification
  recovery: commit owned replacement content, bind exact SHA externally, then rerun all Gates and reviews
```

- [ ] `EXTERNAL_POST_COMMIT`: 对 replacement exact candidate/fixed base 按 `$code-review` 并行完成 Standards/Spec 双轴审查，0 finding；不得回写 candidate。
- [ ] `EXTERNAL_POST_COMMIT`: 在 fresh clean detached replacement exact SHA 从头完成全部 Gate并取得 detached receipt；不得回写 candidate。
- [x] writer 未 integration、push/PR/deploy、访问微信/生产或写入外部系统。

```yaml
change: implement-order-lifecycle-transition-policy-core
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: 8bcdf3d6b1ea41529adaa54f463cc118c69e0e25
candidate_sha: external-post-commit
phase: writer
command_or_action: maintain local-only writer scope and stop before integration push PR deploy WeChat production or any external write
exit_result: PASS
sanitized_summary: only the declared writer worktree and owned repository paths changed; no external mutation was authorized or performed
artifact_or_environment: local writer worktree
unverified_boundary: this receipt does not authorize any future external action
external_asset:
  owner: N/A
  missing: N/A
  recovery: N/A
```
