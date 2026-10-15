# 支付共享PG原子实现设计

Status: APPROVED_ATOMIC / WRITER_VERIFIED / CANDIDATE_NOT_CREATED。六Application与15固定RPC已落地并有真实本地纵切Green；V3完整本地ordinary/race及普通UAT支付矩阵作为历史通过；V4纯尾空白修正的三个受影响包ordinary/race、70源码/readiness重新安装核验及静态检查已通过；候选、独验与main尚未完成。

## 最小完整范围

paymentorder.Application全六方法：Prepare、Confirm、IngestPayment、RunDue、ListPending、MaterializePending；包括Quote FinalizeForPrepayInTx/LoadSnapshotInTx在付款事务内的原子工作、payment_observations归并、pickup_sequences与orders/order_items materialize、独立用户receipt。Provider CreateJSAPI/QueryTransaction/CloseTransaction及NotificationParser继续现有Go实现/端口，调用在持久事务外，不能拆成只prepare可用的半套应用。

本轮不迁移refund、订阅发送、orderproduction持续推进、billing或main宿主。Materialize必须复用orderproduction纯InitialState/Advance算法，但不因此实现其worker。Quote公开Create/Read已实现不等于付款Finalize：Finalize须锁当前源/原quote并重新验证可付，不重新定价；LoadSnapshot只验证冻结聚合。

## 已读原生关键事实

- Prepare先用户receipt、已有prepay恢复，再读openid→lease entropy→冻结now；prepare事务Finalize+窗口检查之后才mint outTradeNo，insert与CREATE_CLAIMED lease同提交；完成commit才调provider。未知create结果落CREATE_UNKNOWN，不直接重复调用CreateJSAPI。
- Prepare keyHash是SHA256原key，无operation:前缀；payment用户scope、action+NUL+decimal target摘要、requestID原hash与商户audit前缀实现不同，不能复用错误receipt codec。
- Confirm允许原用户receipt固定PENDING结果；原流程先existing order/applyReady，再force query，再applyReady，再独立receipt。用户receipt并非与订单materialize同一事务，不擅自合并改变崩溃恢复。
- Ingest先锁prepay，active query核lease owner/version/expiry，然后Normalize真实expectation，观测与prepay状态同提交。付费回调与close竞态不能把PAID覆成CLOSED。
- Query owner entropy在claim前；due cutoff来自批次now，leaseNow却在拿prepay行锁后重新采样。provider Query/Close结束分别采finishedAt并核lease，不以最初now扩大租约。
- Materialize先定位最早符合条件observation，manual按原分支锁当前OWNER，再prepay→observation→冻结Quote；坏付款/报价事实可能持久DEFERRED/PENDING_MANUAL，而非一概rollback。有效路径pickup序列<=9999、order/items、prepay APPLIED和observation APPLIED同Tx。materializedAt在锁/校验后采样并max(successTime)。
- ListPending仅当前enabledOWNER；manual入口当前Owner前于旧receipt，内部apply选不到观测时的existing-order返回保持原顺序，不自行强化成不同业务结果。

## 最终固定 named ABI：15 个入口

```sql
order_payment_receipt_read_v1(p_user numeric,p_action text,p_key_hash text)
order_payment_prepare_replay_v1(p_user numeric,p_quote numeric,p_key_hash text)
order_payment_openid_v1(p_user numeric)
order_payment_prepare_v1(p_user numeric,p_quote numeric,p_key_hash text,p_openid_hex text,p_owner_hex text,p_now timestamptz,p_lease_ns numeric,p_config jsonb)
order_payment_finish_create_v1(p_prepayment numeric,p_owner_hex text,p_version numeric,p_now timestamptz,p_next timestamptz,p_result jsonb)
order_payment_read_v1(p_prepayment numeric)
order_payment_existing_order_v1(p_prepayment numeric)
order_payment_due_v1(p_now timestamptz,p_limit integer)
order_payment_claim_query_v1(p_prepayment numeric,p_cutoff timestamptz,p_force boolean,p_owner_hex text,p_lease_ns numeric)
order_payment_finish_query_v1(p_prepayment numeric,p_owner_hex text,p_version numeric,p_now timestamptz,p_next timestamptz,p_close boolean)
order_payment_observe_v1(p_expected_prepayment numeric,p_source text,p_event_hex text,p_transaction jsonb,p_claim jsonb,p_received_at timestamptz,p_reconcile_ns numeric)
order_payment_materialize_v1(p_prepayment numeric,p_manual boolean,p_owner numeric)
order_payment_pending_v1(p_reason_hex text,p_after numeric,p_limit integer)
order_payment_receipt_append_v1(p_user numeric,p_action text,p_target numeric,p_key_hash text,p_request_hash text,p_response_json text,p_at timestamptz)
order_payment_owner_v1(p_user numeric)
```

原 55 + 新 15 = 70 个受审固定入口；表、列与序列权限必须按实际 SQL 清点，不在需求稿猜数量。函数均 INVOKER、固定服务 UID/role guard、UTC/search_path 固定，普通其他 UID/anon 不可执行，无 generic SQL、SECDEF 或跨请求 session lock。业务 actor 是既有 Order 服务端已认证 user，不把 CloudBase 用户当 Order 身份。

