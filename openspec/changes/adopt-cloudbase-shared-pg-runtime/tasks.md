# Tasks

Status: APPROVED / IMPLEMENTING；原APIKey身份门FAILED已由普通JWT REST/RLS修正并实测；首切片收尾，候选尚未创建。gate_type=W3；ui_level_target=UI0；ui_level_actual=NOT_RUN；base_sha=12164b115441cb2a50c96d05b3173f874042568e；candidate_sha=NOT_CREATED。ownership/dependencies见proposal。勾选只代表该项有实际证据。

## 1. 需求与设计

- [x] 1.1 阅读真实核销、merchant authorizer、audit、migration与CONTEXT，固定首切片REDEEM_ORDER及完整后续依赖。
- [x] 1.2 建立proposal/design/spec/tasks；需求结构Red为目标spec不存在，Green为相同存在检查与OpenSpec strict。
- [x] 1.3 评审root隔离platform probe DDL和Go客户端接口方向，给出角色/uint64/响应丢失具体修正。
- [x] 1.4 评审真实业务函数签名、SQL和受控空库安装的完整设计；闭合密文proof、原request字节摘要、account→order锁序、ledger及安装锁语义。
- [x] 1.5 复核REST/RLS修正后源码、v44实际metadata与最终服务UID生成DDL，仅放行限定UAT管理安装/合成业务实验（证据见末节），不代表候选完成。

## 2. 真实平台前置门

- [x] 2.1 Red：当前环境无本次RPC能力/工件，记录真实受限调用及缺失行为，不将本地SDK或多SQL错误伪作核销业务Red。
- [x] 2.2 Green：安装隔离probe，参数/SQLSTATE、完整uint64、时间/JSON、回滚、16同key并发实测。
- [x] 2.3 真实网关与管理通道角色分别记录；验证非运行角色RPC/直接表访问拒绝、角色继承与可信身份。
- [x] 2.4 Refactor：仅修本探针必要问题并重验受影响场景；清理隔离函数/表和临时授权并复读。

## 3. Go HTTP与真实核销纵向TDD

- [x] 3.1 Client Red：固定SELECT+参数、超限/超时/错误SQLSTATE、uint64/JSON精确解码、取消/redirect/脱敏失败行为。
- [x] 3.2 Client Green：最终最小内部CallRPC固定三函数，无Exec/BeginTx、无自动重试；同测试通过，取代已失败的APIKey入口。
- [x] 3.3 RedeemOrder Red：通过现有Application/handler契约观察共享PG能力缺失与原子性/重放目标。
- [x] 3.4 RedeemOrder Green：proof只读+Go cipher验证+单原子RPC，当前授权、状态+receipt及原有错误/DTO一致。
- [x] 3.5 合法OWNER/SUBACCOUNT、禁用/解绑、密文损坏/proof竞态、不同key已完成、同key不同order等失败回归。
- [x] 3.6 16同key、16不同key竞争、真实提交后丢响应重放、回执插入失败全回滚；与原生同条件结果对照。
- [x] 3.7 Refactor：只处理本切片必要重构；重跑同一故障/并发场景与受影响race；无必要重构则记录理由。

## 4. 真实schema与迁移门

- [x] 4.1 原44SQL hash不变、ledger结构/name/checksum/dirty精确校验、业务constraint及新RPC指纹readiness。
- [x] 4.2 安装互斥内核空库；未知/partial/dirty/checksum drift拒绝；并发安装和中途失败恢复符合已评审事务边界。
- [ ] 4.3 真UAT v44约束fixture执行完整核销G01-G15；记录实际平台role/端点/函数hash/source tree，不能用probe或本地PG16替代。

## 5. Writer和独立验证

- [x] 5.1 root按实际影响面执行Go ordinary/race、vet/build、smoke、owned/gofmt/secret/diff及OpenSpec strict；逐项区分跳过、失败和外部阻断。
- [ ] 5.2 两轴review和C/T/V/R门；本地候选只提交owned paths，完整SHA与clean receipt写仓库外。
- [ ] 5.3 不同verifier在干净detached worktree对精确SHA只读复验全部本change门禁。
- [ ] 5.4 真UAT命令receipt绑定最终candidate SHA、实际schema/RPC/权限指纹；任何变化使旧证据失效。

## 后续依赖（不计本change已完成）

其余fulfillment/identity/quote/paymentorder/refund/outbox/billing/import/config/snapshot适配，全runtime启用，5worker持续运行，完整secret容量与COS，PC同源托管和Mini UI3，以及95Case/25页面、恢复/告警/成本均未在本change交付。不在此勾选，后续按独立主能力拆change。完整UAT外部材料仍待客户/平台owner提供；本change不改Prod、不实际资金、不采购。

