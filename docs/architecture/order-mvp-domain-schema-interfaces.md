# Order MVP 统一领域事实、MySQL Schema 与接口冻结稿

## 0. 控制面

| 项 | 值 |
| --- | --- |
| 产品唯一基线 | `docs/product/online-ordering-system-prd-0818.md` |
| 设计固定点 | artifact base `9b96c64177f88bb0c148df20f91054a2ffc367c5`；authoritative integration `01cc4896c3c6c8531b1a2f08778deeccab3dd43b` |
| Schema / Interface 版本 | `ORDER-MVP-R2` |
| R2 revision / candidate lineage | `R2.1`；parent candidate `8ef6f8f1281af4a3e1df6abc8685ebdde0f3b53d` → 本次定点修正文档所在 commit（exact SHA 由提交回执绑定） |
| 当前状态 | `REVIEW_CANDIDATE`；两轴 Review 均为 0 finding 后才冻结 |
| 已集成迁移 | v1–v13 |
| TX-02 checkpoint | `eddf796454085ad0253a6db4a2f776ece8f51ea8`；v14–v17，WIP / NOT_CANDIDATE / NOT_INTEGRATED |
| 目标容量 | 单门店、4000–5000 用户；日 500–1000 单的本地/一期量级 |

本稿只裁决一期 MVP 的单一事实源、持久化边界、锁序、公共 Module Interface 和 HTTP DTO。所有三端实现必须读取或写入同一套服务端事实；前端 `window.__store`、`globalData`、本地 mock、微信前端支付回调都不是生产事实。

本稿不引入 Redis、MQ、ES、分库分表、分区、event sourcing、每页一表、库存、会员、优惠券、财务汇总表或看板汇总表。外部 provider 调用由同一进程内 worker + MySQL durable row、lease 和 CAS 驱动。

## 1. 领域语言与不变量

| 术语 | 唯一含义 | 禁止混同 |
| --- | --- | --- |
| 用户 | 通过微信小程序浏览和预约的人 | 客户、商户人员 |
| 员工用户 | 主手机号，或附加手机号 + 归一姓名，命中启用白名单的用户 | 商户人员、会员 |
| 商户人员 | 商户账号名单中的主账号或子账号 | 员工用户 |
| 预支付记录 | 调起微信支付前后的临时交易；未确认支付时不是订单、不占取餐号 | 待支付订单 |
| Payment Observation | 验签/解密或可信主动查询后，持久化的微信支付外部事实 | 订单、审计日志、原始回调 |
| 订单 | 服务端确认支付成功后，由不可变 Quote 快照原子投影出的六态交易 | 购物车、预支付记录 |
| 售罄 | 商品 × 取餐日期的不可售事实，可自由开关 | 库存为零、下架 |
| 取餐时间 | `Asia/Shanghai` 下的离散约定时刻 | 窗口、即时取餐 |
| 支付待处理 | 已支付但不能安全自动投影订单的查询口径 | 订单状态、异常订单 |
| 未取餐 | 营业日结束后仍为 `READY_FOR_PICKUP` 的查询口径 | 第七个订单状态 |
| Refund Observation | 验证后的微信退款回调/查询事实 | 退款请求、订单状态 |
| Notification Intent | 一次业务触发对应的一次 durable 外发意图 | 外部 exactly-once、订单状态事件 |

正式不变量：

1. 单门店、单取餐点，仅今天/明天预约，仅午餐/晚餐，固定 `Asia/Shanghai`。
2. 商品只有上下架与按取餐日期售罄；无数量库存、预占、扣减、返还。
3. 用户浏览不要求会话或手机号；首次提交前必须建立会话并绑定微信可信主手机号。
4. 附加手机号只有一组，必须与归一姓名同时命中同一启用白名单；员工与商户名单完全隔离。
5. 价格为整数分；逐商品折后单价先 half-up 到分，再乘数量；折扣率只能 `1..100`，应付必须 `>=1`。
6. Quote 保存服务端主手机号、联系人姓名、身份/白名单/折扣、商品/价格、商品封面 object key、门店/取餐事实及 digest；封面 object key 必须进入 snapshot digest。HTTP 只返回脱敏手机号，不返回完整手机号或内部 digest/version。
7. `quote.expires_at = min(created_at + 10m, pickup_at)`；数量只要求 `>0`，任何整数或金额溢出均 fail closed，不虚构 `<=99`。
8. 一份 Quote 只允许一个 durable prepayment；Provider Create 最多调用一次，任何歧义结果只能 Query。
9. Provider Create/Query/Refund/通知/对象存储调用永远在 DB transaction 外。
10. 微信回调先验签解密，再 durable Observation；只有提交成功才返回 204。回调请求不直接投影订单。
11. 可信 `success_time < effective_deadline` 才可自动投影；等于或晚于 deadline 一律进入人工 shield，不按 `received_at` 或 Apply 时钟判断。
12. 只有有效成功 Payment Observation 才能创建订单。取餐序列、订单、订单明细、Observation/Prepayment 标记必须同事务；失败不烧号。
13. 公开订单状态只有 `RESERVED / PREPARING / READY_FOR_PICKUP / COMPLETED / REFUNDING / REFUNDED`，只允许 PRD 单向转换。
14. 取餐号按取餐日期 `0001..9999`；手工 4 位码只匹配当前营业日期；二维码 token 仅待取餐生成、无时间过期、成功后一次性失效。
15. 退款只有原路全额；Provider 接受请求不等于成功，只有成功 Refund Observation 才进入 `REFUNDED`。
16. 订阅只有待取餐、退款结果两类；拒绝或发送失败绝不改变订单。
17. 每个写命令都重新校验当前身份/权限并使用幂等键；相同键返回首次结果，不重复副作用。
18. SQL、1205、1213 映射 `ErrUnavailable` 并可用新事务重试一次；snapshot/digest 损坏映射稳定 `ErrSnapshotInvalid`，不得混成可重试 SQL 错误。

## 2. PRD 场景到事实的闭合矩阵

| PRD 场景 | 持久化事实 | 派生，不落表 | 外部资产 / fail-closed 点 |
| --- | --- | --- | --- |
| 匿名浏览、会话 | 用户 openid；短期 session hash/expiry | 匿名展示态 | code 换取失败不伪造用户 |
| 个人中心 cosmetic / 客服 | 不新增业务字段 | 头像/昵称只取用户主动触发后微信客户端当次返回的 cosmetic profile；未取得时用中性默认头像/“微信用户”，不持久化、不参与身份/定价/RBAC/audit | 客服固定微信原生 `open-type="contact"`，真实会话仅 L4；cosmetic/客服失败不影响任何业务身份或订单 |
| 主手机号 | 微信可信主手机号及绑定时间 | masked phone | 客户端明文不可信；拒绝不写入 |
| 附加手机号 | 单组手机号、原始姓名、归一姓名、设置时间 | 是否命中员工 | 手机 + 姓名必须同时命中 |
| 员工折扣 | 白名单、启用、版本；全局折扣率/版本 | 已绑定、累计消费/单量 | 名单真实内容为 UAT 资产，不进 Gate 文档 |
| 商户小程序 | 商户账号、角色、启用、绑定 user、auth version | 可见导航 | 每次写操作校验 live account |
| PC 扫码登录 | challenge hash、TTL、批准主体、一次消费、access token hash | QR 画面 | 仅启用主账号；一次消费 |
| 商品/分类/图片 | 分类、商品、价格、餐段、上架、最多 3 个有序 object key | 封面 = 第 1 项 | 对象存储凭据外部；MySQL 不存 URL/密钥 |
| 营业/取餐 | singleton 门店设置、两餐段、`service_dates` | 今天/明天、离散时间点、最近可预约 | 日期行缺失或读取失败按不可预约 |
| 售罄 | `(service_date, product_id)` | 次日自然恢复 | 只影响目标日期 |
| 购物车/Quote | 购物车不存；Quote header + items 不可变快照/digest/expiry | 客户端购物车 | 当前事实漂移时拦截支付；折扣单独漂移允许旧 Quote |
| 发起支付 | Prepayment、不可变 Create 请求、durable `wx_request_payment`、deadline、lease/version | 支付中 UI | appid/mchid/key/cert 为外部配置；剩余不足 1m 不 Create |
| 回调/主动查询 | 规范化 Payment Observation、trusted success time、应用状态 | 回放响应 | 验签/解密先于持久化；无 durable row 不 204 |
| 对账兜底 | Prepayment reconciliation/manual 状态 + Observations | 支付待处理列表 | 自动失败不建异常订单 |
| 订单/取餐号 | 六态 order、不可变 header、items、每日 sequence | 进行中提示、统计 | 仅成功 Observation 投影；同事务不烧号 |
| 自动排产 | order 当前态及转换时刻 | due 列表 | 取餐前 30m；漏跑可重试 |
| 备好/核销 | order 状态、token AEAD ciphertext + lookup hash、key version、发放/消费操作者与时间 | QR 内容 | token 明文不落库；ciphertext 仅待取餐可解密展示，核销/退款即清空；跨日码不误核销 |
| 退款 | 全额 refund、不可变请求、lease/version | 退款待处理列表 | 真实资金/UAT 外部；未知时停 `REFUNDING` |
| 退款回调/查询 | Refund Observation、可信结果时间、应用状态 | 回放响应 | 成功 Observation 才 `REFUNDED` |
| 订阅 | consent 决策；outbox intent/lease/结果 | 补订阅入口 | template 审批外部；失败不动订单 |
| XLSX 导入 | 单 batch 的 digest、预览/错误 JSON、计数、提交结果 | 逐行任务 | 服务端解析；未确认零业务写；无 row/task 表 |
| 审计 | 统一 action audit：actor/action/target/result/reason/idempotency/before/after | 状态历史展示 | 不存 raw PII/provider payload；不用于重建业务状态 |
| 财务/对账/看板/未取餐 | 微信账单原文不落库；本次 bill digest、日期、匹配/单边计数写 `action_audits`，单边支付/退款只更新既有 pending 事实 | orders/items/prepayments/payment observations/refunds 实时账单、逐笔对账、统计/导出 | 账单拉取走 WeChatPay Adapter；只读查询不得修改交易事实，不建账单/汇总表 |

