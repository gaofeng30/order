## Context

当前 base `5ba5340cf9098724c0eb2284fdc5b14cb97be5dc` 已归档 `bootstrap-api-service`，并生成含 6 Requirements/17 Scenarios 的 canonical `openspec/specs/api-service-bootstrap/spec.md`；其中既有 health requirement 冻结 live/ready 进程级 200、405 与 404 公共契约。代码仍只有一个 Go 1.26.5/Gin `order-api`：`config.Load` 读取 HTTP 地址和 shutdown timeout，`main.go` 把无依赖 router 传给 `internal/app.Run`；仓库没有数据库 driver、pool、migration executable、schema history 或真实数据库测试。本 change 以同名 MODIFIED requirement 完整替换既有 health 契约，MySQL 新 capability 不重复定义 HTTP 公共契约。

归档 `production-architecture-baseline` 已经冻结 TencentDB MySQL 8.0、发布前独立 `order-migrate`、forward-only SQL、`schema_migrations`、`GET_LOCK('order_schema_migrate', 30)`、DB+schema readiness，以及生产不自动 migration/down。本 change 只把其中 MySQL 持久化基础落成可实现、可验证的 W3/UI0 单能力，不重开产品或云架构选择。

规划时官方 driver 文档确认 `github.com/go-sql-driver/mysql v1.10.0` 支持 Go 1.24+、MySQL 5.7+、结构化 `mysql.Config`/`NewConnector`、`parseTime`、TLS、dial/read/write timeout 和 `database/sql` pool；本仓库进一步限制为 MySQL 8.0.x。参考：

- https://pkg.go.dev/github.com/go-sql-driver/mysql@v1.10.0
- https://github.com/go-sql-driver/mysql/tree/v1.10.0

当前宿主已只读确认 `mysql`、`mysqladmin`、Docker、Colima 均不存在。本轮 DRAFT 不得安装、下载或启动它们；本地环境记为 `NOT_ESTABLISHED`，真实 W3 记为 `NOT_RUN`。这不是客户/平台 `BLOCKED_EXTERNAL`：apply writer 必须在 Red 前自行建立冻结的专属 Colima/MySQL 8.0 环境，并在 candidate 前闭环真实 W3 PASS。

## Goals / Non-Goals

**Goals:**

- 用一个结构化、脱敏、固定参数的 MySQL 8.0 connector 和有界 pool 支撑 API 与 migrator。
- 用一个 compile-time embedded、checksum 不可漂移、同连接命名锁串行的 forward-only runner 管理 schema。
- 把 live 与 ready 分开：进程可活但 DB/schema 未就绪时明确 503，且任何 API 启动或 health 请求都不迁移。
- 用隔离真实 MySQL 8.0 对 migration 并发、失败、恢复和 schema 门禁建立 W3 证据。
- 让 `serve-persistent-menu-catalog` 只需依赖已集成的 pool/migration/readiness，不重复建立基础。

**Non-Goals:**

- 不创建商品、用户、订单、库存、支付或任何其他业务表，不创建 repository 或业务 API。
- 不实现 catalog、inbox/outbox、worker、事务业务规则、ORM、seed、自动 migration、down migration 或数据 backfill。
- 不实现 SSM SDK、CAM、systemd/CVM/CDB 部署、COS、监控、云 HA、备份恢复演练或 RPO/RTO 证明。
- 不修改 `internal/app/**`、middleware、产品文档、腾讯云指南、quality/loop skills、前端、根治理或历史 archived artifacts。
- 不创建 secret/provider 通用抽象，不允许 production 暂时回退到环境密码。

## Decisions

### D1. 只占用真实装配所需的最小路径

未来 apply 的职责固定为：

```text
services/api/
├── cmd/order-api/main.go              # 配置、pool、ready 装配与 Close owner
├── cmd/order-migrate/main.go          # 唯一 migration CLI
├── internal/config/                   # 环境模式与结构化 DB 配置
├── internal/database/                 # mysql connector、pool、状态检查
├── internal/migrate/                  # 文件校验、history、锁与 runner
├── internal/httpapi/
│   ├── health.go                      # live/ready JSON 映射
│   ├── router.go                      # 注入 readiness function
│   └── router_test.go                 # 200/503/405/404 契约
├── migrations/
│   ├── embed.go                       # package migrations，embed *.sql
│   └── 000001_create_schema_migrations.sql
└── scripts/
    ├── smoke.sh                       # 无 MySQL 的进程/live/ready-503 smoke
    └── mysql-integration.sh           # 外部隔离 MySQL 8.0 W3 Gate
```

