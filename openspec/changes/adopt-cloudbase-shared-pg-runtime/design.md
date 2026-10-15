# Design

## Selected Architecture

保留Go CloudRun、Order Bearer/角色、PC/Mini与COS；共享PG原子业务命令用HTTP RPC。此change只实现REDEEM_ORDER，不把其余命令切成隐藏原生fallback，也不启用未完整的CloudBase后端。root先证明隔离platform fixture，再在真实v44 schema上验证核销。

## Existing Contract

原生执行顺序为校验输入→当前商户account FOR SHARE→order FOR UPDATE→既有receipt匹配重放→READY结构/时间/cipher检查→状态更新→receipt→commit。OWNER/SUBACCOUNT均能核销。账号禁用后旧key不能绕过授权。核销清ciphertext和key_version，保留token_hash/issued_at，record_version恰加1，completed_at=redeemed_at。成功receipt的scope/action/operation hash与原Go request JSON摘要保持。

新key核销已完成订单仍是TransitionNotAllowed；不要移入REDEEM_TOKEN/DATE_CODE的completed特殊重放。核销不产生新订阅通知，MARK_READY才有该事务义务。

## Proof Read and Atomic RPC

Go从受限服务端接口读取当前核销proof并用既有cipher.Open、合法token和常量时间hash比较验证；该预读不能对数据库产生业务写入。原子RPC锁内比较完整record_version/hash/ciphertext/key_version/issued_at，防止预读后被替换，再验证当前状态/时序、变更订单、追加audit receipt。proof不一致返回内部可区分错误，Go有限重读/重验。禁止仅传validated=true。

proof读取接收actor/order/key，返回当前scope/action/key下receipt_exists及完整proof。RPC重放必须在proof校验之前；receipt_exists=true时Go跳过解密，但必须把validated_proof传NULL，不能把未验证proof当可执行凭证。RPC若锁内无receipt且validated_proof=NULL，只返回内部NEEDS_PROOF并有限重读，不执行更新，防止账户重绑导致scope变化的校验旁路。读取到COMPLETED/cipher为空仍允许RPC检查旧回执；当前商户授权始终在重放前核验。

建议`order_redeem_order_v1`为固定typed参数，action固定，actor来自已验Order bearer。account/role/auth_version由锁内查库得出；业务result由RPC构造。REDEEM_ORDER请求原始字节固定为Go json.Marshal的kind+order_id，优先由这两个固定字段构造并验证摘要，禁止用jsonb::text重新序列化算摘要。真实schema的ID/version使用NUMERIC(20,0)，须支持完整uint64而非限制到bigint；数字用精确十进制，HTTP解析不用float64；字节固定编码；时间UTC微秒。root提交具体签名/DDL后需求owner评审，签名未评审不计接口完成。

## Identity and Function Isolation

2026-10-08真实UAT已证明APIKey+exec-pgsql的运行角色是orders owner，public CREATE=true；受控DO实际CREATE TABLE后抛P1099并整体回滚。故该入口违反运行身份无DDL管理能力硬门，不能候选。先前APIKey通道的业务通过只保留为原子命令原型证据，角色与入口相关验收全部重跑。

唯一运行入口改为：Go内部专用普通CloudBase用户登录所得JWT→PostgREST `/v1/rdb/rest/rpc/<固定函数>`。不保留exec-pgsql fallback或任意SQL接口。Client.CallRPC只接源码固定allowlist函数和named参数；proof、command及readiness均经RPC。SQL参数绑定要求等价为JSON named参数与固定RPC路径，数值仍十进制string、proof原始JSON，响应无float64失真；超时/取消/限额/禁redirect/SQLSTATE脱敏/无transport自动重试保持。REST原始jsonb返回形状须真实测定，不能沿用exec的行数组假设。

