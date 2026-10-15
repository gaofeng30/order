# 第五切片设计

Status: IMPLEMENTED / WRITER_V4_GREEN / PRECOMMIT_SNAPSHOT_FROZEN / CANDIDATE_NOT_CREATED。父精确候选a6e597976e4ddee4ed2101ac21e9d1d4a3147983 aggregate independent INDEPENDENT_VERIFIED。最终source-v4全16包ordinary/serialized race各1679PASS，局部容量独立复核与分层普通UAT证据完成；候选前最终review、candidate、exact-SHA独验及main集成仍待。

## 单一路径与事实源

Go原业务/handler保持，固定普通服务JWT调用SECURITY INVOKER RPC，命令在一次调用中锁当前授权及业务源并提交业务+receipt。禁止runtime管理身份、任意SQL、SECDEF、跨请求持锁、Tx shim和native fallback。源为staffdiscount/admin.go、postgresql_admin.go与importbatch/postgresql.go及原XLSX/parser/handler；遇到真实native反例先修正内部设计，不改变已确认公共合同。

## 固定 named ABI

```sql
order_staff_admin_list_v1(p_query_hex text)
order_staff_admin_rate_v1()
order_staff_admin_command_v1(
 p_user numeric,p_kind text,p_staff numeric,
 p_name_hex text,p_name_key_hex text,p_phone_hex text,
 p_enabled boolean,p_rate integer,p_input_status text,
 p_operation_hash text,p_request_id_hash text,p_request_digest text,
 p_receipt_at timestamptz)
order_import_preview_facts_v1(p_kind text)
order_import_preview_v1(
 p_user numeric,p_kind text,p_file_digest text,p_file_size numeric,p_row_count numeric,
 p_token_hash text,p_plan_json text,p_response_json text,
 p_captured_at timestamptz,p_operation_hash text,p_request_id_hash text,
 p_request_digest text,p_receipt_at timestamptz)
order_import_commit_v1(
 p_user numeric,p_token_hash text,p_skip boolean,
 p_operation_hash text,p_request_id_hash text,p_request_digest text,
 p_plan_digest text,p_material jsonb,p_observed_at timestamptz,
 p_receipt_at timestamptz)
```

参数数量1/0/13/1/13/10，总6入口；全部返回一个jsonb闭合outcome与必要rawfacts。无source-refresh循环。Go严格检查数值、时间、JSON/SQLSTATE与响应shape；numeric uint64以规范十进制string传输，不经float64。hash为32byte小写hex；operation_hash=SHA256("operation:"+原key)，request_id_hash按原audit request前缀和空值规则；request_digest为原完整cmd/request的Go Marshal摘要。普通Order用户授权仍由原session/admin middleware与命令当前OWNER执行，平台服务身份不替代用户。

## Staff合同


List返回原单SELECT投影raw facts，Go复用原maskPhone和整数扫描边界，不在RPC新增角色业务条件（原公开管理路由负责当前PC身份，Execute另锁Owner）。query以hex承载原bytes；原native绑定不支持的UTF8不得经JSON替换后变合法搜索。Rate只返回原rate事实，由Go维持1..100校验。

Command的p_input_status固定VALID/INVALID_INPUT/STORAGE_UNAVAILABLE，由Go按kind原规则产生；SQL仅在原executeTx该分支应消费输入的位置检查，当前Owner/settings/receipt/target优先不可改变。只有该kind在业务写与审计中均不使用的材料字段才传SQL NULL，原摘要仍包含全部cmd字段。p_staff在所有五kind均必须传原cmd.StaffID的规范uint64十进制值（含0），不得作为ignored字段置NULL。原audit target默认DISCOUNT_SETTINGS/1；只要原StaffID>0即STAFF/原StaffID，包括CREATE_STAFF与SET_DISCOUNT_RATE；不得改用新建staff ID或按kind重新推断target。name_key由原staffInput/规范化Go逻辑生成，不用通用数据库normalize；SQL约束保持原失败阶段。NULL receipt阶段不写，只有原新命令条件满足后NEED_RECEIPT；Go采样独立audit clock后同RPC完整复核并写。新SQL不得模拟会产生持久副作用的试写来“提前验证”。常见结果OK/REPLAY/NEED_RECEIPT、原InvalidInput/NotFound/Conflict/IdempotencyConflict/Unavailable；未知结果failclosed。

