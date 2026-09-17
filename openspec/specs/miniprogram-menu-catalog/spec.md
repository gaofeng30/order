# miniprogram-menu-catalog Specification

## Purpose
TBD - created by archiving change connect-miniprogram-menu-catalog. Update Purpose after archive.
## Requirements
### Requirement: Public catalog transport uses only the frozen anonymous API

小程序 MUST 从 `app.globalData.apiBaseUrl` 读取唯一 origin。菜单 list MUST 只发送匿名 `GET /api/v1/menu?date=YYYY-MM-DD&time=HH:MM`；商品 detail MUST 继续只发送匿名 `GET /api/v1/catalog/products/:id`。两者均不得附加身份、手机号、授权、价格、库存或降级参数，不得自动重试、缓存或读取 mock。

预约菜单 HTTP 400 且稳定码 `INVALID_MENU_SELECTION` MUST 映射为非敏感 selection invalid；网络失败、HTTP 503、其他非 200 或畸形 200 MUST 映射为 menu unavailable。detail HTTP 404 MUST 继续映射 `PRODUCT_NOT_FOUND`，其他失败继续映射 catalog unavailable。任何失败 MUST NOT 回退 `globalData.menu`、`MENU`、`menuList()`、`itemById()` 或旧 catalog list。

#### Scenario: Selected pickup loads the versioned reservation menu

- **WHEN** 当前 pickup 对应上海日期 `2026-08-22` 与时间 `18:00`
- **THEN** `wx.request` 只收到 `GET <apiBaseUrl>/api/v1/menu?date=2026-08-22&time=18%3A00`
- **AND** 请求不含 Authorization、手机号、用户或商户字段

#### Scenario: Product detail keeps the frozen catalog path

- **WHEN** 用户浏览 ID 为 `9007199254740993` 的商品详情
- **THEN** detail 继续请求 `GET <apiBaseUrl>/api/v1/catalog/products/9007199254740993`
- **AND** menu 迁移不得改变 detail 的 404/503、字符串 ID 或匿名合同

#### Scenario: Reservation menu transport fails

- **WHEN** menu 请求遇到网络失败、503、其他状态或畸形 200
- **THEN** 页面得到稳定、无底层正文的 menu unavailable，且不自动重试或回退 catalog/mock

### Requirement: Public catalog store preserves canonical types and server order

catalog detail store MUST 继续只复制 product 的 `id/category_id/name/description/specification/price_cents`，并保持 canonical 正十进制 string ID、文本 string、非负 safe integer 分价和额外字段剥离。

reservation menu store MUST 精确校验 `selection.date/time/timezone`、`meal.code/cutoff_at/orderable`、非 null categories，以及每个商品的上述 catalog 字段和布尔 `sold_out/orderable`。selection MUST 与请求一致，timezone MUST 为 `Asia/Shanghai`，meal code MUST 只为 lunch/dinner，cutoff MUST 为带 `+08:00` 的 RFC3339；商品 MUST 满足 `orderable === meal.orderable && !sold_out`。store MUST 不排序、不补假 ID、不推断售罄/截单，category/product 数组顺序 MUST 原样保留，额外 provider 字段不得进入页面模型或 cart snapshot。

#### Scenario: Reservation response preserves large IDs and availability

- **WHEN** 合法响应包含超出 JavaScript safe integer 的 string ID、整数分、一个可购商品与一个当日售罄商品
- **THEN** 页面模型保持原顺序和原 ID，分别保留精确 `sold_out/orderable`
- **AND** 价格只由整数分格式化，额外库存、sales、tags 或身份价格字段被丢弃

#### Scenario: Response selection or orderability is inconsistent

- **WHEN** selection 与请求不一致、timezone/meal/cutoff 类型非法，或商品 orderable 不等于 meal orderable 且非售罄
- **THEN** 整个响应 fail closed 为 menu unavailable
- **AND** 不返回部分分类、不修正服务端事实或回退旧 catalog

