# 共享 PG 上的 Go 原子核销命令

Status: APPROVED / IMPLEMENTING。用户已授权整体CloudBase UAT改造，并指定需求owner与开发owner持续协作；本change仅交付首个可独立验收的共享PG核销命令能力，不启用部分后端上线。

## Why

现有共享PG可经HTTP参数SQL访问，但逐请求调用不能保留Go原生事务与session锁。消融已证明客户端换Node不能恢复原子性；需要在数据库的一次RPC中完成完整业务命令，同时保留Order现有公共契约。当前数据库原生网络条件不具备，用户明确优先既有CloudBase且不采购。

## What Changes

- 用Go HTTP适配器携专用普通服务用户JWT，经REST RPC调用受限proof读取和单次原子RPC，实现现有fulfillment.CommandRedeemOrder；Go保持token解密、公共DTO和错误语义，RPC保持当前商户授权、订单锁、核销状态与审计回执同提交。
- 建立受控管理面安装、原44迁移ledger/checksum核验与RPC定义指纹；运行面不持DDL管理能力凭据，不模拟逐查询Tx；APIKey+exec-pgsql已实测违反硬门，改为固定服务UID的INVOKER/RLS并重验。
- 以真实UAT平台primitive门、真实schema核销并发/故障/重放门区分能力探针与命令验收；本地模型不替代真实PG17 HTTP/RPC。

## Ownership and Gate

- gate_type: W3；ui_level_target: UI0；ui_level_actual: NOT_RUN。无UI改动；真实数据库平台证据单独记录，不声称Mini UI3。
- requirement owner: `/root/astra_cloudbase_ablation`，独占本change目录；implementation/deployment owner: `/root`，独占`services/api/**`（包括`internal/cloudbasepg`、`internal/fulfillment`、`internal/migrate`、`sharedpg`）及`cloudbase/migrations/**`中本change新增或修改文件。任何公共路径写入须事先由owner协调。
- delegated writer ownership：root协调实现路径；`/root/uat_release_writer`独占`services/api/internal/cloudbasepg/**`及已分配installer文件；`/root/runbook_platform_limits`独占新增`services/api/internal/migrate/sharedpg_readiness.go`、`services/api/internal/migrate/sharedpg_readiness_test.go`。root保留`services/api/internal/fulfillment/sharedpg*`、`services/api/sharedpg/**`及管理迁移部署；需求owner只读业务源码，仍只写本change工件。
- branch: `codex/order-uat-shared-pg-runtime`；worktree: `/Users/vivix/.codex/worktrees/order-uat-shared-pg/order`。
- base_sha: `12164b115441cb2a50c96d05b3173f874042568e`；candidate_sha: NOT_CREATED（完整SHA提交后保存在仓库外receipt）。
- read-only contracts: 根AGENTS、CONTEXT、质量门禁、现有fulfillment/audit/merchantidentity/migrate、全部44权威PG迁移、PC/Mini公共API。
- dependencies: base中的PG/UAT实现与原生CloudRun配置adapter；尚未集成的依赖不自动满足integration。Mini发布writer独立路径，禁止本change改写。
- non-goals: 全部fulfillment命令、支付/退款等其余runtime改造、部分后端上线、语言重写、采购、Prod、实际资金、push/PR/merge/archive；不新增库存数量模型（本系统是日期售罄）。

## Minimum Success

保留原核销/审计语义，真实UAT角色隔离、参数/SQLSTATE/类型、回滚、16并发、提交后响应丢失幂等，以及受控安装互斥、44ledger/checksum与RPC指纹全部具备实际证据。候选必须完成Writer Gate，独立验证必须另一个干净detached worktree精确SHA。

## External Assets

root拥有UAT管理与测试身份控制；APIKey+exec-pgsql DDL越权已实测FAILED，最终普通专用UID 2108167831809454081的JWT+REST RPC/RLS已真实验证必要DML、DDL拒绝、其他UID/anon隔离及撤销旧JWT授权后恢复；原APIKey证据不继承。当前仅合成核销切片通过，合成资源与临时身份已清理并复读，最终Writer Gate收尾、候选SHA绑定和独立验证尚待闭合。私有存储最小权限适配列后续独立change，保持上传/私有访问/恢复合同，当前维持COS版本控制目标，pgstore只有平台能力等价实证后才可替代，现有broad APIKey不得进入完整runtime；该依赖未闭合前不得部署完整后端。AppSecret、merchant JSON、template、ownerphone、真实Mini UI3与浏览器mp站点安全拒绝不阻断本change开发，但后续95Case/25页面完整UAT仍未通过。本change不处理外部真实资金。
