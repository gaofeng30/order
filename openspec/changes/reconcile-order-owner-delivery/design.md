# Design

唯一验收入口为docs/quality/order-uat-acceptance-20261012.md，唯一后续生产执行入口为deploy/runbooks/order-production-release.md；现有UAT文档保留历史运行细节并指向新入口。

按环境、源码/构建、部署、API、数据库、用户页面、设备区分证据。独立本地SQL和provider矩阵只证明对应切片；低频UAT读取不等于容量；Prod资源存在不等于业务上线。所有未执行的Prod动作标NOT_RUN。

后续人类CloudBase托管意图消解旧技术选型，不改变事务、资金、隔离、恢复目标。当前代码尚未生产适配，Runbook开头明确不可直接发布，并列出准确模块、阶段、验收和停止条件，不提供虚假的一键上线命令。

Red/Green使用仓库外check_docs.py检验两个入口的必需边界与本地链接；openspec严格验证、diff/owned-path和原生Codex review为Writer门。Go/build/UI均N/A：不改运行行为或配置。Refactor无额外结构改动必要，重跑同一内容/链接检查。

候选提交后另一干净detached worktree做exact-SHA只读验证；不因文档通过将整个UAT或Prod标PASS。后续内容改变形成新候选和新独验。回退仅还原本change文档，不触及运行状态。
