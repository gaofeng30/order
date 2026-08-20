## MODIFIED Requirements

### Requirement: The merchant tab bar carries only first-phase tabs

小程序商户端底部导航 MUST 只有订单、核销、菜品三格，MUST NOT 包含工作台或商户中心。

商户中心页 MUST NOT 存在于小程序端：其承载的营业设置、分类管理与开屏图层均属 PC 后台能力。收敛后仍需保留的「切换身份」MUST 由订单页导航栏右侧承担，MUST NOT 留下无入口的孤儿页。

#### Scenario: Merchant tab bar is audited

- **WHEN** 检查底部导航组件的商户端分支
- **THEN** 只声明订单、核销、菜品三格
- **AND** 不声明工作台或商户中心

#### Scenario: The identity screen stays reachable

- **WHEN** 商户进入订单页
- **THEN** 导航栏右侧提供返回身份选择页的入口
- **AND** 商户中心页不存在于仓库中

## ADDED Requirements

### Requirement: The mini program merchant surface is exactly four screens

小程序商户端 MUST 只有订单管理、订单详情、扫码核销、菜品四屏。完整商品配置、分类管理、营业设置、开屏图层与任何名单管理 MUST NOT 存在于小程序端，它们属 PC 后台能力。

菜品页 MUST 只提供可售 / 售罄切换，MUST NOT 提供上架新菜、编辑菜品或上下架开关。

营业状态切换 MUST 保留在小程序商户端的订单页：现场临时休息需要手机可操作。

小程序端 MUST NOT 保留任何「建设中」占位入口。

#### Scenario: Merchant routes are audited

- **WHEN** 检查 `app.json` 的商户端路由与页面目录
- **THEN** 只存在订单管理、订单详情、扫码核销、菜品四屏
- **AND** 菜品编辑、分类管理、营业设置、开屏图层与商户中心均不存在

#### Scenario: A merchant opens the product screen

- **WHEN** 商户进入菜品页
- **THEN** 每行只提供可售 / 售罄切换
- **AND** 不存在上架新菜入口、编辑入口或上下架开关

#### Scenario: A merchant needs to pause service on site

- **WHEN** 商户在订单页切换营业状态
- **THEN** 门店状态随之变化
- **AND** 该能力不依赖 PC 后台

### Requirement: The mini program renders no locally stored launch layer

小程序端 MUST NOT 以本机存储实现开屏装饰图层。图层配置与图片 MUST 由服务端下发；在该接口就位前，小程序 MUST 不渲染任何图层，MUST NOT 保留只能读取本机存储的图层模块或组件。

#### Scenario: The launch layer is audited

- **WHEN** 检查小程序的图层模块、组件与身份选择页
- **THEN** 不存在读取本机存储的图层实现
- **AND** 身份选择页不渲染图层
