# mysql-persistence-foundation Specification

## Purpose

当前 `order-api` 的 `/health/ready` 只证明进程已构建，仓库没有 MySQL 连接、schema migration 或数据库版本门禁；后续商品目录如果直接落库，会被迫自行决定连接、迁移、并发锁和恢复语义。需要先建立一个不含业务表的 MySQL 8.0 持久化基础，使后续数据能力只依赖一条已验证的连接、迁移和 readiness 契约。
## Requirements
### Requirement: Database configuration is structured, bounded, and secret-safe

系统 MUST 锁定 `github.com/go-sql-driver/mysql v1.10.0`，并只允许 database 构造器接收内存中的结构化连接配置；构造器和运行日志 MUST NOT 接收、返回或记录原始 DSN。结构化配置 MUST 包含 host、`uint16` port、database、user、password 与 TLS mode，并固定使用 TCP、`parseTime=true`、UTC location、session `time_zone='+00:00'`、driver 结构化 `mysql.Charset("utf8mb4", "utf8mb4_0900_ai_ci")` option、3 秒连接 timeout、5 秒 read timeout 与 5 秒 write timeout。

TLS mode MUST 且只能是 `required` 或 `disabled`；`required` MUST 验证服务端证书，`disabled` 只允许 dev/test，`skip-verify`、`preferred`、明文 fallback、multi-statements、参数插值和 unrestricted local infile MUST 被拒绝。端口、空字段、TLS mode 和结构化参数必须在建池前校验，错误只返回字段名与枚举原因，不得包含 password、格式化 DSN、host/user/database 组合或连接响应正文。

`ORDER_ENV` MUST 只接受 `development`、`test`、`production`，默认 `development`。dev/test MUST 只通过 `ORDER_DB_HOST`、`ORDER_DB_PORT`、`ORDER_DB_NAME`、`ORDER_DB_USER`、`ORDER_DB_PASSWORD`、`ORDER_DB_TLS_MODE` 注入隔离凭据；`ORDER_DB_DSN` 在全部模式都不受支持。production MUST 拒绝任何 `ORDER_DB_DSN` 或 `ORDER_DB_PASSWORD` 环境变量，并在未来 `load-runtime-secrets-from-ssm` 集成前以脱敏的 `production_secret_source_unavailable` 配置错误阻止启动，不得回退到环境密码、默认密码、文件或通用 provider abstraction。

#### Scenario: Development receives valid structured credentials
- **WHEN** `ORDER_ENV=development` 且六个结构化数据库字段均有效
- **THEN** 配置层产生内存结构化连接配置，driver connector 使用固定的 TCP、UTC、charset、TLS 与 timeout 参数
- **AND** 任何可打印值、错误或日志都不包含 password 或格式化 DSN

#### Scenario: Raw DSN is supplied
- **WHEN** 任一模式存在 `ORDER_DB_DSN`
- **THEN** 配置在建立连接池前失败并返回原始 DSN 不受支持的枚举错误
- **AND** 错误不得回显该变量的值

#### Scenario: Production receives an environment password
- **WHEN** `ORDER_ENV=production` 且存在 `ORDER_DB_PASSWORD` 或 `ORDER_DB_DSN`
- **THEN** 配置在任何网络连接前失败并返回生产密钥来源违规
- **AND** 不读取、不格式化或记录凭据内容

#### Scenario: Production has no raw secret
- **WHEN** `ORDER_ENV=production` 且未提供原始 DSN 或密码环境变量
- **THEN** 当前 change 仍以 `production_secret_source_unavailable` 阻止启动
- **AND** 不宣称 production 已可运行，恢复依赖独立的 `load-runtime-secrets-from-ssm` change

#### Scenario: Unsafe TLS or connection field is supplied
- **WHEN** TLS mode 不是 `required`/`disabled`、production 请求 `disabled`，或 port/host/database/user/password 任一无效
- **THEN** 配置在建池前失败并只返回字段名与枚举原因
- **AND** 不启用 `skip-verify`、`preferred`、明文 fallback、multi-statements、参数插值或 unrestricted local infile

