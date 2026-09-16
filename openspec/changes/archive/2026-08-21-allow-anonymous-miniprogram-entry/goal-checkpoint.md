# Goal Checkpoint

## Current Runtime

| field | value |
| --- | --- |
| module | `allow-anonymous-miniprogram-entry` |
| state | `CANDIDATE` |
| owner | `main Agent` |
| branch | `codex/allow-anonymous-miniprogram-entry` |
| worktree | `/Users/vivix/.codex/worktrees/order-allow-anonymous-miniprogram-entry.Writer` |
| repository_base | `2d1400bd04f224c6c4da9eaca52e272514dff104` |
| candidate_sha | `DERIVE_FROM_GIT_EXTERNAL_HANDOFF; 938ad13fa09b21576d5c85c0cababd208865575a UI2 receipt invalidated by this metadata commit; 7409ceadca4269b976e91e442117e01bbb15dc4b invalidated by candidate Gate` |
| integrated_sha | `none` |
| archive_sha | `none` |
| gate_type | `W2` |
| ui_level_target | `UI2` |
| ui_level_actual | `UI2` |
| writer_gate | `PASS_UI2_REPLAY_REQUIRED_ON_REPLACEMENT` |
| independent_verifier_a | `NOT_RUN` |
| independent_verifier_b | `NOT_RUN` |
| integration | `NOT_RUN` |
| archive | `NOT_RUN` |
| blocker | `none; required local Node and DevTools/test-account UI2 assets are available` |
| next | `externally bind this metadata commit as the replacement exact candidate, rerun candidate Gate and both UI2 scenarios with a fresh receipt, then start verifier A and verifier B without further repository changes` |

## Frozen Runner Base

| field | evidence |
| --- | --- |
| repo_sha | `2d1400bd04f224c6c4da9eaca52e272514dff104` |
| skill_path | `.agents/skills/order-plan-change/SKILL.md` |
| quality_contract | `docs/quality/change-quality-gates.md` |

## Boundary

- One outcome: clicking the launch-page user card directly enters the existing home page without any interactive profile or phone authorization step.
- Owned paths are this OpenSpec directory, three launch page runtime files and one new focused test. Login `app.js/sessionApi/page-harness/package` paths and every Admin/backend/shared contract remain read-only.
- Local Gate is W2 with focused UI1 plus full Mini Program regression and static checks; promotion requires an exact-SHA UI2 receipt covering both manifest scenarios and two independent verifier PASS results.
- No external product decision is open. No push, PR, deployment, upload, real permission grant or review submission is authorized.

## Writer Evidence

- Red exit 1 because the user card still bound `openAuth`; identical Green and Refactor command passed `3/3` after the minimal launch edit.
- Full Mini Program `66/66`, all JS syntax, JSON `27`, direct WXML/WXSS lint, Mini Program plan, change/all strict `25/25`, owned `11/11`, login protected bytes, forbidden runtime and whitespace Gates PASS.
- Candidate `7409ceadca4269b976e91e442117e01bbb15dc4b` failed closed because a native-capability literal existed only in negative-test and non-goal prose. The owned scanner inputs now preserve the same prohibition without the detector literal; runtime behavior and manifest remain unchanged, and the replacement commit must receive a fresh candidate Gate before UI2.
- Candidate `938ad13fa09b21576d5c85c0cababd208865575a` passed candidate Gate and both UI2 scenarios in a 793-path exact archive with only the declared test project configuration injection. Clicking 用户端 reached `pages/home/home` without an authorization layer; closing and reopening returned to a complete launch page; debugger finished at zero errors and warnings. Its external receipt is deliberately stale after this metadata commit, so the derived replacement SHA must replay both scenarios and validate a fresh receipt before verification.
- C9/T10/V8/R9 = 36 with no local hard blocker. UI2 remains pending until an immutable candidate exists; the score does not promote the change.

## Planning Evidence

- Mini Program plan Gate PASS, strict OpenSpec PASS and artifact status complete on exact base `2d1400bd04f224c6c4da9eaca52e272514dff104`.
- Main Agent approved the single selected design after confirming the PRD already fixes the authorization timing and no behavior-changing question remains.
