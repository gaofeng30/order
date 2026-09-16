## Context

当前 `pages/launch/launch` 把“用户端”绑定到 `openAuth()`，展示一个自制授权弹层；只有 `allowAuth()` 才调用既有导航进入首页。弹层声称申请昵称、头像与手机号，但实际并未调用微信原生授权能力，既阻断匿名浏览，也会让提审人员和用户误以为启动即强制收集资料。

已确认 PRD 和生效 MVP spec 都要求：浏览阶段免手机号，启动时不得强制手机号授权，首次提交订单前才执行手机号授权。并行 `connect-miniprogram-session-client` 只负责后台静默微信会话，owned paths 与本 change 不重叠；本 change 不等待也不修改该 candidate。

## Goals / Non-Goals

**Goals:**

- 点击“用户端”直接复用现有 `nav.go('home')` 进入首页。
- 完整删除错误授权弹层的专用 JS、WXML 与 WXSS。
- 以独立 focused UI1 测试和 exact-SHA 微信开发者工具 UI2 验证主路径及启动页相邻回归。

**Non-Goals:**

- 不实现或模拟微信静默会话、手机号原生授权、昵称/头像原生获取、手机号绑定或结算拦截。
- 不决定商户身份、改变商户端入口、修改首页、公共导航、路由表、API、Admin、订单或支付。
- 不为未来授权保留隐藏弹层、feature flag、兼容事件或本地存储。

## Decisions

### 用户端卡片直接使用现有通用导航

把用户端卡片改为 `data-to="home" bindtap="go"`，与启动页已有商户端卡片共用同一个 `go()` 和 `nav.go()`。这样只有一个路由来源，不新增专用 handler，也不修改公共导航表。

曾考虑保留 `allowAuth()` 但直接调用它，或把弹层隐藏；两者都会留下可被重新触发的错误授权语义和死代码，不满足“删除启动授权”的验收边界，因此拒绝。

### 删除整套弹层而非改文案

删除 `auth`/`wxIcon` 数据、inline 微信 SVG、`noop/openAuth/closeAuth/allowAuth`、授权模板和专用样式。改成“继续浏览”文案仍会制造无意义的一步并暗示授权，因此不采用。

### 新测试独立注册，避免并行 owned-path 冲突

新增 `tests/anonymous-entry-ui1.test.js`，只读复用现有 `page-harness.js`，通过直接命令执行 Red/Green/Refactor；不修改 `package.json` 或 `page-harness.js`。完整 `npm test` 继续作为既有回归。这样与并行登录 change 的 owned paths 不重叠。

### UI2 使用精确候选运行两个场景

候选提交后从 exact SHA 构建临时项目，仅注入测试 AppID 环境配置，在微信开发者工具 Stable 2.02.2608040 执行：启动页直接进入首页的主场景，以及冷重编译后启动页品牌/双入口/无授权弹层的相邻回归。receipt 不保存具体 AppID、账号标识或敏感身份值。

## Risks / Trade-offs

- [Risk] 后续静默会话集成后启动路由可能按商户身份分流。→ 本 change 只固定普通用户卡片的匿名进入语义；登录 change rebase 后必须重跑本 change 的有效回归，不能覆盖该约束。
- [Risk] 删除弹层后暴露首页自身的既有错误。→ UI2 主场景必须实际进入首页并检查 candidate 错误；不在本 change 顺手修首页相邻问题。
- [Risk] `package.json` 不注册新 focused 测试。→ writer/verifier/tasks 明确直接执行该文件；完整 npm 回归另行执行，owned-path 边界优先于并行修改共享注册表。

## Migration Plan

1. 在独立 writer worktree 完成真实 Red，再最小删除弹层并取得 Green/Refactor。
2. 跑完整 writer Gate，提交 owned paths，生成 exact candidate 和 UI2 receipt。
3. 两个独立 verifier 对同一 SHA 与 UI2 receipt 均 PASS 后，按用户既有授权仅本地纯快进集成并归档。
4. 回滚只需反向移除本 change 的 candidate bytes；不得在回滚中恢复强制授权行为。若 main 推进或并行登录 change 集成，先重建候选并重验。

## Open Questions

无。授权时机、匿名浏览与验收等级均已有明确产品和治理结论。
