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

### Requirement: Recorded attestations never prove mechanical or independent verification
The receipt MUST store only its controlled profile/binding reference, expected historical verdict, `recorded_attestation_json` labeled `UNTRUSTED_FOR_MECHANICAL_PASS`, and `mechanical_verification=REQUIRED_DERIVED`. It MUST NOT persist a derived mechanical result or derive PASS from receipt text, provenance, Git author/committer, writer identity, or session labels. `INDEPENDENT_VERIFIED` MUST still require a fresh independent `$order-verify-change` run at the exact candidate SHA.

#### Scenario: Writer claims independent PASS
- **WHEN** a writer self-attestation, fake provenance, or commit-author identity claims PASS
- **THEN** mechanical verification remains `UNVERIFIED` unless the controlled exact-SHA profile independently replays successfully
- **AND** no lifecycle state becomes `INDEPENDENT_VERIFIED`

### Requirement: Mechanical verification uses only controlled exact-SHA profiles
The checker MUST derive only `MECHANICAL_PASS`, `EXPECTED_MECHANICAL_FAIL`, or `UNVERIFIED` from `tools/lifecycle-receipts/mechanical-profiles-v1.json`. Each binding MUST fix profile ID/version/hash, change, exact target SHA, tool-source blob, ordered argv/cwd/env/timeout/expected-output steps, `network=false`, `write_scope=temp-only`, and output cap; receipt-supplied commands MUST be ignored and rejected.

#### Scenario: Exact controlled replay passes
- **WHEN** the exact target and every registry/blob/hash/step/environment/expected result match in a clean detached temp worktree
- **THEN** the checker emits the profile's fixed mechanical result
- **AND** identifies the exact profile and target used

#### Scenario: Profile evidence is missing or altered
- **WHEN** target SHA, tool blob/hash, argv, cwd, environment allowlist, expected exit/output, timeout, write scope, or registry binding is missing or mismatched
- **THEN** the checker returns non-zero and `UNVERIFIED`
- **AND** no receipt field or weaker command can compensate

### Requirement: Mechanical execution is bounded and fail closed
The executor MUST invoke argv with `shell=false`, use only allowlisted environment values, run in a validated narrow system-temp detached worktree, route HOME/TMP/cache paths into that root, apply offline tool flags, enforce process-group timeout/output caps, require clean pre/post repository state, and fail safely on cleanup or link ambiguity. `network=false` and `write_scope=temp-only` are controlled profile contracts checked through trusted wrappers and environment constraints, not claims that the Python standard library blocks every raw socket or absolute-path write. A profile requiring stronger unavailable OS isolation MUST return `UNVERIFIED`. Go steps MUST prevalidate an installed matching version and use `GOTOOLCHAIN=local`, `GOPROXY=off`, and `GOSUMDB=off`; automatic toolchain download is forbidden.

#### Scenario: Step times out or escapes its boundary
- **WHEN** a subprocess times out, exceeds output limits, dirties repository state, violates an auditable offline/temp contract, requires unavailable stronger isolation, encounters unsafe links, or cleanup cannot be proven
- **THEN** execution stops at the first decisive error and returns `UNVERIFIED`
- **AND** no later step is reported PASS

### Requirement: Four controlled profiles preserve truthful scope
The registry MUST predeclare `self-evolution-v1`, `old-menu-artifact-fail-v1`, `menu-supersession-v1`, and `lifecycle-receipt-control-v1`. Self-evolution MUST replay only its repository contract and seven-positive/seven-negative checker suite. Old menu MUST structurally reproduce its exact semantic artifact failure with later Gates `NOT_RUN`. Supersession MUST replay its fixed structural, legacy Red, UI1 13/13, provider, static, strict-structure, scope, and product-tree matrix. The bootstrap control profile MUST target this exact candidate after archive and replay the repository-contained parser/Git/registry/executor/profile/runner/forward positive and negative suites, three historical profiles/chain, false-attestation rejection, and non-weakening checks.