## 3. 三端单一事实源

| 事实 | PC Admin 写/读 | 商户小程序写/读 | 用户小程序写/读 | 唯一服务端 owner |
| --- | --- | --- | --- | --- |
| 分类、商品、图片、上下架 | CRUD/导入 | 只读 + 当日售罄 | 浏览 | `Catalog` |
| 门店、餐段、日期营业、开屏 | 配置 | 现场营业状态 | 公开读取 | `Storefront` |
| 员工名单、折扣 | CRUD/导入/统计 | 无 | 身份/报价读取 | `StaffDiscount` |
| 商户账号、PC 会话 | CRUD/扫码登录 | 绑定/鉴权 | 仅显示是否可切换 | `MerchantAuth` |
| Quote | 只读订单快照来源 | 无 | 创建/读取 | `Quote` |
| Prepayment/Payment Observation | 待处理/财务 | 无 | Prepare/Confirm | `PaymentOrder` |
| Order/Items/sequence | 搜索/财务/人工处理 | 泳道/履约 | 我的订单 | `Order` + `Fulfillment` |
| Refund/Observation | 发起/待处理/财务 | 无 | 规则内取消 | `Refund` |
| Consent/Outbox | 只读审计 | 触发备好通知 | 主动授权 | `Subscription` |
| Import batch | 预览/提交 | 无 | 无 | `Import` |
| Audit | 主账号查询 | 写命令产生 | 写命令产生 | `Audit` |

任何页面只能通过上述 Module 的 HTTP Adapter 读写；不允许另建 PC 专用 product/store/order 模型。

## 4. 迁移现状与冻结账本

### 4.1 已有 v1–v17

| 版本 | 状态 | 表/事实 | 冻结裁决 |
| --- | --- | --- | --- |
| v1 | integrated | `schema_migrations` | 保留 checksum、dirty、MySQL named migration lock |
| v2 | integrated | `categories` | v19–v21 分阶段补 bounded name key、version、约束 |
| v3–v4 | integrated | `products` + meal enum | v22–v24 分阶段补最多 3 个有序图片对象、bounded name key、version、price > 0 |
| v5–v6 | integrated | `meal_periods` + 两行 seed | 保留独立配置事实，不迁入 singleton |
| v7 | integrated | `product_sold_out_dates` | 保留复合 PK；不建库存 |
| v8–v10 | integrated | `miniprogram_users`、`miniprogram_sessions`、主手机号 | 保留 opaque revocable session；v18 补单组附加手机号 |
| v11 | integrated | `storefront_settings` | v25–v27 把旧 launch URL fail closed 清空后切换 object key，补全局口味 JSON/version |
| v12 | integrated | `merchant_accounts` | v28 补不可恢复 soft-delete事实，保护历史 FK |
| v13 | integrated | `merchant_action_audits` | v41–v44 rename/backfill/constrain 为唯一 `action_audits`，不建第二张审计表、不丢旧行 |
| v14 | TX-02 WIP | `staff_whitelist` | 集成前直接补 `name_key`；phone 唯一，姓名仅用于双要素 |
| v15 | TX-02 WIP | `discount_settings` | 保留 singleton，rate `1..100`，未初始化 Quote fail closed |
| v16 | TX-02 WIP | `quotes` | 集成前补持久化 `expires_at`；保留完整 header 快照/digest |
| v17 | TX-02 WIP | `quote_items` | 保留行级不可变快照；无 product FK，历史不受商品删除影响 |

v9、v5、v15、v17 虽可在空白设计中折叠，但已经分别承担 token 撤销、餐段独立配置、折扣锁/版本、行级 digest 与金额完整性。为减少表数而新增迁移搬运/删表会增加停机、回滚和双模型风险，不满足“最短正确路径”。

最终物理模型为 25 张业务表（不含 `schema_migrations`）：

| 分类 | 表 | 来源 |
| --- | --- | --- |
| 当前身份/权限 | `miniprogram_users, miniprogram_sessions, merchant_accounts, staff_whitelist, discount_settings, merchant_pc_sessions` | v8–v10/v12、TX-02 v14–v15、new v30 |
| 当前目录/营业 | `categories, products, meal_periods, storefront_settings, service_dates, product_sold_out_dates` | v2–v7/v11、new v29 |
| 交易/履约 | `quotes, quote_items, prepayments, payment_observations, pickup_sequences, orders, order_items` | TX-02 v16–v17、new v31–v35 |
| 退款/通知 | `refunds, refund_observations, notification_consents, notification_outbox` | new v36–v39 |
| 批量/证据 | `import_batches, action_audits` | new v40、v13 原地升级 v41–v44 |

### 4.2 v18–v44 唯一 ledger

旧 TX-03 对 v18–v23 的预留作废；Review 冻结后由 migration-ledger 单 writer 按下列顺序创建。现有 runner 把每个 migration 文件作为一次 `Exec` 执行且 DSN 不开放 multi-statements，因此每个版本只能有一个 `ALTER`、`UPDATE`、`RENAME TABLE` 或 `CREATE TABLE` 语句；API 只在最终 schema version ready 后启动，中间版本均 fail closed。v20/v23 另注册只读 Go preflight hook：runner 已持有同一 connection-scoped migration lock、但尚未 `insertDirty` 时执行；任一失败直接返回稳定 `migration_preflight_failed`，不写 dirty/history、也不执行 SQL。

