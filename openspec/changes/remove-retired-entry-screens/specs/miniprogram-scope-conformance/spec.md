## ADDED Requirements

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
