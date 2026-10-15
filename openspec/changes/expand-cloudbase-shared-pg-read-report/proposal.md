# 共享PG对账与报表完整切片

Status: APPROVED_REQUIREMENTS / LOCAL_FULL24_PASS / ACTUAL_UAT_DATABASE_GATE_PASS / CANDIDATE_NOT_CREATED。gate_type=W3；ui_level_target=UI0；ui_level_actual=NOT_RUN。

## Why

闭合剩余billing、adminreport、audit.Search和storestatus.Apply直连DB端口，保留原公共行为，供后续完整runtime组装。不是新的经营报表或退款政策。

## What Changes

原17公开方法通过15固定RPC实现：billing6，reports9（admin7+audit1+storestatus1）。CSV仍Go分页，金融写端口委托既有payment/refund应用；条件registry90+15=105。不新增公共API、通用SQL、Tx shim、配置或业务容量限制。

## Dependencies

当前worktree已采用独验通过的第七精确候选 ea76e34845a72b15334fb510ac2906e7bbe0ddb0。root独占共同registry并完成105统一安装器与最终本地完整回归；四个开发临时fixture已全部移除。真实UAT105验证已完成；完整Writer Gate、候选与exact独验仍为本切片必需条件；父采用不表示main集成。

批准源：external architecture-ablation/eighth-final-read-report-requirements.md SHA503e6feca346195aa12004205f7573f72714022d0e8e8d36b1bd022999a35d62；JSON SHA f901ec45b494f6baedd61eb98bbe513963c7c2ac46053f487347bfb567fcc956。并行边界见shared-pg-read-report-root/parallel-dependency-ownership.json；本次root明确授权四文档现在写入，替代该旧快照的“after Seventh4docfreeze”时序文字。

## Ownership

root拥有billing5和common7；reports9已移交root；需求脑四文档的实现记录由root维护，不改批准合同。共同文件现由root8单 writer。所有25 owned路径如下，native提取须重新逐路径分配，不自动扩权。

- `services/api/internal/billing/sharedpg.go`
- `services/api/internal/billing/sharedpg_test.go`
- `services/api/internal/billing/sharedpg_reconcile.go`
- `services/api/internal/billing/sharedpg_reconcile_test.go`
- `services/api/sharedpg/billing.sql`
- `services/api/internal/adminreport/sharedpg.go`
- `services/api/internal/adminreport/sharedpg_test.go`
- `services/api/sharedpg/admin_report.sql`
- `services/api/internal/audit/sharedpg_search.go`
- `services/api/internal/audit/sharedpg_search_test.go`
- `services/api/sharedpg/audit_search.sql`
- `services/api/internal/storestatus/sharedpg.go`
- `services/api/internal/storestatus/sharedpg_test.go`
- `services/api/sharedpg/store_status.sql`
- `services/api/internal/cloudbasepg/client.go`
- `services/api/internal/cloudbasepg/client_test.go`
- `services/api/sharedpg/schema.go`
- `services/api/sharedpg/schema_test.go`
- `services/api/sharedpg/access.sql`
- `services/api/sharedpg/redeem_order.sql`
- `services/api/internal/fulfillment/sharedpg_test.go`
- `openspec/changes/expand-cloudbase-shared-pg-read-report/proposal.md`
- `openspec/changes/expand-cloudbase-shared-pg-read-report/design.md`
- `openspec/changes/expand-cloudbase-shared-pg-read-report/tasks.md`
- `openspec/changes/expand-cloudbase-shared-pg-read-report/specs/shared-pg-read-report/spec.md`

## Validation and Limits

先原native公开行为Red，再最小Green和必要Refactor；日期/时钟/扫码身份权限/CSV/金额/并发快照/原子故障/丢ack/容量逐项差分。之后统一普通身份105、原44ledger、最小权限、普通UAT和精确清理，Writer Gate及exact detached独验。当前统一最终source559/owned25完成full24普通/race各2786 PASS、0 FAIL/SKIP和七项静态/构建/smoke/strict检查，runtime receipt SHA1cdbd84ee087bddca3e8eefa3c2da0df0d45baf2d772f9a886eb2be5cb1b0a6d。该证据限定本地writer层；未形成候选；真实云105及两写族并发/回滚/未知ACK、权限/清理已完成，证据在tasks及外部actual-platform-final-closeout-v3.json。main、CloudRun、真实微信/支付、私有版本存储30天恢复、95cases/25pages/UI3仍后续门禁。

## 精确父采用与最小所有权勘误

第七 ea76e34845a72b15334fb510ac2906e7bbe0ddb0 已INDEPENDENT_VERIFIED，full20普通/race各2437 PASS、0 FAIL/SKIP、Spec/Standards0 mustfix；external生命周期SHA5f96cd0022c9e11013e9c6e71ae89614c5364558c402db07863a0a805105d283。root已本地ff采用该父，18未提交路径hash保持，index无相邻改动，adoption receipt SHA151b3266ca0dcdcd954801ef9b46d3575b840521fef8ce07a0e289bf67bb0f92。

Astra外部勘误eighth-readiness-owned-path-erratum.md SHA8825c71a189748088eb3fa3874824b678ad044d6fa57b02ef04e9bf96d716560追加redeem_order.sql，24→25，仅readiness登记15既定ABI；该文件两个核销函数body、原native44保持。common注册105、RLS26/USAGE16，不预授新表权限。四开发fixture已改为统一installer+Check；本地结构/传输Red→Green通过，完整新基线本地writer回归已通过，实际云105已通过，候选独验仍待完成。
