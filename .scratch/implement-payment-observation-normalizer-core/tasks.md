# Tasks and evidence index

- [x] 核验 exact base、初始 clean、目标 branch 不存在并创建 change branch。
- [x] 读取 AGENTS、CONTEXT、canonical PRD 支付/订单小节、质量 Gate、三个指定 Skill 与引用。
- [x] 冻结 owned/read-only、依赖、interface、状态/错误、canonical、RGR、Gate 与外部资产。
- [x] RED：公共 seam 缺失产生真实失败并记录首个决定性错误。
- [x] GREEN：逐个 tracer bullet 完成最小实现。
- [x] Refactor：收窄实现并重跑同一 focused/race suite。
- [x] Mutation/resilience：canonical、precedence、accepted/rejected、malformed、重复与 collision。
- [x] Writer runtime Gate：新 base `5e937f...56a2` focused/race、7 mutation、fresh MySQL、全 test/race、vet/build/smoke、format/PII/owned 均从头 PASS。
- [ ] 候选中文 commit、exact SHA、clean。
- [ ] `$code-review` Standards/Spec 双轴 PASS。
- [ ] fresh clean detached exact-SHA independent verification PASS。

恢复声明：主控已在新 exact base 集成 dependency foundation；旧 FAIL 仍为历史证据，不复用旧
RED/PASS。Candidate、`$code-review` 与 detached verifier 仍须按新 SHA 顺序执行。

Writer runtime receipt 见 `evidence/v13-writer-gates.md`。commit 后才有意义的 exact-SHA diff、
clean、双轴 review 与 detached receipt 保持后置绑定，不通过修改本 commit 自引用回填。

## v13 恢复固定证据

```yaml
change: implement-payment-observation-normalizer-core
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: 5e937f3599a16f4813d6021f4cd2dd637c3156a2
candidate_sha: not-yet-created
phase: writer
command_or_action: audit old writer; git switch -c codex/implement-payment-observation-normalizer-core-v13 5e937f3599a16f4813d6021f4cd2dd637c3156a2
exit_result: PASS
sanitized_summary: old branch remained at b7f484f with exactly two untracked owned roots; new branch did not exist; disjoint WIP moved to the exact v13 base without conflict
artifact_or_environment: /Users/vivix/.codex/worktrees/142a/order
unverified_boundary: master-reported staging and detached PASS do not replace this writer or candidate verifier rerun
external_asset:
  owner: writer/verifier
  missing: N/A
  recovery: acquire a new exact-base RED, then rerun every declared Gate from scratch
```

## v13 首个真实 RED

```yaml
change: implement-payment-observation-normalizer-core
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: 5e937f3599a16f4813d6021f4cd2dd637c3156a2
candidate_sha: not-yet-created
phase: red
command_or_action: GOPROXY=off GOTOOLCHAIN=go1.26.5 go test ./services/api/internal/paymentobservation -run '^TestNormalizeCanonicalizesSuccessfulCallbackAndQueryIdentically$' -count=1
exit_result: FAIL exit-code-1
sanitized_summary: fresh detached exact-base package build failed with no non-test Go files after copying only normalize_test.go; the public Normalize production seam was absent
artifact_or_environment: disposable clean detached worktree; removed after RED; evidence/v13-red.md
unverified_boundary: no production implementation from the writer worktree participated in this RED
external_asset:
  owner: N/A
  missing: N/A
  recovery: run the same focused test against the writer implementation, then run every declared Gate from scratch
```

可执行 WIP Gate 的汇总见 `evidence/rgr.md` 与 `evidence/wip-gates.md`；dependency 审计与独立
foundation change 见 `evidence/dependency-blocker.md`。

## W3 首次邻接回归失败与最小恢复

```yaml
change: implement-payment-observation-normalizer-core
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: b7f484f54decfa38bc36bbed1ada041828d87483
candidate_sha: not-yet-created
phase: writer
command_or_action: first fresh loopback MySQL 8.0.46 run across every historical MySQL integration package plus wechatpay/paymentobservation
exit_result: FAIL exit-code-1
sanitized_summary: exact base has 13 migrations while five protected historical suites still require 10 or v1-v11; current v1-v13 merchantidentity, wechatpay and paymentobservation packages passed
artifact_or_environment: disposable isolated MySQL 8.0.46 container; cleaned after failure
unverified_boundary: historical protected suites remain stale on the exact base and cannot be repaired within owned paths
external_asset:
  owner: integration-base maintainer
  missing: aligned historical MySQL assertions for the 13-migration base
  recovery: historical only; foundation is present in 5e937f...56a2, but current writer and future verifier must rerun the full suite
```

## 初始固定证据

```yaml
change: implement-payment-observation-normalizer-core
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: b7f484f54decfa38bc36bbed1ada041828d87483
candidate_sha: not-yet-created
phase: writer
command_or_action: git status --short --branch; git rev-parse HEAD; git branch --list target; git rev-parse codex/order-delivery-integration
exit_result: PASS
sanitized_summary: initial detached HEAD and integration branch resolved to the exact base, tree/index were clean, and target branch did not exist
artifact_or_environment: /Users/vivix/.codex/worktrees/142a/order
unverified_boundary: does not prove implementation behavior or any external WeChat/MySQL result
external_asset:
  owner: N/A
  missing: N/A
  recovery: rerun before any candidate if base or worktree identity changes
```

## 首个真实 RED

```yaml
change: implement-payment-observation-normalizer-core
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: b7f484f54decfa38bc36bbed1ada041828d87483
candidate_sha: not-yet-created
phase: red
command_or_action: GOPROXY=off GOTOOLCHAIN=go1.26.5 go test ./services/api/internal/paymentobservation -run '^TestNormalizeCanonicalizesSuccessfulCallbackAndQueryIdentically$' -count=1
exit_result: FAIL exit-code-1
sanitized_summary: package build failed because paymentobservation had no non-test Go files; the public Normalize seam was absent
artifact_or_environment: services/api/internal/paymentobservation/normalize_test.go at the first tracer bullet
unverified_boundary: no behavior is implemented or proven at RED
external_asset:
  owner: N/A
  missing: N/A
  recovery: add only the public seam and minimum callback/query SUCCESS normalization, then rerun this exact test
```
