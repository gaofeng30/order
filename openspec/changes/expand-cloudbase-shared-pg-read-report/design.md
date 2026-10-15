# 对账、后台报表、审计和营业状态设计

Status: APPROVED_REQUIREMENTS / LOCAL_FULL24_PASS / ACTUAL_UAT_DATABASE_GATE_PASS / CANDIDATE_NOT_CREATED。gate_type=W3；ui_level_target=UI0；ui_level_actual=NOT_RUN。

当前worktree已采用已独验第七精确父 ea76e34845a72b15334fb510ac2906e7bbe0ddb0，由root独占共同registry，使用统一105安装器与Check。四个开发临时fixture已移除，本地full24普通/race各2786 PASS、0 FAIL/SKIP；实际云105与两写族/权限/清理已通过；完整Writer Gate、候选与exact独验仍待完成。父采用与本地验证不表示main集成或上线。

## 单一路线与范围

继续 Go 业务编排 + ordinary JWT + 固定 SECURITY INVOKER RPC + 服务 UID RLS。第八完成四模块所有现有公开业务方法，保留 Handler/DTO/角色和各自口径。billing 不因当前 main 主要使用 worker 而删去其完整读取端口；adminreport 两写端口仍委托现有支付/退款服务，不复制金融状态机。不新增通用 SQL、跨 REST Tx、事务回调或 admin credential。

| 模块 | 原公开端口 | 数据库边界 | 最小迁移结果 |
|---|---|---|---|
| billing | Summary、ListPayments、ListRefunds、ExportCSV、RunReconcile | 三报表读取；两次独立系统账单查询；对账标记+SYSTEM_EVIDENCE 单事务 | 新 Shared service；复用 CompareBill/CSV 原算法；外部下载不在事务内 |
| adminreport | GetOrder、SearchOrders、Stats、Payments、Refunds、Summary、ExportCSV、ListPending、ProcessPending、RequestRefund | 当前 OWNER 检查后查询；订单/条目、统计等原多语句读取；末两方法只委托 CommandAdapter | 新 Shared application；保留既有 CommandAdapter，不增加退款/支付写 RPC |
| audit | Search | OWNER 查询和脱敏审计投影 | 新 Shared searcher；不实现泛化 ReceiptStore Tx 适配 |
| storestatus | Apply | 当前账号授权→storefront singleton FOR UPDATE→receipt→状态更新及 receipt 原子提交 | 新 Shared core；保留 OPEN/CLOSED/CUTOFF 和现有 fulfillment handler port |

第四已迁移 storefront 配置/Get、catalog/menu/pricing/soldout，以及 objectstore.Confirmed 读取；这些不是第八待重做模块。objectstore.RecordConfirmed 仍直连 Tx，必须与后续实际存储确认/不可变引用契约一起实现，不能用已有 Confirmed reader 冒充完整上传确认。身份、会话、支付、退款、生产推进和订阅分别由此前切片及第七闭合；main 仍 native 组装，不等于这些适配未实现。

## 不可变原契约

### billing
- 范围为非零 From<To、跨度≤366×24h；分页 1..100，cursor 为 id，满页即返回最后 id。Owner 先检查，missing 为 Forbidden、其他数据库错误 Unavailable。Summary 只计算 COMPLETED 订单收入，退款用 created_at 范围；不能与 adminreport 汇总合并。金额/count 扫描范围和 NULL/UTF8 错误保持原样。
- RunReconcile 的 limit 仅校验 1..100，不截断账单条数。下载及 provider 日期/条目校验在 DB 读取前；失败不写。系统账单按固定 +08 日界，先订单付款、再成功且 APPLIED 的退款观察；保留原两个读取阶段。CompareBill 按种类+商户单号匹配，比较金额/币种/状态/provider ID，忽略 OccurredAt；同键不同值双方各计差异，排序不变。
- persistReconcile 使用 date+NUL+digest 的 SHA，原 signed BE64 advisory xact lock、5s lock_timeout、既有账本先验；相同账单已处理返回，不重复投影或取新 audit clock。差异按原顺序查找目标，缺失跳过、去重并按 id 排序，先 prepayment 后 refund；只改原允许状态/version。pending 投影和 SYSTEM_EVIDENCE 同事务。Mismatch 是已提交事实后返回 ErrBillMismatch，不能当错误回滚。未知提交不自动盲重试，重跑靠同 intrinsic 去重。
- audit 时间原在投影之后独立 service.now 采样；不能偷用下载开始、tick 或账单日期。实现前明确保留该消费点的内部材料方案并建立真实差分，不能借本需求批准任意 clock 回调等价。
- ExportCSV 原按 100 分页并每页重新授权，完整积累；无 BOM、七英文列、UTC RFC3339Nano，不能替换为后台 CSV。

