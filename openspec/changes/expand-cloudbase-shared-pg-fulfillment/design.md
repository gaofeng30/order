# 设计：完整履约命令与真实读取

## 权威源码与业务界面

基于 exact 775e6746ded9555f9dd243682db011a869b72fbd，逐读 fulfillment/postgresql.go、handler.go、types.go，orderquery/repository.go、handler.go、types.go，以及 subscription/service.go、postgresql_store.go、CONTEXT.md。以现有代码和已确认规格为准，不统一各命令原本不同的幂等行为。

保留 fulfillment.Application.Execute(ctx, WriteMeta, Command) 与 Result；保留 orderquery.Application 的 ListUser/GetUser/SearchMerchant/GetMerchant/GetMerchantAtState。新增共享读适配只依赖固定 CallRPC、TokenOpener 与时钟；商户身份通过NewSharedPGIdentityReader(client).Identity(ctx,userID)直接实现既有MerchantIdentity接口，不套要求写方法的Service.Store。现有 Order Bearer、HTTP 路径、request/response DTO、no-store 和错误码不变。Go 管理随机、Seal/Open、常量时间hash比较及公共投影；数据库 RPC 管理事务和可信平台 UID。

## 写命令与锁

所有写命令重新核验当前商户账号，锁序 account FOR SHARE→order FOR UPDATE→已有receipt检查；MARK_READY 再 order→consent→outbox，最后追加命令receipt。scope=merchant:userID:accountID，action各命令不同，operation与request摘要精确保持原Go JSON字段顺序/omitempty字节语义。新命令使用Go receiptRequestFor的json.Marshal原始UTF8文本，SQL核对kind/字段集合及各值与固定参数一致后hash原字节，尤其保留<>&及Unicode转义。禁止将新SQL JSONB编码结果随意当原请求摘要；uint64 始终十进制精确，receipt JSON仍原数字合同。

MARK_READY：PREPARING→READY_FOR_PICKUP，ready_at/issued_at为本命令UTC微秒时间、version恰加1。Go crypto/rand读取32字节，RawURLEncoding成token，sha256及现有Cipher.Seal；keyVersion1..65535、cipher1..192字节。明文不落库、不写日志。旧receipt优先，不因新随机/Seal失败阻断重放；预读旧receipt后当前scope改变，未生成材料不能被当合法写入，须有限重读。非PREPARING状态维持原TransitionNotAllowed或原完整性失败，不做额外状态推进。

TOKEN：sha256(token)定位优先；CODE：在Execute定位时固定Asia/Shanghai日期与四位0001..9999号码，重试跨午夜不能改变该次定位日期。live定位无结果才用当前账号scope/action/key的既有receipt恢复目标。锁内重新核验locator与当前order，不信预读目标。原receipt摘要对TOKEN包含原token，对CODE包含原四位字符串、不包含计算日期；新日期若找到另一个live订单，不能旧receipt抢先改写定位优先级。

TOKEN/CODE：当前合法COMPLETED且live locator仍匹配时，新key也追加相应receipt并Replay=true，不增加订单version、不恢复cipher。直接REDEEM_ORDER的新key在COMPLETED仍拒绝。无live locator且无可恢复receipt返回RedemptionInvalid；receipt损坏/多行/目标不符维持Unavailable或原IdempotencyConflict。当前授权先于成功receipt，解绑/禁用后仍拒绝。

READY核销必须Go Open并校验hash；proof包括所有用于完整性验证的字段，在写RPC锁内完整比对，不能只比version。receipt预读允许跳解密但只能传SQL NULL validated proof；scope/locator竞态不允许此NULL变成写证据。继承首切片固定错误、命名unique冲突和有限重试，未知提交只用原key恢复。

## READY consent/outbox 的精确原子边界

SQL 从锁定order生成 recipient_user_id/order_no/pickup_date/time/point，不接受客户端通知载荷覆盖。先验证与现有validIntent/validMessage等价的格式，即便没有consent也不能跳验证。选择该order/kind=READY中grant_sequence最小的ACCEPTED consent（不选择最新、不跳过已消费去取下一条）；无ACCEPTED是通知no-op，订单仍正常READY。