| 版本 | DDL 责任 |
| --- | --- |
| v18 | ALTER `miniprogram_users`：附加手机号/姓名/`extra_name_key VARBINARY(400)`/设置时间/版本；新增列初始全 NULL |
| v19 | ALTER `categories`：新增 nullable `name_key VARBINARY(400)` 与 `record_version` |
| v20 | Go preflight 用冻结 NFKC 算法逐 id 证明 legacy category name 已经等于 canonical（NFKC + Unicode trim）、UTF-8 bytes `1..400` 且 canonical key 无重复；随后 SQL 仅 `name_key=CONVERT(name USING binary)` 复制原 bytes，不在 SQL 内归一/改名 |
| v21 | ALTER `categories`：`name_key NOT NULL`、UNIQUE、trim/version CHECK |
| v22 | ALTER `products`：新增 nullable `name_key VARBINARY(400)`、record version、`images_json` |
| v23 | 同 migration-lock / pre-dirty Go preflight 证明 legacy product name canonical、bounded、无重复；随后 SQL 仅复制原 name bytes 到 key 并把已有图片置空数组，不从 URL 猜 object key |
| v24 | ALTER `products`：name key NOT NULL/UNIQUE、price/name/JSON 约束 |
| v25 | ALTER `storefront_settings`：新增 nullable launch object key、flavors JSON、record version |
| v26 | UPDATE `storefront_settings`：把旧 `launch_png_url` 及其几何字段全部清空；旧 URL 不转换、不继续展示，fail closed |
| v27 | ALTER `storefront_settings`：删除旧 URL 列，建立 object-key/几何 exact group CHECK |
| v28 | ALTER `merchant_accounts`：`deleted_at/deleted_by`、不可恢复删除语义及 RESTRICT self-FK |
| v29 | CREATE `service_dates` |
| v30 | CREATE `merchant_pc_sessions` |
| v31 | CREATE `prepayments` |
| v32 | CREATE `payment_observations` |
| v33 | CREATE `pickup_sequences` |
| v34 | CREATE `orders` |
| v35 | CREATE `order_items` |
| v36 | CREATE `refunds` |
| v37 | CREATE `refund_observations` |
| v38 | CREATE `notification_consents` |
| v39 | CREATE `notification_outbox` |
| v40 | CREATE `import_batches` |
| v41 | RENAME TABLE `merchant_action_audits` TO `action_audits`；旧读写路径在应用切换前保持 schema-behind，不能启动 |
| v42 | ALTER `action_audits`：旧列 rename 到通用列并新增 `entry_kind LEGACY_EVIDENCE|COMMAND_RECEIPT|SYSTEM_EVIDENCE`、actor kind/scope、nullable operation key、response/JSON 列；live account 与 account snapshot 保持为两个独立概念 |
| v43 | UPDATE `action_audits`：旧行标 `LEGACY_EVIDENCE`，request 计算 hash；已有 idempotency hash 才复制为 operation key，否则保持 NULL；允许 `actor_account_id` NULL 而 account id/role/auth version snapshot 非空，逐行保留原 id/occurred_at/result/reason |
| v44 | ALTER `action_audits`：exact entry-kind/actor/snapshot CHECK、nullable-key UNIQUE、live user/account 列各自补 RESTRICT FK；仅新 `COMMAND_RECEIPT` 强制 operation key + replayable response，legacy 不补造 live FK/幂等键 |

迁移 runner 继续单连接 `GET_LOCK`；任一 dirty、schema-behind 或 v41–v43 中间态阻止 API 启动。不得并行分配迁移号。v20/v23 preflight 必须在同一 migration lock 下、写 dirty 前完成；SQL 只复制已证明 canonical 的 bytes。归一改变、越界或重复都拒绝迁移并人工修正源数据，禁止静默改名。

## 5. 冻结 MySQL 数据模型

除特殊说明外：InnoDB、MySQL 8、`utf8mb4_0900_ai_ci`；业务 ID 为 `BIGINT UNSIGNED AUTO_INCREMENT`；时间为 UTC `TIMESTAMP(6)`，营业日期/时刻另用 `DATE` / `TIME`；金额为 `BIGINT UNSIGNED` 整数分；手机号为 E.164 `VARBINARY(16)`；hash 为 `BINARY(32)`。应用在入库前完成 UTF-8、长度、NFKC、空白与 JSON element 校验，DB 再用 CHECK/UNIQUE/FK fail closed。

### 5.1 当前配置与身份

| 表 | 必须字段 | 关键约束与索引 | PII / 生命周期 |
| --- | --- | --- | --- |
| `miniprogram_users` | existing id/openid/login；primary phone/bound at；`extra_phone, extra_name, extra_name_key VARBINARY(400), extra_phone_set_at, record_version` | openid、primary phone 唯一；extra 四字段全空或全非空；两 phone E.164；version > 0；所有 user FK 为 RESTRICT | openid、phone、name 为 PII；不得进 DTO/日志，phone 仅 masked |
| `miniprogram_sessions` | token_hash, user_id, issued_at, expires_at | PK token hash；FK user `ON UPDATE/DELETE RESTRICT`；`expires>issued` | bearer 只返回一次；DB 仅 hash；过期清理 |
| `merchant_accounts` | existing phone/name/role/enabled/binding/version；`deleted_at, deleted_by_account_id` | phone 唯一；binding 成组；deleted 后 enabled=false 且 binding 清空；binding user、deleted_by account 均 RESTRICT FK；最后一个 enabled OWNER 由锁事务守护 | soft-delete 对业务不可恢复，保留历史 actor FK |
| `staff_whitelist` | phone, name, `name_key VARBINARY(400)`, enabled, record_version, created/updated | phone 唯一；E.164；name/key 非空 | phone/name PII；导入覆盖 name 但保留 created/enabled |
| `discount_settings` | singleton, rate_percent, discount_version, whitelist_version, updated_at | id=1；rate 1..100；versions >0 | 无 PII；任一名单写递增 whitelist_version |
| `merchant_pc_sessions` | id, `approval_secret_hash, poll_secret_hash`, state, login_expires_at, approved account/user/auth version/time, consumed_at, access_token_hash/access_issued_at/access_expires_at, created/updated | secret/access hash 均 unique；account/user 全部 RESTRICT FK；`login_expires_at=created_at+2m`；access expiry=`issued+12h`；状态 exact CHECK 见 §5.8 | QR 只含 approval secret，浏览器单独持有 poll secret；2 分钟内 approve+poll，token 只返回一次；12 小时绝对过期、无 refresh/remember-device；并发 session 互不踢出且每次 live-check account |

姓名 `name_key` 的唯一算法为：有效 UTF-8 → Unicode NFKC → 删除全部 Unicode whitespace → 非空 → UTF-8 bytes；附加姓名与白名单必须调用同一实现后 byte-exact 比较。分类/商品 `name_key` 仅做 NFKC + 去首尾 Unicode whitespace，保持中间空格，再 byte-exact 唯一。

### 5.2 目录、营业与售罄

| 表 | 必须字段 | 关键约束与索引 |
| --- | --- | --- |
| `categories` | existing id/name/sort/is_active；`name_key VARBINARY(400), record_version` | UNIQUE name_key；trim/nonempty；version>0；不增加范围索引 |
| `products` | existing id/category/name/description/specification/price/sort/listed/meal；`name_key VARBINARY(400), record_version, images_json` | FK category `ON UPDATE/DELETE RESTRICT`；UNIQUE name_key；price>0；JSON 为 0..3 个有序且 object_key 唯一的 `{\"object_key\":string}`；`specification` 只读、PC/API/导入均不可改；不建 image 表/范围索引 |
| `meal_periods` | code lunch/dinner, cutoff, pickup start/end, interval | PK code；仅两 code；start<=end；interval>0 且离散点可生成；所有更新锁两行固定 lunch→dinner |
| `storefront_settings` | singleton；store/address/pickup/announcement/status；`launch_image_object_key` + position/size；`flavor_options_json, record_version` | id=1；status open/closed/cutoff；launch key 与几何字段成组；flavors 为去重字符串 array；version>0 |
| `service_dates` | service_date, is_open, record_version, updated_by_account_id, updated_at | PK date；boolean/version CHECK；FK merchant account `ON UPDATE/DELETE RESTRICT`；缺行即 closed；不增加范围索引 |
| `product_sold_out_dates` | service_date, product_id | PK `(service_date,product_id)`；FK product RESTRICT；读取失败按 sold out；不保存数量；操作者/时间进入统一 audit |

`product_sold_out_dates` 物理 v7 保持不变；不为页面展示增加重复列。

### 5.3 Quote

`quotes` 保持 v16 header 列，并新增 `expires_at TIMESTAMP(6) NOT NULL`；`UNIQUE(user_id,idempotency_key_hash)`；FK user RESTRICT；contact name/phone、identity kind/source version、discount rate/version、门店/地址/取餐点、pickup date/time/meal、order note、item count、三项金额、snapshot digest、created/expires 均不可变。CHECK：联系信息规范；rate 1..100；item count >0；`original=discount+payable`；payable>0；`expires_at>created_at`。