### Requirement: Home and menu expose complete recoverable list states

菜单 MUST 在实际 `onShow` lifecycle 按当前 pickup 请求 reservation menu，并显式呈现 `loading/empty/error/ready`。每次加载、选择新取餐时间与 retry 前 MUST 进入 loading；只有合法响应的 `categories:[]` 才是 empty。只要 categories 非空即为 ready，并原样保留服务端顺序。较旧请求无论成功或失败都 MUST NOT 覆盖较新的 pickup 请求状态。

首页 MUST NOT 请求 catalog 或 reservation menu，也 MUST NOT 渲染商品列表。菜单失败后只允许用户显式 `retryMenu`，不得自动重试、回退旧 catalog list 或 mock。

#### Scenario: First reservation menu request fails and retry succeeds

- **WHEN** 菜单第一次 `onShow` 请求返回 503，用户点击重试后同一选择返回合法非空 menu
- **THEN** 页面依次呈现 loading、error、loading、ready
- **AND** 只产生两次同路径请求，无自动第三次请求或 mock 内容

#### Scenario: Latest pickup selection wins

- **WHEN** 用户选择新取餐时间后，新请求先完成，旧请求随后才完成
- **THEN** 页面保持新 selection 的 categories 和状态
- **AND** 旧成功或失败均不得覆盖、清空或报错

#### Scenario: Reservation menu is empty

- **WHEN** 合法响应包含完整 selection/meal 与 `categories:[]`
- **THEN** 页面显示 empty 和显式再次加载入口
- **AND** 不展示 seed、旧 catalog 商品或假分类

#### Scenario: Home does not touch product APIs

- **WHEN** 用户进入或再次展示首页
- **THEN** 不产生 catalog 或 reservation menu 请求
- **AND** 首页只保留当前一期门店与导航内容

### Requirement: Detail exposes loading not-found error retry and ready without fallback

详情页 MUST 在实际 `onLoad(opts)` 使用路由原始十进制 string ID 发起 detail GET，并显式呈现 `loading/not_found/error/ready`。200 合法 DTO 才能进入 ready；HTTP 404 MUST 进入 not_found 且 product 为 null；网络/503/其他错误 MUST 进入可重试 error。retry MUST 继续使用同一未数值化 string ID 发起一次新请求。

页面 MUST 不请求 list、不调用 `itemById/menuList`、不回退首商品或 `p001`。ready 只展示冻结文本字段与整数分价格，动作只表示本地选择；不得展示或推断图片、tags、sales、sold/status、availability/orderable、过敏原、库存或真实可下单状态。

#### Scenario: Unknown product returns 404

- **WHEN** 路由 ID 未知或隐藏且 provider 返回 404
- **THEN** lifecycle 按 `loading → not_found`，product 保持 null，页面显示明确未找到状态
- **AND** request count 为 1，不出现 `p001`、首商品、mock 名称或 ready action

#### Scenario: Detail error is retried successfully

- **WHEN** 第一次 detail request 返回 503，用户触发 retry，第二次返回合法 product
- **THEN** 状态按 `loading → error → loading → ready` 变化且两次 URL 使用完全相同的 string ID
- **AND** error 阶段没有 mock fallback，ready product 只来自第二次 response

#### Scenario: Detail keeps an ID beyond JavaScript safe integer

- **WHEN** route 与 provider product ID 都是 `9007199254740993`
- **THEN** request URL、page product、cart key 与 confirm item 都逐字节保持该 string
- **AND** 不调用 `Number`、`parseInt` 或一元加号转换 product/category ID

### Requirement: Cart and confirm use an immutable local catalog snapshot and integer cents

首次选择某商品时 cart MUST 复制当次 canonical server product，entry MUST 只保存该 product snapshot、正整数 qty、flavors 与 note。已有 entry 的数量或偏好变化 MUST 不回查 catalog/mock，也不得隐式改写 product snapshot；remove/clear 后的新选择才建立新 snapshot。