根 `go.mod`/`go.sum` 只增加 MySQL driver；README 只更新当前 DB/readiness、开发启动、migrate 和验证边界。现有 `internal/app.Run` 已能承载任意 handler 且正确处理 server 生命周期，数据库由 `main.go` 在它外层构造/关闭，因此 `internal/app/**` 不需要 ownership。`middleware.go` 也不需要修改。

不通过隐式 `init()` 注册 router 或 migration，因为这会绕过明确装配点、隐藏 owned-path 冲突。根 README、`go.mod` 与 router 装配点和后续 catalog change 串行；不相关 worktree 可继续修改互斥路径。

### D2. 配置只产生结构化内存值，不接受 DSN

新增 `ORDER_ENV`，枚举为 `development|test|production`，默认 `development`。dev/test 数据库输入固定为：

| 环境变量 | 结构化字段 | 校验 |
| --- | --- | --- |
| `ORDER_DB_HOST` | `Host string` | 非空；不得含 scheme、`@` 或 `/` |
| `ORDER_DB_PORT` | `Port uint16` | 十进制 1–65535 |
| `ORDER_DB_NAME` | `Database string` | 非空 |
| `ORDER_DB_USER` | `User string` | 非空 |
| `ORDER_DB_PASSWORD` | `Password string` | 非空，仅 dev/test |
| `ORDER_DB_TLS_MODE` | `TLSMode string` | `required` 或 `disabled` |

`ORDER_DB_DSN` 在所有模式一律拒绝，避免第二配置模型和原始 DSN 传播。production 允许继续读取 host、port、database、user 这类非敏感结构值，但出现 `ORDER_DB_PASSWORD` 或 `ORDER_DB_DSN` 就先返回 `production_secret_environment_forbidden`；两者均不存在时仍返回 `production_secret_source_unavailable`。后续 `load-runtime-secrets-from-ssm` 必须以单独 change 把 SSM 结果组装为同一结构化值，不能在本 change 建通用 provider 或让 production 启动。

配置错误使用稳定 reason；错误的 `Error()`、`slog` 和 tests 只检查字段名/reason。`Password` 不实现 `Stringer`，任何结构体日志都禁止。测试使用 canary secret 并断言全部 stderr/error/body 不包含它。

相较允许 test raw DSN，结构化 test 变量能复用生产参数校验并避免 DSN 出现在命令、进程列表和失败日志；相较现在直接做 SSM loader，它不会越过单能力和外部权限边界。

### D3. driver 参数与 pool 是常量，不再开放配置面

`internal/database` 定义单一 `ConnectionConfig`，`Open(ConnectionConfig) (*sql.DB, error)` 通过 `mysql.NewConfig()` 填充字段，再用 `mysql.NewConnector` 与 `sql.OpenDB` 建惰性 pool。禁止手工字符串拼 DSN。driver 设置固定为：

| 参数 | 冻结值 |
| --- | --- |
| driver | `github.com/go-sql-driver/mysql v1.10.0` |
| network/address | `tcp` + `net.JoinHostPort(host, port)` |
| charset/collation | `utf8mb4` / `utf8mb4_0900_ai_ci` |
| Go time parsing | `ParseTime=true`, `Loc=time.UTC` |
| MySQL session timezone | `time_zone='+00:00'` |
| TLS | `required` → verified TLS；`disabled` 仅 dev/test |
| timeout | dial 3s、read 5s、write 5s |
| unsafe features | multi-statements、interpolate、cleartext/plain fallback、all-files 均 false |
| driver logger | `mysql.NopLogger`；上层只输出枚举错误 |
| pool | max open 10、max idle 10、max lifetime 3m、max idle time 1m |

不开放 pool/timeout/charset 环境变量：当前产品规模和本 change 验收不需要另一组运行选择，后续只有真实容量证据才能改变。`Open` 只完成本地校验与 connector/pool 构造，不 `Ping`；这样网络故障不会让 liveness 消失，也不会把重启变成 schema 操作。

`main.go` 在监听前完成 config 和 pool 构造，构造错误 exit 1；成功后 `defer db.Close()`，把同一个 pool 封装为 readiness function 交给 router，再调用未修改的 `app.Run`。关闭顺序固定为 HTTP 完成 drain、`app.Run` 返回、关闭 pool。`order-migrate` 单独构造/关闭自己的 pool，两个 executable 不共享进程状态。

