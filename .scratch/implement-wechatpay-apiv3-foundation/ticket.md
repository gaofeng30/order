# implement-wechatpay-apiv3-foundation / TX-00

## 状态与目标

- 状态：`APPROVED`（本次 delegated user instruction 明确授权实现 TX-00）。
- 单一目标：在 `services/api/internal/wechatpay/**` 建立不含订单/预支付业务语义的纯微信支付 APIv3 foundation，使调用方可用本地受控输入完成请求签名、微信原始应答/通知验签、通知资源解密，以及五类 typed provider operations。
- tracker：`BLOCKED_LOCAL_GOVERNANCE`。仓库缺少 `docs/agents/issue-tracker.md`，因此不创建或伪造外部 Issue；本文件是本 change 的唯一 Spec/ticket。

## Gate 声明

- `gate_type`: `W3`（支付/退款边界，取最高风险）。
- `ui_level_target`: `UI0`。
- `ui_level_actual`: `UI0`（backend-only；只做静态/本地可执行证据，不宣称 UI 或真实微信联调）。
- owner: TX-00 independent writer。
- branch: `codex/implement-wechatpay-apiv3-foundation`。
- worktree: `/Users/vivix/.codex/worktrees/1bc5/order`。
- `base_sha`: `a56aeec6b041bf1a31988a888956ad851e128469`。
- frozen contract exact SHA: `02562fcd9c4f66a74375d505fb3343c26f06f38e`；四个只读工件已逐 blob 核验一致。
- `candidate_sha`: 形成不可变提交后在外部证据中记录；提交不能包含自身 SHA。
- dependencies: exact code base 与 frozen TX-00 contract；无 migration predecessor，无产品未决项依赖。
- owned paths:
  - `.scratch/implement-wechatpay-apiv3-foundation/**`
  - `services/api/internal/wechatpay/**`
- read-only shared contracts:
  - `AGENTS.md`
  - `CONTEXT.md`
  - `docs/quality/change-quality-gates.md`
  - frozen contract worktree `/Users/vivix/.codex/worktrees/851f/order/.scratch/freeze-transaction-payment-contract/**`
- non-goals: migration、HTTP ingress、cmd/composition/config、app/page、package files、Order/Prepayment DTO、TTL/P2/P4/P5、真实微信连接、push/PR/deploy/main integration。

## 已确认的公开测试 seams

本次委托已固定 seam，无需追加产品决策：

1. 签名 seam：调用方给出 method、canonical request target、原始 body、固定 clock/nonce 和测试 RSA key，观察 canonical message、`WECHATPAY2-SHA256-RSA2048` Authorization 与小程序 `wx.requestPayment` 参数。
2. 信任 seam：调用方给出原始 response/callback body 与完整 Wechatpay headers，按 serial 选择公钥并观察验签成功或 unknown serial、超窗、篡改、错误签名的安全失败。
3. 通知 seam：调用方提交已验签的官方字段格式严格 envelope，使用 32-byte test key 解密 `AEAD_AES_256_GCM` resource；额外/缺失/类型错误、AAD/ciphertext 错误均失败。
4. provider seam：调用方通过 typed operations 发起 JSAPI create、按商户单号/微信单号 query、close、refund create、refund query；本地 loopback server 只充当外部微信边界，响应必须先验签再解析。
5. 错误 seam：调用方只观察稳定错误 kind/status/provider code/retryability，不接收 raw body、签名、密钥、OpenID 或 provider message。

## 最小公共契约

- Go 标准库 RSA-SHA256、AES-256-GCM 和 JSON；不新增依赖。
- runtime origin 固定为 `https://api.mch.weixin.qq.com`，HTTP timeout 有界且拒绝 redirect；仅包内测试构造器可注入 loopback origin、clock、nonce、HTTP client 和公钥集合。
- 原始请求 body 直接进入五行签名串；原始响应/通知 body 直接进入三行验签串，解析前必须验签。
- callback envelope/resource 严格拒绝 unknown/duplicate/trailing/missing/wrong-type 字段。
- typed operations 只表达微信 provider DTO；不得定义订单、预支付记录、幂等持久化或交易状态机 DTO。
- 所有错误文本脱敏；完整 provider body、回调明文、签名、APIv3 key、私钥和 OpenID 不进入错误或证据。

