## Why

仓库已经具备 OpenSpec、七态生命周期、独立验证和受控自进化规则，但当前状态仍分散在 change、checkpoint、Git 与会话交接中，新 session 不能用一个命令可靠回答“现在做到哪、下一步是什么、哪些证据缺失”。需要把现有规则收敛成一个轻量、仓库本地、可检查的执行闭环，让开发更快、更顺、更标准，而不是继续增加人工文档。

## What Changes

- 新增零第三方依赖的 `./tools/harness` 入口，统一提供当前状态、完整性检查、证据化 checkpoint 与自进化 observation 记录。
- 在 Git common dir 的专属 `codex-harness/` 子目录维护机器可读的本地控制账本，使同一仓库的多个 worktree/session 共享状态且不污染业务 diff；task 完成度从现有 `tasks.md` 推导，不复制 task 清单。
- checkpoint 只允许既有七态的合法前进或失败回流，状态必须携带下一步和证据引用；工具可验证的 Git、路径与 task 事实必须现场验证。
- observation 只追加并固定分类，不会在当前模块自动修改规则；合格经验仍通过后续独立 OpenSpec change 晋升。
- 保持受保护的根治理与 `order-run-loop` 字节不变；本 change 只交付可显式运行的工具，接入 runner 必须由后续拥有独立 judge 的控制面 change 完成。

## Capabilities

### New Capabilities

<!-- none -->

### Modified Capabilities

- `loop-engineering-control-plane`: 将已有 ledger、checkpoint、恢复和受控自进化要求落实为一个仓库本地、机器可读、可执行检查的轻量闭环入口。

## Impact

- owner：branch `codex/add-lightweight-harness-loop`、worktree `/Users/vivix/.codex/worktrees/order-harness-loop.Writer` 的唯一 writer。
- owned paths：
  - `openspec/changes/add-lightweight-harness-loop/**`
  - `tools/harness`
  - `tools/tests/test_harness.py`
- 只读共享契约：根 `AGENTS.md`、`.agents/skills/order-run-loop/**`、四个 `$order-*` stage skills、`docs/quality/change-quality-gates.md`、生命周期 receipt 工具与历史归档。
- 运行时本地状态：只允许 `./tools/harness` 原子写入 Git common dir 下的 `codex-harness/state.json` 与 `codex-harness/observations.jsonl`；这些文件是可重建的调度索引，不是产品、生命周期或独立验证证明。
- 依赖：本地 main 已集成的 `loop-engineering-control-plane`、`change-quality-gates` 和 `lifecycle-receipt-control`；无其他 active change 依赖。
- `gate_type=W1`；`ui_level_target=UI0`、`ui_level_actual=UI0`。这是内部开发控制逻辑，不改变产品 UI、公共 API、数据、支付或部署行为。
- 外部资产：无；全部验收使用仓库和本机 Python/Git/OpenSpec。
- 非目标：引入数据库、服务、daemon、CI、第三方包或图编排框架；自动批准、自动集成、自动推送、自动部署；替代 OpenSpec、Git、独立 verifier 或 receipt；在本 change 内修改或激活受保护 runner；修复 README/业务功能；归档三个 legacy change。
- 最小成功标准：在干净 checkout 中，`./tools/harness status` 能从 OpenSpec、Git 和账本显示所有 active change 及每个 task 状态；`./tools/harness check` 对缺失/非法/陈旧状态返回非零；checkpoint 和 observation 通过工具受控写入并可被下一 session 恢复；完整回归与 clean-detached fresh-session 验收通过。