Staff UPDATE 的空手机号是明确例外：`p_phone_hex=NULL` 表示原 `cmd.Phone==""`，不是输入缺失错误。SQL先按原生锁目标行并读取 `convert_from(phone,'UTF8')/enabled`；目标读取或扫描失败映射原 NotFound。健康v44 CHECK保证原手机号为ASCII规范串，锁内直接复用即可，不增加facts/refetch或SOURCE_CHANGED循环。Go只规范化Name/key及非空Phone，把Name/input错误延迟到原分支位置；不能对空Phone提前整体staffInput并拒绝。SET_ENABLED的nil指针则按原生先于目标查询拒绝。maskPhone按原Go字节切片规则。

### Import Preview

facts仅允许原两kind，返回products.name_key、categories.name_key或staff phone的原始集合，原plan解析/重复行判断/价格/规范化留Go。产品facts内部保持原先products后categories读取顺序，不宣称新增RR原子快照。

Preview的p_plan_json、p_response_json是原Go Marshal文本：plan进入原preview_json持久列，response包含原preview token、计数、行错误和expiry；token_hash必须与该response中的token一致。p_file_digest来自原XLSX bytes；file_size/row_count按原整数类型，expiry由captured_at+15min。原合法行错误可进预览；不得将plan错误行当整批InvalidInput。p_kind/计数/文件摘要/response关联做闭合材料一致性校验，不自行重新定价或重新规划导入。

ParseRows→buildPlan→随机token→captured clock顺序在Go保留，**同key旧preview也不跳过这些步骤**。RPC当前Owner后执行新batch/receipt的原流程，duplicate receipt必须先回滚本次batch，再按原request恢复旧response；不因为已经存在receipt而绕过本来先发生的batch约束错误。audit receipt clock NULL材料预检只用于取得必要独立clock，不承诺晚数据库约束失败前绝不采样clock。不得输出token到日志/普通证据；脱敏回执仅记录恢复是否相同。

### Import Commit

首调用p_plan_digest/p_material/p_observed_at/p_receipt_at均NULL。当前Owner→batch锁→原COMMITTED receipt/原状态→锁后clock_timestamp观察expiry→skip规则，然后返回NEED_MATERIAL、完整plan JSON原文、sha256原文及observed_at；本阶段零持久写。Go使用原json.Unmarshal宽容度/plan.Kind检查（不擅加未知字段拒绝），对有效行按原staffName/catalogKey生成闭合p_material。

p_material固定 `{status,rows,new_categories}`：status=VALID/UNAVAILABLE；rows依原plan有效行原顺序，元素 `{row,name_hex,name_key_hex,phone_hex,category_key_hex}`，不允许自行增加/删除有效行；new_categories按plan原顺序 `{name_hex,name_key_hex}`。业务源值仍来自锁内原batch plan，material只补Go规范化结果，禁止其中替换数量/价格/结果计数。INVALID plan通过UNAVAILABLE材料在原优先阶段返回，不能先于COMMITTED/currentOwner结果。只读材料hash使用本次返回原文字节，不能PGjsonb再序列化后比较不同格式。

第二调用回传同observed_at与plan_digest、独立receipt clock；SQL再锁当前Owner/batch，COMMITTED仍先走原receipt。仍为同一个PREVIEWED plan才执行原集合锁和全批写；不得用REST发出前时间替代首次真正拿锁后的observed_at。observed_at仅保留原拿锁后前置到期判断，不保证慢材料跨expiry可提交。最终batch UPDATE必须继续使用真正clock_timestamp()填写committed_at，并保留v40 committed_at<=expires_at CHECK；跨期晚约束失败按原importSQLError映射并全批回滚，不可用observed_at替代最终时刻。新请求首调用重新观察当前时刻。若管理面修改不可变plan，digest不符failclosed，不开始SOURCE_CHANGED循环。

必须真实比较原plan损坏、COMMITTED receipt恢复、锁等待跨expiry与slow-material跨expiry两种反例，证明该两阶段没有把成功/错误优先级换掉。无需引入新业务表、会话锁或通用事务接口。任务中若发现原代码某分支再次采样clock，应依实证调整具体阶段而非把全部时间合并。


## 统一安装、权限与部署

按本轮SQL实际使用列清点最小授权，registry准确55；本地统一SQL实际盘点为21张RLS表与10个USAGE sequence，实际安装/Check ordinary及race各26PASS。最终同source-v2普通UAT55/21/10完整检查已通过；后续源变化仍须重验。延续固定UID RLS/角色成员拒绝、非registry公共relation/column授权及可执行SECDEF漂移拒绝。完整v44 ledger/checksum/catalog及55个函数签名/body/INVOKER/UTC/search_path/volatile/strict等实际事实由Go权威对照，不信数据库自报ready。anon/其他UID不得执行新六函数或读取目标数据，runtime不能DDL/账本写。

