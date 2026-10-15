## ADDED Requirements

### Requirement: Go核销命令保留当前业务契约

系统 SHALL 在共享PG上实现现有REDEEM_ORDER公共行为，并保留当前商户授权、token完整性、订单状态、审计和幂等语义；不得把部分runtime视为完整可上线后端。

#### Scenario: 当前商户核销待取餐订单
- **GIVEN** 已验Order身份绑定启用的OWNER或SUBACCOUNT且订单满足原READY_FOR_PICKUP不变量
- **WHEN** 执行合法REDEEM_ORDER
- **THEN** 订单变为COMPLETED、版本恰加1、completed_at与redeemed_at相同、核销账号来自事务内当前账号
- **AND** ciphertext/key_version清空而token_hash/issued_at保留，成功audit receipt与订单同提交

#### Scenario: 当前密文损坏或预读后变更
- **GIVEN** Go无法验证cipher/hash或RPC锁内完整proof不再等于预读proof
- **WHEN** 执行核销
- **THEN** 不提交业务变更；仅明确的proof竞态允许有限重新读取和验证，不得跳过完整性校验

#### Scenario: 回执预读后账户scope改变
- **GIVEN** Go预读发现旧scope回执并跳过解密，随后绑定账号改变
- **WHEN** 写RPC锁内在新scope未找到回执
- **THEN** 未验证proof不得用于更新，只能返回内部重新验证要求并有限重读

#### Scenario: 当前授权失效后重放
- **GIVEN** 旧key有成功receipt但当前账号已禁用或解绑
- **WHEN** 原请求重试
- **THEN** 按既有授权错误拒绝，不因旧receipt返回成功

#### Scenario: 已完成订单的新命令
- **GIVEN** 订单已完成且请求key无匹配receipt
- **WHEN** 执行REDEEM_ORDER
- **THEN** 返回既有TransitionNotAllowed，不移入扫码或取餐码命令的特殊重放规则

### Requirement: 核销与回执由一次原子RPC完成

系统 SHALL 在一个数据库事务中锁定当前商户授权与订单、检查幂等回执、更新状态并追加回执；预读仅用于无副作用的token验证，不得模拟跨HTTP请求事务。

#### Scenario: 回执写入中途失败
- **GIVEN** 订单状态更新后出现受控的receipt写入失败
- **WHEN** RPC结束
- **THEN** 订单状态/版本/核销字段及receipt全部回滚，测试故障开关不进入公开业务函数

#### Scenario: 十六个相同key并发
- **GIVEN** 同一合法请求与相同scope/action/key
- **WHEN** 16个请求并发核销
- **THEN** 只发生一次订单变更及一条receipt，其他请求返回同一结果的重放

#### Scenario: 十六个不同key竞争
- **GIVEN** 同一待取餐订单和16个不同合法key
- **WHEN** 并发核销
- **THEN** 最多一次成功状态变更，其余按既有状态规则拒绝，不产生重复完成或成功回执

#### Scenario: 提交成功后响应丢失
- **GIVEN** 数据库已提交成功而客户端未收到响应
- **WHEN** 使用原key与原请求重试且当前授权仍有效
- **THEN** 返回原结果且Replay=true，订单版本和receipt数量不再增加，cipher已清空不阻断重放

#### Scenario: 同key不同请求
- **GIVEN** 既有receipt的Go规范请求摘要与当前order_id不同
- **WHEN** 使用相同scope/action/key提交
- **THEN** 返回既有IdempotencyConflict，不重写或覆盖原receipt

### Requirement: HTTP传输保留类型与错误契约

系统 SHALL 仅通过固定命名REST RPC及JSON named参数传值，禁止运行面exec-pgsql、任意SQL和管理员fallback，精确保留整数、时间、JSON与字节，区分业务错误、SQLSTATE、授权错误和未知提交结果。

#### Scenario: 参数包含SQL特殊字符
- **WHEN** 参数包含引号、分号或注入形式文本
- **THEN** 其仅作为值处理，不能改变SQL/RPC目标及权限

#### Scenario: 大整数与时间边界
- **WHEN** 传输2^53两侧整数、2^63+1、NUMERIC(20,0)允许的2^64-1最大ID、分数与超范围输入、微秒UTC时间和JSON null
- **THEN** 合法值无float64舍入、截断或时区变化，非法或越界值明确拒绝，公开DTO契约不改变

#### Scenario: 数据库错误与重试
- **WHEN** 返回40P01、40001或55P03以及非重试业务或权限错误
- **THEN** 仅按原有有限事务尝试处理可重试错误，其他错误保持映射，不因HTTP200或统一500误报成功