### adminreport
- 保留每方法原校验/授权/查询顺序；不把完整 merchant IdentityReader 的额外字段校验强加给原只读 OWNER 检查。GetOrder missing 为 NotFound。订单脱敏、中文状态/动作、取餐号补零、条目 flavors 非 nil 与 JSON 解码校验复用原逻辑。
- Search literal LIKE escaping、id cursor、日期/状态/Unclaimed 保留；Unclaimed 由数据库当前 Asia/Shanghai 日界决定。Stats、付款/退款日期使用原 IANA Asia/Shanghai 规则，不能统一替换为 billing 的固定 +08。退款 APPLIED 用最早匹配成功观察时间，其余用 created_at；可空 order/provider/operator 的原 COALESCE 不改。
- Summary 净额原 int64 转换/减法、SUM/count 扫描边界保留，不顺手改成新溢出规则。ListPending 原 quote/items 投影和重复 quote 映射行为以 native 为准。
- 读取没有原整段 SQL Tx；保留多语句阶段及可见性，不凭“更一致”增加锁或宣称 native 本来就有单快照。实现用固定 RPC 封装各方法，必须测试 header/items 并发时的实际语义，避免把预期未确认的变化写成等价。
- CSV 有 BOM、五中文列、原时间 Format(RFC3339) 和金额格式；pgx 扫描 time.Local 与 REST UTC Location 的 Handler/CSV 字节须差分。ProcessPending/RequestRefund 保留现有 adapter 错误映射、当前权限和 Sixth/Seventh 幂等。

### audit / storestatus
- audit OWNER 查询任意 scan/query 错误原映射 Forbidden；不能套别模块 Unavailable。To 是严格小于日期+' 23:59:59.999999'，不改为包含最后微秒。应核 native 会话时区与固定 RPC UTC 的日期转换，直接比较日期边界。只返回脱敏字段，request hash 前 12 字符，不返回 receipt/request evidence/private JSON。原 uint16 Limit 不可偷收紧到仅 HTTP 默认值。
- storestatus 命令校验先于 dependency，clock 每次尝试在 BeginTx 前 UTC 截微秒、零值失败，包括旧 receipt。当前账号和 singleton 锁、当前状态有效性检查在 receipt 前；OWNER/SUBACCOUNT scope/role/auth snapshot、原 hash/请求摘要、1024B 严格 receipt decoder 保留。无变化也写 UNCHANGED receipt；改变时只改 business_status，不添加配置 version 更新。
- Apply 原最多两次：40P01/40001/55P03 及原 merchantidentity.ErrUnavailable 类别保持封闭映射；不把所有 transport/未知 commit 归 retryable。旧键重放仍校验当前授权，同键异状态冲突。账号→singleton→receipt 锁序及 rollback 不变。

## 冻结的固定 RPC ABI

本稿从完整 native 源码重新清点，最终确认为 15 入口，不以旧提议数量作为依据：billing 6，adminreport 7，audit 1，storestatus 1。CSV 保留 Go 分页循环，不新增 CSV RPC；金融命令仅调用 Sixth/Seventh，不增加财务写入口。条件总 registry 90+15=105，父切片及源清点未完成前不是安装事实。

