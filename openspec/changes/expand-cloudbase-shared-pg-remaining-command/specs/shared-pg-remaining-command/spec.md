# Shared PG 剩余命令规格

Status: APPROVED_ATOMIC / IMPLEMENTED / LOCAL_VERIFIED / UAT_SYNTHETIC_VERIFIED / CANDIDATE_PENDING / INDEPENDENT_NOT_RUN。gate_type=W3；ui_level_target=UI0；ui_level_actual=NOT_RUN。

## ADDED Requirements

### Requirement: 完整业务端口与固定事务
系统 SHALL 保留退款五方法、订阅两方法、生产推进一个方法及原错误/幂等/provider顺序，仅以20固定普通UID RPC替换数据库事务；不得新增通用SQL、Tx回调、业务配置或改变原native接口。

#### Scenario: 已知提交后外调
- **WHEN** 退款claim已提交且需CreateRefund
- **THEN** Go仅按原流程调用provider；提交未知不得盲目重复不可逆调用，原READY/UNKNOWN恢复规则保持。

### Requirement: 退款原子性与消费顺序
系统 SHALL 保留当前权限、locator冻结、原锁序、receipt与状态优先，以及观测、订单/预付款、退款和固定可选通知的原事务范围。

#### Scenario: 所有状态的非零成功时间
- **WHEN** PROCESSING/CLOSED或SUCCESS带非零SuccessTime并到达原observation INSERT
- **THEN** 按原pgx整数微秒编码和PG范围消费；零值为NULL；不能提前到lookup/lease之前或跳过非SUCCESS；失败无原事务部分写。

### Requirement: claim缺失与并发恢复
系统 SHALL 区分claimCreate缺失ErrNotFound、claimOneQuery缺失ErrUnavailable，以及非READY/不eligible/affected!=1的false,nil；保留旧lease/回调胜出和未知ack行为。

#### Scenario: 缺失退款
- **WHEN** 分别调用create claim和query claim但行不存在
- **THEN** 返回各自原错误而非统一claimed=false,nil，公共服务错误处理按原调用链。

### Requirement: 固定通知依赖与时钟
系统 SHALL 用具体Shared订阅依赖的私有启用位保留可选通知，不暴露环境开关或Tx回调；有效intent后、读order/consent前由SQL独立采wall clock，保留原AvailableAt及事务原子性。

#### Scenario: 最早授权已消费
- **WHEN** 最早ACCEPTED consent已消费且没有精确已有outbox
- **THEN** 拒绝并回滚退款原事务，不跳选后续consent；无consent按原成功，无通知依赖不采通知clock。

### Requirement: 订阅和生产推进
系统 SHALL 保留RecordConsent幂等与有限named unique重读、单条SKIP LOCKED通知claim、provider外调后livelease CAS和原临时重试时刻；生产推进保留逐订单事务、30分钟边界、原两次重试及部分计数。

#### Scenario: 过期发送结果
- **WHEN** subscription provider完成时lease已过期或owner/version已变
- **THEN** 原CAS失败不得覆盖新owner结果；不虚构exactly-once送达；生产批处理中途失败不撤回已完成订单。

### Requirement: 编码、权限与容量
系统 SHALL 保留完整uint64/NULL、Go权威摘要与JSON行为、原时间纳秒比较及实际消费时µs编码，固定ordinaryUID/Invoker/RLS/最小列和USAGE权限；仅原健康输出界保留8MiB，不裁剪业务数据。

#### Scenario: 未授权身份
- **WHEN** anon或其他普通UID调用任一新增RPC/读取业务关系
- **THEN** 拒绝或RLS零行；运行身份不能DDL/写ledger/提权；真实current-owner变化仍按业务授权前置拒绝旧receipt。

### Requirement: 验证与发布边界
系统 SHALL 先真实native差分Red→Green/必要Refactor，再完整本地及普通UAT切片、清理和source绑定，最终candidate干净detached独验；缺真实微信材料不冒充真实退款/送达或完整运行通过。

#### Scenario: 形成候选
- **WHEN** Writer Gate和两轴review完成
- **THEN** 仅owned paths形成exact候选并另处独验；未独验不集成，未main不归档，95/25及UI3仍由完整运行验收负责。


### Requirement: 历史 RequestedAt 的原宿主时区
系统 SHALL 在finishCreate刷新receipt时保留原pgx Scan→Go time.Local→json.Marshal的历史时刻offset与失败消费顺序；不得用now offset或硬编码时区代替。

#### Scenario: 历史时刻跨DST
- **WHEN** stored created_at与本次now处于不同时区偏移季节，或callback在材料取得后获胜
- **THEN** 同RPC零写材料阶段取得历史instant，Go计算其Local JSON；最终锁内使用当前业务状态并在原refresh消费点生成receipt，错误回滚，无重复provider调用。

### Requirement: 生产推进保留原时间域与审计编码
系统 SHALL 将原Go截微秒时间用于原决策比较，将原pgx整数编码仅用于持久SQL，并保留Go审计Marshal错误被忽略产生NULL的结果；不得通过timestamptz传参或RFC3339解析新增年份限制。

#### Scenario: 超出JSON年份仍可推进
- **WHEN** 原生year10000推进成功且after JSON编码失败，或year-1原due查询为空成功
- **THEN** Shared结果及持久审计NULL与native一致；同源cutoff先Go Add后编码，raw比较不使用wrapped持久时间。

### Requirement: Preserve historical Local receipt encoding
The implementation SHALL use the original Go host-Local MarshalJSON bytes at refund receipt refresh and SHALL NOT impose PostgreSQL reparsed-instant equality beyond the immutable created_at tagged microsecond guard.

#### Scenario: Historical offset contains seconds
- **WHEN** the original Go Local offset contains seconds omitted by MarshalJSON
- **THEN** fresh and replay results preserve the native receipt encoding and behavior, without introducing a new rejection