### Requirement: One bounded pool follows the process lifecycle

`order-api` MUST 创建一个 `database/sql` pool，固定 `MaxOpenConns=10`、`MaxIdleConns=10`、`ConnMaxLifetime=3m`、`ConnMaxIdleTime=1m`。建池 MUST 只校验本地结构和 connector，不得因启动自动 ping、查询 schema 或执行 migration；数据库可达性与 schema 状态只能由只读 checker 实时分类，HTTP 映射由 `api-service-bootstrap` 的 MODIFIED health requirement 定义。

`main.go` MUST 是 pool 的唯一生命周期 owner：配置或 connector 创建失败时不得监听 HTTP；创建成功后只把 pool/readiness 依赖装配给现有 router，并在 `app.Run` 返回后关闭 pool。`internal/app/**` MUST 保持只读，应用启动、systemd 重启、健康请求和 shutdown MUST NOT 执行 migration。

#### Scenario: Database is unreachable during API startup
- **WHEN** 结构化配置有效但目标 MySQL 当前不可达
- **THEN** `order-api` 建立惰性 pool 并启动 HTTP，只读 checker 报告 `database_unreachable`
- **AND** pool 构造、应用启动与 checker 均不写入 schema

#### Scenario: Process stops
- **WHEN** `app.Run` 因正常 signal 或错误返回
- **THEN** `main.go` 关闭唯一 pool 且不再建立连接
- **AND** 原有 HTTP shutdown timeout 与退出语义保持不变

### Requirement: Embedded migrations have immutable forward-only identity

生产 migration 集合 MUST 从 `services/api/migrations/*.sql` 编译期 embed；文件名 MUST 严格匹配六位连续编号 `NNNNNN_name.sql`、从 `000001` 开始、无缺号或重复。每个文件 MUST 只含一个可由 MySQL 8.0 执行的 SQL statement，不得包含 down、seed、repository/ORM 指令或运行时文件读取。本 change 的初始集合 MUST 且只能包含 `000001_create_schema_migrations.sql`，并只创建 `schema_migrations`；未来业务 change 必须通过自己的 OpenSpec、ownership 和 W3 Gate 追加更高编号业务 migration，不得塞入本 change。

`schema_migrations` MUST 且只能持久化 `version BIGINT UNSIGNED` 主键、`name VARCHAR(255)`、原始文件 bytes 的 SHA-256 `checksum BINARY(32)`、`dirty BOOLEAN` 与可空 `applied_at TIMESTAMP(6)`。已成功版本的编号、name 和 checksum MUST 在每次运行及 readiness 时与当前 embedded 集合完全一致；已应用文件不得被修改、重命名、删除或重新编号。

#### Scenario: Fresh schema is inspected
- **WHEN** 数据库尚不存在 `schema_migrations`
- **THEN** readiness 报告 `schema_uninitialized` 且不创建任何对象
- **AND** 首次 `order-migrate` 只通过 `000001_create_schema_migrations.sql` 创建该基础表并记录 version 1 的 clean checksum

#### Scenario: Migration files are malformed
- **WHEN** embedded 集合存在非六位编号、缺号、重复、多个 SQL statements、down/seed 或首个 migration 不是唯一 schema table DDL
- **THEN** runner 在连接或修改数据库前失败
- **AND** 不产生 `schema_migrations` 或任何业务对象

#### Scenario: Applied migration content drifts
- **WHEN** clean 数据库记录的 name 或 checksum 与同版本 embedded 文件不同
- **THEN** migrate 和 readiness 都返回 `schema_checksum_mismatch`
- **AND** 不自动覆盖 checksum、重跑 SQL 或把漂移解释为 behind

### Requirement: Migration execution is locked, failure-visible, and forward-only

