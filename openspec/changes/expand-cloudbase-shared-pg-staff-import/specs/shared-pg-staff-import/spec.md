## ADDED Requirements

### Requirement: 员工管理完整迁移且保留原公开合同
系统 SHALL 通过固定普通服务JWT RPC实现List、DiscountRate、五kind Execute，保持原Go DTO、mask、错误和版本规则。

#### Scenario: 查询及数值边界
- **WHEN** 员工搜索包含LIKE模式，或消费金额/笔数超出原Go扫描范围
- **THEN** 保持原模式搜索、id降序最多5000和当前绑定关系/STAFF非REFUNDED聚合；超界按原错误关闭，不截断或转float

#### Scenario: 空手机号与命令优先级
- **WHEN** UPDATE的Phone为空，或目标不存在且Name无效
- **THEN** 先当前OWNER、settings及receipt，再锁目标读取，空phone复用当时existing，目标失败保持原NotFound；不提前因空phone或name拒绝
- **AND** SET_ENABLED nil指针保持原先于目标读取的InvalidInput，未使用字段仍参与原完整摘要

#### Scenario: 非业务目标字段仍决定审计目标
- **WHEN** 任一种Staff命令携带原StaffID，尤其CREATE_STAFF或SET_DISCOUNT_RATE
- **THEN** p_staff始终传原uint64值包括0；0写DISCOUNT_SETTINGS/1，非零写STAFF/原StaffID
- **AND** 不改用新建员工ID，不把此字段置NULL，原完整cmd摘要和重放冲突保持

### Requirement: 导入预览保持原解析与幂等顺序
系统 SHALL 完整支持商品和员工XLSX预览，复用原parser/plan规则、15分钟有效期与receipt，不依赖新增对象存储。

#### Scenario: 重复预览及行错误
- **WHEN** 同key再次Preview或文件含合法可报告的行错误
- **THEN** 仍先parse/buildPlan/entropy/clock，再授权与插入；不得提前重放跳过原步骤，行错误按原响应保留
- **AND** duplicate receipt先回滚新batch再恢复原token/响应，晚约束错误不被旧receipt吞掉

### Requirement: 导入提交全批原子与原时间语义
系统 SHALL 单RPC事务完成原集合锁、全部有效行、版本、batch状态/result和receipt，Go材料阶段不得模拟跨HTTP事务。

#### Scenario: 已提交与到期
- **WHEN** batch已COMMITTED或首调用等待行锁跨过到期
- **THEN** currentOWNER与actor/token范围保持，已提交按原操作键receipt优先；首调用拿锁后才观察expiry，不能用发请求前时间放行
- **AND** 首锁observed_at仅用于前置判断；最终committed_at仍为clock_timestamp并受原committed_at<=expires_at约束，慢材料跨期失败保持原映射与全批回滚，下一新请求重新观察

#### Scenario: 故障及并发
- **WHEN** 同token竞争、配置竞争或最后batch/receipt写失败
- **THEN** 保持原锁序、冲突和全批回滚；员工UPDATE不改enabled、名单版本按原一次增加，商品分类/产品sort规则不变
- **AND** 员工成功提交清除preview_json、商品保留；未知提交按原同key恢复，不新增token恢复接口或SOURCE_CHANGED循环

### Requirement: 最小权限固定入口与完整验收
系统 SHALL 仅增加六个固定INVOKER入口，总registry55，以固定服务UID RLS/精确权限及权威readiness核验，runtime无管理凭据或DDL。

#### Scenario: 普通身份与漂移
- **WHEN** anon/其他UID调用或出现额外relation/column/sequence/SECDEF权限
- **THEN** 数据与函数访问被拒或readiness失败关闭；原44ledger/checksum和旧函数body保持，管理安装历史不可强制改写

#### Scenario: 切片完成边界
- **WHEN** 本地与普通UAT矩阵完成
- **THEN** 仍须exact candidate clean detached独验及依赖满足才可集成；不得称CloudRun整体部署、真实微信/COS或95cases/25pages/UI3完成

## Verification status

IMPLEMENTED / WRITER_V4_GREEN / PRECOMMIT_SNAPSHOT_FROZEN / CANDIDATE_NOT_CREATED。最终source-v4全16包ordinary/serialized race各1679PASS/0FAIL/SKIP；三Import与Staff decoder容量P2已完成原生反例和局部独立普通/race复核。平台证据分为V2完整54labels/189responses与V4当前容量/恢复subset，不混称同一版本全矩阵。最终两轴/候选、exact-SHA独验、main集成仍待；完整WeChat/COS/CloudRun/95cases/25pages/UI3未通过。本快照冻结后candidate及独验状态仅外部receipt更新，详见tasks。

