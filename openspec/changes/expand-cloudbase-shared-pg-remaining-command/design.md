# 共享 PG 剩余命令设计

Status: APPROVED_ATOMIC / IMPLEMENTED / LOCAL_VERIFIED / UAT_SYNTHETIC_VERIFIED / CANDIDATE_PENDING / INDEPENDENT_NOT_RUN。gate_type=W3；ui_level_target=UI0；ui_level_actual=NOT_RUN。

权威来源为 external `architecture-ablation/seventh-remaining-command-requirements.md` SHA 9c0390ea1a464969d868a4ab5c95f267a3d02232ea18259ef1b38543ceb3a933，JSON SHA39f5823e88a7f884dd54756e47061036274171720a73e209167fa687fb1b039b。父 exact dde945e3fd1eb8114935cfc40a46606ab6c39c9d已独验；本设计批准不等于实现通过。

## 最小完整范围与顺序

完整refund.Application五方法RequestOrder、RequestPaidPrepayment、IngestRefund、RunDue、ListPending；subscription.RecordConsent/RunDue及退款成功的事务内enqueue；orderadvance.RunProductionDue。orderproduction纯状态算法已存在，不另造“第七状态”或独立无人领取业务。用户未取餐如何退款保持既有原入口/原状态规则，不新增自动退款政策。

推荐本change同时闭合三者，原因是退款成功与notification consent/outbox同一事务不能拆开部署；推进与退款/核销竞争同一orders状态。最短实现顺序：原native基线→订阅consent/claim/finish固定端口→Refund intent/claim/finish/observe含固定enqueue片段→Orderadvance due/advance→统一registry/原handler与worker接口/平台合成矩阵。不得先以单退款意向接口接完整runtime。

## 逐公共方法及原事务事实

### Refund

- RequestOrder先service.valid/meta/reason校验，采service.now.UTC（原不截微秒），然后事务外orders locator一次；2次原可重试错误尝试必须复用同locator，不重读以改变竞态。事务锁当前OWNER（仅MERCHANT）→prepayment→order，核locator.user与amount、USER本人隐藏NotFound，再既有refund+receipt，之后才状态/自助取消严格pickup>now+30min。merchant仅原RESERVED/PREPARING/READY_FOR_PICKUP/COMPLETED。outRefundNo固定ORDER_REFUND_<prepaymentID>，无随机；intent+order REFUNDING/token ciphertext及keyversion清空+merchant/user审计receipt同提交。不要额外清除原未清字段。
- RequestPaidPrepayment只MERCHANT；OWNER→payment锁→既有refund/receipt优先→仅PAID/PENDING_MANUAL新退款，无order_id时SQL NULL而非0；原回包OrderID0。不能强制已有订单或把NULL吞成bigint。
- 两入口既有READY意向仍可claimCreate；其他replay状态直接返回。owner熵在intent提交后、claim之前，失败可能保留READY意向。claim只READY→CREATE_CLAIMED；known commit才CreateRefund一次。30s租约是原常量，不新增配置。finishCreate锁refund核state/kind/owner/version/expiry；若callback已赢，仅按当前行refresh唯一request receipt。provider Create即使SUCCESS也只记PROCESSING，必须观测才能应用。provider错误/不匹配可持久CREATE_UNKNOWN/PENDING_MANUAL，不能一律rollback。
- IngestRefund在service层先validVerifiedRefund（CALLBACK event1..128、ACTIVE_QUERY event空、provider facts原校验），采now后事务外按outRefundNo定位refund/prepayment/nullable order一次；claim目标不同为私有stale lease。事务锁payment→可选order→refund→观测/应用。匹配顺序merchant/outtrade/transaction/outrefund/refundID/amount/currency/state/time保持。观测dedupe是原Go json.Marshal{source,event,omitempty,Refund ProviderRefund原导出字段}后SHA，不是第六NUL算法；保留时间offset/ns与Marshal错误被原忽略的事实，不能顺改。
- 重复refund observation在带claim时释放lease/原nextQuery，并按原APPLIED返回，不重复应用。新成功同Tx更新order REFUNDED或prepayment REFUNDED_WITHOUT_ORDER、refund/observation、固定refund-result outbox/consent及provider evidence。未匹配/处理中/关闭按原pending分支，不把SUCCESS降为CLOSED。未知commit先读事实，不能自动重发CreateRefund。
- RunDue一轮owner熵一次；due截止now与实际leaseNow分开，先尝试每id claimCreate，成功则Create+finish并Pending++；否则claimOneQuery再Query，finishQuery成功走同observe事务，provider错误走单独release。Query stale lease返回false,false,nil（非Unavailable）。原没有的统一retry不加：request/ingest最多2次三SQLSTATE，claim/finish原类别不擅扩。回包计数/ctx取消保留部分result。
- ListPending先事务外当前OWNER，再原id/filter查询；不要擅改成需要锁的快照或新的授权时点。页1..100，原字段完整含ProviderRefundID/PendingReason。

