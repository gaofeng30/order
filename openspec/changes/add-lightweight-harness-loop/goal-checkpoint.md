# Goal checkpoint

## Frozen runner base

| field | value |
| --- | --- |
| goal | 用一个轻量仓库入口闭环当前状态、检查、交接和受控自进化观察 |
| module | `add-lightweight-harness-loop` |
| lifecycle | `IMPLEMENTING` |
| repo_sha | `d817aeb3ac5de29d2695ed17ed6277b737ba3ee8` |
| skill_blob | `0f41f64ad87fc9fd410cb916b4d1562aee92e42f` |
| skill_sha256 | `2781bdda1544106e30e7483c4b500d611df85c79753cc1bd3b717d91d1edaac8` |
| runner_version | `unversioned` |
| owner | branch `codex/add-lightweight-harness-loop`, worktree `/Users/vivix/.codex/worktrees/order-harness-loop.Writer` |
| dependency | local main integrated control-plane, quality-gate and receipt-control baselines |
| blocker | none |
| candidate_sha | `2ff87fa185c13abff011c8c05b912f989b5b5b24` invalidated by cross-worktree stale-state failure; `f6402ceb276e5a6fcd79987d61017fea3e55360f` also invalidated |
| integrated_sha | none |
| archive_sha | none |
| error_fingerprint | `harness-check|2|writer-change-stale-from-main|2ff87fa|local-writer` |
| repeat_count | `1` |
| next | preserve and validate owning worktree metadata, then rerun all Gates for a new candidate |

## Boundary

- `gate_type=W1`; `ui_level_target=UI0`; `ui_level_actual=NOT_RUN`.
- External assets: none.
- Owned paths and non-goals are the exact lists in `proposal.md`.
- Current module uses the frozen runner above. Any observation during implementation is observation-only and cannot alter this module's rules.
- Approval evidence: the user explicitly authorized the minimal faster, standardized, self-evolving closed loop after reviewing the recommended boundary; strict planning PASS remained current on 2026-08-17.

## Invalidated writer verdict

- Candidate `f6402ceb276e5a6fcd79987d61017fea3e55360f` is `FAIL/INVALIDATED`; the current receipt checker correctly rejected direct protected-runner drift.
- Minimal repair is to restore root governance and `order-run-loop` to base bytes and keep the tool explicit. Updating the receipt judge or activating the runner is outside this change.
- Both system-Python-3.9 and protected-runner-drift observations remain `promotion=NOT_RUN`; neither can alter this frozen runner contract.

## Current writer verdict

- Candidate `2ff87fa185c13abff011c8c05b912f989b5b5b24` is invalidated: main checkout could not resolve the active change stored by its writer worktree and falsely marked it stale.
- Protected governance repair remains valid, but writer/fresh-session/verification evidence must be rerun after the focused source-worktree fix.

## Observations

None.
