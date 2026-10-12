# Shared PG payment requirements

验收状态：V3完整本地与限定普通UAT合成provider切片已通过，作为历史证据保留；V4仅15处SQL尾空白修正，三个受影响包和重新安装的70源/readiness核验已通过。候选/独验/main待完成。正式要求不因局部平台结果降低，真实微信/完整runtime/UI3另验。

## ADDED Requirements

### Requirement: 完整支付公共契约
实现 SHALL 保留Prepare、Confirm、IngestPayment、RunDue、ListPending、MaterializePending六方法及原handler/DTO/错误/输入优先级，完整迁移而不启用部分runtime。

#### Scenario: 六方法行为差分
- **WHEN** 当同一合法v44事实经native与共享应用执行时
- **THEN** 结果、持久事实、原错误与provider次数必须一致；registry缺失Red不可替代业务Red。

### Requirement: 原子Prepare与外部副作用
Prepare SHALL 按原receipt/replay/openid/Go lease熵/now顺序，再在单RPC内完成完整Finalize锁/窗口/SQL支付号熵/canonical request/claim。

#### Scenario: 未知提交恢复
- **WHEN** 当claim提交响应丢失时
- **THEN** 不调用CreateJSAPI；再次原入口按持久事实恢复。provider仅在已知有效claim提交后调用，finish与独立receipt保持原范围。

### Requirement: 观测与租约归并
系统 SHALL 保留原Normalize逐字段优先级、全纳秒digest/截止判断、锁后lease时钟及原PAID/CLOSED归并，不增加UTF8或trace业务限制。

#### Scenario: 晚返回竞争
- **WHEN** 当旧owner/version或过期lease的Query/Close晚返回时
- **THEN** 不覆盖当前winner；16回调/close竞争和持久事实须有真实差分。

### Requirement: 订单生成事务
Materialize SHALL 同事务验证冻结quote、生成日序号/order/items、推进prepay及observation；原坏事实DEFERRED/PENDING_MANUAL持久分支不改为一概回滚。

#### Scenario: 手工权限与末尾故障
- **WHEN** 当手工路径选中观测时按原顺序锁当前OWNER；末个item/状态故障回滚整个业务事务
- **THEN** 独立receipt失败不虚构已提交订单回滚。

### Requirement: 固定权限与错误协议
运行身份 SHALL 仅普通服务UID，70固定INVOKER函数及最小RLS/列/sequence权限，不持DDL credential。SQL_FAILURE仅采用design闭合三字段及原命名约束分类。

#### Scenario: 未知约束与权限漂移
- **WHEN** 当未知23505或额外ACL/SECDEF/role权限出现时fail closed
- **THEN** 不新增重试或伪装业务Forbidden；普通其他UID/anon不能调用新业务RPC。

### Requirement: 容量与精确编码
新增15RPC SHALL 按原有界成功输出保留8MiB，旧24增长接口不变；uint64/nanosecond/bytea/canonical JSON与receipt不得损失精度或增业务限制。

#### Scenario: 最大合法事实
- **WHEN** 当最大合法canonical字段、100个ID/pending、uint64max或HTML/Unicode字节出现时原成功仍成功；不得把完整无界quote聚合加入返回wire。
- **THEN** 当最大合法canonical字段、100个ID/pending、uint64max或HTML/Unicode字节出现时原成功仍成功；不得把完整无界quote聚合加入返回wire。

### Requirement: 真实验收与生命周期
本切片 SHALL 完成native RGR、完整受影响本地回归、普通UAT合成provider矩阵/清理与exact candidate独验，分清平台能力和真实微信支付。

#### Scenario: 候选与外部阻断
- **WHEN** 当本地或合成provider通过而真实微信商户材料/CloudRun/UI3未完成时
- **THEN** 只能报告支付切片对应层通过，不能标记完整UAT或main集成。

### Requirement: 保留quote typed重试边界
共享实现 SHALL 仅将原quote.ErrUnavailable对应的确定性失败映射严格单字段QUOTE_UNAVAILABLE_RETRY并保留原最多两次尝试；不得扩大到snapshot损坏、权限、未知提交或晚CAS。

#### Scenario: Prepare与Materialize快照损坏分支不同
- **WHEN** 原冻结报价shape或digest损坏
- **THEN** Prepare返回不可用且不按quote临时错误重试；Materialize按原分支持久QUOTE_SNAPSHOT_INVALID的defer并返回原zero result/false/nil，不将该持久分支改成回滚。


### Requirement: 原持久JSON解码与受信时间材料
固定实现 SHALL 保留canonicalProviderCreateRequest的DisallowUnknownFields、Go字段折叠、null no-op、顺序覆盖与错误类型不可被后值掩盖的语义；锁内digest校验不得省略。Go时间材料来自同一原time对象，SQL不新增独立Gregorian算法或年份业务限制，且仍严格检查闭合wire成员类型。

#### Scenario: 字段别名与错误优先
- **WHEN** 持久JSONB含AppID或ſ别名、null、多个alias、错误类型或未知字段
- **THEN** 按native扫描JSONB文本的原Go规则得到相同canonical request与错误，未知字段/错误类型不得被后续合法alias吞掉。

#### Scenario: 非PAID未消费超范围时间
- **WHEN** ACTIVE_QUERY NOTPAY携带year1000000 SuccessTime，仅参与原digest而不写success_time
- **THEN** 不因提前PG时间转换拒绝；PAID按原pgx整数编码后被PG实际拒绝时才保留原Unavailable及原子回滚，不提前拒绝原可回绕成功的时刻。SQL不能把数字配置字段当string隐式接收。

### Requirement: 原始时刻与持久整数编码分离
系统 SHALL 在PAID持久消费阶段保留原pgx整数微秒、signed64回绕及Infinity编码；原full-ns Normalize和摘要继续使用原Go时刻，不以持久回绕值替代。

#### Scenario: 远年精度与后续扫描
- **WHEN** 原生pgx接受远年时间并存为精确或回绕后的PG时间，或编码成Infinity
- **THEN** 共享写入保留相同值和原raw截止分类；后续原*time.Time扫描不接受Infinity时，共享在同扫描阶段失败且不新增订单、defer或receipt写入，公共worker错误与计数保持原编排。

### Requirement: 格式修正后的精确源码验收
V4 SHALL 保留V3全量回归/平台业务证据的历史身份，并绑定当前源码、受影响回归和平台函数定义；不得把语义等价当字节相同或将未重跑测试标为V4执行。

#### Scenario: SQL尾空白导致候选阻断
- **WHEN** owned-only staged diff发现新SQL行尾空白
- **THEN** 修复并核验内层literal/comment/token等价，刷新prosrc/readiness/hash、受影响三包ordinary/race和实际平台源核验；新exact candidate独验仍全17包ordinary/race，候选前staged检查必须通过。