### Subscription

- RecordConsent原service缺store→InvalidInput，actor0→Unauthenticated，之后meta/key128/request64/输入校验，再采独立now。先按user/key读consent并match(order/kind/decision/templateVersion)，命中不读order、不读audit receipt。新写锁order核本人→最新grant_sequence FOR UPDATE→防MaxUint64→consent+audit同Tx。两次三SQLSTATE；仅uq_notification_consents_user_idempotency/uq_action_audits_command_receipt的23505走原重读，不能泛unique。
- EnqueueInTx不是独立runtime RPC。它必须内嵌到业务事务：锁order/recipient→最早ACCEPTED grant（即使已consumed也不能跳下一个）→outbox精确consent/recipient/template/message语义相等则幂等；无consent返回nil；consumed但无outbox拒绝；insert outbox+consume同Tx。退款Message固定OrderNumber/REFUNDED，其有效性在调用enqueue前，不能因无consent跳过原validIntent。已有履约MarkReady规则不重写，不改变其独立enqueue clock。
- RunDue原owner一次、每次claim恰好1条，避免排队耗尽后面通知的租约；due cutoff固定、本次leaseNow由Go每次预采（不能照第六claim改锁后），30s租约；FOR UPDATE SKIP LOCKED，attempt uint16溢出拒绝。claim提交后SendSubscription，完成后重新now；success/临时/永久三CAS必须state/owner/version/liveexpiry，RowsAffected!=1仍Unavailable。临时retry=max(completedAt,run cutoff)+1min，保证未来cutoff同轮不反复发送；无provider自动retry或虚构exactly-once送达。
- provider RecipientResolver实际由identity.Repository.FindPhoneUser提供OpenID（wechat_subscription_provider.go:173–208），TemplateResolver静态版本配置，token/provider外调仍Go；后续runtime接线复用第三PhoneStore的FindPhoneUser，保持原错误映射及OpenID1..128校验，不新增读取全部用户的权限或新模板配置源。真实template/微信订阅授权/投递到达另验。

### Orderadvance

唯一RunProductionDue：ctx nil/zero now/limit1..100优先InvalidInput，service/db缺失后Unavailable；now UTC截µs。due按RESERVED和pickup_at<=now+30min、pickup_at/id排序，先收集全ids再逐项原两次DB retry。每项Tx锁order，missing/非RESERVED/noDue无改；now早于materializedAt/version0拒绝；PREPARING变更+SYSTEM_EVIDENCE同Tx，scope=SHA256(SYSTEM:ORDER_PRODUCTION)，原before/after时间JSON字节。中途错误返回已完成Scanned/Advanced部分结果，不把整批强改全rollback。保留1秒tick的补偿扫描语义，不换成分钟cron。

## 批准的固定RPC ABI

退款locator是既有只读业务事实接口，用来复用原跨attempt冻结facts；不是泛查询或跨HTTP锁。其结果只来自固定RPC，客户端无权传任意SQL。每个写RPC完整自身原事务，不拆claim与更新、观测与outbox。

