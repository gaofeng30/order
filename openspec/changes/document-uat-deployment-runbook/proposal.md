# Change: 整理 UAT 正确部署 Runbook

## Why

交付需要一份可供生产参考的正确操作指南，涵盖实际资源能力、版本、配置、发布顺序和完成标准，不复述排障过程，也不将尚未执行的部署写成成功。

## What Changes

- 新增 `deploy/runbooks/order-uat-deployment.md`，唯一描述 CloudRun/PG UAT 的部署流程及生产复用边界。
- 用代码、实时只读平台资源和官方文档核对完整变量、原生私网/TLS、常驻 worker、迁移/bootstrap、同源网关、客户端、验收和回退。
- 不更改运行行为、生产架构、已有资料、平台资源或付费设置。

## Impact

- gate_type: W0；ui_level_target/actual: UI0。
- owner/writer: root；branch: `codex/order-uat-deployment-runbook`。
- worktree: `/Users/vivix/.codex/worktrees/order-uat-deployment-runbook/order`。
- base_sha: `69b4dad2fb8f7c03ae07585e982fe60dee2f0fd8`。
- owned paths: `deploy/runbooks/order-uat-deployment.md` 及本 change 的 proposal/design/tasks/spec。
- 只读依赖：PG UAT、CloudRun 身份/材料来源候选；现行生产架构规范、95-case 矩阵与代码配置。
- 文档依据候选的接口事实；实际部署必须使用依赖已集成的正式发布 SHA。文档验证不代表依赖已集成或业务 UAT 已完成。
- 状态：APPROVED（用户明确要求最终 runbook）/ IMPLEMENTING。
- external assets: W0 文档检查不需要真实材料、付费资源或生产变更；这些仍是 runbook 实际执行前置。
