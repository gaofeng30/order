## Why

当前后端只有匿名 catalog/menu 读取，没有把小程序启动时的一次性 `wx.login` code 兑换为内部普通用户和服务端会话的能力。该最小身份地基必须先独立建立，后续手机号绑定、员工/商户识别和受保护交易才能在不泄露微信凭据的前提下分别立项。

## What Changes

- 新增唯一公共路由 `POST /api/v1/auth/miniprogram/session`：小程序启动后提交一次性 code，后端调用微信官方 `code2Session`，原子 find-or-create 内部普通用户并创建一个不透明 Bearer session；成功响应只返回一次 raw token 及固定到期时刻。
- 新增连续 v8 `miniprogram_users` 与 v9 `miniprogram_sessions` migration：前者只保存内部 ID、唯一 openid、创建/最近登录时间；后者只保存 token SHA-256、用户外键、签发/到期时间。不得保存 `session_key`、`unionid`、原始 code 或 raw token。
- 开发与测试环境新增结构化 Mini Program AppID/AppSecret 配置；生产仍由现有 `production_secret_source_unavailable` fail-closed，不在本 change 引入 SSM secret loader 或任意微信 base URL 配置。
- 新增固定官方 HTTPS origin 的微信 client、严格 JSON handler、会话 service/crypto、MySQL 并发与回滚验证，并保持 catalog/menu 匿名路由及精确响应契约不变。

## Capabilities

### New Capabilities

- `miniprogram-user-session`: 定义微信一次性登录 code 兑换、内部普通用户原子建档、不透明短期 Bearer session、错误与敏感信息边界。

### Modified Capabilities

- None.

## Impact

- Owner：同一 Writer agent；branch `codex/establish-miniprogram-user-session`；worktree `/Users/vivix/.codex/worktrees/order-establish-miniprogram-user-session.Writer`。
- Base SHA：`2b83e93cc2a8d2bb16b606068028f34ee662b677`。
- Owned paths：
  - `openspec/changes/establish-miniprogram-user-session/**`
  - `services/api/migrations/000008_create_miniprogram_users.sql`
  - `services/api/migrations/000009_create_miniprogram_sessions.sql`
  - `services/api/migrations/embed_test.go`
  - `services/api/internal/catalog/migrations_test.go`
  - `services/api/internal/catalog/mysql_integration_test.go`
  - `services/api/internal/menu/mysql_integration_test.go`
  - `services/api/internal/migrate/mysql_integration_test.go`
  - `services/api/internal/config/config.go`
  - `services/api/internal/config/config_test.go`
  - `services/api/internal/wechat/**`
  - `services/api/internal/identity/**`
  - `services/api/internal/httpapi/router.go`
  - `services/api/internal/httpapi/router_test.go`
  - `services/api/cmd/order-api/main.go`
  - `services/api/scripts/miniprogram-session-integration.sh`
  - `services/api/scripts/smoke.sh`
- Read-only shared contracts：`AGENTS.md`、`docs/quality/change-quality-gates.md`、`docs/product/online-ordering-system-prd-0818.md` §3/§4/§8/§9/§12–§14、`docs/product/online-ordering-system-prd-0818-review.md`、`openspec/changes/adopt-0818-prd-baseline/specs/mvp-product-baseline/spec.md`、现有 `/api/v1/catalog*` 与 `/api/v1/menu` contract。
- Dependencies：`serve-reservation-menu-availability` 已在 base main 进入 `INTEGRATED`；无未满足的 change 依赖。migration、router 与 `cmd/order-api/main.go` 在本 change 完成前禁止另一个 writer 并行修改。
- `gate_type=W3`；`ui_level_target=UI0`；`ui_level_actual=UI0`。本地必需 external assets 为 none：隔离 MySQL 8 W3 由 writer 管理并作为必跑本地 Gate，不得以 mock 替代。
- Delivery external blocker：真实 Mini Program AppID/AppSecret、已认证小程序账号、可用网络及新鲜真实 `wx.login` code 当前为 `BLOCKED_EXTERNAL/NOT_RUN`；owner 为客户小程序管理员与开发方，条件齐备后才可执行真实平台验收。本地 HTTP stub 只证明 wire/error contract，不得写作微信平台 PASS。
- Non-goals：微信主手机号/`getPhoneNumber`/`/me/bind-phone`、附加手机号/P4、员工白名单/员工判定/折扣/P5、商户手机号绑定/主子角色/RBAC、PC QR/P3、auth middleware/受保护业务 endpoint/profile/logout/refresh、checkout/payment/order、前端、生产 SSM secret loader；P1/P2 也不触及。catalog/menu 继续匿名且 contract 不变。
- 唯一接受裁决：仅当 v8/v9、provider wire、严格 handler、crypto/session service、真实 MySQL 同 openid 并发唯一用户与事务回滚/到期、日志敏感信息、catalog/menu 回归及全部声明 writer Gate 通过，结果才为 `ACCEPT`；任一条件失败即 `REJECT`。`ACCEPT` 只表示本地候选可交独立验证，不把真实微信 `BLOCKED_EXTERNAL/NOT_RUN` 升级为交付 PASS。
