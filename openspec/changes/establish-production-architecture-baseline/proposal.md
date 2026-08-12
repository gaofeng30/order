## Why

一期产品行为和 Go API 进程边界已经归档，但生产承载、数据一致性、迁移、环境隔离、密钥、图片、观测和恢复仍散落在技术文档与客户指南中，并保留数据库二选一、公有读 COS 和长期访问密钥等冲突口径。后续数据库、支付、部署和运维 changes 需要先共同依赖一套不再要求 writer 选型、且能被静态验收的唯一生产架构基线。

## What Changes

- 把一期生产形态冻结为单台境内 CVM 上的 Nginx、systemd 与 Go/Gin 模块化单体，`order-api` 和同进程 worker 共享一个部署单元；数据、图片、密钥和观测分别固定使用 TencentDB MySQL 8.0 双节点多可用区、私有 COS、SSM/CAM、CLS/云监控/CAT。
- 固定同步事务只写 MySQL、外部调用不进入事务；微信回调先写唯一 inbox 再于 5 秒内应答，业务事务产生 outbox，同进程 worker 使用 `SKIP LOCKED`、lease、at-least-once 和幂等消费，连续失败 10 次进入 `DEAD` 并告警。
- 固定 `order-migrate` 的 forward-only SQL、`schema_migrations`、`GET_LOCK`、部署前迁移、数据库与 schema readiness，以及生产不自动执行 down migration的边界。
- 固定 dev、UAT、prod 的资源与身份隔离、配置/密钥加载、私有 COS 预签名 PUT/GET、备份恢复目标、日志指标告警、容量门和唯一升级顺序。
- 在后续 apply 中同步改写技术文档和腾讯云客户指南，明确移除 MySQL/TDSQL-C 二选一、公有读 COS、长期 SecretId/SecretKey 和 CDN 默认启用等第二事实源。
- 只允许真实云账号、地域/VPC/SKU、域名证书、桶名、AppID/商户号、告警接收人和月预算等外部值使用命名外部占位符；RPO/RTO 与容量在实测前只能表述为目标。

## Capabilities

### New Capabilities

- `production-architecture-baseline`: 定义一期生产系统唯一的运行、交易一致性、迁移、隔离、安全、对象存储、观测、容量和恢复基线，供后续实现 changes 只读依赖。

### Modified Capabilities

无。

## Impact

- 状态：`DRAFT`；本轮只创建规划 artifacts，不修改实际技术文档，不执行 tasks，不进入实现。
- `base_sha`：`cb2605f477e58ac5471a0c535b85256c6be80a00`。
- `gate_type`：`W0`，因为 apply 只改变文档契约、链接和内容完整性，不改变运行行为、公共 API 或数据结果。
- `ui_level_target`：`UI0`；本 change 没有用户界面或真实平台运行结果，UI1–UI3 不适用。
- owner：`Production Architecture Writer`；唯一 writer 位于既有 worktree `/Users/vivix/.codex/worktrees/9d75/order` 的 branch `codex/establish-production-architecture-baseline`。
- owned paths：
  - `openspec/changes/establish-production-architecture-baseline/**`
  - `docs/product/online-ordering-system-technical.md`（批准 apply 前只读）
  - `docs/微信小程序开发和运维指南/腾讯云操作指南.md`（批准 apply 前只读）
- 只读共享契约：根 `AGENTS.md`、`docs/quality/change-quality-gates.md`、`openspec/specs/mvp-product-baseline/spec.md`、`openspec/changes/bootstrap-api-service/**`、产品 PRD、客户清单、合同、质量与 loop skills、全部业务代码。
- 依赖：`bootstrap-api-service` 已集成；`formalize-mvp-product-baseline`、`establish-loop-engineering-control-plane`、`enforce-change-quality-gates` 均已归档并进入当前 `main@cb2605f477e58ac5471a0c535b85256c6be80a00`。
- 必要外部资产：candidate 必须能通过 public internet 实际访问文档中的腾讯云/微信支付官方直链，owner 为 `Production Architecture Writer`；规划时该资产可用，candidate 时不可用则记 `BLOCKED_EXTERNAL`，恢复条件为官方页面或网络恢复后重跑链接 Gate。真实云账号、域名、微信身份、数据库和监控权限不是本 W0 change 的必要资产，由客户云管理员、客户平台管理员和后续平台 owner 在对应部署/UAT change 提供。
- 非目标：不实现库存、数据库、支付、worker、迁移、部署或监控；不购买、配置或写入腾讯云/微信；不修改 PRD、客户清单、合同、质量/loop skills 或业务代码；不引入 Kubernetes、微服务、Redis、MQ、读写分离、数据库代理、CLB、CDN、跨地域灾备或 7×24 人工值守。
- 已确认产品约束：库存唯一键为`营业日期 × 餐段 × 商品`，提交订单原子创建 15 分钟软预占；本 change 只引用该归档产品事实，不重新裁决或实现库存。
- 最小成功标准：三份目标文档对唯一组件、同步/异步边界、迁移、隔离、安全、COS、备份恢复、RPO/RTO 目标、观测、容量升级和非目标完全一致；白名单外无占位符或选型歧义；链接、敏感信息、owned paths、strict 与当前仓库 Gate 全部通过。