#### Scenario: Old menu expected failure is replayed
- **WHEN** exact Git blobs reproduce fingerprint `artifact-consistency|semantic|proposal-stale-draft-not-run|6d77bdd|verifier`
- **THEN** the result is `EXPECTED_MECHANICAL_FAIL` and later Gates remain `NOT_RUN`
- **AND** the historical candidate remains `FAILED`

#### Scenario: Environment-dependent evidence is unavailable
- **WHEN** an excluded or required local toolchain is unavailable or incompatible
- **THEN** the affected claim is excluded as declared or becomes `UNVERIFIED`
- **AND** recorded provenance is not used as fallback

#### Scenario: Bootstrap control profile is bound after archive
- **WHEN** `lifecycle-receipt-control-v1` is exact-target-bound to this independently verified candidate after integration/archive
- **THEN** a different clean-detached binding-head verifier reruns its fixed repository-contained suites and three-history chain
- **AND** user-directory quick validation, mutable OpenSpec CLI behavior, Git author/session identity, network, UI2/UI3, and real order/payment remain excluded

### Requirement: Go replay uses a validated local module proxy cache
Go profile steps MUST require an already-installed exact Go version and a safe local `$GOMODCACHE/cache/download`. A trusted wrapper MUST resolve the source cache against unsafe links; construct an explicit temp-only environment with `GOENV=off` and no inherited proxy, private-module, VCS or credential variables; let exact Go obtain the MVS/pruned build list and actual two-target package closure through a `file://` GOPROXY with `GOTOOLCHAIN=local` and `GOSUMDB=off`; and validate the package closure's required `.mod`, `.zip`, and `.ziphash`. It MUST NOT invent metadata, require unrelated cached `.info`, recursively interpret module requirements, or explicitly download non-package-closure zips. It then MUST set `GOPROXY=off`, run `go mod verify`, and run the fixed package tests. It MUST NOT use a network proxy or automatic toolchain download.

#### Scenario: Local Go and cache are complete
- **WHEN** exact local Go, safe cached module artifacts, temporary population, and `go mod verify` all pass
- **THEN** the fixed provider package tests may contribute to the derived mechanical result
- **AND** all Go writes and caches are routed to the validated temporary root

#### Scenario: Local Go cache prerequisite fails
- **WHEN** Go version, cache archive, path/link safety, module checksum, or offline population is missing or mismatched
- **THEN** the profile returns `UNVERIFIED`
- **AND** it does not contact the network, weaken verification, or use recorded attestation as fallback

### Requirement: Archived recovery uses one append-only receipt
After existing candidate verification, integration, and archive Gates, the control plane MUST add exactly one `lifecycle-receipt.md` in the dated archive, keep candidate checkpoint/tasks byte-identical, and store `receipt_head_verification=REQUIRED_DERIVED`. Recovery MUST verify exact Git objects/ancestry, archive path/diff, ownership, candidate artifact digests/tasks, retrospective, unique receipt add, and no later receipt touch.

#### Scenario: Receipt history is valid
- **WHEN** structural/Git checks and the required controlled mechanical replay all pass
- **THEN** recovery reports the immutable receipt head and layered evidence result
- **AND** writes neither the derived head nor PASS back to repository evidence

#### Scenario: Receipt is stale or tampered
- **WHEN** the receipt is missing, duplicated, edited later, ancestry-inconsistent, task-inconsistent, or structurally ambiguous
- **THEN** recovery returns non-zero `NO-GO`
- **AND** archive presence alone does not close recovery

### Requirement: Supersession recovery never launders the old failure
The old-menu receipt MUST store expected verdict `FAILED`, later Gates `NOT_RUN`, and `mechanical_verification=REQUIRED_DERIVED`; the checker MUST re-derive `EXPECTED_MECHANICAL_FAIL` on every recovery. A reciprocal replacement receipt may support only `mechanically_reproducible=true` for the current delivery when its own re-derived mechanical result, integration/archive facts, receipts, links, and exact app/catalog/httpapi tree identities all agree.

