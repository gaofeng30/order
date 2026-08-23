# Tasks: implement-order-production-policy-core

- [x] 核验 exact base、clean detached start、目标 branch 不存在后创建独立 writer branch。
  - Evidence: initial `HEAD=5e937f3599a16f4813d6021f4cd2dd637c3156a2`; clean; branch `codex/implement-order-production-policy-core` created without overwrite.
- [x] 完整读取适用 AGENTS/CONTEXT、canonical PRD §7.1/§7.2/§7.4、质量 Gate、`$codebase-design`、`$tdd`、`$code-review` 与 WIP transaction engineering reference。
  - Evidence: frozen Interface and source classification in `DRAFT.md`/`spec.md`; WIP reference remains `NOT_APPROVED`.
- [x] 冻结 W3/UI0、Interface、状态表、30 分钟字面边界、owned/read-only、非目标、RGR、review/verifier/integration 命令与外部资产。
  - Evidence: `DRAFT.md` and `spec.md`.
- [x] 完成冲突审计。
  - Evidence: exact base has no owned target directories or existing production-policy implementation; adjacent active work does not own these paths.
- [x] 取得首个 public-seam 编译 Red。
  - Evidence: `go test ./services/api/internal/orderproduction -run '^$' -count=1` exited 1 because `State`, `Decision`, `InitialState`, and `Advance` were undefined; the Red was caused only by the missing frozen Interface.
- [x] 逐 tracer 完成 Initial `>30m`、`<30m`、`=30m` Red/Green。
  - Red evidence: each focused tracer first exited 1 with zero state versus `RESERVED`, `PREPARING`, and exact-boundary `RESERVED`; the same focused commands passed after their minimal implementation slices.
- [x] 逐 tracer 完成 Advance 阈值前、恰好阈值、漏跑后 Red/Green。
  - Red evidence: each focused tracer first exited 1 with zero Decision versus the frozen unchanged/changed Decision; the same focused commands passed after their minimal implementation slices.
- [x] 覆盖五个后继态不回退、非法/废止/空状态、零时间、`pickup<=success`。
  - Red evidence: all five successors initially returned zero Decision; invalid-state typed errors were undefined; zero/invalid times initially returned legal states or Decisions. Focused Green now returns unchanged successors and redacted `INVALID_STATE`/`INVALID_TIME` failures.
- [x] 覆盖并发与重复确定性并完成 focused/race Refactor Gate。
  - Evidence: exact 30-minute two-step test, 32-worker repeated determinism test, focused PASS, and `go test -race ... -count=20` PASS after Refactor.
- [x] 杀死五个指定可逆 mutant，并恢复后全绿。
  - Evidence: hardened `verify-mutations.sh` kills Initial `<→<=`, Advance `>=→>`, insufficient-time `PREPARING→RESERVED`, successor rollback, and invalid-state acceptance only when each focused command exits 1 with its named `--- FAIL: Test...` marker; focused and race gates pass after temporary copies are removed.
- [x] 在 fresh loopback MySQL 8.0.46 跑 baseline `verify-mysql.sh full`，另跑 vet/build/smoke/gofmt/diff/owned/PII。
  - Evidence: pinned image reported MySQL `8.0.46` and loopback-only binding; all `services/api` tests and race tests passed; vet/build exited 0; smoke printed `smoke: PASS`; formatting, shell syntax, sensitive-pattern and temporary-resource checks passed before business candidate `39c0ea43067f30e5befd48290e9cb45711415452`. Those dynamic results were later invalidated with that candidate and are not inherited by the replacement.
- [x] 只提交 owned paths，中文完整 commit，并记录 exact business candidate SHA。
  - Evidence: `39c0ea43067f30e5befd48290e9cb45711415452`; clean writer tree after commit; subsequently `INVALIDATED_BY_STANDARDS_GOVERNANCE_AUDIT` and never pushed/integrated/deployed.
- [x] 如实记录旧 Candidate/双审作废并消除 phase/status/checklist 矛盾。
  - Evidence: `DRAFT.md` now records the full invalidated business candidate SHA, replacement-ready status, two-stage final-SHA handoff, and non-inheritance rule.
- [x] 修复 mutation harness 的假阳性并完成 remediation Red/Green。
  - Red evidence: an injected `go` setup failure exited 2, but the old harness printed five `MUTATION_KILLED` lines and exited 0.
  - Green evidence: `verify-mutation-gate.sh` requires the same setup failure to exit 82 before it runs real mutants; the hardened harness requires one exact source match plus exit 1 and the named `--- FAIL: Test...` marker, then kills all five mutants and leaves focused/race green.
