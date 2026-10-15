> 2026-10-10 当前执行约束：用户已明确停止全部测试与回归并删除测试用例；旧测试及 Gate 记录仅作为历史证据，未运行不能记为通过。本轮直接完成 UAT SQL 安装、服务构建部署及小程序预览交付；不等待旧测试门禁。实际数据库已安装 113 个函数并登记 38 条平台执行记录，既有业务数据与账本保持不变。

# 完整组合 Order UAT CloudRun 运行服务

Status: APPROVED_REQUIREMENTS / IMPLEMENTING。gate_type=W3；ui_level_target=UI0；ui_level_actual=NOT_RUN。

## Why

已有固定共享 PG 端口仍未进入 order-api 的真实运行入口。完成 UAT 需要完整路由、原微信与支付 provider、私有版本图片和五个后台任务在同一服务运行。启动配置不能要求共享 PG 不提供的原生连接地址。

## What Changes

以既有 ORDER_ENV=uat 进入具体 UAT 组合，普通专用身份访问固定 INVOKER RPC。复用原路由、会话、provider、worker 与关闭合同。加入专用 UAT 镜像，11 项公开订阅元组由镜像提供，14 项运行材料由服务版本注入。只实现此运行组合，不改变业务规则。

## Dependencies

本地父为 b991bd06e661f0d0d88f4ad1527ad11965f1ceca；第八 exact 双轴及普通/race full24 独验已通过（external shared-pg-read-report-root/lifecycle-receipt-v1.json）。完整候选及发布还依赖版本固定存储及注册表的已验证实现、退款显示修正、已批准财务环境隔离的精确采用，以及本组合独验。R9 单独 worktree 的下列路径与这些依赖无 writer 重叠，允许先做本地配置和组合 RGR；依赖缺失不使用 stub 或降低发布门。已授权本地候选 UAT 部署；不据此执行 main 集成、push 或 Prod 发布。

权威细化：外部 ninth-runtime-implementation-contract-v2.md SHA f49ed31386ebf7e5aaa8afdb1fdbf063a1f5fc9d8cd1b195a387b43d23986c81；JSON ca6b74e5dd27ef4648cb6da03f77b1088e079a5ec016fec00a1b9d3a42302cb2。第九原需求、容量修订及真实输入回执保持其各自层级。

## Ownership

root 协调的 R9 writer 是以下路径唯一 writer；root 暂停并行写入：

- services/api/cmd/order-api/main.go
- services/api/cmd/order-api/uat_runtime.go
- services/api/cmd/order-api/uat_runtime_test.go
- services/api/internal/config/uat_runtime.go
- services/api/internal/config/uat_runtime_test.go
- deploy/cloudrun/Dockerfile.uat
- deploy/cloudrun/Dockerfile.uat.dockerignore
- deploy/runbooks/order-uat-cloudrun-runtime.md
- openspec/changes/compose-uat-cloudrun-runtime/proposal.md
- openspec/changes/compose-uat-cloudrun-runtime/design.md
- openspec/changes/compose-uat-cloudrun-runtime/tasks.md
- openspec/changes/compose-uat-cloudrun-runtime/specs/uat-runtime-composition/spec.md

root 另划入六个 `services/api/cmd/order-api/acceptance_*_e2e_test.go` 的财务构造调用点：core_invariants、local、pc_pages、refund_unclaimed_l3、transaction_order_l3、user_boundaries。仅补原 test 环境及新 Config 参数，不改业务断言，也不执行延期的95场景/25页面。`services/api/cmd/order-api/testdata/uat_readiness.json` 是本组合测试独占的生成夹具；来自 Finance 实际隔离 PG 的普通身份113实体/109可执行元数据，不含真实用户或秘密，不代表真实云 ready，专用 Dockerignore 排除它。

依赖源由对应 writer 维护，现已交回 root 串行组合。R9 worktree 通过 `git show <exact SHA>:<path>` 精确采用 Finance 候选 `61df2807938839b27b85248a553ba1f904239c4e` 的完整86路径（35生产、47必要fixture、4文档），及 Storage 候选 `2811cabfb21005330c38abb205ce7f4810974f8b` 独占的4个文档；原 R9 owned19逐字保留。权威为外部 `finance-progress/finance-candidate-owned-manifest.json` SHA256 `153b299ce0624c1f255b929d5e1d821167a1fa8d21d59492be5bcf40d39bb4e0` 和 `r9-progress/exact-dependency-adoption.json`。Storage 精确候选独验已通过；Finance 精确候选独验正在完成，不据此提前宣称可发布。

历史35生产冻结仅作接线证据；其后唯一生产增量为 Finance native 空账单保存前的cutoff复核，由 Finance Red/Green及两轴审查覆盖。配置、provider构造、HTTP合同与五worker组合源未改变，原真实25项构造证据按此影响范围继承。native main 和六个验收构造调用点的必要财务参数适配同属 R9；不改原业务断言。

## Validation

配置与完整组合分别执行真实 Red/Green；配置失败脱敏，合成材料不接生产。回归 config、cmd/order-api、app 及受影响业务端口的 ordinary/race、静态、构建、smoke、strict、镜像有效配置与秘密扫描。实际 CloudRun 的完整 ready、worker 推进、lease 交接、退出及必要日志/费用/恢复另取真实回执；代码或镜像构建不替代部署。按 2026-10-10 control 与财务执行顺序 v4 复用条件未变的证据，仅重验影响面及必要相邻。鉴权/RLS、事务、资金隔离、支付退款、核销、恢复、PC 与真实小程序核心 UI3 保留；95 场景/25 页面及不影响本次部署的全量、消息/设备/观测详项为 `DEFERRED_PRE_PROD`，未跑不记 PASS。
