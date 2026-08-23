# Order MVP 统一领域事实、MySQL Schema 与接口冻结稿

## 0. 控制面

| 项 | 值 |
| --- | --- |
| 产品唯一基线 | `docs/product/online-ordering-system-prd-0818.md` |
| 设计固定点 | authoritative integration `01cc4896c3c6c8531b1a2f08778deeccab3dd43b` |
| Schema / Interface 版本 | `ORDER-MVP-R1` |
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
6. Quote 保存服务端主手机号、联系人姓名、身份/白名单/折扣、商品/价格、门店/取餐事实及 digest；HTTP 只返回脱敏手机号，不返回完整手机号或内部 digest/version。
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
| 备好/核销 | order 状态、token hash、发放/消费操作者与时间 | QR 内容 | token 不明文落库；跨日码不误核销 |
| 退款 | 全额 refund、不可变请求、lease/version | 退款待处理列表 | 真实资金/UAT 外部；未知时停 `REFUNDING` |
| 退款回调/查询 | Refund Observation、可信结果时间、应用状态 | 回放响应 | 成功 Observation 才 `REFUNDED` |
| 订阅 | consent 决策；outbox intent/lease/结果 | 补订阅入口 | template 审批外部；失败不动订单 |
| XLSX 导入 | 单 batch 的 digest、预览/错误 JSON、计数、提交结果 | 逐行任务 | 服务端解析；未确认零业务写；无 row/task 表 |
| 审计 | 统一 action audit：actor/action/target/result/reason/idempotency/before/after | 状态历史展示 | 不存 raw PII/provider payload；不用于重建业务状态 |
| 财务/看板/未取餐 | 无新事实 | orders/items/prepayments/refunds 实时查询/导出 | 只读查询不得修改事实 |

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
| v2 | integrated | `categories` | v19 补 name key、version、约束 |
| v3–v4 | integrated | `products` + meal enum | v20 补有序图片 object keys、name key、version、price > 0 |
| v5–v6 | integrated | `meal_periods` + 两行 seed | 保留独立配置事实，不迁入 singleton |
| v7 | integrated | `product_sold_out_dates` | 保留复合 PK；不建库存 |
| v8–v10 | integrated | `miniprogram_users`、`miniprogram_sessions`、主手机号 | 保留 opaque revocable session；v18 补单组附加手机号 |
| v11 | integrated | `storefront_settings` | v21 把 launch URL 语义改成 object key，补全局口味 JSON/version |
| v12 | integrated | `merchant_accounts` | v22 补不可恢复 soft-delete 事实，保护历史 FK |
| v13 | integrated | `merchant_action_audits` | v35 原地 rename/扩展为唯一 `action_audits`，不建第二张审计表 |
| v14 | TX-02 WIP | `staff_whitelist` | 集成前直接补 `name_key`；phone 唯一，姓名仅用于双要素 |
| v15 | TX-02 WIP | `discount_settings` | 保留 singleton，rate `1..100`，未初始化 Quote fail closed |
| v16 | TX-02 WIP | `quotes` | 集成前补持久化 `expires_at`；保留完整 header 快照/digest |
| v17 | TX-02 WIP | `quote_items` | 保留行级不可变快照；无 product FK，历史不受商品删除影响 |

v9、v5、v15、v17 虽可在空白设计中折叠，但已经分别承担 token 撤销、餐段独立配置、折扣锁/版本、行级 digest 与金额完整性。为减少表数而新增迁移搬运/删表会增加停机、回滚和双模型风险，不满足“最短正确路径”。

最终物理模型为 25 张业务表（不含 `schema_migrations`）：

| 分类 | 表 | 来源 |
| --- | --- | --- |
| 当前身份/权限 | `miniprogram_users, miniprogram_sessions, merchant_accounts, staff_whitelist, discount_settings, merchant_pc_sessions` | v8–v10/v12、TX-02 v14–v15、new v24 |
| 当前目录/营业 | `categories, products, meal_periods, storefront_settings, service_dates, product_sold_out_dates` | v2–v7/v11、new v23 |
| 交易/履约 | `quotes, quote_items, prepayments, payment_observations, pickup_sequences, orders, order_items` | TX-02 v16–v17、new v25–v29 |
| 退款/通知 | `refunds, refund_observations, notification_consents, notification_outbox` | new v30–v33 |
| 批量/证据 | `import_batches, action_audits` | new v34、v13 原地升级 v35 |

