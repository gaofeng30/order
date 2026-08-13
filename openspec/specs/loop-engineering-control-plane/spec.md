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
批准后的 writer MUST 原地升级 `.agents/skills/order-run-loop/`，最终只保留精简的 `SKILL.md`、现有 `agents/openai.yaml` 和一层 `references/self-evolution.md`；不得创建重复 Skill、scripts、assets、README、嵌套 references 或其他运行资源。`SKILL.md` frontmatter 必须包含准确的 `name` 与触发型 `description`，正文使用祈使式并保持低上下文成本；详细自进化协议只放在该一层 reference，不复制四个 stage skills。`agents/openai.yaml` 必须继续使用带引号的 string，且 `default_prompt` 显式包含 `$order-run-loop`。

#### Scenario: Existing Skill is upgraded after approval
- **WHEN** change 获得批准并进入 apply
- **THEN** writer 原地编辑现有 `order-run-loop` 并只新增 `references/self-evolution.md`
- **AND** 不运行初始化器覆盖现有 Skill，不创建第二个 Skill 或其他资源目录

#### Scenario: Skill package is validated
- **WHEN** writer 完成 `SKILL.md`、reference 与 metadata 检查
- **THEN** `skill-creator/scripts/quick_validate.py .agents/skills/order-run-loop` 返回成功
- **AND** structure、metadata、四 skill 引用、薄层无复制、状态机、评分表、session/升级/停止规则、self-evolution 链接和 owned-path 检查全部通过

#### Scenario: Reference depth is inspected
- **WHEN** reviewer 枚举 `.agents/skills/order-run-loop/`
- **THEN** 唯一新增规则文件位于 `references/self-evolution.md`
- **AND** `SKILL.md` 之外没有第二层 reference、checker script、asset 或 README

### Requirement: Every module freezes one recoverable runner base
At the start of each module, `order-run-loop` MUST persist an immutable runner base containing the full repository SHA, the `SKILL.md` git blob SHA, the file SHA256, and either an explicit runner version or an explicit unversioned marker. The same checkpoint MUST contain the module identity, lifecycle state, candidate/integrated/archive SHAs, error fingerprint count, dependency and blocker state so another session can recover from repository evidence, OpenSpec tasks and exact SHAs without relying on chat history.

#### Scenario: A module starts
- **WHEN** the main Agent allocates a module lane before any module action
- **THEN** it records the repository SHA, Skill blob, Skill SHA256 and version marker in the module checkpoint
- **AND** all four values are computed from the checked-out repository rather than copied from an unsupported status claim

#### Scenario: A session resumes after interruption
- **WHEN** a new main session resumes an incomplete module
- **THEN** it reconstructs state from the repository checkpoint, OpenSpec artifacts/tasks, full candidate or integration SHAs and current clean status
- **AND** it does not treat old chat memory as lifecycle or validation evidence

#### Scenario: The runner file changes during a module
- **WHEN** the current module observes a different Skill blob, content SHA or version after its base was frozen
- **THEN** the module keeps executing against the frozen base contract and records the difference as an observation or blocker
- **AND** it does not rewrite the current module base or silently adopt the changed rule

### Requirement: Module execution is observation-only for runner evolution
During a module, the control plane MUST append observations without changing the runner rules used by that module. Every observation MUST have exactly one immutable class: `candidate`, `environment`, `checker`, or `external`; a later candidate derived from another class MUST be a new record that references its source rather than mutating the source class.

#### Scenario: An implementation lesson appears mid-module
- **WHEN** a writer, verifier or integrator discovers a possible workflow improvement while the module is active
- **THEN** the main Agent records the observation and its evidence under exactly one allowed class
- **AND** no current-module Skill, reference, Gate or routing decision changes because of that observation

#### Scenario: A checker defect suggests a general rule
- **WHEN** a `checker` observation is reproducible and may qualify for dedicated DRAFT screening
- **THEN** the ledger retains the original checker record and creates a separate `candidate` record referencing it
- **AND** DRAFT admission and later promotion are evaluated only on the candidate record

#### Scenario: An environment or external observation is recorded
- **WHEN** an event depends on one machine, transient tool state, unavailable credentials, qualifications, authority or another real-world asset
- **THEN** it is recorded as `environment` or `external` with owner and recovery evidence as applicable
- **AND** it is not disguised as a generally applicable runner rule