不选择 ORM，因为本 change 没有 entity/repository；不选择 migration framework，因为命名锁同连接、checksum/dirty/too-new 和日志契约已经非常小且特定，引入框架会增加未使用行为。

### D4. migration 文件是一条语句、不可变、compile-time embedded

`services/api/migrations/embed.go` 只暴露 read-only `fs.FS`。`internal/migrate.Load(fs.FS)` 在任何 DB 访问前验证：

1. 文件只在根层级且匹配 `^[0-9]{6}_[a-z0-9_]+\.sql$`；
2. version 从 1 连续递增，无重复/缺号；
3. UTF-8、无 BOM、LF、非空、以 newline 结束；trim 后恰有一个 `;` 且只能是最后一个非空字符；
4. 禁止 `DELIMITER`、`SOURCE`、`LOAD DATA`、down/seed 命名；每文件只允许一条 SQL statement；
5. SHA-256 对原始 file bytes 计算，name 是完整文件名。

基础集合只有 `000001_create_schema_migrations.sql`，内容只执行 `CREATE TABLE IF NOT EXISTS schema_migrations (...)`，使用 InnoDB 与 `utf8mb4_0900_ai_ci`。表列固定为：

```sql
version BIGINT UNSIGNED PRIMARY KEY,
name VARCHAR(255) NOT NULL,
checksum BINARY(32) NOT NULL,
dirty BOOLEAN NOT NULL,
applied_at TIMESTAMP(6) NULL
```

runner 在 table 不存在时只允许 version 1；执行后核对 `information_schema` 中 engine/column/type/nullability/primary key，再插入 version 1 clean history。`IF NOT EXISTS` 只用于处理“表已创建但 clean history 尚未写入”的首次崩溃窗口；表结构不精确时返回 `database_incompatible`，不得采用未知表。

后续文件也保持一条 statement。这个限制避开通用 SQL splitter、driver multi-statements 和一文件内部分成功；多个 DDL 必须拆成连续 migration files。相较允许多语句，它让每个 dirty version 对应一个可定位的 MySQL statement，并使 failure/recovery 证据可执行。

### D5. 命名锁、history 和 dirty 状态只走一个连接

`internal/migrate` 的核心入口固定为：

- `Load(fs.FS) ([]Migration, error)`：纯文件校验；unit tests 可传 `fstest.MapFS`。
- `Run(context.Context, *sql.DB, []Migration) (Result, error)`：获取连接、加锁、校验并应用。
- `Check(context.Context, *sql.DB, []Migration) State`：只读 ping/version/session/history，用于 readiness。

`Run` 从 `db.Conn(ctx)` 取得专用 `*sql.Conn`，先执行 `SELECT GET_LOCK('order_schema_migrate', 30)`。取得锁后所有 `SELECT VERSION()`、session/schema 查询、migration SQL、history insert/update 和 `RELEASE_LOCK` 都只能调用这个 `*sql.Conn`。defer 顺序是尝试 release，再 close connection；release 失败仍返回失败，close 保证连接级 lock 不回到 pool。lock timeout 不开放配置。

执行状态机固定为：

1. 验证 server version 前缀为 `8.0.`，`@@session.time_zone='+00:00'` 且 connection charset 为 `utf8mb4`。
2. 读取 history；任一 dirty 立即 `schema_dirty`，不执行 SQL。
3. 对每条 clean history 验证连续 version、name、checksum；未知/更高/缺历史为 `schema_too_new`，内容漂移为 `schema_checksum_mismatch`。
4. history 是 embedded 严格前缀时按序 forward。
5. version 1 按 D4 的首次规则创建/核表并插入 clean。
6. version 2+ 先插入 dirty row，再执行唯一 statement；成功才原行更新为 clean 与 `CURRENT_TIMESTAMP(6)`，失败保留 dirty 并停止。
7. 完全一致时返回 `applied_count=0`，不写 `applied_at`。

MySQL DDL 不被伪装为跨文件事务；checksum 和 dirty 是故障可见边界。失败 version 不算 clean“已应用”，但保留 dirty 记录防止静默重跑。runner 不提供自动清 dirty、down、force 或 repair。

并发测试使用一个临时 migration statement 创建 probe table，并把 `CONNECTION_ID()` 和 `IS_USED_LOCK('order_schema_migrate')` 写入；断言相等。两个 runner 同时运行时，一个执行副作用，另一个获得锁后观察 current 并零写入；若真实持锁超过 30 秒，等待者按契约失败。