`quote_items` 保持 v17 并在 TX-02 集成前补 `image_object_key_snapshot VARBINARY(1024) NULL`：PK `(quote_id,line_number)`；FK quote `ON UPDATE/DELETE RESTRICT`；product id 仅作历史引用、不设 FK；name/source digest、封面 object key、两种 unit price、quantity、两种 subtotal、flavors JSON、line note 均不可变。CHECK quantity>0、line>0、折后价<=原价、乘法不溢出且 subtotal 精确、flavors 为 array。每个 flavor 必须是当前 singleton 全局口味选项中的唯一成员；口味配置漂移属于 current-fact drift。Header/item 汇总及每行 `image_object_key_snapshot` 必须进入同一 versioned canonical digest，并由同一 transaction 生成和复核；产品无图片时 snapshot 为 NULL，之后补图不改历史 Quote。

### 5.4 Prepayment 与 Payment Observation

`prepayments`：

- identity：`id, user_id RESTRICT FK, quote_id UNIQUE RESTRICT FK, idempotency_key_hash, out_trade_no UNIQUE`；`UNIQUE(user_id,idempotency_key_hash)`。
- immutable expected facts：`expected_appid, expected_mchid, expected_amount_cents, currency='CNY', provider_create_request_json, provider_create_request_digest, effective_deadline`。
- provider result：`provider_state ENUM('READY','CREATE_CLAIMED','CREATE_UNKNOWN','PAYMENT_REQUESTED','NOT_PAID','PAID','CLOSED')`，`create_attempted_at`，`wx_request_payment_json`，`provider_prepay_id`，`last_queried_at`。
- materialization：`materialization_state ENUM('AWAITING_PAYMENT','READY','APPLIED','PENDING_MANUAL','REFUNDED_WITHOUT_ORDER')`，`pending_reason_code, materialized_at nullable`。
- concurrency：`lease_kind ENUM('CREATE','QUERY') nullable, lease_owner BINARY(16), lease_expires_at, record_version, next_reconcile_at, created_at, updated_at`；lease 三字段全空或全非空；version>0；仅建范围索引 R1 `(next_reconcile_at,id)`，claim 后再按 state/CAS 筛选。

规则：第一次持久化 Create request 后不可修改。唯一 Create claim transaction 以 CAS 同时完成 `READY→CREATE_CLAIMED`、写 `create_attempted_at`、设置 CREATE lease、递增 version 后提交，才允许 lease owner 在事务外调用 Create；这一步一旦提交就视为可能已外调。进程在外调前/中崩溃、lease 过期或 Create 超时都只能 Query，分别从 `CREATE_CLAIMED/CREATE_UNKNOWN` 收敛，永不再次 Create。只有距离 effective deadline 至少 1 分钟才可 claim。Provider/Observation 状态只能按可信度与终态单调前进，乱序 callback/query 不得把 `PAID` 回退为 `NOT_PAID/CLOSED`。

`payment_observations`：

- `id, prepayment_id FK, dedupe_key UNIQUE, source CALLBACK|ACTIVE_QUERY, provider_event_id nullable, out_trade_no, transaction_id nullable, provider_state PAID|NOT_PAID|CLOSED`。
- `validation MATCH|MISMATCH`、稳定 `mismatch_code`、`amount_cents, currency, success_time nullable, received_at`。
- `materialization_mode AUTO|DELAYED_MANUAL`。Observation DDL CHECK 只约束同表 nullable group：AUTO 必须 MATCH+PAID 且 success_time 非空；它不能跨表比较 deadline。ingress/apply transaction 必须先 `prepayments ... FOR UPDATE`，再以该锁行的 `effective_deadline` 判定 trusted `success_time < deadline`；等于/晚于、字段不匹配或快照异常均为 DELAYED_MANUAL，且不得使用 `received_at`。fresh MySQL 测试固定覆盖 `< / = / >` 三边界及乱序重放。
- `apply_state NEW|APPLIED|DEFERRED, apply_reason_code, applied_at, record_version`；UNIQUE non-null transaction id；FK prepayment `ON UPDATE/DELETE RESTRICT`；不增加范围索引。

不保存 raw callback、签名头、解密明文或证书。无效签名/解密失败不形成可信 Observation；已验证但字段 mismatch 必须 durable 并进入 shield。

### 5.5 Order、履约与取餐序列

`pickup_sequences(service_date PK, last_number, updated_at)`，CHECK `0<=last_number<=9999`。订单创建事务在锁完 Observation/Prepayment 后，用同一连接原子执行 `INSERT ... VALUES(service_date,LAST_INSERT_ID(1),now) ON DUPLICATE KEY UPDATE last_number=LAST_INSERT_ID(last_number+1),updated_at=VALUES(updated_at)`，随即在同 transaction 读取 `LAST_INSERT_ID()` 并插入 order/items；首次日期初始化为 1，并发日期行由主键串行化，超过 9999 触发 CHECK 整笔回滚。sequence 与 order 同 tx，事务回滚不烧号；禁止先建 sequence 行再另事务建单。

`orders`：

- identity：`id, order_no UNIQUE, user_id FK, quote_id UNIQUE, prepayment_id UNIQUE, payment_observation_id UNIQUE, record_version`。
- immutable header：Quote 的 contact/identity/discount/store/address/pickup/date/time/meal/note/item_count/三项金额/snapshot digest 全量复制；另存 `pickup_at UTC, transaction_id UNIQUE, paid_at, materialized_at`。
- fulfillment：`pickup_number`，`state` 六态，`preparing_at, ready_at, completed_at, refunding_at, refunded_at`。
- redemption：`redemption_token_ciphertext VARBINARY(192) nullable, redemption_token_hash BINARY(32) UNIQUE nullable, redemption_key_version SMALLINT UNSIGNED nullable, redemption_issued_at, redeemed_by_account_id nullable, redeemed_at`。plaintext 使用 CSPRNG，ciphertext 为带 nonce/tag 的版本化 AEAD envelope；hash 只用于 lookup/幂等，只有 READY 可解密展示，核销或进入退款即清空 ciphertext/key version并保留 hash 防重放。
- constraints/indexes：UNIQUE `(pickup_date,pickup_number)`；pickup 1..9999；token/timestamp exact CHECK 见 §5.8；FK user/quote/prepayment/observation/account 全部 `ON UPDATE/DELETE RESTRICT`；仅建范围索引 R2 `(state,pickup_at,id)`。

`order_items`：PK `(order_id,line_number)`，FK order `ON UPDATE/DELETE RESTRICT`；逐字复制 Quote item（含 `image_object_key_snapshot`）；不 FK 当前 product、不建销量索引，小规模销量按 order 日期范围扫描聚合。

订单初态必须先 `InitialState(success_time,pickup_at)`，再 `Advance(initial,materialized_at,pickup_at)`，同一事务补赶到 `PREPARING`；不得直接凭 Apply 当前时钟跳过 policy。

### 5.6 Refund 与 Refund Observation

`refunds`：

- `id, prepayment_id UNIQUE RESTRICT FK, order_id UNIQUE nullable RESTRICT FK, out_refund_no UNIQUE, idempotency_key_hash, amount_cents, currency='CNY', reason_code, requested_by_user/account nullable RESTRICT FK, created_at`。
- 不可变 `provider_request_json/digest`；`provider_state READY|CREATE_CLAIMED|CREATE_UNKNOWN|PROCESSING|SUCCESS|CLOSED`；`create_attempted_at, provider_refund_id nullable`。唯一 Create claim transaction 同时 CAS `READY→CREATE_CLAIMED`、写 attempted_at、CREATE lease/version 后提交；其后 provider 在 tx 外调用，任何崩溃/超时/过期 lease 恢复都只 Query，永不第二次 Create。
- `materialization_state AWAITING_PROVIDER|APPLIED|PENDING_MANUAL`、`pending_reason_code, materialized_at nullable`；Create/Query lease 三字段、record version、next query、updated_at；不增加范围索引，MVP 退款行按 PK 小批 CAS 扫描。
- CHECK 恰有 user 或 merchant requester；order 存在时 amount 必须等于 order payable，支付待处理退款则等于 prepayment expected amount，由事务校验；一笔 prepayment 最多一笔全额退款。

`refund_observations`：`id, refund_id RESTRICT FK, dedupe_key UNIQUE, source, provider_event_id, out_refund_no, provider_refund_id, provider_state PROCESSING|SUCCESS|CLOSED, validation, mismatch_code, success_time, received_at, apply_state NEW|APPLIED|DEFERRED, apply_reason_code, applied_at, record_version`；不增加范围索引。只有 MATCH + SUCCESS 可把有订单的交易从 `REFUNDING` 推进到 `REFUNDED`。

### 5.7 Subscription、Import 与统一 Audit

