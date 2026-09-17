## Why

微信开发者工具 Stable 2.02.2608040 首次编译当前 main 时拒绝 `pages/profile/profile.wxml`：关闭标签与当前打开标签不匹配。现有 `lint_wx.py` 只检查条件分支关系并盲目弹栈，导致自动测试错误放行了一个会阻断整个小程序编译的问题。

## What Changes

- 删除个人中心模板中一个多余的 `</view>`，恢复合法嵌套，不改变文案、事件、样式或业务状态。
- 扩展现有 WXML 结构检查，使其拒绝多余、错序和未闭合标签，同时保留现有 `wx:if/wx:elif/wx:else` 检查。
- 增加一个受控错序标签夹具，证明结构检查不会再次假绿。

## Capabilities

### New Capabilities

- `miniprogram-wxml-compilation`: 小程序已登记 WXML 必须通过静态结构检查并可在微信开发者工具中实际编译，个人中心主路径保持原有用户行为。

### Modified Capabilities

- 无。

## Impact

- `gate_type`: `W2`；`ui_level_target`: `UI2`；当前 `ui_level_actual`: `UI2`。该结果必须由候选提交后生成的外部 exact-SHA receipt 绑定；正式 AppID、真实登录、支付、上传、提审和真机仍为 `NOT_RUN`。
- Owner：当前主 Agent；branch `codex/fix-miniprogram-profile-wxml-structure`；worktree `/Users/vivix/.codex/worktrees/order-fix-miniprogram-profile-wxml-structure.Writer`。
- `base_sha`: `501e22ef22a87de0e2467de3dcb1a0e5b728f075`。
- Frozen runner：repository SHA 同 `base_sha`；`order-run-loop` blob `0f41f64ad87fc9fd410cb916b4d1562aee92e42f`；SHA256 `2781bdda1544106e30e7483c4b500d611df85c79753cc1bd3b717d91d1edaac8`；version `UNVERSIONED`。
- Owned paths：本 change 目录、`apps/wechat-miniprogram/pages/profile/profile.wxml`、`apps/wechat-miniprogram/tests/lint_wx.py`、`apps/wechat-miniprogram/tests/wx-compile-ui1.test.js`。
- Read-only shared contracts：其余小程序页面/组件/脚本/样式、`apps/wechat-miniprogram/package.json`、根 `project.config.json`、Go/Admin、PRD、根 specs、AGENTS 与 Skills。
- Dependencies：无；本 change 集成后解除 `connect-miniprogram-session-client` 的微信开发者工具编译阻断。
- Required external assets：已安装并登录的微信开发者工具 Stable 2.02.2608040、已分配测试号和模拟器，当前具备；正式 AppID、支付、后端和真机不属于本 UI2 change。
- Non-goals：个人中心 UI/文案/事件/身份逻辑变更，登录、手机号、Admin、后端、正式 AppID、上传、体验版、提审、部署或外部系统写入。
- 唯一裁决：静态负例必须先 Red，同一命令在最小修复后 Green/Refactor；完整小程序测试和 exact-SHA 微信开发者工具 UI2 主路径/相邻回归均 PASS，且两个独立 verifier 对同一 SHA 均 PASS 才接受。任一必需 Gate 失败即 `REJECT`。