### Requirement: Module-boundary screening may admit an observation candidate to DRAFT
At a module boundary, a candidate observation MUST be queued for or created as a dedicated runner-change `DRAFT` only when its record contains reproducible evidence, a generalizable or safety-critical rationale, explicit non-weakening intent, an executable regression-check plan and an executable minimal-context fresh-session forward-test plan. This screening MUST NOT be recorded as approval, implementation, candidate SHA, independent verification or rule promotion.

#### Scenario: A candidate passes DRAFT admission screening
- **WHEN** the module retrospective contains all five required evidence or plan fields
- **THEN** the main Agent may queue or create one dedicated OpenSpec `DRAFT` whose source observation remains traceable
- **AND** the record explicitly states that admission is not promotion and changes no active runner rule

#### Scenario: A DRAFT admission field is missing
- **WHEN** reproducibility, applicability/safety, non-weakening intent, regression plan or forward-test plan is absent
- **THEN** the candidate remains in the observation ledger and no dedicated change is opened from it
- **AND** a score or another observation cannot replace the missing field

### Requirement: Implemented rule promotion passes every non-weakening Gate
A rule implemented by a dedicated change MUST be promoted only when reproducibility and generalizability or safety-critical impact are revalidated, non-weakening checks pass, the implemented regression check passes, an independent minimal-context fresh-session forward-test passes against the exact candidate SHA in a clean detached worktree, and the complete independent verification passes for that same SHA. Missing, failed or `BLOCKED_EXTERNAL` evidence MUST reject promotion, and scores MUST NOT compensate for a failed promotion Gate.

#### Scenario: An exact candidate satisfies every promotion Gate
- **WHEN** all revalidated facts, non-weakening checks, implemented regression, clean-detached exact-SHA forward-test and complete independent verification have current PASS evidence
- **THEN** the exact rule candidate may be marked promoted but remains inactive pending local-main integration
- **AND** its source observation, change, exact SHA and verifier attestation remain traceable

#### Scenario: One post-implementation promotion Gate is missing
- **WHEN** any revalidation, non-weakening, regression, exact-SHA forward-test or full independent verification evidence is absent or fails
- **THEN** the rule is not promoted and no runner rule changes
- **AND** C/T/V/R or another successful check cannot override the missing Gate

#### Scenario: A proposal weakens an existing Gate
- **WHEN** a candidate would skip a required test, delete a failure scenario, loosen an assertion, replace tested production behavior with a mock, add a false-green construct or bypass exact-SHA independence
- **THEN** promotion fails as a hard blocker
- **AND** the control plane preserves the existing stronger rule

### Requirement: Promoted rules activate only for a later module
A promoted rule MUST become available only after its dedicated change is integrated into local main, and only to a module whose runner base is captured after that integration. The active module MUST never switch runner versions in place; if the current Goal has no authorized later runner-change lane, DRAFT-admissible observations MUST remain queued for a later Goal.

#### Scenario: A rule change integrates while another module is active
- **WHEN** local main receives an independently verified runner change after the active module froze its base
- **THEN** the active module continues with its frozen runner
- **AND** the integrated rule may be selected only when the next module captures a new base

#### Scenario: A module ends with DRAFT-admissible observations
- **WHEN** its retrospective contains candidates that pass DRAFT admission screening but the current Goal does not authorize another runner change
- **THEN** those candidates are retained in the next-Goal queue
- **AND** the module does not reopen or self-edit `order-run-loop`

### Requirement: Checker regressions are portable and fail truthfully
The self-evolution acceptance suite MUST cover zero-match counting, structured Markdown field parsing, literal backticks across tool layers, explicit `awk` case semantics, bounded health-recovery polling, safe temporary directories and archive output with a trailing newline. The checks MUST use only repository/local standard tooling, MUST return non-zero for a violated contract and MUST NOT use `|| true` or another construct that converts an unexpected failure into PASS.

#### Scenario: Zero matches are counted
- **WHEN** the fixture contains no matching record and zero is an allowed result
- **THEN** the checker returns numeric count `0` without treating the search tool's no-match status as an execution failure
- **AND** a real invocation or parsing error still fails the check

