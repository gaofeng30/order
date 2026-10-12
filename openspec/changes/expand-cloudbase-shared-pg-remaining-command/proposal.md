# 共享 PG 退款、订阅与订单推进

Status: APPROVED_ATOMIC / IMPLEMENTED / LOCAL_VERIFIED / UAT_SYNTHETIC_VERIFIED / CANDIDATE_PENDING / INDEPENDENT_NOT_RUN。gate_type=W3；ui_level_target=UI0；ui_level_actual=NOT_RUN。

## Why

退款成功必须与订单/预付款、观测以及可选订阅 consent/outbox 在原事务中提交；订单推进与退款、核销竞争同一状态。完整迁移这三个现有模块，才能继续完整运行接线，不能只接退款意向就宣称后端完成。

## What Changes

保留退款五方法、订阅两方法、订单推进一个方法及原 HTTP/provider/幂等合同；用20个固定普通服务 UID INVOKER RPC，父70→目标90。Go保留外部provider与原重试编排；每个RPC完整承担原单次事务，不提供SQL或Tx回调。既有44 schema/ledger、所有native接口不变。

## Dependencies and Lifecycle

base_sha: `dde945e3fd1eb8114935cfc40a46606ab6c39c9d`。branch: `codex/order-uat-shared-pg-remaining-command`。worktree: `/Users/vivix/.codex/worktrees/order-uat-shared-pg-remaining-command/order`。
父第六已INDEPENDENT_VERIFIED，实际ordinary/race各17包1944 PASS/0FAIL/SKIP、Spec/Standards零mustfix；external `shared-pg-payment-independent/verification-receipt.json` SHA37253e64e2257e542971c8920a19ae08126dd89962d27c0089a85856775f4b90。父未集成main；本change开发可进行，集成依赖仍须核验。
本切片本地20包ordinary/race各2437 PASS、0 FAIL/SKIP；普通UAT已安装90固定RPC，并完成三模块合成provider业务与权限矩阵。实现545个Go/SQL文件绑定source-freeze-v3，完整证据见tasks。候选形成后exact SHA只记外部lifecycle receipt，避免同提交自引用；干净detached独验完成前不集成，未集成不archive。

## Ownership

Astra冻结需求/spec合同；各leaf源码已移交，root接管四工件的实际完成记录，不改变批准合同。root为运行编排/common与唯一UAT writer。共24 paths：root10、refund leaf4、subscription/production leaf6、docs4。

### root

- `services/api/internal/refund/sharedpg.go`
- `services/api/internal/refund/sharedpg_test.go`
- `services/api/internal/subscription/sharedpg.go`
- `services/api/internal/subscription/sharedpg_test.go`
- `services/api/internal/cloudbasepg/client.go`
- `services/api/internal/cloudbasepg/client_test.go`
- `services/api/sharedpg/schema.go`
- `services/api/sharedpg/schema_test.go`
- `services/api/sharedpg/access.sql`
- `services/api/sharedpg/redeem_order.sql`

### staff_import_standards_review

- `services/api/internal/refund/sharedpg_store.go`
- `services/api/internal/refund/sharedpg_store_test.go`
- `services/api/sharedpg/refund_commands.sql`
- `services/api/sharedpg/refund_observation.sql`

### fifth_spec_precommit

- `services/api/internal/subscription/sharedpg_store.go`
- `services/api/internal/subscription/sharedpg_store_test.go`
- `services/api/internal/orderadvance/sharedpg.go`
- `services/api/internal/orderadvance/sharedpg_test.go`
- `services/api/sharedpg/subscription.sql`
- `services/api/sharedpg/order_production.sql`

### Astra

- `openspec/changes/expand-cloudbase-shared-pg-remaining-command/proposal.md`
- `openspec/changes/expand-cloudbase-shared-pg-remaining-command/design.md`
- `openspec/changes/expand-cloudbase-shared-pg-remaining-command/tasks.md`
- `openspec/changes/expand-cloudbase-shared-pg-remaining-command/specs/shared-pg-remaining-command/spec.md`

## External Assets and Non-goals

隔离本地PG由各writer独占，不能并发污染共享角色；固定UAT普通服务身份/管理安装由root持有，秘密不进repo/log。合成退款/订阅provider可证明业务时序，实际微信商户材料与AppSecret已从历史安全存储恢复，官方AppSecret只读校验成功；订阅模板实际为0，老板姓名/手机号及真机授权/支付/退款/送达仍缺验收证据；缺失不阻本切片开发，但不能计真实支付/通知验收。
本change不改main/worker宿主、真实provider配置、小程序、PC、COS、billing/adminreport/audit、采购或Prod。CloudRun五worker与95cases/25pages/UI3属于后续完整运行目标，不能用本切片PASS替代。

## Verification

native/v44完整公共方法差分Red→最小Green→必要Refactor；16并发/幂等/lease/未知ack/provider次数/晚故障回滚及跨支付履约竞争。统一90 installer和最小ACL/readiness、完整影响面ordinary/race/static/smoke/strict。随后普通UAT20入口与负角色、源/清理回执，再Writer Gate、两轴review、本地候选及clean detached exact验证。具体任务及已完成的本地/普通UAT层证据见tasks；最终工件review、owned-only候选及exact SHA独立验证按生命周期继续。

补充批准：finishCreate同RPC零写材料阶段取得历史created_at，Go生成该时刻host Local JSON，最终原消费点刷新receipt；无新增20ABI/24owned/构造。详见design；退款差分及整切片源码已冻结，实际验证见tasks；未形成exact SHA独验前不称集成完成。

Production真实时间反例要求两入口p_now类型timestamptz→jsonb；仅私有签名类型修正，20入口/24paths不变。原始比较、pgx持久时间与Go审计JSON事实分离，见design。

## 后续财务政策依赖

用户已确认UAT/Prod继续共用现有微信支付商户、使用真实资金，支付/退款号和对账按环境隔离，无法确认归属的交易保留异常。此政策由独立change `isolate-shared-merchant-payment-environments` 实施，依赖本切片和read-report精确候选独验。本切片仍保持迁移前的退款号与交易规则；本次迁移PASS不能证明共用商户的环境隔离已实现。
