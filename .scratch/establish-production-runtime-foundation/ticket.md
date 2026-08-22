# establish-production-runtime-foundation

## 固定边界

- `base_sha`: `2c549f2c413a06ec160b4e886fb16cd62e18176f`
- 技术基线：单台境内 Tencent Cloud CVM、Nginx、systemd、TencentDB MySQL 8.0、SSM/CAM Role；Docker/Kubernetes/CloudBase Run 均为 N/A。
- `openspec/**` 是只读历史工件，不再作为 Codex/Cursor 工具入口，本 change 不修改。
- 不购买、创建、修改或探测腾讯云/微信真实资源，不提交真实 Secret，不部署。
- `main.go`、router、migration、业务模块、`apps/**` 均只读；API 不自动 migration/down，`order-migrate` 仍是唯一 forward-only 入口。

## 单一技术决策

腾讯云 SSM SecretName 使用唯一规则 `order-{env}-{purpose}`。`env` 与进程 `ORDER_ENV` 相同；`purpose` 必须来自当前代码真正需要的固定 allowlist：

- `db-password`
- `wechat-miniprogram-app-secret`

本 change 不新增 `session-signing-key` 或任何未来 purpose。当前 production 的两个实际名称固定为 `order-production-db-password` 和 `order-production-wechat-miniprogram-app-secret`，不由 EnvironmentFile 覆盖。尚无真实 SSM Secret，故不存在兼容读取、双轨命名或迁移。

官方依据（访问日期 2026-08-23）：

- 腾讯云 `CreateSecret.SecretName` 只允许字母、数字、`-`、`_`，首字符必须是字母或数字：<https://cloud.tencent.com/document/product/1140/40529>
- 腾讯云 Go SDK 提供 `common.DefaultCvmRoleProvider()`，并建议按需安装 common 与产品包：<https://github.com/TencentCloud/tencentcloud-sdk-go>
- SSM `GetSecretValue` 使用 `SecretName`、`VersionId`，当前版本为 `SSM_Current`：<https://cloud.tencent.com/document/api/1140/40522>

## Ownership

- `.scratch/establish-production-runtime-foundation/**`
- `services/api/internal/config/**` 及本 change 专属 config 测试
- `deploy/**`（仅本 change 新文件）
- 根 `go.mod`、`go.sum`（只允许腾讯云官方 Go SDK 所需依赖）
- `docs/product/online-ordering-system-technical.md` 仅 §2.5 命名/权限语义
- `docs/微信小程序开发和运维指南/腾讯云操作指南.md` 仅 §3.5 命名/权限语义

## 配置与 SecretSource seam

### EnvironmentFile（仅非敏感值）

root-owned、权限不宽于 `0640` 的 EnvironmentFile 只包含：

- `ORDER_ENV=production`
- `ORDER_API_HTTP_ADDR=127.0.0.1:8080`
- `ORDER_API_SHUTDOWN_TIMEOUT`
- `ORDER_DB_HOST` / `ORDER_DB_PORT` / `ORDER_DB_NAME` / `ORDER_DB_USER` / `ORDER_DB_TLS_MODE`
- `ORDER_WECHAT_MINIPROGRAM_APP_ID`
- `ORDER_TENCENT_REGION`

production 只要存在 `ORDER_DB_PASSWORD`、`ORDER_DB_DSN`、`ORDER_WECHAT_MINIPROGRAM_APP_SECRET`、`TENCENTCLOUD_SECRET_ID`、`TENCENTCLOUD_SECRET_KEY`、`TENCENTCLOUD_TOKEN` 或 `TENCENTCLOUD_CREDENTIALS_FILE`，即 fail closed。不得启用 SDK 默认 env/profile provider chain，只允许 `common.DefaultCvmRoleProvider()`。

### SSM/CAM Role

