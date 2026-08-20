## ADDED Requirements

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
