> 2026-10-10 当前执行约束：用户已明确停止全部测试与回归并删除测试用例；旧测试及 Gate 记录仅作为历史证据，未运行不能记为通过。本轮直接完成 UAT SQL 安装、服务构建部署及小程序预览交付；不等待旧测试门禁。实际数据库已安装 113 个函数并登记 38 条平台执行记录，既有业务数据与账本保持不变。

# Enable Order UAT CloudRun credentials

Status: UAT_SCOPE_CONFIRMED / IMPLEMENTING. The user authorized UAT closeout, including its ordinary access, configuration, deployment and repairs, excluding domain ICP. No repeated stage authorization is required.

## Why

The exact UAT backend uses CVM instance-role credentials for both SSM and COS. A real CloudRun container cannot obtain that provider's credentials. The official CloudBase MCP implements environment API Key exchange through `/capi/credential`. Bounded UAT probes with the backend's native Go SDK versions confirmed temporary credential exchange and COS HeadBucket HTTP200. SSM returned ResourceUnavailable.NotPurchased; successful SSM access after activation remains unverified. Both diagnostic keys were deleted and their absence verified.

## What Changes

- Use the native `CLOUDBASE_APIKEY` and `CLOUDBASE_ENV_ID` only for UAT, exchanging temporary Tencent credentials at the fixed HTTPS CloudBase endpoint.
- Refresh cached credentials before expiry, coordinate concurrent refresh, and reject unavailable or invalid credentials without exposing provider data.
- Connect the same environment selection to the existing four-purpose SSM boundary and native COS adapter. Preserve atomic credential snapshots when COS signs a request or URL.
- Keep production's CVM role provider and all existing business, PostgreSQL TLS, secret and upload contracts.

## Gate and Ownership

- gate_type: W3; ui_level: UI0 for this backend credential slice. Full cloud UAT and mini UI3 remain separate requirements.
- owner/controller: 01a11042-3601-7850-a38f-974660ec7462; branch `codex/order-uat-cloudrun-credentials`; worktree `/Users/vivix/.codex/worktrees/order-uat-cloudrun-credentials/order`.
- base_sha: 53ad8bc655dce474da2f54280865a6e7dfa6cc40; candidate_sha: NOT_CREATED.
- owned paths: `services/api/internal/config/{cloudbase_credentials.go,cloudbase_credentials_test.go,config.go,config_test.go,tencent_ssm.go,wechatpay.go,redemption_token.go}`, `services/api/internal/objectstore/{cos_adapter.go,cos_adapter_test.go}`, `services/api/cmd/order-api/{main.go,cos_object_store.go,cos_object_store_test.go}`, and this change.
- read-only contracts: root AGENTS, quality gates, existing OpenSpec business contracts, Go SDK interfaces, existing PC/mini code and prior candidate evidence.
- dependencies: exact backend e95b8f01f6aa38871b87d4a0f2e5d8595be60141 and PC 53ad8bc655dce474da2f54280865a6e7dfa6cc40, independently verified locally but not integrated into main. Main integration remains dependent on their lifecycle.
- non-goals: production changes, permanent Tencent AK/SK, raw DB/AppSecret environment values, fake cloud providers, SQL-over-HTTP, relaxed TLS, new UI/business behavior, paid purchasing, real financial transactions, remote publication or main integration.
- external assets/owner/recovery: UAT cloud account and real WeChat/merchant assets remain customer-owned. PG native connection/TLS and SSM activation/secrets are pending. Preserve failed receipts and resume real cloud acceptance after these assets are available; do not mark the full UAT goal complete from local adapter checks.

## Validation

Use the public Tencent provider/credential interfaces and native SDK HTTP boundaries with isolated synthetic responses and a controlled clock. Record vertical Red/Green evidence for exchange, refresh/concurrency, failure/endpoint safety, environment routing and COS atomic signing. Run affected ordinary/race tests, offline vet/build, isolated HTTP smoke, formatting/diff and strict OpenSpec checks. Local candidate and clean detached exact-SHA independent verification are separate milestones. Real CloudRun/SSM/COS object and full UAT checks remain NOT_VERIFIED until actually executed.