consent recipient必须匹配order owner；锁existing outbox(order,kind)。已有行仅当consentID/recipient/templateConfigVersion/message JSON全部相等才no-op；不相等整命令失败。consumed_at已存在但没有outbox为Unavailable。新行PENDING、attempt0、version1、无lease/provider/sent，availableAt来自命令意图，created_at/consumed_at来自受控通知时钟；使用p_now与p_enqueue_at分别传命令与通知时刻，后者内部clock默认time.Now并具确定性测试seam；不得伪称原实现两个clock天然完全相同。插入outbox与更新consumed_at精确一行、订单更新及receipt同事务，任一步失败全部回滚。

subscription.ErrForbidden/ErrIdempotencyConflict与fulfillment同名error不同；原mapApplicationError对上述订阅失败最终返回fulfillment.Unavailable，新实现不得误映业务Forbidden/IdempotencyConflict。只迁移enqueue原子事实，RecordConsent、worker lease/send/fencing与微信送达均不在此change。

## 五个读取端口

列表保留id DESC、id<after、limit+1判断NextAfterID、limit1..100、空数组非null；用户仅owner，Active仅RESERVED/PREPARING/READY且不能同时state；商户保留state/date及order_no精确、四位pickup_number精确或phone后缀搜索，LIKE的=/%/_仍按原义转义。

详情返回orders及按line_number排序items的原始facts，在Go复用/等价保留validRecord、时间历史、金额溢出/乘积/总和、连续line/count、UTF8与flavors非null等校验。用户owner不符伪装NotFound；联系人只输出mask，商户永不输出token；仅用户READY经Open+hash返回token。状态/actions/NotificationOptions/时间投影及CANCEL严格大于30分钟阈值不变，查询不能推进状态。

GetMerchantAtState不是普通强制状态覆盖：先校验当前完整record，再允许等于当前状态或具有相应历史时间的READY/COMPLETED投影，屏蔽未来时间，不能伪造历史；用于成功命令之后即使订单已继续推进仍返回原receipt状态的合法响应。

隐藏只读依赖：native authorizeMerchant→Service.Identity→ReadIdentity涉及miniprogram_users、discount_settings、staff_whitelist、merchant_accounts，并校验phone/extraPhone+canonicalNameKey、recordVersion、rate/whitelistVersion等。不能仅用enabled binding代替；漂移原本Unavailable仍须Unavailable。root将native扫描事实及投影提取为package-private identityRecord/projectIdentity；native ReadIdentity复用投影，原Service.Identity继续validIdentity(false)。新增SharedPGIdentityReader.Identity调用固定RPC解码同facts、projectIdentity并validIdentity(false)。不新增未实现Login方法或完整Store，不迁移这些主体写流程，也不使用stub声称读取已完整。

## RPC 与运行权限

沿用Client.CallRPC固定allowlist，无任意SQL/管理员fallback。写侧固定一个order_fulfillment_proof_v1及三个order_mark_ready_v1/order_redeem_token_v1/order_redeem_current_date_code_v1，proof只接受三新增kind固定enum；首REDEEM_ORDER定义保持。读侧四固定RPC覆盖五方法（merchant detail函数含historical state），另一个identity只读facts RPC；不接受自由where/table。具体named参数按下文固定签名，不新增公共业务API。所有业务SQL SECURITY INVOKER、固定UTC/search_path、普通authenticated+固定serviceUID；新增表仅必要列DML和sequence USAGE，未知用户无行读写，禁止CREATE/owner/BYPASSRLS/role membership。

root统一registry/installation/readiness；worker只写独占履约SQL。扩展readiness到所有实际函数签名/body/security/settings和新增表policy/ACL/sequence，保留原44ledger/hash/catalog校验。原已applied管理历史字节不改，以新版本安装。fixture清理只删本次合成marker，不删最终服务用户和已验业务定义。

## 验证与交付

以相同真实v44 fixture对照native与shared的Application及handler；这次handler读取必须实际共享orderquery，不沿用首切片readerStub作完成证据。覆盖正常、重放、全状态/权限/损坏、日期边界、过滤与历史投影、consent所有分支、故障点、16并发、提交后响应丢失。真实UAT普通JWT至少完整主链与每类关键原子/权限/恢复风险，记录source/SQL/role/UID/readback；本地PG16不宣称真PG17。所有故障资产清理再readiness。

