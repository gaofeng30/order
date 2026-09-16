## Why

已归档的 `enforce-miniprogram-user-regression-gate` 已把小程序 TDD 与用户回归 Gate 接入主线，但仓库控制的 lifecycle profile registry 没有该 change 的固定机械重放定义，因此无法按当前 run-loop 规则追加 exact-target binding 与可信 receipt。这个缺口必须在分配下一个小程序业务模块前由独立控制面 change 关闭，不能让业务 change 编写或降低自己的 judge。

## What Changes

- 为 exact governance candidate `f5719e98690d0b1301ed567c8b616e074846b445` 增加一个固定、零网络、只写系统临时目录的机械 profile，重放其 Mini Program Gate、Harness、前端与后端非弱化回归。
- 在受控 profile registry 增加这一项未绑定 definition，并最小扩展现有 runner 的固定 profile/binding 阶段：当前四个 binding 保持有效；只有 profile change 自身已精确归档后才允许第五个 governance binding-only append。
- 由 candidate 固定 governance 历史 archive 与本 profile change 未来 archive 的完整预期 canonical bytes/checker；loader 只执行 exact profile candidate 中的 checker/runner authority，归档工作树里的副本只作比较对象。
- 增加 wrapper、registry、archive trust 与 binding history 的聚焦正反例，证明错误 target、缺文件、Gate 弱化、产品树变化、归档/绑定顺序错误、非预期写入或输出都会 fail closed。
- 该 change 独立验证、集成并归档后，才允许另一个已授权 integrator 追加 governance target binding；随后分别完成 binding-head 与 receipt-head 的 clean-detached 验证，业务模块仍不得把机械重放表述为 actor independence。

## Capabilities

### New Capabilities

- `miniprogram-gate-lifecycle-profile`: 定义双 Gate 治理 candidate 的固定机械重放范围、profile registry 契约、失败语义及后续 binding/receipt 边界。

### Modified Capabilities

无。

## Impact

- Primary outcome：仓库拥有一个在业务模块之前独立产生、可由 lifecycle receipt runner 固定执行的 Mini Program 双 Gate profile；本 change 自身不伪造或提前绑定目标。
- 唯一接受裁决：仅当 profile wrapper 的 target/source/命令/输出/临时写边界固定，正反例、完整控制工具回归、Mini Program UI1、Go 回归、registry non-weakening、owned-path 与 clean-detached exact-SHA 验证全部 PASS 时为 `ACCEPT`；任一失败为 `REJECT`。
- Gate：`gate_type=W1`；`ui_level_target=UI0`；DRAFT 时 `ui_level_actual=NOT_RUN`。本 change 无产品 UI；UI1+ 为 `N/A`，不是 PASS。
- Owner：当前主 Agent；branch `codex/add-miniprogram-gate-receipt-profile`；worktree `/Users/vivix/.codex/worktrees/order-add-miniprogram-gate-receipt-profile.Writer`。
- `base_sha`: `b53cb520c4cac0505803a9d1e6dcc1807ac34540`。
- Owned paths：
  - `openspec/changes/add-miniprogram-gate-receipt-profile/**`
  - `tools/lifecycle-receipts/mechanical-profiles-v1.json`
  - `tools/lifecycle-receipts/profiles/miniprogram_user_regression_gate.py`
  - `tools/lifecycle-receipts/profile_runner.py`
  - `tools/lifecycle-receipts/tests/test_miniprogram_user_regression_profile.py`
- Read-only shared contracts：根 `AGENTS.md`、`order-run-loop` Skill 与 self-evolution reference、`tools/miniprogram_gate.py`、`tools/harness`、bindings/schema/judge、其他 profiles/tests/receipts、归档 governance change、全部产品代码、PRD 与 canonical specs。
- Dependencies：当前 main 已包含并归档 `enforce-miniprogram-user-regression-gate`；其 exact candidate 为 `f5719e98690d0b1301ed567c8b616e074846b445`，archive head 为 `b53cb520c4cac0505803a9d1e6dcc1807ac34540`。现有 lifecycle profile runner、schema 与 bootstrap control 已在 base 中生效。
- Required external assets：none。只使用仓库锁定的 Python、Git、Node/npm 与离线 Go toolchain/cache；网络、微信账号、真机、DevTools、secret 和外部系统均不需要。
- Runner freeze：`repo_sha=b53cb520c4cac0505803a9d1e6dcc1807ac34540`；`skill_blob=0f41f64ad87fc9fd410cb916b4d1562aee92e42f`；`skill_sha256=2781bdda1544106e30e7483c4b500d611df85c79753cc1bd3b717d91d1edaac8`；`runner_version=UNVERSIONED`。
- Candidate-owned archive Gate outputs：`openspec/changes/add-miniprogram-gate-receipt-profile/checks/**`，包含 governance `f5719e9… → b53cb52…` 历史归档和本 change `C → A` 未来归档所需的只读 checker 与完整 canonical 预期 bytes；它们必须在 exact-C verifiers 中执行，不能由 archive 阶段生成。
- Authorized archive-only derived path：本 change 的 dated archive move 与新增 `openspec/specs/miniprogram-gate-lifecycle-profile/spec.md`；archive Gate 必须证明 `A` 的唯一 parent 是 exact candidate `C`、除完整 `R100` move 与该 canonical spec `A` 外无其他变化。
- Post-archive delivery-only paths：`tools/lifecycle-receipts/mechanical-bindings-v1.json` 的唯一第五项 append，以及 `openspec/changes/archive/2026-08-21-enforce-miniprogram-user-regression-gate/lifecycle-receipt.md`；两者不是 candidate writer 路径，必须由 integrator 分阶段提交并分别独立验证。
- Non-goals：不修改 Mini Program、Admin、Go 产品行为、Gate/Harness/judge/schema、历史 receipt、PRD 或现有 canonical specs；candidate 不追加 governance binding/receipt，不声称 actor independence，不分配预约菜单业务 writer，不推送、创建 PR、部署或写外部系统。
- 本 change 完成后仍需单独授权并顺序完成 profile change 的集成/归档、governance binding-only commit、不同 clean-detached binding-head PASS、receipt-only commit 与另一 clean-detached receipt-head derivation；全部关闭前，下一个业务模块保持 `NO-GO`。
