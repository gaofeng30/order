## ADDED Requirements

### Requirement: PC stores daily sell-out apart from shelf state

PC 后台 MUST 使用与小程序相同的模型：`status` 只表达上下架，当日售罄存为 `(productId, serviceDate)` 记录。两端 MUST NOT 各自定义商品销售状态的形状。

PC 的售罄开关（含批量）MUST 写入当前营业日的记录；上下架开关 MUST 只改 `status`。同一次操作 MUST NOT 同时改动两个维度。

#### Scenario: PC product states are audited

- **WHEN** 检查 PC 商品种子与菜品页的读写点
- **THEN** `status` 只出现 `'on'` 与 `'off'`
- **AND** 售罄开关不修改 `status`，上下架开关不写售罄记录

### Requirement: Rebuilding a paid order checks sell-out by pickup date

§7.3 的补建订单校验 MUST 按该笔支付的**取餐日期**判断商品是否售罄，MUST NOT 按当前时刻的全局商品状态判断。

#### Scenario: A pending payment is rebuilt

- **WHEN** 主账号补建一笔「有支付无订单」的条目
- **THEN** 售罄校验针对该条目的取餐日期
- **AND** 其他日期的售罄记录不参与该判断