### 4.2 v18–v35 唯一 ledger

旧 TX-03 对 v18–v23 的预留作废；Review 冻结后由 migration-ledger 单 writer 按下列顺序创建，每个文件只做一个原子 `ALTER` 或一张 `CREATE TABLE`：

| 版本 | DDL 责任 |
| --- | --- |
| v18 | ALTER `miniprogram_users`：附加手机号/姓名/归一 key/设置时间/版本 |
| v19 | ALTER `categories`：`name_key`、record version、trim/unique |
| v20 | ALTER `products`：`name_key`、record version、`image_object_keys_json`、price/name checks |
| v21 | ALTER `storefront_settings`：launch object key、全局 flavors JSON、record version |
| v22 | ALTER `merchant_accounts`：`deleted_at/deleted_by`，不可恢复删除语义 |
| v23 | CREATE `service_dates` |
| v24 | CREATE `merchant_pc_sessions` |
| v25 | CREATE `prepayments` |
| v26 | CREATE `payment_observations` |
| v27 | CREATE `pickup_sequences` |
| v28 | CREATE `orders` |
| v29 | CREATE `order_items` |
| v30 | CREATE `refunds` |
| v31 | CREATE `refund_observations` |
| v32 | CREATE `notification_consents` |
| v33 | CREATE `notification_outbox` |
| v34 | CREATE `import_batches` |
| v35 | ALTER/RENAME v13 为 `action_audits`，保留已有审计行并扩展 actor/target/idempotency |

迁移 runner 继续单连接 `GET_LOCK`；任一 dirty migration 阻止 API 启动。不得并行分配迁移号。

## 5. 冻结 MySQL 数据模型

除特殊说明外：InnoDB、MySQL 8、`utf8mb4_0900_ai_ci`；业务 ID 为 `BIGINT UNSIGNED AUTO_INCREMENT`；时间为 UTC `TIMESTAMP(6)`，营业日期/时刻另用 `DATE` / `TIME`；金额为 `BIGINT UNSIGNED` 整数分；手机号为 E.164 `VARBINARY(16)`；hash 为 `BINARY(32)`。应用在入库前完成 UTF-8、长度、NFKC、空白与 JSON element 校验，DB 再用 CHECK/UNIQUE/FK fail closed。

### 5.1 当前配置与身份

| 表 | 必须字段 | 关键约束与索引 | PII / 生命周期 |
| --- | --- | --- | --- |
| `miniprogram_users` | existing id/openid/login；primary phone/bound at；`extra_phone, extra_name, extra_name_key, extra_phone_set_at, record_version` | openid、primary phone 唯一；extra 四字段全空或全非空；两 phone E.164；version > 0 | openid、phone、name 为 PII；不得进 DTO/日志，phone 仅 masked |
| `miniprogram_sessions` | token_hash, user_id, issued_at, expires_at | PK token hash；FK user RESTRICT；`expires>issued`；index `(user_id,expires_at)` | bearer 只返回一次；DB 仅 hash；过期清理 |
| `merchant_accounts` | existing phone/name/role/enabled/binding/version；`deleted_at, deleted_by_account_id` | phone 唯一；binding 成组；deleted 后 enabled=false 且 binding 清空；最后一个 enabled OWNER 由锁事务守护 | soft-delete 对业务不可恢复，保留历史 actor FK |
| `staff_whitelist` | phone, name, `name_key`, enabled, record_version, created/updated | phone 唯一；E.164；name/key 非空；index `(enabled,id)` | phone/name PII；导入覆盖 name 但保留 created/enabled |
| `discount_settings` | singleton, rate_percent, discount_version, whitelist_version, updated_at | id=1；rate 1..100；versions >0 | 无 PII；任一名单写递增 whitelist_version |
| `merchant_pc_sessions` | id, `approval_secret_hash, poll_secret_hash`, state, login_expires_at, approved account/user/auth version/time, consumed_at, access_token_hash/expiry, created/updated | 两个 secret hash 均 unique；access hash unique nullable；状态字段成组 CHECK；index `(state,login_expires_at)` | QR 只含 approval secret，浏览器单独持有 poll secret；poll 事务才生成 access token并只返回一次；每次请求 live-check account |