```sql
order_refund_order_locator_v1(p_order numeric)
order_refund_request_order_v1(p_actor_kind text,p_actor numeric,p_locator jsonb,p_reason_hex text,p_key_hash text,p_request_hash text,p_now jsonb,p_notify_url_hex text)
order_refund_request_paid_v1(p_actor numeric,p_prepayment numeric,p_reason_hex text,p_key_hash text,p_request_hash text,p_now jsonb,p_notify_url_hex text)
order_refund_claim_create_v1(p_refund numeric,p_owner_hex text,p_now jsonb)
order_refund_finish_create_v1(p_refund numeric,p_owner_hex text,p_version numeric,p_request jsonb,p_provider_error boolean,p_observed jsonb,p_now jsonb)
order_refund_read_v1(p_refund numeric)
order_refund_observation_locator_v1(p_out_refund_no_hex text)
order_refund_observe_v1(p_locator jsonb,p_source text,p_event_hex text,p_observed jsonb,p_dedupe_hash text,p_claim jsonb,p_now jsonb,p_enqueue boolean)
order_refund_due_v1(p_cutoff jsonb,p_lease_now jsonb,p_limit integer)
order_refund_claim_query_v1(p_refund numeric,p_owner_hex text,p_cutoff jsonb,p_now jsonb)
order_refund_release_query_v1(p_refund numeric,p_owner_hex text,p_version numeric,p_now jsonb)
order_refund_pending_v1(p_owner numeric,p_after numeric,p_limit integer)
order_subscription_consent_read_v1(p_user numeric,p_key_hash text)
order_subscription_consent_record_v1(p_user numeric,p_order numeric,p_kind text,p_decision text,p_template_version numeric,p_key_hash text,p_request_hash text,p_now jsonb)
order_subscription_claim_v1(p_cutoff jsonb,p_lease_now jsonb,p_owner_hex text)
order_subscription_sent_v1(p_outbox numeric,p_owner_hex text,p_version numeric,p_provider_message_hex text,p_completed_at jsonb)
order_subscription_temporary_failure_v1(p_outbox numeric,p_owner_hex text,p_version numeric,p_code text,p_completed_at jsonb,p_next_at jsonb)
order_subscription_permanent_failure_v1(p_outbox numeric,p_owner_hex text,p_version numeric,p_code text,p_completed_at jsonb)
order_production_due_v1(p_now jsonb,p_limit integer)
order_production_advance_v1(p_order numeric,p_now jsonb)
```

共20新入口：refund12、subscription6、production2；若第六70最终不变则目标90，仅规划数，不预授ACL/RLS/sequence。所有函数INVOKER/固定服务UID/UTC/search_path；no helperRPC/SECDEF/管理key。原schema44/ledger不动，按实际语句逐列最小权限与extra grant/readiness检查。

### 批准的私有wire与消费边界

- ID/version为numeric规范decimal strings，全uint64，只有原nullable/after/sentinel可0；order_id NULL由closed locator保留bool+value，不经signed bigint。金额退款原uint64，不照第六int64截断；实际schema约束另保留。
- p_now/p_cutoff/p_lease_now/p_completed_at/p_next_at为固定Time对象wire（epoch_seconds text、nanosecond整数、zero boolean、原canonical UTC text），Go同一Time同源。Refund/Subscription原不提前truncate，比较与fresh receipt可能需要ns，PG仅持久阶段按原pgx截µs；不能把第六truncate规则照搬。production虽已truncate仍需保留原Go时间/JSON事实，两个入口现使用jsonb，见后文真实反例修订。固定Go codec从同一time.Time生成上述字段，SQL不自造Gregorian一致性算法；持久字段仅在原消费位置转PG时间，未消费字段不提前转换。原生ns/negativeepoch/hostLocal/fresh-vs-replay差分是实现硬门，不加输入cap。
- p_observed固定ProviderRefund所有原字段；字符串hex、Amount/Total全uint64text、SuccessTime保Go原offset/nanosecond与zero，不允许任意JSON业务。p_dedupe_hash由原Go原Marshal计算（包括原忽略Marshal error后的sha(empty)），SQL受信固定Go事实不重排jsonb。p_request为claim时锁内实际canonical provider request，同样typed闭合；finish晚时只能比较原claim事实，不重新读另一个request来改mismatch。
- request_order p_locator来自固定初次RPC(id/prepayment/user/pickup)，每attempt复用；observation locator来自outRefundNo固定查找、每retry冻结退款/预付款/nullable订单。SQL最终原锁与核验不能省略。pending仍原两条statement权限→数据，不引入额外current-auth锁。
- p_enqueue仅表达既有可选退款通知依赖。Shared refund内部构造保留私有启用位，由组合层是否连接固定Shared订阅实现决定；它不来自环境变量、HTTP或业务配置，不新增通用notifier接口。nil依赖不enqueue，非nil连接只能是本项目固定订阅实现，组合层不得把不可用依赖冒充nil。原任意*sql.Tx callback不是REST可执行端口，不宣称远程执行用户回调。已亲核cmd/order-api/main.go:226正式组装唯一为refund.New(...).WithNotificationEnqueuer(newRefundSubscriptionAdapter(notificationService))；subscription.Service.now为私有字段且无WithClock。原native接口不删改，本切片只替换这条实际组合的数据库实现。
- 原subscription.EnqueueInTx先validIntent，再独立service.now().UTC，然后进入store读order/consent；无consent也已采时。固定SQL在相同有效通知分支入口、order/consent读取之前采clock_timestamp，created_at/consumed_at使用该独立时刻，next_attempt_at仍refund调用的AvailableAt。这是明确的时钟执行宿主调整（Go默认wall clock→DB wall clock），保持采样位置与独立性，不声称任意私有注入时钟等价。不把时刻提前到Go请求前、不只在找到consent后采、不复用refund now；无有效intent/无通知依赖/无order不采此clock。原message验证失败、冲突outbox/已消费无outbox及约束失败仍整体退款回滚。真实默认clock锁barrier与时间因果/readback必须验证，不能靠固定相同timestamp制造等价。
- subscribe claim原store可接limit但public RunDue总1；固定新RPC只claim一条，保持SKIP LOCKED与完整消息验证，原internal任意limit测试不扩大公共功能。sent允许合法UTF8<=128 ProviderMessageID包括原可为空/控制字符情况，不能添加Go原没有的过滤。
- 错误只闭合业务tag/原三SQLSTATE+精确named unique；退款私有stale lease单独tag供finishQuery映射false,false,nil。Exception包住全部当前业务写保证late failure rollback，不能误rollback原durable pending/defer。无从error message解析私密字符串。

