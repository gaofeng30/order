## ADDED Requirements

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