#### Scenario: Complete replacement chain is recovered
- **WHEN** old FAIL, reciprocal links, replacement `MECHANICAL_PASS`, receipt histories, exact stage facts, and product trees all match
- **THEN** current delivery is reported mechanically reproducible
- **AND** neither old candidate nor recorded attestation is reported PASS or `INDEPENDENT_VERIFIED`

#### Scenario: Old candidate is upgraded
- **WHEN** any field or chain attempts to assign PASS to the old candidate
- **THEN** validation returns non-zero
- **AND** product-tree equality, score, provenance, or archive presence cannot override the FAIL

### Requirement: Profile admission and bootstrap do not weaken lifecycle Gates
For every future business module, a separate preceding control-plane change MUST define its profile and trusted wrapper, receive independent exact-SHA verification, integrate/archive into local main, and be included in the business module's frozen runner base. The business module MUST NOT define or modify its own judge. After the business candidate independently verifies and pure-FF integrates/archives, only the integrator MAY append the exact-target binding; a different clean-detached verifier MUST pass that binding-head before a later receipt commit, and another clean-detached verifier MUST derive the receipt-head before the next module starts. Binding and receipt commits MUST NOT edit the candidate. This change alone MAY bootstrap with its predeclared `lifecycle-receipt-control-v1`, but MUST still pass candidate, binding-head, and receipt-head verifiers. Receipts and mechanical replay MUST NOT replace any existing Gate.

#### Scenario: Business module tries to define its own judge
- **WHEN** a business module defines or edits its own profile/wrapper, or freezes a base before the preceding control-plane change is independently verified and archived in local main
- **THEN** planning or candidate validation returns `NO-GO`
- **AND** that profile cannot judge the business module

#### Scenario: Future candidate lacks a closed evidence chain
- **WHEN** a future module lacks independent candidate PASS, pure-FF integration/archive, integrator-only exact binding, independent binding-head PASS, receipt commit, or independent receipt-head derivation
- **THEN** receipt closure returns `NO-GO`
- **AND** the runner does not allocate the next module from that archive

#### Scenario: Bootstrap change closes its own receipt
- **WHEN** this change first passes a different clean-detached exact-SHA verifier, then integrates/archives, binds the fixed integrated tool blob, and a different verifier passes the exact binding-head
- **THEN** its later receipt may be added with `REQUIRED_DERIVED` and undergo a separate clean-detached receipt-head check
- **AND** the exception cannot be reused or cited as actor-independence proof

### Requirement: Fresh-session recovery is repository bound
The current candidate verifier MUST use a clean detached exact-SHA worktree and minimal repository context, rerun the controlled receipt/profile/chain checks, preserve the old FAIL and stage routes, reject invalid fixtures, and leave the repository clean.

#### Scenario: Fake forward result is presented
- **WHEN** a result names a nonexistent or wrong candidate, attached/dirty repository, fake receipt output, or writer-authored PASS
- **THEN** validation returns non-zero
- **AND** only repo-local checker results at the exact detached SHA are accepted