## 成功响应容量与时间裁定

20入口全部保留现有8MiB成功响应上限，不增加growing白名单。依据是实际公开写入链路，不是只数行数：refund请求最多一份Refund/receipt；claim最多一份ProviderCreateRequest（64/64/64/80/2048原校验），read/finish仅一行；pending最多100行，provider_refund_id受v36/v37 BYTEA<=128、reason/pending VARCHAR64与原输入约束，金额/ID固定uint64；due最多100id。subscription consent单行、claim恰一条，合法message64/256等固定字段、模板/计数固定数字，finish仅结果；production100id/boolean。用hex/JSON转义后的最坏合法输出仍远低于8MiB。原provider输入字符串可很长，不等于成功响应可回显全部输入；SQL不得在错误/诊断里返回原provider材料。

不新增业务长度限制。退款receipt用原json.Unmarshal（未知字段接受、非法JSON拒绝），不能套第六1024严格decoder；subscription message原json.Unmarshal同理。前述容量结论针对原公开接口可写出的正常持久状态；不是声称SQL手工扩入任意巨大未知字段也有相同上界。实际writer如发现合法原API能产出超过8MiB响应，必须先记录原生反例并调整对应容量分类，不能裁剪/分页/缩短数据。

外部seventh-time-json-probe.go/log在现有Go1.26.5实际执行成功：+08:00与123456789ns原JSON逐字保留；year1000000/year-1 Marshal返回error且raw为空，SHA为空字节摘要；zero time编码0001-01-01。此为Go原语证据，不是refund数据库/接口PASS。refund validProviderRefund不对非SUCCESS的SuccessTime增加范围限制，原dedupe忽略Marshal错误；固定Go继续直接调用原算法。原 ingestOnce 在 observation INSERT 前将所有非零 SuccessTime 转 UTC 并绑定 success_time，PROCESSING/CLOSED 等非 SUCCESS 状态也消费非零时间；只有 IsZero 时绑定 SQL NULL。固定 SQL 必须在原 observation INSERT 消费位置复现 pgx 有符号 int64 整数微秒编码（含溢出 wrap/特殊值）、PG 接受范围及失败后的原事务回滚；不能提前到定位、锁、claim/验证之前转换，也不能因非 SUCCESS 跳过转换。原 ValidateProviderRefund/validSuccessTime、dedupe 和 mismatch 判断仍使用原 Go 时间事实，不能以持久化 wrap 后时间替代。原fresh Refund receipt Marshal若失败，保留其原事务错误/rollback；原hostLocal扫描与fresh UTC/nanosecond回包差异分别验证，不将所有时间强改同一字符串。

### 闭合材料结构

