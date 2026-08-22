## ADDED Requirements

### Requirement: The pickup point is a single shared constant

一期为单门店单取餐点（§3.1、§5.5）。取餐点 MUST 是一个常量字符串，两端 MUST 引用同一取值，MUST NOT 在任何页面硬编码第二处取餐点文字。

取餐点 MUST NOT 拆成名称与地址两个字段：客户只提供一个字符串，拆分会迫使写入方各自挑一个字段，从而在不同写入路径上装入不同语义的值。

订单生成时 MUST 固化取餐点快照（§7.2），且该快照 MUST 与配置中的取餐点一致 —— 结算写入的值与种子订单携带的值 MUST NOT 是两种东西。

订单详情 MUST 只展示一次取餐地点。

#### Scenario: A user places an order and opens it

- **WHEN** 用户完成结算并打开订单详情
- **THEN** 订单快照里的取餐点与配置中的取餐点一致
- **AND** 取餐地点在页面上只出现一次

#### Scenario: Pickup point data is audited

- **WHEN** 检查两端的取餐点配置与全部订单快照
- **THEN** 只存在一个取餐点取值
- **AND** 不存在名称与地址分离的二元结构
