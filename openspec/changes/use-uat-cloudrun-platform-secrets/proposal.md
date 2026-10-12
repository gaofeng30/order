> 2026-10-10 当前执行约束：用户已明确停止全部测试与回归并删除测试用例；旧测试及 Gate 记录仅作为历史证据，未运行不能记为通过。本轮直接完成 UAT SQL 安装、服务构建部署及小程序预览交付；不等待旧测试门禁。实际数据库已安装 113 个函数并登记 38 条平台执行记录，既有业务数据与账本保持不变。

# Order UAT 使用 CloudRun 平台变量读取四用途材料

Status: APPROVED / IMPLEMENTING。2026-10-08 用户要求按已核实的现有 CloudBase 功能完成 UAT，并授权并行实现；此前 CloudBase/排除 SSM 的 UAT 选择适用于本切片，生产规范保持。

## Why

精确 base 候选将 UAT 四用途材料绑定到 SSM。当前 CloudRun 真容器 canary 已证明服务版本变量可覆盖镜像默认值、保留多行/中文 JSON，更新版本后读到新值；SSM 并不是 CloudBase 平台必需服务。仅将 UAT 的既有 SecretSource 换成固定平台变量，保留所有解析和真实供应商调用，是满足现有 UAT 目标的最小完整改动。

## What Changes

- UAT 四个 logical purpose 唯一映射到 `ORDER_DB_PASSWORD`、`ORDER_WECHAT_MINIPROGRAM_APP_SECRET`、`ORDER_WECHATPAY_API_V3`、`ORDER_REDEMPTION_TOKEN_KEY`；没有 provider selector、缺值回退或文件旁路。
- 数据库/AppSecret 配置及支付、核销公开 loader 复用当前严格解析。材料仅在进程启动/显式 loader 调用时读取，轮换须更新平台配置并启动新 revision。
- 本 scoped delta 只替换 `enable-uat-cloudrun-credentials` 候选中的 UAT SSM/no-business-env 来源与轮换约束；原候选及历史 exact-SHA receipt 不变，不扩展为本切片证明。
- 生产继续 CVM role + SSM。COS 的 CloudBase API Key 临时凭据交换、verified PostgreSQL TLS、DSN/永久腾讯凭据禁令与全部业务语义不变。

## Gate and Ownership

- gate_type: W3；ui_level_target/actual: UI0。本切片不触及 UI、SQL、支付交易/退款流程或生产部署。
- owner/writer: `/root/native_env_writer`；controller: `/root`。
- branch: `codex/order-uat-platform-secrets`；worktree: `/Users/vivix/.codex/worktrees/order-uat-platform-secrets/order`。
- base_sha: `bec51e6fe1017bcd04d471ea74f47d0dc1391168`；candidate_sha: 在提交后由仓库外 lifecycle receipt 记录，避免自引用 SHA。
- owned paths: `services/api/internal/config/{config.go,config_test.go,uat_boundary_test.go,tencent_ssm.go,cloudrun_secrets.go,cloudrun_secrets_test.go,cloudbase_credentials.go,cloudbase_credentials_test.go}` 与本 change。
- read-only contracts: AGENTS/CONTEXT、质量门禁、旧 CloudRun credential delta、支付/核销 parser、database/COS/三个 cmd 消费者、PC/mini 与生产部署。
- dependencies: 精确 base credential 候选，以及其 backend `e95b8f01f6aa38871b87d4a0f2e5d8595be60141`、PC `53ad8bc655dce474da2f54280865a6e7dfa6cc40`；尚未集成 main，不能凭依赖候选自动集成。
- non-goals: 新 PG/付费升级、SSM 激活、真实客户凭据收集、真实资金、SQL-over-HTTP、TLS 放宽、长期腾讯 AK/SK、生产变更、push/merge/archive。
- external assets: root 控制 UAT 真 CloudRun，真实材料仍归客户管理员。exact-SHA 合成四用途云 probe、native PG/TLS、真实支付/真机及全量 95/25 UAT 分别未运行；后几项不作为本纯配置候选的本地 Writer Gate，也不得由本地 PASS 冒充。

## Validation

现有公开 `Load`、`LoadCloudWeChatPayMaterial`、`LoadCloudRedemptionTokenMaterial` 是 Red/Green seam；真实 RSA 支付 client 与版本化 AES cipher 构造覆盖消费者。用全 HTTP 拦截证明材料读取不联系 SSM/identity endpoint。验证缺值、非法材料、取消、用途隔离、生产禁止 env、恢复/重复并发读取及日志脱敏；重跑 config/三个 cmd ordinary/race、database TLS focused、COS 回归、vet/build、隔离 development HTTP smoke、gofmt/diff/owned/secret 与 OpenSpec strict。

候选之后由不同 verifier 在干净 detached worktree 对完整 SHA 只读执行同一检查；root 再执行 exact-SHA、完整合成支付 JSON 形态的 CloudRun probe。真实 PG/资金/设备/全量 UAT 仍是后续独立外部验收，不能写 PASS。