`p_locator`在request_order固定为order_id/prepayment_id/user_id的规范decimal text与pickup_at Time；observe固定为refund_id/prepayment_id/order_id（SQL NULL表示原无订单）。`p_claim`只有SQL NULL或refund_id/owner_hex/version，owner固定16bytes。`p_request`只包含原ProviderCreateRequest八字段（out_trade_no、transaction_id、out_refund_no、reason、notify_url、amount_cents、total_cents、currency），原bytes使用对应hex字段、整数text；不能将Go JSON替换无效UTF8的行为提前成新拒绝。`p_observed`固定MerchantID/OutTradeNo/TransactionID/OutRefundNo/RefundID/State/AmountCents/TotalCents/Currency/SuccessTime，前五原bytes用hex，State/Currency原枚举text，金额decimal text，Time含原offset canonical与UTC同源事实。材料shape固定不是对原尚未消费值新增业务校验，先原lookup/lease/状态优先级，再在原消费位置验证。

返回使用固定业务outcome加该入口原结果事实，未知outcome failclosed；不回显provider输入/SQL/detail。退款 claimCreate 锁内 SELECT 无行返回 ErrNotFound；claimOneQuery 同位置任何 Scan 错误（含无行）返回 ErrUnavailable。两者返回 bool 都为 false，但错误不是 nil：只有 claimCreate 非 READY、claimOneQuery 不 eligible 或各自 UPDATE RowsAffected!=1 才是 false,nil；不能将缺失统一成无错误的 claimed=false，也不能将 query 缺失改成 NotFound。底层decoder保留原nullable、Go原json.Unmarshal容忍行为；转换错误在原发生位置映原Unavailable。订单BYTEA order_no在通知真正validIntent前保留原bytes，不因无通知分支提前UTF8拒绝。请求摘要、观测dedupe、receipt JSON沿原Go权威编码，不以PG jsonb::text替代规范字节。

## 最小ACL与sequence

所有20函数固定普通服务UID、INVOKER、UTC/search_path及既有角色防护；anon/其他UID不可执行/读写业务。目标registry为父70+20=90。新增关系仅refunds/refund_observations：SELECT及原INSERT明确列，UPDATE仅原state/provider/lease/version/timestamps/materialization/observation apply列；无DELETE。已有orders追加refund状态/refunding_at/refunded_at/record_version/updated_at与原ciphertext/keyversion，保留production preparing_at；prepayments仅原退款materialization列；notification_consents追加原INSERT列并保consumed_at；notification_outbox追加claim/mark涉及state/attempt/next/lease/version/provider_message/error/sent列。action_audits追加退款refresh所需response_json UPDATE，原INSERT不扩泛UPDATE。当前OWNER锁复用既有必要UPDATE，不额外授id。

新USAGE只有refunds_id_seq、refund_observations_id_seq、notification_consents_id_seq；outbox/action_audits序列复用既有，任何sequence不授SELECT/UPDATE。父24RLS/13sequence扩为26RLS/16USAGE，已由最终registry、普通UAT Check和负权限矩阵核验。每个最终INSERT/UPDATE列由对应native固定语句派生并纳入权威readiness；未知extra relation/column/sequence、可执行SECDEF、继承/SET-role、CREATE权限继续failclosed。原44/ledger不改，受控版本化管理安装；不自动撤未知对象权限。

## Owned paths（人员和完整路径以proposal为准）

- root：refund/sharedpg.go与_test.go负责五公共方法/provider/部分计数、subscription/sharedpg.go与_test.go负责RecordConsent/RunDue新store接入；common六文件client.go/test、schema.go/test、access.sql、redeem_order.sql仅readiness；必要当前服务纯构造提取逐路径另审。
- refund leaf writer：internal/refund/sharedpg_store.go及_test.go；sharedpg/refund_commands.sql、refund_observation.sql。包含退款结果固定enqueue片段，单writer避免跨事务代码所有权冲突。
- subscription/production leaf writer：internal/subscription/sharedpg_store.go及_test.go、internal/orderadvance/sharedpg.go及_test.go；sharedpg/subscription.sql、order_production.sql。只对已批准独立公共端口实施，无跨writer共编同一SQL。
- Astra当前四change docs。原main/worker宿主、真实provider配置/Wechat/COS/Billing/adminreport/audit/objectstore不本轮修改；如需worker interface纯接缝，先列精确路径和为何必需。

## 实施与验收硬门

