## Context

项目当前是单仓库原型，下一阶段会同时建设微信小程序、Web 管理端、Go API、数据库和 CI/CD。开发者只有两人，但会借助多个 Agent 和 worktree 并行工作。治理的核心不是增加审批层级，而是让每个并行单元都有清晰边界、单一写入者和可复现证据。

## Goals / Non-Goals

**Goals:**

- 用一个共享规则入口和四个职责单一的 skills 固化研发闭环。
- 让 OpenSpec change 小到可独立理解、实现、验收、回滚和集成。
- 支持互不依赖的 changes 并行，冲突只在真实依赖或文件所有权上发生。
- 让 TDD 和独立验证都绑定到具体 change 与精确 SHA。

**Non-Goals:**

- 不在本 change 中重构项目目录或实现业务功能。
- 不配置 GitHub 分支保护、CI、部署环境或外部权限。
- 不为尚不存在的发布体系创建 launch/release skills。
- 不追溯改造原型阶段的历史提交。

## Decisions

### D1. `AGENTS.md` 是规则单一事实源

根 `AGENTS.md` 只承载所有参与者都必须遵守的硬规则；`CLAUDE.md` 与 Cursor always-on rule 只指向它，避免复制后漂移。OpenSpec 保存单个变更的契约，`.agents/skills/` 保存可复用执行步骤。Cursor 的同名 skills 只做薄 wrapper；原 `planning-with-files` 与旧 TDD skill 也改为兼容 wrapper，不再维护第二套流程。

### D2. 一个 change 对应一个可独立验收的能力

change 必须只有一个主目标、一个验收结论、一名写入责任人和一组声明的 owned paths。若一部分可以在不依赖另一部分的情况下独立上线或回滚，就拆成不同 change；共享前置条件通过显式依赖表达，不合并成“大 spec”。

不使用固定 LOC 或任务数量限制，因为文档移动与订单状态机的复杂度不能用同一数字衡量。以业务能力和验收边界判断大小。

### D3. Gate 只作用于当前 change

单个 change 使用 `DRAFT → APPROVED → IMPLEMENTING → CANDIDATE → INDEPENDENT_VERIFIED → INTEGRATED → ARCHIVED` 状态流。一个 change 未完成只阻塞依赖它或与它争用 owned paths 的 change，不阻塞无关模块在其他 worktree 继续研发。

### D4. TDD 证据写回 tasks

行为实现遵循 Red → Green → Refactor。Red 记录失败测试或失败验收命令，Green 记录最小实现后的通过命令，Refactor 后重跑相同验证。文档、配置和纯移动使用链接、结构、契约或内容完整性检查作为测试，不为追求形式编造单元测试。

### D5. 验证者使用独立干净 worktree

实现者本地验证通过后提交候选 SHA。验证者从该 SHA 创建 detached worktree，只运行 specs 和 tasks 要求的检查，不编辑业务文件。SHA、规格、任务或实现发生任何变化，旧验证立即失效；失败返回实现 worktree 修复并产生新候选 SHA。

### D6. 四个最小项目 skills 覆盖完整闭环

- `order-plan-change`：边界、依赖、owned paths、验收和 OpenSpec artifacts。
- `order-implement-tdd`：按 change 执行 Red → Green → Refactor。
- `order-verify-change`：在独立 worktree 验证精确 SHA。
- `order-integrate-change`：检查依赖、验证状态、主分支更新和归档条件。

评审、上线和发布在相应基础设施出现后再独立建 change 与 skill，当前不提前设计。

## Risks / Trade-offs

- [change 拆得过细造成协调成本] → 以独立验收和回滚为边界，不按文件数量机械拆分。
- [两个 worktree 修改同一共享文件] → 创建 change 时声明 owned paths，同一路径同一时刻只允许一个 writer。
- [验证后又修改导致证据失真] → SHA 变化即强制重新验证，不允许口头沿用结果。
- [工具专属规则发生漂移] → 工具入口保持生成或薄引用，业务规则只写在 `AGENTS.md` 与项目 skills。

## Migration Plan

1. 完成 change-governance spec 和 tasks。
2. 初始化 OpenSpec 的 Codex、Cursor 工具入口。
3. 建立根规则入口和四个项目 skills。
4. 验证 OpenSpec、skills、链接和生成文件。
5. 在独立 worktree 验证候选 SHA，集成后再 archive 本 change。

回滚时整体撤销本 change；不会影响现有原型运行。

## Open Questions

无。
