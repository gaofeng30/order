## MODIFIED Requirements

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

## ADDED Requirements

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
