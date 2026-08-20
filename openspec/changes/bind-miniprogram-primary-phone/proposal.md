## Why

当前已登录小程序用户已有 hash-only Bearer session，但服务端尚不能把用户主动触发 `getPhoneNumber` 得到的一次性 code 安全绑定为唯一主手机号。该能力是首次结算手机号拦截与个人中心主动绑定共用的最小后端契约，且必须先于员工白名单、算价和交易模块独立验收。

## What Changes

- 新增唯一公共路由 `POST /api/v1/me/bind-phone`：只接受 route-specific Bearer session 与严格 JSON `code`，不安装全局 auth middleware；成功与已绑定重试均返回同一 HTTP 200 脱敏结果，不新增 profile GET。
- 新增固定官方 `stable_token` 普通模式缓存与 `getPhoneNumber` client：进程内合并并发 token refresh、按 `expires_in` 提前刷新，并用当前 session 对应 openid 调用官方当前支持的 openid/code 绑定校验；手机号 code、传输/协议失败与 invalid-token 失败均不自动重放。
- 新增连续 v10 `ALTER miniprogram_users`：nullable E.164 主手机号、首次绑定时间、手机号唯一键与成对 NULL CHECK；不修改 v8/v9。
- 在 provider 调用之外完成数据库事务：锁定当前用户、一次绑定，并稳定处理同号幂等、异号冲突、跨用户唯一冲突与同一 code 并发拒绝后的绑定状态重读。
- 保持真实微信验收为显式 delivery Gate；本地 stub 只证明 wire/cache/error，不能证明真实主体资格、额度、凭据、网络或新鲜 code。

## Capabilities

### New Capabilities

- `miniprogram-primary-phone`: 定义受 Bearer session 保护的微信主手机号兑换、E.164 标准化、一次绑定、并发/唯一性、错误与敏感信息边界。

### Modified Capabilities

无。

## Impact

- Primary outcome：已登录用户从首次结算拦截或个人中心主动点击 `getPhoneNumber` 后，只通过 `POST /api/v1/me/bind-phone` 一次绑定当前微信身份的主手机号，并收到稳定脱敏结果。
- Owner：同一 Writer agent；branch `codex/bind-miniprogram-primary-phone`；worktree `/Users/vivix/.codex/worktrees/order-bind-miniprogram-primary-phone.Writer`。
- Base SHA：`73cf1e04d6661ad08cbd2b03c2bb8324d4769f4d`。
- Owned paths：
  - `openspec/changes/bind-miniprogram-primary-phone/**`
  - `services/api/migrations/000010_add_miniprogram_primary_phone.sql`
  - `services/api/migrations/embed_test.go`
  - `services/api/internal/catalog/migrations_test.go`
  - `services/api/internal/catalog/mysql_integration_test.go`
  - `services/api/internal/menu/mysql_integration_test.go`
  - `services/api/internal/migrate/mysql_integration_test.go`
  - `services/api/internal/identity/mysql_integration_test.go`
  - `services/api/internal/identity/phone_*.go`
  - `services/api/internal/wechat/**`
  - `services/api/internal/httpapi/router.go`
  - `services/api/internal/httpapi/router_test.go`
  - `services/api/cmd/order-api/main.go`
  - `services/api/scripts/smoke.sh`
  - `services/api/scripts/miniprogram-phone-integration.sh`
- Read-only shared contracts：`AGENTS.md`、`docs/quality/change-quality-gates.md`、`.agents/skills/order-run-loop/references/self-evolution.md`、0818 PRD §4.1/§8/§9/§12–§14/§15.6.6、`openspec/changes/adopt-0818-prd-baseline/specs/mvp-product-baseline/spec.md`、`openspec/changes/establish-miniprogram-user-session/**`、v8/v9 migration 与现有 catalog/menu/session 公共契约。
- Dependencies：`establish-miniprogram-user-session` 已在 exact base main `73cf1e04d6661ad08cbd2b03c2bb8324d4769f4d` 进入 `INTEGRATED`；无未满足的 change 依赖。migration、`internal/wechat/**`、router 与 main 在本 change 完成前不得有另一个 writer 并行修改。
- Gate：`gate_type=W3`；`ui_level_target=UI0`；DRAFT 时 `ui_level_actual=NOT_RUN`。实现阶段 writer-managed 隔离 MySQL 8 是必跑本地 Gate，不得以 mock 替代。
- Delivery external assets：已认证非个人主体小程序、手机号能力与可用额度、真实 AppID/AppSecret、官方网络和一次用户新鲜点击产生的 `getPhoneNumber` code 当前均为 `BLOCKED_EXTERNAL/NOT_RUN`；owner 为客户小程序管理员与开发方，条件齐备且另获真实调用授权后恢复。
- Non-goals：附加手机号/P4、姓名、员工白名单/员工身份/折扣/P5、商户名单/RBAC、PC QR/P3、checkout/order/payment、前端、解绑/换号、全局 auth middleware、profile、Redis/DB token cache、生产 SSM；P1/P2 也不触及。不新增兼容 route、可配置微信 origin、token refresh endpoint 或 provider 自动重试。
- 唯一接受裁决：仅当 v10、route-specific auth、固定 provider wire、token cache/concurrency、一次绑定/唯一性/事务/恢复、稳定错误、敏感信息、真实 MySQL W3、catalog/menu/session 回归及全部 writer Gate 通过时，结果为 `ACCEPT`；任一条件失败即 `REJECT`。本地 `ACCEPT` 不把真实微信 `BLOCKED_EXTERNAL/NOT_RUN` 升级为交付 PASS。