`notification_consents`：`id, order_id RESTRICT FK, user_id RESTRICT FK, kind READY|REFUND_RESULT, grant_sequence, decision ACCEPTED|REJECTED, template_config_version, idempotency_key_hash, decided_at, consumed_at nullable`；UNIQUE `(order_id,kind,grant_sequence)` 和 `(user_id,idempotency_key_hash)`；accepted 才可消费，一个 consent 最多关联一个 outbox。

`notification_outbox`：`id, order_id RESTRICT FK, consent_id UNIQUE RESTRICT FK, kind, recipient_user_id RESTRICT FK, immutable_message_json, template_config_version, state PENDING|IN_FLIGHT|SENT|FAILED_PERMANENT, attempt_count, next_attempt_at, lease_owner/expires, record_version, provider_message_id nullable, last_error_code nullable, created/sent_at`；UNIQUE `(order_id,kind)`；仅建范围索引 R3 `(state,next_attempt_at,id)`。发送调用在 transaction 外；本地唯一只保证单一 intent，不宣称微信 exactly-once。

`import_batches`：`id, kind PRODUCT|STAFF, actor_account_id RESTRICT FK, file_object_key nullable, file_digest, file_size_bytes, row_count, preview_token_hash UNIQUE, preview_json, new_count, update_count, error_count, state PREVIEWED|COMMITTED|EXPIRED, commit_idempotency_key_hash nullable, skip_invalid nullable, result_json nullable, expires_at, committed_at, record_version, created/updated`；UNIQUE `(actor_account_id,commit_idempotency_key_hash)`（nullable）；CHECK `1<=file_size_bytes<=10485760`，PRODUCT `1<=row_count<=500`，STAFF `1<=row_count<=5000`；不增加范围索引。preview/commit 只用这一行；staff PII preview 在 commit/expiry 后清空，临时 object 由对象存储生命周期清理。

v41–v44 将 `merchant_action_audits` 分阶段原地变为唯一 `action_audits`：

- `id, entry_kind LEGACY_EVIDENCE|COMMAND_RECEIPT|SYSTEM_EVIDENCE, actor_kind USER|MERCHANT|SYSTEM|PROVIDER, actor_scope_hash, actor_user_id nullable, actor_account_id nullable, actor_account_id_snapshot nullable, actor_role_snapshot nullable, actor_auth_version nullable`。
- `action, target_type, target_id nullable, target_key_hash nullable, operation_key_hash, request_id_hash nullable, result SUCCEEDED|REJECTED, reason_code, before_state_json, after_state_json, response_json, occurred_at`。
- UNIQUE `(actor_scope_hash,action,operation_key_hash)`；NULL operation key 只允许 legacy/system evidence且不参与去重；live actor user/account 非空时各自为 RESTRICT FK，snapshot 不是 live FK；不增加范围索引，按 PK cursor + exact target/actor 过滤。
- CHECK：新 COMMAND_RECEIPT 的 USER 需 live user，新 MERCHANT 需 live user+account，且 operation key/response 均非空；LEGACY_EVIDENCE 允许 operation key NULL，也允许只有 account snapshot 而无 live account；SYSTEM/PROVIDER live user/account 均空。snapshot 三字段全空或全非空。JSON 只含最小状态和可重放的非 PII 结果；不保存手机号、姓名、openid、raw provider payload。

`action_audits` 同时是统一证据流与 authenticated business command 的 durable receipt，但不是 aggregate 状态源。业务 mutator 先锁/修改 aggregate，最后 insert `COMMAND_RECEIPT`；若 `(actor_scope,action,operation_key)` duplicate，整个业务 transaction 必须 rollback，随后在无业务锁的新只读 transaction 读取既有 `response_json` 并原样 replay。禁止先锁 receipt 再锁业务行；认证/provider intrinsic dedupe 仍由各自事实表承担。订单、配置、退款等当前状态永远从 aggregate 读取，禁止用 audit/receipt 重建。

### 5.8 状态 / nullable CHECK 真值表

DDL 的 CHECK 必须逐行等价于下表，不能只检查“若干字段一起为空”。`NN`=NOT NULL，`N`=NULL，`PAIR`=两列同空同非空；时间先后同时 CHECK。

| Aggregate / state | 必须 NN | 必须 N / 其他精确规则 |
| --- | --- | --- |
| PC `WAITING` | login secrets、created/login expiry | approval、consume、access 全 N |
| PC `APPROVED` | approval account/user/auth version/time | consume、access 全 N；`approved_at<=login_expires_at` |
| PC `CONSUMED` | approval 全组、consumed_at、access hash/issued/expiry | 无 refresh/remember 字段；`consumed_at<=login_expires_at`，`access_expires_at=access_issued_at+12h` |
| Prepayment `READY` | immutable request/deadline/version | create_attempted/prepay id/wx payment N |
| Prepayment `CREATE_CLAIMED` | create_attempted_at、CREATE lease owner/expiry | prepay id/wx payment N；恢复路径只允许 Query |
| Prepayment `CREATE_UNKNOWN` | create_attempted_at | prepay id/wx payment N |
| Prepayment `PAYMENT_REQUESTED` | create_attempted_at、prepay id、wx payment | 三者不可部分存在 |
| Prepayment `NOT_PAID/PAID/CLOSED` | create_attempted_at | prepay id 与 wx payment 为 PAIR；`PAID/CLOSED` 不可回退 |
| Prepayment materialization `AWAITING_PAYMENT/READY` | state | pending_reason/materialized_at N |
| Prepayment materialization `APPLIED/REFUNDED_WITHOUT_ORDER` | materialized_at | pending_reason N |
| Prepayment materialization `PENDING_MANUAL` | pending_reason | materialized_at N |
| Payment/Refund Observation `validation=MATCH` | normalized identity fields | mismatch_code N；`MISMATCH` 则 mismatch_code NN |
| Payment Observation `PAID` | transaction_id、amount、currency、success_time | 非 PAID 的 transaction_id/success_time N；DDL 对 AUTO 只检查 MATCH/PAID/success_time 同表组，deadline 由锁 prepayment 的 ingress/apply tx 判定 |
| Observation `NEW` | — | apply_reason/applied_at N |
| Observation `APPLIED` | applied_at | apply_reason N |
| Observation `DEFERRED` | apply_reason | applied_at N |
| Order `RESERVED` | — | preparing/ready/completed/refunding/refunded、全部 token 字段 N |
| Order `PREPARING` | preparing_at | ready/completed/refunding/refunded、全部 token 字段 N |
| Order `READY_FOR_PICKUP` | preparing/ready、ciphertext/hash/key version/issued | completed/refunding/refunded/redeemed N |
| Order `COMPLETED` | preparing/ready/completed、token hash/issued、redeemed account/time | ciphertext/key version/refund times N |
| Order `REFUNDING/REFUNDED` | refunding_at；REFUNDED 再要求 refunded_at | ciphertext/key version N；hash/issued 为 PAIR；redeemed account/time/completed 为三列同空同非空；历史 `ready=>preparing`，`completed=>ready` |
| Refund `READY` | immutable request | create_attempted/provider refund id N |
| Refund `CREATE_CLAIMED` | create_attempted_at、CREATE lease owner/expiry | provider refund id N；恢复路径只允许 Query |
| Refund `CREATE_UNKNOWN` | create_attempted_at | provider refund id N |
| Refund `PROCESSING/SUCCESS/CLOSED` | create_attempted_at、provider refund id | 终态不可回退；PENDING_MANUAL iff pending_reason NN，APPLIED iff materialized_at NN |
| Outbox `PENDING` | next_attempt_at | lease/sent_at/provider_message N |
| Outbox `IN_FLIGHT` | lease owner/expiry | sent_at/provider_message N |
| Outbox `SENT` | sent_at | lease/error N；provider_message_id 可空 |
| Outbox `FAILED_PERMANENT` | last_error_code | lease/sent_at/provider_message N |
| Import `PREVIEWED` | preview JSON/expiry | commit key/skip/result/committed N |
| Import `COMMITTED` | commit key/skip/result/committed | staff preview JSON 必须清空；committed_at<=expiry |
| Import `EXPIRED` | state | preview/commit/skip/result/committed 全 N |
| Audit `LEGACY_EVIDENCE` | actor kind/scope、action/result/reason/time | operation key/response 可 N；live account 可 N；account snapshot 三列全空或全非空，且可在无 live account 时 NN |
| Audit `COMMAND_RECEIPT` | authenticated live actor、operation key、response、action/result/time | USER 的 account N；MERCHANT 的 live account NN+RESTRICT FK；snapshot 可选但成组 |

