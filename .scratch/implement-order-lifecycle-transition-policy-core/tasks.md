# Tasks: implement-order-lifecycle-transition-policy-core

- [x] 核验 exact base、clean detached start、目标 branch 不存在，并在当前 worktree 创建唯一 writer branch。
  - Evidence: `HEAD=8bcdf3d6b1ea41529adaa54f463cc118c69e0e25`; `git status --short --branch` 仅显示 detached header；`BRANCH_ABSENT` 后创建 `codex/implement-order-lifecycle-transition-policy-core`。
- [x] 完整读取根 `AGENTS.md`、`CONTEXT.md`、canonical PRD 指定章节、`docs/quality/change-quality-gates.md`、`$codebase-design`（含 `DEEPENING.md`）、`$tdd`（含 tests/mocking）和 `$code-review`。
  - Evidence: 本 DRAFT/spec 只使用 canonical PRD 批准语义，并冻结公共 seam、Interface 与 replace-don't-layer 测试策略。
- [x] 冻结 W3/UI0、owner/base/branch/worktree、依赖、owned/read-only/non-goals、Interface、完整矩阵、错误/时间边界、RGR、命令、外部资产和 `external-post-commit` candidate 策略。
  - Evidence: `DRAFT.md` 与 `spec.md`。
- [x] 如实记录 tracker governance pending，不配置 tracker。
  - Evidence: `docs/agents/issue-tracker.md` 不存在；本 scratch Spec 是明确 change 的实施来源。
- [x] public-seam 编译 Red。
  - Evidence: `go test ... -run '^$'` exit 1，唯一决定性错误为 `TransitionInput` 与 `Transition` undefined；加入冻结 Interface 后同命令 exit 0。
- [x] 逐 tracer 完成五类合法转换的 Red -> Green。
  - Evidence: 备好、核销、用户取消 `>30m`、四前态 owner refund、verified refund 均先因零 Decision exit 1，随后相同 focused test PASS。
- [x] 覆盖 USER_CANCEL `>30m` / `=30m` / `<30m`、零时间、制作中拒绝。
  - Evidence: boundary tracer 首次得到 nil error；零 observed 首次错误返回 `REFUNDING`，零 pickup 首次错误 kind；最小修复后严格 `>30m` 成功，其余 fail closed。
- [x] 覆盖完整 current x trigger 矩阵、invalid/deprecated state、unknown trigger 与所有失败零 Decision。
  - Evidence: 完整矩阵 Red 精确暴露 22 条非法边返回 nil error；Green 验证 8 条合法边、22 条非法边及全部 typed fail-closed 输入。
- [x] 覆盖非取消 trigger 时间独立、输入不变、重复/并发确定性。
  - Evidence: 四类非取消 trigger 在六态上对零/反向时间结果一致；值输入未变化；32 workers x 100 repetitions 使用独立字面期望通过。
- [x] Refactor 后 focused、race `-count=20`、determinism 全绿，且现有 InitialState/Advance 测试未改且通过。
  - Evidence: `go test ... -count=1` 与 `go test -race ... -count=20` exit 0；existing tests 未编辑。
- [x] mutation infrastructure shield 通过，8 个指定 mutant 均由 exit 1 + 指定命名 FAIL marker 杀死，恢复后全绿。
  - Evidence: simulated tool exit 2 被 shield 转为 exit 82；八行 `MUTATION_KILLED ... exit=1 marker=--- FAIL: Test...`；workspace source 未被 harness 修改。
- [x] fresh loopback-only MySQL 8.0.46 全 `services/api` test/race 邻接回归通过；不宣称证明 DB 事务。
  - Evidence: `MYSQL_ENV ... version=8.0.46 binding=127.0.0.1 ... profile=full`；全普通/race suites exit 0，临时容器已清理。
- [x] vet/build/smoke/gofmt/diff/owned/protected/sensitive/phase/exit_result 静态 Gate 通过。
  - Evidence: vet/build exit 0；smoke standalone 重跑打印 `smoke: PASS`；format/diff/shell/scope audit PASS；证据只使用协议枚举且无敏感原文。
- [x] 候选提交前置完整：只包含 owned paths，中文 commit 将形成 exact candidate，SHA 由 external post-commit receipt 绑定。
- [ ] `EXTERNAL_POST_COMMIT`：对 exact candidate/fixed base 按 `$code-review` 并行完成 Standards/Spec 双轴审查，0 finding；不得回写 candidate。
- [ ] `EXTERNAL_POST_COMMIT`：fresh clean detached exact candidate SHA 从头完成全部 Gate，writer/verifier clean；不得回写 candidate。
- [x] 不 integration、不 push/PR/deploy、不访问微信/生产。

## Evidence records

每个阶段只登记首个决定性脱敏证据。`candidate_sha` 在 commit 前始终为 `external-post-commit`；历史或另一 SHA 的 PASS 不得继承。

```yaml
change: implement-order-lifecycle-transition-policy-core
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: 8bcdf3d6b1ea41529adaa54f463cc118c69e0e25
candidate_sha: external-post-commit
phase: red
command_or_action: public seam compile plus one focused command per legal transition, cancel boundary, state-trigger matrix, invalid state, unknown trigger, and zero time
exit_result: FAIL
sanitized_summary: missing Transition Interface, zero Decisions, nil errors on 22 illegal edges, and zero observed time incorrectly allowing refund exposed the required behavior before implementation
artifact_or_environment: writer worktree external-package tests
unverified_boundary: Red evidence alone does not prove Green, persistence, or external integration
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
phase: writer
command_or_action: focused and race count 20; eight-mutation shield; fresh MySQL 8.0.46 full services test and race; vet; build; smoke; static scope and sensitive audits
exit_result: PASS
sanitized_summary: public Transition seam, strict cancel boundary, complete matrix, determinism, eight mutation sensitivities, full adjacent MySQL regression, and static writer gates passed
artifact_or_environment: writer worktree plus disposable loopback-only MySQL 8.0.46
unverified_boundary: pure Module and adjacent MySQL regression do not prove DB state-writer transactions, real refund, redemption, notification, QR/token, integration, or production
external_asset:
  owner: writer/verifier
  missing: writer N/A; verifier must independently recreate
  recovery: rerun all declared commands from exact candidate in fresh detached worktree
```