cart count、line total 与 grand total MUST 使用 integer `price_cents × qty`，且 MUST 拒绝非正整数 qty 或 unsafe cents 结果。cart/confirm MAY 在每次读取时临时派生 `price_text` 与既有 promo/pay 调用签名需要的兼容元值，但派生 MUST 只由 cents 确定性生成，MUST NOT 写回 snapshot/store 或成为金额事实。

确认页 MUST 只从 cart snapshot/view 构造商品展示、编辑与合计，不调用 catalog、`data.itemById()`、`data.menuList()` 或 `globalData.menu`。现有 `loadPromo/recalc/openCoupon/pay` handler 与入口 MUST 保留且不得因新的 item shape 抛错；`utils/api.js`、`utils/promo.js` 与 result/order/payment/history 页面 MUST byte-unchanged。snapshot 只证明本地浏览/选择结果，不锁价、不锁库存、不表示 availability/orderable，不替代未来 quote/order validation。

#### Scenario: Product is selected and confirmed from its server snapshot

- **WHEN** 用户从 list/detail 选择 string ID 商品并填写 qty/flavors/note 后进入 confirm
- **THEN** global cart entry 精确保存 canonical product 字段与用户输入，confirm 显示相同 ID、name、integer cents 单价、数量和整数分小计/合计
- **AND** 存储的 product snapshot 不含 price text、status、stock、availability、orderable、tag、sales、image 或 mock 字段

#### Scenario: Confirm keeps the existing all-scope promo path operable

- **WHEN** cart 含 canonical snapshot，受控现有 mock 状态只提供 all-scope coupon，confirm 执行真实 `onLoad/loadPromo/recalc/openCoupon`
- **THEN** promo UI 可渲染且不因 snapshot item shape 抛错，商品 name/string ID/兼容元值来自 snapshot-derived view
- **AND** catalog request 与 mock-menu read 均为 0，category/item-scoped coupon 适用性仍标记未验证

#### Scenario: Existing mock pay handler is not broken by the catalog item shape

- **WHEN** 受控 fixture 对非空 snapshot cart 调用真实 confirm `pay` handler
- **THEN** handler 继续生成现有 P0 mock order tuple 并执行既有导航，tuple 的 string product ID 与兼容价格只由 snapshot/cents 派生
- **AND** 不读取 catalog/mock-menu，不修改 order/payment/history 页面，也不把该 non-regression 记为真实 order/payment PASS

#### Scenario: Original response changes after selection

- **WHEN** 选择完成后测试修改原 response 对象，并让后续 catalog request 失败
- **THEN** cart 与 confirm 仍显示首次选择时复制的 snapshot 和相同 cents 合计
- **AND** confirm lifecycle 的 catalog request count 为 0

#### Scenario: Legacy menu access is unavailable during confirmation

- **WHEN** harness 将 `globalData.menu` 读取设置为抛错后加载 confirm
- **THEN** confirm 仍从 cart snapshot 达到稳定商品明细和合计结果
- **AND** 不执行 `itemById/menuList`，不把 legacy mock 缺失改写为 empty/error 商品

#### Scenario: Local snapshot boundary is presented

- **WHEN** 菜单、详情或确认页展示 catalog product 与本地合计
- **THEN** 页面只把动作描述为本地选择，并明确真实价格/库存/下单仍需后续服务端校验
- **AND** 本 change 不产生 quote、order、payment、availability 或 locked-price PASS

### Requirement: Local Node harness proves UI1 and preserves higher-level external blockers

仓库 MUST 在 `apps/wechat-miniprogram/` 提供零第三方依赖的局部 package/lock 与 Node UI1 harness。harness MUST 从实际仓库路径加载 App、Page JS、behavior、catalog API/store/cart 和 WXML 状态契约，实际调用相关 lifecycle 与 retry/selection handlers；只允许模拟微信宿主 API 与 `wx.request` transport，不得复制 production 页面状态机或用 fake store 替代被测模块。