姓名 `name_key` 的唯一算法为：有效 UTF-8 → Unicode NFKC → 删除全部 Unicode whitespace → 非空 → UTF-8 bytes；附加姓名与白名单必须调用同一实现后 byte-exact 比较。分类/商品 `name_key` 仅做 NFKC + 去首尾 Unicode whitespace，保持中间空格，再 byte-exact 唯一。

### 5.2 目录、营业与售罄

| 表 | 必须字段 | 关键约束与索引 |
| --- | --- | --- |
| `categories` | existing id/name/sort/is_active；`name_key, record_version` | UNIQUE name_key；trim/nonempty；version>0；existing visibility index |
| `products` | existing id/category/name/description/spec/price/sort/listed/meal；`name_key, record_version, image_object_keys_json` | FK category RESTRICT；UNIQUE name_key；price>0；JSON array length 0..3；catalog/menu indexes；不建 image 表 |
| `meal_periods` | code lunch/dinner, cutoff, pickup start/end, interval | PK code；仅两 code；start<=end；interval>0 且离散点可生成；所有更新锁两行固定 lunch→dinner |
| `storefront_settings` | singleton；store/address/pickup/announcement/status；`launch_image_object_key` + position/size；`flavor_options_json, record_version` | id=1；status open/closed/cutoff；launch key 与几何字段成组；flavors 为去重字符串 array；version>0 |
| `service_dates` | service_date, is_open, record_version, updated_by_account_id, updated_at | PK date；boolean/version CHECK；FK merchant account RESTRICT；缺行即 closed；index `(is_open,service_date)` |
| `product_sold_out_dates` | service_date, product_id | PK `(service_date,product_id)`；FK product RESTRICT；读取失败按 sold out；不保存数量；操作者/时间进入统一 audit |

`product_sold_out_dates` 物理 v7 保持不变；不为页面展示增加重复列。

### 5.3 Quote

`quotes` 保持 v16 header 列，并新增 `expires_at TIMESTAMP(6) NOT NULL`；`UNIQUE(user_id,idempotency_key_hash)`；FK user RESTRICT；contact name/phone、identity kind/source version、discount rate/version、门店/地址/取餐点、pickup date/time/meal、order note、item count、三项金额、snapshot digest、created/expires 均不可变。CHECK：联系信息规范；rate 1..100；item count >0；`original=discount+payable`；payable>0；`expires_at>created_at`。

`quote_items` 保持 v17：PK `(quote_id,line_number)`；FK quote RESTRICT；product id 仅作历史引用、不设 FK；name/source digest、两种 unit price、quantity、两种 subtotal、flavors JSON、line note 均不可变。CHECK quantity>0、line>0、折后价<=原价、乘法不溢出且 subtotal 精确、flavors 为 array。每个 flavor 必须是当前 singleton 全局口味选项中的唯一成员；口味配置漂移属于 current-fact drift。Header/item 汇总与 digest 由同一 transaction 生成和复核。

### 5.4 Prepayment 与 Payment Observation

`prepayments`：

- identity：`id, user_id FK, quote_id FK UNIQUE, idempotency_key_hash, out_trade_no UNIQUE`；`UNIQUE(user_id,idempotency_key_hash)`。
- immutable expected facts：`expected_appid, expected_mchid, expected_amount_cents, currency='CNY', provider_create_request_json, provider_create_request_digest, effective_deadline`。
- provider result：`provider_state ENUM('READY','CREATE_UNKNOWN','PAYMENT_REQUESTED','NOT_PAID','PAID','CLOSED')`，`create_attempted_at`，`wx_request_payment_json`，`provider_prepay_id`，`last_queried_at`。
- materialization：`materialization_state ENUM('AWAITING_PAYMENT','READY','APPLIED','PENDING_MANUAL','REFUNDED_WITHOUT_ORDER')`，`pending_reason_code`。
- concurrency：`lease_kind ENUM('CREATE','QUERY') nullable, lease_owner BINARY(16), lease_expires_at, record_version, next_reconcile_at, created_at, updated_at`；lease 三字段全空或全非空；version>0；index `(next_reconcile_at,provider_state,id)`、`(materialization_state,updated_at,id)`。

