## Why

`enforce-miniprogram-user-regression-gate@f5719e98690d0b1301ed567c8b616e074846b445` 已完成 exact-SHA 验证、集成、归档和后置 binding，但其不可变 `goal-checkpoint.md` 早于当前 lifecycle receipt 约定：表格记录 `lifecycle=IMPLEMENTING`、`candidate_sha=none`，且没有 `## Module base`/`## Current Runtime`。现有受保护 `verify_receipt.py` 因而确定性返回 `checkpoint runtime section count must be 1, got 0`；直接修改它或 `profile_runner.py` 又会破坏已绑定控制面的精确字节信任。

现在需要一个只承认该 exact 历史对象的兼容验证入口，在不改写历史、不放宽普通 receipt、不使旧 `--list` 退化的前提下完成原治理 change 的 receipt closure。

## What Changes

- 新增一个 exact-target-only lifecycle receipt 兼容验证器；它固定旧 checkpoint/task 摘要、受保护 v1 judge/runner/schema/binding blobs、archive ancestry 和唯一 receipt 路径，只把这一份已知的旧 checkpoint 解释为 pre-convention candidate evidence。
- 为该 exact change 使用不被 v1 `--list` 枚举的 `legacy-lifecycle-receipt.md`；兼容验证器复用受保护 v1 的 schema、Git/archive/owned-path/task/profile 验证，并额外输出原始 `IMPLEMENTING`/`none` 与兼容解释边界。
- 增加正反例测试：精确历史对象通过；任意 checkpoint/hash/change/path/blob/receipt/history/机械 profile 变化 fail closed；现有 v1 receipt `--list`/`--chain` 和 Mini Program profile 保持通过。
- 仅在本 change 独立 exact-SHA 验证、local-main 集成和归档后，才允许后续 integrator 单独添加该 legacy receipt；receipt-head 仍需另一独立 Agent 在 clean detached exact SHA 派生 PASS。

## Capabilities

### New Capabilities

- `legacy-miniprogram-gate-receipt`: 仅为固定 governance candidate `f5719e9…` 提供非弱化、可审计的 pre-convention receipt 兼容验证。

### Modified Capabilities

- `miniprogram-gate-lifecycle-profile`: 将该 exact governance change 的后置 receipt 路径和验证命令改为 legacy 专用入口，其他 change 不变。
- `loop-engineering-control-plane`: 声明标准 v1 receipt 的唯一 exact 历史例外及其不可扩展、不可改写边界。

## Impact

- Owner：当前主 Agent；branch `codex/admit-legacy-miniprogram-gate-receipt`；worktree `/Users/vivix/.codex/worktrees/order-admit-legacy-miniprogram-gate-receipt.Writer`。
- Owned paths：
  - `openspec/changes/admit-legacy-miniprogram-gate-receipt/**`
  - `tools/lifecycle-receipts/verify_legacy_miniprogram_receipt.py`
  - `tools/lifecycle-receipts/tests/test_legacy_miniprogram_receipt.py`
- Archive-only derived paths：本 change 的 dated archive move，以及 `openspec/specs/legacy-miniprogram-gate-receipt/spec.md`、`openspec/specs/miniprogram-gate-lifecycle-profile/spec.md`、`openspec/specs/loop-engineering-control-plane/spec.md` 的声明 delta。
- Post-archive delivery-only path：`openspec/changes/archive/2026-08-21-enforce-miniprogram-user-regression-gate/legacy-lifecycle-receipt.md`；不属于 candidate writer。
- Dependency：local main `07688c7cdcc957bd9fed48f8c408c929a930c23f`，其中 Mini Program profile exact binding-head 已由另一 clean-detached Agent PASS。
- Non-goals：不修改受保护 `verify_receipt.py`、`profile_runner.py`、schema、profile/binding registry、Harness、Skill、历史 checkpoint/tasks、任何小程序/Admin/Go 产品代码、公共 API/数据；不声称 actor independence、UI2/UI3 或真实微信执行；不推送、建 PR、部署或写外部系统。
- 最小成功标准：focused Red 命中当前 `checkpoint runtime section count`；Green 只接受固定历史对象并派生 `PASS_DERIVED`；全部 lifecycle receipt、Mini Program profile、OpenSpec strict 和 ending-clean Gate PASS；双独立 exact-C PASS 后才允许本地集成/归档，另一个 exact-R PASS 后才关闭原 receipt TODO。
