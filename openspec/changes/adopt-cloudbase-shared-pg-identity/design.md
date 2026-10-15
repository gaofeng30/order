# 身份闭合设计

## 权威与实现顺序

依据当前native identity、merchantidentity与order-bootstrap源码，外部 shared-pg-identity-port-map.md（SHA256 432c004d00bf62544f7b7efa8a426c3719418ee135b470bd95ce851beeb95dd9）为已读端口索引，不是测试回执。顺序唯一：Session/Phone→merchant Store→账号/PC→管理bootstrap→统一registry/真实平台/候选。保留现有Service/handler/provider，实现其存储端口；无未支持方法stub或native fallback。

所有RPC固定named参数。numeric正整数为完整uint64、传十进制string；BYTEA通过hex保留OpenID/NUL/非UTF8，hash为hex64，NULL不得与空串互换。Go继续canonicalization、NFKC/nameKey、JSON Marshal及hash，可信服务参数不得冒充外部客户端可直接调用。平台固定authenticated+service UID，业务actor由原Order认证提供。SQL回业务拒绝envelope与抛出storage异常分开：需要保留拒绝audit时必须正常提交再由Go映业务error，不能RAISE回滚已承诺拒绝证据。

## 固定运行 ABI

下列新函数统一public、SECURITY INVOKER、固定UTC/search_path。时间为timestamptz，hex及digest为text；outcome为固定业务enum，raw facts由Go保留原校验/投影，不用统一新错误模型替代原error。

| 函数（均以order_开头、_v1结尾） | named参数 | 结果职责 |
|---|---|---|
| identity_session_create | p_openid_hex text,p_token_hash text,p_issued_at timestamptz,p_expires_at timestamptz | 原子upsert用户+插session，成功确认 |
| identity_session_find | p_token_hash text,p_now timestamptz | user_id string或not_found |
| identity_session_revoke | p_token_hash text | 仅删除所给hash，零行成功 |
| identity_phone_read | p_user numeric | openid_hex、primary_phone_hex/null |
| identity_phone_bind | p_user numeric,p_phone_hex text,p_bound_at timestamptz | 锁user、不可变绑定/精确named phone unique映射 |
| merchant_login_start | p_user numeric,p_code_hash text,p_request_hash text,p_at timestamptz | already_bound/minimal identity或openid_hex；拒绝audit提交 |
| merchant_login_complete | p_user numeric,p_phone_hex text,p_code_hash text,p_request_hash text,p_at timestamptz | 当前绑定/phone/audit原子结果 |
| merchant_login_recover | p_user numeric,p_code_hash text,p_request_hash text,p_started_at timestamptz,p_at timestamptz | 只依据同code及当前authVersion的合格已提交证据恢复 |
| identity_extra_phone_set | p_user numeric,p_phone_hex text,p_name text,p_name_key_hex text,p_operation_hash text,p_request_hash text,p_request_digest text | 原子extra/pricing/严格receipt重放 |
| merchant_account_current | p_user numeric | 原OWNER account事实/NotFound |
| merchant_account_list | p_user numeric,p_query text | 原owner检查及未删除账户列表，不加分页 |
| merchant_account_command | p_actor numeric,p_kind text,p_account numeric,p_name text,p_phone text,p_role text,p_enabled boolean,p_operation_hash text,p_request_hash text,p_request_digest text,p_receipt_at timestamptz,p_input_status text | 固定4kind；集合锁/lastowner/receipt原子结果 |
| pc_login_begin | p_approval_hash text,p_poll_hash text,p_now timestamptz | 新intent ID及2min到期 |
| pc_login_approval_read | p_login numeric,p_approval_hash text | 原state/expires事实，provider前Go判断 |
| pc_login_approve | p_user numeric,p_login numeric,p_approval_hash text | 锁当前OWNER后条件批准 |
| pc_login_poll | p_login numeric,p_poll_hash text,p_now timestamptz,p_access_hash text,p_entropy_failed boolean | 首次p_now=NULL，锁后采时返回observed_at；第二次冻结该时刻；状态优先 |
| pc_session_authenticate | p_token_hash text,p_now timestamptz | 当前owner版本校验后的user ID |
| pc_session_revoke | p_token_hash text | 仅撤所给access hash |

