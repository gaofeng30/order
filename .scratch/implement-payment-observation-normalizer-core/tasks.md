# Tasks and evidence index

- [x] 核验 exact base、初始 clean、目标 branch 不存在并创建 change branch。
- [x] 读取 AGENTS、CONTEXT、canonical PRD 支付/订单小节、质量 Gate、三个指定 Skill 与引用。
- [x] 冻结 owned/read-only、依赖、interface、状态/错误、canonical、RGR、Gate 与外部资产。
- [x] RED：公共 seam 缺失产生真实失败并记录首个决定性错误。
- [x] GREEN：逐个 tracer bullet 完成最小实现。
- [x] Refactor：收窄实现并重跑同一 focused/race suite。
- [x] Mutation/resilience：canonical、precedence、accepted/rejected、malformed、重复与 collision。
- [x] Writer runtime Gate：新 base `5e937f...56a2` focused/race、7 mutation、fresh MySQL、全 test/race、vet/build/smoke、format/PII/owned 均从头 PASS。
- [x] 候选内容与中文 commit 流程已冻结；final exact SHA 与 clean 使用 `external-post-commit` handoff 绑定，不在 immutable commit 内自引用。
- [x] `$code-review` 双轴 workflow 已发现并驱动两个真实 P2 修复；replacement review 与 verifier 命令包及独立 handoff 已冻结。
- [ ] replacement exact SHA Standards/Spec 双轴 `0 findings`；实际 verdict 只由 post-commit handoff 绑定。
- [ ] fresh detached exact-SHA independent verification PASS；实际 verdict 只进入外部 verifier receipt。

首次 candidate `25af340...5415` 的 Standards 为 0 finding，Spec 报告 1 个真实 P2：malformed
transaction 被 callback/state 错误遮蔽。已先用组合测试取得 RED，再最小修复并从 focused 到 fresh
MySQL/full/race/vet/build/smoke 全量重跑；旧 candidate 与两份 review 均失效，新双审仍待执行。

第二个 candidate `30537be...85dc` 的 Standards 为 0 finding，Spec 报告 1 个真实 P2：未检查
typed Transaction 中所有字符串的 NUL。已用 12 字段表驱动测试取得 RED，补齐结构校验并证明
excluded provider/PII 字段不改变 Observation/dedupe；再次全量重跑。该 candidate/review 也已失效。

恢复声明：主控已在新 exact base 集成 dependency foundation；旧 FAIL 仍为历史证据，不复用旧
RED/PASS。Candidate、`$code-review` 与 detached verifier 仍须按新 SHA 顺序执行。

Writer runtime receipt 见 `evidence/v13-writer-gates.md`。commit 后才有意义的 exact-SHA diff、
clean、双轴 review 与 detached receipt 保持后置绑定，不通过修改本 commit 自引用回填。

## Immutable candidate handoff

```yaml
change: implement-payment-observation-normalizer-core
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: 5e937f3599a16f4813d6021f4cd2dd637c3156a2
candidate_sha: external-post-commit
phase: writer
command_or_action: freeze owned implementation/spec/tasks and commit with a Chinese message; bind final exact SHA, clean, replacement two-axis review and fresh detached verification outside the immutable commit
exit_result: PASS
sanitized_summary: writer runtime gates passed on the final implementation; two prior exact candidates were invalidated after Spec findings were reproduced and fixed; the last pre-governance candidate received Standards and Spec zero-finding verdicts
artifact_or_environment: final owned writer tree before the governance replacement commit
unverified_boundary: the replacement exact SHA does not exist inside its own content; its clean state, fresh full-diff review and detached Gate verdict must be recorded by immutable external handoff receipts
external_asset:
  owner: writer handoff, independent Standards/Spec reviewers, detached verifier
  missing: N/A
  recovery: after commit, pin the exact SHA, rerun both review axes on the full base diff, then run every declared Gate in a fresh clean detached worktree
```

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
exit_result: FAIL
sanitized_summary: exit code 1; fresh detached exact-base package build failed with no non-test Go files after copying only normalize_test.go; the public Normalize production seam was absent
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
exit_result: FAIL
sanitized_summary: exit code 1; exact base has 13 migrations while five protected historical suites still require 10 or v1-v11; current v1-v13 merchantidentity, wechatpay and paymentobservation packages passed
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
exit_result: FAIL
sanitized_summary: exit code 1; package build failed because paymentobservation had no non-test Go files; the public Normalize seam was absent
artifact_or_environment: services/api/internal/paymentobservation/normalize_test.go at the first tracer bullet
unverified_boundary: no behavior is implemented or proven at RED
external_asset:
  owner: N/A
  missing: N/A
  recovery: add only the public seam and minimum callback/query SUCCESS normalization, then rerun this exact test
```