#### Scenario: Markdown fields are parsed
- **WHEN** a checkpoint contains exact field lines, fenced examples, colons or formatting tokens in values
- **THEN** the checker reads the intended structural field and ignores lookalikes in fenced content
- **AND** a missing or duplicate required field fails with the first decisive error

#### Scenario: Backticks cross a tool boundary
- **WHEN** fixture data contains literal backticks or shell metacharacters
- **THEN** the checker passes data as an argument vector or standard input without shell evaluation and receives the literal bytes unchanged
- **AND** command substitution is never used to transport untrusted fixture content

#### Scenario: Case semantics are explicit
- **WHEN** a contract field is case-insensitive and the fixture uses mixed case
- **THEN** the checker normalizes or explicitly performs case-insensitive matching instead of relying on default `awk` case behavior
- **AND** case-sensitive fields remain exact by contract

#### Scenario: Health recovers or times out
- **WHEN** health becomes ready within the configured attempt/time bound
- **THEN** bounded polling returns PASS after the observed ready state
- **AND** a fixture that never becomes ready returns FAIL at the bound without an unbounded wait or fourth blind retry

#### Scenario: A temporary surface is created and removed
- **WHEN** a checker needs a temporary directory
- **THEN** it creates a narrowly named system temporary directory, resolves and validates the exact target, rejects links/broad/empty/out-of-bound targets and enumerates it before cleanup
- **AND** any validation or cleanup failure stops without escalating to a broader deletion primitive

#### Scenario: Archive output ends with a newline
- **WHEN** an archive command emits the expected path or SHA followed by LF or CRLF
- **THEN** the checker removes only the record terminator and compares the exact remaining value successfully
- **AND** meaningful non-newline characters are not silently stripped

### Requirement: Fresh-session forward-test proves the thin control behavior
The verifier MUST run a fresh session against the exact candidate SHA in a clean detached worktree using only root governance, `SKILL.md`, its one-level self-evolution reference and a minimal scenario fixture. The session MUST demonstrate runner-base capture, observation-only behavior, classification, DRAFT admission without promotion, rejected incomplete screening, local-main-gated next-module activation and unchanged routing to the four stage skills; writer self-test or a prior SHA MUST NOT substitute for this evidence.

#### Scenario: A minimal fresh session follows the protocol
- **WHEN** an independent session receives only the declared minimal context and scenario fixture at the candidate SHA
- **THEN** its structured result captures the base, appends observations without modifying the current runner, rejects an incomplete DRAFT screen, labels a complete screen as DRAFT-admissible but not promoted, activates only a supplied local-main-integrated rule for the next module and selects the existing stage handlers correctly
- **AND** the forward-test validator returns PASS without repository changes

#### Scenario: The fresh session edits current rules or drifts routing
- **WHEN** the session changes the active runner, applies a rule to the same module, weakens a Gate or routes a lifecycle state to a different handler
- **THEN** the forward-test fails with that first decisive mismatch
- **AND** the candidate cannot become `INDEPENDENT_VERIFIED`

### Requirement: Evolution preserves metadata, routes and existing hard Gates
The upgraded package MUST keep the Skill name and directory consistent with `agents/openai.yaml`, keep its default prompt explicitly invoking `$order-run-loop`, preserve the existing four handler mappings, lifecycle states, lane limit, exact-SHA invalidation, third-identical-fingerprint stop, `BLOCKED_EXTERNAL`, readiness weights and hard-Gate precedence. Detailed evolution rules MUST extend rather than reproduce or weaken the stage skills.

#### Scenario: Metadata and legacy routing are inspected
- **WHEN** the contract checker reads `SKILL.md`, `agents/openai.yaml` and the one-level reference
- **THEN** name/path/default-prompt metadata agree and all four original handler mappings remain machine-searchable
- **AND** lifecycle, lane, retry, external-block, scoring and stop invariants remain present

#### Scenario: A stage procedure is copied into the control plane
- **WHEN** `SKILL.md` or its reference reproduces the implementation, verification or integration stage procedure instead of routing to it
- **THEN** the thin-layer check fails
- **AND** the duplicated procedure is removed before candidate creation
