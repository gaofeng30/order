## ADDED Requirements

### Requirement: Repository governance has one authoritative source

仓库 MUST 以根 `AGENTS.md` 作为跨工具硬规则的单一事实源，以 `.agents/skills/` 作为项目执行流程的单一事实源；工具专属入口不得复制完整规则或维护冲突流程。

#### Scenario: Claude or Cursor starts a repository task

- **WHEN** Claude 或 Cursor 加载各自的项目入口
- **THEN** 工具入口要求先读取根 `AGENTS.md`
- **AND** 工具专属 skill 通过薄 wrapper 指向 `.agents/skills/` 的对应流程

#### Scenario: A legacy planning or TDD skill is invoked

- **WHEN** Cursor 触发旧 `planning-with-files` 或 `test-driven-development` 名称
- **THEN** 兼容入口转交 `order-plan-change` 或 `order-implement-tdd`
- **AND** 不创建 OpenSpec 之外的第二事实源

### Requirement: Each change has one independently acceptable capability

每个 OpenSpec change MUST 只承载一个可独立理解、实现、验收和回滚的主能力，并声明唯一 owner、owned paths、依赖和验收标准。

#### Scenario: A proposal contains separable outcomes

- **WHEN** proposal 中的两个结果可以分别验收或回滚
- **THEN** 团队将其拆成两个 change
- **AND** 通过显式依赖表达先后关系，而不是放入同一个大 change

#### Scenario: A change is ready for implementation

- **WHEN** proposal、design、capability spec 和 tasks 已完成
- **THEN** 主目标、非目标、owned paths、依赖和验收命令均不存在会改变行为的未决项
- **AND** change 才能进入 IMPLEMENTING

### Requirement: Independent changes can progress in parallel

团队 MUST 允许无依赖且 owned paths 不冲突的 changes 在不同 worktree 并行推进，不得用其他 change 的本地 Gate 全局阻塞研发。

#### Scenario: Two changes have disjoint ownership

- **WHEN** 两个 changes 没有依赖关系、公共契约冲突或 owned paths 重叠
- **THEN** 两个 writer 可以同时在各自 worktree 规划、实现或验证

#### Scenario: Two changes contend for a shared path

- **WHEN** 两个 changes 需要修改同一个文件或公共契约
- **THEN** 该路径同时只分配给一个 writer
- **AND** 另一个 change 声明依赖或重新划定边界

### Requirement: Implementation follows change-local TDD

行为、契约、配置或文档实现 MUST 先产生可观察的失败证据，再完成最小实现并在重构后重跑同一验证。

#### Scenario: Behavior is implemented

- **WHEN** writer 开始实现 change 的行为要求
- **THEN** tasks 记录 Red 的失败测试或验收结果
- **AND** tasks 记录 Green 与 Refactor 后的通过命令

#### Scenario: A change is not conventional application code

- **WHEN** change 只涉及文档、配置、迁移或文件移动
- **THEN** writer 使用链接、结构、契约、迁移或内容完整性检查建立 Red 和 Green 证据
- **AND** 不为满足形式创建无业务价值的测试

### Requirement: Candidate verification is bound to an exact SHA

change MUST 在实现 worktree 完成本地验证并产生候选提交后，由另一干净 worktree 对精确 SHA 进行只读验证。

#### Scenario: Candidate passes independent verification

- **WHEN** verifier 在 detached worktree 检出候选 SHA 并执行 specs 与 tasks 的验收命令
- **THEN** 验证结果记录候选 SHA、命令和结果
- **AND** change 进入 INDEPENDENT_VERIFIED

#### Scenario: Candidate changes after verification

- **WHEN** 实现、规格、任务或提交 SHA 在验证后发生变化
- **THEN** 旧验证 MUST 失效
- **AND** 新 SHA MUST 重新经过独立验证

#### Scenario: Verification fails

- **WHEN** verifier 发现任一验收条件失败
- **THEN** verifier 不在验证 worktree 修改业务文件
- **AND** writer 在实现 worktree 修复并提交新的候选 SHA

### Requirement: Integration preserves verified behavior

只有依赖满足且候选 SHA 通过独立验证的 change 才能集成；集成基线变化后 MUST 重跑受影响验证，集成完成后才能 archive。

#### Scenario: Main has advanced since candidate creation

- **WHEN** change 在集成前需要 rebase、merge 或重建到新的 main
- **THEN** 更新后的提交视为新候选 SHA
- **AND** 重跑本地及独立验证

#### Scenario: Change is archived

- **WHEN** change 已集成 main 且所需检查通过
- **THEN** 团队执行 OpenSpec archive
- **AND** 未集成 change 不得仅因实现完成而 archive