- [ ] 对 replacement exact SHA 并行完成 Standards/Spec 双轴审查，零 finding。
- [ ] 在 fresh clean detached worktree 对 exact SHA 从头重跑全部 Gate，writer/verifier 均 clean。

## Evidence records

以下记录只保留脱敏后的首个决定性结果；不记录凭据、原始请求/响应、个人数据或可重放标识。

```yaml
change: implement-order-production-policy-core
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: 5e937f3599a16f4813d6021f4cd2dd637c3156a2
candidate_sha: 39c0ea43067f30e5befd48290e9cb45711415452
candidate_status: INVALIDATED_BY_STANDARDS_GOVERNANCE_AUDIT
phase: red
command_or_action: focused public-seam compile and one focused command per Initial, Advance, successor, invalid-state, and invalid-time tracer
exit_result: FAIL
sanitized_summary: missing Interface, zero state/Decision, missing typed errors, and unsafe legal outputs exposed every frozen behavior before implementation
artifact_or_environment: writer worktree public seam tests
unverified_boundary: Red evidence does not prove Green or external integration
external_asset:
  owner: N/A
  missing: N/A
  recovery: N/A
```

```yaml
change: implement-order-production-policy-core
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: 5e937f3599a16f4813d6021f4cd2dd637c3156a2
candidate_sha: 39c0ea43067f30e5befd48290e9cb45711415452
candidate_status: INVALIDATED_BY_STANDARDS_GOVERNANCE_AUDIT
phase: refactor
command_or_action: focused package test; race count 20; temporary-copy mutation harness
exit_result: PASS
sanitized_summary: public-seam behavior, exact 30-minute two-step boundary, concurrent determinism, redacted errors, and all five mutation sensitivities passed with targeted exit-1 test failures
artifact_or_environment: local Go 1.26.5 and isolated temporary mutation copies, cleaned on exit
unverified_boundary: pure-function evidence does not prove DB atomicity, scheduler recovery, payment confirmation, or production deployment
external_asset:
  owner: N/A
  missing: N/A
  recovery: rerun the same repository commands
```

```yaml
change: implement-order-production-policy-core
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: 5e937f3599a16f4813d6021f4cd2dd637c3156a2
candidate_sha: 39c0ea43067f30e5befd48290e9cb45711415452
candidate_status: INVALIDATED_BY_STANDARDS_GOVERNANCE_AUDIT
phase: writer
command_or_action: repair-version-scoped baseline verify-mysql.sh full; vet; build; smoke; format and scope audits
exit_result: PASS
sanitized_summary: fresh loopback-only MySQL 8.0.46 full services/api test/race passed; vet/build/smoke and static audits passed
artifact_or_environment: disposable mysql:8.0.46-oraclelinux9 plus local Go 1.26.5; temporary resources cleaned
unverified_boundary: MySQL is an adjacent schema/transaction regression only and does not prove this Module policy; no UI, real payment, push, PR, integration, deployment, or production write ran
external_asset:
  owner: writer/verifier
  missing: N/A for writer; verifier must independently recreate
  recovery: rerun full profile from exact candidate in a fresh detached worktree
```

```yaml
change: implement-order-production-policy-core
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: 5e937f3599a16f4813d6021f4cd2dd637c3156a2
candidate_sha: 39c0ea43067f30e5befd48290e9cb45711415452
candidate_status: INVALIDATED_BY_STANDARDS_GOVERNANCE_AUDIT
phase: writer
command_or_action: third-party governance audit; exported failing go function against old mutation harness; harden source and behavior-failure proof
exit_result: PASS
sanitized_summary: old candidate and all old review/runtime receipts invalidated; old harness falsely exited 0 for infrastructure exit 2, while the new shield exits 82 and all five targeted mutants exit 1 with named test markers
artifact_or_environment: owned task artifacts and isolated temporary mutation copies
unverified_boundary: replacement final SHA, new dual review, and detached full Gate are pending and cannot inherit any old result
external_asset:
  owner: writer/verifier
  missing: replacement external SHA handoff and fresh detached receipt
  recovery: commit owned remediation, bind exact SHA externally, then rerun all declared gates from the fixed base
```
