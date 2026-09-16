## Why

当前启动页点击“用户端”不会进入匿名首页，而是先弹出自制的“昵称、头像与手机号”授权窗；用户只有点击“允许”才能继续。该行为直接违反已确认 PRD 的“浏览免手机号、启动时不得弹手机号授权、首次提交订单前才授权”规则，也是微信提审前必须消除的用户侧阻断。

## What Changes

- 用户点击启动页“用户端”后直接进入现有首页，不显示、不保留任何昵称、头像或手机号授权弹层。
- 删除只为该错误弹层服务的页面状态、事件、模板和样式；不增加替代提示、兼容分支或存储。
- 增加独立 UI1 契约测试，并对 exact candidate 在微信开发者工具执行 UI2 主路径与相邻启动页回归。

## Capabilities

### New Capabilities

- `miniprogram-anonymous-entry`: 定义普通用户从启动页直接匿名进入用户端、浏览阶段不触发资料或手机号授权，以及启动页相邻内容保持可用的行为。

### Modified Capabilities

- 无。

## Impact

- `gate_type`: `W2`；`ui_level_target`: `UI2`；规划时 `ui_level_actual`: `NOT_RUN`。
- Owner：主 Agent；branch `codex/allow-anonymous-miniprogram-entry`；worktree `/Users/vivix/.codex/worktrees/order-allow-anonymous-miniprogram-entry.Writer`。
- `base_sha`: `2d1400bd04f224c6c4da9eaca52e272514dff104`。
- Owned paths：`openspec/changes/allow-anonymous-miniprogram-entry/**`、`apps/wechat-miniprogram/pages/launch/launch.js`、`apps/wechat-miniprogram/pages/launch/launch.wxml`、`apps/wechat-miniprogram/pages/launch/launch.wxss`、`apps/wechat-miniprogram/tests/anonymous-entry-ui1.test.js`。
- Read-only shared contracts：根治理、质量 Gate 与 stage Skills；`docs/product/online-ordering-system-prd-0818.md`；生效 `mvp-product-baseline`；`app.json`、`utils/util.js`、`utils/navBehavior.js`、用户端首页、商户端页面、公共组件与全部其他测试；并行登录 change 的 `app.js`、`sessionApi.js`、`page-harness.js`、`package.json` 与其 OpenSpec 全部只读。
- Dependencies：当前 main 已包含生效的 0818 MVP 基线与已归档的个人中心 WXML 编译修复；不依赖尚未集成的登录 candidate。与登录 change 的 owned paths 不重叠。
- Required assets：本地 Node.js、微信开发者工具 Stable 2.02.2608040、已分配官方测试号与可复现启动页入口，当前均具备；UI2 receipt 必须绑定 exact candidate SHA。无需真机、手机号授权或后端。
- Non-goals：微信静默会话原生调用、手机号绑定、昵称/头像获取、结算授权、商户身份判断、商户端功能、首页内容/API、菜单、订单、支付、订阅消息、推送、PR、部署或提审。
- 唯一接受裁决：只有 focused Red→Green→Refactor、完整小程序回归、static/strict/owned/sensitive Gate 以及 exact-SHA UI2 的“用户端直接进入首页”和“启动页相邻内容正常”两个场景全部通过，才可形成 `CANDIDATE` 并进入独立验证；任一失败为 `REJECT`。