18新运行RPC加前12共30；ReadIdentity复用order_identity_projection_v1。上述ABI已依据native输入优先级反例冻结；若实际native差分要求额外proof/字节表达，先给具体反例审定，不自行扩大public合同。业务拒绝enum与SQLSTATE对照由writer逐native错误分支列测试，不预猜通用23505重试。

## Session、Phone 与商户登录

Mini Service保留一次微信exchange→32byte entropy→RawURL bearer→SHA256→24h会话；SQL不接raw bearer。不因collision再mint、不重试provider。issued<=now<expires，logout只删所给hash。OpenID原BYTEA<=128，不添加ASCII/NUL限制。Phone已绑定先返回不调用provider；拒绝code后按原再读并发绑定，不能泛化成功兜底。

merchant Service保持 Start短事务→一次provider→Complete短事务，拒绝provider走Recover。Start user/account SHARE；Complete user/account UPDATE、必要phone account UPDATE；phone unique用受控子事务回滚该phone绑定但仍提交PHONE_IN_USE审计。Recover只接受同user/account/authVersion/code、FIRST_BINDING或CONCURRENT_BINDING_CONFIRMED且occurred>=started；ALREADY_BOUND不能证明拒绝code成功。40P01/40001/55P03最多原2次DB尝试，不重新消费code；未知commit不臆断失败、不额外重放provider。

LEGACY登录scope SHA256(MERCHANT_LOGIN+NUL+BE64user)，codeHash原domain separator，requestHash=SHA256(rawrequestID)。Extra USER scope同二进制BE64，operation/request均raw无prefix；digest为canonical后原Go Phone/Name大写字段Marshal。Extra锁user→discount→相关staff按ID，receipt严格解析并核当前extra仍匹配，pricing返回旧receipt快照。extraSetAt/audit使用原SQL clock_timestamp，不套履约时钟。原合法Unicode/NUL key只传hash，不缩窄输入。

## 账号与 PC

账号单固定enum command RPC保留CREATE/UPDATE/SET_ENABLED/DELETE四kind。锁全部undeleted账号ORDER BY id，再找当前enabled OWNER，再receipt，最后新参数校验/写入/receipt。lastowner不可禁/删/降级；同值新key仍增record/auth version；softdelete保留phone占用。digest来自原AccountCommand Marshal（Kind/AccountID/Name/Phone/Role/Enabled），不是HTTP小写DTO或canonical后新模型。merchant scope文本，operation/request有原prefix；account response按原Go receipt解析宽度，不导入Extra的严格decoder。新receipt使用独立Go audit clock UTC截微秒；旧receipt不被新clock失败提前阻断。

Begin先两个独立32byte秘密，SQL仅hash；QR/2min原格式。Approve三阶段：读WAITING+有效期→既有mini.Login/provider→锁当前OWNER+重检intent/expiry。中间Login已提交的binding/audit不因后段expiry被回滚。

Poll使用一个固定RPC的两次调用，不持HTTP跨调用锁。第一次material=NULL/entropy_failed=false，锁intent并按原id/hash、expiry、WAITING、APPROVED顺序判断；WAITING直接返回，CONSUMED/expired拒绝，只有APPROVED返回NEED_MATERIAL。Go此时取32byte秘密；成功或失败都调用同RPC锁内重检：状态/expiry优先，仍APPROVED且entropy_failed才Unavailable；仍APPROVED且有效hash才一次CONSUMED。无材料不能写。竞态已经CONSUMED时不可被本地entropy错误抢先变Unavailable；不将preflight当最终写授权。明确native Poll本身不查账号currentAuth，不能增加新拒绝；Approve finalize和Authenticate必须原当前owner/version核权。确定性barrier差分验证WAITING不取熵、消费竞争/到期/entropy失败优先级，发现不可等价时停止该路径修正而非降低门禁。

