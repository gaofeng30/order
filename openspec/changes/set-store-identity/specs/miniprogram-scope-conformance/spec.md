## ADDED Requirements

### Requirement: Store identity has one source and order pickup comes from the order

一期为单门店（§3.1）。门店标识 MUST 只有名称与地址两个字段，MUST NOT 保留分店字段 —— 一个恒等于门店名的字段只会让写入方再挑一次，重演字段分裂。

任何页面展示**订单的**取餐地点时 MUST 取自该订单的 `pickupPoint` 快照，MUST NOT 取自门店当前配置。§7.2 要求生成订单时固化取餐点；读配置意味着配置一改历史订单跟着改，与快照的意义相反。

已确认的门店名称与地址 MUST NOT 与任何页面上的字面量冲突。

#### Scenario: A merchant opens an order

- **WHEN** 商户在小程序商户端查看订单详情
- **THEN** 取餐地点取自该订单的快照
- **AND** 与用户端订单详情展示的是同一个值

#### Scenario: Store identity is audited

- **WHEN** 检查两端的门店配置与全部页面
- **THEN** 不存在分店字段
- **AND** 不存在与已确认值冲突的门店名或地址字面量