### Requirement: 运行身份与数据库函数隔离

系统 SHALL 从可信平台上下文核验运行身份并在事务内核验当前Order商户；业务端不持DDL管理员凭据，PC和Mini不得访问核销proof、审计表或交易写接口。

#### Scenario: 非运行角色访问
- **WHEN** anon、非专用服务UID的authenticated或未知角色经真实网关调用核销RPC或直接读取/修改受保护业务表
- **THEN** RPC请求被拒绝、表读取无业务行且写入无业务变更，不能只以REVOKE EXECUTE或管理Role模拟作为完整证据

#### Scenario: 请求伪造业务身份或角色
- **WHEN** 客户端传入actor/account/role/response字段企图覆盖服务端身份与审计
- **THEN** 现有Order Bearer身份及事务内账号决定授权和审计，伪造字段不被采信

#### Scenario: 专用普通服务身份调用
- **GIVEN** 后端以普通专用CloudBase用户登录并取得有效JWT
- **WHEN** 经REST RPC访问proof、核销或readiness
- **THEN** 可信authenticated角色和固定auth.uid共同授权，INVOKER及service UID专属RLS只放行必要表/列DML，原Order Bearer与商户授权保持
- **AND** 专用UID允许必要直接表DML但没有DDL/owner/BYPASSRLS/角色提升能力，普通用户UID不能借相同authenticated角色访问业务数据

#### Scenario: 运行身份过期或被撤销
- **WHEN** JWT临近过期、重新登录失败或管理面撤销固定服务UID授权
- **THEN** 后端只以singleflight重新signin续期并缓存access token，不保存refresh token；失败fail closed，旧UID不得通过RLS，不回退APIKey或exec-pgsql

#### Scenario: 数据库入口与COS凭据边界
- **GIVEN** PG入口已切换普通服务JWT而现有COS Provider仍需broad APIKey
- **WHEN** 判断完整runtime是否可启用
- **THEN** 必须保持禁止启用，直到独立私有存储最小权限身份与合同等价change完成；本切片不接runtime可单独验证数据库命令，不声称进程管理能力已消除

### Requirement: 管理安装保留项目迁移事实

系统 SHALL 使用受控版本化管理面安装并保留原44SQL字节及Order ledger；RPC签名/定义有独立指纹，应用readiness核验实际schema而非仅HTTP连通。

#### Scenario: 空库安装与重复执行
- **GIVEN** 在安装互斥内核实目标为空或已精确完成本版本
- **WHEN** 安装或重复执行受控manifest
- **THEN** 首次得到44个正确name/checksum/dirty记录及目标RPC定义，重复执行不产生额外业务/ledger副作用

#### Scenario: 并发安装或中途失败
- **WHEN** 两个安装者竞争或迁移中途故障
- **THEN** 同key安装锁串行，不交错破坏history；单事务空库安装中途失败时全部44 SQL及ledger回滚，不留下成功prefix，不能逐HTTP模拟session锁

#### Scenario: 未知历史或checksum漂移
- **WHEN** 初始空库路径发现未批准业务对象，或任一路径发现partial/dirty history或原SQL/hash冲突
- **THEN** 首切片空库安装入口fail closed，不直接迁移未知数据、不删除或改写原44权威文件

#### Scenario: 完整v44之后存在额外未使用对象
- **GIVEN** 原44 ledger和全部相关结构精确匹配，另有独立RPC或无关表/函数
- **WHEN** 重复运行v44安装检查
- **THEN** 沿用native readiness允许无关对象并只读幂等返回，不把相关对象或依赖漂移误归为无关

#### Scenario: 运行定义漂移
- **WHEN** ledger、约束、RPC签名/定义或service UID/RLS/ACL有效权限与候选工件不符
- **THEN** 受限readiness RPC返回actual metadata供Go与候选对照；对照失败则readiness失败，平台HTTP200不能覆盖漂移

### Requirement: 验收证据按实际层级交付

系统 SHALL 分别记录本地行为、真实共享PG平台、真实核销命令、候选及独立验证；本change不代表全量UAT或Mini UI3。

#### Scenario: 仅能力探针通过
- **GIVEN** 隔离fixture RPC通过但真实v44核销/角色/恢复仍未完成
- **WHEN** 报告进度
- **THEN** 只标平台能力范围通过，实际未运行项保留NOT_RUN或BLOCKED_EXTERNAL

#### Scenario: 源或部署定义变化
- **WHEN** candidate、实现、规格、schema、RPC定义或权限配置改变
- **THEN** 对应旧验证失效，新候选在干净detached worktree重验，不沿用旧SHA或消融模型PASS
