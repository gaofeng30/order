## ADDED Requirements

### Requirement: The order card shows no piece count and aggregates items numerically

用户端订单列表卡片 MUST NOT 展示商品件数，MUST NOT 在页面数据里保留产生它的字段。

对 `items` 的任何数值聚合 MUST 选取数值列 —— 数量、原价或折后价。MUST NOT 选取 id 列或名称列：名称是字符串，累加会得到拼接结果而非数字，且该错误在渲染前不会抛异常。

取餐号徽章 MUST 只展示号码本身并居中，MUST NOT 附加「号」一类的说明标签。

订单卡片上的操作按钮 MUST 单行展示，MUST NOT 因所在行的空间分配而折行或被压缩至文字溢出。

#### Scenario: The order list is rendered

- **WHEN** 用户打开我的订单
- **THEN** 卡片不展示件数
- **AND** 取餐号徽章内只有号码
- **AND** 「取消预约」在一行内完整显示

#### Scenario: Item aggregation is audited

- **WHEN** 检查全部页面中对 `items` 的聚合
- **THEN** 每一处选取的都是数值列
- **AND** 不存在选取名称列的聚合