## Evidence

### 需求结构与审查

- change: adopt-cloudbase-shared-pg-runtime
  gate_type: W3
  ui_level_target: UI0
  ui_level_actual: NOT_RUN
  base_sha: 12164b115441cb2a50c96d05b3173f874042568e
  candidate_sha: NOT_CREATED
  phase: red
  command_or_action: test -f openspec/changes/adopt-cloudbase-shared-pg-runtime/specs/cloudbase-shared-pg-runtime/spec.md
  exit_result: 1
  sanitized_summary: 本轮需求工件尚不存在；仅是文档结构Red，不是业务行为Red。
  artifact_or_environment: isolated writer worktree
  unverified_boundary: 无业务实现、无真实平台核销或独立验证。
  external_asset: N/A

1.1证据：读取12164b1 fulfillment/postgresql.go、merchantidentity/authorizer.go、audit/postgresql.go、migrate/postgresql.go与CONTEXT，确认直接OrderID核销仍必须cipher.Open；原v20/v23有Go侧名称预检；记录在design。

1.3证据：2026-10-08只读review仓库外`shared-pg-platform-probe/probe.sql`，两表两函数固定命名、INVOKER与role allowlist、NOT NULL故障回滚；同意隔离探针范围，要求真实REST/exec通道role分别取证，探针不替代真实业务冲突映射。root实测exec-pgsql current_user为cloudbase_postgres_postgres_o7vmhmxa、claims NULL，不猜JWT身份；schema NUMERIC(20,0)支持完整uint64。

Go接口评审接受内部Client.Exec固定SELECT+绑定、8MiB限额、无自动retry；要求禁redirect、limit+1/尾随JSON校验、精确数字、结构化SQLSTATE脱敏、proof当前授权和已完成重放、不确定提交仅同key有限重试。接口方向审查不等于代码通过。

### 待执行的验收操作

实现文件与测试名落定后root将实际命令、exit和artifact逐项补充到本owner维护的证据中；不得将尚不存在的测试名当已可执行命令。已可执行规格命令为`openspec validate adopt-cloudbase-shared-pg-runtime --strict`及owned-path `git diff --check`。Go验证按真实受影响包使用仓库Go1.26.5 ordinary/race/vet/build；真实平台步骤须记录环境、SQL/function hash、角色及终态。最终完整SHA在外部receipt，避免为自引用SHA改动candidate。

需求结构Green：`test -f openspec/changes/adopt-cloudbase-shared-pg-runtime/specs/cloudbase-shared-pg-runtime/spec.md` exit0；`openspec validate adopt-cloudbase-shared-pg-runtime --strict` exit0（Change is valid）；owned-path `git diff --check` exit0。仅文档结构通过，业务Red/Green未运行。无需额外结构重构；保持一个主能力和四份权威工件。

### 真实平台阶段性证据（root部署者回传，尚非完整任务PASS）

2026-10-08，UAT隔离probe管理任务`task-cdc6610a`终态Succeed。root实际16同key：1 applied/15 replay；2^53+1及uint64最大值返回十进制字符串；receipt NOT NULL故障返回`DATABASE_23502`，订单版本不增、回执不增加，证实单调用回滚。REST APIKey的实际current_user=service_role，区别于exec-pgsql的环境suffix角色。

有效临时publish key调用RPC/表GET/PATCH均`DATABASE_42501`；同key调用exec-pgsql出现HTTP200且body `DATABASE_PERMISSION_DENIED`，实现必须新增错误体识别Red，不能把状态码200当成功。authenticated身份验证及临时资产清理仍待root回执；不整项勾选2.2/2.3。

探针时间返回+08:00，当前只证明instant保持，未证明UTC字符串输出。真实业务函数须固定TimeZone=UTC并验证微秒输出/偏移输入/非法时间；不得把本探针记作UTC格式PASS。所有上述仅隔离fixture的真实PG17平台证据，不是v44真实核销、候选或95/25验收。

真实平台追加已由需求owner只读核验`shared-pg-platform-probe/live-probe-results.json`：authenticated JWT的RPC/表GET/PATCH均HTTP403+DATABASE_42501；anon/authenticated均不是service_role成员。丢响应探针先DISCARDED_AFTER_PROVIDER_RESPONSE，再独立读到lost-response receipt，最后原key返回replay=true且version保持3，明确是提交后恢复。并发fresh为16请求1非replay/15replay；另first保留2^53+1、maxID保留2^64-1，不能混淆其不同fixture ID。临时资源cleanup仍待回执。