- `config.LoadWithSecretSource(...)` 是测试 seam；生产 `config.Load()` 创建仅使用 CVM instance role 的官方 SDK adapter。
- adapter 从 `ORDER_TENCENT_REGION` 选择 SSM region；endpoint 固定为 `ssm.tencentcloudapi.com`，无运行时 override。
- 两个 `GetSecretValue` 均读取 `SSM_Current`；文本分别装配到 `Database.Password` 与 `MiniProgram.AppSecret`，不接受二进制 Secret 或原始 DSN。
- region、CAM metadata、临时凭据、SSM、Secret 缺失/空/非法任一失败均在监听端口前 fail closed，不回退到文件、命令行、长期凭据或内置值。
- 错误和日志只保留稳定 reason/字段；不得包含 Secret、SecretName、临时 SecretId/SecretKey/token、endpoint/provider 原始响应。
- CAM Role 资源级策略只允许当前环境上述两个 Secret 的 `ssm:GetSecretValue`。

## systemd / health / recovery

- 新增低权限 systemd unit、无 Secret 的 EnvironmentFile 模板、健康检查脚本与 CVM 重建 runbook；Docker 明确 N/A。
- 发布顺序：仅破坏性 migration 前确认手动备份 → 同一发布物运行 `order-migrate` → 重启 `order-api` → `/health/live` → `/health/ready` → 外部 CLS/CAT/云监控 Gate。
- liveness 只证明进程；导流和发布成功必须以 `/health/ready` 的 DB/schema 结果为准。
- CVM 重建只恢复发布物、Nginx/systemd、EnvironmentFile，绑定当前环境 CAM Role 后迁移/启动/检查 readiness；CVM 本地不恢复业务事实。
- 本地只用 fake SecretSource 验证配置，用静态 gate 验证 deploy 工件；不访问真实云，不宣称真实 CAM/SSM、备份恢复或 RPO/RTO PASS。

## Red → Green → Refactor / Review / Verify

- [x] Contract Red：`rg -n '/order/\{env\}/' docs/product/online-ordering-system-technical.md docs/微信小程序开发和运维指南/腾讯云操作指南.md` 在 base 命中两处无效 slash 契约。
- [x] Behavior Red：production 完整非敏感配置仍固定返回 `production_secret_source_unavailable`；新增成功 seam 测试首次编译失败为 `undefined: LoadWithSecretSource`。
- [x] Green：fake SecretSource 可装配两个 production Secret，且请求名称精确等于固定 allowlist 派生值。
- [x] Green：缺失/空/非法配置、错误环境、直接 Secret/长期腾讯云凭据、CAM/SSM/内容错误均 fail closed。
- [x] Green：canary Secret、SecretName、临时凭据和 provider 原始响应不进入错误或启动日志。
- [x] Green：官方 SDK 仅使用 CVM role provider；依赖只改 `go.mod` / `go.sum` 所需条目。
- [x] Green：systemd/EnvironmentFile/health/recovery 静态 gate 通过，Docker=N/A。
- [x] Refactor：目标测试、`go test ./services/api/...`、race、vet、build、既有 smoke 通过。
- [ ] Review：Standards / Spec 双轴 review 无未解决 finding。
- [ ] Verify：中文提交后在另一个 clean detached worktree 对 exact candidate SHA 只读复验。

决定性证据：

- `go test ./services/api/... -count=1`、`go test -race ./services/api/... -count=1`、`go vet ./services/api/...`、`go build ./services/api/...`：PASS。
- `bash services/api/scripts/smoke.sh`：`smoke: PASS`。
- `bash deploy/checks/verify-runtime-foundation.sh`：`RUNTIME_FOUNDATION_GATE=PASS`。
- `bash deploy/checks/verify-production-startup.sh`：`PRODUCTION_STARTUP_GATE=PASS`，持久腾讯云凭据 canary 在监听前被拒绝且日志无泄漏。
- `CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath`：`order-api` 与 `order-migrate` 均为静态 ELF x86-64。

## 外部 Gate

- `BLOCKED_EXTERNAL`：真实地域、CVM/CDB、CAM Role、两个 SSM Secret、域名/HTTPS、微信 AppID/AppSecret、CLS/CAT/云监控均未提供或未验证。
- `NOT_RUN`：真实 SSM/CAM 调用、systemd/CVM 启动、CDB 主备切换、备份恢复、生产部署、目标 RPO/RTO。