管理迁移保留已applied历史字节，不force伪造修复/盲重放；已实际执行后的官方history登记按下节执行。若完整安装超过平台blob限制，可按已验证机制只省略与当前已安装历史逐字一致的函数定义，保留所有必要新/变更函数及ACL；必须预比较、实际旧→delta→Check、故障rollback和全部最终body对照。没有实测前不声称增量安装PASS。

## 验证与证据边界

原生真实v44同输入/同durablefacts先Red→Green；NULL材料与独立clock机制分别证明currentauth、receipt、目标/校验优先，不声称能零写模拟所有晚constraint失败。Import首锁observed_at必须用真实锁等待跨到期反例验证；第二阶段慢材料跨到期仍受最终数据库clock_timestamp与原CHECK限制，必须取得真实错误阶段/全回滚差分；本地到期子门已由tasks所列12PASS回执确认，统一安装与普通UAT业务/故障层亦已由最终回执通过。Quote持源SHARE锁与名单/折扣/目录/导入写入互斥作为直接相邻回归。

平台以真实普通JWT、真实Go业务/原HTTP、合成微信provider执行六RPC及决定性16并发/丢ack/末尾故障/权限撤销；loopback原handler不等于CloudRun部署。保留错误harness尝试并区分修正，清理仅明确synthetic数据/临时身份/constraints，保留既有meal等baseline行逐字一致。最终ledger44、55body/ACL/RLS/普通Check以及源码hash绑定。Writer本地、UAT平台、candidate、independent、main集成各层独立，未运行的门禁仍为NOT_RUN。

Staff List持久CreatedAt返回Go后恢复time.Local，与native pgx扫描和原HTTP序列化一致；命令fresh零值CreatedAt保持原UTC零值，不添加时区配置或新姓名长度上限。

## 已取消caller的有限内部错误差异

原生target行Scan把ctx取消映NotFound；共享HTTP中断只能知道transport取消，无法可信识别数据库正处OWNER/settings还是target阶段。因此仅在ctx.Err()!=nil的已取消调用，允许共享返回Unavailable而不复刻native NotFound；不将此诊断记为完整错误类别等价。有效ctx下真实target不存在、原扫描/解码失败及其他业务错误仍保持原映射。不得将全部Update timeout映NotFound，不增facts查询、跨HTTP锁、隐式重试或Tx shim。

初始target-cancellation-diagnostic.jsonl只证明内部ctx/error差异；后续本地真实HTTP/锁barrier子门已证明该实验无响应和无副作用，见tasks。该本地实验已实际观察锁等待取消、HTTP无响应、事务终止/锁释放及表/版本/receipt不变，不能泛化为所有断连。网络响应未知不能据本例承诺所有取消都等于数据库回滚：已提交但响应丢失继续原同key恢复合同。本地限定取消子门已实证；不宣称所有云请求取消都有相同数据库终止行为。

Import Products的category/product name_key INSERT保留native string(catalogKey)参数经PG bytea文本输入的晚消费语义。共享材料仍rawhex，但对应INSERT在原阶段convert_from(rawKeyBytea,'UTF8')::bytea，不直接decode后写；categories map仍按原Go rawkey。single/double反斜杠及Unicode差分须真实RGR，不能升级原native算法来消除反例。

## 管理请求大小的有界readiness增量

完整旧history须保留；删旧Query会产生history冲突，不允许force/repair绕过。对已验证未变的48函数不重发定义；管理DO仅处理同名同signature readiness：入口核批准旧49的identity/属性/prosrc SHA，六处固定substring逐个精确出现次数断言，替换后最终prosrc SHA必须等于编译源码55 expected，再以源码同属性重建。禁止任意SQL输入、泛化字符串编辑器或未知源码容错；不产生helperRPC/表/SECDEF。

新6函数、readiness重建、discount/staff/import三目标关系与两sequence的精确ACL增量同一正式管理迁移事务。所有异常必须回滚；repeat仅接受已精确55且完整权限满足的分支。未参与变化的48body保持，未知对象权限不自动修复。要求本地旧49→delta→全55Check、部署中故障全回滚、repeat、未知源码拒绝且无写；随后云完整55源码/属性/ACL实际readback与普通Check。payload测量仅是该管理通道限制证据，不推断未经证实的官方阈值。旧49→delta/repeat/rollback/未知源码无写及晚故障/权限漂移的六项本地验证已通过；最终云55完整readback与普通Check已通过，见tasks。

## 固定55入口的成功响应容量策略