- 所有 ID/version 为规范十进制字符串绑定 numeric，完整 uint64；允许 0 的仅 after、auto materialize 的 p_owner、observe 的按 outTradeNo 定位 sentinel。其他 ID >0。禁止 bigint 截断/JSON float。hash 为小写 hex64，owner 是非零 hex32；raw byte 字符串用 hex，不用 JSON 的 UTF8 替换掩盖原行为。
- `p_lease_ns`/`p_reconcile_ns` 是原 Go time.Duration 正纳秒整数，<=MaxInt64，不能擅降到微秒配置精度；时间加法先遵循原 Go 计算再按实际 pgx 参数微秒截断。Go 原来已采样的 now/next/received_at 仍按原 `.UTC().Truncate(microsecond)` 传入。SQL 锁后时刻取 `clock_timestamp()`，不提前在 Go 采样，也不引入时钟 offset/config。
- 统一 wire 用闭合 `outcome` + 原始事实，成功 numeric 全 text，bytea 全 hex；receipt JSON 为原始 text，不经 jsonb 重新编码再交 decoder。nullable 字段保留 null。NOT_FOUND、IDEMPOTENCY_CONFLICT、INVALID_INPUT、FORBIDDEN、QUOTE_UNAVAILABLE、UNAVAILABLE 与原方法错误映射对应；Normalize 原拒绝/持久 defer 不是同一种错误。SQLSTATE/constraint metadata 须保留既有可重试与命名唯一键判定，未知错误 fail closed，权限失败不伪装业务 Forbidden。

### 只读入口与独立 receipt

receipt_read 返回 found + 原 evidence/response，Go 原 strict decoder/digest/target 校验；不提前读取 openid 或调用熵源。只允许三个原 action。prepare_replay 严格按原顺序先 user+key，后 quote；两次查询使用原 READ COMMITTED statement 顺序，不合成新快照/锁：先命中另一 quote 为 Conflict，quote 命中另一 user 为 NotFound。每次命中仍需 validStored。

openid 返回原 bytea hex；missing 为 NotFound，空或非法 bytes 不能在这层擅自提前拒绝（后续原 provider request 校验才拒绝）。read 返回完整 prepayment raw 事实并由原 codec 校验。existing_order 保留独立查找。due 原 OR 条件/id 排序/1..100；pending 原 reason 等值/id after/1..100，当前 OWNER 查询由 Go 按原先后单独 owner RPC 完成，不把 SQL 再授权引入不同竞态优先级；manual 有选中观测的写事务内仍重新锁 OWNER。

receipt_append 独立提交，action 固定决定 target_type/reason_code；scope = SHA256(`USER\0` + BE64 user)，key = SHA256 原 key，digest = SHA256(action + NUL + decimal target)，request_hash = SHA256 原 RequestID。Go 原 JSON Marshal response；原 1024/256 字节限制与 strict decode 保持。命名 duplicate 后走 receipt_read 比较原 response；不将不同结果吞成成功，不合并进支付/订单事务。BE64 用 numeric 的精确 div，不用会舍入的 `/`。

### Prepare：在原锁内生成支付号，不再 NULL 材料往返

Go 原顺序：ready→meta→用户 receipt→prepare_replay→openid→lease owner entropy→now→最多两次原类别 DB 尝试。即使 config/输入材料稍后无效，也不提前遮蔽上述结果。SQL 在同一事务完成完整 Quote Finalize 原 source 锁、quote/item 校验、deadline/window，再生成 outTradeNo，再构造并校验 request、insert+CREATE_CLAIMED。不先释放源锁去 Go mint，不使用 SOURCE_CHANGED/refetch 循环。

内部随机原语采用 PostgreSQL builtin `gen_random_uuid()` 的 96 个随机位：`uuid_send` bytes 0..5 与 10..15 拼接（SQL substring 起点 1/11、各 6）；避开 version/variant 位，输出仍 `O<quoteID>-<24 lowercase hex>`。这是内部 CSPRNG 来源替换，不能换公开格式/随机位数、加扩展或用时间/序列伪随机。官方 PostgreSQL 17 UUID 文档确认 v4 random；目标 UAT PG17.11 authenticated 的 builtin_execute/10 次 16-byte UUID 与 12-byte 提取形状已由 `shared-pg-staff-import-root/sixth-uuid-capability-readonly.json` 实测通过；10 个 distinct 样本不证明密码学质量，质量依据下列官方实现。隔离失败/调用阶段已由本地差分及平台claim/provider次数验证。源/窗口拒绝不得先消耗该原语；失败原位置 Unavailable，全事务回滚。

p_config 固定四键 `appid_hex,mchid_hex,description_hex,notify_url_hex`，p_openid_hex 保留原 bytes。request 必须按原 ProviderCreateRequest 的 11 字段顺序生成 Go json.Marshal 等价紧凑 UTF8/HTML escaping/Unicode U+2028/2029 字节及 SHA256；原 byte 长度、NUL、UTF8 校验在原阶段，不能 JSON 替换后放行。返回 claim(id/owner/version) 与实际 canonical request，Go 严格 codec 后才能外调 CreateJSAPI。不增加任意 JSON/姓名/整体响应业务上限。