2026-10-08需求owner接受管理面单事务DO空库安装方案：同native bigint advisory key的xact锁，锁内核空库，44原SQL+ledger一个事务；失败全回滚，不保留旧runner的逐版本prefix恢复；partial/populated未知历史拒绝，RPC指纹另核。此为设计评审结果，安装实现/并发/故障仍NOT_RUN。业务proof设计追加NULL validated_proof，防receipt预读后账户scope变化形成解密旁路，具体DDL仍待review。

### 业务草稿第一轮需求审查

需求owner只读审查`services/api/internal/fulfillment/sharedpg.go`、`services/api/sharedpg/redeem_order.sql`及client。未改业务文件，未启动independent verifier（无candidate）。发现并交root修复：所有23505重试需收窄为命名receipt unique冲突，SQL局部捕获后必须重新RAISE以回滚整个调用；JSONB数值等价不能使小数order_id腐坏receipt绕过原Go严格uint64解码；nil context错误需保留原ErrInvalidInput。确认NULL validated_proof已用于skip-decrypt路径。FOR SHARE需UPDATE权限已由PG17官方SELECT文档核对，建议只授相关表单列UPDATE满足锁权限，而非全列账户修改。

安装接口review：SharedPostgreSQLInstallSQL()无运行时SQL/表名/fs输入，内部固定权威FS+原Load+schema_contract生成DO，方向通过。完整v44重复检查沿用native readiness允许额外未使用对象（RPC/独立无关表函数），但44结构/依赖/ledger漂移拒绝；初始空库对未批准对象仍拒绝。要求installer测试明确覆盖这两个路径，避免RPC安装后错误阻止v44幂等复核。

错误映射审查追加：RPC P1002才表示当前Order商户权限拒绝；底层42501可能为runtime缺表/锁/函数权限，原native映ErrUnavailable，不应伪装业务Forbidden。已要求root分离并加账号禁用与运行权限配置错误对照测试。

root回传首轮修复已由需求owner只读复核：nil context回归原InvalidInput；receipt INSERT局部只将指定constraint的23505转P1008并重新RAISE；Go只重试P1008；receipt order_id严格JSON number+正整数文本；merchant/audit只UPDATE(id)、order只必要列；42501不再映业务Forbidden。root报告本地真实44schema首7项TestSharedPG通过，包括16同key/不同key、native同故障回滚、提交后丢结果、无version变化的cipher竞态、SUBACCOUNT与禁用重放，具体命令/日志待追加，不冒充真实UAT。

隔离platform probe清理任务task-a01cf4de终态Succeed，root复读order_atomic_*对象=0；临时APIKey/user保留用于紧接的真实业务验收，因此2.4完整临时授权清理尚不勾选。RPC readiness草稿检查body/signature/INVOKER/UTC/search_path；需求owner追加proisstrict=false、volatility=VOLATILE、parallel=UNSAFE漂移负例，防NULL validated_proof下被STRICT配置跳过函数。

1.4设计/源码review闭合：sharedpg_install.go与native postgresql_schema.go逐项对照，固定FS/Load/44hash、单DO同锁、完整44允许unused对象的只读核验及原catalog谓词一致，未发现需先改的安装问题。业务函数初轮mustfix已落地；readiness新增STRICT/volatility漂移测试仍属后续实现验证项。此勾选只代表接口/DDL设计审查，真实平台44安装/核销与独立验证均尚未完成。


### 最终源码审查及本地切片证据（非候选独立验收）

需求owner复读`shared-pg-local-test/fulfillment-green.jsonl`，确认真实本地v44 fixture的12项TestSharedPG为12PASS/0FAIL/0SKIP，包括同key16、不同key16、同key跨订单16、回执故障与native同条件回滚、提交后丢结果重放、cipher无version变化proof竞态、SUBACCOUNT/禁用重放、42501映Unavailable、scope变化且坏cipher不写及小数receipt拒绝。初始undefined符号只是编译缺失，不能作为W3业务Red；42501映射与STRICT/STABLE漂移有实际行为Red→Green，精确摘录由root维护`red-evidence.md`。尚不以本地12项代替真实UAT或完整handler验收。

