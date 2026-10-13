# PC 后台精修实施任务

- 来源：docs/product/pc-admin-ui-polish-checklist.md（P1–P45、C01–C03）；docs/product/pc-admin-ui-predevelopment-audit.md（D01–D15、V01–V15）。
- 基线：b56c5b78b916096783a84a20fb92cadcba5532fd；分支 worktree-pc-admin-polish。
- owned paths：apps/web-admin/**、services/api/internal/importbatch/**、services/api/internal/catalog/postgresql_admin.go、services/api/sharedpg/catalog.sql、services/api/cmd/order-api/main.go、services/api/cmd/order-api/uat_runtime.go、services/api/sharedpg/import_batch.sql、services/api/sharedpg/schema.go、services/api/sharedpg/access.sql、tools/pc-admin-preview.cjs、本任务文档、本分支纯测试文件/fixture/runner（按本轮删除要求）。小程序生产文件不修改。
- 接口：GET /api/v1/import/batches/:id/result，仅当前启用主账号读取其本人已提交批次；结果含 kind、batch_id、rows（row、id、outcome、name、phone_masked、category、meal、price_cents、description）。成功记录在导入事务内采集稳定 ID 并存入 result_json；不保存完整手机号到结果快照。现有提交响应增加 kind/rows，保留已有字段，复用幂等 receipt。
- 最小成功标准：已确认范围代码完成、生产编译成功、本地预览可打开；实际 UAT 与接口上线状态单独记录，不以静态检查或编译替代。
- 用户覆盖：停止测试、回归与 Gate，删除本分支自动化测试/纯测试 helper/fixture/runner。Red/Green/独验不运行、不登记 PASS。

- [x] 共用组件与导航、响应式布局、样式规范。
- [x] 菜品、裁剪补图、批量操作、分类。
- [x] 导入结果接口及两类导入页面。
- [x] 员工、账号、订单、财务、支付待处理。
- [x] 营业设置、工作台、开屏、登录。
- [x] 删除纯测试资产；保留生产/实际 UAT 工具。
- [x] 生产编译、语法检查、本地预览及实际结果记录，见 implementation.md。
- [ ] 完整响应式与业务实际 UAT；新增导入接口部署待单独授权，不登记通过。

- [x] 追加品牌资源要求：PC 侧栏、账号卡片、开屏布局参考、浏览器页签统一使用用户指定的 apps/wechat-miniprogram/assets/logo.jpeg，原样复制至 apps/web-admin/assets/logo.jpeg；不修改小程序资源。文件一致性和引用检查完成，浏览器视觉 UAT 待整体页面完成后进行。

- [x] 用户追加修正：下架优先显示“不可售”并禁用售卖开关；单条/批量下架共用列表展示；重新上架读取原当日售罄记录，不额外写入售罄事实。`node --check apps/web-admin/pages/products.js` 与 `git diff --check` 成功；未运行测试或 Gate，本次实际浏览器确认未执行。

- [x] 用户追加调整 P37：恢复 b56c5b7 的原有字号层级及页面内辅助字号，移除固定档位覆盖；保留响应式、行高、点击区域和业务修正。未运行测试、回归或 Gate；实际视觉 UAT 待用户预览确认。

- [x] P46：菜品批量删除入口、不可恢复确认清单、复用单条删除、重复提交保护、失败名称/原因弹窗与列表刷新。仅修改 PC 页面及任务文档；未操作 UAT 业务数据，未运行测试、回归或 Gate。
- 静态检查：`node --check apps/web-admin/pages/products.js`、`git diff --check` 成功；浏览器实际删除验收未执行。

- [x] 核对 P46 空分类规则：现有 DELETE_PRODUCT 不检查分类剩余数量、不删除分类；顾客端 menu.js 的 applySearch 已过滤空 products。无需修改该业务逻辑；仅补充确认记录，未运行测试或操作线上数据。另发现售罄记录存在产品外键 RESTRICT，不能将潜在关联冲突归因于空分类；未扩大本次修改范围。

- [x] 用户再次授权修正删除：在单条删除原子事务中锁定菜品、先删除售罄关联再删除菜品；批量删除复用该路径。不删除分类或历史订单。
- [x] 用户纠正 P4：上传 XLSX 后展示逐行可编辑预览与补图入口；保存修改重新生成工作簿并调用现有预览接口校验，确认前不创建菜品；图片引用经过确认检查后进入预览与最终导入事务。
- [x] 扩展已有 preview multipart 的可选 product_images（按工作表行号索引，最多三张/行），响应 rows[].product 返回解析字段与原始售价/餐段文本；不暴露员工完整手机号。预览幂等摘要包含图片引用；commit 路径及 token 幂等规则保持。
- [x] 生产 Go 编译、修改 JS 的 node --check 与 git diff --check；未运行测试、回归或 Gate。后端 API/SharedPG 尚未部署，完整真实业务 UAT 未执行。

- [x] 用户追加列表密度调整：勾选列从 112px 收窄为 80px，菜品列固定为 360px，表格最小宽度从 1240px 调整为 1072px，末尾操作列承接剩余宽度；保留字号、图片、行高。JS 语法与 diff 格式检查完成；未运行测试/回归/Gate，未完成浏览器视觉验收。

- [x] 用户追加分类拖拽排序：最左侧手柄、插入位置标识、既有完整排序接口、重复提交保护，保留上下移；同步覆盖 P17。继续在本交付 worktree 修改 categories.js/app.css，预览 worktree 仅同步文件。node --check 与 git diff --check 通过；遵从停测要求未执行测试/回归/Gate，浏览器实际 UAT 待完成。
