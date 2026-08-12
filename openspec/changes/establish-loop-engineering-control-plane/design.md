## Context

仓库已有 `$order-plan-change`、`$order-implement-tdd`、`$order-verify-change`、`$order-integrate-change` 四个 change 级执行 skill，也已有唯一 OpenSpec 状态流，但缺少跨 change 的常驻控制面。没有控制面时，主 Goal 容易重复开 session、过早建立 verifier、按文档数量而非阻断风险调度，或用总分掩盖 P0/P1 和 strict FAIL。

本 change 只新增 `.agents/skills/order-run-loop/` 的薄编排层；根治理、四个既有 skill、业务代码和产品文档都是只读依赖。当前规划状态为 `DRAFT`，批准前不创建 skill 文件。

方法总纲唯一采用 Addy Osmani 于 2026-06-07 发布的 [Loop Engineering](https://addyosmani.com/blog/loop-engineering/)（访问日期：2026-08-12）。[Ralph](https://ghuntley.com/ralph/) 是更早的简单执行环；[ReAct](https://arxiv.org/abs/2210.03629) 与 [Reflexion](https://arxiv.org/abs/2303.11366) 是前置研究机制。它们只用于说明谱系，不作为本 skill 的同义名称或并列协议。

## Goals / Non-Goals

**Goals:**

- 用一个常驻主 Goal 在固定 session 预算内调度最多两条独立 change lane。
- 以仓库状态机、精确 SHA 和可执行证据驱动每个转移。
- 固定阻断项选择、失败回流、外部阻塞、人工升级、评分和停止算法。
- 通过薄路由复用四个既有 order skill，不复制其 change 内步骤。
- 为批准后的 skill 提供确定的目录、metadata、初始化和验证方法。

**Non-Goals:**

- 不修改根 `AGENTS.md`、既有四个 skill、业务代码或产品文档。
- 不在本轮创建 `.agents/skills/order-run-loop/`，不进入实现或验证状态。
- 不引入新的 agent 框架、队列、锁服务、持久化服务或外部依赖。
- 不把 readiness 文档评分当作任一 change 的代码完成、独立验证或集成证据。

## Decisions

### 1. 只实现薄控制面

`order-run-loop` 仅维护全局队列、lane/session 预算、状态证据、评分和停止判断。change 内动作按状态唯一转发：规划调用 `$order-plan-change`，实现调用 `$order-implement-tdd`，候选验证调用 `$order-verify-change`，集成调用 `$order-integrate-change`。

不选“把四套流程复制进一个大 skill”，因为复制会形成第二套规则源并使治理变更漂移；不选“通用 Ralph 无限循环”，因为它没有仓库的状态、权限、精确 SHA 和独立验证 Gate。

### 2. 状态只由进入证据和退出证据推进

| 转移 | 进入前提 | 唯一 handler | 退出证据 |
| --- | --- | --- | --- |
| 队列 → `DRAFT` | blocker 已缩成单一、可独立验收和回滚的 change；owner、owned paths、依赖、非目标明确 | `$order-plan-change` | proposal/design/specs/tasks 齐全；`openspec validate <change> --strict` PASS；仍无批准声明 |
| `DRAFT` → `APPROVED` | DRAFT 退出证据齐全 | 主 Agent | 明确批准事实 + 当下 strict PASS |
| `APPROVED` → `IMPLEMENTING` | 依赖满足；branch/worktree/owner/owned paths 匹配；strict PASS | `$order-implement-tdd` | 第一个可观察 Red 证据 |
| `IMPLEMENTING` → `CANDIDATE` | change-local tasks、验收、strict、owned-path 检查全部 PASS | `$order-implement-tdd` | 本地提交的完整 SHA + 命令结果 + clean status |
| `CANDIDATE` → `INDEPENDENT_VERIFIED` | 有不可变完整 candidate SHA | `$order-verify-change` | 另一 clean detached worktree 对 exact SHA 的 PASS attestation |
| `INDEPENDENT_VERIFIED` → `INTEGRATED` | 依赖与 required review 满足；有明确集成授权；attestation 未失效 | `$order-integrate-change` | 实际 main integrated SHA + 集成检查结果 |
| `INTEGRATED` → `ARCHIVED` | change 已在 main 且交付证据完整 | `$order-integrate-change` | archive 校验 PASS；归档事实 |

任何代码、spec、tasks、rebase、merge 或 SHA 变化都使旧验证失效。状态不接受 agent 自述、moving ref 或未提交 working tree 作为证据。

### 3. 主 Goal 常驻，lane slot 固定为两个

主 Goal session 持有总队列和评分，不退出换 session。总活跃上限为主 session 加两个 lane slot；每条 lane 同一时刻只有一个执行 session。writer 从 `DRAFT` 到 `CANDIDATE` 复用原 session 和 worktree。

只有完整 candidate SHA 才把所属 lane 从 writer 切换为独立 verifier；DRAFT、探索、APPROVED、未提交实现不得开 verifier。验证失败后，writer 在原 lane session 修复并产出新 SHA；随后复用原 verifier session，但为新 SHA 重建 clean detached worktree并完整重验。verifier 始终占用该 lane slot，不增加第三个执行 slot。

两条 change 只有在无依赖、无公共契约冲突且 owned paths 不重叠时并行。存在任一冲突即串行；已有两条 active lane 时，第三项只入主队列。

### 4. 每轮选择最高风险的最小阻断项

主 Goal 每轮执行以下确定算法：

1. 从未处置项中筛出 OPEN P0/P1；排除已有 active lane、依赖未满足或 owned paths/公共契约与 active lane 冲突的项。
2. 对候选按 `severity 降序 → 可解除的后续依赖数降序 → 独立验收边界升序 → 稳定 change 名升序` 排序。
3. 取首项，并缩到仍能独立理解、验收和回滚的最小 owned-path 边界；若无法满足则先规划依赖 change。
4. 若需要真实资质、账号、密钥或外部事实且本地不可得，记为 `BLOCKED_EXTERNAL`，记录 owner、缺失证据和恢复条件，再继续选择无依赖的本地项。

不选“先到先得”或“分数最低类别优先”，因为它们可能让低影响大任务抢占真正 P0/P1，或把不可独立验收的范围塞进单一 change。

### 5. 同一真实错误指纹第三次停止

错误指纹由失败命令、退出码、首个决定性错误、candidate SHA 或环境标识组成。一次真实失败后只做针对该错误的最小修复；同一指纹连续第三次出现时，lane 停止，保留最近 checkpoint，主动回传三次证据和一个推荐裁决，不执行第四次盲试。

人工升级只允许两类：真实外部事实/资质/密钥/不可逆授权，或同指纹三次失败。普通产品或技术决策由主 Agent 按已确认 PRD、OpenSpec、公共契约和仓库规则作单一裁决。

### 6. session 主动回传，主会话只做增量快照

每个执行 session 在完成当前阶段、阻塞或需要裁决时主动回传不超过 10 行：change、状态、结论、完整 SHA（无则明确写无）、验证、自评、blocker/next。主 Goal 用 cursor 执行 `timeoutMs: 0` 非阻塞快照，只消费新增事件；仍有规划、选择或证据核对可做时继续本地工作，只有无其他可执行规划时才做一次有界等待。

不选固定频率轮询，因为不变快照会消耗上下文并诱发重复 session；不选无限等待，因为主 Goal 仍承担调度与决策。

### 7. 固定 100 分 rubric，硬阻断优先于分数

| 维度 | 权重 | 得分证据 |
| --- | ---: | --- |
| 产品决策/范围 | 25 | 已确认决策、边界、owner、非目标和验收对象可追溯 |
| 跨端事实源与状态 | 15 | 唯一事实源、状态机、跨端读写责任和异常转移已决定 |
| 资金/库存/鉴权 | 15 | 支付退款、库存并发、鉴权责任及失败语义有可执行契约 |
| API 与可执行验收 | 15 | API 契约、调用方/时机及真实可运行验收命令完整 |
| 架构/数据/恢复 | 15 | 架构边界、数据一致性、迁移/备份/恢复路径已决定 |
| 质量门禁/独立验证 | 10 | strict、TDD、exact-SHA 独立验证和集成 Gate 可执行 |
| 外部依赖治理 | 5 | 外部 owner、缺失资产、恢复条件和 `BLOCKED_EXTERNAL` 完整 |

`Score = Σ 每个维度已取得的证据分`，总分上限 100。停止准备度 Goal 的唯一公式为：

`READY = Score >= 85 AND OPEN(P0/P1) = 0 AND first_blocking_openspec_strict = PASS`

任一硬条件不满足即 `NO-GO`，不得用其他维度超额得分覆盖。实现 Goal 另按每个 change 的验收、候选 SHA、独立验证和集成状态停止；只写完文档不算代码完成。

### 8. 批准后按 skill-creator 确定性生成

实现开始后先执行一次结构性 Red，再使用 skill-creator 的初始化器生成目录和 metadata：

```sh
python3 /Users/vivix/.codex/skills/.system/skill-creator/scripts/init_skill.py order-run-loop \
  --path .agents/skills \
  --interface 'display_name=Order Run Loop' \
  --interface 'short_description=有界调度 OpenSpec change 并以证据闭环长任务' \
  --interface 'default_prompt=Use $order-run-loop to coordinate this goal through bounded OpenSpec change lanes.'
```

初始化不传 `--resources`。最终目录只保留 `SKILL.md` 和 `agents/openai.yaml`；`SKILL.md` frontmatter 只有准确的 `name`、触发型 `description`，正文使用祈使式并保持少于 500 行；YAML string 全部带引号，`default_prompt` 显式包含 `$order-run-loop`。完成后运行：

```sh
python3 /Users/vivix/.codex/skills/.system/skill-creator/scripts/quick_validate.py .agents/skills/order-run-loop
```

不选手工搭目录，因为初始化器和验证器共同固定结构；不创建 scripts/references/assets/README，因为该控制面没有独立工具或大体量参考材料。

## Risks / Trade-offs

- [两个 lane 可能降低表面并行度] → 以无冲突、可验证的吞吐优先；等待项留在主队列，不扩 session。
- [评分可能被主观抬高] → 每分必须绑定决策、artifact 或命令证据，且三个硬 Gate 永远优先。
- [薄 skill 可能遗漏 change 内细节] → 只路由到四个现有 skill；细节随各自单一事实源演进。
- [错误指纹过宽或过窄] → 固定使用命令、退出码、决定性错误、SHA/环境，升级时一并提交原始证据。
- [外部阻塞长期悬挂] → 强制记录 owner 和恢复条件；它保留 NO-GO 事实，但不占无依赖本地 lane。

## Migration Plan

1. 本 change 保持 `DRAFT`，完成四件规划 artifact 与 strict validation，并提交仅规划文件的本地 commit。
2. 主 Agent 明确批准后，由原 writer 在同一 branch/worktree 使用上述 `init_skill.py` 命令进入实现。
3. 依 tasks.md 完成 Red/Green/Refactor、自测、strict 与 owned-path 检查，形成 candidate SHA。
4. 在另一 clean detached worktree 对 exact SHA 独立验证；通过后才允许按现有集成 skill 推进。
5. 回滚仅删除本 change 新增的 `.agents/skills/order-run-loop/` 并恢复对应 OpenSpec 状态；四个既有 skill 和根治理无需迁移。

## Open Questions

无。协议、评分权重、session 上限、升级边界和 skill metadata 已由主 Goal 固定；新的产品或技术分歧应作为普通主 Agent 裁决，不在本 change 留占位。