已核验安装writer外部`shared-pg-install/receipt.json`：ordinary27PASS、race27PASS、scanner相邻16PASS，均0FAIL/0SKIP；最终仅并发测试失败cleanup调整，针对同测试race1PASS。27包含子测试计数，不宣称27个独立顶层用例。覆盖真实pg_locks同native key互斥、v31后故障全批回滚/重试、完整44加unused对象重复不分配写XID且ledger/applied_at不改、7类未知对象与14种history/schema漂移。生成SQL410653bytes，两次完全相同，SHA256=e72cd4fb651c112babdbc0ceb5692bf42857b52d26207e255931b157857423fa；原44 SQL、旧runner/scanner/schema contract无改动。仅G11/G12本地层证据，不冒充CloudBase并发/故障层结果。

root回传真实UAT管理v44安装task-b6478198终态Succeed并读到44条ledger；逐44checksum、重复安装无写与真实核销正在验证，未据此勾选4.1-4.3。需求owner已放行本次核销DDL用于已授权UAT合成切片，审查SQL SHA256=d785cca943faa24c7eebeaabaa3693fc32244e17a2bdcbad290493f1507b5950；对应sharedpg.go=d4cd9e19cced838dbd9a369af00f1f7c2793b77a7c74f8f0d48af73823f0b78f、schema.go=9566ba1464771f543200e2b47c8255e5954c50ea7a0e3f691ecadb2e40d9fe91。函数readiness已含STRICT=false、VOLATILE、PARALLEL UNSAFE。源变化须重新核验指纹。

Spec轴precommit review：基线12164b115441cb2a50c96d05b3173f874042568e解析通过，HEAD无新提交；`git diff <base>`为空，改动实际全为owned untracked，已逐文件审查。未发现scope creep或修复后新增已证业务错误。尚需补client超时/取消行为测试（tasks3.1），以及exec网关实际runtime角色的DDL能力/凭据隔离证据（spec“业务端不持DDL管理员凭据”）；管理SDK与APIKey为不同凭据不等于数据库权限隔离。真实UAT业务、完整Writer Gate、candidate/独立验证与最终SHA绑定仍保留未完成。


### 运行权限真实失败与已选修正（取代前述APIKey运行入口）

需求owner亲读`shared-pg-platform-probe/business-runtime-receipt.json`：原子核销Go→真HTTP→真实v44已观察16同key1写15replay、16不同key1写15TransitionNotAllowed、提交后丢响应原key重放、uint64 max；仅scoped prototype。runtime实际是orders owner/public CREATE=true，受控DO成功CREATE后抛P1099 ORDER_DDL_ALLOWED_ROLLED_BACK（req0ac5d117-cc26-4ecf-a484-cfc7af5e44a5），复读无残留（req84bacb19-c48f-40a5-b897-693661cbb1f0）。因此G13运行身份无DDL能力硬门FAILED，不得形成candidate；先前源码/DDL设计放行只适用原型，不构成身份验收。

需求owner与root确认唯一修正为普通专用CloudBase用户JWT→REST RPC、INVOKER+固定service UID+必要DML的RLS。专用UID必要直接DML允许，普通用户UID与anon拒绝；Order Bearer不变。不保留exec双轨、不引入高权DEFINER。COS Provider去除broad APIKey独立change为完整runtime硬依赖，本切片不接runtime。所有入口/权限/函数指纹相关旧PASS失效并重验，原44安装事实与同条件原生业务合同证据保留。

- [x] 6.1 身份Red：记录owner DDL实测失败，验证普通专用用户signin/重复signin续期/可信UID/角色与管理入口拒绝；运行凭据没有DDL或角色提升能力。
- [x] 6.2 REST client Red→Green：仅固定allowlist CallRPC/named参数、真实jsonb形状、SQLSTATE/整数/null/时间、超时取消/限额/redirect/脱敏；移除运行Exec。
- [x] 6.3 RPC/RLS Red→Green：三函数可信role+UID守卫、必要列GRANT/策略USING与WITH CHECK、普通UID与anon无行读写、专用UID必要DML允许、禁止额外policy放宽、撤销UID后旧JWT拒绝。
- [x] 6.4 固定readiness RPC只读actual metadata，Go精确核验函数、RLS/ACL/role、service UID与v44；STRICT/DEFINER/policy/权限漂移负例。
- [ ] 6.5 原核销完整并发/故障/重放/权限矩阵经真REST专用JWT重跑并绑定新源/DDL/hash，清理临时身份与探针。
- [ ] 6.6 Writer Gate及新candidate独立验证；完整runtime保持禁用，COS去broadAPIKey依赖明确未完成。


