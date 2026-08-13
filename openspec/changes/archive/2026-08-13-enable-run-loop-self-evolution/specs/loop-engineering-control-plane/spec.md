## ADDED Requirements

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

## MODIFIED Requirements

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