```text
order_billing_summary_v1(p_owner numeric,p_from jsonb,p_to jsonb)
order_billing_payments_v1(p_owner numeric,p_from jsonb,p_to jsonb,p_after numeric,p_limit integer)
order_billing_refunds_v1(p_owner numeric,p_from jsonb,p_to jsonb,p_after numeric,p_limit integer)
order_billing_payment_facts_v1(p_from jsonb,p_to jsonb)
order_billing_refund_facts_v1(p_from jsonb,p_to jsonb)
order_billing_reconcile_v1(p_date text,p_digest_hex text,p_matched numeric,p_provider_only jsonb,p_system_only jsonb)
order_admin_report_get_v1(p_owner numeric,p_order numeric)
order_admin_report_search_v1(p_owner numeric,p_state_hex text,p_date_hex text,p_query_hex text,p_unclaimed boolean,p_after numeric,p_limit integer)
order_admin_report_stats_v1(p_owner numeric,p_at jsonb)
order_admin_report_payments_v1(p_owner numeric,p_from_hex text,p_to_hex text,p_after numeric,p_limit integer)
order_admin_report_refunds_v1(p_owner numeric,p_from_hex text,p_to_hex text,p_after numeric,p_limit integer)
order_admin_report_summary_v1(p_owner numeric,p_from_hex text,p_to_hex text)
order_admin_report_pending_v1(p_owner numeric,p_after numeric,p_limit integer)
order_audit_search_v1(p_owner numeric,p_action_hex text,p_target_hex text,p_from_hex text,p_to_hex text,p_after numeric,p_limit integer)
order_store_status_apply_v1(p_actor numeric,p_desired text,p_operation_hash text,p_request_hash text,p_request_digest text,p_now jsonb)
```

所有 RPC 是固定服务 UID、SECURITY INVOKER/固定 search_path 的私有实现入口，业务 actor 仍是 p_owner/p_actor。共用 authenticated 不等于经营 OWNER。输入并非任意查询/表名/SQL。标量 uint64 以规范十进制字符串传 numeric，按消费点检查整型 0..18446744073709551615；cursor可0，业务id是否允许0按原入口，不统一加非零条件。

Time jsonb固定 {epoch_seconds:十进制text,nanosecond:0..999999999,zero:boolean,canonical_utc:text}，同一 Go time.Time 生成。Go原validRange/zero/先后校验在原位置；SQL只在该原SQL语句使用参数时按原pgx有符号整数µs编码转换，含int64 wrap、PG范围及±infinity原scan失败，不使用float epoch。不得在OWNER检查前因输入Time不可PG表示而改错误优先级。p_from_hex等为原Go bytes的lowercase偶长hex：保留非法UTF8/NUL，消费时才按原文本参数编码失败，不让JSON替换为UFFFD；日期字符串没有新白名单，原SQL合法解析和错误保留。业务日期与会话UTC按实际 native连接配置核验，audit原To排除最后1µs边界必须真测。

p_provider_only/p_system_only 是 CompareBill原排序结果的两个数组；每项固定 {kind,out_trade_no_hex,out_refund_no_hex}，仅传projectPendingFacts真正消费的目标字段，providerID/OccurredAt/金额不在此阶段重验；它们已经参与原Go CompareBill，不能SQL重写比较算法。数组元素顺序、两个数组长度和concat顺序保留。p_date为normalized date.Format原串，p_digest_hex原32bytes；p_matched为原uint64。SQL按date+NUL+digest重建intrinsic及signedBE64 advisory锁，严格shape在对应消费位置检查；不添加行数或总bytes业务上限。原validBillEntries未限制provider输入总量，管理/网关请求容量需真实可表达大账单验证；超限不得截账单或部分投影。

StoreStatus的operation/request/digest沿原Go私有算法，p_now每次尝试前UTC截µs、zero失败，包括旧receipt；单RPC当前授权→singleton锁/状态→receipt→update+receipt。只对原40P01/40001/55P03和确定发生于原Authorizer的merchantidentity.ErrUnavailable最多2次；固定tag AUTHORIZATION_UNAVAILABLE用于后者，不把网络/未知commit/任意DB错误都塞入此tag。business outcome严格封闭，成功响应为原Result，receipt JSON仍1024B严格decoder。管理员输入不能作为平台服务UID。

### 读取快照、错误和投影