认证实现收敛：需求owner接受client writer的singleflight重复signin续期，仅内存缓存access token+expiry，不保存refresh token、不实现refresh失败fallback。官方Auth OpenAPI支持该普通用户名密码登录；refresh本身会使旧refresh立即失效，仅作为已核平台事实。实现验收以重复signin UID/role恒定、到期/并发/取消/登录失败与重启重登为准，不强迫引入无必要旋转状态。此为内部工作负载身份策略，不改变Order用户会话契约。


### 存储合同复核（后续change，不扩大本次业务写入）

实际读取objectstore/service.go、handler.go、cos_adapter.go及web-admin/data/api.js：PC仅要求PUT并透传URL/headers，Mini读取object_key/url，不硬编码COS host/header。故撤回“仅换provider即破公开合同”的过强限制。下一验证路线是普通服务JWT+私有pgstore Adapter，保留DTO/intent-PUT-confirm/key/图片确认与私有读取；SDK有signed PUT/GET只证明接口存在。签名PUT真实≤15分钟、不可覆盖/并发、MIME/大小/摘要/真实图片、CORS、匿名/其他UID拒绝与baseline spec103/119的版本控制/非当前30日恢复必须实际通过。官方FAQ默认无自动版本管理，TTL/恢复仍未证，禁止放宽硬门或自制版本系统扩范围。COS-only CAM亦需管理员换票负测，不能先验视为安全供应方式。旧文段提到“COS最小CAM独立change”仅历史提案，由本节和design当前方案取代；完整runtime仍禁用。

新RLS草稿需求review已交root：role membership使用USAGE可能漏可SET ROLE的NOINHERIT角色；sequence必须精确核所有权限而非只USAGE；readiness缺实际v44ledger/约束核验。待writer修复并以漂移负例验证后再放行UAT定义。proof RawMessage/NULL与REST对象形状已复核正确。


### REST/RLS最终源码与UAT实验放行

需求owner逐读最新sharedpg/schema.go、redeem_order.sql、migrate/sharedpg_readiness.go及test、fulfillment/sharedpg.go及test、client。前三mustfix已闭合：MEMBER检查拒绝NOINHERIT可SET ROLE继承；sequence精确权限集合只允许authenticated USAGE；schema_migrations加入第四张固定UID RLS/SELECT-only表，readiness返回真实44ledger/history shape/column-default/constraint/index/sequence facts，由Go固定Load(FS)+schema_contract精确对照，UseNumber且尾随JSON拒绝。auth.role guard、RawMessage proof及NULL重放保持。root报告26类漂移/readonly/unused与installer相邻focused race通过，15项核销/权限本地测试通过；序列UPDATE和NOINHERIT均有真实行为Red→Green。详细执行证据由相应writer保存，需求owner不把其报告提升独立验证。

2026-10-08最终普通服务UID文本2108167831809454081真实signin成功，identity probe返回current_user=authenticated、session_user=cloudbase_authenticator、auth.role=authenticated、UID精确匹配、DDL SQLSTATE42501（reqbabfc7e0-85d9-4fd5-bcf5-02783b63636e）；复读无probe表残留。普通JWT调用exec合法SELECT1得到HTTP200 DATABASE_PERMISSION_DENIED，不能误算成功。其他普通UID与anon探针分别403/401+42501。重复signin返回同UID/role，旧token仍可调用probe，不能声明重新登录会撤销旧token。

/capi/credential同既有Provider的{env}请求：普通JWT返回INVALID_COMMON_PARAM/Appid missing且无凭据tuple；APIKey相同payload返回code0+tuple。该负例仅证明普通token不能经这一已用交换请求获得宽凭据，不代表正式IAM403、不代表穷尽全部换票/提权入口。函数probe只捕42501，若CREATE意外成功必抛P1099使整调用回滚；源码检查通过。

最终UID生成`shared-pg-platform-probe/uat-runtime-rpc.sql`已亲核209795bytes、SHA256=735bc4a205dbfbac8e1882f44d2fe664b2601f831f5471ba79a55030df793ea9、三固定函数、无marker残留，UID出现11处；与已审模板替换后前后逐字相同。需求owner明确放行管理pending20261008210100（原5 applied history不变）及随后普通JWT真实合成命令矩阵/清理。新身份业务Red为readiness缺失ready=false、order405旧admin guard返回Unavailable且订单仍READY/version2/receipt0（root req03c04b69）。新身份业务Green仍待实际运行，旧APIKey业务PASS不继承。

client旧receipt为ordinary69/race69（17顶层）0FAIL/0SKIP。Standards随后发现首次leader ctx取消影响其他waiter并缓存失败1秒，正在由client writer按确定性Red→Green修复；需求owner已见源码改共享独立15秒bounded认证context、各caller各自等待，但新ordinary/race回执未到，不能沿用旧69PASS为最终状态。此问题不阻断受控UAT RPC安装实验；candidate仍须完成修复与所有最终门禁。


