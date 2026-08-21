## ADDED Requirements

### Requirement: Order lines carry a product name snapshot

订单项 MUST 形如 `[id, name, qty, price, discountedPrice, flavors?, note?]`，其中 `name` 是下单当刻固化的商品名称快照。

渲染订单的任何路径 MUST NOT 按 `id` 回查商品表取名称。`id` 保留用于退款、对账与销量归集，MUST NOT 作为显示名称的来源。

名称 MUST 与原价、折后价同批固化：§5.6 要求订单固化价格事实，名称属于同一类事实，两者 MUST NOT 采用不同的存储策略。

`name` MUST 位于必填段内（`id` 之后、`qty` 之前），MUST NOT 追加在 `flavors` / `note` 这两个可选尾项之后 —— 否则元组不再有稳定 arity，未填口味的订单会取到错误的列。

#### Scenario: A product is renamed after an order was placed

- **WHEN** 商品在下单后改名或从目录移除
- **THEN** 历史订单仍显示下单当时的名称
- **AND** 渲染过程不发起任何按 id 的商品查询

#### Scenario: The order line shape is audited

- **WHEN** 检查两端订单种子的每一行订单项
- **THEN** 每行至少五项，第二项为非空字符串名称
- **AND** 数量、原价、折后价三项均为整数

### Requirement: A freshly placed order opens on every user surface

结算写出的订单 MUST 能被订单列表与订单详情直接渲染，MUST NOT 抛异常。

结算写出的字段类型 MUST 与种子订单一致：同一字段 MUST NOT 在一处是数字、在另一处是格式化字符串。

商品图片不在订单中固化，因此商品不在本地目录时订单详情 MUST 回落占位图，MUST NOT 因此报错。

#### Scenario: A user places an order and opens it

- **WHEN** 用户完成结算后打开「我的订单」与订单详情
- **THEN** 两个页面都正常渲染，列出商品名称与数量
- **AND** 两个页面都不抛异常

#### Scenario: Order totals keep one type

- **WHEN** 比较结算新建的订单与种子订单的金额字段
- **THEN** 两者类型相同
- **AND** 不存在一处为字符串、一处为数字的字段
