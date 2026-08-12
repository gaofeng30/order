# loop-engineering-control-plane Specification

## Purpose
定义 order 主 Goal 如何以有界 session、证据门禁和固定评分循环调度独立 OpenSpec changes。
## Requirements
### Requirement: Control plane uses one named loop-engineering method
`order-run-loop` MUST 采用 Addy Osmani 于 2026-06-07 发布的《Loop Engineering》作为方法总纲，并明确 Ralph 是更早的简单执行环、ReAct 与 Reflexion 是前置研究机制；skill 不得把三者写成同一方法或同义名称。

#### Scenario: Method lineage is inspected
- **WHEN** reviewer 检查 `order-run-loop` 的方法来源说明
- **THEN** skill 明确引用 Addy Osmani 2026-06-07《Loop Engineering》
- **AND** Ralph、ReAct、Reflexion 只以各自的历史定位出现

### Requirement: Control plane is a thin router over existing change skills
`order-run-loop` MUST 只承担主 Goal 的选择、调度、证据判断和停止决策，并分别把 change 规划、实现、验证和集成交给 `$order-plan-change`、`$order-implement-tdd`、`$order-verify-change`、`$order-integrate-change`；不得复制四个 skill 的步骤、测试细节或集成规则。

#### Scenario: All existing roles are reachable
- **WHEN** reviewer 检查 `order-run-loop/SKILL.md`
- **THEN** 文件显式引用四个既有 `$order-*` skill
- **AND** 每个状态只指向一个职责明确的 handler

#### Scenario: Thin layer is checked for duplication
- **WHEN** reviewer 将新 skill 与四个既有 skill 对比
- **THEN** 新 skill 只保留跨 change 的控制面契约
- **AND** 不包含既有 skill 的完整标题、任务步骤或替代实现

### Requirement: Every lifecycle transition is evidence-gated
控制面 MUST 使用仓库唯一状态流 `DRAFT → APPROVED → IMPLEMENTING → CANDIDATE → INDEPENDENT_VERIFIED → INTEGRATED → ARCHIVED`，并为每个转移同时定义进入前提、handler 和退出证据。状态不得依据 agent 自述推进。

#### Scenario: Draft becomes approved
- **WHEN** proposal、design、specs 和 tasks 完整且 strict validation 通过
- **THEN** change 仍保持 `DRAFT`，直到主 Agent 获得明确批准
- **AND** 批准事实与 strict PASS 共同构成进入 `APPROVED` 的证据

#### Scenario: Approved change starts implementation
- **WHEN** change 已批准、依赖满足、owner/worktree 与 owned paths 匹配且 strict validation 通过
- **THEN** 控制面调用 `$order-implement-tdd`
- **AND** writer 开始第一个 Red 后 change 才进入 `IMPLEMENTING`

#### Scenario: Implementation produces a candidate
- **WHEN** change-local tasks、验收命令与 strict validation 全部通过且 diff 未越过 owned paths
- **THEN** writer 提交并回传完整候选 SHA
- **AND** `CANDIDATE` 退出证据包含该 SHA、本地命令结果和 clean status

#### Scenario: Candidate becomes independently verified
- **WHEN** `$order-verify-change` 在另一 clean detached worktree 对完整候选 SHA 执行全部验收
- **THEN** 只有 exact-SHA PASS attestation 可进入 `INDEPENDENT_VERIFIED`
- **AND** 代码、spec、tasks、rebase、merge 或 SHA 后续变化使旧 attestation 立即失效

#### Scenario: Verified candidate is integrated and archived
- **WHEN** 依赖满足、候选 exact-SHA PASS、required review 通过且已获得集成授权
- **THEN** 控制面调用 `$order-integrate-change` 并分别记录 candidate SHA 与 integrated SHA
- **AND** 只有实际集成 main 且集成检查通过后才能进入 `ARCHIVED`

### Requirement: Main Goal and change lanes have fixed session limits
主 Goal session MUST 常驻并维护全局状态。每轮最多同时运行两条独立 change lane，总活跃执行 session 不得超过主 session 加两个 lane slot；每条 lane 同一时刻只能有一个 writer，且从 `DRAFT` 到 `CANDIDATE` 复用同一 session 和 worktree。

#### Scenario: Independent changes run in parallel
- **WHEN** 两个 change 无依赖、无公共契约冲突且 owned paths 不重叠
- **THEN** 主 Goal 可以同时调度两条 change lane
- **AND** 每条 lane 保持自己的单 writer session/worktree

#### Scenario: A third lane is proposed
- **WHEN** 已有两条 active change lane
- **THEN** 控制面不得创建第三条 lane 或额外 writer session
- **AND** 新 blocker 留在主 Goal 队列等待 slot

#### Scenario: Dependencies or ownership conflict
- **WHEN** 两个候选 change 存在依赖、公共契约冲突或 owned paths 重叠
- **THEN** 控制面将它们串行化
- **AND** 不用额外 session 或锁掩盖冲突

### Requirement: Verifier starts only for an exact candidate
控制面 MUST 只在完整候选 SHA 存在时启用一个独立 verifier。DRAFT、探索、未提交 working tree 或 moving ref 不得创建 verifier；验证失败后的新 SHA 复用原 verifier session，但必须重建 clean detached worktree。verifier 占用所属 lane slot，不能扩张总 session 上限。

#### Scenario: Draft requests early verification
- **WHEN** change 仍为 DRAFT、APPROVED、IMPLEMENTING 或没有完整候选 SHA
- **THEN** 控制面拒绝创建 verifier
- **AND** 由当前 planner/writer 完成本阶段的自检