client P2最终回执已由需求owner亲读：`client-rest-receipt.json` SHA256=d2a2455ea64ac31a78f6891d75d0baf7411fbfe5d7484654eed55fd635f33e2d，client.go=414f0af776dfa9614602bdbe34132f66c1909d270d67ba661abbe887212f3b9c。最终ordinary71/race71均PASS（19顶层，含子测试71），0FAIL/0SKIP，vet/gofmt通过；第9项决定性Red为leader取消污染健康waiter，修复后signin1/RPC2且后续缓存可用。所有caller取消时真实15秒HTTP认证deadline结束flight并可随后重登，不留无限等待。旧69回执明确superseded。ctx/HTTP timeout为既有正确传输机制在新REST的回归，不伪称单独行为Red；非决定性的偶然通过探针保留但不计Red。Client本地缺陷闭合，仍不等于真实平台业务矩阵或候选独立验证。


### G01–G15最终普通身份阶段准入矩阵

本表以需求owner亲读`shared-pg-platform-probe/business-runtime-receipt.json`的restIdentity/restBusiness*字段为准，旧APIKey字段不计新身份通过。最终管理task-88795f3c Succeed、SQL735bc4a205dbfbac8e1882f44d2fe664b2601f831f5471ba79a55030df793ea9、UID2108167831809454081；尚无候选完整SHA。PASS只代表表中限定层级，PARTIAL含未验项，不以本地代替真UAT。

| 门 | 当前真实证据 | 剩余准入边界 |
| --- | --- | --- |
| G01 固定调用/参数 | 本地CallRPC allowlist/named JSON及注入文本测试PASS；真REST参数命令成功 | 本地恶意文本与真实非法参数已归档；最终source/candidate绑定待完成 |
| G02 SQLSTATE/重试 | 本地精确映射/无transport重试PASS；真REST42501与exec HTTP200错误体实证 | 最终实际错误矩阵与同key有限重试证据绑定候选，不能称全部云错误已注入 |
| G03 类型 | 真REST uint64 max精确；fraction/overflow/null/zero拒P1001、2^63+1保持至SQL、非法time22007/infinity拒P1001、null proof P1005；+08:00.123456读回UTC12:40:00.123456 | scoped PASS；保留来源与最终candidate绑定，旧probe不混算 |
| G04 平台身份 | 最终UID/authenticated/非owner及DDL42501；其他UID proof拒绝、orders读/更新0行；anon拒绝；服务UID非必要DML/ledger写拒绝 | 三RPC/四表有效ACL/RLS已由实际readiness核验，代表性真实拒绝已覆盖；固定UID撤销旧JWT拒绝与恢复ready=true已闭合；capi同payload无tuple不代表所有提权入口均拒绝 |
| G05 当前业务授权 | 真REST OWNER/SUBACCOUNT及审计role正确；禁用和正确解绑后旧receipt Forbidden，恢复后Replay=true | scoped PASS；restCurrentAuthorizationCorrected实证；原23514失败保留，不计通过 |
| G06 token/proof | 真REST坏cipher与barrier确认proof预读后cipher翻转/解绑均拒绝，READYv2/receipt0，分别恢复 | scoped PASS；restProofCipherRace/restProofScopeRace；最终source/candidate绑定待完成 |
| G07 原子回滚 | 真REST临时receipt CHECK故障GoUnavailable，独立前后READYv2/receipt0/cipherSHA256一致；本地native对照PASS | scoped PASS；故障约束移除后原key完成，最终清理回执归档待完成。辅助raw请求错路径404及首次错列名42703不计证明 |
| G08 并发 | 真REST同key16=1变更15重放、异key16=1成功15拒绝；跨order同key16=1变更7重放8冲突，独立读唯一完成/receipt | scoped PASS，仍待最终candidate绑定 |
| G09 响应丢失 | 真REST丢响应原keyReplay=true；独立读订单/receipt仍一次 | scoped PASS，仍待最终candidate绑定 |
| G10 幂等状态 | 真REST跨order并发冲突、新key已完成拒绝、scope预读竞态和失效授权旧key拒绝 | scoped PASS；原有本地差分保留，最终candidate绑定待完成 |
| G11 v44合同 | 原44逐checksum匹配；真REST完整readiness PASS含ledger/columns/defaults/constraints/indexes/sequences | scoped PASS；实际metadata与新candidate source/hash绑定后才最终PASS |
| G12 安装恢复 | 本地27/26类及相邻race通过；真UAT重复ledger hash不变，实际同keygranted true/false等待已观察 | latest互斥holder504超时、waiter200，应记录“互斥/超时释放”，不能写两installer均成功；管理故障恢复证据与业务回滚分列 |
| G13 定义/有效权限 | 真REST readiness PASS、role/UID/ACL/RLS及schema事实Go核验；client P2普通/race71PASS | 固定UID撤销后旧JWT读改0行/GoForbidden/readinessfalse、恢复ready=true；清理探针/临时身份及复读已完成；完整runtime存储宽key问题仍硬阻断但不混同本切片客户端 |
| G14 公共合同 | Application本地native差分；新增8个handler场景同真实v44两实现，状态/body/cache-control对照 | 新增差分及最终affected ordinary/race各109PASS已亲读回执。reader为相同既有stub，orderquery未移植，不称完整HTTP后端 |
| G15 候选与独验 | 源/DDL hash与真实任务回执已有，candidate未创建 | Writer Gate收尾、精确SHA提交、干净detached独验、平台receipt最终绑定均未完成 |