规则：第一次持久化 Create request 后不可修改；`create_attempted_at` 一旦非空永不再次调用 Create。Create 超时写 `CREATE_UNKNOWN`，随后只能 Query。只有距离 effective deadline 至少 1 分钟才可首次 Create。Provider/Observation 状态只能按可信度与终态单调前进，乱序 callback/query 不得把 `PAID` 回退为 `NOT_PAID/CLOSED`。

`payment_observations`：

- `id, prepayment_id FK, dedupe_key UNIQUE, source CALLBACK|ACTIVE_QUERY, provider_event_id nullable, out_trade_no, transaction_id nullable, provider_state PAID|NOT_PAID|CLOSED`。
- `validation MATCH|MISMATCH`、稳定 `mismatch_code`、`amount_cents, currency, success_time nullable, received_at`。
- `materialization_mode AUTO|DELAYED_MANUAL`：仅 MATCH + PAID + trusted `success_time < effective_deadline` 为 AUTO；等于/晚于、字段不匹配或快照异常均为 DELAYED_MANUAL。
- `apply_state NEW|APPLIED|DEFERRED, apply_reason_code, applied_at, record_version`；UNIQUE non-null transaction id；index `(apply_state,id)`、`(prepayment_id,received_at,id)`。

不保存 raw callback、签名头、解密明文或证书。无效签名/解密失败不形成可信 Observation；已验证但字段 mismatch 必须 durable 并进入 shield。

### 5.5 Order、履约与取餐序列

`pickup_sequences(service_date PK, last_number, updated_at)`，CHECK `0<=last_number<=9999`。仅在订单事务中 `SELECT ... FOR UPDATE` 后加一；事务回滚即不占号。

`orders`：

- identity：`id, order_no UNIQUE, user_id FK, quote_id UNIQUE, prepayment_id UNIQUE, payment_observation_id UNIQUE, record_version`。
- immutable header：Quote 的 contact/identity/discount/store/address/pickup/date/time/meal/note/item_count/三项金额/snapshot digest 全量复制；另存 `pickup_at UTC, transaction_id UNIQUE, paid_at, materialized_at`。
- fulfillment：`pickup_number`，`state` 六态，`preparing_at, ready_at, completed_at, refunding_at, refunded_at`。
- redemption：`redemption_token_hash UNIQUE nullable, redemption_issued_at, redeemed_by_account_id nullable, redeemed_at`。
- constraints/indexes：UNIQUE `(pickup_date,pickup_number)`；pickup 1..9999；token 字段成组；FK user/prepayment/observation/account RESTRICT；`(user_id,materialized_at DESC,id DESC)`、`(state,pickup_at,id)`、`(contact_phone_snapshot,materialized_at,id)`。

`order_items`：PK `(order_id,line_number)`，FK order RESTRICT；复制 Quote item 并新增 `image_object_key_snapshot nullable`；不 FK 当前 product。index `(product_id,order_id)` 支撑销量实时聚合。

订单初态必须先 `InitialState(success_time,pickup_at)`，再 `Advance(initial,materialized_at,pickup_at)`，同一事务补赶到 `PREPARING`；不得直接凭 Apply 当前时钟跳过 policy。

### 5.6 Refund 与 Refund Observation

`refunds`：

- `id, prepayment_id UNIQUE FK, order_id UNIQUE nullable FK, out_refund_no UNIQUE, idempotency_key_hash, amount_cents, currency='CNY', reason_code, requested_by_user/account nullable, created_at`。
- 不可变 `provider_request_json/digest`；`provider_state READY|CREATE_UNKNOWN|PROCESSING|SUCCESS|CLOSED`；`create_attempted_at, provider_refund_id nullable`。
- `materialization_state AWAITING_PROVIDER|APPLIED|PENDING_MANUAL`、`pending_reason_code`；Create/Query lease 三字段、record version、next query、updated_at；索引 `(provider_state,next_query_at,id)`、`(materialization_state,updated_at,id)`。
- CHECK 恰有 user 或 merchant requester；order 存在时 amount 必须等于 order payable，支付待处理退款则等于 prepayment expected amount，由事务校验；一笔 prepayment 最多一笔全额退款。