### Requirement: 取消边界不得误报业务结果
系统 SHALL 对有效caller context保留原业务错误，对已取消context的HTTP transport失败保持failclosed且不隐式重试。

#### Scenario: 目标行锁等待时取消
- **WHEN** caller context已取消且HTTP无法得知数据库执行阶段
- **THEN** 允许Unavailable而不复刻native target Scan误映NotFound；不得泛化所有Update timeout为NotFound
- **AND** 此为明确内部差异，须单独证明受控取消无副作用；不能据此承诺所有断连都未提交，未知提交仍按原同key恢复

### Requirement: 保留商品导入原键编码
系统 SHALL 保持原name_key字符串参数进入PG bytea的晚输入语义，不以传输适配改变原业务成功或失败集合。

#### Scenario: 商品导入name_key晚输入语义
- **WHEN** Product/category key含反斜杠或Unicode
- **THEN** 保留原Go规范化及PG bytea文本输入的晚消费结果，包括原失败/回滚，不用raw bytea直写改变成功集合
- **AND** 内层材料与六ABI不因此新增公共业务限制

### Requirement: 管理增量必须以完整源码为权威
系统 SHALL 在管理请求大小受限时保留原历史并以已批准旧源码和最终编译源码双重hash约束有界增量，不让runtime获得管理能力。

#### Scenario: readiness固定替换与未知源码
- **WHEN** 管理DO重建readiness或发现当前prosrc不是批准旧版/精确新版
- **THEN** 每处固定替换验证原出现次数和最终hash，同事务应用新函数及精确ACL；未知源码拒绝且无写，失败完整回滚
- **AND** 最终全函数body/属性/权限由独立readback和普通Check确认，不能仅凭DO完成或preview成功放行

### Requirement: 固定RPC传输不得缩小原合法数据集合
系统 SHALL 保持原行数/字段/文件/业务规则，按design明确24个成功响应可增长入口移除HTTP200人工8MiB截断；其余31、signin与非200维持原限制，不对未知或未来RPC自动扩展。

#### Scenario: 合法聚合、材料与重放超过通用cap
- **WHEN** 已审24名单内Staff/目录/报价聚合或Import facts、NEED_MATERIAL、REPLAY成功响应超过8MiB
- **THEN** 客户端不以新增字节cap、姓名限制、裁剪或公共分页改变原结果；严格JSON、ctx和身份规则仍生效
- **AND** 必须验证24/31逐名策略和三Import实际Client/publicHTTP Red→Green，并完成目标CloudBase大facts定点验收；不得用本地通过替代平台，也不得将其他源码推导项说成已逐一极值云测

### Requirement: 分步迁移登记必须反映真实执行
系统 SHALL 仅登记实际成功且全量核验过的迁移，未知结果先查证并保持旧历史不变。

#### Scenario: 直接管理事务与历史登记分步失败
- **WHEN** 官方批量迁移受体积限制，已采用ExecutePGSql执行版本化事务再Repair登记
- **THEN** 只有实际执行且完整源码/权限核验通过后才能登记applied，分别保存实际执行Query与官方recorded Query；旧历史相对fresh pre-install官方baseline逐字不变
- **AND** 执行或登记响应未知必须先读库与history判态；APPLIED_UNREGISTERED不能算部署/候选完成，不得因缺history盲重放SQL

#### Scenario: 官方记录与提交原文存在空白差异
- **WHEN** recorded Query与executed/submitted Query不完全相同
- **THEN** 分别记录hash并以官方转换规则或精确差异加官方PG词法证明语义不变，核实SQL字面量/函数body/注释边界，不能以去全部空白相同当成语义或字节等价
- **AND** 未核验的登记不能计部署完成，不覆盖旧历史或盲重执行

### Requirement: 业务响应解码保持既有容量合同
系统 SHALL 在Staff命令结果解码中保留原合法响应集合，不以传输层之外的人工上限再次改变结果。

#### Scenario: 小命令返回已有大合法字段
- **WHEN** 合法Staff导入已持久化长Name，原公开SET_STAFF_ENABLED以小请求返回该员工或重放其receipt
- **THEN** Staff response codec不得用新增8MiB上限改变native成功结果，before摘要小界与严格JSON/digest仍生效
- **AND** 验证须区分已提交但响应解码失败与无写失败，并证明同key恢复不重复写；不能以公开入参16KiB推导已有持久响应上限
