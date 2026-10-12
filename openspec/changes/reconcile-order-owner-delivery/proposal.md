# Change: 对账全局UAT验收与生产发布前置

## Why

用户要求完整UAT和可操作生产Runbook。现有三份文档虽然更新b3交付，仍沿用历史停测限制及CVM/MySQL表述，缺少当前容量、恢复和平台分项证据。本change仅修正文档，不伪装运行实现或Prod完成。

## What Changes

- 新增全局UAT验收页和单一CloudBase生产发布Runbook，关联本轮隔离验证、真实低频UAT采样和平台只读结果。
- 现有三份runbook/handoff增加当前入口，明确旧方案/停测为历史；保留原实证。
- 固定责任、版本、证据层次、人工停点和生产代码/容量/恢复硬前置。

## 本轮授权与历史停测的适用范围

2026-10-12人类在产品经理会话明确要求新建Owner：“目标是在UAT跑完整个端到端测试流程，真正让用户用起来”，并要求“尽量完整测试整个流程。今晚跑完所有能跑的流程”。新Owner创建时的可信任务交接还明确：历史停止套件约束需结合本轮完整测试授权重新界定，不盲目恢复已删全仓套件。本轮用户直接提供的AGENTS要求按影响面完成必要验证及范围内修复。

本worktree继承AGENTS末尾2026-10-10停测条款，但该条款原文本身写明“新的测试执行以用户后续指示为准”。上述是后续人类指示，允许本任务必要局部隔离测试/审查/文档Gate，不允许恢复无关485个旧测试文件。这里记录适用例外，不修改跨任务AGENTS规则，也不把代码review误判为新的用户禁令。

来源会话：产品经理01a12bbb-ec58-79a3-bc80-7ad3d2d599d9的人类新Owner请求；本Owner01a12c27-b8f2-76e2-a5ec-75b89d6eed9d创建交接。原始CloudBase托管方向及资源证据另见生产审计报告。

## Impact

- Status: APPROVED / IMPLEMENTING；依据本轮用户Owner统筹、完整UAT及生产Runbook授权。
- gate_type: W0；ui_level_target/actual: UI0。
- owner/writer: Order全局Owner；branch codex/order-owner-acceptance。
- worktree: /Users/vivix/.codex/worktrees/order-owner-acceptance/order。
- base_sha: bd29721a2640ea9b56d853304651f768c713a691。
- owned paths: deploy/runbooks/order-production-release.md、docs/quality/order-uat-acceptance-20261012.md、deploy/runbooks/order-uat-deployment.md、deploy/runbooks/order-uat-cloudrun-runtime.md、docs/quality/order-production-delivery-handoff.md、本change目录。
- 只读依赖：b3实际小程序/组合独验、a833 API专项发布、分片PC发布、当前官方文档、原始人类CloudBase托管意图、独立Prod资源只读查询。
- 非目标：不改业务代码、生产运行配置或密钥，不Prod写入、不新采购、不推送/PR/main集成。
- 外部资产：文档W0检查不需要生产写权限。完整业务验收缺实际参与者/体验版/通知设备，Prod备份计划只读受CAM限制，分别记录而不阻断独立文档交付。
- candidate_sha: 提交后记录在仓库外收据，不自引用提交。
