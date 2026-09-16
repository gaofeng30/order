# Goal Checkpoint

## Current Runtime

| field | value |
| --- | --- |
| module | `connect-miniprogram-reservation-menu` |
| state | `CANDIDATE` |
| owner | `main Agent` |
| branch | `codex/connect-miniprogram-reservation-menu` |
| worktree | `/Users/vivix/.codex/worktrees/order-connect-miniprogram-reservation-menu.Writer` |
| repository_base | `0dea0acc0ed8189b7bd7bf06cf3d32b5b8890d93` |
| candidate_sha | `DERIVE_FROM_GIT_EXTERNAL_HANDOFF` |
| integrated_sha | `none` |
| archive_sha | `none` |
| gate_type | `W2` |
| ui_level_target | `UI2` |
| ui_level_actual | `UI2` |
| writer_gate | `PASS_REPLACEMENT_UI2_REPLAY_REQUIRED` |
| independent_verifier_a | `NOT_RUN` |
| independent_verifier_b | `NOT_RUN` |
| integration | `NOT_RUN` |
| archive | `NOT_RUN` |
| blocker | `none; Node, read-only backend contract tests, DevTools, assigned test account and controlled loopback service are available` |
| next | `commit this initial UI2 evidence as a metadata-only replacement candidate, then rerun candidate Gate and both UI2 scenarios from scratch with a fresh external receipt before independent verification` |

## Frozen Runner Base

| field | evidence |
| --- | --- |
| repo_sha | `0dea0acc0ed8189b7bd7bf06cf3d32b5b8890d93` |
| skill_path | `.agents/skills/order-plan-change/SKILL.md` |
| quality_contract | `docs/quality/change-quality-gates.md` |

## Boundary

- One outcome: the Mini Program menu reads the selected date/time from the integrated reservation-menu endpoint and surfaces server sold-out/orderable facts without fallback.
- Owned paths are this OpenSpec, two new reservation modules, menu JS/WXML/WXSS, one new focused test and the existing catalog UI1 test whose obsolete menu-list assertions are superseded.
- Login `app.js/sessionApi/page-harness/package`, backend, catalog detail/store, data/pickup, cart/confirm/order/payment and all Admin paths remain read-only.
- Local Gate is W2 with focused UI1, complete Mini Program/static and read-only backend contract regression; promotion requires exact-SHA UI2 success/error scenarios and two independent verifier PASS results.
- No external product decision is open. No push, PR, deployment, upload, permission grant or review submission is authorized.

## Planning Evidence

- Mini Program plan Gate PASS, change strict PASS, all strict `26/26`, artifact status complete and dependency commit `2b83e93cc2a8d2bb16b606068028f34ee662b677` is an ancestor of exact base.
- Main Agent approved the single selected anonymous reservation-menu design after confirming backend fields, sold-out/cutoff semantics, login path separation, UI2 assets and non-goals require no further product decision.

## Implementation Evidence

- Valid Red observed the existing menu request `/api/v1/catalog` instead of the selected-date/time reservation endpoint before any runtime edit.
- The identical focused command passed Green and Refactor `9/9`; existing Mini Program regression passed `62/62`; all JS, JSON `27`, direct WX lint and read-only backend provider contracts passed.
- Runtime additions are confined to the declared anonymous reservation transport/store and menu page. Login, detail, cart, picker source, checkout, Admin and backend bytes remain unchanged.
- Writer score is `C9/T10/V8/R9 = 36`, hard blockers `0`; this permits CANDIDATE only and does not replace exact-SHA UI2 or independent verification.
- Initial exact candidate `7f79fcc3cbae0f603edc6a51768994e746551ed7` passed both UI2 scenarios in WeChat DevTools: service-order and sold-out actions were correct, one 503 did not auto-retry, and the visible retry produced the sole follow-up 200 and restored ready. Its external receipt SHA256 is `9860401b407a7721f15825d910f6069b1cbad49212bb0daf48d86438bc5786a8`; committing this evidence invalidates that SHA and receipt, so no independent verifier may rely on them.
