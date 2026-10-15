# Tasks

Status: APPROVED / IMPLEMENTING。W0 / UI0；base、ownership 与依赖见 proposal。

- [x] 1.1 从固定 base 建独立 worktree，读取规则和代码/平台依据。
- [x] 1.2 Red：`test -f deploy/runbooks/order-uat-deployment.md` exit 1，目标指南不存在。
- [x] 2.1 写最终正确步骤，覆盖资源、完整配置、构建、原生TLS、migration/bootstrap、常驻API、路由、客户端与回退。
- [x] 2.2 Green/Refactor：同一文件检查、完整性/链接/命令静态检查、OpenSpec strict、owned/diff/敏感检查。
- [x] 3.1 两轴 review、Writer Gate、自审评分与 owned-only 提交前核对。
- [ ] 3.2 本地候选提交；完整 SHA 与提交后 clean 保存仓库外 receipt。
- [ ] 4.1 不同 verifier 的 clean detached exact-SHA 文档验证。

候选 full SHA 和提交后的任务完成/clean 记录保存仓库外 receipt，不修改已提交的任务快照。未 push、集成、部署、采购；完整 UAT/prod 验收不由本 W0 change证明。

## Evidence

- Red：创建正文前实际执行 `test -f deploy/runbooks/order-uat-deployment.md`，exit 1（文件不存在）。Green 与 Refactor 重跑相同命令均 exit 0。
- Green/Refactor：仓库外 `runbook-document-checks/verify-runbook.py <writer-worktree>` exit 0；33 个源码必需变量无缺漏、4 个相对链接存在、4 个 Bash 片段语法通过、严格 JSON 字段与四份生产 SecretName 完整、44 个 PG migration 文件对应、Dockerfile 用户/端口/入口正确。没有执行正文的部署或资金命令。
- `openspec validate document-uat-deployment-runbook --strict`、`git diff --cached --check` exit 0；owned 路径仅正文及本 change 4 个 Markdown，无真实材料。
- Spec/Standards 两轴分别收尾为 0 findings；迁移后权限交接和静态路由开关已修正并复核。无额外重构需要，重复文档检查确认正确流程与公共契约保持。
- Writer C9/T10/V8/R9=36；完整 W0 范围检查及失败/回退条件覆盖，V8 仅候选验证包就绪，不声称 independent PASS。
- Go/race、UI、真实数据库/资金、云部署与生产检查 N/A：本 change 只新增操作文档，不改变这些实现；真实执行仍必须满足正文 Gate。