专用服务用户与Order业务用户无绑定关系：CloudBase UID只标识Go工作负载，actor仍来自现有已验证Order Bearer，商户account/role仍在事务中读取。登录使用已启用的普通用户名密码方式；只在后端保存该普通用户凭据及内存token，不使用APIKey、JWT签名密钥或管理用户凭据。选定单一续期方式为仅缓存access token及到期时间，在到期前singleflight重新signin；不保存refresh token、不调用refresh endpoint、不设计失败fallback。必须实际验证重复signin的UID/role恒定、到期边界、并发续期/取消、失败共享与重启重登；失败fail closed，不回退管理员入口。官方refresh会旋转并失效原refresh token，本实现不引入该状态；signout/禁用账号后的已有JWT失效时序须另作平台实测，不凭文档猜测即时撤销。

函数保持SECURITY INVOKER、固定search_path与全限定业务对象。守卫同时要求可信current_user=authenticated、auth.role()=authenticated、auth.uid()等于管理安装固定的专用服务UID；UID不由调用参数、可自改user_metadata或请求头自行决定。UID为部署公开标识，不是secret，纳入候选部署manifest/指纹，不使用运行时任意配置改写SQL。

原44迁移文件与表owner不变；在独立管理版本中为本切片三表启用RLS，仅向authenticated授原子命令需要的表/列权限与sequence USAGE。SELECT/UPDATE/INSERT策略按动作限制，UPDATE同时USING/WITH CHECK固定服务UID。专用UID直接REST执行这些必要DML是允许的后端权限，不保证只能经RPC写；代码仍仅调用固定RPC，普通Order客户端绝不取得该JWT。其他authenticated UID、anon/缺JWT须RPC拒绝、表读无业务行、表写无业务变更（SELECT为200空集可作为无泄漏证据）。拒绝额外permissive policy、PUBLIC或继承权限绕过；固定UID替换/撤销后旧UID即失去策略授权。

readiness改为固定无参INVOKER RPC，受同服务UID守卫，只读系统目录并输出actual metadata；Go对照候选预期函数body/signature/STRICT=false/VOLATILE/PARALLEL UNSAFE/UTC/search_path、RLS与policy/ACL/role及v44合同，不采信数据库自报ready=true。候选工件中三个函数和部署UID均需独立指纹；不开放通用SQL或任意catalog查询函数。

真实角色硬门：authenticated非owner、无superuser/BYPASSRLS/createrole/createdb/public CREATE、不能SET ROLE到高权角色；普通JWT无法exec-pgsql管理SQL，也不能利用已暴露函数获得DDL。用隔离INVOKER探针实际尝试CREATE后无论成功失败均回滚并复读残留；意外成功判FAIL，不靠HTTP错误码推断未执行。更改token/UID、伪造actor之外的role字段、普通用户访问三表/三个RPC均独立验证。

## Private Storage Dependency

整体CloudRun runtime不得继续通过CLOUDBASE_APIKEY换存储临时凭据；只改PG入口不能消除进程内管理能力。当前命令切片不接runtime，存储身份及适配为独立后续change的硬依赖。最小权限CAM子身份也不能仅凭COS策略就判安全：还须证明不能经CloudBase client_credentials换超级管理员token，或经/capi/credential获得更宽权限；目前没有这样的实测证明，也没有已核实CloudRun自定义workload role的可执行配置。

按实际代码，前端只识别PUT和opaque URL/headers/object_key，Service另行校验5MiB、PNG/JPEG、实际MIME、图片解码、精确字节数/SHA256及确认回执。因此provider变化本身不是公共合同变化；先前一概禁止pgstore属于过强限制，现撤回。已完成合同复核后，当前仍维持私有COS的版本保留目标；普通服务JWT+私有pgstore只有证实相同TTL/恢复条件才可成为内部等价Adapter，不默认投入实现。Go/PC/Mini公开DTO、intent→PUT→confirm顺序、服务端生成key、确认后才可引用、私有短期GET均保持，现有COS生产决策不改。

必须先实证pgstore的签名PUT有效期不超过15分钟（现实现10分钟）、签名GET到期、禁止同key覆盖及并发首次写、5MiB/MIME限制、确认阶段精确size/digest/真实图片拒绝、普通UID/匿名读写列举隔离、真实浏览器CORS及headers透传。SDK支持signed PUT/GET不等于这些条件通过；上传签名未暴露每对象type/size/expiry参数，需与当前COS签名负例做实际差分，不能把发送header当作已绑定或把未确认坏对象当作已确认成功。

