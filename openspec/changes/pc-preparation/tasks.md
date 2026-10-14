# PC 备餐管理

- 基线：ac123661f4971b9701cc4bc407ea5eefb7f47abb；独立分支 feature/pc-preparation。
- Owned paths：apps/web-admin/{app.js,app.css,index.html,data/api.js,pages/preparation.js,pages/orders.js,pages/dashboard.js}、fulfillment/{handler.go,admin_handler.go}、API 两个启动入口、tools/pc-admin-preview.cjs、三端与 PC 清单及本工件。
- 依赖：复用当前已确认订单状态与 OWNER PC 登录；需部署新增 PC API 入口，不更改 SQL。
- [x] 核对现状：PC 无备餐页面和 ready 入口；手机已有单笔备好，PC 登录凭据不能直接用于手机接口。
- [x] 实现日期餐段、完整分页、只读菜品汇总和固定范围批量备好。
- [x] 复用 fulfillment 命令，仅抽出传入已鉴权 actor 的共享入口；不另建状态机。
- [x] 本地预览适配上游新增 /pc/ 资源基址，支持根地址与 /pc/ 入口。
- [x] 更新已确认清单，不变更相邻功能。
- [x] `GOCACHE=/tmp/order-pc-preparation-go-cache go build -o /tmp/order-pc-preparation-api ./cmd/order-api` 退出 0；Go 默认缓存不可写后改用临时缓存。`node --check`（preparation.js、data/api.js、app.js、pc-admin-preview.cjs）与 `git diff --check` 通过。
- [ ] 浏览器响应式及实际业务 UAT：CUA 返回 Chrome / iab 不可用；未核验视觉或真实业务写入，不登记为通过。
- 用户停测覆盖：未运行测试、回归、Gate，不将历史证据记为本轮 PASS。未获授权不推送或部署。

## 追加切片：A1–A7 与体验 3/4/8/10

- [x] A1/A2：订单分页、已显示数量、加载/失败/原位重试、筛选响应隔离；详情读取当前页面已加载集合，不再依赖其他页面共享订单缓存。
- [x] A3/A4：工作台携带具体订单信息跳转；待制作转到当天备餐。
- [x] A5/A6/A7：会话内筛选记忆、跨页批量进度及结果、统计口径文案；退出登录清理。
- [x] 体验4/8/10：按取餐时间分组，成功更新时间与手动刷新，结果留存和失败项定位。
- [x] 体验3：只突出备注颜色，取消原先按备注拆分汇总；其余明确拒绝建议不做。
- [x] 修改 JS 的 node --check、git diff --check；只改前端与文档，无需重复 Go 编译。未运行测试/回归/Gate。
- [ ] 浏览器视觉及响应式：精确 localhost 标签绑定未找到；自动审批拒绝枚举 Chrome 标签，原因是暴露无关浏览元数据。未绕过该限制，未进行 UI 或业务验收。
