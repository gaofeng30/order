## ADDED Requirements

### Requirement: Every product declares a meal period

商品记录 MUST 保存餐段可售字段，取值 MUST 为 `all`（全天）、`lunch`（午餐）、`dinner`（晚餐）三者之一。

保存商品的契约 MUST 把餐段可售视为必填并校验取值：缺失或非三选一 MUST 拒绝保存。批量调价等复用保存契约的路径 MUST 透传该字段，MUST NOT 在更新时丢失。

餐段的取值集合与展示标签 MUST 由契约层单一提供，页面 MUST 从契约渲染，MUST NOT 各自硬编码。

#### Scenario: Seeded products are inspected

- **WHEN** 在无 DOM 的运行环境中加载商品种子
- **THEN** 每条商品都带合法的餐段可售取值

#### Scenario: A product is saved without a meal period

- **WHEN** 保存商品时未提供餐段可售，或提供了三选一以外的值
- **THEN** 保存被拒绝并提示选择餐段可售
- **AND** 既有商品不被修改

#### Scenario: The product screen is audited

- **WHEN** 检查菜品管理页
- **THEN** 表格含餐段列且从契约的标签映射渲染
- **AND** 编辑表单含必填的餐段可售下拉，选项从契约的取值集合渲染，提交时带上该字段