先native逐方法行为Red、现有PG基线；以以上冻结语义验证notifier独立clock/原错误传播、Go Marshal时间offset/不可编码、原scan NULL/fulluint64、原receipt宽decoder和fresh/replay边界。每个完整原子纵切RGR，不用registry缺函数当业务Red。公共handler错误与原服务计数逐项差分，失联调用保留原unknown-commit错误而非臆测成功。

实际16：同/异key同退款资源、USER/MERCHANT跨actor、claim多worker、callback/query/create交错、refund与payment materialize/fulfillment redeem/orderadvance同订单竞争；原成功者唯一、原错误优先、token即时失效。未知Create/Query/Send响应及DB ack丢失先独立读回再原入口恢复，禁止重复不可逆调用掩盖。末尾audit/receipt/observe/order/outbox/consume故障整组原范围rollback；无consent成功、consumed最早grant不跳过、已有冲突outbox不能半退款。

普通UAT合成provider全链路/关键并发/最小权限/清理与source绑定，真实Refund微信验签/真实模板送达另列；前70/已迁移支付履约影响面回归，exact candidate detached两轴与runtime独验。本切片已有冻结源码、本地完整影响面与普通UAT合成provider验证证据，详见tasks；用户UI/真实微信仍未验收。第七完成仍不等于billing、全部后台查询/存储/CloudRun五worker/95cases25pages/UI3闭合。

合同与实际结果分开记录：退款、订阅、订单推进源码已冻结；完整20包ordinary/race各2437 PASS，普通UAT20新增入口与权限/清理矩阵通过。最终候选和exact SHA独立验证尚未完成，见tasks。

## 最终构造接缝批准：不暴露不可执行的 Tx 回调

`subscription.NewSharedPG(client, provider) *subscription.SharedService`：新类型直接持有固定 sharedStore/provider/now/owner，显式实现原签名 RecordConsent、RunDue 两方法。保留原两算法、校验/熵/时钟/发送次序；不嵌入原 Service，不实现或暴露 EnqueueInTx。原私有 store 包含 enqueueInTx(*sql.Tx)，因此不得把不支持该方法的 SharedStore 塞入原 newService，也不得填充假实现。此条取代先前“包装原 subscription.Service”的构造提议。

`refund.NewSharedPG(client, provider, notifyURL, notifications *subscription.SharedService) *refund.SharedService`：新包装持有原私有 *Service，显式转发原 RequestOrder、RequestPaidPrepayment、IngestRefund、RunDue、ListPending，以及返回包装自身的 WithClock。不得匿名嵌入原 Service 或暴露 WithNotificationEnqueuer。内部原 Service.notifier 固定 nil；SharedStore 满足原 refundStore 时保留原私有方法签名，NotificationEnqueuer 参数仅接收内部 nil，不执行远程 Tx 回调。构造时由具体 notifications != nil 得到 SharedStore 私有 enqueueEnabled；固定 observe RPC 在本事务内执行订阅规则。nil 保留原可选无通知语义，非 nil 不是发送调用，也不能把实际不可用依赖降为 nil。

正式组装必须使用同一 ordinary client/环境构造二者；这是第九批 main 接线断言，不新增公开 bool、环境变量、HTTP 参数、通用 notifier 或跨库兜底。仅支持已核实 main.go:226 的固定退款→subscription 依赖；native New/WithNotificationEnqueuer/EnqueueInTx API 保持原样。构造接缝现已实现并由本切片合成provider验证；第九批main宿主接线仍待执行。不改变原20 RPC ABI或通知时钟裁定。

## 原生消费点纠正（2026-10-09，只读源码核实）

已逐行核对 refund/postgresql_store.go claimCreate、ingestOnce、claimOneQuery。此次仅纠正以上两处错误需求描述；20 个 named ABI、参数数量、所有权、构造器、通知依赖不变。原文完整快照保存在 seventh-before-refund-native-consumption-correction/；本次没有运行 PG、业务测试或云测试，不把源码核实算业务 Red/Green。

实现差分必须加入：非 SUCCESS 且非零 SuccessTime 的原 INSERT 消费（含正常、微秒精度、超范围/整数 wrap 与错误回滚）；零时间为 NULL；缺失 create 与 query 的不同错误、非 READY/不 eligible、CAS affected!=1；定位/lease/验证优先级与未写断言。上述消费点差分已在冻结leaf及完整影响面验证闭合，具体公共方法仍按原service调用链吸收错误，不把私有返回错误直接猜作HTTP状态。

## finishCreate 历史时刻宿主 Local 材料（批准）

