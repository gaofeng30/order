## Why

当前 `order-api` 的 `/health/ready` 只证明进程已构建，仓库没有 MySQL 连接、schema migration 或数据库版本门禁；后续商品目录如果直接落库，会被迫自行决定连接、迁移、并发锁和恢复语义。需要先建立一个不含业务表的 MySQL 8.0 持久化基础，使后续数据能力只依赖一条已验证的连接、迁移和 readiness 契约。

## What Changes

- 锁定 `github.com/go-sql-driver/mysql v1.10.0`，以结构化内存配置构造 MySQL 8.0 连接池；固定连接参数、TLS、UTC、`utf8mb4`、超时、池大小、生命周期和脱敏边界，不接收原始 DSN。
- 新增唯一迁移入口 `order-migrate`、embed 的单调递增 forward-only SQL、`schema_migrations`、SHA-256 checksum、dirty/too-new 检查，以及在同一 `*sql.Conn` 上持有 `GET_LOCK('order_schema_migrate', 30)` 的串行执行。
- 把 `GET /health/live` 保持为进程存活契约，把 `GET /health/ready` 升级为真实数据库可达且 schema 与当前二进制完全兼容的 200/503 JSON 契约；`order-api` 启动和请求路径绝不自动迁移。
- 冻结 apply writer 自行建立的专属可回收 Colima profile 与摘要锁定的官方 MySQL 8.0 镜像，并增加真实集成验收，覆盖首次、重复、并发、失败、脏版本、behind、unreachable、too-new、无自动迁移和迁移后 ready；mock/fake 只覆盖内部编排，不能替代 W3 PASS。
- 生产模式拒绝原始 DSN 或密码环境变量。当前 change 不实现 SSM/CAM，生产配置必然因缺少未来 `load-runtime-secrets-from-ssm` 而 fail fast，不宣称生产可启动。
- 更新根 README 与现有 smoke，使“无数据库、readiness 仅代表进程”的说明不再失真；本 change 必须在 `serve-persistent-menu-catalog` 之前集成。

## Capabilities

### New Capabilities

- `mysql-persistence-foundation`: 定义结构化 MySQL 8.0 连接、显式 forward-only migration、schema 兼容门禁及真实数据库 W3 验收。

### Modified Capabilities

- `api-service-bootstrap`: 将 `/health/ready` 从无外部依赖时的进程就绪契约改为数据库和 schema 就绪契约；`/health/live`、405、404、middleware 与 HTTP 生命周期不变。

## Impact

- 状态：`APPROVED`。本轮只记录规划批准，不执行任何 apply task；acceptance verdict 仍只证明四类规划 artifact 完整、所有 tasks 未勾选、`openspec validate establish-mysql-persistence-foundation --strict` 和规划结构/owned-path 检查通过，不得把 APPROVED、mock 或缺失的真实 MySQL 资产写成实现 PASS。
- `approval_date`：`2026-08-13`；`approver`：`主 Agent`。批准依据：单能力 `W3/UI0`，canonical `api-service-bootstrap` health 以完整 MODIFIED delta 表达，MySQL 连接、迁移、readiness、真实 W3 与 production secret 边界均唯一冻结，无行为未决，owned paths、依赖、非目标和 42 tasks 完整，strict PASS。该记录是主 Agent 在用户授予的自主裁决范围内作出的规划裁决，不表述为用户亲自确认。
- `base_sha`：`5ba5340cf9098724c0eb2284fdc5b14cb97be5dc`；`candidate_sha`：尚未产生。
- `gate_type`：`W3`，因为 apply 将改变 migration、持久化 schema、并发锁和 readiness 数据结果；`ui_level_target=UI0`，`ui_level_actual=NOT_RUN`，无用户界面。
- owner：`MySQL Persistence Planning Writer`；branch `codex/establish-mysql-persistence-foundation`；独立 worktree `/Users/vivix/.codex/worktrees/7f1c/order`。同一路径不得存在第二 writer，且不得回退其他 worktree 的改动。
- owned paths：
  - `openspec/changes/establish-mysql-persistence-foundation/**`
  - `go.mod`
  - `go.sum`
  - `services/api/cmd/order-api/main.go`
  - `services/api/cmd/order-migrate/**`
  - `services/api/internal/config/**`
  - `services/api/internal/database/**`
  - `services/api/internal/migrate/**`
  - `services/api/internal/httpapi/health.go`
  - `services/api/internal/httpapi/router.go`
  - `services/api/internal/httpapi/router_test.go`
  - `services/api/migrations/**`
  - `services/api/scripts/mysql-integration.sh`
  - `services/api/scripts/smoke.sh`
  - `README.md`