### Requirement: Bootstrap binding admission is stage-aware and exact
The lifecycle receipt control plane MUST accept exactly the three immutable historical bindings while the repair candidate is unbound. Candidate `C` MUST contain `checks/verify_archive.py` and the complete target bytes in `checks/expected-canonical-loop-engineering-control-plane-spec.md`, and both exact-`C` verifier recipes MUST exercise the archive-checker fixtures. Every archive-Gate judgment MUST execute the checker blob from exact `C`, and that checker MUST execute the runner-validator blob from exact `C`; the corresponding `A` files MUST be compared subjects only. It MUST accept one additional `lifecycle-receipt-control-v1` binding only when uniquely derived repair archive `A` has exactly one parent equal to `C`; `C→A` consists solely of a complete byte-identical `R100` move of every active change blob to one dated archive plus one canonical spec `M`; the canonical blob at `A` equals the expected fixture blob at `C` byte-for-byte; protected registry/wrapper/executor/judge blobs are identical at `C` and `A`; and a unique later binding-only commit `B` descends from `A`. The additional binding MUST target `d0b70a077bcaa64c401837eb0e9b6f27035210a0`; match the fixed controlled profile definition, hash, argv, environment, expected results, wrapper and repaired executor bytes at both `C` and `A`; and leave those controlled bytes identical at the current head. The loader MUST derive only Git/byte/mechanical results, MUST report actor independence as `NOT_PROVEN_BY_MECHANICAL_REPLAY`, and MUST NOT consume thread pointers or create a lifecycle state.

#### Scenario: Candidate remains valid and unbound
- **WHEN** the repair candidate contains exactly the three historical bindings and no bootstrap binding
- **THEN** registry loading and the existing historical profiles remain valid
- **AND** the candidate receives verification only from an ordinary fresh clean-detached exact-SHA verifier

#### Scenario: Exact post-archive bootstrap binding is admitted
- **WHEN** the dispatcher loads checker bytes from literal full `C`, that checker loads runner-validator bytes from the same `C`, those trusted bytes prove `HEAD=A` and clean, the sole parent and complete same-relative `R100` dated move, the sole canonical `M` blob equals the expected fixture at `C`, protected blobs `A=C`, and one later binding-only `B` appends the fixed `lifecycle-receipt-control-v1` target with every definition/source/blob/hash/execution field matching `C` and `A`
- **THEN** the loader accepts exactly four bindings and the binding-head verifier may execute the fixed bootstrap profile
- **AND** `--all-historical` still selects only the three historical profiles

#### Scenario: Archive Gate is exact and read-only
- **WHEN** a fixed direct-argv bootstrap at exact clean `HEAD=A` loads `checks/verify_archive.py` from the literal full `C` Git blob, passes `--repo . --candidate <literal-full-C-SHA> --archive <literal-full-A-SHA>`, and that checker loads its runner validator from the same `C`
- **THEN** success exits `0` with stdout exactly `archive-gate=PASS` plus one LF and no mutation
- **AND** any non-full SHA, wrong HEAD, dirty status, wrong parent/rename/path, canonical extra/missing/reordered byte, protected registry/wrapper/executor/judge drift, or other ambiguity exits `1` at the first decisive error

#### Scenario: Archive carries an unverified change
- **WHEN** `A` supplies the checker or runner used as judgment authority, `A` is a merge, its sole parent is not exact `C`, an active change blob is missing or not moved with same-relative `R100` byte identity, the canonical blob differs by any extra/missing/reordered byte from the expected fixture at `C`, or any other path including registry, wrapper, executor, judge, Skill, product, or test source changes in `C→A`
- **THEN** loading fails closed before the bootstrap profile executes
- **AND** archive presence, ancestry alone, author identity, or a later matching working-tree file cannot compensate

#### Scenario: Binding is premature or malformed
- **WHEN** bootstrap is bound before repair archive, at the same archive stage, more than once, with an extra binding, or with a wrong target, definition, hash, path, blob, argv, environment, expected result, timeout, network/write contract, or output cap
- **THEN** loading fails closed before profile execution
- **AND** receipt text, provenance, author identity, or a receipt-supplied command cannot compensate

#### Scenario: Binding history or repaired source drifts
- **WHEN** the archive/binding commits are missing, ambiguous or ancestry-inconsistent, the binding commit changes another path, the binding is later edited, or current controlled sources differ from exact `C`/`A`
- **THEN** loading returns `UNVERIFIED` with the first decisive error
- **AND** no historical receipt is rewritten or upgraded

