## Context

Exact base `73cf1e04d6661ad08cbd2b03c2bb8324d4769f4d` 已集成 `POST /api/v1/auth/miniprogram/session`、v8/v9、固定 `code2Session` client 与内部 `identity.Service.Authenticate`，但没有受保护的业务 route 或手机号字段。0818 PRD 要求浏览免手机号、首次提交订单前完成微信主手机号授权，并在个人中心提供主动绑定入口；本 change 只建立这两个调用时机共用的后端绑定能力，不进入员工识别或交易。

2026-08-20 核对的微信官方现行文档确认：[`getStableAccessToken`](https://developers.weixin.qq.com/miniprogram/dev/server/API/mp-access-token/api_getstableaccesstoken.html) 只支持固定官方 POST，普通模式为 `force_refresh=false`；[`getPhoneNumber`](https://developers.weixin.qq.com/miniprogram/dev/server/API/user-info/phone-number/api_getphonenumber.html) 使用 access token query、必填 code，并当前支持可选 openid 来校验 openid/code 绑定，成功返回 phone info 与 watermark AppID。客户端官方文档同时确认 code 五分钟有效且只能消费一次、能力限已认证非个人主体，并存在体验/付费额度。平台事实只冻结 wire 和 delivery Gate，不替代本地业务验收。

本 change 为 `gate_type=W3`、`ui_level_target=UI0`，DRAFT 时 `ui_level_actual=NOT_RUN`。隔离 MySQL 8 是实现阶段必跑本地 Gate；真实微信资产当前为 `BLOCKED_EXTERNAL/NOT_RUN`。

## Goals / Non-Goals

**Goals:**

- 复用现有 hash-only session Authenticate，对唯一 `/api/v1/me/bind-phone` 做 route-specific Bearer 鉴权。
- 以固定官方 stable token + getPhoneNumber 单次 wire 兑换、openid 绑定校验与 watermark AppID 校验取得主手机号。
- 标准化并唯一保存 E.164 主手机号，保证一次绑定、并发幂等、冲突稳定和无跨 provider 事务。
- 证明 token cache/并发 refresh、provider wire/error、真实 MySQL v10/锁/唯一性/恢复和全链路敏感信息边界。

**Non-Goals:**

- 附加手机号/P4、姓名、员工白名单/员工识别/折扣/P5、商户登录/RBAC、PC QR/P3、P1/P2、checkout/order/payment 或前端。
- 解绑、换号、profile GET、全局 middleware、logout/refresh、Redis/DB access-token cache、生产 SSM loader、可配置 provider origin、兼容 route 或自动 provider 重试。
- 修改 v8/v9、保存 access token/phone code/provider body，或把本地 stub 写成真实微信 PASS。

## Decisions

### Keep one exact HTTP contract and authenticate only this route

新增 `identity.PhoneHandler`，只注册 `POST /api/v1/me/bind-phone`。调用方是已建立 session 的小程序：首次结算拦截返回绑定页后，或个人中心用户主动点击 `getPhoneNumber` 后，立即把该点击产生的 code 发送到此 route；启动和匿名浏览不调用。

请求必须为最多 1 KiB 的 `application/json`，body 精确为一个非空、最长 256 bytes 的 `code`，且 header 精确为一个合法 `Authorization: Bearer <opaque-session-token>`。unknown/duplicate/trailing/oversize/blank 请求返回 HTTP 400；缺失、畸形或过期 session 返回 401；Authenticate 存储故障返回 503。handler 直接复用 `identity.Service.Authenticate`，不装全局 middleware，因此 health/catalog/menu/session creation 继续匿名。

成功、同一手机号幂等、请求到达时已经绑定，以及同 code 并发拒绝后已观察到当前用户完成绑定，均返回 exact HTTP 200：

```json
{"primary_phone_bound":true,"masked_phone":"+*********1234"}
```

`masked_phone` 保留 `+`，把规范化号码除最后最多四位之外的每一位替换为 `*`，并始终隐藏至少一位。已经绑定的合法重试在 provider 前短路，因此不会消费新 code，也不会借此实现换号；只有两个请求都在未绑定状态进入 provider 后，后到事务取得异号时才返回 already-bound 冲突。

稳定错误 envelope 只允许：HTTP 400 `INVALID_REQUEST`、HTTP 401 `UNAUTHENTICATED`、HTTP 422 `PHONE_CODE_REJECTED`、HTTP 409 `PHONE_IN_USE` 或 `PRIMARY_PHONE_ALREADY_BOUND`、HTTP 503 `PHONE_BINDING_UNAVAILABLE`。错误与成功均不返回内部 user ID、openid、完整手机号、code、access token、AppSecret 或 provider body。

### Use one in-process stable-token manager on fixed official endpoints

`internal/wechat` 新增一个 runtime phone client，复用现有结构化 AppID/AppSecret，但 main 不接收 base URL。runtime 固定调用：

1. `POST https://api.weixin.qq.com/cgi-bin/stable_token`，JSON 精确包含 `grant_type=client_credential`、AppID、AppSecret 与 `force_refresh=false`；
2. `POST https://api.weixin.qq.com/wxa/business/getuserphonenumber?access_token=...`，JSON 精确包含 getPhoneNumber code 与当前已认证用户的 openid。

package-local tests 可注入两个 `httptest` endpoint；该入口不进入 config/main。每个 wire 调用使用三秒 whole-request timeout、拒绝 redirect、16 KiB response cap、严格单 JSON object，并使用独立的 non-reusing HTTP/1.1 transport 禁用 keep-alive 与 HTTP/2，确保服务端不透明重放 phone code。transport/protocol、invalid-token、watermark、token 或 provider schema 失败均不重试当前 code。

token manager 只在进程内保存 access token 与 refresh deadline。一次 cache miss 只有一个 goroutine 发起 stable-token 请求，其他并发 caller 等待同一结果；成功按 `expires_in` 计算过期，并在到期前五分钟停止复用。若返回剩余有效期不超过五分钟，token 只供当前合并批次使用而不进入可复用 cache。invalid-token 结果可驱逐 cache，但当前手机号 code 仍直接 503，不自动 refresh/replay；后续新点击 code 才可触发新 token 获取。未采用 Redis/DB cache、后台 refresh 或强制刷新，因为单实例内正确性已满足当前范围，跨实例缓存一致性不是本 change 契约。

### Bind provider data to both authenticated openid and configured AppID

Repository 通过已认证 `user_id` 只读取当前 v8 `openid`，provider 请求携带该 openid，利用官方当前契约校验 phone code 的用户绑定关系。成功响应还必须满足 `watermark.appid` 与配置 AppID byte-exact 相等；不匹配按 503 fail closed。

规范手机号只由 `countryCode + purePhoneNumber` 构造：两者必须为 ASCII digits，country code 首位不得为零，组合后必须符合 `+` 加 1–15 位数字且总字节数不超过 `VARBINARY(16)`。`phoneNumber` 必须为合法非空字符串但不作为规范化来源，避免不同国家展示格式破坏唯一键。任何格式/字段矛盾均视为 provider protocol unavailable，不写库。

官方当前 getPhoneNumber 专用错误 `40013`（code AppID 不匹配）与 `40029`（code invalid）映射 422；配额、系统、传输、protocol、通用 invalid-token 与未知错误映射 503。实现不得依赖或透传 `errmsg`。未采用客户端提交手机号或本地解密，因为它们绕过当前官方 server API 与可信来源。

### Use one short transaction after the provider call

数据库调用分三段：Authenticate transaction 外查 active session；无事务预读当前用户的 openid/绑定状态；仅在确实未绑定时调用 provider；得到规范手机号后才开启绑定事务。事务以 `SELECT ... FOR UPDATE` 锁当前 user 并重读手机号：

- 仍未绑定：写入手机号与首次绑定时间；
- 已绑定同号：保持原号码/时间并返回同一 200；
- 已绑定异号：不改数据并返回 409 `PRIMARY_PHONE_ALREADY_BOUND`；
- 另一 user 已占用该号：唯一键冲突回滚并返回 409 `PHONE_IN_USE`。

provider 调用期间绝不持有数据库事务。若同一 phone code 并发调用中一个 provider wire 成功并绑定、另一个收到 code rejected，失败分支只重读当前用户绑定状态：若已绑定则返回当前 200，否则返回 422；不重试 provider。该恢复只证明最终绑定状态，不把 provider 拒绝伪装成第二次兑换成功。

### Add one forward-only v10 ALTER without rewriting v8/v9

`000010_add_miniprogram_primary_phone.sql` 只包含一个 `ALTER TABLE miniprogram_users`：新增 nullable `primary_phone VARBINARY(16)`、nullable `primary_phone_bound_at TIMESTAMP(6)`、`UNIQUE KEY uq_miniprogram_users_primary_phone (primary_phone)`，以及要求两字段同时 NULL 或同时非 NULL 的 CHECK。现有行保持两字段 NULL，MySQL UNIQUE 允许多行 NULL。

手机号按 ASCII E.164 bytes 比较，避免 collation/格式等价导致唯一性漂移。首次绑定时间使用注入 clock 的 UTC microsecond 值；幂等重试不得改写。没有 down migration；binary rollback 可继续忽略 additive columns，删除或改号必须另立 forward migration/change。

### Gate local proof separately from real platform proof

本地 provider tests 只证明 method/path/query/body、openid、watermark、cache/并发、exact-one-attempt、错误和 canary secrecy。writer-managed 隔离 MySQL 8 必须证明 v1→v10/repeat、exact schema、锁/同号/异号/跨 user 唯一、事务回滚、same-code 恢复及既有 catalog/menu/session 回归。

真实已认证非个人主体、手机号能力/额度、真实凭据、官方网络与新鲜点击 code 当前保持 `BLOCKED_EXTERNAL/NOT_RUN`；缺任一资产时不得把 stub 或数据库 PASS 升级为真实平台 PASS。

## Risks / Trade-offs

- [Risk] access token 在缓存期内被平台提前判无效。→ 驱逐 cache、当前 code 直接 503 且不重放；客户端必须由下一次用户点击取得新 code。
- [Risk] phone code 已被 provider 消费但数据库提交失败。→ 返回稳定 503，不持跨 provider transaction，也不自动重试；用户重新点击后用新 code，数据库仍保持未绑定或已提交的唯一最终状态。
- [Risk] 两个不同手机号同时绑定同一用户。→ user row lock 串行化，首个提交获胜，后者稳定 409 且不覆盖。
- [Risk] 多实例各自持有 stable token cache。→ 官方普通模式在 token 有效期内返回同一 token；本 change 只承诺进程内合并，不引入未要求的分布式 cache。
- [Risk] v10 是 forward-only。→ 上线前 rollback 只回退 binary；迁移后旧 binary 忽略 additive nullable columns，任何结构回撤另立 migration。
- [Risk] 真实主体/额度/网络无法本地模拟。→ 保持 delivery Gate `BLOCKED_EXTERNAL/NOT_RUN`，不影响本地候选但阻塞真实平台交付结论。

## Migration Plan

1. DRAFT 经主会话明确批准后进入 IMPLEMENTING，按 tasks 先取得 provider/API/schema/concurrency 的真实 Red。
2. 添加单语句 v10 并更新所有 embedded/catalog/menu/migrate/identity migration 断言；在隔离 MySQL 8 验证 v1→v10 与 repeat。
3. 实现 wechat token/phone client、identity phone repository/service/handler、route/main/smoke，始终保持 provider 调用在 transaction 外。
4. 重跑同一 focused RGR、真实 MySQL W3、全 Go/static/smoke/回归、strict、diff 与敏感信息 Gate；只在全部 PASS 后形成 candidate。
5. 由不同 verifier 在 clean detached worktree 对 exact candidate 从头只读验证。push、PR、integration、archive、deploy、真实平台调用与生产写入均需各自单独授权。

## Open Questions

无行为、公共契约、数据、授权或验收方式的未决问题。官方当前已支持请求携带 openid；若该 primary contract 在实现前移除或改变该字段，当前设计失效并必须回到 DRAFT，不得静默省略绑定校验。