除 v1–v17 已集成目录读取索引及 PK、UNIQUE、FK 所需定点索引外，v18–v44 只新增三个非唯一范围索引：R1 `prepayments(next_reconcile_at,id)`、R2 `orders(state,pickup_at,id)`、R3 `notification_outbox(state,next_attempt_at,id)`。4000–5000 用户量级的财务、退款、审计、销量和导入过期查询用主键 cursor 小批扫描/日期条件，不为报表增加写放大索引。

## 6. Derived 查询与明确不建表

| Derived | 来源与必要索引 |
| --- | --- |
| 当前营业、最近可预约、时间点 | storefront + service_dates + meal_periods + now |
| 员工已绑定/累计消费/累计单量 | users + staff whitelist + paid orders/refunds |
| 进行中订单、待制作、未取餐 | orders PK/owner FK + R2 state/pickup |
| 支付待处理 | prepayments.materialization_state |
| 退款待处理 | refunds.materialization_state / provider_state |
| 当日/月营收、订单、退款额 | orders + refunds；`REFUNDING/REFUNDED` 分口径 |
| 商品销量排行 | order_items join orders；只计有效口径 |
| 财务明细/CSV | orders transaction/payment snapshots + refunds |

明确不建：carts、product_images、inventory、stock_reservations、members、levels、coupons、order_state_events、pending_payments、unclaimed_orders、refund_items、dashboard_stats、finance_daily、product_sales、staff_stats、import_rows/tasks、provider raw inbox、对象存储元数据表。

## 7. 公共 Module Interfaces

Module 不导出 MySQL Repository interface，不 import Gin/HTTP DTO。`staffidentity`、`quotepricing`、`paymentobservation`、`orderproduction` 作为对应深 Module 内部纯实现保留。公共写入元数据统一为：

```go
type WriteMeta struct {
    ActorUserID    uint64
    IdempotencyKey string
    RequestID       string
}

type PageQuery struct { AfterID uint64; Limit uint16 } // Limit 1..100
```

`WriteMeta` 是所有已认证、用户触发业务 mutator 的唯一元数据；Module 必须验证 `ActorUserID` 与认证 principal 一致，并以 `(actor scope, command, IdempotencyKey)` 在 action receipt 持久化首次非 PII 响应。认证 code exchange/QR consume 不接受客户端幂等键，分别以 provider code digest、`login_id+secret_hash` intrinsic dedupe；provider ingress 以 event/transaction intrinsic dedupe；worker 以 row id+record version/lease intrinsic dedupe。这三类例外不得伪造 `WriteMeta`。

| Module | 冻结入口 |
| --- | --- |

| `Identity` | `IssueSession(ctx, LoginCode) (Session,error)`；`Authenticate(ctx, Token) (UserID,error)`；`View(ctx, UserID) (UserIdentity,error)`；`BindPrimaryPhone(ctx, UserID, PhoneCode) (UserIdentity,error)`；`SetExtraPhone(ctx, WriteMeta, ExtraPhoneClaim) (UserIdentity,error)`；前两项 mutation 使用 provider-code intrinsic dedupe |
| `MerchantAuth` | `Identity(ctx,UserID)`；`LoginMini(ctx,UserID,PhoneCode)`；`BeginPCLogin(ctx)`；`ApprovePCLogin(ctx,UserID,LoginID,ApprovalSecret,PhoneCode)`；`PollPCLogin(ctx,LoginID,PollSecret)`；`AuthenticatePC(ctx,Token)`；`ExecuteAccount(ctx,WriteMeta,AccountCommand)`；`AuthorizeInTx(ctx,tx,UserID,Action,Target)`；auth/QR 用 intrinsic dedupe，账号业务命令必须 WriteMeta |
| `Storefront` | `Public(ctx,at) (PublicStorefront,error)`；`PickupOptions(ctx,at)`；`ResolvePickupInTx(ctx,tx,Selection,observedAt)`；`Configure(ctx,WriteMeta,SettingsCommand)`；`SetBusinessStatus(ctx,WriteMeta,Status)` |
| `Catalog` | `Browse(ctx,MenuQuery{Date,Time},OptionalUserID)`；`Detail(ctx,ProductDetailQuery{ProductID,Date,Time},OptionalUserID)`；date+time 均强制且按当前营业/餐段/售罄 fail closed；`Execute(ctx,WriteMeta,CatalogCommand)`；`ReorderProducts(ctx,WriteMeta,CategoryID,OrderedProductIDs)`；specification 只读，任何写 command/DTO 不含它；吸收现 `menu` projection |
| `StaffDiscount` | `View(ctx,UserID)`；`Execute(ctx,WriteMeta,StaffDiscountCommand)`；`ResolveInTx(ctx,tx,UserID)`；`staffidentity.Resolve` 内部化 |
| `Quote` | `Create(ctx,WriteMeta,CreateInput)`；`Read(ctx,UserID,QuoteID)`；`FinalizeForPrepayInTx(ctx,tx,UserID,QuoteID,at)`；`LoadSnapshotInTx(ctx,tx,QuoteID)` |
| `PaymentOrder` | `Prepare(ctx,WriteMeta,QuoteID)`；`Confirm(ctx,WriteMeta,PrepaymentID)`；`IngestPayment(ctx,VerifiedPayment)`；`RunDue(ctx,Now,Limit)`；`ListPending(ctx,OwnerUserID,Filter,PageQuery)`；`MaterializePending(ctx,WriteMeta,PrepaymentID)`；ingress/worker 用 intrinsic dedupe |
| `Order` | `MaterializePaidInTx(ctx,tx,PaidMaterialization)`；`GetUser`；`ListUser`；`SearchMerchant`；`RunProductionDue(ctx,Now,Limit)`；查询不推进状态 |
| `Fulfillment` | `Execute(ctx,WriteMeta,FulfillmentCommand)`；command 仅 `MARK_READY / REDEEM_TOKEN / REDEEM_CURRENT_DATE_CODE / REDEEM_ORDER` |
| `Refund` | `RequestOrder(ctx,WriteMeta,OrderID,Reason)`；`RequestPaidPrepayment(ctx,WriteMeta,PrepaymentID,Reason)`；`IngestRefund(ctx,VerifiedRefund)`；`RunDue(ctx,Now,Limit)`；`ListPending` |
| `Subscription` | `RecordConsent(ctx,WriteMeta,ConsentInput)`；`EnqueueInTx(ctx,tx,NotificationIntent)`；`RunDue(ctx,Now,Limit)` |
| `Import` | `Preview(ctx,WriteMeta,ImportKind,XLSX)`；`Commit(ctx,WriteMeta,PreviewToken,SkipInvalid)` |
| `Audit` | `AppendReceiptInTx(ctx,tx,WriteMeta,CommandResult)`，必须 insert-last；`ReplayReceipt(ctx,ActorScope,Action,OperationKey)`；`AppendEvidenceInTx(ctx,tx,AuditEntry)`；`Search(ctx,OwnerUserID,AuditFilter,PageQuery)` |
| `Billing` | `Summary(ctx,OwnerUserID,BillingRange)`；`ListPayments(ctx,OwnerUserID,BillingQuery,PageQuery)`；`ListRefunds(...)`；`ExportCSV(...)`；`RunReconcile(ctx,BillDate,Limit)`；前三项只读派生，reconcile 以 bill date+provider bill digest intrinsic dedupe，把单边账投影到既有 prepayment/refund pending 事实并写 audit，不建账单/汇总表 |

仅三个真实外部 seam，各自必须有 production adapter 与 deterministic fake adapter：

1. `MiniProgramAdapter`：ExchangeLoginCode、ExchangePhoneCode、SendSubscription。
2. `WeChatPayAdapter`：CreateJSAPI、QueryTransaction、CloseTransaction、ParsePaymentNotification、CreateRefund、QueryRefund、ParseRefundNotification、DownloadTransactionBill；下载账单返回受限 stream+provider digest，由 Billing 当次派生核对。deterministic fake 的 `DownloadTransactionBill` 按 bill date 返回稳定排序/稳定 digest，并可显式注入微信侧单边账或系统侧单边账，禁止用随机行掩盖 reconcile 错误。
3. `ObjectStoreAdapter`：PutImage、PublicURL；MySQL 只保存 object key。

