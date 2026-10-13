# 三端 UAT 汇总提交记录

- 日期：2026-10-13；用户明确授权将当前小程序用户端、商户端与 PC 端改动汇总为一个 UAT 提交并推送，本次明确允许 HTTPS。
- 远端核对：HTTPS fetch 得到 UAT 基线 `b56c5b78b916096783a84a20fb92cadcba5532fd`；不修改 origin 持久化配置，不强制推送。
- 来源：PC `5081cbc`；小程序 `27e3387` 加当前未提交的生产文件、Logo 和 cart-confirm 页面。汇总前后逐文件内容核对，共 161 条来源路径一致。
- 范围：三端界面/交互、分餐段购物车与逐单支付、PC 批量操作、导入前编辑补图及结果查询、删除菜品售罄关联清理、需求与实施文档。
- 按本轮停测指令保留 PC 分支的测试资产删除，不带入小程序分支新增的测试代码、测试 runner、红绿记录与 Gate 工件。小程序清单中的历史测试结论仅属历史，不能视为本提交的验证证据。
- 排除：根目录与小程序工作区 project.config.json 仅末尾换行变化、无关客户资料/输出/本地工具与模型配置；原开发工作区未改写。
- 本次检查：49 个修改/新增 JavaScript 文件通过 node --check；小程序 JSON 解析成功；git diff --check 无误；go build -o /tmp/order-three-client-uat-api ./services/api/cmd/order-api 成功。
- 未执行：自动化测试、回归、Gate、微信上传、部署、真实支付、导入或删除业务数据。未宣称完整多尺寸与真机 UAT 通过。
- 待办仍以 three-client-backend-update-checklist.md 为准：首页地址字段绑定待修改，B1–B5 中待对齐项未自行实现；本次推送不代表后端或 SharedPG 函数已经发布。