finish_create：p_result=NULL 表示原 finishCreateUnknown；非 NULL 是已由 Go 原 validateProviderCreateResult 通过的固定 `{prepay_id_hex,request_payment_json}`（后者原 Marshal text）。success 的 Rows!=1 允许 read 当前 PAID/CLOSED；unknown 不允许。同一条件包含 lease kind/owner/version/expiry，provider invalid 仍先 best-effort unknown 后返回 Unavailable。已提交 claim 的响应丢失返回 Unavailable，不因未知结果再次调用 provider；下一 Prepare 按原持久 replay，后续 RunDue reconcile。

### Query、Observe：完整原子归并

query owner Go entropy 保持 claim 前；claim SQL 锁 prepay 后采 leaseNow，再按原 terminal/material/activelease/due 顺序。p_cutoff 是 RunDue 批次时刻，不替代 leaseNow。provider Query/Close 在 claim 提交后、事务外；每次返回后 Go 重新采 finishedAt 并检查 lease。finish_query 的 p_close=true 只对应原 closeExpiredClaim，false 对应 release；保持两者原 ignore RowsAffected，不能统一成 CAS 失败。Create/Query/Close 均不由底层 client 自动重试。

observe 去掉 p_material：同调用先按 expected ID（0 则原 outTradeNo）锁 prepayment→validStored→claim owner/version/expiry→原 Normalize→dedupe/状态归并/commit。Normalize 固定算法直接移植到此 SQL，不新增 RPC/helper 或通用规范化库。p_transaction 为闭合 raw 事实：`appid_hex,mchid_hex,out_trade_no_hex,transaction_id_hex,trade_type_hex,trade_state_hex,trade_state_desc_hex,bank_type_hex,attach_hex,payer_openid_hex,currency_hex,payer_currency_hex,total,payer_total,success_zero,success_epoch_seconds,success_nanosecond,success_rfc3339nano`。金额为 int64 十进制 string；success 是 Go 原 time 的无损纯 wire，秒为整数 text、纳秒 0..999999999，zero 单独保留。固定 codec 校验成员/类型及这些时间表示一致性，不在 Go 提前返回原应锁后发生的 Normalize 业务拒绝。

不能只把 success time 绑定 timestamptz 后再计算：原去重 digest 与 AUTO deadline 判断使用全纳秒，持久 success_time 才按 Go 截微秒；负 epoch、1ns deadline 两侧、非零 offset 必测。p_claim 为 SQL NULL 或 `{owner_hex,version}`，期满判断基于本次锁内 prepayment，不能信任请求自带 expiry。event 为原任意 bytes hex，无新增 trace 长度/NUL 限制；12 个 transaction string 的 NUL 检查包括业务暂未消费的 trace 字段，但不增加原 Normalize 没有的 UTF8 检查。Dedupe canonical bytes 用 bytea 拼 NUL，保持 mismatch 优先序与原 RFC3339Nano 字节。重复观测仍可能推进原 version，不能误定为所有持久字段完全不变。

### Materialize：完整锁内算法，没有 NULL material

先原最早 eligible observation 定位（不改变成 SKIP LOCKED），没有则 existing order/PENDING；manual 仅该分支选到观测后锁当前 OWNER，然后 prepay→observation→冻结 Quote。保持原 snapshot/digest/金额/时间校验区别；原 `deferUnsafeObservation` 可以提交 DEFERRED/PENDING_MANUAL 后返回零 result/false/nil，不能改成异常而回滚或臆造 PENDING DTO。

Quote LoadSnapshot 不重新定价、不校验额外 receipt；完整原 validStoredQuote/shape/digest 移入此原子函数。materializedAt 在校验后 SQL clock 截微秒，并 max(successTime)；原 InitialState `<30min` 与 Advance `>=pickup-30min` 两步、零时刻/有效时间错误保持。pickup sequence 上限9999、order、全部 items、prepay APPLIED、observation APPLIED 一次 commit，任何中途故障整批回滚；无人工 material payload、共享 Tx shim、单独 quote finalize commit。SQL 实现的纯算法必须与原 Go 精确差分，不用已有 Quote Create PASS 代替 Finalize/LoadSnapshot 证明。

## 单一路径与 owned paths

所有相对路径以本change WT的 services/api/ 为根；仅明确owned paths可实现。

