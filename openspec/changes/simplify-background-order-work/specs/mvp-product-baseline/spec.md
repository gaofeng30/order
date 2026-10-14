## ADDED Requirements

### Requirement: Time-derived preparation without idle scans
服务端 MUST 保留取餐前 30 分钟自动进入制作中的业务规则，包括精确边界；MUST NOT 依赖每秒制作扫描。列表、详情、筛选、统计及出餐命令 MUST 采用一致的门槛。用户取消 MUST 独立校验服务器时间。

#### Scenario: Preparation scan is absent
- **GIVEN** 一个已支付预约订单尚未到取餐前 30 分钟
- **WHEN** 时间到达该门槛且没有后台制作任务运行
- **THEN** 后端查询呈现制作中，禁止用户取消，并允许合法商户标记备好

#### Scenario: Refund state is preserved
- **GIVEN** 一个退款中或已退款订单
- **WHEN** 到达制作门槛
- **THEN** 时间规则不得把订单改回制作中

### Requirement: Exception-only recovery scheduling
异常补偿 MUST 仅由持久化未解决异常驱动，以 30 分钟为重试间隔；没有异常 MUST NOT 持续轮询；实例重启或回收 MUST NOT 丢失待办。正常请求、回调与通知首次发送 MUST 不等待重试周期。

#### Scenario: No pending anomaly
- **GIVEN** 没有未解决异常
- **WHEN** 服务空闲
- **THEN** 不重复查库探测异常

#### Scenario: Worker is recycled
- **GIVEN** 已持久保存一个异常待办
- **WHEN** 当前执行实例在唤醒前退出
- **THEN** 待办能够被可靠恢复，且重复唤醒不造成重复资金或通知副作用
