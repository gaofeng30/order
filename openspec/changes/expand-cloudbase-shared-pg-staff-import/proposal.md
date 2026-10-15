# 共享 PG 员工管理与导入闭合

Status: IMPLEMENTED / WRITER_V4_GREEN / PRECOMMIT_SNAPSHOT_FROZEN / CANDIDATE_NOT_CREATED。change_id: expand-cloudbase-shared-pg-staff-import。gate_type=W3；ui_level_target=UI0；ui_level_actual=NOT_RUN。最终source-v4的16包ordinary/serialized race各1679PASS、0FAIL/SKIP，静态及分层普通UAT业务/容量/恢复/清理证据完成。候选前最终两轴与暂存核验、candidate、exact-SHA独验、main集成仍待，详见tasks。

## Why

既有共享PG提供目录、报价与身份端口，但员工折扣管理和XLSX导入仍依赖原生事务。本切片完整迁移Staff Application三方法/五命令以及Import Application两方法/商品和员工两类导入，闭合名单与目录写入依赖，不启用部分后台。

## What Changes

- 普通服务UID经六个固定INVOKER RPC完成原读写；保留Go解析、规范化、公开DTO、错误、完整request摘要和receipt。
- 五种Staff命令均保留原StaffID及其审计target（0对应DISCOUNT_SETTINGS/1，非零对应STAFF/原ID），包括CREATE/SET_RATE。原当前OWNER、锁序、版本、导入行规则、全批回滚、锁后到期与审计时间不变，最终committed_at数据库时钟及到期CHECK不改；不存在通用SQL执行器或跨HTTP事务。
- 统一registry从49到55，按实际SQL清点新增RLS/列ACL/sequence及严格readiness，原44迁移和ledger不改。
- 公共Client按55个已审入口输出合同采用精确24项HTTP200成功响应无人工cap白名单；其余31项/signin/非200仍8MiB。跨前序切片影响由本change回归，不修改前四冻结工件，不新增业务容量限制。

## Ownership and Dependencies

base_sha: a6e597976e4ddee4ed2101ac21e9d1d4a3147983。第四Spec轴已PASS，aggregate independent已INDEPENDENT_VERIFIED；前序changes未集成main。正式WT为/Users/vivix/.codex/worktrees/order-uat-shared-pg-staff-import/order，branch codex/order-uat-shared-pg-staff-import，HEAD精确等于上述base。集成依赖另核，candidate_sha提交后只记录外部receipt，避免自引用。

所有源码路径以services/api/为根：

- 原staff writer已完成并结束；root接管 internal/staffdiscount/{sharedpg_admin.go,sharedpg_admin_test.go}，仅本次大Name命令response解码cap的必要修复/回归。sharedpg/staff_admin.sql保持原writer结果，本容量修复不改SQL。
- import writer仅 internal/importbatch/{sharedpg.go,sharedpg_test.go,plan.go,postgresql.go}、sharedpg/import_batch.sql。plan.go及postgresql.go仅必要package-private纯plan函数提取；不得顺改native provider业务。
- root/common仅 internal/cloudbasepg/{client.go,client_test.go}、sharedpg/{schema.go,schema_test.go,access.sql,redeem_order.sql}六路径；最后一项仅readiness事实投影，既有业务函数body不变。
- Astra仅本change proposal.md、design.md、tasks.md、specs/shared-pg-staff-import/spec.md四工件。四工件已从批准staging拷入；后续本change四工件由Astra单独维护。

共18个实际owned路径。公用fixture若需变动，先具体划分新增路径/理由；不自动修改其他writer的文件。root拥有外部平台harness与回执。独立验证者不能是本轮业务writer。

## Minimum Success and Boundaries

Staff五命令和两类Preview/Commit必须完整原生差分、原handler、真实并发/回滚/未知提交与权限门。必需受影响ordinary/race、构建、静态检查、统一安装和普通UAT矩阵、清理、精确candidate clean detached双轴独验分别完成。仅需求批准不勾实现任务。

本轮不改main、支付/退款/worker、storestatus、报告、对象上传或确认Record；XLSX保持现有multipart内存解析和file_object_key=NULL。后续仍保留五持续worker、私有字节不可变/原生版本30天恢复、PC同源、真实微信材料及95cases/25pages/UI3，ICP除外。AppSecret仅是现有directHTTP provider依赖，目标UAT免密微信路线尚未验证，不在本轮改变。无Prod、采购、push/PR/merge或整套runtime上线。

## External Assets

root独占外部shared-pg-staff-import-root中的平台迁移artifact/harness/回执。平台管理请求携完整历史后出现大小阻断，非PG容量或采购前提。批准仅管理面旧readiness源码hash绑定的有界DO增量生成；不得修改已冻结Go/SQL业务源、旧history、原44或新增runtime接口。该有界delta已完成本地验证与目标UAT执行、全量核验、官方历史登记，完整限定证据见design/tasks。

实际部署采用官方ExecutePGSql受审版本化事务→全量核验→官方RepairPGUserMigrationHistory登记。history24/旧23fresh baseline逐字不变；新Query三处外围LF差异经限定词法核验，executed/stored分别存SHA。当前状态RECORDED_FORMAT_VARIANT_VERIFIED，已非APPLIED_UNREGISTERED；不提升为整套runtime或真实微信验收。

## Writer文档冻结

四工件为候选提交前快照，冻结后不为填写同一提交SHA再次修改。candidate与clean detached独验状态由外部生命周期receipt绑定实际完整SHA；本快照CANDIDATE_NOT_CREATED不表示后续外部动作未发生。任何获授权源码/规格修复产生新候选并重验，不能继承旧独验。