既有production-architecture-baseline spec103/119还明确桶版本控制与非当前版本30日保留。官方pgstore FAQ说明version字段不等于自动版本管理；其底层版本控制/恢复能力仍待实测。不可用内容寻址key或默认不覆盖替代删除后的恢复保证，不新增自制版本系统来绕开该缺口。TTL与恢复任一硬门未满足，pgstore替换不得进入实现验收；当前COS最小身份依赖保持BLOCKED，完整runtime不启用；本核销切片可独立继续。不引入CAM长期密钥作为暗中fallback。新实际配置与业务材料仍须重测5KB EnvParams。

官方依据：普通JWT及auth.uid/GRANT/RLS见https://docs.cloudbase.net/authentication-v2/auth/auth-pg ，RPC见https://docs.cloudbase.net/database/postgresql/rpc ；pgstore访问与版本边界见https://docs.cloudbase.net/storage/pg/serving 与https://docs.cloudbase.net/storage/pg/faq 。以上官方能力均不等于本环境已验收。

## Controlled Installation

管理面versioned manifest，运行面RPC；原44SQL字节/hash/name/version保留，Order schema_migrations仍是项目事实源。平台migration history只记录平台安装。RPC新版本与定义指纹另存，不改原v44业务约束。readiness检查ledger、业务结构和RPC签名/定义。

原runner逐版本SQL+ledger同事务，session advisory锁跨commit；v20/23还执行Go目录名称预检及表EXCLUSIVE锁。不能仅拼44文件称等价迁移。选定首轮安装方案为管理面单事务DO：取得与native相同bigint namespace/key的xact advisory lock，在锁内检查空库，然后顺序执行通过原scanner/hash检查的44权威SQL及其ledger写入。任何失败整批回滚，空库仍空；这不等同旧逐版本commit/prefix恢复。已有精确完整44 ledger时逐项核hash/name/dirty及业务结构，匹配才幂等返回；该路径沿用native readiness允许额外未使用对象，以容纳独立RPC工件，但不放过44合同对象或依赖漂移。初始空库路径未知对象、partial/dirty/history冲突均fail closed。空库检查涵盖冲突业务表/视图等public对象，仅允许事先核实的平台对象；probe先清理。v20/v23名称预检在已核空库无历史数据，首轮不支持populated升级。全事务设置并核验UTC、UTF8、public和standard_conforming_strings。两installer互斥、故障回滚/重试、漂移均需实际证明；不得逐HTTP请求pg_advisory_lock/unlock。

管理SDK preview→push→task终态可传内存SQL，避免MCP在脏root目录生成文件。新function允许范围局限版本化审核工件，不关闭原scanner、不发布任意SQL执行器。真实变更失败不自动DROP业务表；隔离probe清理独立回执。

## Verification and Invalidation

通过现有Application/handler公共行为验证合法/非法/并发/重放，原生同条件作比较。真实DB旁路读取只用于证明审计/版本/事务不变量，不以复制实现断言自证。故障注入不进入公开RPC fail参数。40P01/40001/55P03按原有最多两次尝试，不泛化重试。

candidate、源、spec、SQL定义、schema、权限、配置任一变化使旧对应验证失效。实现/本地PASS、真UAT平台门、候选、独立验证、95/25完整UAT分别记录。尚无候选，不启动独立verifier。

## Follow-up Dependencies (not implemented here)

后续顺序：其余fulfillment locator/mark-ready通知→身份/登录/报价→paymentorder/refund/outbox/worker/billing/import/config/audit/一致快照读→全runtime/Bootstrap/真实secret容量/COS→CloudRun五worker持续运行与PC同源/Mini发布→真实UI3及95/25、恢复/告警/成本。4个1秒worker及1个小时worker不改成分钟cron。原Prod CVM/MySQL8/SSM决策不变。


## Next Complete Capability Slice