#### Scenario: Failed candidate is repaired
- **WHEN** verifier 对候选 SHA 返回 FAIL 且 writer 产生新 SHA
- **THEN** 控制面复用同一个 verifier session
- **AND** verifier 为新 SHA 重建 clean detached worktree 并从头执行验收

### Requirement: Scheduler chooses the smallest highest-risk blocker
每轮 MUST 先在未处置的 P0/P1 中选择当前最高风险项，再将其缩小为一个可独立理解、验收和回滚的 change。风险相同时优先选择能解除最多后续依赖的最小 owned-path 边界；P2 或普通优化不得抢占仍可执行的 P0/P1。

#### Scenario: Multiple local blockers are eligible
- **WHEN** 队列中同时存在多个无依赖冲突的 P0/P1
- **THEN** 控制面先比较 severity，再比较被解锁依赖，最后比较验收边界大小
- **AND** 只把排序最高且可独立验收的 blocker 分配给空闲 lane

#### Scenario: Blocker depends on unavailable external assets
- **WHEN** blocker 需要真实资质、账号、密钥或外部事实且本地无法取得
- **THEN** 控制面将其记为 `BLOCKED_EXTERNAL` 并记录 owner、缺失证据和恢复条件
- **AND** 不把它记为 PASS，也不阻塞无依赖且 owned paths 不冲突的本地 lane

### Requirement: Escalation and retry policy is deterministic
控制面 MUST 以失败命令、退出码、首个决定性错误、候选 SHA 或环境标识形成真实错误指纹。同一指纹连续第三次出现时必须停止该 lane、保留 checkpoint 并请求人工裁决，不得执行第四次盲试。人工升级仅允许真实外部事实/资质/密钥/不可逆授权或同指纹三次失败；普通产品与技术决策由主 Agent 按已确认契约裁决。

#### Scenario: Same real error repeats three times
- **WHEN** 同一错误指纹在修复循环中连续第三次出现
- **THEN** lane 进入需人工裁决状态并停止重试
- **AND** 回传包含三次证据、当前 checkpoint 和一个推荐决策

#### Scenario: Ordinary implementation choice appears
- **WHEN** 选择不涉及外部事实、不可逆授权或已触发三次失败
- **THEN** 主 Agent 依据 PRD、OpenSpec 和仓库规则作出单一裁决
- **AND** 不为普通产品或技术判断创建人工升级

### Requirement: Sessions actively hand off compact evidence
每个执行 session 在完成、阻塞或需要裁决时 MUST 主动向主 Goal 回传不超过 10 行，至少包含 change、状态、结论、完整 SHA 或明确无 SHA、验证结果、自评和 blocker/next。主 Goal MUST 使用带 cursor 的非阻塞快照收敛状态；只有没有其他可执行规划时才进行一次有界等待，不得轮询不变状态。

#### Scenario: Lane reaches a terminal point for the turn
- **WHEN** lane 完成当前阶段、被阻塞或需要裁决
- **THEN** lane 无需主 Goal 轮询即主动发送不超过 10 行的证据摘要
- **AND** 主 Goal 可仅依据新 cursor 增量更新控制面状态

#### Scenario: Other planning remains available
- **WHEN** 主 Goal 仍有可执行的选择、规划或证据核对
- **THEN** 主 Goal 只取非阻塞快照并继续本地工作
- **AND** 不进入有界等待

### Requirement: Readiness score is fixed and cannot override hard blockers
开发准备度 MUST 使用固定 100 分 rubric：产品决策/范围 25、跨端事实源与状态 15、资金/库存/鉴权 15、API 与可执行验收 15、架构/数据/恢复 15、质量门禁/独立验证 10、外部依赖治理 5。分数只能由可追溯决策、artifact 或命令证据获得，且不得覆盖 OPEN P0/P1、strict FAIL、未授权或真实外部阻塞。

#### Scenario: Readiness reaches the numeric threshold
- **WHEN** 总分达到或超过 85
- **THEN** 只有 `OPEN P0/P1 = 0` 且首批阻断型 OpenSpec 全部 strict PASS 时才停止准备度 Goal
- **AND** 任一硬阻断仍存在时 verdict 仍为 NO-GO

#### Scenario: Documentation is complete but implementation is not
- **WHEN** 一个实现 Goal 只有规划文档或 OpenSpec strict PASS，尚未满足该 change 的代码验收
- **THEN** 控制面不得把它标记为实现完成
- **AND** 继续按该 change 自身的 CANDIDATE、独立验证和集成 Gate 判断

### Requirement: Skill packaging is deterministic and minimal
批准后的 writer MUST 使用 `skill-creator/scripts/init_skill.py` 在 `.agents/skills/order-run-loop/` 初始化 skill，不创建 scripts、references、assets、README 或其他辅助文件；最终只保留 `SKILL.md` 和 `agents/openai.yaml`。`SKILL.md` frontmatter 必须包含准确的 `name` 与触发型 `description`，正文使用祈使式并保持低上下文成本；`agents/openai.yaml` 必须使用带引号的 string，且 `default_prompt` 显式包含 `$order-run-loop`。

#### Scenario: Skill is initialized after approval
- **WHEN** change 获得批准并进入 apply
- **THEN** writer 使用 `init_skill.py order-run-loop --path .agents/skills` 及显式 interface 参数初始化
- **AND** 不手工搭建替代目录结构

#### Scenario: Skill package is validated
- **WHEN** writer 完成 `SKILL.md` 和 `agents/openai.yaml`
- **THEN** `skill-creator/scripts/quick_validate.py .agents/skills/order-run-loop` 返回成功
- **AND** structure、metadata、四 skill 引用、薄层无复制、状态机、评分表、session/升级/停止规则和 owned-path 检查全部通过
