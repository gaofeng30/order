## ADDED Requirements

### Requirement: One lightweight command exposes the current development loop
仓库 MUST 提供零第三方依赖的可执行 `./tools/harness`，并只通过 `status`、`check`、`checkpoint` 与 `observe` 四个子命令完成状态发现、完整性验证、交接更新和自进化观察记录。工具 MUST 从当前 Git checkout、active OpenSpec change、`tasks.md` 与 Git common dir 下的专属本地状态推导结果，不得启动服务、连接外部系统、推送、部署、批准或集成 change。

#### Scenario: Fresh session discovers the current checkout
- **WHEN** 新 session 在任一仓库 worktree 运行 `./tools/harness status`
- **THEN** 输出 MUST 包含当前 HEAD、branch、dirty 状态、每个 active change 的生命周期、task 汇总、当前 task、blocker、下一步和证据摘要
- **AND** 另一个共享同一 Git common dir 的 worktree MUST 读取同一份本地控制状态

#### Scenario: JSON output supports agents and checks
- **WHEN** 调用方运行 `./tools/harness status --json`
- **THEN** 工具 MUST 输出稳定的机器可读 JSON 且不得混入人类提示文本

### Requirement: Task state is derived without duplicating the task list
工具 MUST 从每个 active change 的 `tasks.md` 解析稳定 task ID 与 checkbox；已勾选 task 为 `DONE`，state 记录的 `current_task` 为 `DOING`，blocker 指向的未完成 task 为 `BLOCKED`，其余未完成 task 为 `TODO`。本地状态 MUST NOT 复制 task 标题、顺序或完成 checkbox。

#### Scenario: Every task has one visible state
- **WHEN** `tasks.md` 含已完成、当前、阻塞和其他未完成 task
- **THEN** `status` MUST 为每个 task 输出且只输出 `DONE`、`DOING`、`BLOCKED` 或 `TODO` 之一

#### Scenario: State points to an invalid task
- **WHEN** `current_task` 或 blocked task 不存在、重复、已经勾选或彼此冲突
- **THEN** `check` MUST 返回非零并说明 `WHAT`、`WHY` 与 `FIX`

### Requirement: Checkpoints enforce the existing lifecycle without becoming proof
`checkpoint` MUST 仅接受既有状态流 `DRAFT → APPROVED → IMPLEMENTING → CANDIDATE → INDEPENDENT_VERIFIED → INTEGRATED → ARCHIVED` 的同态更新、相邻前进，以及 `CANDIDATE`/`INDEPENDENT_VERIFIED` 因失败或失效返回 `IMPLEMENTING`。每次更新 MUST 提供非空 evidence 与 next；`BLOCKED_EXTERNAL` MUST 作为 blocker 维度而不是新增 lifecycle。账本 MUST 明确标记为可重建的调度索引，不能替代 OpenSpec、exact-SHA、独立 verifier、集成或 archive 证据。

#### Scenario: Legal checkpoint is shared across worktrees
- **WHEN** main control session 对 active change 写入合法 checkpoint
- **THEN** 工具 MUST 以原子替换更新 Git common dir 下唯一状态文件
- **AND** 后续 session MUST 能读取 lifecycle、current task、blocker、next 与 evidence

#### Scenario: Illegal jump or unsupported PASS is rejected
- **WHEN** 调用方跳过生命周期状态、把 blocker 当 lifecycle、缺少 evidence/next，或为 candidate/integration 提交无效 Git SHA
- **THEN** 命令 MUST 返回非零且不得改变原状态文件

#### Scenario: Legacy state is bootstrapped explicitly
- **WHEN** 已集成但尚未采用工具的 legacy active change 需要首次登记
- **THEN** 只有显式 `--bootstrap`、完整 Git SHA、可验证 ancestry 与 evidence 同时存在时才可直接登记为 `INTEGRATED`

### Requirement: Harness checks fail closed with remediating output
`check` MUST 校验本地状态 schema、active change 覆盖、change/task 引用、生命周期、必要 Git 对象与 ancestry、blocker 结构和 observation 记录；未初始化、陈旧、缺失、重复、非法或不可验证输入 MUST 返回非零。所有失败 MUST 使用 `WHAT`、`WHY`、`FIX` 三段式信息，且不得用宽松解析、默认 PASS 或异常吞噬制造 false green。

#### Scenario: Active change lacks a checkpoint
- **WHEN** active OpenSpec change 没有本地状态记录
- **THEN** `status` MUST 显示 `UNKNOWN`
- **AND** `check` MUST 返回非零并给出最小 checkpoint 命令方向

#### Scenario: Repository and state are consistent
- **WHEN** 所有 active change、task、Git 引用与 observation 均满足契约
- **THEN** `check` MUST 返回零并输出单一 PASS 摘要

### Requirement: Self-evolution records observations but never self-edits
`observe` MUST 只追加 `candidate`、`environment`、`checker` 或 `external` 四类不可变 observation，每条包含唯一 ID、时间、summary、reproducible evidence 与 next。该命令 MUST NOT 修改 `AGENTS.md`、Skill、OpenSpec spec、Gate 或现有 observation；candidate 只有在模块边界通过独立 OpenSpec change 才能晋升和在下一模块生效。

#### Scenario: Repeated friction becomes a reviewable observation
- **WHEN** agent 用完整 class、summary、evidence 与 next 运行 `observe`
- **THEN** 工具 MUST 原子追加一条记录并由 `status` 展示未筛选 observation 数量
- **AND** 当前 runner 和 lifecycle MUST 保持不变

#### Scenario: Invalid observation cannot enter the loop
- **WHEN** class 非法、字段为空、ID 重复或 observations 文件损坏
- **THEN** `observe` 或 `check` MUST 返回非零且不得覆盖既有记录

### Requirement: The harness stays a thin convenience layer
根 `AGENTS.md` 与 `order-run-loop` MUST 只增加对 `./tools/harness` 的最小开工、checkpoint、检查和观察入口；四个 stage skills、质量 Gate、exact-SHA 失效、独立验证、lane 限制、授权边界和 receipt 规则 MUST 保持不变。Harness 失败只能暴露或记录问题，不能放宽、跳过或替代原 Gate。

#### Scenario: Existing governance remains authoritative
- **WHEN** reviewer 对候选执行 thin-layer 与 non-weakening 检查
- **THEN** 新入口 MUST 只索引和校验现有事实
- **AND** 不得复制 stage skill 步骤、自动改变业务代码或把 recorded state 当成独立 PASS