`order-migrate` MUST 先从 pool 取得一个专用 `*sql.Conn`，再在该同一连接上执行 `GET_LOCK('order_schema_migrate', 30)`；schema 检查、所有 migration SQL、version 写入和 `RELEASE_LOCK` MUST 全部使用该连接。锁结果为 0、NULL、超时或查询错误 MUST 失败退出且不修改 schema；进程异常或显式释放都不得把锁转移到 pool 中继续复用。

取得锁后，runner MUST 先验证目标是 MySQL `8.0.x`、session 为 UTC/`utf8mb4`，再验证全部 clean 历史。首次 migration 成功创建表后才能写入 clean version 1；后续每个 version MUST 先插入 `dirty=true, applied_at=NULL`，执行唯一 SQL statement，成功后才更新为 `dirty=false` 和数据库 UTC `applied_at`。失败 MUST 保留 dirty row、不得存在该 version 的 clean 成功记录、不得继续后续 version；再次运行遇到任一 dirty row MUST 在执行 SQL 前失败。

数据库最高 clean version 高于 embedded latest、存在 embedded 集合未知版本或缺失较低历史 MUST 返回 `schema_too_new`；低于 latest 且历史一致时为 behind，runner MUST 按编号依次补齐；完全一致时重复运行 MUST 零写入成功。runner MUST NOT 自动 down、删除列、清 dirty、恢复备份或执行应用业务代码。

#### Scenario: Two runners migrate concurrently
- **WHEN** 两个 `order-migrate` 同时连接同一 behind 数据库
- **THEN** 只有持有命名锁的同一 MySQL connection 执行 SQL，另一个最多等待 30 秒
- **AND** 最终每个 version 只有一条 clean 记录和一次可观察 schema 副作用；等待者取得锁后只做一致性检查并零写入成功，未取得锁则失败

#### Scenario: Lock and migration use the same connection
- **WHEN** migration SQL 在真实 MySQL 8.0 中比较 `IS_USED_LOCK('order_schema_migrate')` 与 `CONNECTION_ID()`
- **THEN** 两者相等且所有 version/history 操作观察到同一 connection ID
- **AND** 释放或关闭该连接后命名锁不再被持有

#### Scenario: A migration statement fails
- **WHEN** version 2 或更高的唯一 SQL statement 在真实 MySQL 8.0 返回错误
- **THEN** runner 立即失败，保留该 version 的 dirty row 且不记录 clean 成功或执行后续 version
- **AND** 下次运行因 `schema_dirty` 在任何 migration SQL 前停止

#### Scenario: Behind schema is migrated
- **WHEN** clean 历史是当前 embedded 集合的严格前缀
- **THEN** runner 按升序只应用缺失 versions并更新 clean checksum
- **AND** 再次运行不修改 schema 或 history

#### Scenario: Schema is too new
- **WHEN** 数据库含当前二进制未知或高于 embedded latest 的 clean version
- **THEN** runner 以 `schema_too_new` 失败且零写入
- **AND** 不删除、降级或忽略未知 version

### Requirement: Migration CLI has one deterministic contract

`services/api/cmd/order-migrate` MUST 是唯一 migration executable。无参数执行时 MUST 加载与 `order-api` 相同的结构化数据库配置、运行全部 pending forward migrations并关闭 pool；成功或已最新退出码为 0，配置、连接、版本、锁、checksum、dirty 或 SQL 失败退出码为 1，任何参数或 flag 都返回最短 usage 并退出码 2。

CLI MUST 向 stderr 输出 JSON `slog`：成功只记录 `event=migration_complete`、`from_version`、`to_version`、`applied_count`、`duration_ms`；失败只记录 `event=migration_failed`、稳定 `reason`、非敏感 `version`（若有）与 `duration_ms`。日志、usage 和 error MUST NOT 包含原始 SQL、DSN、password、host、user、database、TLS material 或 MySQL 响应正文。

