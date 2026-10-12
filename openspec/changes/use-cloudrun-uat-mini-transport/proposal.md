> 2026-10-10 当前执行约束：用户已明确停止全部测试与回归并删除测试用例；旧测试及 Gate 记录仅作为历史证据，未运行不能记为通过。本轮直接完成 UAT SQL 安装、服务构建部署及小程序预览交付；不等待旧测试门禁。实际数据库已安装 113 个函数并登记 38 条平台执行记录，既有业务数据与账本保持不变。

# 通过既有 HTTPS 网关访问 Order UAT

状态：IMPLEMENTING；2026-10-10 续接完成 HTTPS 源码准备，实际 UI0，Writer UI1/UI2、手机 UI3 未运行；candidate_sha: null。

## 问题与结果

体验版 trial 需要访问既有 Order UAT。恢复版本使用 wx.cloud.callContainer；root 在目标微信 IDE CloudBase v2.0.3 实际看到 UAT 环境，但选择入口提示“暂不支持选择PostgreSQL环境”，该接入前置不满足。固定 callContainer 是实现选择，可按同一业务契约改用既有 HTTPS 网关和 wx.request。

trial 唯一入口固定为 `https://order-uat-d1gtzpzj50615c300-1473835269.ap-shanghai.app.tcloudbase.com`。保持 wx.login → 原后端 Code2Session → Order bearer、所有 path/method/DTO/status、异步原生随机幂等键及显式重试策略。develop 保留原 localhost；release 空配置安全失败。无 callContainer 双轨、运行 fallback、CloudBase 前端认证或数据库 SDK。

## 范围与交付门

- gate_type: W2；目标 ui_level: UI3；实际 UI0/fake SDK，不代表 UI1/UI2/UI3。
- owner: `/root/mini_https_prepare`；branch: `codex/resume-mini-cloudrun-transport-20261010`。
- worktree: `/Users/vivix/.codex/worktrees/order-uat-mini-cloudrun-transport/order`。
- base_sha: `6cb0f05000eb2adf4382f94d6ec96292921b6062`，保护已有 UI 的 archive snapshot；恢复 HEAD `942d26c1ce1da31b6ec7dea41c7827b65ad22499` 也是 snapshot，均不是候选。
- candidate_sha: null；恢复时 worktree/index 为空；本轮不形成候选、不上传体验版。

本 change 原 owned paths：

- `apps/wechat-miniprogram/app.js`
- `apps/wechat-miniprogram/utils/runtimeEndpointConfig.js`
- `apps/wechat-miniprogram/utils/runtimeEndpoint.js`
- `apps/wechat-miniprogram/utils/sessionApi.js`
- `apps/wechat-miniprogram/utils/apiClient.js`
- `apps/wechat-miniprogram/utils/cloudrunTransport.js`
- `apps/wechat-miniprogram/utils/reservationMenuApi.js`
- `apps/wechat-miniprogram/tests/cloudrun-transport-ui0.test.js`
- `apps/wechat-miniprogram/tests/runtime-endpoint-ui0.test.js`
- `apps/wechat-miniprogram/tests/session-ui0.test.js`（本轮仅显式设置 non-ready 用例的空配置前提）
- `openspec/changes/use-cloudrun-uat-mini-transport/**`

本轮实际只改 endpoint config/validator、现有 transport 薄层、apiClient 的非 develop 相对图片保护、三个受影响测试和五份 change 工件。app/sessionApi/reservationMenuApi 已共享薄层，无需再改。所有业务页、8 个原随机键 caller、14 个 await 调用点与 newIdempotencyKey 函数只读保护。根主控独占 IDE、浏览器和云操作；其他 change 使用各自 worktree。

## 依赖与非目标

复用既有 UAT 网关到 order-api 的 HTTP 路由、有效 TLS、固定 AppID `wx15e136b256d4dc5a` 的 request 合法域名、真实 Code2Session 和后端核心链路。用户本轮确认已保存该 request 合法域名；正式手机正常模式仍须实测。root 的公开网关 HEAD 得到 404/cbrgw，只证明网关 TLS/响应存在，不能证明已运行业务 API。

root 本轮确认云服务仍无运行实例，migration 仍被平台 InternalError/code 105 阻断；本文不将客户端源码 Green 解释为后端可用。此 change 不修改云资源、后台权限、微信配置、资金或支付额度。

Writer 实际 UI1/UI2 与完整 Writer Gate 后才能形成 candidate；exact SHA 独立验证和 manifest 全部手机 UI3 场景仍是独验/集成前置。当前 UI Gate 为 BLOCKED_EXTERNAL：root 负责恢复后端服务、目标账号/设备和受控场景后续接；Node 不能替代这些门。未经单独授权不推送、集成、部署或上传体验版。