亲核原 refreshRequestReceipt 在 finishCreate 更新之后读取 refunds.created_at，pgx v5.9.2 binary Scan 用 time.Unix，默认 Go time.Local；随后 json.Marshal(view) 把该历史时刻的实际 offset 写入 response_json。不能硬编码 UTC/Asia/Shanghai，也不能使用 p_now 的 offset 推断历史DST；fresh request仍用原now.UTC纳秒，replay直接复用原stored JSON decoder，不统一改写。

唯一方案：复用 order_refund_finish_create_v1 的两阶段固定材料协议，外层7参数不变。首次 p_now 保留原四字段并增加 receipt_time:null；RPC只锁本refund，返回 NEED_RECEIPT_TIME 与该created_at精确事实，零业务写。时间事实为 tagged finite PG-epoch signed64微秒十进制text，或正/负infinity/null标记；不得借JSON RFC3339限制可读年份。Go按原pgx整数decode取得instant，再用此instant.In(time.Local).MarshalJSON生成JSON scalar原字节，不依赖当前年份offset或PG时区表。没有额外读取RPC、缓存、provider request假字段或native类型修改。

第二次同RPC沿用原claim/provider结果及完全相同now四字段，p_now.receipt_time为固定 {stored_kind,stored_microseconds,json_hex,codec_status}；stored_microseconds仅FINITE非空，codec_status为VALID/SCAN_UNAVAILABLE/MARSHAL_UNAVAILABLE，非VALID的json_hex为SQL NULL。Go仅从首次固定事实生成；SQL严格shape且在锁内核对当前created_at事实，变化则无写Unavailable、不重定位循环。VALID json_hex仅合法JSON字符串scalar，作为受信Go宿主Local事实使用；不声称SQL独立验证宿主tzdata。SQL不能拿此材料覆盖业务now或持久created_at。

最终调用重新执行原state/kind/owner/version/expiry判断、原provider更新、读取当前全部Refund字段，再到原refreshRequestReceipt的Scan/Marshal消费点使用材料；codec错误只在该位置产生Unavailable并回滚全部当前写。callback已赢仍刷新其最新state/receipt，不能用首次快照回包。原created_at以外的Scan/Marshal错误同样保留。首次材料阶段不提前对provider结果或receiptJSON做业务拒绝，不追加新熵或provider调用。第二次未知提交遵循原错误/恢复，不自动补写或重发Create。

两阶段新增网络失败仍按原Unavailable且无第一阶段副作用；不宣称网络故障与native单调用完全相同。原固定now在等待材料期间不重采，lease与slow-provider边界保持。此内层wire只用于finish_create；其他Time参数仍原四字段。

必须真实native/shared Red→Green：UTC与time.Local动态DST地点，stored冬/夏与now相反季节、不同年份、负epoch/微秒、不可Marshal年份及±infinity Scan错误；fresh与replay完整receipt字节；callback在两阶段间获胜、created_at变化拒绝、late更新错误优先、最终fault整事务rollback、第二阶段提交后丢ack恢复且provider次数不增。这些退款差分已由leaf完成，最终V3回执与545源码冻结见tasks；其结果只证明本切片合同，不能替代真实微信和完整runtime验收。20 ABI/24owned/构造器不变。

phase1与phase2行缺失/锁读取失败均保留finishCreate原ErrUnavailable。expired/stale/state非CREATE_CLAIMED不制造新错误：phase1只取材料，phase2按原条件跳过provider更新，仍refresh当前行。phase1提交只读事务释放锁；phase2重新加原FOR UPDATE锁；created_at事实比较必须在任何写之前，变化返回Unavailable且不循环。

## Production 原始时间与持久时间分离（真实反例后的 ABI 修订）

两个固定入口改为 order_production_due_v1(p_now jsonb,p_limit integer)、order_production_advance_v1(p_order numeric,p_now jsonb)。仅这两处参数类型改变；入口数量20、外层参数数量、24owned及权限不变，common必须同步signature/OID/GRANT/readiness，不能同时保留旧timestamptz overload。

原RunProductionDue先校验原now非零，再UTC截µs。due p_now为原四Time字段加 cutoff（同源now.Add(30min)的四Time字段）；Go计算Add之后才按原pgx整数编码供SQL查询，不用先转换/取模now后在PG加interval。advance p_now为原四Time字段加 after_json_hex（Go以原State/PreparingAt结构json.Marshal所得bytes的小写hex；原错误被忽略产生nil时为JSON null）。这些是固定内部材料，不增加RPC/公开方法/年份或总bytes业务限制。

