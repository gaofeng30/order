# Goal Checkpoint

| field | value |
| --- | --- |
| module | `admit-legacy-miniprogram-gate-receipt` |
| lifecycle | `CANDIDATE` |
| gate_type | `W1` |
| ui_level_target | `UI0` |
| ui_level_actual | `NOT_RUN` |
| base_sha | `07688c7cdcc957bd9fed48f8c408c929a930c23f` |
| candidate_sha | `the full SHA of the commit containing this checkpoint; recorded by Git and Harness immediately after commit` |
| owner | main Agent; branch `codex/admit-legacy-miniprogram-gate-receipt`; worktree `/Users/vivix/.codex/worktrees/order-admit-legacy-miniprogram-gate-receipt.Writer` |
| dependency | exact-B Mini Program binding PASS and local-main integration at `07688c7cdcc957bd9fed48f8c408c929a930c23f` |
| blocker | none; receipt delivery remains post-archive and independent-verifier gated |
| error_fingerprint | `none` |
| repeat_count | `0` |
| next | require two independent clean-detached verifiers on the literal final candidate SHA |

## Frozen runner base

| field | evidence |
| --- | --- |
| repo_sha | `07688c7cdcc957bd9fed48f8c408c929a930c23f` |
| skill_blob | `0f41f64ad87fc9fd410cb916b4d1562aee92e42f` |
| skill_sha256 | `2781bdda1544106e30e7483c4b500d611df85c79753cc1bd3b717d91d1edaac8` |
| runner_version | `unversioned` |

## Self-evolution admission

- Reproducible evidence: exact-B 上为 governance change 添加 schema-valid 标准 receipt 后，受保护 v1 checker 固定失败于 `checkpoint runtime section count must be 1, got 0`。
- Applicability/safety: 仅影响一个已归档、checkpoint 早于当前约定的 exact governance candidate；缺口阻断已授权 closure，但一般化兼容会掩盖真实错误。
- Non-weakening: 不改受保护 judge/runner/schema/profile/binding，不更改历史字节；仅接受固定 change/SHA/hash/blob/path，并保留 v1 `--list`/`--chain` 原行为。
- Regression plan: exact positive、逐字段/逐 blob/path/history 负例、现有 receipt 全套、Mini Program profile、OpenSpec strict、owned/protected/sensitive/clean Gate。
- Forward-test plan: 在 clean detached exact candidate 中先验证无 legacy receipt fail closed；archive 后的临时 exact receipt-head 只在唯一 legacy 文件、精确 ancestry 和机械 profile PASS 时派生 PASS，任何一个字节变化失败。
- Admission status: 用户已授权继续并尝试由 Loop Engineering 自我闭环；四件套与 checkpoint 已由主 Agent 审核并通过 strict，批准进入 TDD，但不等于 promotion、integration、archive 或 receipt PASS。
- Red/Green: direct v1 receipt Red was the exact runtime-section failure; the focused implementation Red was missing adapter code; after minimal implementation the same suite passes `18/18`, while the production candidate still rejects missing legacy receipt.
- Baseline observation: `obs-20260821T130908297461Z-e4a16d` records that the historical full discovery still expects two now-archived active change directories and four profiles. Stable parser/Git/menu suites, archived forward suite and runner contract pass; no historical/protected test is rewritten inside this repair.
- Writer Gate: protected v1 list preserves four historical receipts; chain preserves old FAIL plus supersession PASS; exact Mini Program profile returns `MECHANICAL_PASS` with binding hash `878d…`; disposable implementation-head `ac7a35a… → A=d24eda8… → R=5a6bbce…` runs the production compatibility CLI to `PASS_DERIVED` while reporting original `IMPLEMENTING`/`none`, then old v1 list/chain remain PASS. Those preview SHAs are disposable evidence only, not promotion authority.
