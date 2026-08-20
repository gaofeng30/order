## MODIFIED Requirements

### Requirement: Home and menu expose complete recoverable list states

菜单 MUST 在实际 `onShow` lifecycle 请求 catalog list，并显式呈现 `loading/empty/error/ready`。每次加载与 retry 前 MUST 先进入 loading；只有成功响应的 `categories:[]` 才是 empty。只要 categories 非空就 MUST 为 ready，启用空分类 MUST 保留为顺序不变的 ready group，products 固定为空数组。

error MUST 提供显式 retry action；retry MUST 发起一次新的真实 list request。任一 error/empty MUST 不显示 mock 商品。菜单 MUST 不携带 `p001`/`p005` 等假 product ID 或静态商品名称/价格。

**首页 MUST NOT 请求 catalog list，也 MUST NOT 渲染任何商品列表或招牌位。** 首页只承载门店信息、营业状态、门店公告与一期入口；商品浏览的唯一入口是菜单。原「首页招牌取前四件」的约束随之作废。

#### Scenario: First list request fails and retry succeeds

- **WHEN** 菜单第一次 `onShow` request 网络失败，用户触发 retry，第二次返回合法非空 catalog
- **THEN** 页面状态按 `loading → error → loading → ready` 变化且总请求数为 2
- **AND** error 阶段无 mock 商品，ready 内容只等于第二次 server response

#### Scenario: Catalog has no active categories

- **WHEN** list 返回 `{"categories":[]}`
- **THEN** 菜单进入 empty 并提供再次加载 action
- **AND** 不回退 `data.CATS`、`MENU` 或任何本地商品

#### Scenario: Active category has no products

- **WHEN** list 返回一个或多个 `products:[]` 的 category
- **THEN** 菜单保持 ready、按 server 顺序渲染每个 category 与空 group
- **AND** 不把该响应降为 empty、不隐藏空分类、不从 mock 填充商品

#### Scenario: Home does not touch the catalog

- **WHEN** 用户进入首页并触发 `onShow`
- **THEN** 不产生任何 catalog list request
- **AND** 首页不渲染商品卡、招牌位或任何 list 状态