### D6. `order-migrate` 只接受零参数和三个退出类别

`cmd/order-migrate/main.go` 保持薄入口：构造 JSON `slog` → 拒绝非零参数 → `config.Load` → `database.Open` → `migrate.Load(migrations.FS)` → `migrate.Run` → close。退出码冻结为：

- `0`：所有 pending version 成功，或已经 current；
- `1`：configuration、connector、unreachable、incompatible、lock、dirty、too-new、checksum 或 statement failure；
- `2`：出现任意参数/flag，打印 `usage: order-migrate` 后不连接 DB。

成功日志只有 `event/from_version/to_version/applied_count/duration_ms`；失败日志只有 `event/reason/version/duration_ms`。底层 MySQL error 只用于进程内分类/包裹，不能进入 `slog`、stderr 或返回 body。没有 `up/down/status/force/repair/seed` 子命令，避免新的数据写入口。

### D7. health 接收一个窄 readiness function，不反向依赖 migration CLI

`httpapi.NewRouter` 改为显式接收 `ReadinessFunc func(context.Context) ReadinessResult`；`ReadinessResult` 只包含 `Ready bool` 与 spec 枚举 reason。`main.go` 的 closure 为每次 request 创建 2 秒 context，并调用 `migrate.Check` 读取真实 pool 与 embedded migrations。HTTP 包不接收 password/config，也不构造 DB。

`health.go` 保留独立 liveness handler，不触碰 readiness function。ready 映射固定为：

| 状态 | HTTP/body |
| --- | --- |
| current | 200 `{"status":"ok"}` |
| ping/context/connection failure | 503 `{"status":"not_ready","reason":"database_unreachable"}` |
| 非 MySQL 8.0 或 session/schema table shape 错误 | 503 `database_incompatible` |
| table 不存在 | 503 `schema_uninitialized` |
| dirty | 503 `schema_dirty` |
| clean strict prefix | 503 `schema_behind` |
| 未知/更高/缺失历史 | 503 `schema_too_new` |
| name/checksum drift | 503 `schema_checksum_mismatch` |

每次 ready 都实时检查，无 success cache；数据库恢复或显式 migrate 后下一请求即可 200。health 不返回 `expected/current_version`，减少内部 schema 暴露。现有 method 405、unknown 404、request ID、访问日志、recovery 和 app lifecycle保持不变。

`scripts/smoke.sh` 使用 development 结构化配置指向本地确定不可达端口，不启动数据库；它验证 live 200、ready 503/database_unreachable、404、SIGTERM clean exit 及无效配置 exit 1。这个 smoke 只证明进程边界，不算 W3 DB PASS。

### D8. apply writer 建立专属 Colima/MySQL 8.0，再运行真实 Gate

DRAFT 环境状态固定为 `NOT_ESTABLISHED`、真实 W3 为 `NOT_RUN`。apply 获批后，任何 Red 前先执行环境任务，owner 就是当前 writer，不等待客户或平台：

1. 安装或核验 Homebrew stable Colima v0.10.3，arm64 bottle SHA-256 固定为 `a9dfd1fa0a4aee62fef75974f39f174e4da774f7ba495c43dd0bcc23633381b8`；不复用默认或其他项目 profile。
2. 创建 profile `order-mysql-w3`：`--runtime docker --arch aarch64 --vm-type vz --cpus 2 --memory 4 --disk 20 --kubernetes=false --mount none`，不启用外部 network address。
3. 通过 profile 内 Docker pull `docker.io/library/mysql@sha256:0e7040b532c0f2ac8cb822695d33025522acd5252175cb104a5929aa66b40222`；该 digest 是 Docker Official Image `mysql:8.0.45-oraclelinux9` 的 `linux/arm64` manifest。核对 repo digest 与 architecture，禁止 tag-only run。
4. 在权限 0600 的 `mktemp -d` 中生成随机 root/test password env file；容器名固定 `order-mysql-w3`，只用 `127.0.0.1` 随机 host port 映射 3306，不挂宿主目录或复用 volume。
5. 等待容器 health，并通过结构化 Go connector 执行 `SELECT VERSION()` 确认为 `8.0.x`；只把 host/port、脱敏 profile 与临时 env 文件路径交给当前 shell，不输出 secret。