阶段结论：新身份入口/原子核销的关键真实路径已成立，首切片尚未最终验收；95Case/25页面完整UAT仍未通过。云回滚、UID撤销恢复及proof竞态已具实证；清理和精确候选验证不予预勾。真实互斥后timeout仅如实记录，不把它扩展成业务故障或自动重试部署。


矩阵增量已亲读restTypes/restBadCipher/restSubaccountMicroseconds/restReceiptFaultVerified并更新G03/G05/G06/G07。root报告正确解绑、fault drop后原key完成、四表policy撤销后旧JWT读改0行/GoForbidden/readinessfalse与policy恢复进行中；回传与文件flush有时间差，暂按报告记录，不把文件中首次解绑23514或错误raw404伪作成功。有效权限全体覆盖由实际readiness的三函数body/ACL及四表RLS/策略精确匹配与真实正负代表请求共同证明，不机械增加所有表×所有方法的重复云写。剩真正proof竞态/撤销恢复/清理与最终candidate验收继续。

存储判断最终收敛：当前维持COS版本控制与非当前30日恢复目标，pgstore仅在平台原生恢复等价实证时才可替代；禁止另写不可变path+PG元数据版本系统来偷换bucket恢复合同。该结论取代此前“下一优先实施pgstore”的暂定方向，不改Prod、不新增用户选择；最小身份仍为完整runtime外部依赖。


最终affected writer回执已收：`final-writer-receipt.json` SHA256=d9821ae4f0e54585623bb7919f20695af613c01674869ff6639dd404e6eb9b7d；普通/race各101PASS（client71+fulfillment30，47顶层合计），0FAIL/0testSKIP，sharedpg包级no-test-files单列；focused migrate仅^TestSharedPostgreSQL普通/race各63PASS（10顶层），0FAIL/0SKIP。vet、全services/api host与linux/amd64 CGO0 build、fmt exit0；459项源码跑前后manifest hash33f4f12552ade304a7f049b9802e0659dcd3aad72fc200ff87c19b87050934df稳定。旧resetbase native测试NOT_RUN，不扩大其范围。此仍非完整Writer Gate或candidate/独验。

依已证实阶段勾选2.1-2.3、3.1-3.4、6.1/6.2/6.4：平台能力/真实最终角色、Client实际Red→Green与普通身份业务Red→Green、实际metadata readiness均有证据；旧Exec方案不据此恢复有效。其余涵盖撤销恢复、清理、proof竞态或最终生命周期的整项继续未完成。6.1的凭据隔离限定本client只持普通用户名密码+anon publishkey，/capi证据边界如前，不含未实现的COS Provider。


### 最终普通身份增量与候选前收束

需求owner亲读restCurrentAuthorizationCorrected（解绑req e55e646f-1471-40c8-8cb9-cf40f7ec4b84、恢复req 7ceac146-241f-447a-b69a-3e954cd6c979）；restUIDRevocation task-44afa6fc旧JWT读取/更新0行、proof P1002、GoForbidden且readinessfalse；restAuthorizationRestore task-7f1e696c ready=true。restProofCipherRace/restProofScopeRace均barrierObserved=true，明确proof读取完成后变更，拒绝且READY/version2/receipt0，分别恢复。restCrossOrders真Go16路为1fresh/7replay/8conflict、otherErrors0；req 6e66ec4a-c1d7-4ee9-81aa-036ff0d877e8复读401仍READYv2/receipt0，404 COMPLETEDv3/receipt1。原错误路径/错列/约束失败历史保留而不计PASS。