当前首切片收尾后，下一唯一最短切片为“订单履约闭环”：补齐MARK_READY、REDEEM_TOKEN、REDEEM_CURRENT_DATE_CODE，并实现orderquery.Application的ListUser/GetUser/SearchMerchant/GetMerchant/GetMerchantAtState读端口；MARK_READY涉及的订阅consent匹配及outbox入队必须与订单变更同RPC提交。REDEEM_ORDER复用本次已验证实现，不重写client、身份入口、44安装或readiness框架。扫码/date-code定位、订单锁与状态/receipt判断合为一次事务，不先用不一致查询猜目标。Go仍持有cipher和token语义，保留原handler/DTO/错误与各命令既有重放差异。

现有subscription.enqueueInTx与quote.FinalizeForPrepayInTx等sql.Tx耦合不能照搬跨HTTP：只在迁移对应业务能力时，把完整事务边界收进该固定RPC；不能做通用逐query事务shim。履约切片只要求outbox原子事实和原有consent/去重规则，真实微信发送仍由原Provider接口及后续worker切片负责，不把入队当送达。验收包括原生同条件差分、全部四种命令、token/date locator竞争、READY与通知同回滚/重放、用户与商户读权限、真实JWT/RLS并发及handler合同；不得为它单独启用不完整生产型后端。

剩余完整backend端口按依赖推进如下；每组只增加本能力固定RPC与必要RLS/ACL/readiness定义，不批量开放authenticated全库。

| 顺序 | 业务端口组 | 保留与原子边界 |
| --- | --- | --- |
| 1 当前之后 | fulfillment.Application完整四命令、orderquery.Application五读、通知事务入队 | token验证、当前商户授权、命令特定重放与订单/receipt/outbox同提交 |
| 2 | identity持久会话/手机绑定、merchantidentity与PC session/admin、受控Bootstrap | Order Bearer不变；外部WeChat provider不替换；账号禁用/auth_version与会话撤销一致 |
| 3 | catalog/storefront、merchantsoldout/storestatus、staffdiscount、objectstore.ConfirmationStore | 现有菜单/门店/日期售罄模型不变；图片须先确认再引用；售罄不是数量库存 |
| 4 | quote.Application与paymentorder的prepay快照事务 | 报价一致快照、幂等、expiry、当前价格/售罄验证及quote finalize/prepayment原子提交；不跨请求session advisory锁 |
| 5 | paymentorder/refund/orderadvance、subscription consent与delivery store | 外部微信调用留Go Provider；固定RPC claim/lease/fencing/结果落库，保留4个1秒worker，不在事务内等待外网 |
| 6 | billing、importbatch、adminreport、audit search/commands | 金额/账单事实、导入预检与提交边界、完整只读快照/游标、授权与审计保持；billing小时worker不改 |
| 7 | 唯一完整runtime composition、config/readiness/Bootstrap、私有存储、PC同源 | 全部应用端口完成后一次接线；真实secret map≤5KB、五worker常驻CPU/重启/滚动、COS最小身份与恢复、PC/Mini真实链路 |

后续仍需95Case/25页面与真实UI3、支付/订阅实际材料、恢复/告警/费用验证；业务代码可先用既有Provider seam独立开发，但不以fake provider验收真实外部完成。

## Storage Recovery Decision After Contract Review

不采用“不可变path+PG版本元数据”作为现有30日恢复合同的替代实现。immutable path能防应用覆盖，PG记录能追溯引用，但无法恢复物理删除/覆盖后的二进制；补复制、保留、GC及恢复流程将新增完整版本系统，并不能直接满足已确认bucket versioning原文。故当前保留私有COS与其版本控制/非当前30日保留目标，不实现自制版本子系统，也不要求用户重新选择既有决策。

pgstore作为内部Adapter本身不被禁止；只有平台底层真实版本/lifecycle映射及恢复后Storage元数据仍可访问得到验证，才可作为等价实现取代COS。当前正式PG API未暴露versioning/lifecycle/restore，FAQ默认无自动版本管理，上传TTL文案与schema也须实測，不猜字段。COS最小权限运行身份仍待证明，宽APIKey/未知可换管理员票据的CAM凭据均不能上线；这是最终runtime依赖BLOCKED，不阻断其他backend端口开发，也不降低权限或恢复硬门。