| Owner | 唯一 owned paths | 职责 |
|---|---|---|
| root | internal/paymentorder/sharedpg.go、sharedpg_test.go | Application 六方法编排/constructor/typed结果、provider 时序与真实 handler 差分 |
| payment worker | internal/paymentorder/sharedpg_prepare.go、sharedpg_prepare_test.go、sharedpg_observation.go、sharedpg_observation_test.go、sharedpg_materialize.go、sharedpg_materialize_test.go、sharedpg_receipts.go、sharedpg_receipts_test.go；sharedpg/payment_prepare.sql、payment_observation.sql、payment_materialize.sql、payment_receipts.sql | 15 RPC 的固定 leaf codec、四 SQL 的完整事务及原生差分 |
| root/common | internal/cloudbasepg/client.go、client_test.go；sharedpg/schema.go、schema_test.go、access.sql、redeem_order.sql | 70 registry、实际最小 ACL/RLS/序列与 readiness 完整指纹；最后文件仅 readiness 投影，原 Redeem 业务 body 不变 |
| Astra 需求 owner | 本change四 OpenSpec 文档 | 需求/验收/任务证据，Astra唯一writer |

root+worker 14 新文件、common 6、docs 4，共24预分配路径。四 SQL 中 prepare 放 prepare/replay/openid/read/finish_create；observation 放 due/claim/finish_query/observe；materialize 放 existing_order/materialize/pending/owner；receipts 放 read/append。跨文件只共享同包私有确定结构，开始实现前可协同声明字段，但不互写他人路径。既有 native 与 quote.sql 均只读，不先授权宽泛 helper 提取。

最短顺序：(1) 先 receipt/raw codecs 与只读入口真实 Red→Green；(2) Prepare 锁内完整 Finalize/request/claim 与 finishCreate+provider spy；(3) claim/query/observe 单事务与失 ack；(4) materialize/manual/due/六方法 handler；(5) 统一 registry+普通 UID 平台矩阵+清理，再 Writer Gate/candidate/独验。所有阶段仍属同一完整支付 change，不能先启只 Prepare 的 runtime。

## RGR硬门

1. 六Application方法与原handler/native实际v44成功→共享缺入口Red，闭合完整durablefacts；真实bytes/Unicode/u64/int64/epoch/nullable以及strict receipt损坏差分。
2. provider spy精确次数/顺序，Create提交后断连、provider成功DBack丢失、Query/Close超lease及跨进程kill恢复；不得因为网络重试重复不可逆支付创建。
3. 真16同key/不同key/相同quote/回调重复与跨transaction冲突；callback PAID和close并发、过期lease抢占、old owner/version晚返回不覆盖winner。
4. Finalize与第五名单/折扣/目录/店铺变更锁竞争；原身份语义变化拒付但只折扣版本变化不重定价；Quote无receipt GET规则不被误用成付款source弱校验。
5. 订单序号上限/并发唯一、items中途/末尾observation与prepay故障全部回滚，坏事实持久defer完整、manual当前auth/旧receipt优先。
6. 真锁等待lease时钟/窗口、ctx取消及无deadline活性；原有最多2次DB retry仅对原类别，不增加provider重试或固定业务timeout。
7. 统一leastpriv/installer、普通UAT合成provider全链路与关键并发/unknownack/cleanup、exactSHA独验；真实微信支付商户材料和手机支付另为外部门，不能以合成providerPASS代替。

本版记录第六实现、本地Writer Gate与限定普通平台/cleanup已通过。父第五exact 5f5af0cd05eafbddfb660631adf6e12daf450a87已独验通过；本切片candidate与独验仍由root按生命周期推进，main/CloudRun/95 cases/25 pages/真实微信支付尚未闭合。

## NULL、时钟、输入优先级的决定性反例清单

- nil provider/executor/clock/lease source 的 ready 失败先于 meta；无效 meta 的 key/request UTF8/NUL/128 字节规则原样。有效 receipt/已有 prepay 跳过 openid/lease entropy/outTrade entropy；另一 target 同 key 冲突先于坏 source，receipt 损坏先于 provider 调用。
- openid 缺失先于失败 entropy；存在但空/NUL/invalidUTF8 的 openid 不早拒绝，原 Finalize 失败先于 request 校验。过期报价+不可用随机源不应错误地报告 entropy；已完成 Finalize 后随机源失败不能写 prepay。SQL NULL 只作明确 nullable facts/unknown result/absent claim，不能成为准备材料的新状态机。
- Query 终态依旧先调用 Go lease source；拿锁后 leaseNow 必测持锁跨 expiry。Prepare now 是原前采样值，不能顺手换为锁后 now；Observe receivedAt 也是原前采样值，不能因为锁等待自行改过期判断。Materialize/unsafe defer 是原锁后采样，保留各自阶段。
- callback missing prepay+trace NUL→先 NotFound；已存在但坏 stored request→先 Unavailable；有效 prepay+过期 claim+trace NUL→先 Unavailable；正常 claim+trace NUL→InvalidInput。无 NUL 的 invalidUTF8 交易 bytes 按原 Normalize 接受/不匹配规则，不能 convert_from 提前制造拒绝。
- provider Create 仅已知新 claim commit 后 1 次；错误、invalid result、finish 成功/unknown 的晚响应与终态容忍分别对照。模拟持久提交后丢 ack 必须独立 readback 后再原入口恢复，不能用后台自动重发 Create 掩盖。
- manual OWNER disable/unbind 与旧 receipt/current claim 锁竞争；无 eligible observation 的原 early existing/PENDING 与有 observation 的锁内重新授权分别验证。不得擅加自动路径 OWNER 限制。
- batch 100 上限/计数语义、pickup9999边界/两付款同日、manual与auto16竞争、观测 dedupe/callback与close16竞争；末个 item/prepay/observation/receipt 故障分别验证原事务范围。独立 receipt 失败时业务已提交允许存在，不能虚构全局回滚。