锁内先按原Scan处理pickupAt/materializedAt的NULL/Infinity/类型错误，然后原state优先；仅RESERVED执行原productionDecision。比较使用原Go now秒/纳秒事实与真实扫描的pickupAt.Add(-30min)、materializedAt，不使用已wrap的持久now。原zero、version0、missing/非RESERVED/noDue行为不变。只有实际UPDATE preparing_at/updated_at、审计occurred_at消费时才执行原pgx int64微秒编码/模2^64有符号wrap/±sentinel及PG范围检查，按原SQL位置失败整体rollback，不用float epoch。before JSON恒为原字节，after_json_hex在原audit阶段消费；合法year10000原Marshal失败被忽略，必须写SQL NULL而不是SQL重造非NULL JSON。

Go材料由同一截µs时间对象生成，SQL严格closed shape；原canonical_utc是事实文本不是必须能被PG/RFC3339解析的日期输入，year-1/10000等不得因transport新增拒绝。Go错误不提前变成Unavailable；未知时刻/损坏材料按原消费阶段failclosed，不能改变原有效请求。保持原两次指定SQLSTATE重试、逐订单事务和部分Scanned/Advanced计数。

已核真实Red回执：shared-pg-remaining-command-subscription-production/red-production-time-domain-receipt.json，exit1，1PASS/3FAIL/0SKIP（含父）；日志SHA e9e7ed459261e6226ea165647d3d9acc4b26478e6b08330fbfa57f5a10c79269。year10000 native推进1/1且after_state_json=NULL，共享旧实现非NULL；year-1 native空扫描成功，共享旧RFC3339绑定Unavailable。此为真实业务Red，不是Green。必须补同场景Green及wrap可接受时刻、负epoch/±sentinel、锁内raw比较与persist区别、晚fault回滚；没有实测前不得标PASS。

追加真实wrap Red（root/leaf回执）：year600000/586600 native Scanned1/Advanced1且after_state_json为SQL NULL，共享旧文本时间拒绝；red-production-wrapped-domain.jsonl SHA1f1356864d837fd1f836fed194131a675e4fe174b27d153bbc970ec51d9a0364。after_json_hex明确是原Go结构Marshal字节或NULL，不是SQL to_char模拟；canonical_utc仅同源时间事实，不能替代原Marshal结果。year0000/BCE不新增限制。Go内部极端秒边界若出现合法公共反例，先实证再精确移植，不凭推测添加上限。

## 当前私有时间材料澄清

finishCreate 的 VALID JSON scalar 是 Go 对历史instant使用host Local执行原MarshalJSON的结果。历史时区可能含秒级offset，而原MarshalJSON仅输出分钟offset；不得附加 `json scalar::timestamptz = created_at` 条件。仍严格绑定原created_at tagged kind/signed微秒、闭合字段/status及单合法JSON字符串；原late错误与整事务回滚不变。

order locator 可返回 `pickup_kind/pickup_microseconds` tagged PG事实，由Go原pgx扫描语义生成既有p_locator.pickup_at同源材料；不新增RPC/参数，不用SQL重造BC/Local字符串。原locator Scan的NULL/infinity错误仍保留，即便后续业务不读取该缓存时间；最终业务比较使用锁内订单事实。

production after_json_hex传原Go Marshal字节或NULL；v42的after_state_json列实际为JSONB，成功材料在原INSERT消费为JSONB，Marshal失败沿原SQL NULL，不把列误称BYTEA。当前推进与订阅已完成本地子包验收，整切片状态见tasks。

## 已实现范围与后续环境隔离

本切片证明三模块在普通CloudBase服务身份下保持既有业务、事务和恢复规则；ledger SELECT是既有readiness必需权限，INSERT/UPDATE/DELETE及DDL拒绝。90 RPC、26RLS、16USAGE、25业务表清理及44迁移账本均有实际回执。100条合法退款due/pending由原业务入口生成并经真实Go Client读取；最大合法字段保留本地native/shared差分证据，不冒充云端最大字段实验。

新确认的共用商户环境隔离属于后续独立change，不修改本设计的迁移等价基线；在该change部署前，不能声称UAT与Prod支付/退款号及整商户对账已安全隔离。