据此勾选3.5/3.6及6.3。4.1/4.2按本地严格漂移/全批故障恢复加真实UAT44checksum/readiness/只读重复/互斥证据勾选；不声称在已安装UAT破坏性重演全44回滚。G14新增源码已读，两个真实v44命令实现经过同handler的8种状态/body/no-store差分；读取投影是相同既有reader stub，orderquery仍未迁移。先前101/63回执是新增G14前的范围，最终affected ordinary/race须更新source manifest后才闭合5.1。

候选前必需：清理明确合成fixture/临时probe与临时身份、复读保留v44/三RPC/四表RLS/final服务UID及publishable key的有效权限；最终源码稳定manifest上的ordinary/race/vet/build/smoke与fmt/secret/owned/diff/OpenSpec；两轴review及仓库C/T/V/R评分附证据、零未闭合mustfix。仅上述完成可生成本地CANDIDATE，不需把候选后独验提前伪作已完成。完整SHA写仓库外receipt后绑定现有平台证据的实际source/DDL/权限指纹；若候选有实质变更必须重验受影响证据。随后另一个干净detached worktree对精确SHA只读独验，未独验不能集成；不为在提交内容中写自身SHA产生自引用循环。4.3/5.2–5.4/6.5–6.6仍保留未完成至其完整条件闭合。


### 规格冻结前最终核验

需求owner亲读最终cleanup：task-e31fe212 Succeed，req1ddbfd3f-8f19-4df8-b945-075af8e9aa18复读九类合成业务fixture计数均0、probe_functions=0、fault_constraints=0、ledger=44，Go readiness=true。临时user删除success1/failed0，临时APIKey删除且inventory剩余0；保留最终普通服务UID2108167831809454081、anon publishable登录凭据及三RPC/四表固定UID RLS。Keychain密码未读取或写入仓库。2.4据此闭合。

最终writer receipt SHA256=847209be366a79cd756b2f2486163b9b9f77c4dc786946aa63f79aa558a9cdc3，459项源码manifest=4b90341b147ecb7d4b027e822b0b168f7799cae2be0ecfff13c927f95e1ef20e，验证前后稳定。ordinary/race各109PASS（client71+fulfillment38、48顶层），0FAIL/0testSKIP；sharedpg包no-test-files另记。vet/fmt PASS；唯一增量为G14测试，migrate ordinary/race各63及host/linux build因对应输入未改保留，不机械重跑。root另回传smoke PASS。必要修正均已有相应Red→Green及最终受影响验证；无额外必要重构，3.7闭合。5.1仍待root最后owned/secret/staged diff检查，不提前勾选。

最终Spec轴：无新增missing/partial实现、scope creep或已证错误实现；范围符合“Go核销命令保留当前业务契约”“运行身份与数据库函数隔离”“验收证据按实际层级交付”。剩余partial仅生命周期：最终stage/secret/两轴收尾、候选SHA绑定、clean detached独验；完整runtime/读模型/UI3/95Case/25页面明确在后续，不偷换为本change PASS。

C9/T10/R9与当前切片证据相符。按docs/quality/change-quality-gates.md第10节，V8明文要求“已提交exact SHA、clean detached验证包完整但结果待verifier”，故提交前36分只能为拟定评分；候选提交及外部验证包形成后才能记录V8/总36。此不要求独验提前完成，V9/INDEPENDENT_VERIFIED与集成仍pending。需求工件于本段后冻结，后续候选完整SHA、评分终态和独验receipt保存在仓库外，避免自引用或为更新状态改变待验SHA。


最终stage收尾追加（仅闭合5.1）：root实测15个owned staged文件，git diff --cached --check exit0、未暂存diff为空；459源码逐hash重核仍等于4b90341b147ecb7d4b027e822b0b168f7799cae2be0ecfff13c927f95e1ef20e。需求owner亲读shared-pg-local-test/final-owned-secret-scan.json，15文件activeCredentialMatches=0，范围为实际password、anon publishable凭据和普通runtime access token；不扩大为任意未知secret已穷尽证明。smoke及OpenSpec strict PASS。5.2的Spec review与拟定C9/T10/R9已具证据，Standards最终delta及candidate/V8/exactSHA外部包由root提交时闭合；5.2整体、5.3/5.4继续pending。本段是提交前最后需求状态更新，root仅重新暂存本工件并完成最终检查；此后规格永久冻结，候选与独验终态仅更新仓库外receipt。