Module-owned 稳定错误：通用 `ErrInvalidInput / ErrUnauthenticated / ErrForbidden / ErrNotFound / ErrIdempotencyConflict / ErrUnavailable`；Quote 另有 `ErrExpired / ErrQuoteStale / ErrItemUnavailable / ErrPickupCutoffPassed / ErrPaymentAmountTooSmall / ErrSnapshotInvalid`；MerchantAuth 有 `ErrLastOwner / ErrAccountNotAvailable / ErrPCLoginExpired / ErrPCSessionExpired`；Fulfillment 有 `ErrTransitionNotAllowed / ErrRedemptionInvalid`；Import 有 `ErrInvalidFile / ErrInvalidTemplate / ErrFileTooLarge / ErrTooManyRows / ErrPreviewExpired`；Billing 有 `ErrBillUnavailable / ErrBillMismatch`。错误不得携带 PII。Payment Apply 遇 `ErrUnavailable` 保留 durable Observation 并重试；遇 `ErrSnapshotInvalid` 把 Observation/Prepayment 置 `DEFERRED/PENDING_MANUAL`，绝不伪造新 Pending 记录或丢失外部事实。

## 8. HTTP DTO 冻结

### 8.1 统一编码

- Prefix 固定 `/api/v1`；PRD §15.6 中省略 prefix 的路径均加此前缀。
- JSON ID 一律十进制 string；金额整数分；UTC 时间 RFC3339Nano；取餐日期 `YYYY-MM-DD`；取餐时刻 `HH:mm`。
- 写请求严格 JSON：拒绝未知字段、重复 key、无效 UTF-8、尾随内容、超限 body；multipart 仅 upload/import。
- 所有已认证业务 mutator 必须且只能有一个 `Idempotency-Key` 并映射 `WriteMeta`；login/phone/QR 认证交换与 provider callback 禁止客户端键，使用 §7 intrinsic dedupe；Authorization 存在但无效时不得降级匿名。
- 错误统一 `{"error":{"code":"STABLE_CODE","message":"redacted message"}}`；任何响应不含 openid、完整手机号、内部 source version/digest、raw provider payload、密钥。
- `200` 幂等重放，`201` 新建，`202 {"state":"PENDING"}` 支付/退款仍未确定，`204` verified callback durable 后；400/401/403/404/409/422/503 按稳定错误映射。

### 8.2 核心 DTO

```text
Money fields: *_cents: integer >= 0
Image: {object_key:string,url:string}       // url 是 Adapter 生成的读 URL，非 DB 事实
Product: {id,category_id,name,description,specification,meal_period,images[],listed,sold_out,
          original_unit_price_cents,staff_unit_price_cents?}
PickupOption: {date,meal_period,time,cutoff_time,available,reason?}
UserIdentity: {primary_phone:{bound,masked_phone?},extra_phone:{set,masked_phone?,name?},
               pricing_identity:{kind,rate_percent},merchant:{bound,role?}}
Quote: {id,contact:{name,masked_phone},identity:{kind},discount:{rate_percent},
        store:{name,address},pickup:{date,time,meal_period,point},order_note,items[],
        original_subtotal_cents,discount_cents,payable_cents,created_at,expires_at}
Prepayment: {id,state,expires_at,wx_request_payment?}
OrderSummary: {id,order_no,state,pickup_date,pickup_time,pickup_point,pickup_number,
               payable_cents,materialized_at,available_actions[]}
OrderDetail: OrderSummary + {contact,identity,discount,items,transaction_id?,paid_at,
                            redemption_token?,transition_times,notification_options}
RefundView: {id,order_id?,state,amount_cents,requested_at,provider_refund_id?}
```

`wx_request_payment` 只含小程序 `requestPayment` 所需字段 `{timeStamp,nonceStr,package,signType,paySign}`，响应和日志均 `Cache-Control: no-store`；durable JSON 不经 audit/log 复制。

### 8.3 用户与公开路由

| 方法/路径 | 请求 DTO | 成功 DTO / 语义 |
| --- | --- | --- |
| POST `/auth/miniprogram/session` | `{code}` | `{session:{token,expires_at}}` |
| GET `/storefront/settings` | — | `{storefront:{name,address,pickup_point,announcement,business_status,launch_layer?,flavors}}` |
| GET `/catalog`；GET `/catalog/products/:id?date=YYYY-MM-DD&time=HH:mm` | optional Bearer；detail 的 date/time 必填 | `{categories:[...]}` / `{product}`；匿名只原价；detail 缺 date/time 为 400，营业/餐段/售罄当前事实不可读为 503；`specification` 只读返回 |
| GET `/menu/pickup-options` | — | `{dates:[{date,available,meal_periods:[...]}]}` |
| GET `/menu?date=&time=` | optional Bearer | `{selection,store_status,categories}`；未知当前事实 503/不可下单 |
| POST `/me/bind-phone` | `{code}` | `{primary_phone:{bound:true,masked_phone}}` |
| GET `/me/primary-phone` | — | `{primary_phone:{bound,masked_phone?}}` |
| POST `/me/extra-phone` | `{phone,name}` | `{extra_phone:{set:true,masked_phone,name},pricing_identity}` |
| GET `/me/identity` | — | `{identity:UserIdentity}` |
| POST `/me/merchant-login` | `{code}` | `{merchant:{bound:true,role}}` |
| POST `/quotes` | `{contact_name,pickup_date,pickup_time,order_note,items:[{product_id,quantity,flavors,note}]}` | `201/200 {quote}`；客户端不传价格/身份 |
| GET `/quotes/:id` | — | `{quote}`；非 owner 与不存在统一 404 |
| POST `/orders/prepay` | `{quote_id}` | `201/200 {prepayment}`；剩余 <1m 或 stale 为 409，不调 provider |
| POST `/orders/confirm` | `{prepayment_id}` | `200 {state:"ORDER_CREATED",order_id}` 或 `202 {state:"PENDING"}`；不接受客户端 success |
| GET `/orders`；GET `/orders/:id` | filters/page | `{orders:[...],next_after_id?}` / `{order}` |
| POST `/orders/:id/cancel` | `{reason}` | `{order,refund}`；仅 RESERVED 且 >30m |
| POST `/orders/:id/subscriptions` | `{kind:"READY"|"REFUND_RESULT",decision:"ACCEPTED"|"REJECTED"}` | `{subscription:{kind,decision,available}}` |

### 8.4 商户小程序路由

| 方法/路径 | 请求 DTO | 成功 DTO |
| --- | --- | --- |
| GET `/merchant/orders` | `state,date,q,after_id,limit` | `{orders,next_after_id?}`；phone 只 masked |
| GET `/merchant/orders/:id` | — | `{order}` |
| PUT `/merchant/store-status` | `{status:"open"|"closed"|"cutoff"}` | `{store_status}` |
| PUT `/merchant/products/:id/soldout` | `{service_date,sold_out}` | `{product_id,service_date,sold_out}` |
| POST `/merchant/orders/:id/ready` | `{}` | `{order}`；仅 PREPARING→READY_FOR_PICKUP |
| POST `/verify/scan` | `{token}` | `{order}`；token 不写日志 |
| POST `/verify/code` | `{pickup_number:"0001"}` | `{order}`；仅当前营业日期 |
| POST `/merchant/orders/:id/redeem` | `{}` | `{order}`；用于跨日未取餐逐单核销 |

### 8.5 PC Admin 路由