#### Scenario: No migrations are pending
- **WHEN** CLI 连接到 schema 与 embedded 集合完全一致的 MySQL 8.0
- **THEN** 以 0 退出并记录 `applied_count=0` 的单条完成摘要
- **AND** 不更新 `applied_at` 或执行 migration SQL

#### Scenario: CLI receives an argument
- **WHEN** 调用方传入任意 positional argument 或 flag
- **THEN** CLI 不连接数据库、输出脱敏 usage 并以 2 退出
- **AND** 不提供 down、force、seed、repair 或自动清 dirty 子命令

#### Scenario: Migration fails
- **WHEN** CLI 遇到配置、网络、版本、锁、checksum、dirty 或 SQL 错误
- **THEN** CLI 以 1 退出并记录稳定枚举 reason
- **AND** 不回显凭据、DSN、原始 SQL 或数据库错误正文

### Requirement: W3 acceptance uses an agent-managed real MySQL 8.0

本 change 在 DRAFT 阶段 MUST 不安装、下载或启动 MySQL、Docker、Colima 或其他 runtime；进入 IMPLEMENTING 时本地环境状态从 `NOT_ESTABLISHED` 开始，真实 W3 测试状态为 `NOT_RUN`。获得 apply 批准后，writer MUST 在任何 Red 前自行安装或核验 Homebrew stable Colima v0.10.3；arm64 bottle SHA-256 MUST 为 `a9dfd1fa0a4aee62fef75974f39f174e4da774f7ba495c43dd0bcc23633381b8`。writer 随后 MUST 建立唯一专属 profile `order-mysql-w3`：`linux/arm64`、Docker runtime、2 CPU、4 GiB memory、10 GiB disk、无 Kubernetes、无 workspace mount、仅 loopback port forwarding。

writer MUST 在实际 pull/run 前从 Docker Official Registry 枚举当前可用的精确 8.0.x tags并选择最新 patch，不得因本地缓存降级，不得运行浮动 `8.0`/`latest`。2026-08-13 当次冻结为 `mysql:8.0.46-oraclelinux9`，OCI manifest list digest `sha256:7dcddc01f13bab2f15cde676d44d01f61fc9f99fe7785e86196dfc07d358ae2b`，`linux/arm64/v8` platform digest `sha256:213bbfaf699693a40a20a12bb4342d2589a15a3dc7153db698eaed252a92458e`；pull/run 后 MUST 核对 repo digest、architecture 与容器内 `SELECT VERSION()`，不得运行 MariaDB、mock server 或不同 digest。一次性随机凭据 MUST 只存在于权限为 0600 的临时 env file/进程内存，实例只绑定 `127.0.0.1` 的随机 host port，不得记录凭据或 DSN。

`services/api/scripts/mysql-integration.sh` MUST 只连接这个已建立的专属 profile，不负责选择另一 runtime。脚本 MUST 要求结构化的 `ORDER_TEST_MYSQL_HOST`、`ORDER_TEST_MYSQL_PORT`、`ORDER_TEST_MYSQL_USER`、`ORDER_TEST_MYSQL_PASSWORD`、`ORDER_TEST_MYSQL_TLS_MODE`、`ORDER_TEST_MYSQL_INSTANCE=order-mysql-w3` 和显式安全闩 `ORDER_TEST_MYSQL_ISOLATED=YES`；apply 前缺失时保持 `NOT_ESTABLISHED/NOT_RUN`，apply 开始后环境建立或字段校验失败 MUST 记 FAIL 并由 writer 从首个错误继续修复，不得转交客户/平台或冒充 `BLOCKED_EXTERNAL`。

