# 后台订单任务简化

授权：2026-10-14 用户要求删除自动制作定时扫描，保留自动进入制作中与 30 分钟取消门槛；其他仅保留必要异常补偿，并验证消融和 E2E。

- gate_type: W3
- ui_level_target: UI2；actual: NOT_RUN
- owner/writer: root
- branch: codex/order-background-simplification
- base_sha: b1a348b96dfc6ac21fd513eb44c186eb42484dff
- owned paths: services/api/cmd/order-api/*worker*.go、main.go、uat_runtime.go；services/api/internal/orderproduction、orderquery、fulfillment、adminreport；services/api/sharedpg 的订单读取/履约/统计与本切片测试；docs/product 中相关 PRD；本 change。openspec/specs/mvp-product-baseline/spec.md；其他补偿路径尚未修改。
- 非目标：业务状态枚举、支付/退款金额、权限、生产部署、计费开通。
- 外部依赖：UAT 当前资源隔离，真实平台 E2E 待环境恢复；不得把本地数据库验证当作微信真机通过。

## 验收
- [x] Red：30 分钟精确边界与停用自动制作扫描后的读取/筛选/履约失败证据。
- [ ] Green：有效状态按服务器时间计算，取消独立校验；列表、详情、统计、出餐一致。
- [ ] 补偿任务逐项核对；无每秒空转，及时路径不受影响。
- [x] PRD 与规格同步（规则确认不代表补偿实现完成）。
- [ ] 消融、幂等、并发、故障、本地链路验证。
- [ ] Refactor：只整理本切片重复规则；复跑同一验证。
- [ ] Writer gate 与候选 SHA。
- [ ] 干净 detached worktree 独立验证 exact SHA。
- [ ] UAT/小程序/PC E2E，明确阻断与未运行。

## 本轮证据（2026-10-14）

- Red：`go test ./services/api/internal/orderproduction -count=1` 精确 30 分钟 initial state 返回 RESERVED，预期 PREPARING。修复后同命令 PASS。
- Red：`ORDER_TEST_PG_BIN=/opt/homebrew/opt/postgresql@16/bin go test ./services/api/sharedpg -run TestTimeDerivedOrders -count=1`：没有制作 worker 时预期制作中列表 2 条，实际空；接着原始 RESERVED 的到期出餐报 ORDER_TRANSITION。逐切片修复后 PASS。
- 本地 Native PostgreSQL：真实 schema/RPC/权限及 state/history CHECK；三条已支付订单固定夹具，初始化时按外键依赖补齐，不触达微信支付。覆盖精确边界、前后边界、列表筛选、详情、状态数量、读不写库、出餐幂等、提前出餐拒绝、取消 v2 RPC 边界、退款状态不覆盖、并发同幂等键只更新一次。
- GPT-6 Luna xhigh 只读复核执行 Go race 与 SharedPG 本地 PG 回归 PASS；不是 exact candidate Gate。复核后的新增并发/取消用例由 writer 实际执行 PASS，尚需 verifier 复验。
- 制作 worker 已从两个启动入口移除并删除对应 runner 文件；其余补偿 worker 尚未替换，仍存在每秒扫描。
- 30 分钟按需调度原型已移出业务源码至 `/tmp/order-background-scheduler-prototype`，因未满足多实例可靠唤醒，不作为候选。
- UAT 平台阻断：CloudRun detail RequestId `58eb4a0e-8e0d-47cc-827b-6133b6492fca`，返回 resource isolated。没有本轮部署、上传、推送或真实 E2E 通过声明。
- 当前状态：IMPLEMENTING / BLOCKED_EXTERNAL（平台与端侧）；还存在时间源一致性与 direct PG 回归未完成，不能形成 CANDIDATE。

- 最终确认口径：异常发生后登记待补偿工作，失败后间隔 30 分钟再试，成功即停止；禁止无异常时固定周期扫描。此前 10 分钟口径作废。
- 本轮完整 `GOPROXY=off GOTOOLCHAIN=go1.26.5 go test ./services/api/...` PASS；`git diff --check` PASS；`openspec validate simplify-background-order-work --strict` PASS。这些结果不覆盖尚未替换的补偿调度、时间源偏差、direct PG 运行链路及真实 UAT E2E。

## 本轮授权扩展与验证（2026-10-14 续）

- 用户明确要求三项修复、后台任务收尾、边界及 E2E；验收通过再部署 UAT，GitHub 网页逐文件提交。旧仓库“停测”批次说明被此次明确测试要求覆盖。
- 新增 owned paths：apps/web-admin/pages/orders.js 及选中行测试；apps/wechat-miniprogram/pages/admin-orders/admin-orders.js 及刷新测试；billing worker、billing service/completed tests；SharedPG billing/readiness/registry、cloudbasepg allowlist；本 change manifest。其余既有改动保留。
- [x] 3.1 Red：商户页慢刷新重复请求 3 != 1；PC 选中行读取计数 11 != 1；已完成账期新服务仍下载；跨日失败阻塞新账期；手动刷新合并后错误提示 0 != 1。
- [x] 3.2 Green：7项 Node 测试通过；Go 日调度/失败/跨日、成功账期复用；原生 PG 最新成功/拒绝记录及商户范围核验通过。
- [x] 3.3 Refactor：只修复原路径、增加读取已有对账结果的固定 RPC，无新存储或缓存服务；同组验证复跑。
- [ ] 3.4 UI2、完整 PC/小程序 E2E、支付/退款/通知恢复验收。
- [ ] 3.5 全部硬约束满足后的候选、独立 exact SHA、部署与逐文件发布。

对账正常改为北京时间10:05处理前一天，成功后次日再唤醒；仅已发生异常的账期30分钟重试。依据微信官方 https://pay.wechatpay.cn/doc/v3/merchant/4013071215 的次日10点后账单可用规则。单个失败账期不再阻塞新一天。已完成结果复用现有 action_audits，重建service也不会重复下载。多个实例同时首次处理同日仍可能重复下载；审计幂等不等于下载去重，尚未宣称消除该竞态。既有跨进程/长停机历史失败账期恢复尚未闭环。

UAT 重新只读查询仍被隔离，RequestId `2f7b4b1d-916e-4e39-a187-eed2b28bee64`（2026-10-14 17:25北京时间）。因此真实 E2E/部署阻断；无计费开通、无部署、无 GitHub 写入。本轮实际已发布文件数为0。

后台支付/退款/通知仍使用原每秒worker；可靠条件唤醒尚无已实测平台路线，不以纯内存timer或固定30分钟全局扫描替代。不得形成可发布候选。原有时间源一致性待核对，未因Node/Go通过而忽略。

- 补强原生数据库链路：真实 Go SharedService → 已安装 reconcile_v2 → action_audits → 重建 SharedService → completed_v1；第二次令外部账单 FakeProvider 不可用，仍读回相同 digest/date/matched，PASS。这里数据库/RPC真实，微信账单提供方仍是隔离Fake，不是微信支付E2E。
- Luna xhigh只读复核发现的手动刷新失败提示缺陷已修复并由其重跑7项Node通过。复核对象仍是WIP，不是exact candidate独验。
- manifest plan校验、OpenSpec strict、diff-check通过；只证明任务计划/文档完整，不替代UI2。