每个read RPC为VOLATILE，按原原生语句顺序执行多个独立SQL statement并在进入下一statement之前完成前一结果原Scan所需的null/type/范围/时间检查。不要用一个JOIN或大CTE将原多个statement强制单snapshot，也不增加FOR UPDATE。Header→items、Stats三阶段、Summary多阶段、pending header→quote items均保留顺序；须用真实并发barrier证明后续stage可见已提交变化，不能仅从VOLATILE声明宣称等价。原Go rows/scan错误后不执行下一stage。RPC输出raw事实，Go复用原stateLabel/actions/mask/flavors/CSV算法；不要重复JSON归一化原私有receipt。

SQL中包含原OWNER读取，不另调用完整IdentityReader；native查询错误映射必须区别：audit owner任何错Forbidden；billing/adminreport仅无行Forbidden，其余Unavailable。已授权后并发撤销的当前请求按原分阶段事实处理，后续请求必须拒绝，不擅加每个阶段重新鉴权。合法Go方法输入Limit为uint16，包括某些未做非零/≤100校验的直接调用，原Handler限制不是全部公共Go限制。原audit直接Limit=0的len==0分支存在索引边界，先以native实证记录，不借迁移顺手修正或把panic猜成业务错误。

金额全uint64，count/quantity分别按原目标uint32/uint64、折扣uint8检查，不改成统一numeric宽容读；SUM越界仍Unavailable。adminreport NetCents在Go按原int64转换减法，保持wrap。数据库读取time.Time按原driver time.Local还原；billing CSV明确.UTC，后台CSV保留原Location的RFC3339。nullable/empty slices遵循原方法（Items可nil、flavors不得nil、ProductSales空时nil），不一律补空数组。Pending重复quoteID的index map最后项得items保持原结果。

### 对账时钟与提交

原billing.now为私有默认wall clock，无公开WithClock。此处批准与第七相同的唯一最小宿主调整：SQL在projectPendingFacts全部成功之后、audit INSERT消费位置采一次clock_timestamp，保留独立时钟及晚采样，不声称任意测试注入clock等价。replay先验成功不采新clock。各pending UPDATE各自clock_timestamp仍独立。该调整不新增参数、配置、材料预读循环，也不释放advisory锁后重做projection；必须真实锁barrier证明取时晚于被阻塞projection完成。审计精确ns在落库仍原µs，不把多个clock强行相等。

已处理intrinsic仅return成功（Go仍返回本次CompareBill结果以及原Mismatch）；matched也写SYSTEM_EVIDENCE。未知commit返回原Unavailable，不自动provider重下载/重投影；调用者重跑由同intrinsic避免重复。并发不同digest同日仍是不同intrinsic，不新增日期唯一规则。缺目标跳过，原任意UPDATE affected=0接受行为保持。

### 成功响应容量冻结

Growing HTTP200名单：两billing facts；admin get/search/stats/payments/refunds/pending；audit search。理由分别为无条数上限、合法产品/联系人/商户姓名/快照增长、直接uint16 Limit；Top10不意味着name字节受限。其余六入口 billing summary/payments/refunds/reconcile、admin summary、storestatus apply 保持现有8MiB成功界：summary只定量标量，billing分页100且原持久订单/退款标识有受限健康schema与写入来源，reconcile只封闭状态，storestatus小固定DTO。必须将原schema标识长度证据写入实现回执，如发现健康native来源突破则先纠正分类，不追加业务限制。错误/signin仍bounded；所有成功严格singleJSON/UseNumber，growing不等于任意RPC取消上限。CSV Go完整汇聚、分页每页重新OWNER检查，无新导出条数/金额/姓名上限。

## 所有权和最小验证

需求 owner：本change四OpenSpec文档。billing writer：internal/billing/sharedpg.go、sharedpg_test.go、sharedpg_reconcile.go、sharedpg_reconcile_test.go、sharedpg/billing.sql。report writer：adminreport/sharedpg.go、sharedpg_test.go、sharedpg/admin_report.sql；audit/sharedpg_search.go、sharedpg_search_test.go、sharedpg/audit_search.sql；storestatus/sharedpg.go、sharedpg_test.go、sharedpg/store_status.sql（均 services/api/ 下）。root/common：internal/cloudbasepg/client.go、client_test.go、sharedpg/schema.go、schema_test.go、access.sql、必要统一 fixture internal/fulfillment/sharedpg_test.go。这是预分配清单，源文件抽取若确有必要须逐 path 重新分配；不默认授权修改所有 native 文件。main/worker 接线归后续独立 runtime change。