| 路由组 | 冻结请求/响应 |
| --- | --- |
| POST `/admin/auth/qrcode` | request `{}`；response `{login_id,poll_secret,qr_payload,expires_at}`；challenge 固定 2 分钟，`qr_payload` 内含不同的 approval secret，poll secret 只给浏览器一次 |
| POST `/me/admin-login/approve` | `{login_id,approval_secret,code}` → `{approved:true}`；微信扫码端调用 |
| POST `/admin/auth/poll` | `{login_id,poll_secret}` → `202 {state:"WAITING"}` 或 `{state:"APPROVED",session:{token,expires_at}}`；token 只返回一次，session 自签发起固定 12 小时绝对过期；无 refresh、无记住设备、并发 session 不互踢 |
| `/admin/categories[/:id]`、PUT `/admin/categories/order` | Category CRUD；order `{ids:[...]}`，删除有商品 409 |
| `/admin/products[/:id]`、PUT `/admin/products/order`、PUT `/:id/status`、PUT `/:id/soldout` | Product CRUD；字段仅 images/name/price/category/meal/description，禁止 specification；order `{category_id,ids:[...]}` 为该分类完整顺序且用上移/下移 UI；soldout 带 service_date |
| POST `/upload` | multipart image → `{image:{object_key,url}}`；只允许约束内图片/透明 PNG |
| GET/PUT `/admin/settings`、`/admin/meal-periods`、`/admin/launch-layer` | 读写 Storefront/两餐段/日期营业/开屏；不接受第三餐段或第二取餐点 |
| GET/PUT `/admin/discount-rate` | `{rate_percent:1..100}` |
| `/admin/staff-whitelist[/:id]` | phone/name CRUD + enabled；响应 phone masked + derived bound/spend/orders |
| `/admin/merchant-accounts[/:id]` | phone/name/role CRUD + enabled；最后 owner 保护；响应 phone masked |
| POST `/admin/products/import/preview`；POST `/admin/staff-whitelist/import/preview` | multipart `.xlsx`，body 最大 10 MiB；PRODUCT 最多 500 数据行，STAFF 最多 5000 数据行 → `{preview_token,new_count,update_count,error_count,new_categories?,rows:[{row,outcome,reason?}]}`；超限整批拒绝 |
| POST `/import/commit` | `{preview_token,skip_invalid}` → `{batch_id,new_count,update_count,skipped_count}`；同 token 幂等 |
| GET `/admin/orders` | 六态/date/number/order/phone/unclaimed filters → page |
| GET `/admin/stats` | derived `{today_revenue_cents,today_orders,month_revenue_cents,month_orders,refund_cents,pending_production,product_sales[]}` |
| GET `/admin/finance/payments`、`/refunds`、`/summary`、`/export` | derived read/CSV；不写事实 |
| GET `/admin/pending-payments` | materialization_state PENDING_MANUAL page |
| POST `/admin/pending-payments/:id` | `{action:"MATERIALIZE"|"REFUND",reason}`；owner-only；manual materialize 仍验证 trusted payment/snapshot，不复核已允许 override 的 live drift |
| POST `/admin/orders/:id/refund` | `{reason}` → `{order,refund}` |
| GET `/admin/audits` | target/action/time page；只返回脱敏证据 |

### 8.6 Provider ingress

| 方法/路径 | 规则 |
| --- | --- |
| POST `/payments/wechat/notify` | WeChatPay Adapter 验签/解密并归一化；PaymentOrder durable Observation commit 后才 204；DB unavailable 返回非 2xx 促重试 |
| POST `/refunds/wechat/notify` | 同上，写 Refund Observation；callback 内不调用退款/通知 provider，不直接推进订单 |

## 9. 事务与锁顺序

任何事务先无锁定位 ID，再按下列全局 rank 获取 `FOR UPDATE`；锁后重新验证定位条件。相同表一律 PK 升序：`merchant_accounts → miniprogram_users → import_batches → discount_settings → staff_whitelist → storefront_settings → service_dates → meal_periods(lunch→dinner) → categories → products → product_sold_out_dates → quotes → quote_items → prepayments → payment_observations → pickup_sequences → orders → refunds → refund_observations → notification_consents → notification_outbox → action_audits(last)`。不存在反向例外。

1. Identity/account/config：actor account（如有）→ user/target aggregate → children PK 升序 → audit。最后 owner 检查锁全部 enabled owners 按 id。
2. Quote Create/Finalize：先无锁读取 quote locator（Create 无此步），再 user → discount → staff → storefront/date/meal → category/product/soldout → quote/items，锁后复核 quote locator/digest；折扣单独漂移允许，其余漂移拒绝。
3. Payment Prepare/Apply：quote/items → prepayment → observation → pickup sequence → 新 order/items → audit/outbox。callback 仅 prepayment → insert observation；204 前 commit。Create/Query 先 claim+commit，外调后新事务按同一 rank CAS。
4. Production/Fulfillment：order → audit/outbox；token/hash 或 pickup date+number 只用于无锁定位，锁后复核。每次只锁一单。
5. Refund request/apply：先无锁由 order/refund 找 prepayment id；随后 prepayment → order（若有）→ refund → refund observation → audit/outbox。callback 仅 refund → insert observation。退款 provider 始终在事务外。
6. Consent/notification：order → consent → outbox；sender 仅 outbox claim+commit，外调后按 outbox CAS。
7. Import：actor account → import batch → category/product 或 staff（各自 PK 升序）→ audit；同一 batch 只提交一次。

死锁 1205/1213 只允许整个新事务重试一次并重新授权/读取 live facts。不得通过先锁 audit 做幂等，也不得在持锁 transaction 内 sleep、HTTP 或 provider call。

## 10. 容量与索引结论

4000–5000 用户、日 500–1000 单使用单 MySQL 足够：业务 identity、日期+取餐号、token hash、out_trade/transaction/refund IDs 使用 PK/UNIQUE/FK 定点索引；本轮仅新增 §5.8 的 R1–R3 三个范围索引。看板按日/月条件以 PK cursor 小批扫描订单和明细，不需预聚合。Worker 每批最多 100 行，`FOR UPDATE SKIP LOCKED` 只用于 R1/R3 短 lease claim；业务 aggregate 仍用确定锁序。归档/分区/读副本不在一期。

## 11. 冻结后实现 DAG 与 ownership

```text
ORDER-MVP-R2 dual review = 0
        |
        +-- PC Admin: Catalog/Storefront/StaffDiscount/MerchantAuth/Import CRUD + pages
        +-- Transaction: Quote -> PaymentOrder -> Observation -> Order -> Refund
        +-- User Mini: browse -> identity -> quote -> prepay/confirm -> order/cancel/subscription
        +-- Merchant Mini: orders -> ready -> redeem -> soldout/store status
        |
        +-- CP single writer: v18-v44 ledger + shared httpdto + router/main/config/workers
        |
        +-- fresh MySQL fake-provider E2E + UI1 + exact-SHA detached verification
```

共享 `router/main/config/worker`、migration ledger、`internal/httpdto` 只有 CP writer；四 lane 的 feature package/page 路径不得重叠。真实 WeChat/腾讯/资金/UI3 单列 external TODO，不阻止 fake-provider local E2E。

## 12. 两轴 Review 清单

### Product / Spec

- PRD §4–§14 每个场景是否在矩阵中有持久化或明确 derived 来源。
- 身份隔离、当前事实与历史快照、六态、支付成功建单、对账兜底、退款、幂等、PII、审计是否闭合。
- PC、商户小程序、用户小程序是否共享同一 owner；是否出现 mock/第二套数据。
- External-only 项是否与缺代码分离。

### Standards / MySQL

- 每个 FK/UNIQUE/CHECK 是否支持 fail-closed；nullable group 是否闭合；JSON 是否只用于有界整对象。
- 索引是否对应真实查询/lease；是否有重复/低选择性/写放大索引。
- 所有锁序、provider transaction 边界、CAS/lease、回调 durable-first 是否无反向路径。
- v18–v44 是否单 writer、顺序稳定、每个 DDL 可独立恢复；v13 数据是否无损迁移。
- 4000–5000 用户是否无需 Redis/MQ/ES/汇总表；是否仍有缺表或 PRD 外表。

Review 规则：任一 finding 修改本稿即使旧 Review 失效；最终 `FROZEN` SHA 不再修改，由两名只读 reviewer 分别返回 0 finding 后解锁实现。

## 13. 外部资产与本地可完成边界

本地 fake-provider 必须完成：openid/phone exchange seam、JSAPI Create/Query/callback、退款 Create/Query/callback、订阅发送、对象存储 object key/URL、PC QR approve/poll、deterministic transaction bill download（稳定 digest/排序）及支付/退款单边账 reconcile、fresh MySQL 全链路与三端 UI1。

只列 `TODO/BLOCKED_EXTERNAL`：真实 appid/mchid、AppSecret、APIv3 key、商户私钥/平台证书、真实支付/退款资金、微信订阅模板审批、真实手机号/商户/员工名单、COS bucket/CAM/域名、DevTools 登录与 UI3、提审/发布。秘密、证书内容、DSN、真实 PII 不入业务表、客户端、日志、Gate 台账或本文档。
