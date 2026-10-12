# UAT 管理台售罄营业日刷新

基线：`67efa16b898901cbfba3c5ca1cb302d69524dff9`；现场 API 部署为 `7e7f2f9a75e87a4f311e0ff7e2496022d8569f82`。唯一 writer 路径为 `apps/web-admin/data/api.js` 与本任务文档，复用 root 已授权的 UAT worktree；其他并行任务的源码不在本切片范围内。

用户已停止测试套件和重复回归；本切片不新增、恢复或运行测试用例，只进行必要语法、差异检查及后续真实 UAT 验收。root 统一选择性提交与 PC 单文件发布。

- [x] 1.1 Red 与定位：北京时间 2026-10-12 00:17，PC 正常菜品管理标记 F（product_id=12）售罄，顾客明确选 10/12 午餐 11:30 仍返回 `sold_out=false`、`orderable=true`。root PG 读回售罄记录只在 10/10、10/11，没有 10/12；跨午夜的 PC 写入了旧营业日 10/11。`data/api.js` 的 `today()` 只返回缓存 `state.settings.service_date`，列表与默认售罄写入未刷新 settings。共享 PG settings 从 `clock_timestamp() AT TIME ZONE 'Asia/Shanghai'` 生成权威日期；写入与菜单查询都按请求日期执行。旧临时 10/11 售罄已由 root 用正常 PC 流程恢复，PG 只剩此前 10/10 记录；设置页权威日期已显示 10/12。
- [x] 1.2 最小实现与静态检查：`listProducts()`、`getProduct()` 先获取现有 `/admin/settings`，将本次返回快照日期固定用于产品查询；列表售罄集合用相同日期。`setSoldOut()` 仅在没有显式日期时刷新 settings，固定该日期用于写入与本地集合；显式日期保留原契约。`node --check apps/web-admin/data/api.js`、`git diff --check -- apps/web-admin/data/api.js` 无错误。源码 SHA256：`4fd6644f7e0faf98f88562e3f2da44dee4886f33a32b9f4684e8bbf65ba040d8`。
- [x] 1.3 Refactor：无需重构；只修改上述三个既有函数，不新增抽象、配置、浏览器日期计算或时区替代，不改 SQL、服务端、菜单、售罄按营业日隔离的业务契约。settings 获取失败会沿既有异常路径停止，不使用旧日期继续写入。未运行自动测试套件或生命周期 Gate。
- [x] 2.1 真实 UAT Green：root 精确源提交 `700031c13e105bd32a0bc88ffbb511693a254271` 完成三个新增 await 边界的原始会话保护；干净 detached 精确源码独立审查通过。单个 Hosting 文件 `web-admin/data/api.js` 实际上传成功，30233 B，源 SHA256 `c656b128c3c009afa19656b8aba66a4212cc153adca9c022b9a0ef7adf5f6afb`，远端对象 ETag 与源 MD5 精确相同。正常 PC reload/标记 F 售罄，官方 PG 读回正确10/12；顾客正常选10/12午11:30显示售罄/不可购，实际tap禁用按钮一次，购物车数量0/空items不变，未进入confirm、无新quote/prepay。正常 PC 恢复后可购，PG只保留原10/10历史标记。实际外部回执 `root-F12-700031c-date-green-restored.json`，顾客回执 `customer-execution-receipt.json`。原始脚本浏览器导航 ERR_BLOCKED_BY_CLIENT，未重试绕过；不登记HTTP响应体摘要。

当前状态：SOURCE_COMMITTED / SOURCE_REVIEWED / ACTUAL_UAT_PUBLISHED / ACTUAL_CASE06_GREEN_AND_RESTORED。554b854 初次源码的新增 await 会话 guard 问题已由700031c修复，554未发布；旧阻断历史保存在独立回执。未运行自动测试套件或正式生命周期 Gate，不登记 formal CANDIDATE / INDEPENDENT_VERIFIED、main集成或完整18项UAT PASS。