profile/container preflight 任一步失败都记 FAIL，由 writer按首个真实错误最小修复；不是 `BLOCKED_EXTERNAL`，未成功前不进入 Red。生产 SSM/CAM/云账号依然是独立 change 的外部门禁，但与这个本地 W3 环境无关。

`scripts/mysql-integration.sh` 只做安全前置和运行既有 Go tests：

1. 要求 `ORDER_TEST_MYSQL_HOST/PORT/USER/PASSWORD/TLS_MODE/INSTANCE`、`ORDER_TEST_MYSQL_INSTANCE=order-mysql-w3` 与 `ORDER_TEST_MYSQL_ISOLATED=YES`；不打印值。
2. apply 前缺失表示 `NOT_ESTABLISHED/NOT_RUN`；apply 中缺失或不匹配则 FAIL，不启动替代 runtime 或请求用户提供数据库。
3. 用 `GOPROXY=off GOTOOLCHAIN=go1.26.5 go test -race` 运行锁定的 `TestMySQL8Integration`。

integration test 通过结构化 connector 先连接系统 `mysql` schema，执行 `SELECT VERSION()` 并只接受 `8.0.`；然后生成 `order_test_<128-bit-random-hex>`，只对这个精确名称执行 `CREATE DATABASE`/`DROP DATABASE`。测试账号只存在于专属本地 profile并拥有创建/删除测试 schema 权限。cleanup 在 `defer`/`t.Cleanup` 中执行；名称解析、前缀或本次创建记录不匹配时停止，绝不扩大删除。

同一真实实例矩阵固定为：

| 阶段 | Red | Green / Refactor 复验 |
| --- | --- | --- |
| 首次/重复 | table 不存在，ready=uninitialized | 首次 version 1 clean；重复零写入 |
| 并发/同连接 | 两 runner 争用同一 DB | 一次副作用；lock ID=connection ID；最终一条 clean history |
| statement 失败/dirty | version 2 使用确定非法 SQL | 无 clean v2、dirty 保留、后续拒绝且无 v3 |
| behind | clean history 为 embedded 严格前缀 | ready=behind，runner 补齐后 current |
| unreachable | 使用已创建后关闭的 loopback listener 地址 | 2 秒内 ready=database_unreachable |
| too-new | 测试事务插入 current+1 clean row | ready/migrate=too-new 且零写入 |
| 应用不自动迁移 | 空 schema 启动真实 `order-api` binary | live 200、ready uninitialized、表仍不存在 |
| 迁移后 ready | 同一运行 API 外部执行真实 `order-migrate` binary | 不重启 API，下一 ready=200 |

fake/mock 只允许测试 `Load` 文件错误、health 映射和内部 error classification。没有锁摘要真实 MySQL 8.0 全矩阵与随机 schema cleanup PASS，writer 不得形成 candidate，C/T/V/R 分数不能绕过。

writer candidate PASS 后保留专属 profile，但 verifier 必须删除旧容器并从同一锁定 digest 建全新容器再测 exact SHA，避免复用 writer 数据。final independent PASS 后 agent 只删除 `order-mysql-w3` 容器和 profile data disk并确认 profile 不存在；不得触碰默认或其他 Colima profile。任一清理失败即停止并报告精确目标，不用更强或更宽删除命令。

### D9. forward fix、二进制回退和 dirty 修复边界

成功 migration 后不自动 down。若新 binary 有问题，只有上一 binary 声明兼容当前 schema 时才能回退；否则先提交更高编号 forward fix。破坏性 schema 和云备份策略属于未来业务/部署 change，本 change 不把 HA、备份、RPO/RTO 当验收。

dirty 表示真实结果未知，自动 runner/readiness持续阻断。恢复顺序固定为：

1. 停止继续迁移并保留 dirty 证据；
2. 在隔离副本复现并核对实际 schema、原文件 checksum 和失败前后状态；
3. 取得针对该数据库的单独写授权与已验证恢复点；
4. 用 review 过的窄 SQL 修到原 version 的预期最终 schema；
5. 人工核对后才能窄范围更新对应 dirty row，再以新的更高 version forward fix 后续变化；
6. 从 failure 点重跑完整 Gate。

当前 CLI 不实现第 4–5 步的自动命令，避免未经授权清 dirty。测试 cleanup 只 drop 本次 `order_test_` schema；失败时若安全条件不满足，保留 FAIL 并由当前 writer 处理，不转交客户/平台、不重试更强删除。

### D10. W3/UI0 Gate、exact SHA 与验证失效

