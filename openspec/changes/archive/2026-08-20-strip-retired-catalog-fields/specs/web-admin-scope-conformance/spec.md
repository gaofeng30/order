## ADDED Requirements

### Requirement: The PC admin catalog carries no retired field

PC 后台种子与契约层 MUST NOT 保存或接受标签、过敏原、月售与数量库存。菜品管理表格 MUST NOT 提供库存列或销量列，菜品编辑表单 MUST NOT 提供库存输入项。

菜品的售罄与上下架控件 MUST 保留并保持可用。

#### Scenario: Seed and contract are loaded and inspected

- **WHEN** 在无 DOM 的运行环境中加载种子与契约模块
- **THEN** 商品不含 `tags`、`allergens`、`sold`、`stock` 任一字段，契约不接受数量入参
- **AND** 销售状态与保存商品的契约仍然可调用

#### Scenario: Product table and editor are audited

- **WHEN** 检查菜品管理页面
- **THEN** 表格不含库存列与销量列，编辑表单不含库存输入
- **AND** 售罄与上下架控件仍然存在

### Requirement: The PC admin dashboard todo list matches the first-phase scope

工作台待办 MUST 只统计待制作数。库存告急与待取超时 MUST NOT 出现在待办中。

商品销量排行 MUST 保留。

#### Scenario: Dashboard todos are audited

- **WHEN** 检查工作台待办
- **THEN** 不存在库存告急与待取超时条目
- **AND** 销量排行仍然渲染