`refund_observations`：`id, refund_id FK, dedupe_key UNIQUE, source, provider_event_id, out_refund_no, provider_refund_id, provider_state PROCESSING|SUCCESS|CLOSED, validation, mismatch_code, success_time, received_at, apply_state NEW|APPLIED|DEFERRED, apply_reason_code, applied_at, record_version`；索引 `(apply_state,id)` 与 `(refund_id,received_at,id)`。只有 MATCH + SUCCESS 可把有订单的交易从 `REFUNDING` 推进到 `REFUNDED`。

### 5.7 Subscription、Import 与统一 Audit

`notification_consents`：`id, order_id FK, user_id FK, kind READY|REFUND_RESULT, grant_sequence, decision ACCEPTED|REJECTED, template_config_version, idempotency_key_hash, decided_at, consumed_at nullable`；UNIQUE `(order_id,kind,grant_sequence)` 和 `(user_id,idempotency_key_hash)`；accepted 才可消费，一个 consent 最多关联一个 outbox。

`notification_outbox`：`id, order_id FK, consent_id UNIQUE FK, kind, recipient_user_id FK, immutable_message_json, template_config_version, state PENDING|IN_FLIGHT|SENT|FAILED_PERMANENT, attempt_count, next_attempt_at, lease_owner/expires, record_version, provider_message_id nullable, last_error_code nullable, created/sent_at`；UNIQUE `(order_id,kind)`；index `(state,next_attempt_at,id)`。发送调用在 transaction 外；本地唯一只保证单一 intent，不宣称微信 exactly-once。

`import_batches`：`id, kind PRODUCT|STAFF, actor_account_id FK, file_object_key nullable, file_digest, preview_token_hash UNIQUE, preview_json, new_count, update_count, error_count, state PREVIEWED|COMMITTED|EXPIRED, commit_idempotency_key_hash nullable, skip_invalid nullable, result_json nullable, expires_at, committed_at, record_version, created/updated`；UNIQUE `(actor_account_id,commit_idempotency_key_hash)`（nullable）；index `(state,expires_at,id)`。preview/commit 只用这一行；staff PII preview 在 commit/expiry 后清空，临时 object 由对象存储生命周期清理。

v35 将 `merchant_action_audits` 原地变为唯一 `action_audits`：

- `id, actor_kind USER|MERCHANT|SYSTEM|PROVIDER, actor_scope_hash, actor_user_id nullable, actor_account_id nullable, actor_role_snapshot nullable, actor_auth_version nullable`。
- `action, target_type, target_id nullable, target_key_hash nullable, operation_key_hash, request_id_hash nullable, result SUCCEEDED|REJECTED, reason_code, before_state_json, after_state_json, response_json, occurred_at`。
- UNIQUE `(actor_scope_hash,action,operation_key_hash)`；indexes `(target_type,target_id,occurred_at,id)`、`(actor_account_id,occurred_at,id)`。
- CHECK：USER 只需 user；MERCHANT 需 user+account；SYSTEM/PROVIDER 两者均空。JSON 只含最小状态和可幂等重放的非 PII 结果；不保存手机号、姓名、openid、raw provider payload。

Audit 可用于幂等结果重放和证据查询，但各 aggregate 当前表仍是唯一业务状态源；禁止用 audit 重建订单或配置。

## 6. Derived 查询与明确不建表

| Derived | 来源与必要索引 |
| --- | --- |
| 当前营业、最近可预约、时间点 | storefront + service_dates + meal_periods + now |
| 员工已绑定/累计消费/累计单量 | users + staff whitelist + paid orders/refunds |
| 进行中订单、待制作、未取餐 | orders 的 user/state/pickup indexes |
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
    ActorUserID   uint64
    IdempotencyKey string
    RequestID      string
}