现有首切片通过事实按未变输入保留；新增定义/权限必须验新漂移，不机械重跑无关库或云写。Writer ordinary/race/vet/build/smoke/格式/secret/owned/OpenSpec；两轴review、候选exactSHA及干净detached独验。依赖未集成不得集成，任何source/spec/平台定义变化使相关旧证据失效。全runtime、5worker、存储及95/25仍未完成。

## 固定内部接口

worker独占SQL为services/api/sharedpg/fulfillment.sql。proof接受actor/kind/order/key/request/date/locator_live/token_hash：TOKEN/CODE首次order与live为NULL完成定位，后续固定首次resolved ID/live且只刷新该目标proof，MARK_READY固定原ID/live=true。返回原完整redemption proof扩充user_id/order_number_hex/pickup_time/pickup_point、receipt_exists、locator_live。三写RPC返回精确order_id字符串/state/changed/replay；MARK_READY另收p_enqueue_at、可空token_material，TOKEN/CODE另收冻结locator。P1009映RedemptionInvalid，订阅错误P1007映Unavailable。上述主体及下列NUL保持方案已审定，内部接口冻结；实现仍须真实Red→Green证据。

必须保持validPlainToken既有合法输入：包含内部U+0000的UTF8 token不因PG不能把NUL解码成text而被新增拒绝。native仅在Go hash定位；新SQL不能直接把request转jsonb并提取token来悄然缩窄合同。Go原始JSON text及SHA256可作为可信后端命令参数，SQL仍严格核固定kind/locator及原receipt摘要、完整proof；JSON形状与hash传值按下文固定scalar方案实现，不新增通用JSON解码器。

### 已冻结的写RPC参数

- order_fulfillment_proof_v1(p_actor numeric,p_kind text,p_order numeric,p_key text,p_request text,p_locator_date date,p_locator_live boolean,p_token_hash text)。首次TOKEN/CODE定位order/live为NULL；之后冻结返回ID/live。MARK_READY order原ID/live=true。p_token_hash仅TOKEN为hex64，其他kind为SQL NULL。
- order_mark_ready_v1(p_actor numeric,p_order numeric,p_key text,p_request_id text,p_now timestamptz,p_enqueue_at timestamptz,p_request text,p_proof jsonb,p_token_material jsonb,p_receipt_at timestamptz)。material={hash:hex64,ciphertext:hex(1..192bytes),key_version:uint16}或receipt预读路径SQL NULL。
- order_redeem_token_v1(p_actor numeric,p_order numeric,p_key text,p_request_id text,p_now timestamptz,p_request text,p_locator_live boolean,p_locator_date date,p_proof jsonb,p_token_hash text,p_receipt_at timestamptz)。date必须SQL NULL。
- order_redeem_current_date_code_v1(p_actor numeric,p_order numeric,p_key text,p_request_id text,p_now timestamptz,p_request text,p_locator_live boolean,p_locator_date date,p_proof jsonb,p_receipt_at timestamptz)。date为本次Execute首次固定合法上海日期。

proof返回{proof:完整快照,receipt_exists:boolean,locator_live:boolean}，proof.id为精确resolved ID。写结果{order_id:十进制字符串,state:string,changed:true,replay:boolean}。NULL表示未生成/未验证而非跳过写入校验。新Code/Token回执恢复和bounded retry不能重算首次locator或更换ID。

TOKEN的p_request禁止转jsonb或对整个对象调用json_object_keys/->/->>，因为真实PG16这些操作也会解码NUL并失败。只接受固定前缀{"kind":"REDEEM_TOKEN","token":及末尾}，截出中间完整token JSON文本，要求其经json类型解析后json_typeof为string；中间必须恰一个完整JSON scalar，故额外key/重复key/未知key/第二个尾随JSON不能成立。首尾引号及固定envelope保持Go规范形状，不写通用JSON解析器。Go保持validPlainToken、json.Marshal与SHA256，SQL接p_token_hash并核hex64，作为可信服务端参数使用；再核数据库locator与完整cipher proof。先以内部NUL、HTML特殊字符、U+2028/中文token建立真实native-vs-shared差分，不改变合法输入范围。

### 审计独立时钟与读取签名

