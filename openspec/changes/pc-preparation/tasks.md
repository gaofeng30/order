# PC 备餐管理

- 基线：ac123661f4971b9701cc4bc407ea5eefb7f47abb；独立分支 feature/pc-preparation。
- Owned paths：apps/web-admin/{app.js,app.css,index.html,data/api.js,pages/preparation.js}、fulfillment/{handler.go,admin_handler.go}、API 两个启动入口、tools/pc-admin-preview.cjs、三端与 PC 清单及本工件。
- 依赖：复用当前已确认订单状态与 OWNER PC 登录；需部署新增 PC API 入口，不更改 SQL。
- [x] 核对现状：PC 无备餐页面和 ready 入口；手机已有单笔备好，PC 登录凭据不能直接用于手机接口。
- [x] 实现日期餐段、完整分页、只读菜品汇总和固定范围批量备好。
- [x] 复用 fulfillment 命令，仅抽出传入已鉴权 actor 的共享入口；不另建状态机。
- [x] 本地预览适配上游新增 /pc/ 资源基址，支持根地址与 /pc/ 入口。
- [x] 更新已确认清单，不变更相邻功能。
- [x] `GOCACHE=/tmp/order-pc-preparation-go-cache go build -o /tmp/order-pc-preparation-api ./cmd/order-api` 退出 0；Go 默认缓存不可写后改用临时缓存。`node --check`（preparation.js、data/api.js、app.js、pc-admin-preview.cjs）与 `git diff --check` 通过。
- [ ] 浏览器响应式及实际业务 UAT：CUA 返回 Chrome / iab 不可用；未核验视觉或真实业务写入，不登记为通过。
- 用户停测覆盖：未运行测试、回归、Gate，不将历史证据记为本轮 PASS。未获授权不推送或部署。
