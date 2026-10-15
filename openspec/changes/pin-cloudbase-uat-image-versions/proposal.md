# UAT 图片版本固定与运行身份

Status: APPROVED_IMPLEMENTATION / IN_PROGRESS / CANDIDATE_NOT_FORMED。gate_type=W3；ui_level_target=UI0；ui_level_actual=NOT_RUN。

## Why

启用 COS 版本管理后，旧 PUT 链接仍可覆盖同名暂存图片。仅固定该旧版本会使图片变成非当前版本，在既定30天清理后失效。UAT 必须保持已确认图片可读，并保留既有非当前版本30天恢复合同。

## What Changes

仅 UAT shared-PG 组合新增具体图片服务：校验原上传字节，发布到服务端独占的一次性物理 key，以原子命令回执固定公开 key、物理 key 和 COS 原生版本。原公开 key、上传/确认 HTTP 合同及 native/File 组合不变。新增3个固定 INVOKER RPC；不新建表或改历史44迁移。专用 UAT CAM 源身份仅换取该桶 images/ 的1800秒 STS，COS 只使用临时三元组。

## Dependencies and Lifecycle

base_sha: `b991bd06e661f0d0d88f4ad1527ad11965f1ceca`。branch: `codex/order-uat-version-pinned-storage`；worktree: `/Users/vivix/.codex/worktrees/order-uat-version-pinned-storage/order`。第七、第八均已独验；第八 lifecycle receipt SHA `47ffe37f02f566d8db7924c1a50aa9b52e893e575731d7c2d7354133ea99782e`，本 worktree 已 fast-forward 采用，原16个未提交文件逐个 hash 保持不变，index 为空。6个共同文件在独占 writer 时段完成接线，最终源码registry为108函数、26张RLS表、16序列，历史44迁移不变。完整Writer ordinary/race各25包3056 PASS和七项静态检查通过（full25-writer-runtime-v2 receipt SHA `5925015dd848ba0ad5a3f5ce5819ed0c4df2e4c15bdf7536b9958279db5f3b19`）。安装前绑定的云端基线为105函数、36项平台历史和44项业务账本；历史Push同步InternalError仅保留为该提交通道故障证据，不再阻断已授权的官方直接SQL路线。真实应用矩阵、原版本恢复、实际DB+全部引用图片备份恢复及精确清理已闭环；候选及独验仍待完成。main组合由运行部署change负责。

当前安装路线：root在同一UAT库的事务内经官方 `ExecutePGSql` 执行冻结delta，精确读回定义、owner、ACL、数据、序列及旧36项平台历史/44项业务账本，并由普通身份完整readiness核验108；仅在实际安装完成后经 `RepairPGUserMigrationHistory` 登记 Version `20261009193330` 与精确 Query，再读回37项平台历史和同字节完整37项 `Preview`。安装与登记分属两个接口，不宣称跨接口原子；未知结果先读回，已安装不得重执行105基线guard。实际安装已闭环：事务内执行42ms、108精确定义及权限/旧104不变、数据/序列/44业务账本/旧36历史不变，实际Query登记及完整37 Preview全部applied且无pending/conflicts。专用服务账号保存凭据恢复后，原普通身份完整readiness108通过。真实4.2已闭环，见 external `actual-image-core/closure-receipt.json`；候选及独验尚未完成。执行与读回证据由root另存 `direct-pg-install-closure/`。

权威：external `architecture-ablation/storage-version-runtime-requirements.md` SHA e0a438cbc3e5f78d6bae65d80af817e96dced28ead7044e11921d21046273230。既有用户明确授权 UAT 接入、配置、部署和权限，专用源身份适用性已由 root 记录在 `version-pinned-storage-root/uat-source-identity-authority-decision.json`。真实专用 CAM 源身份已创建；原生 COS 版本 Enabled、images/非当前版本30天生命周期配置、原 Go STS/COS 正向及负权限和精确版本恢复已读回，聚合 receipt SHA `8727665393b3c3d63f06babfcf04fb1678e2cd7f34122b981ea79f5e68697960`。这不证明已观察30天到期、CloudRun或手机UI3。真实应用与备份恢复另由4.2闭环证明。

## Ownership

单 change 的22个 owned paths：objectstore 新 `cos_versioned.go`、`cos_versioned_test.go` 由 storage_cos_versioned_leaf 写；root 写该包 `sharedpg_versioned.go`、`sharedpg_versioned_test.go`、`sharedpg_version_confirmation.go`、`sharedpg_version_confirmation_test.go` 和 `cos_adapter.go`；config 新 `uat_cos_sts.go`、`uat_cos_sts_test.go`；sharedpg 新 `image_version_confirmation.sql` 及聚合 `access.sql`、`schema.go`、`schema_test.go`、`redeem_order.sql`；共同 `cloudbasepg/client.go`、`cloudbasepg/client_test.go`（仅3已批准名称与普通传输）；根 `go.mod`、`go.sum`（仅官方 STS 依赖有实际需要时）；本 change 四工件。不得覆盖其他 writer。`cos_adapter.go` 仅加新方法 marker 激活的有界响应钩子，避免 SDK 无界读取；旧方法行为不变。

## Verification

原同名覆盖 Red → 新具体服务 Green；PNG/JPEG、5MiB、原错误优先、16并发单 winner、当前 OWNER、跨 scope、丢返回/重启、缺失/损坏 pin、同 snapshot 签名/expiry、STS32并发及最小权限。隔离 PG/RPC 普通与 race；最终 registry/readiness/44账本、必要相邻回归、Writer Gate、两轴 review、本地候选及 clean detached exact 验证。真实 COS/IAM/恢复、CloudRun 和 PC/Mini 完整95场景25页/UI3单独验收。

客户端名单勘误：external storage-client-owned-path-erratum.md SHAaa10c124507c71f4a8f0d866c073e5c7716580774d247568db211e6cd2f265c3，20→22；不允许泛型绕过名单或改变8MiB成功界。client/registry共同文件仍等第八独验+adoption后root串行，不与root8并行写。
