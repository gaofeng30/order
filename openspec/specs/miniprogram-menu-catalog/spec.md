# miniprogram-menu-catalog Specification

## Purpose
TBD - created by archiving change connect-miniprogram-menu-catalog. Update Purpose after archive.
## Requirements
### Requirement: Public catalog transport uses only the frozen anonymous API

小程序 public catalog client MUST 只从 `app.globalData.apiBaseUrl` 读取 origin，UI1 本地唯一默认值 MUST 为 `http://127.0.0.1:8080`。client MUST 且只能发送匿名 `GET /api/v1/catalog` 与 `GET /api/v1/catalog/products/:id`；不得附加身份、手机号、日期、餐段、库存、价格或降级参数，不得自动重试、缓存或读取 mock。

list/detail 的 HTTP 200 MUST 只向 public store 交付 provider 冻结字段。detail HTTP 404 MUST 映射为 `PRODUCT_NOT_FOUND`；网络失败、HTTP 503、其他非 200 或畸形 200 MUST 映射为非敏感 catalog unavailable。任何失败 MUST NOT 回退 `globalData.menu`、`MENU`、`menuList()`、`itemById()` 或内置商品。

#### Scenario: Anonymous list and detail requests use the local UI1 base

- **WHEN** UI1 页面在默认 app 配置下进入列表和 string product ID `9007199254740993` 的详情 lifecycle
- **THEN** `wx.request` 依次收到 `GET http://127.0.0.1:8080/api/v1/catalog` 与 `GET http://127.0.0.1:8080/api/v1/catalog/products/9007199254740993`
- **AND** request 不含身份、日期、餐段、库存、价格、fallback 或 request body，ID 不经过 JS number

#### Scenario: Catalog transport fails

- **WHEN** `wx.request` 网络失败、返回 503、返回其他非 200 或返回不符合冻结 DTO 的 200
- **THEN** client/store 返回统一非敏感 unavailable，页面进入可重试 error
- **AND** 不返回 mock 商品、不自动发送第二次请求、不暴露底层 response 或网络错误正文

#### Scenario: Detail returns provider 404

- **WHEN** detail GET 返回 HTTP 404 与 `PRODUCT_NOT_FOUND`
- **THEN** client/store 保留可识别的 not-found 结果供详情页进入 `not_found`
- **AND** 不调用 list、不选择第一件商品、不把 404 改写成通用 ready/error 商品

### Requirement: Public catalog store preserves canonical types and server order

catalog store MUST 只复制 category 的 `id/name/products` 与 product 的 `id/category_id/name/description/specification/price_cents`。id/category_id MUST 为规范正十进制 string，文本字段 MUST 为 string，`price_cents` MUST 为非负 safe integer；额外字段不得进入 public view 或 cart snapshot。store MUST 不排序、不按 tag/status/sold 筛选、不补假 ID，服务端 category/product 数组顺序 MUST 原样保留。

金额展示 MUST 只由 integer cents 确定性格式化；业务运算不得把元浮点作为事实源。合法的大于 `Number.MAX_SAFE_INTEGER` 的 string ID MUST 从 response、store、route、cart 到 confirm 逐字节保持。

#### Scenario: Large string ID and integer cents are stored

- **WHEN** provider 返回 product `id="9007199254740993"`、`category_id="9007199254740995"` 与 `price_cents=12345`
- **THEN** store 和页面保留两个 ID 的原 string，并将价格确定性显示为 `123.45`
- **AND** 不产生 numeric/scientific ID、浮点 cents、元价格事实字段或精度损失

#### Scenario: Repeated list responses are rendered stably

- **WHEN** provider 两次返回相同且已排序的 category/product arrays
- **THEN** store、菜单 group 与首页 flatten 顺序在两次运行中完全相同
- **AND** client 不按名称、tag、status、sales 或本地 mock 顺序重新排列

#### Scenario: Unsupported provider fields are present

- **WHEN** test fixture 附带 stock、sold、status、tags、image、availability、orderable 或其他非冻结字段
- **THEN** public store、页面 view model 与 cart snapshot 均不保留或展示这些字段
- **AND** 字段存在不产生可售、售罄、锁价或可下单结论

### Requirement: Home and menu expose complete recoverable list states

首页与菜单 MUST 在实际 `onShow` lifecycle 请求 catalog list，并显式呈现 `loading/empty/error/ready`。每次加载与 retry 前 MUST 先进入 loading；只有成功响应的 `categories:[]` 才是 empty。只要 categories 非空就 MUST 为 ready，启用空分类 MUST 保留为顺序不变的 ready group，products 固定为空数组。

error MUST 提供显式 retry action；retry MUST 发起一次新的真实 list request。任一 error/empty MUST 不显示 mock 商品。首页招牌 MUST 按 server category 顺序再按各自 product 顺序 flatten 后取前四件；不得按 tags/sold/status 筛选。首页和菜单 MUST 不携带 `p001`/`p005` 等假 product ID 或静态商品名称/价格。

#### Scenario: First list request fails and retry succeeds

- **WHEN** 首页或菜单第一次 `onShow` request 网络失败，用户触发 retry，第二次返回合法非空 catalog
- **THEN** 页面状态按 `loading → error → loading → ready` 变化且总请求数为 2
- **AND** error 阶段无 mock 商品，ready 内容只等于第二次 server response

#### Scenario: Catalog has no active categories

- **WHEN** list 返回 `{"categories":[]}`
- **THEN** 首页与菜单进入 empty 并提供再次加载 action
- **AND** 不回退 `data.CATS`、`MENU` 或任何本地商品

#### Scenario: Active category has no products

- **WHEN** list 返回一个或多个 `products:[]` 的 category
- **THEN** 菜单保持 ready、按 server 顺序渲染每个 category 与空 group，首页也保持 ready
- **AND** 不把该响应降为 empty、不隐藏空分类、不从 mock 填充商品

#### Scenario: Home selects the stable leading products

- **WHEN** server 按多个 category 返回超过四件商品且 fixture 的 tag/status/sold 值缺失
- **THEN** 首页只渲染按 category/product 原顺序 flatten 的前四件
- **AND** 结果不依赖 tag、sales、stock、availability、图片或假 ID

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
