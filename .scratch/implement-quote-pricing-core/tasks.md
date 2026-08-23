# Tasks: implement-quote-pricing-core

- [x] 核验 exact base、clean detached start、目标 branch 不存在后创建独立 writer branch。
  - Evidence: initial `HEAD=8bcdf3d6b1ea41529adaa54f463cc118c69e0e25`; clean detached; target branch absent; `codex/implement-quote-pricing-core` created without overwrite.
- [x] 完整读取适用 AGENTS/CONTEXT、canonical PRD §5.6/§6.4/§15.6.2/§15.6.5、质量 Gate、`$codebase-design`（含 `DEEPENING.md`）、`$tdd` 与 `$code-review`。
  - Evidence: `DRAFT.md`/`spec.md` 固定唯一 in-process `Calculate` seam、调用方时机与 read-only 范围。
- [x] 冻结 W3/UI0、Interface、公式/half-up/顺序/溢出/错误、owned/read-only、依赖/非目标、RGR、review/verifier/integration 命令与外部资产。
  - Evidence: `DRAFT.md` and `spec.md`; tracker 缺失如实标记 `GOVERNANCE_PENDING`，未越权配置。
- [x] 完成冲突审计。
  - Evidence: exact base 上两个 owned 目录均不存在；仓库内无 `quotepricing` Module 或同名 Interface。
- [x] 取得首个 public-seam 编译 Red，再只补齐可编译 surface。
  - Red evidence: `go test ./services/api/internal/quotepricing -run '^$' -count=1` exit `1`; external seam test reported only undefined `Input`, `Result`, and `Calculate`.
  - Green evidence: the same compile-only command passed after adding only the frozen types and zero-value `Calculate` surface.
- [x] 逐 tracer 完成 `101×85%, qty2`、逐商品非整单舍入、数量在舍入后相乘的 Red/Green。
  - Red evidence: three named focused tests separately exited `1` with zero Result, one missing output line, and payable `8` from subtotal rounding versus required `9`.
  - Green evidence: the same focused tests passed with `202/172/30`, two `1分×50%` lines totaling `2`, and `5分×50%=3分` then quantity `3` totaling `9`.
- [x] 逐 tracer 完成 `100%`、`0%`、`0元` 与非法 rate/price/quantity/空购物车的 Red/Green。
  - Red evidence: named focused tests separately exposed missing typed errors, rejected legal `0/100%`, rejected legal zero price, accepted non-positive quantity, and returned a nonzero-shape Result for empty lines; each exited `1`.
  - Green evidence: the same tests now accept both rate boundaries and zero-price lines while returning zero Result plus stable redacted `INVALID_RATE`/`INVALID_PRICE`/`INVALID_QUANTITY`/`EMPTY_LINES` errors for invalid inputs.
- [x] 逐 tracer 完成折扣乘法、行乘法、cross-line sum overflow 与零 Result typed/redacted error 的 Red/Green。
  - Red evidence: named focused tests separately exited `1` after unchecked arithmetic wrapped `MaxInt64×2` to a negative line subtotal, discount multiplication to a wrong zero, and `MaxInt64+1` to `MinInt64`.
  - Green evidence: the same tests now return exact zero `Result` plus stable redacted `OVERFLOW`; original/payable line multiplication and original/payable cross-line additions all use checked arithmetic.
- [x] 覆盖输出顺序、输入不变、重复/并发确定性并完成 Refactor focused/race Gate。
  - Evidence: distinct-line order/caller-slice test, 100-repeat determinism, 32-worker concurrency test, focused `-count=1`, and race `-count=20` all passed after Refactor; no implementation change was needed for the purity checks because the prior ordered loop already satisfied them.
- [x] mutation infrastructure shield 拒绝非行为失败，至少九个指定可逆 mutant 全部由命名 target test exit `1` 杀死。
  - Evidence: injected `go` exit `2` caused harness exit `82` and was accepted only as shield PASS; all nine real mutants then exited exactly `1` with their named `--- FAIL: Test...` marker.
- [x] fresh loopback-only MySQL 8.0.46 跑全 `services/api` test/race；另跑 vet/build/smoke/gofmt/diff/owned/protected/sensitive。
  - Evidence: fresh pinned image reported version `8.0.46` and loopback-only ephemeral binding; full `services/api` test/race passed; vet/build exited `0`; smoke printed `smoke: PASS`; gofmt, shell syntax, owned/protected and high-confidence sensitive-pattern audits passed. Exact base diff checks are repeated post-commit.
- [ ] 只提交 owned paths，中文完整 commit；通过 immutable external handoff 绑定 exact candidate SHA。
- [ ] 对 exact SHA 并行完成 Standards/Spec 双轴审查，零 finding。
- [ ] 在 fresh clean detached worktree 对 exact SHA 从头重跑全部 Gate，writer/verifier 均 clean。

> Post-commit review/verifier 两项在实际 external receipt 前必须保持 pending，不预宣 PASS。integration 不在本 change 授权范围内。

## Evidence records

以下记录逐阶段追加；只保留首个决定性、脱敏结果，不记录凭据、请求原文或个人数据。

```yaml
change: implement-quote-pricing-core
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: 8bcdf3d6b1ea41529adaa54f463cc118c69e0e25
candidate_sha: external-post-commit
phase: red
command_or_action: public-seam compile-only test
exit_result: exit-1
sanitized_summary: external test could not resolve Input, Result, or Calculate because the frozen Interface was absent
artifact_or_environment: writer worktree with public_seam_test.go
unverified_boundary: compile Red proves no pricing behavior or Green result
external_asset:
  owner: N/A
  missing: N/A
  recovery: add only the frozen Interface surface and rerun the same command
```

```yaml
change: implement-quote-pricing-core
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: 8bcdf3d6b1ea41529adaa54f463cc118c69e0e25
candidate_sha: external-post-commit
phase: writer
command_or_action: fresh repair-version-scoped verify-mysql.sh full; vet; build; smoke; format; shell syntax; owned/protected/sensitive audits
exit_result: PASS
sanitized_summary: fresh loopback-only MySQL 8.0.46 full services/api test/race passed; vet/build/smoke and static scope audits passed
artifact_or_environment: disposable mysql:8.0.46-oraclelinux9 and local Go 1.26.5; temporary resources cleaned
unverified_boundary: MySQL is adjacent regression only and does not prove this pure Module pricing; no caller integration, UI, real order/payment, push, PR, deployment, or external write ran
external_asset:
  owner: writer/verifier
  missing: verifier must independently recreate all assets from exact SHA
  recovery: rerun full profile in a fresh detached worktree after immutable candidate handoff
```

```yaml
change: implement-quote-pricing-core
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: 8bcdf3d6b1ea41529adaa54f463cc118c69e0e25
candidate_sha: external-post-commit
phase: refactor
command_or_action: focused package test; race count 20; mutation gate with infrastructure failure shield
exit_result: PASS
sanitized_summary: public-seam money examples, typed redacted fail-closed errors, ordered immutable input handling, concurrent determinism, shield, and nine targeted mutation sensitivities passed
artifact_or_environment: local Go 1.26.5 and isolated temporary mutation copies cleaned on exit
unverified_boundary: pure Module evidence does not prove caller identity/config policy, database behavior, Quote/Prepay/Order creation, UI, integration, or deployment
external_asset:
  owner: N/A
  missing: N/A
  recovery: rerun the same repository commands
```