测试 MUST 先由 `SELECT VERSION()` 证明目标为真正 MySQL `8.0.x`，再只创建随机前缀 `order_test_` 的一次性 schema，覆盖首次/重复/并发 migration、锁同连接、失败不产生 clean version、dirty、behind、unreachable、too-new、应用绝不自动迁移、迁移后 ready；pass、fail 或中断都 MUST 尝试只删除本次已解析且前缀匹配的 schema，清理失败 MUST 使 Gate FAIL 并报告非敏感 schema 名。mock/fake 只能为内部编排制造 Red，不得替代这些真实存储场景。writer 可在候选后保留该专属 profile 给 exact-SHA verifier 重建空容器；最终验证完成后 MUST 删除容器、profile 与 data disk，失败时停止并报告，不得扩大删除。

#### Scenario: DRAFT is created before the local runtime
- **WHEN** 本轮只创建和校验 DRAFT，且宿主尚无 Colima/MySQL 8.0
- **THEN** 规划记录环境 `NOT_ESTABLISHED`、真实 W3 `NOT_RUN`，strict 可按规划完整性判定
- **AND** 不安装/启动 runtime，不记 `BLOCKED_EXTERNAL`，也不把 unit/mock 结果升级为 W3 PASS

#### Scenario: Apply begins without the local runtime
- **WHEN** change 已批准并准备进入 Red，但 `order-mysql-w3` profile 或锁摘要 MySQL 8.0 尚未就绪
- **THEN** writer 先完成冻结的 Colima profile、digest、loopback、随机凭据和 cleanup preflight
- **AND** 环境 Gate 失败由 writer 修复，未建立前不得执行 Red 或形成 candidate

#### Scenario: Real MySQL integration passes
- **WHEN** 脚本在 `order-mysql-w3` 的锁摘要真实 MySQL 8.0 上完成全部冻结场景且随机 schema 清理成功
- **THEN** 证据记录 Colima/engine 版本、image digest、脱敏 profile、命令退出码和场景摘要
- **AND** 不记录凭据、DSN、原始 SQL、server response 或与本次随机 schema 无关的数据

#### Scenario: Cleanup target is unsafe
- **WHEN** 测试 schema 为空、无法解析、不以 `order_test_` 开头或不属于本次创建记录
- **THEN** 清理立即失败且不得扩大、重试或改用更强删除方式
- **AND** Gate 保持 FAIL 并由当前 writer 处理，不转交客户/平台，不改用更强删除方式

#### Scenario: Exact-SHA verification finishes
- **WHEN** writer candidate 与 clean detached verifier 都已对锁摘要实例取得真实 W3 PASS
- **THEN** 负责验证闭环的 agent 删除 `order-mysql-w3` 容器、profile 和 data disk，并确认 profile 不存在
- **AND** 清理失败立即停止并报告精确目标，不删除其他 Colima profile 或宿主数据

### Requirement: Recovery is explicit and never a down migration

生产和测试失败恢复 MUST 保持 forward-only。应用回退 MUST 只使用兼容当前 schema 的旧二进制；schema 修正 MUST 使用新的更高编号 forward-fix migration，或在单独授权和已验证恢复点下恢复数据库。dirty version MUST 阻止自动继续；owner 必须先在隔离副本确认实际 schema、原 migration checksum 与修复 SQL，再通过单独审查和写授权执行窄范围人工修复，系统不得提供自动清 dirty、force 或 down 入口。

本 change 的验收清理 MUST 仅删除本次随机测试 schema；不得把腾讯云 HA、自动备份、恢复演练或 RPO/RTO 目标当成本 change 的 PASS 证据。

#### Scenario: New binary must be rolled back
- **WHEN** 新二进制在 migration 已成功后需要回退
- **THEN** 仅当上一版支持当前 schema 才回退二进制，否则先实施新的 forward fix
- **AND** 不执行自动 down 或删除 migration history

#### Scenario: Dirty schema needs repair
- **WHEN** migration 失败并留下 dirty version
- **THEN** 自动 runner 与 readiness 持续失败，直到 owner 完成隔离复现、备份/恢复点确认、review 和单独写授权
- **AND** 修复不得由当前 CLI 自动清 dirty 或隐藏失败 history