1. 各公开方法 native/v44 成功与失败先真实 Red，随后最小 Green；复用原 Handler 验 DTO、CSV 字节、脱敏、uint64、NULL/UTF8/日期/host Location、当前授权。不以缺函数测试替代完整业务 RGR。
2. 对账匹配/不匹配/重复账单、provider 在 DB 外、同 intrinsic 16 并发、commit 后丢 ack、晚 audit 故障全 rollback、跨支付/退款状态只投影合法目标；取消/超时保留具体层证据。Storestatus 三状态、同/异键 16 并发、禁用账号旧 receipt、零 clock、晚 receipt 故障、未知 ack 恢复。
3. 报表按实际对账口径建立交叉反例，不能要求两种口径数值相同；CSV 多页重新授权，真实 growing response 走 Client+Handler。避免没有原业务依据的新额度/校验。
4. 全 registry/精确 body/ACL/RLS/readiness、原 44 ledger、仅必要 UPDATE/INSERT/sequence 权限；role 任意提升与外部关系授权 fail closed。目标 tables/seq 从实际 SQL 清点，不预授未来权限。
5. 统一 installer 下 affected ordinary/race、static/smoke/strict，ordinary UAT 全正入口、负角色、写入原子性与增长硬门、合成数据清理/源绑定；exact candidate detached 双轴+runtime独验。当前完整验收尚未完成；各真实子门只在tasks取得回执后更新。

## 完整 runtime 仍欠的接缝

第九/后续必须单独完成 main 依赖组装、Shared identity/session/phone/provider recipient、既有 CommandAdapter、订阅固定退款依赖、迁移/bootstrap 管理面和 ordinary runtime 分离。对象存储只读 Confirmed 已有，RecordConfirmed 写入、普通最小凭据、确认后字节不可变引用、私有 URL ≤900s、原生非当前版本 30 天恢复仍需独立完整证据。旧 COS 桶曾实测未开启 versioning/lifecycle；不能把旧 signed PUT409 当版本开启后不可覆盖证据。

CloudRun 五常驻 worker 保持原四个 1 秒循环和 billing 1 小时 ticker；billing 原 cloud-only、无立即首跑，不换分钟触发器。保留已测 EnvParams 5KiB 限制，真实最终 map/秘密交付/CPU常驻及终止 lease 需宿主实证。PC same-origin、Mini 真 UI3、监控告警、备份/恢复及管理责任与真实微信/支付材料继续后续门禁。已有 CloudBase 资源优先；不采购、不把原 native TLS/CVM 当必需前提，也不把 SDK/RPC 可用称全 runtime 已部署。95 cases/25 pages、真实 provider、UI3、main 集成均未通过。

## 固定构造与公共 caller

新增 `billing.NewSharedPG(client, provider BillProvider) *SharedService`，显式实现原Application五方法、私有默认now宿主调整如上；不嵌入native Service以假装db非空。`adminreport.NewSharedPGApplication(client, commands CommandAdapter) *SharedPGApplication` 显式十方法，ProcessPending/RequestRefund原样委托；nil commands仍Unavailable。`audit.NewSharedPGSearcher(client) *SharedPGSearcher` 仅Search。`storestatus.NewSharedPG(client, clock func() time.Time) *SharedPGCore` 仅Apply，固定锁内authorizer逻辑不接收假的Authorizer(*sql.Tx)。原构造不改，main接线后续第九负责。

实际caller为cmd/order-api/main.go中billing worker、adminreport handler及admin command adapter（其GetOrder读取）、audit handler、fulfillment依赖storestatus Apply。不因billing五方法暂未全部暴露HTTP而漏实现。任何必要纯投影提取须单独明确native源路径ownership后做最小抽取，默认新文件实现/复用现有私有函数。

需求审批已完成；当前开发进行中，完整business/platform/Writer/candidate/independent尚未完成。旧初稿逐字快照 eighth-original-read-report-snapshot/；final文件单独保存，不修改旧文或第七/第九。若新的native决定性反例证明此固定wire改变原错误/消费顺序，先纠正需求再实现，不能在代码里暗加fallback。
