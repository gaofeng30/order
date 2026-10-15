# Prod 图片存储简化

依据：2026-10-15 用户确认“以原始 PRD 为准，无明确要求就删除”。原始 PRD 0818 §6.8 没有原生版本历史或图片误删后 30 天恢复要求；§9.5 核心数据备份恢复方案保留。

基线及已部署 SHA：149423eac8b179cf453e224534c64f84fe458086，SSH main 读回一致。独立 branch/worktree：codex/order-production-ablation-20261015，/tmp/order-production-ablation-20261015。

所有权：image_ablation 负责 objectstore 与 image_version_confirmation.sql 及其 SQL 验证；主 agent 负责 runtime 接线、生成校验及发布文档；文档 agent 负责限定生产规格、Runbook 与技术指南的收敛。依赖：图片实现完成后统一运行 catalog 校验，禁止并行写同文件。

- [x] Red：当前适配器合法 PNG 缺 x-cos-version-id 失败，普通读取成功，见 diagnostics/image-ablation/red.log。
- [x] 对照：真实 adapter/signature/content checks -race PASS；并发行为只在模型验证，不冒充真实 COS。
- [x] 原始 PRD 核对：移除没有产品依据的图片历史恢复要求；保留私有访问、hash 校验、持久幂等。
- [x] Green（本地）：Prod 单对象不可覆盖确认，UAT 原有版本对象不受影响；隔离 SQL 回执并发与重启恢复通过，真实 COS 仍待验。
- [ ] SQL 定义、ACL/RLS、身份及账本按新候选生成核对；不重跑成功迁移。
- [x] Refactor：仅移除 Prod 原生版本依赖，保留 UAT 与鉴权、摘要校验、持久幂等；不扩展其他业务模块。
- [ ] Writer Gate、候选及独立验证。
- [ ] 逐文件 GitHub 网页发布与读回、集成 main、精确 SHA 部署及平台业务验收。

本地通过不是生产发布。COS 版本控制和相关费用未开启；真实平台写入/读写请求费用与图片处理开通仍按用户成本边界执行。部署时须只更新差异函数，保留已成功迁移。当前 NO-GO，整体验收未完成。

## 本地验证回执

- `go test ./services/api/...` PASS（未设置数据库变量的集成用例会跳过，不当作实证）。
- `ORDER_TEST_PG_BIN=/opt/homebrew/opt/postgresql@16/bin go test ./services/api/sharedpg -run ^TestInstalledCatalogPassesReadiness$ -count=1` PASS，真实隔离 PG16 验证 UAT/Prod 目录、函数定义和权限。
- 图片独立 PG 测试：原定义 Red=UNAVAILABLE，新定义 16 并发确认只落 1 条回执、数据库重启回放、权限和冲突 PASS，见 diagnostics/image-ablation/IMPLEMENTATION.md。
- `go vet` objectstore/sharedpg/order-api PASS；`git diff --check` PASS。
- 完整生产函数 SQL 已按普通 UID 与正式 origin 从当前源码生成，仅本地文件，不重复执行迁移；尚未安装新函数或部署新镜像。
