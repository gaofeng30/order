## ADDED Requirements
### Requirement: 报价信息保持同一权威快照
小程序 SHALL 在报价前显示预估金额，在报价成功后从同一 Quote 呈现商品行、原价小计、折扣和应付。分单位 SHALL 转为两位小数的元显示。
#### Scenario: 员工折扣与变价
- **WHEN** Quote 返回原价 42 分、折扣 8 分、应付 34 分
- **THEN** 分别显示 0.42、0.08、0.34 元，商品行不继续显示旧购物车单价
#### Scenario: 不可售选择
- **WHEN** Quote 返回 QUOTE_SELECTION_UNAVAILABLE
- **THEN** 保留选择与表单，持续显示原因和返回菜单入口，不生成预支付
### Requirement: 修改选择保留确认表单
小程序 SHALL 保留原确认页后打开菜单修改选择；返回 SHALL 刷新购物车和取餐选择而保留联系人与备注。当前交易快照锁定时 SHALL 拒绝编辑。
#### Scenario: 返回菜单再确认
- **WHEN** 不可售错误后进入菜单重新选择并确认
- **THEN** 返回原确认页，联系人与两级备注仍在，新选择重新显示预估，无旧Quote
#### Scenario: 已有预支付
- **WHEN** 用户在pending或结果未知状态尝试编辑
- **THEN** 不修改既有交易快照，后续确认仍使用同一个prepayment
