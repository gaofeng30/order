## MODIFIED Requirements

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