## 随机源依据与证据边界

官方 [PostgreSQL 17 UUID](https://www.postgresql.org/docs/17/functions-uuid.html) 描述 gen_random_uuid 的 v4 random。官方 [REL_17_STABLE uuid.c](https://raw.githubusercontent.com/postgres/postgres/REL_17_STABLE/src/backend/utils/adt/uuid.c) 的 gen_random_uuid 先用 pg_strong_random 填满16字节，只改0-based bytes6/8的版本/variant位；uuid_send 原样输出16字节。官方 [pg_strong_random.c](https://raw.githubusercontent.com/postgres/postgres/REL_17_STABLE/src/port/pg_strong_random.c) 使用系统密码学随机设施并在不可用时失败。因此抽取未改动的12字节保留96位 CSPRNG 输出，此为源码支持的推论；不是由10次样本作统计证明，也不宣称审计了云厂商私有编译产物。此最小内部实现已批准，不新增 pgcrypto/付费/权限或变更用户支付规则。

ABI首次批准时的历史证据仅包括原生源码顺序与PG17 builtin能力；当时第六SQL/Go实现、完整原生差分及合成provider平台矩阵尚未完成。后续V3实现、本地ordinary/race各1944 PASS及限定普通UAT矩阵/清理已完成；当前V4仅格式修正，分层结果见文末及tasks，依据见tasks最终证据表、final-v3-all17-ordinary-receipt.json、final-v3-all17-race-receipt.json及final-70-source-44-ledger-29-history-readiness-receipt.json。真实微信材料/验签端到端仍未闭合；第五容量回执始终不作为第六通过依据。

## 第六15入口成功响应容量：全部保留8MiB

2026-10-09只读审定。**growing HTTP200白名单新增项为零**；第六15个入口全部保留现Client默认8MiB，不承袭第五所有聚合型接口的例外。分类依据是成功输出，而非请求参数或所读取表是否可能很大；不新造业务限额。目标registry70中第五24增长项保持，第六15均bounded。

| 固定入口组 | 原始输入/持久来源与成功输出界 |
|---|---|
| receipt_read | 原payment evidence严格256B、response严格1024B；Go原decoder拒绝更大值，不会拒绝原可成功receipt。actor/action/ID/hash固定。 |
| receipt_append | Go原response Marshal <=1024B，固定状态回执；冲突后独立receipt_read，不返回任意全audit集合。 |
| prepare_replay、read | 单prepayment事实：原有效ProviderCreateRequest固定11字段，appid/mchid各64B、description128B、outTradeNo/timeExpire各64B、notifyURL2048B、payer128B、quoteID20B、digest64B与整数/CNY。合法创建持久原canonical JSON有界；wx参数由原validateProviderCreateResult限制prepayID128、timestamp32、nonce128、package256、paySign1024B，其他只有有限标量/hash/时间。整体即使JSON/hex双重转义也远小于8MiB，不包含quote/items。 |
| prepare | 同上有界实际canonical request与claim ID/owner/version；完整Quote Finalize/source可能很大但留在SQL内，不作为成功wire返回。 |
| openid | v44 miniprogram_users.openid bytea<=128B；hex最多256字符，加固定found/outcome。 |
| finish_create、finish_query、observe | 原有界状态/outcome/claim结果，不返回完整provider交易trace、source event、quote或支付观测集合；输入trace可能增长不使固定成功输出增长，不能回显输入调试信息。 |
| existing_order、materialize | 固定订单ID/state/applied结果；materialize原完整snapshot/items只供锁内计算/写库，不能为便利将全部聚合加入返回ABI。 |
| due | 原limit1..100，仅ID集合，20位uint64字符串。 |
| claim_query | 单prepayment与固定lease owner/version，界同read；不得额外返回完整关联quote。 |
| pending | 原limit1..100，id、pending reason（原校验/持久有限reason字段）与CreatedAt；无items/name/trace集合。 |
| owner | 当前enabled OWNER count/boolean固定标量，不返回merchant_accounts.Name。 |

权威：paymentorder/types.go、postgresql_prepare.go(validStored/canonicalProviderCreateRequest)、service.go(validateProviderCreateResult/maxWorkerBatch)、receipts.go(maxReceiptJSONBytes/decodeCommandEvidence)、postgresql_materialize.go(listPending)、v44 000008/000031/000032。这是原健康业务可写/可成功读的数据合同判定，不是新增存储检查；损坏JSON/异常字段需继续按native错误，不借容量策略改变语义。若实现增加了原批准wire之外的任意raw聚合，须先回需求审定，不偷偷自动uncap。

最低验证：15精确名HTTP200的8MiB+1拒绝及合法边界通过；代表prepare/read/claim最大合法字段（JSON HTML/Unicode转义与hex）通过，receipt256/1024与超界保持原结果；due/pending100含uint64max。保持第五24名单完整不变，不因70 registry更新漏开/误开。本切片现已完成这些本地容量验证及普通UAT最大字段/100行精确wire验证；第五V4云容量仍不冒作第六证据。


## 固定 SQL 失败传输（Writer Gate已通过，候选独验待完成）

不扩展通用 Client、不解析可能含私密值的 message/details。需要原生数据库分类的固定 leaf 使用唯一闭合失败对象：`{"outcome":"SQL_FAILURE","sqlstate":"23505","constraint":"uq_prepayments_quote"}`。三成员均必需，禁止额外成员及成功材料；`constraint` 仅允许原四项 `uq_prepayments_quote`、`uq_prepayments_user_idempotency`、`uq_prepayments_out_trade_no`、`uq_action_audits_command_receipt`。`40P01`/`40001`/`55P03` 使用同一对象且 `constraint=""`。其他组合全部 fail closed，不根据泛 `23505` 重试。

适用固定函数为 prepare、observe、claim_query、materialize、receipt_append；其中命名 23505 只在 prepare/receipt_append 的原分类路径返回。SQL 用覆盖函数全部业务写入的最外层 EXCEPTION 子块，并以 GET STACKED DIAGNOSTICS 取得 RETURNED_SQLSTATE/CONSTRAINT_NAME。进入 handler 前整个业务子事务已回滚，外层不得留下写入；序列消耗不承诺回滚。未知 SQLSTATE/约束原样 RAISE，不转换为业务 Conflict。数据库正常持久 DEFERRED/PENDING 结果不属于异常，不因此回滚。

Go 对成功 HTTP 中这个闭合对象严格解码，只重建 Code/ConstraintName 两个字段的 pgconn.PgError，复用原 isDuplicate/isRetryablePostgreSQL；其他字段保持空。现有 CloudBase error 若携带精确 40P01/40001/55P03，也可映射原分类，以覆盖函数外/提交阶段的错误；不依赖所有异常都能在函数内捕获。原 max2/命名重复后的重读位置不变，receipt_append 命名重复按原重读，不自行新增整体循环。权限、未知唯一约束、取消与未知提交 transport 不增加自动重试。只有已知提交的有效 claimedCreate 才可调用 provider，SQL_FAILURE 绝不能触发外部付款创建。

必要 RGR：四命名约束各精确识别；无关23505不进入重复分支；三可重试状态保持原次数；伪造/多余/缺少字段失败；晚步故障所有事务内业务事实回滚；失败 envelope 后 provider spy=0；真实普通身份 REST 传输类别与未知 ack 保持原恢复语义。以上为批准并已实现的私有错误协议；局部验证不代替完整Gate，不新增 RPC/ABI 参数/业务规则。

## 唯一协作接口（批准）

采用root writer-coordination-draft原案，不新增配置/接口层。root在sharedpg.go声明私有sharedPaymentExecutor，其唯一方法CallRPC(context.Context,string,map[string]any,any) error；SharedPGApplication仅client、provider、config、now、leaseOwner五字段。NewSharedPGApplication(client,provider,config)保留原零duration默认值，now=time.Now，leaseOwner=randomLeaseOwner；ready的nil与原配置优先级不改变。root实现六Application公共方法、ready、queryOne/provider Close编排、ingestPayment/applyReady原重试wrapper及finishPrepareCommand/finishConfirmCommand。

leaf在*SharedPGApplication上实现原Service签名的replayUserCommand/appendUserCommand、findPrepareReplay/readPrepaymentByID/prepareOnce/finishCreateSuccess/finishCreateUnknown、claimQuery/releaseQuery/listDuePrepaymentIDs/ingestPaymentOnce、applyReadyOnce/listPending/isEnabledOwner/existingOrder。唯一新增leaf适配为readOpenID(ctx,userID)(string,error)、finishClosedQuery(ctx,claim,now) error（仅数据库finish，provider Close仍root）。复用原prepaymentRecord、claimedCreate、queryClaim、commandReceiptResponse，不重新定义或修改native类型。leaf拥有闭合wire/failure decoder；root只消费上述原typed facts，不重复声明codec。

RPC raw对象采用私有闭合结构，nullable按原facts保留；typed codec必须执行原validStored/canonicalProviderCreateRequest及receipt校验。允许leaf选择不影响本稿named ABI的私有Go字段名，但不得添加新公共DTO、任意聚合响应、提前业务校验或静默吞未知字段。SQL_FAILURE精确三成员协议如上；type正确但业务错误的事实仍按原阶段消费。支付provider只能由root编排调用，leaf不持provider外调职责。

当前保持一个12路径leaf writer；无需额外worker或公共路径拆分。并行root common/app与leaf SQL/codec足够，具体新增共享fixture须先明确路径所有权。

## 原quote临时不可用的封闭重试标签

批准仅prepare/materialize叶子返回严格单字段 `{"outcome":"QUOTE_UNAVAILABLE_RETRY"}`，映射原 `quote.ErrUnavailable`，由root原最多两次wrapper消费。不得加message、rawSQL、source或成功材料。仅原quote权威调用阶段中确实映射为该错误的确定性读取/扫描/源验证失败可用；未发生业务写，或函数已将整个业务子事务回滚。不能把所有shape/digest失败、任意SQL错误或HTTP错误都归此类。

原quote/prepay.go与provider.go normalizeSnapshotReadError保留ErrSnapshotInvalid、其余读取错误归ErrUnavailable；Finalize当前identity/settings/resolve/store/date/meal/products的特定读取失败归quote.ErrUnavailable。实现须逐原语句对应，snapshot形状/digest异常属于ErrSnapshotInvalid。Prepare原mapPrepareError将SnapshotInvalid及PrimaryPhoneRequired最终映射payment ErrUnavailable不重试；Expired/QuoteStale/ItemUnavailable/PickupCutoffPassed/PaymentAmountTooSmall及RequireCreateWindow原错误映射ErrQuoteUnavailable，不使用新重试tag。

Materialize不得套用Prepare的snapshot映射：原applyReadyOnce对LoadSnapshotInTx的ErrSnapshotInvalid/ErrNotFound进入deferUnsafeObservation("QUOTE_SNAPSHOT_INVALID")，成功时持久DEFERRED/PENDING_MANUAL并返回zero result/false/nil；不是UNAVAILABLE回滚。只有其原quote.ErrUnavailable来源进入新tag。晚CAS私有P1000必须rollback后UNAVAILABLE；未知HTTP/commit/权限继续ErrUnavailable、不新增重试，既有40P01/40001/55P03精确分类不变。

必要RGR：可恢复quote读取失败第一次tag第二次成功且provider最多一次；两次失败终态Unavailable；Prepare坏snapshot/缺phone不retry；Materialize坏/缺snapshot实际defer持久而不rollback；新tag额外/缺字段拒绝；lateCAS与unknown/permission不retry。此协议批准仅用于恢复原typed重试语义，已有本地差分证据，本地最终Gate与限定普通平台矩阵已通过；15签名/70registry/公共错误不变。

## 成功时间canonical text的受信编码边界

真实native ACTIVE_QUERY/NOTPAY的SuccessTime可为year1000000并只参与digest、不写success_time；共享锁前to_timestamp导致22008已构成行为Red。批准最短私有wire澄清：固定Go leaf从同一UTC time对象生成success_rfc3339nano、epoch_seconds和nanosecond，保证材料同源；SQL严格成员/类型，直接使用该canonical文本进行原digest，不声称独立Gregorian一致性验算。只有原PAID分支真正持久success_time时才转换PG时间，保持lookup→validStored→claim→Normalize→持久原阶段。无新日期算法、范围限制、配置或RPC参数。必须真实year1000000 NOTPAY Red→Green，PAID经原pgx整数编码后实际被PG拒绝时，仍与原Unavailable/全事务零部分写一致；可合法回绕持久化的原输入不得提前拒绝，并保留负epoch/ns/offset差分。

## 持久请求JSON的原Go解码语义

真实stored-json-case-red.log证明：持久JSONB将appid改为AppID、原canonical digest不变时，native Confirm创建订单，共享原strict小写实现Unavailable。固定SQL在原validStored消费位置与原锁内复现原canonicalProviderCreateRequest：通过json_each(req::text::json) WITH ORDINALITY按实际JSONB序列化文本顺序消费；ASCII大小写及原tag相关ſ折叠；未知字段拒绝，合法alias后值覆盖前值，null对string/int64不覆盖；任一错误类型或int64溢出不能被后续alias修复。原对象输入不强制恰11原键，最终归一字段仍须全部原必填/类型/长度/语义与canonical JSON/hash验证。纯读取继续直接复用原Go decoder；不更新持久raw、不新增RPC/参数/helper，也不添加公共alias功能。

Go1.26.5 unicode.SimpleFold穷举相关ASCII的非ASCII映射仅ſ→S，已有外部证据。case Green SHA ee4b1104ac9a4c7a614a6de549e8ac53e861a928a85137da1e16dae3b929e8e4；8alias差分Green SHA d93ce7f86186335ccecd41374a276cdc09697d9b0ba24f528675322a3b8f4f69。以上都是本地PG16，不是普通UAT。

受信Go时间仅免SQL独立Gregorian验算，不免私有wire成员/类型检查。p_config固定四成员必须string且合法hex；不得通过->>将JSONnumber转成合法配置字节。守卫须留在原配置消费位置，不能提前到receipt/replay之前。该补充已取得普通role直接RPC Red→Green（四number字段、uppercase hex及unknown成员），无claim/receipt副作用且原lookup/replay优先不变；该行为Gate绑定V3；当前V4仅尾空白变化，源hash/受影响回归另行绑定，不将旧manifest冒充当前源码。

Refactor保持最小：继续原Application/private record签名与typed错误；没有新通用事务/Unicode/日期库。leaf仅删除无业务作用的空source-item循环并格式化owned tests。必要行为修复均有对应差分，不为重构扩大范围。

## 原pgx整数时间持久化与Infinity消费

PAID持久success_time必须按锁定pgx/v5 v5.9.2原二进制编码精确产生：Unix()*1000000+Nanosecond()/1000-946684800000000，以mod 2^64还原signed64。±int64哨兵保留PG ±infinity；其余从2000-01-01 UTC叠加整数天与整数文本余微秒，不经float秒。year100000三种纳秒原浮点少64µs、year600000原pgx可wrap到15445而旧SQL拒绝，均有真实Red→Green；不新增年份限制。

该编码仅用于持久值。Normalize原full-ns截止判断仍用raw secs*1e9+nanos与deadline exactnumeric比较，摘要仍原canonical text，不能用wrapped success_at改变AUTO/DELAYED_MANUAL。后续materialize在原observation扫描位置拒绝非finite时间；native扫描*time.Time失败必须Unavailable/零写，不允许SQL读取Infinity后新增defer/receipt。RunDue仍按原编排吸收相应内部错误与计数，不把内部Unavailable改成新的公共错误。39项本地时间/消费差分Green回执SHA255bee496a1d75048cb52fa6a1311e711caa7fdbea08613e44d5ca62ebafa061；PG17普通身份定点原生oracle复验已通过，真实微信外部支付另验。

## UAT受控管理安装与最终核对

完整25迁移Push因Blob data size exceeds limit且无TaskId被拒，独立回读仍55body/24history及原业务baseline。随后官方SDK执行版本20261009080500/order_shared_pg_payment的固定BEGIN事务，带前55源码/44ledger与后70源码/44ledger guard；70实际body与ordinary Go Check通过后才Repair applied登记，现25history。旧24逐项原Query回读已按字节/SHA确认不变；加四次临时fault/ACL安装清理，最终29条history。

本次executed704160B SHA5a37d5d26385f901d3589b93c3783fa9384a6fc113bfd403c2e7cfd62a777063；stored704153B SHA2d9608a262f5ec22c8d08a7cb0542e4dbc08b1c75594ab26f095dc157bad7944。需求独立词法核验7LF全在外层分号之后，18个dollar体、112字符串、7行注释逐字不变，状态RECORDED_FORMAT_VARIANT_VERIFIED。仅证明本次登记语义一致，不推断服务端通用正规化算法；两hash分别保留。审查JSON SHA7182edd4236b62637d166b813e6ecf65697b5cfffb8b70b8c7d3caac31d02e1d。实际安装/readiness不等于支付业务、CloudRun或完整UAT通过。

四个probe迁移20261009083500、083600、084000、084100执行/登记格式分别有7/7/4/3段外层分号后空白差异，完整词法核验dollar体、字符串、注释终止及token不变，保留两类SHA，不推导平台formatter。外部审查SHA38f835027cd10c5c173c6cc81f041dc5516b6c944ee80a09c1a3ad99a50ca547。

平台验收以原始wire/Go DTO为数值权威：100 due最后ID是字符串18446744073709551615，ListPending原DTO JSON numeric的同一完整数值保留在capacity-pending-exact-wire.jsonl；JS JSON.parse舍入是控制器诊断，不是业务输出改成string或允许精度丢失。provider为合成，ack恢复中的合成provider恢复基线与application Create/Query/Close调用数分开记录。C3同资源竞争的201/409/503按真实native对照及持久唯一性判断，不承诺每次调度具有固定状态计数。

## V4 最终格式修正与证据继承

真正staged diff检查在候选前发现4个新SQL文件15处尾空白，exit2，候选未创建；此前普通git diff未覆盖未跟踪文件。修复仅删除14空白行各1空格及DECLARE末尾1空格，保留LF。外部sixth-v4-trailing-whitespace-lexical-review逐行证明全部是函数体词法空白，内层字符串/注释/token无变化；但prosrc字节确实变化，因此新freeze和readiness/平台版本不可省略。

V3完整17包×1944及业务平台矩阵仍明确标V3历史。V4受影响sharedpg/migrate/paymentorder ordinary/race各471、3包、零失败/跳过；其余14包按未变输入保留历史证据，未声称V4重跑。临时PG初次socket路径超过macOS103bytes属于fixture启动失败，tests_started=false；改任务临时短路径后完整执行，不计产品Red。V4新exact candidate独验仍必须全部17包ordinary/race。

20261009091500受控SDK guarded Execute→完整70body/44ledger+ordinaryCheck→Repair完成，旧29Query逐字不变，新总30。14个变化定义与当前源精确相同；25业务表count/hash无变。执行450396B与登记450395B只差COMMIT后的末尾LF（zero-based450395），所有函数体/字符串/注释/token字节相同，具体词法回执确认RECORDED_FORMAT_VARIANT_VERIFIED，不推断通用格式化算法。完整30history官方Preview executable；仅新版本Preview报missing29所以未写；已有Blob上限实证后不盲重试更大Push payload。V4只做源码/readiness更新，不冒称重跑V3支付矩阵。