## Writer / verifier / integration DoD

- Writer：真实 Red→Green→Refactor；focused/full/race/vet/build/diff/owned/sensitive scan 全通过；只提交 owned paths；中文完整 commit；worktree/index clean。
- Review：两个独立 reviewer 对同一 exact SHA 分别做 Standards 与 frozen TX-00 Spec 审查；任一 finding 由 writer 修复形成新 SHA 后，两轴从头重审。
- Verifier：在另一个新建、干净、detached worktree 对 exact candidate SHA 只读重跑全部声明 Gate、owned audit 与敏感扫描。
- Integration：本次不做；只有独立 PASS 且依赖进入目标 main 后，由获单独授权的 integration owner 处理。不得由 TX-00 修改 composition/main。

## 外部资产矩阵

| 资产 | owner | 当前状态 | 恢复条件 |
| --- | --- | --- | --- |
| 正式 AppID 与 mchid 绑定/产品权限 | 客户商户管理员 + 开发方 | `BLOCKED_EXTERNAL` | 后台绑定与权限脱敏实证 |
| 商户 API 证书/私钥/serial 安全托管 | 客户商户管理员 + 开发方 | `BLOCKED_EXTERNAL` | 有效材料配对与托管实证，不输出 PEM |
| 微信支付公钥/平台证书与轮换 | 客户商户管理员 + 开发方 | `BLOCKED_EXTERNAL` | 当前 serial 验签与轮换实测 |
| 32-byte APIv3 key 安全托管 | 客户商户管理员 + 开发方 | `BLOCKED_EXTERNAL` | 正式 key 已设置并能解密受控回调，不输出 key |
| 公网 HTTPS notify URL 原始 body/header 保真 | 开发方/平台 owner | `BLOCKED_EXTERNAL` | TLS、外网、无登录态、代理保真实测 |
| 查单/关单/退款连通与账单权限 | 客户商户管理员 + 开发方 | `BLOCKED_EXTERNAL` | 同一受控交易完成已验签 operations/对账 |
| 真金白银 UAT | UAT owner + 客户商户管理员 | `BLOCKED_EXTERNAL` | 授权账号、最小金额、精确脱敏标识与退款最终态闭环 |

## 可执行 RGR 与最终 Gate

- Red/Green/Refactor focused：`GOPROXY=off GOTOOLCHAIN=go1.26.5 go test ./services/api/internal/wechatpay -run '<当前 tracer test>' -count=1`；同一 tracer 在实现前 FAIL、最小实现后 PASS，Refactor 后原命令再 PASS。
- focused all：`GOPROXY=off GOTOOLCHAIN=go1.26.5 go test ./services/api/internal/wechatpay -count=1`。
- race：`GOPROXY=off GOTOOLCHAIN=go1.26.5 go test -race ./services/api/internal/wechatpay -count=1` 与 `./services/api/...`。
- full：`GOPROXY=off GOTOOLCHAIN=go1.26.5 go test ./services/api/... -count=1`。
- static/build：`GOPROXY=off GOTOOLCHAIN=go1.26.5 go vet ./services/api/...`；`GOPROXY=off GOTOOLCHAIN=go1.26.5 go build ./services/api/...`；`test -z "$(gofmt -l services/api/internal/wechatpay)"`。
- repository smoke：`GOPROXY=off GOTOOLCHAIN=go1.26.5 bash services/api/scripts/smoke.sh`。
- diff/ownership：`git diff --check a56aeec6b041bf1a31988a888956ad851e128469...HEAD`；`git diff --name-only a56aeec6b041bf1a31988a888956ad851e128469...HEAD` 必须只命中 owned paths。
- sensitive scan：只扫描 owned diff/文件中的 secret/certificate/key/header/raw callback/OpenID canary；结果只能是文件/规则摘要，不输出匹配到的敏感值。
- change-specific OpenSpec：`N/A`，因为本次 owned paths 明确禁止 `openspec/**`，且本 change 使用用户要求的 scratch 单一 Spec/ticket；不得伪造 PASS。
