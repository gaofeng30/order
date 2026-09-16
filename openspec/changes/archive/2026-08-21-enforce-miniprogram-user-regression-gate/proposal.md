## Why

当前仓库虽然在根规则、质量文档和 stage Skills 中要求 Red → Green → Refactor 与 UI 证据，但 `openspec validate` 不校验这些语义，`tools/harness` 也能在没有小程序用户侧回归计划或 exact-SHA 运行凭证时推进生命周期。单靠 Agent 勾选任务和文字 PASS 无法保证“每个小程序模块同时经过 TDD 和用户侧回归”。

本 change 把该约定变成 fail-closed 控制面：小程序 change 缺任一 Gate 时，机器校验必须拒绝对应状态推进，且旧模块的既有门禁不得被削弱。

## What Changes

- 为所有触及 `apps/wechat-miniprogram/**` 的新 OpenSpec 增加结构化 `miniprogram-gates.json`：声明真实 Red/Green/Refactor task/command、用户主路径与相邻回归场景、原生能力及最低 UI 等级。
- 固定用户侧最低标准：普通小程序模块至少 UI2；涉及 `wx.login`、手机号、支付、扫码或订阅消息等平台原生能力至少 UI3。
- 扩展本地 Harness：形成 `CANDIDATE` 前校验 TDD 三段任务与 exact candidate manifest；进入 `INDEPENDENT_VERIFIED` 前要求绑定同一 candidate SHA、覆盖全部场景且达到最低等级的结构化用户回归凭证。缺失、低等级、旧 SHA、场景不全或 `BLOCKED_EXTERNAL` 均拒绝推进。
- 增加标准库 checker、正反例回归和最小上下文 forward-test，覆盖缺 manifest、假完成 TDD、缺用户回归、低等级、原生能力漏报、旧 SHA 复用及完整合法链路。
- 最小更新根规则、质量协议、OpenSpec tasks 规则及 plan/implement/verify/integrate Skills，使人工流程与机器 Gate 指向同一事实源。

## Capabilities

### New Capabilities

- `miniprogram-user-regression-gate`: 定义小程序 OpenSpec 的结构化 TDD/用户侧回归计划、UI2/UI3 分级、exact-SHA 凭证与机器阻断行为。

### Modified Capabilities

- `change-quality-gates`: 将小程序 TDD 与用户侧回归提升为不可由评分或低等级证据替代的硬 Gate。
- `loop-engineering-control-plane`: 生命周期推进读取机器校验结果；小程序 exact-SHA 用户回归缺失时不得进入 `INDEPENDENT_VERIFIED` 或 `INTEGRATED`。

## Impact

- `gate_type=W1`：只改变仓库内部开发控制与验证行为；`ui_level_target=UI0`，不改变产品、公共 API、数据库或线上状态。
- Owner：当前主 Agent；branch `codex/enforce-miniprogram-user-regression-gate`；worktree `/Users/vivix/.codex/worktrees/order-enforce-miniprogram-user-regression-gate.Writer`。
- `base_sha=2299da6013c06f4ae7dcad535373c67b66c80ebb`。
- Frozen runner：`order-run-loop` blob `0f41f64ad87fc9fd410cb916b4d1562aee92e42f`，SHA256 `2781bdda1544106e30e7483c4b500d611df85c79753cc1bd3b717d91d1edaac8`，version `unversioned`。
- Owned paths：本 change 的 `openspec/changes/enforce-miniprogram-user-regression-gate/**`；`AGENTS.md`；`openspec/config.yaml`；`docs/quality/change-quality-gates.md`；`tools/harness`；`tools/miniprogram_gate.py`；`tools/tests/test_harness.py`；`tools/tests/test_miniprogram_gate.py`；`.agents/skills/order-plan-change/SKILL.md`、`order-implement-tdd/SKILL.md`、`order-verify-change/SKILL.md`、`order-integrate-change/SKILL.md`。
- Read-only：`.agents/skills/order-run-loop/**`（含 self-evolution 协议）、全部产品代码、PRD、公共 API/数据 spec、现有 lifecycle receipt 工具与历史归档。
- Dependency：现有 `add-lightweight-harness-loop@79c4f1d8ebf300bf2f9f4c226fcd2c2aa2643963` 已集成 main；本 change 不依赖未集成的 session 客户端候选。
- DRAFT admission：reproducible evidence 为 `tools/harness` 当前 `validate_checkpoint_args/validate_record` 不解析 TDD 或用户回归语义；该缺口对所有未来小程序 change 通用且属于防止假验收的安全关键规则；本 change 明确 non-weakening；回归计划为 Harness/checker 正反例与全仓治理测试；forward-test 为 clean detached exact-SHA 中创建最小小程序 change，证明缺 Gate 失败、完整 UI3 链路通过。
- Required local assets：Python 3 标准库、Git、OpenSpec CLI；均已具备。无外部资产。
- Non-goals：不修改或集成现有 session 候选、不执行真实微信回归、不修改小程序业务代码、不新增 GitHub workflow、不变更远端 required check/branch protection、不推送/PR/部署/归档。
- Activation：本规则只有 independently verified、授权集成 local main 后，才对其后冻结 runner base 的小程序模块生效；当前模块和旧 candidate 不原地切换规则。main 推进后，旧 session candidate 若要集成必须按既有 moving-main 规则形成新 candidate，并接受本 Gate。
- 唯一 ACCEPT：原始 false-green Red 可复现；实现后全部正反例、non-weakening、strict、owned/protected、完整回归、clean-detached exact-SHA forward-test 与独立验证均 PASS。任一缺失或 `BLOCKED_EXTERNAL` 为 REJECT；C/T/V/R 不得补偿。
