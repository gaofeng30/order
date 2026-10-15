# 支付环境隔离

gate_type: W3；ui_level_target: UI0；ui_level_actual: UI0。此 change 无 UI diff；实际资金/手机 UI3 和发布验收由 root 单独记录，后端本地证据不替代它们。

## ADDED Requirements

### Requirement: 新请求固定环境且可恢复
系统 MUST 为 Cloud uat/production 新支付和退款编号分别使用 OU1P/OU1R 与 OP1P/OP1R，32 ASCII，14 随机字节。既有请求和摘要不改写，纯回放不重新生成编号。

#### Scenario: 支付 Prepare
首次 Prepare 提交完整请求、claim 后调用 provider；provider 返回前无最终 payment.prepare receipt。返回后原公开结果、Created 和 receipt 时序保持。

#### Scenario: 完整账单
每个合法明细保留 ordinal、父关系及原身份；未知归属不静默删除，本环境真丢单保持严格差异。

#### Scenario: 实际系统证明更正
完整 receipt 留存实际消费的 `system_facts`，摘要绑定账单与系统 proof；同一事务复核后保存。系统付款或退款的原请求 proof 更正时产生新 receipt identity，保留旧证据；相同证明回放仍只有一份 receipt。

#### Scenario: 已观察旧支付继续物化
PAYMENT locator 仅从本环境完整 MATCH 观察解析唯一 transaction ID，再以完整身份通过原 proof/受审阅确认检查。不同 ID 冲突或缺完整观察保持不可用；MATCH 本身不成为环境证明。

#### Scenario: 完整安装器关闭旧绕过入口
安装器与 readiness 显式绑定 UID、environment、两个正式回调 URL 四值；新五个入口加入完整 registry，实际实体 113、普通身份可执行 109。旧 payment.prepare、两 refund.request 和 billing.reconcile v1 函数实体及 body 保留，但 PUBLIC/anon/authenticated 的有效执行权限均撤回，普通 client 在 HTTP 前拒绝旧入口。任一四值、实际 UID、函数定义或有效权限漂移均使 readiness 不可用。
