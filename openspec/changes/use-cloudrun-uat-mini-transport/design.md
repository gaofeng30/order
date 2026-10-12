> 2026-10-10 当前执行约束：用户已明确停止全部测试与回归并删除测试用例；旧测试及 Gate 记录仅作为历史证据，未运行不能记为通过。本轮直接完成 UAT SQL 安装、服务构建部署及小程序预览交付；不等待旧测试门禁。实际数据库已安装 113 个函数并登记 38 条平台执行记录，既有业务数据与账本保持不变。

# 具体传输实现

trial runtimeEndpoint 的唯一形态为 `{state:'ready',envVersion:'trial',origin:'https://order-uat-d1gtzpzj50615c300-1473835269.ap-shanghai.app.tcloudbase.com',errorCode:''}`，apiBaseUrl 与 origin 相等。配置与消费者 validator 都要求这个精确归一化 HTTPS origin；旧 tuple、另一 HTTPS 域名、HTTP、路径/query/fragment/userinfo 和 loopback 均拒绝。无需 wx.cloud。release 必须显式具备合法 HTTPS 配置；当前为空，绝不自动借用 UAT。develop 继续原严格 127.0.0.1 HTTP origin。

保留现有 `cloudrunTransport.requestRuntime` 模块名和 session/API/预约菜单三个消费者的 import；模块现在只有 `wx.request({...options,url:origin+path})`。它校验完整 ready endpoint；原 options/callback 原样传递。没有一次初始化状态、callContainer、替代域名、service/env 选择、自动写重放、新 timeout 或 CloudBase JWT/前端数据库身份。

session 仍严格 201 和 access_token/expires_at/token_type=Bearer 三字段；API仍只成功200/201/202/204，并保留 response.data或{}、业务 error.code/statusCode、网络 API_UNAVAILABLE。401、网络与同步 wx.request 异常沿原错误映射，session 失败清空原凭据。Authorization/Idempotency-Key/JSON header 不变，旧菜单 query 编码与 200/400 INVALID_MENU_SELECTION 不变，读取等待原 sessionPromise。

相对 `/api/v1/objects/` 只在 develop 允许；trial/release 都拒绝，避免 HTTPS 形态让旧 isCloudRunEndpoint 判断失效。后端 COS 签名 HTTPS URL 仍逐字返回，versionId/query 不解析重建，develop 相对对象路径保留。

## 路线依据与实际边界

[CloudBase HTTP 网关官方说明](https://docs.cloudbase.net/service/introduce)确认标准 HTTP/HTTPS 可访问环境云资源并关联 CloudRun，无需复杂 SDK；默认域名适用于开发测试。此 change 只用于 UAT，不配置生产域名。官方[小程序认证排错](https://docs.cloudbase.net/recipes/fix-auth-wechat-miniprogram)说明 HTTP 接口的 request 合法域名需要在小程序后台配置；本文仅采用域名要求，不引入该文自定义登录方案。

本轮 root 在 IDE 看到 PostgreSQL 环境选择不支持；用户确认 request 域名已保存。公开 HEAD 404/cbrgw 不等于业务路由已就绪。真正入口与业务结果由 root 以目标 AppID、体验版和手机正常模式 UI3 实测；平台105迁移/服务尚未恢复，当前 BLOCKED_EXTERNAL。

## 范围内验证

HTTPS app→session→读/写、菜单 query、状态/错误、登录等待、同 key 显式重试、SDK 随机等待/拒绝及图片原字节由受影响测试观察 wx.request 边界。仅重验改变的 transport/runtime 和必要 develop/session/媒体/菜单邻接；原 73 项及 SDK random 5 项是历史低层证据，当前 HTTPS 改动不沿用其完整 PASS。newIdempotencyKey 与 8 caller/14 await 未变，按 SHA/逐字保护复用其未变部分，不重复无变化 RNG selector，不扩展95项/25页/Prod。薄层原位置足以完成此切片，无范围外 refactor。
