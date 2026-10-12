## ADDED Requirements

### Requirement: 保留 Order Mini 会话与手机号合同
系统 SHALL 经固定普通JWT RPC实现原SessionStore三方法和PhoneStore两方法，保持现有Service、微信provider与公共token/DTO。

#### Scenario: 建立会话与单token退出
- **WHEN** Go完成一次微信exchange及安全随机token生成
- **THEN** 用户upsert与hash-only session同事务，issued包含边界且expiry不包含边界；碰撞或失败不重调provider，不额外mint
- **AND** logout只撤所给hash，零行成功，OpenID原始bytes不被新增text限制拒绝

#### Scenario: 不可变手机号与并发恢复
- **WHEN** 已绑定、同用户并发绑定、跨用户争用或provider拒绝code
- **THEN** 保留原先读后provider、已有绑定绕过provider、拒绝后重读当前绑定、named phone unique错误及不可变phone规则

### Requirement: 商户登录与用户补充身份原子证据
系统 SHALL 复用原Service的Start/provider/Complete或Recover阶段，保留当前授权、审计与receipt各自字节合同。

#### Scenario: 登录业务拒绝需要持久审计
- **WHEN** 当前账号不可用或phone冲突导致原生业务拒绝
- **THEN** 提交相同LEGACY拒绝audit后返回原业务error，不能用RAISE将审计回滚，也不能留下部分绑定

#### Scenario: 被拒绝code的并发成功恢复
- **WHEN** provider拒绝code
- **THEN** 仅同code、当前账号authVersion及开始时间之后的合格已提交成功证据可恢复；ALREADY_BOUND或任意已有绑定不足以恢复

#### Scenario: Extra phone重放
- **WHEN** 同key重放或当前extra已被后续命令修改
- **THEN** 保留原canonical Go请求digest、USER scope/rawkey hash和严格receipt解析；仅当前extra匹配才返回原pricing快照，否则原冲突

### Requirement: 账号管理保持当前 OWNER 与最后管理员约束
系统 SHALL 以单固定枚举命令实现原四账号动作，同事务保持集合锁、当前actor、receipt和last-owner不变量。

#### Scenario: 并发管理及旧receipt
- **WHEN** 修改、禁用、删除或同key重放
- **THEN** 按ID锁全部undeleted账号，先核当前enabled OWNER再receipt，再新输入校验；最后enabled OWNER不能禁删降级
- **AND** 同值新key仍增version，softdelete不释放phone，旧receipt不被新clock故障阻断

#### Scenario: 命令未使用字段及不可编码输入
- **WHEN** 原命令未使用的Name/Phone/Role含特殊bytes，或实际使用字段无法编码为PG文本
- **THEN** 未使用字段不新增拒绝且完整原命令仍参与digest；使用字段按原InvalidInput或Unavailable区别在当前授权/receipt及原命令优先级之后判断，不因JSON替换而成功写入

### Requirement: PC 登录保留一次消费与错误优先级
系统 SHALL 保留两分钟intent、十二小时access、三阶段批准和当前owner版本认证，秘密仅Go生成且SQL只存hash。

#### Scenario: WAITING或过期不受熵故障阻断
- **WHEN** Poll看到WAITING、已消费或expired
- **THEN** 先返回原WAITING/Expired结果，不能提前生成entropy或以熵失败覆盖状态结果

#### Scenario: APPROVED并发消费
- **WHEN** 预读APPROVED后Go准备材料期间发生消费或到期
- **THEN** 同固定RPC锁内重检状态优先；entropy失败亦需重检，只在仍APPROVED时Unavailable；只有一个调用获得raw token
- **AND** 不新增Poll当前账号检查；Authenticate仍逐请求核当前OWNER绑定/authVersion，批准后段仍核当前OWNER和expiry

#### Scenario: 行锁等待与熵生成跨到期
- **WHEN** 首次Poll等待intent行锁或Approve后段等待OWNER行锁跨过到期时刻
- **THEN** 在取得相应锁后采数据库时间并按原优先级拒绝到期请求；Poll首调用传NULL并返回observed_at
- **AND** 同一次Poll取熵后的第二调用冻结observed_at，保持原slow entropy跨到期可成功合同，仍重检当前状态且不恢复已消费秘密

#### Scenario: 消费已提交但响应丢失
- **WHEN** PC消费提交后响应未知
- **THEN** 保留原未知/后续已消费行为，不添加rawtoken持久化、receipt恢复或自动二次消费

### Requirement: Bootstrap 仅管理面执行原精确初始合同
系统 SHALL 使用固定管理RepeatableRead事务保留原五输入、三组状态及默认值，不授持续runtime配置初始化权限。

#### Scenario: 初始、重复与漂移
- **WHEN** storefront id1、discount id1及OWNER集合均无，或已有完整状态
- **THEN** 仅三组均无时原子created，仅逐字段exact时unchanged，否则conflict；不要求其他业务表或SUBACCOUNT为空

#### Scenario: 管理初始化并发或故障
- **WHEN** 多个同输入bootstrap竞争或末步骤故障
- **THEN** 保留原有界事务重试及精确named unique映射，结果收敛且失败全部回滚；普通服务身份不能调用bootstrap或写初始化配置

### Requirement: 固定权限、分层验证与依赖
系统 SHALL 仅注册18个新增固定SECURITY INVOKER业务RPC并保留前12签名，核最小列/sequence ACL、固定UID RLS及额外关系/SECDEF漂移。

#### Scenario: 运行权限与真实业务验证
- **WHEN** 普通服务UID、其他UID和anon调用
- **THEN** 只有固定服务UID可执行受审业务能力；runtime无管理凭据、任意SQL、bootstrap、DDL或配置写权限，新增用户/账号/PC/Mini会话INSERT仅原生所需列，readiness精确核有效INSERT列集并拒绝额外列授权；真实UAT正负例与清理均有回执

#### Scenario: 切片验收与外部材料
- **WHEN** 本地差分、handler和平台业务矩阵通过
- **THEN** 只报告本切片层次，真实微信未齐保持外部阻断；候选/干净detached独验/依赖main集成各自核验，完整runtime与95Case/25页面和UI3不自动完成
