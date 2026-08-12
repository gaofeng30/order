# api-service-bootstrap Specification

## Purpose

当前仓库只有两个基于内存 mock 的前端原型，没有可运行的服务端工程；订单、支付、商品等后续 change 因而缺少共同且可测试的承载边界。先建立一个不含业务逻辑的 Go API 进程基线，后续垂直能力即可在互斥 owned paths 中独立实现，而不重复搭建启动、健康检查和进程生命周期。
## Requirements
### Requirement: API service has one deterministic build entry

系统 MUST 在根 Go module 中提供唯一 `order-api` 可执行入口，并保持 bootstrap 不包含任何业务模块或外部系统客户端。

#### Scenario: Service packages build from repository root

- **WHEN** writer 在仓库根执行 `go build ./services/api/...`
- **THEN** 所有 API service packages 编译成功
- **AND** module 声明 Go 1.26 language version 与 `toolchain go1.26.5`
- **AND** 构建不要求数据库、微信、云账号或其他外部服务

#### Scenario: Bootstrap scope is inspected

- **WHEN** reviewer 检查 `go.mod`、`go.sum` 和 `services/api/**`
- **THEN** 唯一直接应用框架依赖是锁定版本的 Gin
- **AND** 不存在 ORM、Redis、消息队列、支付 SDK、业务 entity、repository 或业务 endpoint

### Requirement: Startup configuration is explicit and fail-fast

系统 MUST 在监听端口前加载并校验 `ORDER_API_HTTP_ADDR` 与 `ORDER_API_SHUTDOWN_TIMEOUT`，并为 HTTP Server 设置非零安全 timeout。

#### Scenario: Service starts with defaults

- **WHEN** 两个环境变量均未设置
- **THEN** 服务监听 `:8080`
- **AND** 优雅退出上限为 `10s`

#### Scenario: Valid overrides are provided

- **WHEN** `ORDER_API_HTTP_ADDR` 是可监听地址且 `ORDER_API_SHUTDOWN_TIMEOUT` 是大于 0 的 Go duration
- **THEN** 服务使用这两个覆盖值启动

#### Scenario: Shutdown timeout is invalid

- **WHEN** `ORDER_API_SHUTDOWN_TIMEOUT` 无法解析或小于等于 0
- **THEN** 服务在监听端口前返回明确配置错误并以非零状态退出

#### Scenario: Listen address cannot be bound

- **WHEN** 配置地址无效或端口已被占用
- **THEN** 服务返回监听错误并以非零状态退出

### Requirement: Health contract cannot mask routing failures

系统 MUST 提供两个只读 JSON 健康端点。`GET /health/live` MUST 只表达当前 HTTP 进程存活，并始终返回 HTTP 200、`application/json` 和 `{ "status": "ok" }`。`GET /health/ready` MUST 在独立 2 秒 context 内验证数据库可达、目标为 MySQL `8.0.x`、session 为 UTC/`utf8mb4`，且 `schema_migrations` 与当前 embedded migration 集合编号、name、checksum、dirty 状态完全兼容；它 MUST NOT 创建连接配置、执行 migration、修改 schema 或清理 dirty history。

ready 成功 MUST 返回 HTTP 200、`application/json` 和 `{ "status": "ok" }`。失败 MUST 返回 HTTP 503、`application/json` 和 `{ "status": "not_ready", "reason": "<enum>" }`，其中 reason MUST 且只能是 `database_unreachable`、`database_incompatible`、`schema_uninitialized`、`schema_dirty`、`schema_behind`、`schema_too_new` 或 `schema_checksum_mismatch`；响应与日志不得包含连接字段、凭据、DSN、SQL 或数据库错误正文。

非 GET 方法对已知健康路径 MUST 返回 405，未知路径 MUST 返回 404。database 不可达或 schema 不兼容 MUST NOT 改变 liveness，避免把依赖故障误判为进程死亡；数据库恢复或显式迁移完成后，下一次 readiness 请求 MUST 根据实时状态恢复 200，无缓存成功状态。

#### Scenario: Liveness is requested
- **WHEN** 客户端发送 `GET /health/live`
- **THEN** 响应状态为 200、内容类型为 `application/json`
- **AND** JSON body 等于 `{ "status": "ok" }`，且不访问数据库