UI1 matrix MUST 覆盖首请求失败后 retry 成功、空目录、启用空分类、404、503/网络失败、大于 JS 安全整数 string ID、integer cents、稳定渲染、no mock fallback、snapshot confirm，以及现有 all-scope promo/pay handler/入口 non-regression。UI2 微信开发者工具/体验版与 UI3 真机 MUST 保持 `BLOCKED_EXTERNAL`，不得由 Node PASS 降级或替代。

#### Scenario: Legacy Red is reproduced before implementation

- **WHEN** candidate 的 harness/tests 在隔离的 base `94e04bf26e37e93299c26ef2c9c8aa7552619444` 小程序文件上运行 focused legacy boundary
- **THEN** list lifecycle 断言到达实际 request count `0`，unknown detail 断言到达实际 fallback ID `p001`
- **AND** Red 不得只因 `catalogApi.js`、`catalogStore.js` 或其他新模块缺失而失败

#### Scenario: Candidate UI1 matrix passes on exact source

- **WHEN** writer 与 independent verifier 分别在当前 writer tree 和 clean detached exact candidate SHA 运行局部 npm test
- **THEN** 真实 page lifecycle、error/retry、WXML state/action、transport URL/type/order、snapshot 与 confirm promo/pay non-regression matrix 全部 PASS
- **AND** 测试没有删除/放宽 Red、没有用 mock page/store/client 替换 production 代码，candidate 外的结果不计入 PASS

#### Scenario: Higher UI assets remain unavailable

- **WHEN** UI1 Node matrix PASS 但没有锁定微信开发者工具/体验版、真实 HTTPS 域名、项目权限、指定真机与受控账号
- **THEN** `ui_level_actual` 最高为 UI1，UI2 与 UI3 分别记录 owner、缺失资产和恢复条件
- **AND** 不宣称微信平台编译、体验版、真机、真实域名或生产行为已验证

### Requirement: Reservation menu surfaces server orderability without inventing fallback

菜单 MUST 展示服务端返回的所有 browseable 商品。`sold_out=true` 的商品 MUST 标记“售罄”；`meal.orderable=false` 且商品未售罄时 MUST 标记“已截单”。任一 `orderable=false` 商品 MUST 保留详情浏览，但 MUST 不渲染选择/步进器，且 JS handler MUST 拒绝新增选择或加购。可购商品继续使用既有 customize/cart snapshot；snapshot MUST 只保留冻结 catalog 字段，不把 menu availability 当作 checkout、锁价或库存保证。

选择新取餐时间后菜单 MUST 立即按新 date/time 重新读取；页面不得沿用旧 selection 的 `sold_out/orderable`。checkout 强校验、已有 cart snapshot 失效处理和支付均属于后续 change。

#### Scenario: Sold-out product remains browseable but cannot be selected

- **WHEN** 服务端返回 `sold_out=true, orderable=false` 的商品
- **THEN** 商品仍按服务端顺序展示并可进入详情，同时显示“售罄”
- **AND** 点击原选择区域不打开 customize、不增加 cart，页面不显示步进器

#### Scenario: Meal is cut off without product sellout

- **WHEN** meal `orderable=false` 且商品 `sold_out=false, orderable=false`
- **THEN** 商品显示“已截单”而不是“售罄”
- **AND** 页面不得猜测新的取餐时间或把商品标为可购

#### Scenario: Orderable product uses the existing cart snapshot boundary

- **WHEN** 商品 `sold_out=false, orderable=true` 且用户完成既有 customize 选择
- **THEN** cart 继续保存 id/category/text/integer-cents 的本地快照和数量/偏好
- **AND** snapshot 不保存 `sold_out/orderable`，也不声明 checkout、锁价、库存或支付已验证
