# Goal Checkpoint

| field | value |
| --- | --- |
| change | fix-miniprogram-profile-wxml-structure |
| state | CANDIDATE |
| base_sha | 501e22ef22a87de0e2467de3dcb1a0e5b728f075 |
| candidate_sha | derive-from-git-external-handoff |
| integrated_sha | none |
| owner | current main Agent |
| branch | codex/fix-miniprogram-profile-wxml-structure |
| worktree | /Users/vivix/.codex/worktrees/order-fix-miniprogram-profile-wxml-structure.Writer |
| gate_type | W2 |
| ui_level_target | UI2 |
| ui_level_actual | UI2 |
| runner_repo_sha | 501e22ef22a87de0e2467de3dcb1a0e5b728f075 |
| runner_skill_blob | 0f41f64ad87fc9fd410cb916b4d1562aee92e42f |
| runner_skill_sha256 | 2781bdda1544106e30e7483c4b500d611df85c79753cc1bd3b717d91d1edaac8 |
| runner_version | UNVERSIONED |
| error_fingerprint | ARTIFACT_CONSISTENCY:stale-candidate-ui2-metadata:b0247b68 |
| repeat_count | 1 |
| dependency | none; integration unblocks connect-miniprogram-session-client |
| blocker | none; DevTools and assigned test account available |
| next | Commit the metadata-only replacement candidate, bind a fresh exact-SHA UI2 receipt, then require two independent verifier PASS results before local integration. |

## Readiness

- First blocking OpenSpec strict: PASS.
- OPEN P0/P1: one P1 Mini Program compile blocker owned by this change.
- External Gate: the assigned official test account and DevTools simulator produced UI2; formal AppID, real login, profile-data authorization, payment, upload, review and physical device are outside this change and remain NOT_RUN.
- Writer score: `C=9 T=10 V=8 R=9`, total `36`, hard blockers `0`; verifier independence remains pending.
- Writer Gate: focused `2/2`, full Mini Program `66/66`, direct lint, JS/JSON, plan/change/all strict, diff, owned and sensitive checks PASS on final pre-commit bytes.
- Invalidated candidate `b0247b68f89b0d28c36ddff61133c4fcd163a8bd`: product, static and UI2 runtime Gates passed, but verifier A correctly returned `FAIL` because this checkpoint and task 5.2 still declared UI2 pending. No PASS from that SHA is reusable.
- Current verdict: CANDIDATE by external-post-commit convention; product bytes are unchanged from the invalidated runtime-tested SHA, but the replacement SHA still requires its own exact-SHA receipt and two fresh independent verifications.

## Runner Observations

- `checker`: the existing Node wrapper and `lint_wx.py` returned PASS while WeChat DevTools rejected a mismatched close tag; this change repairs the product compilation checker without modifying the frozen loop runner.
- `candidate`: verifier A exposed stale candidate metadata after all functional/UI2 checks passed; fail-closed rejection returned the writer to `IMPLEMENTING`, and this replacement updates only owned OpenSpec evidence before a full exact-SHA rerun.
