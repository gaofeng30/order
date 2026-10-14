# 读取链路修复证据

基线：`0aba16b086bd3d2770e7dce41ccb86311e697587`（SSH fetch 的 uat）。独立实现分支 `codex/order-read-performance`；未改用户原 checkout、未推送或部署。

## 结论与第一性原理

餐段是否可售需要的是营业/截单/商品存在性事实，不需要整份菜单、身份价格和图片；浏览需要图片地址，不需要后端重复下载验证固定版本原图；角标需要计数，不需要每一笔订单明细。按这三个边界删除无用工作，不增 Redis、跨请求缓存、索引、后台任务或基础设施。

| 路径 | 基线问题/Red | 本地 Green |
|---|---|---|
| 小程序餐段 | 四可用餐段触发四次完整菜单探测，再读选中菜单；测试报 Full menu must not be fetched | pickup-options 一次，服务端 EXISTS 汇总；选中菜单读一次；无无关餐段故障耦合 |
| 已有支付恢复 | 售罄/截单后 confirm 调用 0 次，断言 `0 !== 1` | 已有预支付先确认，调用 1 次并解锁；新购不可用仍 0 次、无新预支付 |
| 图片地址 | PublicURL 测试触发 COS HTTP，object calls=1 | 16 图一次 pin RPC，SQL一次读取全部命中receipt、0 次 COS GET/HEAD、0 原图下载；去重；版本缺失不猜地址；浏览无图继续；Confirmed 保留原图检查 |
| PC订单角标 | 测试收到 `/admin/orders?limit=100`；1000单诊断需11请求 | `/admin/orders/counts` 一次，仅6状态数；非法/负数拒绝，权限保持 OWNER |
| 报表统计 | 全历史聚合，20万旧单参与日期转换 | 同索引、200505行夹具结果完全一致：`28500/19/602468/402/101`；基线51.885/56.046/55.327ms，改后12.850/7.937/6.971ms |
| CSV | 完整订单+商品明细 RPC；窄读取测试失败 | 仅ID游标和5个输出字段；100条分页、BOM/列顺序/时间格式/金额/排序保留；不累积订单对象 |
| CSV分页撤权 | 审查发现初版 native 优化仅鉴权1次，读2页；回归Red实测 | 每页鉴权恢复；第二次撤权即 ErrForbidden，返回nil，不读第二页 |

以上耗时来自**隔离 PostgreSQL 和模拟依赖**，不代表线上 P95、负载容量或手机渲染耗时。先前消融中的图片16次RPC+16次下载约1.77MB，与本次调用预算测试属于不同验证层，不能当线上节省值。

## 必要依赖修复

新读 RPC 需要更新 SharedPG 定义、ABI、权限、运行时白名单和 readiness 数量。过程中发现基线已有114个函数而 Check 写113、readiness 漏 `order_import_result_v1`。将本次 readiness 对齐到116（原114+新2），补漏项；没有改变该函数权限或实现。将相同 readiness 测试注入临时基线副本，实测 `local definition drift prevents runtime readiness before RPC`；候选源码通过。此项仅证明本地定义可进入 readiness RPC，不证明远端 readiness 已通过。

## 验证入口和边界

`sh openspec/changes/simplify-read-paths/checks/run-local.sh`：真实 JS 模块、Go HTTP与适配器、race、vet、隔离 PG16 查询/RPC、OpenSpec 和小程序 TDD manifest。PostgreSQL 脚本自行创建临时目录和 Unix socket、不使用环境DSN/生产凭据，finally关闭清理。

独立审查 Standards 与 Spec 各发现同一项 native 导出分页撤权回归；已按 Red→Green 修正。批量SQL再审通过；与旧函数针对正常/重复/缺pin/冲突/错误key/非法actor逐项比较一致。无其他可行动发现。不得将该审查冒充 exact SHA 验证或真机验收。

UI实际等级：UI0。当前实现 worktree 没有微信运行窗口（`cant find runtimeid by projectpath`），匹配本次契约的 SQL/API 未部署；UI2菜单主路径及UI3真实支付恢复为 BLOCKED_EXTERNAL，不进行真实资金操作。部署授权、匹配环境、测试账号/设备是续接条件。缺少 receipt 时不进入 INDEPENDENT_VERIFIED/集成。

## API和发布约束

- 新增 `GET /api/v1/admin/orders/counts`，沿用管理端认证和 OWNER 校验。成功 `{"counts":{"RESERVED":0,"PREPARING":0,"READY_FOR_PICKUP":0,"COMPLETED":0,"REFUNDING":0,"REFUNDED":0}}`；全历史、无筛选，与原角标口径一致。没有新增任何订单详情字段；错误使用既有403/503契约，响应no-store。
- pickup-options 外部结构不变，available 现在包含该日期/餐段至少一个在售、未售罄且分类启用的商品；无身份/价格字段。新API遇到未升级旧SQL的缺失 available_meals 会失败关闭，不能静默当成全无货。
- 图片使用确认的 published_key+version_id 生成短期URL；不回退到未固定版本。丢失/错误 pin 时消费者显示现有缺图态。PC编辑读取缺 pin 仍失败，避免保存时丢失原图引用。新批量pin RPC上限128，Go分块；商品写引用仍最多3图且验证字节。
- 发布需要SQL→API→前端同一受控窗口，先验证服务身份readiness及OWNER拒绝用例，再验证菜单选择/价格/图片和CSV对账。API回滚须同步恢复其匹配SQL定义（readiness校验函数体hash）；不涉及数据迁移。
- 上线验收采集 `/menu/pickup-options`、`/menu`、`/catalog`、`/admin/orders/counts`、`/admin/stats`、`/admin/finance/export` 的P50/P95、错误率、请求数，覆盖冷启动/并发/失败态。SLA目标和线上数据尚未确认，不声明SLA达标或容量提升比例。

## 2026-10-14 UAT 部署回归发现与修复

- 部署 0e440ad 后 `/health/ready` 返回 503 database_incompatible，菜单读取 200。
- 新增 `TestInstalledCatalogPassesReadiness`：在临时 Unix socket PostgreSQL 16 完整初始化 v1-v44 schema、安装全部 RPC，以 authenticated 角色读取元数据并运行实际 Check；修复前稳定失败。
- 修复 import_result 在期望注册表中的顺序、运行可调用数 109→112，以及 import_result 缺失的 UTC 函数配置。完整回归由 Red 转 Green，随后 race、run-local 全部通过。未放宽健康检查或权限契约。
- 此回归已加入 run-local；本地原生 PostgreSQL 验证不冒充 CloudBase 用户侧 E2E。
