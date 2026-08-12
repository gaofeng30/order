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

- 状态：`CANDIDATE`。主 Agent 于 2026-08-13 明确授权从 APPROVED exact SHA `c17ba4a5bfe7556b779fac093925df609358fe05` 进入 apply；writer 已完成 Red→Green→Refactor、锁摘要真实 MySQL 8.0 W3 与全部 writer Gate，等待 exact-SHA independent verification。
- `approval_date`：`2026-08-13`；`approver`：`主 Agent`。批准依据：单能力 `W3/UI0`，canonical `api-service-bootstrap` health 以完整 MODIFIED delta 表达，MySQL 连接、迁移、readiness、真实 W3 与 production secret 边界均唯一冻结，无行为未决，owned paths、依赖、非目标和 42 tasks 完整，strict PASS。该记录是主 Agent 在用户授予的自主裁决范围内作出的规划裁决，不表述为用户亲自确认。
- 实施环境校正：同一授权把专属 `order-mysql-w3` profile 的磁盘从规划值 20 GiB 收紧为 10 GiB；这不改变产品、数据或公共契约，以下 artifacts 以 10 GiB 为唯一执行值。
- `base_sha`：`5ba5340cf9098724c0eb2284fdc5b14cb97be5dc`；`candidate_sha=SELF`（由本地 candidate commit 生成并在 handoff 绑定精确 SHA）。
- `gate_type`：`W3`，因为 apply 改变 migration、持久化 schema、并发锁和 readiness 数据结果；`ui_level_target=UI0`，`ui_level_actual=UI0`，无用户界面，JS syntax 与 42 JSON 静态检查 PASS。
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
- 本地研发前置：进入 apply 时宿主没有 `mysql`、Docker 或 Colima；writer 已自行安装 Colima 0.10.3、Docker CLI 29.7.2 与 Lima 2.2.0，并建立唯一 `order-mysql-w3` profile，当前环境状态为 `ESTABLISHED`、真实 W3 为 `PASS`。2026-08-13 当次 Docker Official Registry 显示最新 8.0 patch 为 `mysql:8.0.46-oraclelinux9`，manifest list digest 为 `sha256:7dcddc01f13bab2f15cde676d44d01f61fc9f99fe7785e86196dfc07d358ae2b`，`linux/arm64/v8` platform digest 为 `sha256:213bbfaf699693a40a20a12bb4342d2589a15a3dc7153db698eaed252a92458e`；writer 已从同一 platform digest 重建 fresh 容器并完成全矩阵，当前唯一容器在 loopback 随机端口健康运行，使用 0600 一次性凭据与 noexec/nosuid 1 GiB tmpfs，不含 host bind/volume，随机测试 schema 残留为零。生产 SSM/CAM/云账号仍属于独立 change/外部门禁，但不阻塞本地 W3。
- 非目标：全部业务表、repository 和业务 API；catalog/user/order/inventory/payment；inbox/outbox/worker；ORM、seed、自动 migration、down migration；SSM SDK、CAM、部署、COS、云 HA/备份/RPO 证明；前端；通用 secret/provider abstraction；`internal/app/**`、middleware、产品文档、腾讯云指南、quality/loop skills、`apps/**`、`AGENTS.md` 和历史 archived artifacts。
- 最小实现成功标准：结构化连接和敏感配置边界、migration 不变量、200/503 readiness 契约及专属 Colima 中真实 MySQL 8.0 Red→Green→Refactor 矩阵全部通过；所有 writer Gate 与 exact-SHA independent verification 完成；diff 只含 owned paths 且 worktree clean。环境尚未建立或测试尚未运行时只能记录 `NOT_ESTABLISHED`/`NOT_RUN`，不得形成 candidate；apply writer 必须自行闭环到真实 W3 PASS，不向客户或平台转交本地环境。
