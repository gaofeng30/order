# miniprogram-scope-conformance Specification

## Purpose
定义微信小程序端暴露的页面、路由、入口、全局态、接口契约与算价链必须与生效 spec `mvp-product-baseline` 的一期范围保持一致，不得存在已排除能力的实现、入口或残留表述。
## Requirements
### Requirement: The mini program exposes no route for an excluded capability

`apps/wechat-miniprogram/app.json` 的 `pages` 列表 MUST NOT 包含任何实现生效 spec 排除能力的路由。会员等级、会员名单、名单导入、会员编辑、优惠券列表、优惠券编辑与用户卡包的页面目录 MUST NOT 存在于仓库中。

#### Scenario: Route list is audited

- **WHEN** 检查 `app.json` 的 `pages` 列表
- **THEN** 其中不存在 `my-coupons`、`admin-levels`、`admin-members`、`admin-member-edit`、`admin-member-import`、`admin-coupons`、`admin-coupon-edit`
- **AND** 这些页面对应的目录在仓库中不存在

#### Scenario: Excluded page is reintroduced

- **WHEN** 任一 change 为排除能力新增页面目录或 `app.json` 路由
- **THEN** scope 一致性检查失败
- **AND** 该能力必须先取得新的正式范围授权

### Requirement: The mini program holds no state or contract for an excluded capability

小程序 MUST NOT 在 `app.js` 的 `globalData` 中保存会员等级、会员名单、优惠券或券使用计数。`utils/data.js` MUST NOT 导出这些能力的种子数据。`utils/api.js` MUST NOT 声明这些能力的后端接口契约。算价引擎 `utils/promo.js` 与券票卡组件 MUST NOT 存在。

#### Scenario: Global state is inspected

- **WHEN** 加载小程序 App 实例并检查 `globalData`
- **THEN** 不存在 `levels`、`members`、`coupons`、`couponUsed` 任一键

#### Scenario: Contract layer is inspected

- **WHEN** 加载 `utils/api.js` 并枚举其导出
- **THEN** 不存在会员等级、会员名单、名单导入、优惠券或用户卡包的任何方法

#### Scenario: Seed data is inspected

- **WHEN** 加载 `utils/data.js` 并枚举其导出
- **THEN** 不存在 `LEVELS`、`MEMBERS`、`COUPONS`、`MY_COUPON_USED` 任一导出

### Requirement: Checkout pricing shows only what the first-phase scope defines

结算页 MUST NOT 提供选券入口、券弹层或等级折扣展示。在全局折扣率实现之前，结算页 MUST 只展示商品小计并令应付金额等于商品小计。

订单 MUST NOT 固化会员等级或优惠券相关字段。客户端 MUST NOT 依赖任何算价引擎模块完成结算。

#### Scenario: User checks out without any discount mechanism

- **WHEN** 用户加购商品后进入结算页并提交订单
- **THEN** 结算页展示商品小计且应付金额等于商品小计
- **AND** 生成的订单不含等级或优惠券字段

#### Scenario: Checkout template is audited

- **WHEN** 检查结算页 WXML
- **THEN** 不存在选券入口、券弹层或等级折扣行
- **AND** 结算页脚本不引用算价引擎模块

### Requirement: User and merchant entries drop excluded capabilities

个人中心 MUST NOT 展示「我的优惠券」入口或会员等级胶囊。商户中心 MUST NOT 展示「会员与营销」分组。菜品编辑 MUST NOT 包含删除菜品时摘除优惠券适用范围的联动。

#### Scenario: Profile entries are audited

- **WHEN** 检查个人中心页面与模板
- **THEN** 不存在优惠券入口、等级胶囊或券数量字段

#### Scenario: Merchant center entries are audited

- **WHEN** 检查商户中心页面与模板
- **THEN** 不存在「会员与营销」分组及其下属入口

#### Scenario: Product deletion no longer touches coupons

- **WHEN** 商户在菜品编辑页删除一个菜品
- **THEN** 操作不引用任何优惠券数据
- **AND** 不产生与券适用范围相关的提示