### Requirement: Repair promotion and Goal0 closure are separately gated
Before APPROVED/implementation, the main Gate MUST read a current independent thread final proving exact `d0b70a077bcaa64c401837eb0e9b6f27035210a0` PASS because its archived checkpoint/tasks explicitly do not. Before integration/archive, dispatcher-supplied literal full repair candidate `C` MUST pass both a no-secret minimal-context forward-test and ordinary exact-SHA independent verification in separate verifier-created clean detached worktrees; both MUST run the archive-checker fixtures against the exact expected canonical bytes in `C`; and the main Gate MUST read both current exact-`C` finals. A repository audit pointer or `wait_cursor` MUST NOT itself prove PASS. The repair MAY reach `ARCHIVED` at trusted `A`, but Goal0 MUST remain `NO-GO` until an authorized integrator appends exact binding-head `B`, a different verifier returns a current exact-`B` final after fixed profile `MECHANICAL_PASS`, the integrator adds later receipt-head `R`, and another verifier returns a current exact-`R` final after deriving `PASS_DERIVED` without persisting it.

#### Scenario: Historical dependency handoff is pending or unavailable
- **WHEN** exact `d0b70a...` handoff is active/PENDING, missing, unreadable, stale, non-final, wrong-SHA, or not PASS even though repository archive/checkpoint/receipt text exists
- **THEN** the main Gate returns `NO-GO` for APPROVED/implementation
- **AND** waits for the current verifier or dispatches a fresh exact-SHA verifier instead of inferring PASS from repository text

#### Scenario: Current exact handoff is readable
- **WHEN** the main Gate calls `read_thread` with recorded `threadId` and `hostId` and the current final contains the expected full SHA, commands, result, and limitations
- **THEN** the independent Gate may consume that current handoff for actor/session separation
- **AND** an optional stored `wait_cursor` is only `wait_threads.afterCursor`; it may be unavailable across callers without affecting final readability, MUST NOT be reused as `read_thread`'s separate pagination cursor, and never becomes loader input or mechanical proof

#### Scenario: Minimal-context forward-test passes independently
- **WHEN** the dispatcher supplies repository, change name, and literal full `C`; the verifier creates a clean detached exact-SHA worktree under the verification contract; and the fixed recipe proves exact/detached/clean pre/post state while running the loader and archive-checker local temporary-Git positive plus A-controlled checker/runner, canonical-byte/judge/path/parent/rename/HEAD negatives without network or secrets
- **THEN** the minimal-context Gate records PASS for exact `C`
- **AND** that PASS does not replace the separate ordinary independent-verification record

#### Scenario: Repair is archived before Goal0 closure
- **WHEN** exact `C` is independently verified and trusted `A` is integrated/archived but `B`, independent binding-head PASS, later `R`, or independent receipt-head derivation is missing, stale, dirty, non-detached, wrong-SHA, out of order, or performed by a forbidden role
- **THEN** this repair change may be `ARCHIVED` while Goal0 remains `NO-GO`
- **AND** the runner MUST NOT treat repair archive state as Goal0 completion

#### Scenario: Goal0 closure completes in order
- **WHEN** the integrator-only exact `B`, different verifier's current exact-`B` handoff, later receipt-only `R`, and another verifier's current exact-`R` handoff all match their dispatcher-supplied literal SHAs, declared commands, mechanical outputs, and clean-state results
- **THEN** Goal0 receipt closure may be reported complete
- **AND** mechanical replay still does not prove actor/session independence

#### Scenario: Repair archive precedes external receipt closure
- **WHEN** this repair itself has ordinary candidate PASS and is integrated and archived while the original control receipt remains unbound
- **THEN** the repair may reach `ARCHIVED` without using the bootstrap profile to attest itself
- **AND** the later exact `B`, independent binding-head PASS, receipt-only `R`, and separate receipt-head derivation remain required before Goal0 is closed