- 只读共享契约：根 `AGENTS.md`、`docs/quality/change-quality-gates.md`、归档 `production-architecture-baseline` 与 `mvp-product-baseline`、canonical `openspec/specs/api-service-bootstrap/spec.md`、归档 `openspec/changes/archive/2026-08-12-bootstrap-api-service/**`、其余 `openspec/specs/**`、`services/api/internal/app/**`、`services/api/internal/httpapi/middleware.go`、产品文档、腾讯云指南、quality/loop skills、`apps/**` 和其他历史归档 artifacts。
- 依赖：只读遵循已归档 `production-architecture-baseline` 的 MySQL 8.0、独立 `order-migrate`、`schema_migrations`、`GET_LOCK`、部署前迁移和 DB+schema readiness；`bootstrap-api-service` 已在 base 上归档，其 canonical `api-service-bootstrap` health requirement 是本 change 的 MODIFIED 前置。`serve-persistent-menu-catalog` 必须等待本 change 进入 `INTEGRATED`，并与本 change 串行占用根 README、`go.mod` 和 router 装配点。
- 本地研发前置：当前宿主已确认没有 `mysql`、Docker 或 Colima，状态为 `NOT_ESTABLISHED`，真实 W3 测试为 `NOT_RUN`；这不是客户/平台 `BLOCKED_EXTERNAL`，也不是人工 TODO。获得 apply 批准后，writer 必须在任何 Red 前自行安装/核验 Homebrew stable Colima v0.10.3（当前 arm64 bottle SHA-256 `a9dfd1fa0a4aee62fef75974f39f174e4da774f7ba495c43dd0bcc23633381b8`），建立唯一 profile `order-mysql-w3`，用 `linux/arm64` 官方 `mysql:8.0.45-oraclelinux9` 镜像摘要 `sha256:0e7040b532c0f2ac8cb822695d33025522acd5252175cb104a5929aa66b40222` 启动只绑定 loopback 的隔离实例，生成一次性凭据并记录清理责任；candidate 前必须完成真实 W3 PASS。生产 SSM/CAM/云账号仍属于独立 change/外部门禁，但不阻塞本地 W3。
- 非目标：全部业务表、repository 和业务 API；catalog/user/order/inventory/payment；inbox/outbox/worker；ORM、seed、自动 migration、down migration；SSM SDK、CAM、部署、COS、云 HA/备份/RPO 证明；前端；通用 secret/provider abstraction；`internal/app/**`、middleware、产品文档、腾讯云指南、quality/loop skills、`apps/**`、`AGENTS.md` 和历史 archived artifacts。
- 最小实现成功标准：结构化连接和敏感配置边界、migration 不变量、200/503 readiness 契约及专属 Colima 中真实 MySQL 8.0 Red→Green→Refactor 矩阵全部通过；所有 writer Gate 与 exact-SHA independent verification 完成；diff 只含 owned paths 且 worktree clean。环境尚未建立或测试尚未运行时只能记录 `NOT_ESTABLISHED`/`NOT_RUN`，不得形成 candidate；apply writer 必须自行闭环到真实 W3 PASS，不向客户或平台转交本地环境。