PC Authenticate保留expires>now及绑定/owner/authVersion，不新增issued<=now。Poll一次消费后未知响应不能恢复raw token，保持原Unavailable/后续Expired；不新增receipt/rawtoken落库/自动传输重试。仅当前hashlogout，过期亦可幂等删除。

## 管理 bootstrap

管理面固定审定SQL，仅原五个显式输入，原RepeatableRead事务及storefront id1→discount id1→OWNER集合ORDER BY id锁序。三组全无才created；仅原逐字段exact才unchanged，否则conflict。不是全库empty假设；其他SUBACCOUNT可存在且仍按native结果处理。原8flavors顺序、closed、100折扣、version1/NULLgroup/CURRENT_TIMESTAMP保持。一次性工具不在API启动调用，runtime无该函数EXECUTE、配置INSERT或管理密钥。

已亲读管理能力回执 shared-pg-bootstrap-management-rr-probe.json：Request895d7761-81ac-4f47-a10c-88f6594fea4f，BEGIN ISOLATION LEVEL REPEATABLE READ→首次SELECT返回repeatable read/off→COMMIT。正式方案为固定管理事务BEGIN RR、审定业务语句、COMMIT，由owner一次性执行，非运行端泛SQL；不在已启动DO后设置隔离。原两次事务重试及3个named unique保持。能力探针与后续业务验证分开：本地差分及云端空三组RR 16并发/末步回滚已完成，证据见tasks；不以早期无写探针代替。

## 权限与交付

新增runtime两session表，累计12业务表RLS；新增users/accounts/PC序列USAGE，连原audit/outbox累计5sequence。users仅必要INSERT/phone-extra-lastlogin UPDATE；accounts原CRUD必要列、无DELETE；sessions必要INSERT/SELECT及所给hash DELETE；PC必要UPDATE；discount/staff仅SELECT+UPDATE(id)行锁，不配置写。原known26表/16seq清单、列ACL、额外关系及SECDEF扫描保持，bootstrap不进运行allowlist。实际列权限随固定SQL逐项对照，不授ALL。

本地v44真实native差分、原HTTP与真实Order会话中间件、provider spy调用数、currentauth/lastowner/拒绝审计、并发/故障/未知commit矩阵。真UAT普通JWT验证业务存储，管理bootstrap仅合成五输入并清理；真实微信未齐单列BLOCKED_EXTERNAL，不能用spy冒充。Writer ordinary/race/vet/build/smoke/secret/owned/strict→candidate→clean detached独验；依赖未集成不得集成，完整95/25及UI3目标保留。


## 构造、响应与源码归属冻结

identity.NewSharedPGRepository(client)实现原SessionStore/PhoneStore；merchantidentity.NewSharedPGStore(client)实现完整Store，ReadIdentity复用现有SharedPGIdentityReader；NewSharedPGAdminApplication(client,mini)保留原AdminApplication。clock/entropy只加包内测试seam，无用户配置。函数业务结果统一使用`outcome`固定字符串作为内部判别，附该端口原始facts；只有实际已提交业务拒绝才返回业务outcome，SQL故障保SQLSTATE并脱敏。不得把HTTP200的provider错误envelope当业务成功。具体outcome集合逐native error分支审查后冻结，不吞未知值。

业务writer SQL为identity_commands.sql（Session/Phone/merchant Store）、merchant_admin.sql、pc_session.sql；管理bootstrap固定SQL在cmd/order-bootstrap同owned范围，不加入CallRPC。root唯一维护registry/access/readiness/client。签名以本文件named参数表为唯一权威，原12签名保留；其中Extra共7参数、account command最终12参数。