Staff长名单P2后，Standards又实测三Import成功facts/NEED_MATERIAL plan/REPLAY response超过8MiB：native成功而Client拒绝。原SQL已正确，不修改55函数/ABI/业务限额；修复第五公共Client并按全16包及平台定点证据重绑最终source-v4，旧v2不是新源PASS。

仅下列24个精确RPC名称的HTTP200响应不设人工body字节cap；其余31/signin/非200继续8MiB。不能按前缀、所有read或未来RPC自动放开。native全量集合、无界TEXT、内部Application结果/receipt及XLSX共享字符串扩张是依据；有限行数或某公开POST body限额不证明整个成功输出有界。逐55项依据与机器名单保存在外部architecture-ablation/fifth-rpc-success-response-capacity-decision.md / policy.json。

- `order_fulfillment_proof_v1`
- `order_query_list_user_v1`
- `order_query_get_user_v1`
- `order_query_search_merchant_v1`
- `order_query_get_merchant_v1`
- `order_merchant_account_current_v1`
- `order_merchant_account_list_v1`
- `order_merchant_account_command_v1`
- `order_catalog_browse_v1`
- `order_catalog_categories_v1`
- `order_catalog_products_v1`
- `order_catalog_command_v1`
- `order_menu_snapshot_v1`
- `order_storefront_get_v1`
- `order_storefront_admin_v1`
- `order_storefront_configure_v1`
- `order_display_pricing_facts_v1`
- `order_quote_create_v1`
- `order_quote_read_v1`
- `order_staff_admin_list_v1`
- `order_staff_admin_command_v1`
- `order_import_preview_facts_v1`
- `order_import_preview_v1`
- `order_import_commit_v1`

固定host/路由、普通UID、ctx/原timeout、严格单JSON/UseNumber/EOF与业务codec保留；HTTP200 code/message错误信封仍failclosed并脱敏，超大错误不输出正文，不能声称读取前能辨识错误信封并单独限读。ReadAll可沿用，无需新流式框架/容量配置/公共分页/数据裁剪。原字段/文件/行数有效上限不改变。

必须三Import实际Client+原handler Red→Green，24/31逐名传输策略测试及signin/非200/未知route/畸形与多JSON/错误信封/取消/timeout回归。目标UAT大facts实际普通JWT→Preview及持久读回、清理/55Check为新硬门；Staff9.7MB旧实测保留但不代替facts。其他白名单是代码合同推导+client路由验证，不声称已逐一大规模云测；有限样本也不证明无限平台容量。最终source-v4全16包ordinary与串行race、静态/manifest/秘密扫描/两轴重验，不回改前四冻结候选。

## 当前部署单一路径：事务执行后登记历史

59,498B compact delta连同原23历史请求实际2,093,994B；官方Preview可执行但Push仍Blob size拒绝且没有TaskId，readback仍49/23且新版本不存在。compact-migration-rejected-receipt.json是实际拒绝证据，不能推断官方数字阈值或继续盲重试。当前批准改为官方管理ExecutePGSql执行受审版本化事务，再官方RepairPGUserMigrationHistory登记已执行事实；不是略过历史管理或假报applied。