type PageQuery struct { AfterID uint64; Limit uint16 } // Limit 1..100
```

| Module | 冻结入口 |
| --- | --- |
| `Identity` | `IssueSession(ctx, LoginCode) (Session,error)`；`Authenticate(ctx, Token) (UserID,error)`；`View(ctx, UserID) (UserIdentity,error)`；`BindPrimaryPhone(ctx, UserID, PhoneCode) (UserIdentity,error)`；`SetExtraPhone(ctx, WriteMeta, ExtraPhoneClaim) (UserIdentity,error)` |
| `MerchantAuth` | `Identity(ctx,UserID)`；`LoginMini(ctx,WriteMeta,PhoneCode)`；`BeginPCLogin(ctx)`；`ApprovePCLogin(ctx,WriteMeta,LoginID,ApprovalSecret,PhoneCode)`；`PollPCLogin(ctx,LoginID,PollSecret)`；`AuthenticatePC(ctx,Token)`；`ExecuteAccount(ctx,WriteMeta,AccountCommand)`；`AuthorizeInTx(ctx,tx,UserID,Action,Target)` |
| `Storefront` | `Public(ctx,at) (PublicStorefront,error)`；`PickupOptions(ctx,at)`；`ResolvePickupInTx(ctx,tx,Selection,observedAt)`；`Configure(ctx,WriteMeta,SettingsCommand)`；`SetBusinessStatus(ctx,WriteMeta,Status)` |
| `Catalog` | `Browse(ctx,MenuQuery,OptionalUserID)`；`Detail(ctx,ProductID,OptionalUserID)`；`Execute(ctx,WriteMeta,CatalogCommand)`；吸收现 `menu` 的 product projection |
| `StaffDiscount` | `View(ctx,UserID)`；`Execute(ctx,WriteMeta,StaffDiscountCommand)`；`ResolveInTx(ctx,tx,UserID)`；`staffidentity.Resolve` 内部化 |
| `Quote` | `Create(ctx,UserID,IdempotencyKey,CreateInput)`；`Read(ctx,UserID,QuoteID)`；`FinalizeForPrepayInTx(ctx,tx,UserID,QuoteID,at)`；`LoadSnapshotInTx(ctx,tx,QuoteID)` |
| `PaymentOrder` | `Prepare(ctx,WriteMeta,QuoteID)`；`Confirm(ctx,UserID,PrepaymentID)`；`IngestPayment(ctx,VerifiedPayment)`；`RunDue(ctx,Now,Limit)`；`ListPending(ctx,OwnerUserID,Filter,PageQuery)`；`MaterializePending(ctx,WriteMeta,PrepaymentID)` |
| `Order` | `MaterializePaidInTx(ctx,tx,PaidMaterialization)`；`GetUser`；`ListUser`；`SearchMerchant`；`RunProductionDue(ctx,Now,Limit)`；查询不推进状态 |
| `Fulfillment` | `Execute(ctx,WriteMeta,FulfillmentCommand)`；command 仅 `MARK_READY / REDEEM_TOKEN / REDEEM_CURRENT_DATE_CODE / REDEEM_ORDER` |
| `Refund` | `RequestOrder(ctx,WriteMeta,OrderID,Reason)`；`RequestPaidPrepayment(ctx,WriteMeta,PrepaymentID,Reason)`；`IngestRefund(ctx,VerifiedRefund)`；`RunDue(ctx,Now,Limit)`；`ListPending` |
| `Subscription` | `RecordConsent(ctx,WriteMeta,ConsentInput)`；`EnqueueInTx(ctx,tx,NotificationIntent)`；`RunDue(ctx,Now,Limit)` |
| `Import` | `Preview(ctx,WriteMeta,ImportKind,XLSX)`；`Commit(ctx,WriteMeta,PreviewToken,SkipInvalid)` |
| `Audit` | `AppendInTx(ctx,tx,AuditEntry)`；`Search(ctx,OwnerUserID,AuditFilter,PageQuery)` |

仅三个真实外部 seam，各自必须有 production adapter 与 deterministic fake adapter：

1. `MiniProgramAdapter`：ExchangeLoginCode、ExchangePhoneCode、SendSubscription。
2. `WeChatPayAdapter`：CreateJSAPI、QueryTransaction、CloseTransaction、ParsePaymentNotification、CreateRefund、QueryRefund、ParseRefundNotification。
3. `ObjectStoreAdapter`：PutImage、PublicURL；MySQL 只保存 object key。

Module-owned 稳定错误：通用 `ErrInvalidInput / ErrUnauthenticated / ErrForbidden / ErrNotFound / ErrIdempotencyConflict / ErrUnavailable`；Quote 另有 `ErrExpired / ErrQuoteStale / ErrItemUnavailable / ErrPickupCutoffPassed / ErrPaymentAmountTooSmall / ErrSnapshotInvalid`；MerchantAuth 有 `ErrLastOwner / ErrAccountNotAvailable / ErrPCLoginExpired`；Fulfillment 有 `ErrTransitionNotAllowed / ErrRedemptionInvalid`；Import 有 `ErrInvalidFile / ErrInvalidTemplate / ErrTooManyRows / ErrPreviewExpired`。错误不得携带 PII。Payment Apply 遇 `ErrUnavailable` 保留 durable Observation 并重试；遇 `ErrSnapshotInvalid` 把 Observation/Prepayment 置 `DEFERRED/PENDING_MANUAL`，绝不伪造新 Pending 记录或丢失外部事实。

## 8. HTTP DTO 冻结

### 8.1 统一编码

- Prefix 固定 `/api/v1`；PRD §15.6 中省略 prefix 的路径均加此前缀。
- JSON ID 一律十进制 string；金额整数分；UTC 时间 RFC3339Nano；取餐日期 `YYYY-MM-DD`；取餐时刻 `HH:mm`。
- 写请求严格 JSON：拒绝未知字段、重复 key、无效 UTF-8、尾随内容、超限 body；multipart 仅 upload/import。
- 所有有副作用的业务请求必须且只能有一个 `Idempotency-Key`；Authorization 存在但无效时不得降级匿名。
- 错误统一 `{"error":{"code":"STABLE_CODE","message":"redacted message"}}`；任何响应不含 openid、完整手机号、内部 source version/digest、raw provider payload、密钥。
- `200` 幂等重放，`201` 新建，`202 {"state":"PENDING"}` 支付/退款仍未确定，`204` verified callback durable 后；400/401/403/404/409/422/503 按稳定错误映射。

### 8.2 核心 DTO

```text
Money fields: *_cents: integer >= 0
Image: {object_key:string,url:string}       // url 是 Adapter 生成的读 URL，非 DB 事实
Product: {id,category_id,name,description,meal_period,images[],listed,sold_out,
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
| GET `/catalog`；GET `/catalog/products/:id` | optional Bearer | `{categories:[...]}` / `{product}`；匿名只原价 |
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
| POST `/admin/auth/qrcode` | request `{}`；response `{login_id,poll_secret,qr_payload,expires_at}`；`qr_payload` 内含不同的 approval secret，poll secret 只给浏览器一次 |
| POST `/me/admin-login/approve` | `{login_id,approval_secret,code}` → `{approved:true}`；微信扫码端调用 |
| POST `/admin/auth/poll` | `{login_id,poll_secret}` → `202 {state:"WAITING"}` 或 `{state:"APPROVED",session:{token,expires_at}}`；token 只返回一次 |
| `/admin/categories[/:id]`、PUT `/admin/categories/order` | Category CRUD；order `{ids:[...]}`，删除有商品 409 |
| `/admin/products[/:id]`、PUT `/:id/status`、PUT `/:id/soldout` | Product CRUD；字段仅 images/name/price/category/meal/description；soldout 带 service_date |
| POST `/upload` | multipart image → `{image:{object_key,url}}`；只允许约束内图片/透明 PNG |
| GET/PUT `/admin/settings`、`/admin/meal-periods`、`/admin/launch-layer` | 读写 Storefront/两餐段/日期营业/开屏；不接受第三餐段或第二取餐点 |
| GET/PUT `/admin/discount-rate` | `{rate_percent:1..100}` |
| `/admin/staff-whitelist[/:id]` | phone/name CRUD + enabled；响应 phone masked + derived bound/spend/orders |
| `/admin/merchant-accounts[/:id]` | phone/name/role CRUD + enabled；最后 owner 保护；响应 phone masked |
| POST `/admin/products/import/preview`；POST `/admin/staff-whitelist/import/preview` | multipart `.xlsx` → `{preview_token,new_count,update_count,error_count,new_categories?,rows:[{row,outcome,reason?}]}` |
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

1. Identity bind：`miniprogram_users X` → insert audit；merchant binding：`miniprogram_users X → merchant_accounts X → audit`。
2. Account/config command：live actor `merchant_accounts X` → target aggregate → children 按 PK 升序 → audit 最后。最后 owner 检查锁全部 enabled owners 按 id。
3. Quote Create：named idempotency lock → `miniprogram_users → discount_settings → staff_whitelist → storefront_settings → meal_periods lunch/dinner → categories/products ascending → soldout ascending → quote/items`。
4. Quote Finalize：`quotes X → quote_items`，再按 Create 的 current-fact 顺序读取；折扣 rate/version 单独漂移允许，身份/商品/营业/售罄漂移拒绝。
5. Payment Prepare：Finalize Quote → insert prepayment/claim lease → commit；Create 在外部；CAS finalize。Query 同样先 claim lease、commit、外调、CAS。
6. Payment callback：外部验签/解密 → `prepayment → insert observation` → commit → 204。Apply worker：`payment_observation X → prepayment X → quote.LoadSnapshotInTx → provider identity/out_trade/amount/CNY/transaction 复核 → pickup_sequence X → order/items → observation/prepayment markers → audit/outbox`。
7. Production worker：orders 用 `(state,pickup_at,id)` 分页，逐单 transaction `order X → audit`；不批量长锁。
8. Fulfillment：live account → `order X → token fields → audit/outbox`。扫码先用 token hash 唯一索引定位，再锁 order；手工码用 current service date + pickup number 唯一索引定位。
9. Refund request：live user/account → `order X → refund insert → audit`，commit 后外调；refund callback durable；Apply 为 `refund_observation X → refund X → order X nullable → audit/outbox`。
10. Consent：`order X → consent → optional outbox`；Sender claim outbox lease并 commit，外调后 CAS。
11. Import commit：live owner → `import_batch X → categories/products/staff rows ascending → versions → audit`；同一批只提交一次。

死锁 1205/1213 只允许整个新事务重试一次并重新授权/读取 live facts。不得在持锁 transaction 内 sleep、HTTP 或 provider call。

## 10. 容量与索引结论

4000–5000 用户、日 500–1000 单使用单 MySQL 足够：用户/订单 owner 查询、state+pickup scheduler、日期+取餐号、token hash、out_trade/transaction/refund IDs 均为定点或短范围索引；看板按日/月范围聚合订单和明细，不需预聚合。Worker 每批最多 100 行，`FOR UPDATE SKIP LOCKED` 只用于 outbox/due-work 短 lease claim；业务 aggregate 仍用确定锁序。归档/分区/读副本不在一期。

## 11. 冻结后实现 DAG 与 ownership

```text
ORDER-MVP-R1 dual review = 0
        |
        +-- PC Admin: Catalog/Storefront/StaffDiscount/MerchantAuth/Import CRUD + pages
        +-- Transaction: Quote -> PaymentOrder -> Observation -> Order -> Refund
        +-- User Mini: browse -> identity -> quote -> prepay/confirm -> order/cancel/subscription
        +-- Merchant Mini: orders -> ready -> redeem -> soldout/store status
        |
        +-- CP single writer: v18-v35 ledger + shared httpdto + router/main/config/workers
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
- v18–v35 是否单 writer、顺序稳定、每个 DDL 可独立恢复；v13 数据是否无损迁移。
- 4000–5000 用户是否无需 Redis/MQ/ES/汇总表；是否仍有缺表或 PRD 外表。

Review 规则：任一 finding 修改本稿即使旧 Review 失效；最终 `FROZEN` SHA 不再修改，由两名只读 reviewer 分别返回 0 finding 后解锁实现。

## 13. 外部资产与本地可完成边界

本地 fake-provider 必须完成：openid/phone exchange seam、JSAPI Create/Query/callback、退款 Create/Query/callback、订阅发送、对象存储 object key/URL、PC QR approve/poll、fresh MySQL 全链路与三端 UI1。

只列 `TODO/BLOCKED_EXTERNAL`：真实 appid/mchid、AppSecret、APIv3 key、商户私钥/平台证书、真实支付/退款资金、微信订阅模板审批、真实手机号/商户/员工名单、COS bucket/CAM/域名、DevTools 登录与 UI3、提审/发布。秘密、证书内容、DSN、真实 PII 不入业务表、客户端、日志、Gate 台账或本文档。