### Requirement: The mini program exposes no retired entry screen

品牌选择页与小程序商户端经营工作台 MUST NOT 存在于 `app.json` 的 `pages` 列表或仓库中。路由封装 MUST NOT 提供指向品牌选择页的跳转方法。

#### Scenario: Retired routes are audited

- **WHEN** 检查 `app.json` 的 `pages` 列表与页面目录
- **THEN** 不存在 `pages/brand/brand` 与 `pages/admin-dashboard/admin-dashboard`
- **AND** 路由封装不含指向品牌选择页的方法

### Requirement: The home screen carries no marketing surface

首页 MUST NOT 渲染营销 Banner 轮播、入群模块、会员面板、推荐商品或今日招牌。首页脚本 MUST NOT 保留这些能力对应的数据、轮播索引或事件处理器。

首页服务宫格 MUST NOT 提供会员中心或联系客服入口。

#### Scenario: Home surfaces are audited

- **WHEN** 检查首页脚本与模板
- **THEN** 不存在 Banner、轮播索引、入群、会员面板、招牌位或推荐商品
- **AND** 服务宫格不含会员中心与联系客服

### Requirement: The merchant tab bar carries only first-phase tabs

小程序商户端底部导航 MUST 只有订单、核销、菜品三格，MUST NOT 包含工作台或商户中心。

商户中心页在导航收敛后 MUST 仍然可达，MUST NOT 成为无入口的孤儿页。

#### Scenario: Merchant tab bar is audited

- **WHEN** 检查底部导航组件的商户端分支
- **THEN** 只声明订单、核销、菜品三格
- **AND** 不声明工作台或商户中心

#### Scenario: Merchant center remains reachable

- **WHEN** 商户进入订单页
- **THEN** 页面提供进入商户中心的入口
- **AND** 商户中心页仍存在于仓库中

### Requirement: Seed data owned only by a removed screen is dropped

仅供已删除页面使用的种子数据 MUST 一并移除，MUST NOT 作为无主常量保留在种子模块中。

#### Scenario: Orphan seed is audited

- **WHEN** 加载种子模块并枚举导出
- **THEN** 不存在仅由已删除的经营工作台使用的销量排行常量

### Requirement: Product records carry no retired catalog field

商品记录 MUST NOT 保存标签、过敏原、月售或数量库存。种子数据、接口契约与页面数据 MUST 一并不含这四类字段。

商品的销售状态字段 MUST 保留：可售性由上下架与售罄开关承担，见生效 spec `mvp-product-baseline` 的 `Product availability uses a per-service-date sellout switch`。

#### Scenario: Seed products are inspected

- **WHEN** 加载种子模块并逐条检查商品
- **THEN** 不存在 `tags`、`allergens`、`sold`、`stock` 任一字段
- **AND** 每条商品仍带销售状态字段

#### Scenario: Product contract is inspected

- **WHEN** 检查商品接口契约的入参、校验与新建默认值
- **THEN** 不接受数量入参、不做库存校验、不为新建商品填充标签或过敏原
- **AND** 销售状态契约仍然可调用

### Requirement: Merchant product screens show no retired catalog field

商户端菜品列表 MUST NOT 展示库存数、库存告急标记或月售。菜品编辑 MUST NOT 提供库存输入项。

菜品列表 MUST 保留售罄与上下架控件。

#### Scenario: Merchant product list is audited

- **WHEN** 检查商户端菜品列表的页面数据与模板
- **THEN** 行数据不含 `stock`、`sold`、`tags`、`allergens` 或告急标记
- **AND** 售罄切换控件仍然存在

#### Scenario: Merchant product editor is audited

- **WHEN** 检查菜品编辑页的脚本与模板
- **THEN** 不存在库存字段、库存输入或数量校验

#### Scenario: Sale status still works after the fields are gone

- **WHEN** 商户对某商品切换售罄再切回可售
- **THEN** 该商品的销售状态随之变化
- **AND** 过程不依赖任何数量字段