依据：manager-node 5.9.0 lib/database/index.js的executePGSql/repairPGUserMigrationHistory（后者只改history不执行SQL）；[官方CLI repair](https://docs.cloudbase.net/cli-v1/db/postgresql/management)明确支持已在库中执行SQL后的历史补齐；[官方API](https://cloud.tencent.com/document/product/876/132258)明确只维护user_schema_migrations。

1. root独占该环境迁移流程，先固定新Version/Name/实际Query字节及SHA，原23历史Query以安装前fresh官方Describe实际返回为权威baseline逐条保存，不能用先前submission原文冒充平台recorded字节。事务wrapper、锁与检查计入实际执行artifact；登记Query必须是该原文，不能换成未执行的fullInstallSQL。
2. ExecutePGSql提交单一事务，包含有界delta与后置断言。事务中失败全部rollback；完成后独立读取全部55body/signature/属性/ACL/RLS/ledger44，并用普通服务JWT Check通过，才允许登记。
3. Execute响应丢失先查状态，禁止凭history缺失重放：exact55且新history缺失为APPLIED_UNREGISTERED，只补登记；exact49且确认无进行中的旧事务才允许重执行相同artifact；混合/未知/漂移停止并诊断，不repair。此中间态不是部署成功，不放候选或并行另一迁移。
4. Repair使用Status=applied、同Version/Name/原Query、诚实Reason记录已直接执行及校验回执；不能执行SQL或覆盖旧23。登记响应未知先detail/list；精确match即完成，仍缺失且55仍精确才仅重试登记，同Version异Name/Query立即停止，不能覆盖修齐。
5. 最终必须history24且原23相对fresh官方baseline逐字不变、新Query按下述执行/登记对应规则核验、55Check/ledger44通过。保存执行/登记RequestId与两阶段状态、失败续接和最终readback证据。真实业务及大响应门独立，history闭合不等于完整平台验收。

这使用官方接口已有历史补齐能力，不修改runtime接口/权限/业务源；以后同类管理体积限制沿同一执行-核验-登记协议，不在两条迁移路径之间自动fallback。

### submitted与recorded Query的证据口径

执行前fresh Describe23与先前内存submission中13条有空白差异，Name/Version相同；该诊断未触发旧历史重写。安装前baseline必须取fresh Describe真实Query，安装后用同API逐字节比对23 unchanged。历史submission SHA保留为提交证据，与官方stored SHA分别记录，不回改旧receipt。

去全部空白后一致只用于诊断，不能证明SQL语义等价，因为字符串、dollar body与注释内空白有意义。新版本分别记录executed/submitted Query SHA与stored Query SHA；相同则闭合，不同则先核官方规范化算法及实际变更位置，证明字面量/函数body/注释边界未改变及存储内容确实对应执行SQL，诚实标为官方规范化。没有此证明时为REGISTRATION_UNVERIFIED，不能仅凭55Check或自定义去空白匹配宣布完成，不盲重执行或覆盖history。

### 本次登记三LF差异已完成限定核验

执行artifact 59,583B SHA8df687ebe37e4fe62c36b4d85f6bbbccb5bfdbf0b9daae41da02eb5fd0b69ae5；stored 59,580B SHA5031b53f7d5f87f1762e1bd1ac971aaaeaf27f92c2de628f54210953d0ada4d2。独立逐byte diff仅删除offset47350、59574、59582的LF；词法scope检查全部在外层literal/dollar/comment之外：已结束GRANT后的重复换行、已结束DO后的重复换行、COMMIT分号后的EOF换行。前两仍保留换行，所有分号与quoted body逐字不变。根据[PG17官方词法](https://www.postgresql.org/docs/17/sql-syntax-lexical.html)命令分隔规则，本次命令含义不变，可记RECORDED_FORMAT_VARIANT_VERIFIED并闭合此登记；不是newRegistrationExact=true，不宣称服务端一般规范化算法。

前述要求官方算法旨在避免任意去空白等价；对于此次精确三处外围空行，可由具体字节差异与官方词法规则独立证明，无需推断未公开算法。其他版本或其他位置差异必须重新核验，不自动套用。本次Execute RequestId70410415-2a82-4c25-8fa3-913fad92e266，Repair RequestId4631fa37-339d-40d9-aac4-605bd935faa8；actual history24/旧23freshbaseline不变及55普通Check已读回，完整限定业务与9.7MB容量门现亦已通过最终平台回执，候选/独验/main仍待。

## 同一容量合同的Staff响应解码层

Client放行24入口不代表业务codec已经满足容量合同。亲读sharedpg_admin.go确认`sharedStaffDecode(*f.Response,8<<20,&r)`仍会拒绝大合法response。SET_STAFF_ENABLED可由小公开请求返回此前合法XLSX导入的大Name；原16KiB命令body限制不能证明该响应有界。此公开路径必须保持native成功集合，批准仅移除Staff command fresh/replay response人工cap，保留before/evidence的256字节界、digest、JSON有效/单值EOF/未知字段与类型检查，不改变SQL/ABI或既有业务限制。

Standards已取得实际原链路Red及V4普通/race Green，原V3失败发生在SQL已提交之后；V4目标云原key恢复/新key/replay/List与大facts也已通过，证据层见tasks。决定性验证已先用合法worksheet导入持久长Name，再原HTTP小body SetEnabled：native成功/shared旧codec503，并独立读回证明事务是否已提交；V4修复后同case成功且同key重放不重复写，不能把503误称无副作用。root接管Staff Go+test两路径，其他所有权不变。修复后新source manifest/必要回归与两轴重新绑定，旧V3仅Client两源变化的结果不自动证明后续Staff源修复。

V4完整16包ordinary/serialized race已各1679PASS/0FAIL/SKIP。新隔离PG认证harness故障、仅临时HBA修正与完整同源重跑如tasks记录；真实UAT认证未修改，旧失败保留。候选及独验需按冻结快照另行绑定，不用局部复核替代。
