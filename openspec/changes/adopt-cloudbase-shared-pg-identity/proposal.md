# 共享 PG 身份、商户账号与 PC 会话闭合

Status: IMPLEMENTED / SCOPED_UAT_VERIFIED（候选及独立验证待完成）。gate_type=W3；ui_level_target=UI0；ui_level_actual=NOT_RUN。

## Why

现有共享履约与订单查询仍缺少真实 Order 会话和商户身份写端。本切片迁移既有身份业务端口，保留微信 provider、Order bearer 和公共接口，使后续完整 runtime 能使用真实身份适配器；不将平台服务 JWT 当成业务用户。

## What Changes

- 实现 SessionStore 三方法、PhoneStore 两方法、merchant Store 五方法（复用已实现 ReadIdentity）。
- 实现 AdminApplication 八方法，包括账号管理、PC intent/批准/一次消费、当前权限认证与退出。
- 独立管理面一次性 bootstrap，保持原三组精确初始状态及 RepeatableRead 合同；持续 runtime 不获得该入口或配置写权限。
- 扩展固定 INVOKER RPC registry、普通服务 UID RLS、最小列/sequence ACL 与精确 readiness；保留原44/schema/ledger和已交付命令。

## Ownership and Dependencies

base_sha: e4657b11eadc2c87a0d6606038ff3c97d49b9b00；branch: codex/order-uat-shared-pg-identity；worktree: /Users/vivix/.codex/worktrees/order-uat-shared-pg-identity/order；candidate_sha由提交后外部receipt记录。

需求owner仅写本change proposal/design/tasks/spec四工件。root独占 sharedpg/schema.go、access.sql、schema_test.go、cloudbasepg/client.go/client_test.go及redeem_order.sql内readiness事实投影（首两核销业务body不改）固定registry/ACL/readiness及受控云操作；新增owned services/api/internal/fulfillment/sharedpg_test.go仅修本次ACL变更直接相关的权限故障fixture，不改既有核销业务body。业务writer独占新增 internal/identity/sharedpg*.go、internal/merchantidentity/sharedpg_store*.go/sharedpg_admin*.go/sharedpg_pc*.go及对应测试、sharedpg/identity_commands.sql/merchant_admin.sql/pc_session.sql；原生文件仅必要投影/校验复用需先与root划清具体路径。同一业务writer独占管理bootstrap新增 cmd/order-bootstrap/sharedpg*.go及测试、固定管理SQL；该SQL不得注册为runtime RPC。禁止改第二切片冻结四工件、原44及客户端。

依赖 expand-cloudbase-shared-pg-fulfillment exact e4657b11eadc2c87a0d6606038ff3c97d49b9b00：干净detached独验已PASS；首二切片均未main集成，开发可继续，集成依赖不自动满足。

## Minimum Success

真实v44 native/shared差分及原handler链路通过；当前授权、字节/hash、时钟、并发、拒绝审计提交、未知提交和PC熵优先级均保留。普通JWT真UAT矩阵与管理bootstrap隔离/负例及清理通过；完成Writer Gate、候选exactSHA、干净detached独验后才可称本切片通过。

## External Assets and Non-goals

只使用已授权Order UAT。真实微信AppSecret/手机号code/客户owner资料未齐，只能以provider spy证明调用协议，不能称微信认证已联调；合成bootstrap输入不得替代客户资料。无AppID变更、默认设置/角色/DTO/token变更、采购、Prod、push/PR/merge授权。完整runtime wiring、其他业务DB端口、5持续worker、私有存储30天恢复、CloudRun/static部署、UI3及95Case/25页面仍后续。无泛SQL、SECDEF、管理员credential或bootstrap权限进入持续runtime。

## 当前交付层次

本地必需八包ordinary/race分批各897测试项通过；普通服务JWT真UAT的身份业务、16并发、未知提交、回滚、管理RR bootstrap与清理已完成，证据见tasks。当前30 RPC/12 RLS表/5seq通过最终普通身份readiness；44 ledger保持。提交前源与四工件核验、候选exactSHA及clean detached独验分别完成后才进入相应状态。真实微信和完整runtime/UI验收不继承切片通过。