账号审计clock采用同一固定command RPC两阶段：第一次p_receipt_at为SQL NULL，锁集合、核当前actor/旧receipt与原命令校验优先级后，在需新receipt时返回NEED_MATERIAL而不持久写任何业务行。Go仅此时采样包内receiptNow，第二次带时刻调用同RPC重新锁定与完整复核；旧receipt、新输入拒绝及last-owner拒绝不得被新clock采样提前阻断。native可存year1，不能以IsZero新增拒绝。该预读不宣称预测全部数据库约束失败，也不得增加试写/回滚框架模拟未来失败；未知提交遵循原receipt恢复边界。

AccountCommand未被当前kind使用的Name/Phone/Role仅传SQL NULL，但完整原command仍参与Go digest。实际使用字段若含无法传PG text的bytes，不允许JSON替换后成功写入，也不能在Go提前错误越过当前授权/旧receipt；已依据native实测新增第12参数p_input_status text，仅VALID/INVALID_INPUT/STORAGE_UNAVAILABLE三值。该状态在当前owner及receipt、原target/lastowner/命令输入优先级后使用，不以含糊NULL或魔法sentinel区分错误。使用的Name含NUL/invalidUTF8为STORAGE_UNAVAILABLE；Phone/Role无效、或无效Phone同时Name不可编码时为INVALID_INPUT；原完整digest不变。未知status failclosed，不能作为绕过校验的新公共入口。


INSERT权限按实际列授予：miniprogram_users仅openid/created_at/last_login_at三列；merchant_accounts为native显式14列；merchant_pc_sessions为intent六列；miniprogram_sessions四列。不为省略列名单授整表INSERT。readiness以authenticated_insert_column_names精确对照实际有效可INSERT列，包含PUBLIC/继承及table-wide授权的影响，额外列即漂移拒绝。既有audit/outbox受审整表INSERT保持，UPDATE仍仅必要列；固定30函数/12表/5sequence不变。


PC Poll clock最终裁定：两次RPC使用同一逻辑Poll捕获的now，不在取entropy后重采。native持intent锁后、entropy前只采样一次，slow entropy跨过login expiry仍以原now消费；共享实现不能新增此时到期拒绝。第二RPC仍重检当前state/hash/expiry，但expiry比较冻结now；另一次新Poll重新采样。须有slow entropy跨界native/shared成功差分及首次已expired零entropy负例。


### 锁等待跨到期修正（真实Red/Green已闭合）

仅保持slow entropy同一次now尚不足：native Poll先取得intent行锁再采now，Approve finalize先锁当前OWNER再采now；若共享实现请求前Go采时，等锁跨expiry可能错误放行。该问题已由下述真实Red/Green修复；旧版本证据不覆盖新源码。

唯一修正路线：Poll首次调用p_now=NULL，由SQL取得intent锁后使用clock_timestamp采样并返回observed_at；只有NEED_MATERIAL才Go生成entropy。第二次带该observed_at并冻结，不重采，以保留原slow entropy跨expiry可成功；当前state/hash重检和Expired仍优先于entropy失败。Approve finalize在owner锁后由SQL采当前时刻再比较expiry，provider三阶段保持。不得用时钟offset/new配置或“网络本来不同”豁免此边界。最终Approve删除p_now，仅3参；Poll仍5参且首NULL/第二observed_at。权威frozen-rpc-signatures.json SHA256 cab6a9cadbc03b6115321078f355c4055ad2e1ed5f72573aaf7e7cc87d209e1a已核，1.3签名确认闭合。

验收须由外连接锁住intent或OWNER、确认请求实际阻塞、越过expiry、释放且不消费/不变更auth，证明native拒绝、旧shared错误放行，再同场景Green；另重跑首次expired零entropy、slowentropy冻结成功、并发已消费优先及审批provider副作用边界。数据库锁后时间是运行端唯一判定时钟，不添加对外可选时钟来源。

## 验证状态

最终普通服务身份UAT安装、业务矩阵、最小权限及清理已完成，锁后clock和native错误优先级由本地真实PG差分覆盖；当前证据以tasks所列最终回执为准。候选/独验/main集成尚未完成，设计不构成完整runtime或真实微信通过声明。
