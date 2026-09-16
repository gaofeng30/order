## Why

当前小程序菜单页仍调用旧的匿名 catalog list，只能展示长期上架商品，完全不读取已经集成到 main 的预约菜单合同 `GET /api/v1/menu?date=YYYY-MM-DD&time=HH:MM`。因此用户看不到所选日期/餐段的按日售罄、餐段截单和最终 `orderable`，页面还会继续把不可购买商品显示为可选择。这是进入真实 checkout 之前最早的用户侧数据错误。

## What Changes

- 新增小程序预约菜单 API/store，只针对当前已选取餐日期和时间读取 `/api/v1/menu`，严格校验 selection、meal、categories 与商品售罄/可购事实，不回退 catalog 或 mock。
- 菜单页在 `onShow`、显式重试和选择新取餐时间后加载预约菜单；较旧请求不得覆盖较新的选择。
- 保留不可购商品的浏览入口，但按服务端事实显示“售罄”或“已截单”，并禁止新增选择或加购；旧 catalog detail、购物车快照、checkout 和全部 Admin 保持边界不变。
- 更新被本能力取代的 menu-list catalog 测试/规格，并以 exact-SHA 微信开发者工具 UI2 验证正常可购/售罄展示和错误重试恢复。

## Capabilities

### New Capabilities

- 无。

### Modified Capabilities

- `miniprogram-menu-catalog`: 菜单 list 从旧 catalog 切换到按所选日期/时间的预约菜单合同，并消费服务端 `sold_out/orderable`；旧 catalog detail 与本地购物车快照边界继续保留。

## Impact

- `gate_type`: `W2`；`ui_level_target`: `UI2`；规划时 `ui_level_actual`: `NOT_RUN`。
- Owner：主 Agent；branch `codex/connect-miniprogram-reservation-menu`；worktree `/Users/vivix/.codex/worktrees/order-connect-miniprogram-reservation-menu.Writer`。
- `base_sha`: `0dea0acc0ed8189b7bd7bf06cf3d32b5b8890d93`。
- Owned paths：`openspec/changes/connect-miniprogram-reservation-menu/**`、`apps/wechat-miniprogram/utils/reservationMenuApi.js`、`apps/wechat-miniprogram/utils/reservationMenuStore.js`、`apps/wechat-miniprogram/pages/menu/menu.js`、`apps/wechat-miniprogram/pages/menu/menu.wxml`、`apps/wechat-miniprogram/pages/menu/menu.wxss`、`apps/wechat-miniprogram/tests/reservation-menu-ui1.test.js`、`apps/wechat-miniprogram/tests/catalog-ui1.test.js`。
- Read-only shared contracts：根治理、质量 Gate 与 stage Skills；0818 PRD；生效 `mvp-product-baseline`、`miniprogram-menu-catalog` 与 `reservation-menu-availability` 合同；后端 menu handler/model/router/migrations；`app.js`、`package.json`、`page-harness.js`、`catalogApi.js`、`catalogStore.js`、`data.js`、`util.js`、detail/confirm/cart 及其他页面/测试；并行登录 change 的全部 owned paths 与 OpenSpec。
- Dependencies：当前 main 已包含 `serve-reservation-menu-availability` 的 `/api/v1/menu` 运行合同和已归档匿名入口；不依赖尚未集成的登录 candidate，与其 owned paths 不重叠。
- Required assets：本地 Node.js、只读 Go contract tests、微信开发者工具 Stable 2.02.2608040、已分配官方测试号、可切换成功/503 响应的本地 loopback 菜单服务，均可用；UI2 receipt 必须绑定 exact candidate SHA。无需正式 AppID、真机、手机号、支付或 Admin。
- Non-goals：不改取餐 picker 的配置发现/演示时钟，不实现 checkout 强校验、报价、订单、支付、登录/会话/手机号、详情 API 迁移、购物车失效清理、营业状态、Admin 或后端写接口；不 push、PR、部署、上传或提审。
- 唯一接受裁决：只有 focused Red→Green→Refactor、完整小程序/旧 catalog detail/后端 menu contract 回归、static/strict/owned/sensitive Gate，以及 exact-SHA UI2 的“预约菜单可购/售罄事实”和“503 后显式重试恢复”两个场景全部通过，才可形成 `CANDIDATE` 并进入双独立验证；任一失败为 `REJECT`。