本 change 的唯一最高风险是 `W3`，UI 为 `UI0`。DRAFT 只可获得规划 strict/结构/owned PASS，不得获得实现 C/T/V/R verdict。批准后必须按 tasks 先形成可观察 Red，再最小 Green，同一真实 MySQL 矩阵 Refactor 重跑。

writer candidate 必须同时通过：focused unit、真实 MySQL script、gofmt、全 API test/race/vet/build、无 DB smoke、strict、diff/owned/protected、README 事实、依赖锁定及敏感扫描。候选目标评分为 `C=10、T=10、V=8、R=8`，总分 36；V=8 只表示完整 exact-SHA verifier 包待执行，不是 independent PASS。

verifier 只接收已提交完整 SHA，在另一个 clean detached worktree 重跑全部 Gate并确认结尾 clean。以下任一变化立即使旧验证失效：Go code、SQL bytes/name/number、driver/go.sum、proposal/design/spec/tasks、README/smoke、验收命令、base、依赖、rebase/merge 或 candidate SHA。

## Risks / Trade-offs

- [本机尚无 Colima/MySQL 8.0] → DRAFT 记 `NOT_ESTABLISHED/NOT_RUN`；apply writer 在 Red 前自行建立专属 profile并锁 digest，candidate 前真实 W3 PASS，mock 不替代。
- [production 当前必然不能启动] → 明确依赖单能力 `load-runtime-secrets-from-ssm`；拒绝临时环境密码，避免把过渡路径变成生产契约。
- [一条 statement 一文件会增加 migration 文件数] → 换取无 SQL splitter、无 multi-statements、失败 version 精确和更小恢复面。
- [MySQL DDL 不能跨文件事务回滚] → checksum、dirty、命名锁、单 statement 和 forward-fix 让失败显式；不宣称原子跨版本。
- [固定 pool 可能不是最终容量最优] → 当前只有基础连接；只有后续真实容量 Gate 失败才独立调整，不预留配置。
- [readiness 每次查 history 增加少量 DB 读] → health 只查小表与 server/session 元数据，换取不缓存过期成功；容量变化另立 change。
- [TLS required 可能需要未来 CDB 证书准备] → dev/test 可显式 disabled；production 仍由 SSM/部署 change 提供可验证 TLS material，不能 skip-verify。
- [手工 dirty 修复需要外部授权] → 这是保护真实数据的刻意边界；当前 CLI 不提供可误触的 force/repair。

## Migration Plan

1. 本轮只提交 DRAFT proposal、一份新 capability spec、一份 MODIFIED delta spec、design 和全未勾选 tasks；不修改 Go、SQL、README、依赖或外部系统。
2. 获得明确批准后，在同一 writer worktree 重新读取四类 artifacts、质量门禁和 exact main，确认 owned paths 无并行 writer并进入 `APPROVED/IMPLEMENTING`。
3. 在任何 Red 前由 writer 安装/核验锁定 bottle 的 Colima v0.10.3，建立 `order-mysql-w3`，拉取并验证冻结的 `linux/arm64` MySQL digest、loopback、随机临时凭据和 cleanup 边界；失败由 writer闭环，不能转给客户/平台。
4. 新增失败 tests/fixtures：配置/脱敏、文件校验、schema 状态、health 503、CLI，并在专属真实 MySQL 8.0 记录首次/并发/dirty/too-new/无自动迁移的 Red。
5. 最小实现 config → database pool → embedded version 1 → migrate runner/CLI → readiness 装配；不改 `internal/app/**` 或业务路径。
6. 重跑同一 Red 变 Green，更新 smoke/README，再做 Refactor；真实 MySQL integration 必须创建并清理本次随机 schema。
7. 完成全 writer Gate，只提交 owned paths形成 CANDIDATE；另一个 clean detached worktree 删除旧容器、从同一 digest 建空容器后对精确 SHA 全量复验。
8. independent PASS 后 agent 删除本 change 的容器、profile 和 data disk并确认不存在；本 change `INTEGRATED` 后，`serve-persistent-menu-catalog` 才能基于当前 schema/pool 串行修改共享装配点。

应用 rollback 只回退到支持当前 schema 的 binary；schema rollback 只走更高编号 forward fix 或另行授权的已验证恢复，不运行 down。测试 rollback 只删除本次随机 `order_test_` schema。

## Open Questions

无。生产 SSM/CAM、云部署/备份以及商品目录 schema 均已明确拆为其他单能力 change，不是本实现 writer 的待选项。