原audit.ReceiptStore.AppendInTx在追加时调用独立s.now().UTC()，不使用fulfillment app.now或subscription时钟。三个新增command据此各追加p_receipt_at timestamptz，仅用于新receipt occurred_at；Go内部receiptNow默认time.Now，不增加环境配置或公共方法。MARK_READY共10参、TOKEN11参、CODE10参，proof仍8参；首两条RedeemOrder业务定义不改。每次实际新写尝试临近RPC采样，旧receipt保持原时刻。pgx现有二进制编码通过Nanosecond()/1000截微秒，HTTP文本发送前须按实际native精度验证并UTC截微秒，不能把额外纳秒交PG文本解析后四舍五入而偏差1us；不宣称跨网络采样瞬间完全一致。

root读侧固定：order_query_list_user_v1(p_user numeric,p_state text,p_active boolean,p_after numeric,p_limit integer)；order_query_get_user_v1(p_user numeric,p_order numeric)；order_query_search_merchant_v1(p_user numeric,p_state text,p_date text,p_search text,p_after numeric,p_limit integer)；order_query_get_merchant_v1(p_user numeric,p_order numeric,p_state text)。identity为order_identity_projection_v1(p_user numeric)。query numeric事实全部text，详情和items在单RPC一致快照读取。商户先经Go完整Identity校验，查询RPC再次核当前enabled绑定/OWNER或SUBACCOUNT以拒绝期间撤销；不是以简化SQL取代Go身份校验。state参数合法性在入口核验，历史投影权限仍由Go现有函数判定。

### 一致快照与最小权限范围

readiness增加public业务命名空间固定10表/2sequence之外的有效relation权限扫描：anon/authenticated含PUBLIC/继承的表级及列级读写/reference/trigger/truncate与sequence USAGE/SELECT/UPDATE均不能越过固定allowlist；table/partition/view/materialized/foreign等关系均纳入。无运行权限unused对象仍允许，平台auth/pg_catalog不混入本业务清单。仅读metadata并failclosed，不自动撤未知授权。实际原44未迁移表SELECT、额外列UPDATE/额外sequence授权后拒绝及撤销恢复必须留证据。

详情SQL的VOLATILE内部两个SELECT不保证同snapshot；PG17官方Function Volatility Categories明确VOLATILE每内部query取新snapshot（https://www.postgresql.org/docs/17/xfunc-volatility.html）。order/items与商户当前授权合并一个SELECT实现一致快照，保持固定函数类别，不把单RPC误写成snapshot证据。另新MARK_READY/TOKEN重放不能被新app/enqueue时钟故障提前阻断，CODE首次定位取日期的原依赖保留；该优先级已通过native反例及修复后的差分验证。

新增最小权限扫描同样覆盖public中未审SECURITY DEFINER例程的有效EXECUTE：按OID/完整签名对照固定12签名，不只比较函数名；anon/authenticated经PUBLIC/继承可执行即拒绝。已审名字的未知overload也属漂移。无运行EXECUTE的unused SECDEF允许保留，官方auth/pg_catalog不受此业务扫描影响；仅metadata判断不自动撤权。该门针对实际可绕RLS的高权入口，不要求禁止全部unused函数。


### CloudBase 已知业务关系默认授权纠正

真实 UAT metadata 已发现当前 allowlist 之外原 v44 业务关系对 anon/authenticated 的默认有效授权；本地原生 PostgreSQL 无该默认配置，因此本地 Green 不替代这一平台门禁。受控 access.sql 按 postgresql_schema_contract.json 权威最终清单显式撤销 PUBLIC/anon/authenticated 对25张业务表、另加 schema_migrations 账本及16个 sequence 的权限，再仅授予当前10表/2sequence所需精确权限。不得把44迁移误称44表。名单须以结构检查证明与权威清单一致；原44 SQL、账本、schema结构和管理owner不变。

表级 REVOKE 不作为显式列级授权已清除的证明；受管表如存在列 ACL，必须按权威列清单显式撤销后复核有效权限。此修正只处理已知受管资源，不自动枚举撤销未知对象、不扩大到平台 auth/pg_catalog、不顺改默认权限或 schema USAGE。未知关系有效授权与未审可执行 SECDEF 仍由 readiness failclosed，交由范围核实。

真实平台验收必须记录纠正前授权事实、受控管理任务成功、纠正后 allowlist 外有效授权为零及当前精确 ACL/RLS、12 RPC/44 ledger 对照、普通 UID Check 成功和管理通道仍可读取。本轮真实平台已按上述条件闭合该权限门，回执见tasks；只代表限定业务RPC安装，不代表完整runtime部署。
