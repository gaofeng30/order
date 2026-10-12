> 2026-10-10 当前执行约束：用户已明确停止全部测试与回归并删除测试用例；旧测试及 Gate 记录仅作为历史证据，未运行不能记为通过。本轮直接完成 UAT SQL 安装、服务构建部署及小程序预览交付；不等待旧测试门禁。实际数据库已安装 113 个函数并登记 38 条平台执行记录，既有业务数据与账本保持不变。

# Tasks

原 callContainer 1.1–2.1 已完成记录属于历史 snapshot；原 73 项及 SDK random 5 项低层结果保留在 2026-10-06 artifact，不代表当前 HTTPS 源码或真实 UI Gate。恢复 HEAD 942d26c1ce1da31b6ec7dea41c7827b65ad22499，candidate_sha: null。

- [x] 4.1 Red：复用原公开 app→session→apiClient seam，trial 未提供 wx.cloud 却要求固定 HTTPS 登录/读/写；`node --test --test-name-pattern='trial public login read and write' apps/wechat-miniprogram/tests/cloudrun-transport-ui0.test.js`，node exit1，1FAIL/0PASS/0SKIP，session error != ready；trial-https-red.tap。图片相邻 selector `media keep exact signed HTTPS` 随后 exit1、1FAIL，旧 CloudRun guard 不再提供约定 OBJECT_URL_INVALID；media-https-red.tap。
- [x] 4.2 Green：唯一精确 HTTPS 配置、ready 校验和单 wx.request 薄层；同主路径/图片 selector 各 exit0、1PASS。`node --test apps/wechat-miniprogram/tests/cloudrun-transport-ui0.test.js apps/wechat-miniprogram/tests/runtime-endpoint-ui0.test.js` exit0、57PASS/0FAIL/0SKIP；affected-https-green.tap。session-api-DTO/Bearer/key/显式重试/错误/版本签名与 fail-closed 均保持；只有 UI0 合成请求。
- [x] 4.3 Refactor：现有共享薄层保留模块名，无需新抽象、重命名 consumer 或范围外重构；重跑4.2同一命令 exit0、57PASS/0FAIL/0SKIP；affected-https-refactor.tap。受影响 JS 语法与 git diff --check、change strict 检查通过；SHA保护newIdempotencyKey/8caller/14await和AppID原配置。
- [x] 4.4 必要邻接：session non-ready 旧前提因 trial 已有 HTTPS 改变，未改 fixture 先 exit1（trial发生1次login）；session-fixture-red.tap。显式空配置后，`node --test --test-name-pattern='ready runtime endpoint exchanges|non-ready runtime endpoint|public storefront accepts|public object URLs reject|menu failure clears' apps/wechat-miniprogram/tests/session-ui0.test.js apps/wechat-miniprogram/tests/public-read-storefront-ui0.test.js apps/wechat-miniprogram/tests/reservation-menu-ui1.test.js` exit0、7PASS/0FAIL/0SKIP；https-adjacent-green.tap。原develop登录/本地图片/菜单失败显式恢复及release空配置不变。未重跑无变化8caller、RNG5 selector或原73全套。
- [x] 4.5 维护scope/design/spec/gates：当前唯一HTTPS路径、精确trialorigin、request原生能力；官方HTTP网关/小程序合法域名要求；恢复snapshot不是candidate。
- [ ] 2.2 实际Writer UI1/UI2与完整Writer Gate；BLOCKED_EXTERNAL，root独占IDE/browser，平台105迁移与无运行服务尚未恢复；本轮Node不记UI1/2。
- [ ] 3.1 满足Writer Gate后形成精确candidate SHA；当前null，本轮源码准备不提交候选/推送/上传。
- [ ] 3.2 root体验版/手机正常模式UI3全部主路径和相邻回归，绑定exact candidate、AppID/精确HTTPSorigin/后端环境，真实Code2Session/Bearer/最终业务结果；BLOCKED_EXTERNAL，用户已确认保存request合法域名仍不能替代运行。
- [ ] 3.3 独立clean detached exact-SHA只读验证及全部声明场景receipt；NOT_RUN，失败回到writer修复新SHA。
- [ ] 3.4 依赖满足、独验和UI3通过后按单独授权集成；NOT_RUN。

本轮决定性收据根：`/Users/vivix/.codex/visualizations/2026/10/10/01a12421-c305-7371-b534-45fdffd1b88a/mini-https-progress/`。测试仅使用合成值，无凭据或真实用户资料。外部资产owner为root/UAT owner与客户小程序管理员；恢复条件为平台迁移/服务恢复、正常域名校验下目标版本可运行、真实账号/设备及既有受控场景可验。
