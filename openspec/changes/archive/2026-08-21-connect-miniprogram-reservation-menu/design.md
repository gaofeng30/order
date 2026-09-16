## Context

main 已提供匿名 `/api/v1/menu?date=YYYY-MM-DD&time=HH:MM`，响应包含服务端确认的 selection、meal 截单状态、分类顺序、按日售罄与最终可购事实。小程序菜单仍调用 `/api/v1/catalog` 并把所有返回商品当作可选择，导致客户端展示与服务端预约事实脱节。旧 catalog detail 和购物车快照已有稳定合同，本 change 不替换它们。

并行 `connect-miniprogram-session-client` 修改 `app.js/sessionApi/page-harness/package`，本 change 对这些路径全部只读；预约菜单是匿名读，不等待登录 candidate。

## Goals / Non-Goals

**Goals:**

- 当前取餐选择映射成上海时区今天/明天的严格日期，并只发送一个匿名预约菜单 GET。
- 严格校验并原样保留服务端 category/product 顺序、字符串 ID、整数分和 `sold_out/orderable`。
- 菜单显式呈现 loading/empty/error/ready、售罄/截单不可购和显式 retry；切换取餐时间后读取新选择，旧响应不能覆盖。
- exact-SHA UI2 使用受控 loopback 服务验证成功与失败恢复，而不记录 AppID、query 原文或身份值。

**Non-Goals:**

- 不从后端发现完整取餐时间配置，不修改当前 picker、默认时间或演示时钟；服务端拒绝选择时只显示 error，不自动改选或使用最近时间。
- 不做 checkout/prepay/payment/order、价格/库存锁定、购物车自动清理、详情迁移、登录/手机号或 Admin。
- 不修改后端 menu 合同、旧 catalog detail、共享 harness/package 或登录 candidate。

## Decisions

### 为预约菜单建立独立 API/store

新增 `reservationMenuApi.js` 负责唯一匿名 GET，新增 `reservationMenuStore.js` 负责上海日期映射和响应校验。旧 `catalogApi/catalogStore` 继续只服务商品详情与购物车快照，避免把日期/售罄字段塞进已冻结的 catalog 合同。

API 从 `app.globalData.apiBaseUrl` 读取 origin，按固定顺序发送 `date`、`time`，不附加身份、手机号或自动重试。`400 INVALID_MENU_SELECTION` 映射稳定 selection error，网络、503、其他状态与畸形 200 统一为无底层正文的 unavailable。

store 对固定请求选择执行闭环校验：`selection.date/time` 必须与请求一致，timezone 固定 `Asia/Shanghai`；meal code 只允许 lunch/dinner，cutoff 必须为 `+08:00` RFC3339，布尔类型精确；商品必须满足 `orderable === meal.orderable && !sold_out`。额外 provider 字段不进入页面模型或 cart snapshot。

### 页面以请求代次序阻止旧响应覆盖

每次 `loadMenu()` 递增页面私有 request ID，并在完成前再次比对当前 ID；较旧成功或失败都不得覆盖新选择的 loading/ready/error。`onShow`、`retryMenu` 和 `pickPickerTime` 是仅有加载入口，不增加隐式轮询、缓存或自动重试。

选择新时间先更新既有全局 pickup、关闭 picker、刷新标签，再请求新菜单。日期只在用户最终选择时间时生效；本 change 不改变 picker 数据源。

### 不可购商品可浏览但不能新增购物车动作

菜单仍展示服务端返回的售罄/截单商品及详情入口。`sold_out=true` 显示“售罄”；餐段不可购而商品未售罄显示“已截单”。两者均不渲染选择/步进器，并由 `add/openCustomize` 再做 JS guard，避免只靠模板。

已有 cart snapshot、购物车弹层和 checkout 不在本 change 自动删除或重验；它们必须在后续 checkout 强校验 change 处理。本 change 只保证当前菜单不能新增服务端已判定不可购的商品。

### 更新旧 menu-list 规格而保留 detail/cart 合同

`catalog-ui1.test.js` 中把菜单必须调用 catalog list 的旧断言改为 catalog detail/store 的保留回归；新的 `reservation-menu-ui1.test.js` 只读复用现有 harness 并覆盖预约 transport/store/page。`package.json` 与 `page-harness.js` 不修改，新 focused 文件由 tasks/verifier 直接执行，避免并行登录 owned-path 冲突。

### UI2 使用可控 loopback 菜单服务

候选从 exact SHA 解包，仅注入测试 AppID 环境配置。受控本地服务先返回含可购与售罄商品的合法响应，验证菜单渲染和不可购动作；第二场景返回 503，切换为合法响应后用户点击重试恢复。receipt 只记录版本、场景、边界和哈希，不记录具体 AppID、原始 query 或身份数据。

## Risks / Trade-offs

- [Risk] 当前 picker 仍来自本地已确认初始配置，未来后台改配置后可能选择到服务端不接受的点。→ 本 change 对 400 fail closed 且不猜测；配置发现必须另开 API/OpenSpec，不能在这里扩合同。
- [Risk] 已在 cart 中的旧 snapshot 可能在菜单刷新后变为不可购。→ 不自动删除用户选择；后续 checkout 强校验是下单前权威边界，本 change 明确不声称购物车/支付安全。
- [Risk] onShow 与选时请求交错。→ request ID 使只有最新选择可写页面状态，并用反序完成测试证明。
- [Risk] login candidate 后续 rebase 可能触发新的 app onLaunch 请求。→ owned paths 不重叠；UI2 和 verifier 检查菜单请求仍匿名且无额外身份字段。

## Migration Plan

1. 先修改/新增 owned tests，真实 Red 证明菜单仍调用旧 catalog 且不认识预约事实。
2. 最小实现 API/store/page/WXML/WXSS，取得相同 focused Green 与 Refactor，再跑完整回归和只读后端合同测试。
3. 提交 exact candidate，在微信开发者工具执行两条 UI2 manifest 场景并生成外部 receipt。
4. 两个独立 verifier 对同一 SHA/receipt 均 PASS 后，按用户既有授权仅本地纯快进、集成回归并归档。
5. 回滚只反向移除本 change bytes，菜单恢复旧 catalog list；不得顺手改登录、checkout、后端或 Admin。

## Open Questions

无。服务端预约菜单字段、售罄/截单语义、旧 detail/cart 边界和当前验收环境均已由生效合同固定。