#### Scenario: Readiness is requested for a current schema
- **WHEN** 真实 MySQL 8.0 可达且 clean migration history 与当前 embedded 集合完全一致
- **THEN** 响应状态为 200、内容类型为 `application/json`
- **AND** JSON body 等于 `{ "status": "ok" }`

#### Scenario: Database is unreachable
- **WHEN** readiness 在 2 秒内不能完成数据库 ping
- **THEN** 响应状态为 503 且 JSON body 等于 `{ "status": "not_ready", "reason": "database_unreachable" }`
- **AND** 同一进程的 liveness 仍返回 200

#### Scenario: Schema is not ready
- **WHEN** schema table 不存在、存在 dirty、behind、too-new 或 checksum drift
- **THEN** readiness 返回 503 与对应的稳定 reason
- **AND** 不执行 migration、down、repair 或任何 schema 写入

#### Scenario: Database shape is incompatible
- **WHEN** 可达目标不是 MySQL `8.0.x`，或 session timezone/charset 不符合冻结配置
- **THEN** readiness 返回 503 与 `database_incompatible`
- **AND** 不暴露 server version/comment 或连接信息

#### Scenario: Health path uses a disallowed method
- **WHEN** 客户端对任一健康路径发送非 GET 请求
- **THEN** 响应状态为 405
- **AND** 响应不得伪装为健康成功 body

#### Scenario: Unknown path is requested
- **WHEN** 客户端请求未注册路径
- **THEN** 响应状态为 404

### Requirement: Every request is safely observable

系统 MUST 为每个请求生成服务端 request ID、输出结构化访问日志并从 panic 中恢复，同时不得记录敏感请求内容。

#### Scenario: Normal request completes

- **WHEN** 任意请求完成
- **THEN** 响应包含非空 `X-Request-ID`
- **AND** JSON 访问日志包含同一 request ID、method、path、status 和 duration
- **AND** 日志不包含 raw query、body、authorization、cookie 或其他 header 值

#### Scenario: Handler panics

- **WHEN** 测试 handler 在处理请求时 panic
- **THEN** middleware 返回 500 且进程保持可服务
- **AND** 错误日志可由同一 request ID 关联但响应不暴露 panic 或 stack

### Requirement: Process shuts down within a bounded lifecycle

系统 MUST 响应 `SIGINT` 与 `SIGTERM`，停止接收新请求并在配置的 shutdown timeout 内等待在途请求。

#### Scenario: Termination signal arrives with no blocked request

- **WHEN** 运行中的 `order-api` 收到 `SIGTERM`
- **THEN** 服务在 shutdown timeout 内正常退出
- **AND** smoke 验收观察到退出状态为 0

#### Scenario: In-flight request finishes before timeout

- **WHEN** 服务停止时存在能在 shutdown timeout 内完成的请求
- **THEN** 服务等待该请求完成后正常退出

#### Scenario: Graceful shutdown exceeds its timeout

- **WHEN** 在途请求或 shutdown 操作超过配置上限
- **THEN** 服务返回可观察错误并以非零状态退出

### Requirement: Candidate has repeatable local and independent gates

候选实现 MUST 在 writer worktree 完成格式、测试、race、vet、build、真实进程 smoke、严格 OpenSpec 和 owned-path 验收，并由另一干净 worktree 对精确 SHA 重跑。

#### Scenario: Writer produces a candidate

- **WHEN** writer 准备提交候选
- **THEN** `gofmt` check、`go test ./services/api/...`、`go test -race ./services/api/...`、`go vet ./services/api/...`、`go build ./services/api/...` 和 `bash services/api/scripts/smoke.sh` 全部通过
- **AND** `openspec validate bootstrap-api-service --strict` 与 owned-path diff 检查通过

#### Scenario: Exact candidate is independently verified

- **WHEN** verifier 在干净 detached worktree 检出候选完整 SHA
- **THEN** verifier 重跑全部验收命令并确认 worktree 仍然 clean
- **AND** 只有精确 SHA 获得 PASS，任何后续代码、spec、tasks、rebase 或 merge 都使结果失效
