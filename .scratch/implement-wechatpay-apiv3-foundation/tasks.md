# implement-wechatpay-apiv3-foundation evidence

## Checklist

- [x] 核验 exact base、merge-base、适用规则、质量门禁、`$tdd` 与 `$code-review`。
- [x] 逐 blob 核验 frozen contract 四个工件来自 exact SHA `02562fcd9c4f66a74375d505fb3343c26f06f38e`。
- [x] 声明 W3/UI0、依赖、owned/read-only paths、外部资产、RGR 命令与三阶段 DoD。
- [x] Red/Green：请求 canonical message、APIv3 Authorization、小程序支付参数签名。
- [x] Red/Green：原始 body/header/serial 验签与 unknown serial/时间窗/篡改失败。
- [x] Red/Green：callback 严格 envelope/resource 与 AES-256-GCM 解密失败语义。
- [x] Red/Green：五类 typed operations 与响应先验签后解析。
- [x] Red/Green：非 2xx、签名/解密、timeout、429、5xx 稳定类型化错误。
- [x] Refactor：收窄导出面并重跑同一 focused/full/race/vet/build/smoke/scan。
- [ ] 只提交 owned paths 的中文完整 commit，记录 exact candidate SHA。
- [ ] 两个独立 reviewer 对同一 SHA 完成 Standards/Spec 双 PASS。
- [ ] 另一 clean detached worktree 对 exact SHA 只读验证 PASS。

## Evidence

只在命令/动作实际发生后追加脱敏证据。`candidate_sha` 在不可变提交前为 `not-yet-created`；提交内不能记录自身 SHA。

```yaml
change: implement-wechatpay-apiv3-foundation
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: a56aeec6b041bf1a31988a888956ad851e128469
candidate_sha: not-yet-created
phase: writer
command_or_action: exact-base and frozen-contract provenance audit
exit_result: PASS
sanitized_summary: HEAD, delivery branch, and merge-base matched exact base; all four read-only contract blobs matched exact SHA 02562fcd
artifact_or_environment: local writer worktree and read-only frozen contract worktree
unverified_boundary: no implementation, real WeChat asset, provider network, payment, refund, callback, UI, push, deploy, or integration ran
external_asset:
  owner: customer merchant administrator and development/UAT owners
  missing: all formal WeChat Pay assets listed in ticket.md
  recovery: provision and authorize assets, then run a separately authorized controlled real-WeChat gate
```

```yaml
change: implement-wechatpay-apiv3-foundation
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: a56aeec6b041bf1a31988a888956ad851e128469
candidate_sha: not-yet-created
phase: red
command_or_action: GOPROXY=off GOTOOLCHAIN=go1.26.5 go test ./services/api/internal/wechatpay -run '^TestCreateRefundRequiresOneTransactionIdentifier$' -count=1
exit_result: exit-1
sanitized_summary: refund create reached the provider when transaction_id/out_trade_no were both absent or both present
artifact_or_environment: two invalid typed refund requests and a call-counting loopback boundary
unverified_boundary: local one-of validation was missing
external_asset:
  owner: N/A
  missing: N/A
  recovery: enforce the frozen provider one-of rule and rerun the identical focused command
```

```yaml
change: implement-wechatpay-apiv3-foundation
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: a56aeec6b041bf1a31988a888956ad851e128469
candidate_sha: not-yet-created
phase: red
command_or_action: GOPROXY=off GOTOOLCHAIN=go1.26.5 go test ./services/api/internal/wechatpay -run '^TestTypedOperationsUseOfficialEndpointsAndVerifiedResponses$' -count=1
exit_result: exit-1
sanitized_summary: the verified refund DTO lacked success_time, so a SUCCESS query could not carry the provider terminal timestamp
artifact_or_environment: distinct PROCESSING create response and SUCCESS query response fixtures
unverified_boundary: terminal refund timestamp was absent from the typed operation result
external_asset:
  owner: N/A
  missing: N/A
  recovery: add the single typed success-time field and rerun the identical operation test
```

```yaml
change: implement-wechatpay-apiv3-foundation
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: a56aeec6b041bf1a31988a888956ad851e128469
candidate_sha: not-yet-created
phase: red
command_or_action: GOPROXY=off GOTOOLCHAIN=go1.26.5 go test ./services/api/internal/wechatpay -run '^TestRuntimeClientUsesFixedBoundedNonReplayingTransport$' -count=1
exit_result: exit-1
sanitized_summary: runtime client used the default nil Transport instead of a dedicated fresh HTTP/1 transport for signed requests
artifact_or_environment: public NewClient construction with only synthetic local key material
unverified_boundary: runtime transport could not prove fresh connections or disabled HTTP/2 negotiation
external_asset:
  owner: N/A
  missing: N/A
  recovery: install the minimal dedicated transport and rerun the identical focused command
```

```yaml
change: implement-wechatpay-apiv3-foundation
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: a56aeec6b041bf1a31988a888956ad851e128469
candidate_sha: not-yet-created
phase: green
command_or_action: GOPROXY=off GOTOOLCHAIN=go1.26.5 go test ./services/api/internal/wechatpay -run '^TestClientClassifiesProviderFailures$' -count=1
exit_result: exit-0
sanitized_summary: timeout, 429, 503, and terminal 400 mapped to distinct stable kinds/status/provider-code/retryability without provider message/body leakage
artifact_or_environment: bounded local HTTP timeout and four synthetic provider outcomes
unverified_boundary: no real provider throttling, outage, retry scheduler, or business terminal-state behavior
external_asset:
  owner: development/UAT owners
  missing: authorized real provider connectivity and controlled failure gate
  recovery: execute separately authorized provider failure/UAT scenarios after formal assets exist
```

```yaml
change: implement-wechatpay-apiv3-foundation
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: a56aeec6b041bf1a31988a888956ad851e128469
candidate_sha: not-yet-created
phase: red
command_or_action: GOPROXY=off GOTOOLCHAIN=go1.26.5 go test ./services/api/internal/wechatpay -run '^TestClientClassifiesProviderFailures$' -count=1
exit_result: exit-1
sanitized_summary: build failed because timeout, rate-limit, and provider-unavailable stable error kinds did not exist
artifact_or_environment: bounded loopback timeout plus sanitized 400/429/503 provider envelopes
unverified_boundary: callers could not distinguish required retry and terminal HTTP failure categories
external_asset:
  owner: N/A
  missing: N/A
  recovery: add only the missing stable categories and safe provider-code parsing, then rerun the identical command
```

```yaml
change: implement-wechatpay-apiv3-foundation
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: a56aeec6b041bf1a31988a888956ad851e128469
candidate_sha: not-yet-created
phase: green
command_or_action: GOPROXY=off GOTOOLCHAIN=go1.26.5 go test ./services/api/internal/wechatpay -run '^TestTypedOperationsUseOfficialEndpointsAndVerifiedResponses$' -count=1
exit_result: exit-0
sanitized_summary: both transaction query variants, close, refund create, and refund query used exact signed method/target/body contracts and parsed only verified responses
artifact_or_environment: five loopback provider operations with fixed clock/nonce and signed synthetic transaction/refund responses
unverified_boundary: no real provider connectivity, account permission, transaction state, refund funds, or business persistence
external_asset:
  owner: customer merchant administrator and development/UAT owners
  missing: formal query/close/refund permissions, connectivity, and controlled transaction
  recovery: authorize assets and run separately controlled real-WeChat operation gates
```

```yaml
change: implement-wechatpay-apiv3-foundation
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: a56aeec6b041bf1a31988a888956ad851e128469
candidate_sha: not-yet-created
phase: red
command_or_action: GOPROXY=off GOTOOLCHAIN=go1.26.5 go test ./services/api/internal/wechatpay -run '^TestTypedOperationsUseOfficialEndpointsAndVerifiedResponses$' -count=1
exit_result: exit-1
sanitized_summary: build failed because both transaction queries, close, refund create/query, and refund request DTOs did not exist
artifact_or_environment: five official endpoint/method/query/body fixtures with signed synthetic responses
unverified_boundary: no typed query, close, or refund operation behavior existed
external_asset:
  owner: N/A
  missing: N/A
  recovery: implement only the five missing provider operations and rerun the identical focused command
```

```yaml
change: implement-wechatpay-apiv3-foundation
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: a56aeec6b041bf1a31988a888956ad851e128469
candidate_sha: not-yet-created
phase: green
command_or_action: GOPROXY=off GOTOOLCHAIN=go1.26.5 go test ./services/api/internal/wechatpay -run '^TestParseTransactionNotificationVerifiesDecryptsAndRejectsInvalidDTOs$' -count=1
exit_result: exit-0
sanitized_summary: signed raw callback verified before parsing; AES-256-GCM decrypted with AAD; unknown/duplicate/missing DTO fields and wrong AAD failed closed without payload leakage
artifact_or_environment: generated RSA keys, 32-byte synthetic key, fixed callback clock/nonce, and official-format transaction fixture
unverified_boundary: local fixture does not prove formal key custody, proxy header/body preservation, public HTTPS callback, durable inbox, or real payment
external_asset:
  owner: customer merchant administrator and platform/development owners
  missing: formal trust/key material and public HTTPS callback path
  recovery: provision assets and execute separately authorized callback/UAT gates
```

```yaml
change: implement-wechatpay-apiv3-foundation
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: a56aeec6b041bf1a31988a888956ad851e128469
candidate_sha: not-yet-created
phase: red
command_or_action: GOPROXY=off GOTOOLCHAIN=go1.26.5 go test ./services/api/internal/wechatpay -run '^TestParseTransactionNotificationVerifiesDecryptsAndRejectsInvalidDTOs$' -count=1
exit_result: exit-1
sanitized_summary: build failed because ParseTransactionNotification, SignatureHeaders, and the decryption error kind did not exist
artifact_or_environment: signed strict-envelope fixtures with local AES-256-GCM data and generated provider RSA key
unverified_boundary: no callback verification, decryption, or strict resource DTO behavior existed
external_asset:
  owner: N/A
  missing: N/A
  recovery: implement the minimal callback verifier/decrypter/DTO slice and rerun the identical focused command
```

```yaml
change: implement-wechatpay-apiv3-foundation
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: a56aeec6b041bf1a31988a888956ad851e128469
candidate_sha: not-yet-created
phase: green
command_or_action: GOPROXY=off GOTOOLCHAIN=go1.26.5 go test ./services/api/internal/wechatpay -run '^TestCreateJSAPIPrepayRejectsUntrustedResponses$' -count=1
exit_result: exit-0
sanitized_summary: unknown serial, timestamps more than five minutes stale/future, and raw-body tampering all failed closed with stable retryable trust errors and no fixture leakage
artifact_or_environment: signed loopback responses using one generated provider key, fixed clock, and synthetic payloads
unverified_boundary: local cryptographic fixtures only; no key rotation fetch, proxy, real response, or provider network
external_asset:
  owner: customer merchant administrator and development owners
  missing: formal provider public key/certificate and rotation evidence
  recovery: provision current trust material and run an authorized provider response gate
```

```yaml
change: implement-wechatpay-apiv3-foundation
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: a56aeec6b041bf1a31988a888956ad851e128469
candidate_sha: not-yet-created
phase: red
command_or_action: GOPROXY=off GOTOOLCHAIN=go1.26.5 go test ./services/api/internal/wechatpay -run '^TestCreateJSAPIPrepayRejectsUntrustedResponses$' -count=1
exit_result: exit-1
sanitized_summary: build failed because the typed Error did not expose the required stable Retryable decision
artifact_or_environment: unknown-serial, stale/future timestamp, and tampered-raw-body provider fixtures
unverified_boundary: callers could not distinguish retryable trust failures through the public error seam
external_asset:
  owner: N/A
  missing: N/A
  recovery: add the minimal stable retryability method and rerun the identical focused command
```

```yaml
change: implement-wechatpay-apiv3-foundation
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: a56aeec6b041bf1a31988a888956ad851e128469
candidate_sha: not-yet-created
phase: green
command_or_action: GOPROXY=off GOTOOLCHAIN=go1.26.5 go test ./services/api/internal/wechatpay -run '^TestCreateJSAPIPrepaySignsRequestAndRequestPayment$' -count=1
exit_result: exit-0
sanitized_summary: the exact official-format request body, five-line canonical signature, Authorization metadata, verified raw response, and four-line requestPayment signature passed
artifact_or_environment: loopback provider with locally generated 2048-bit merchant/provider RSA keys and fixed clock/nonces
unverified_boundary: local fixture only; no real merchant material, provider network, callback, payment, refund, or platform result
external_asset:
  owner: customer merchant administrator and development/UAT owners
  missing: formal WeChat Pay assets and authorized real transaction
  recovery: run a separate controlled external gate after assets and authority exist
```

```yaml
change: implement-wechatpay-apiv3-foundation
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: a56aeec6b041bf1a31988a888956ad851e128469
candidate_sha: not-yet-created
phase: red
command_or_action: GOPROXY=off GOTOOLCHAIN=go1.26.5 go test ./services/api/internal/wechatpay -run '^TestCreateJSAPIPrepaySignsRequestAndRequestPayment$' -count=1
exit_result: exit-1
sanitized_summary: build failed because newClient, Config, JSAPICreateRequest, Amount, and Payer did not exist
artifact_or_environment: exact-base writer worktree with the first public-seam tracer test only
unverified_boundary: no APIv3 request, response verification, or requestPayment behavior existed
external_asset:
  owner: N/A
  missing: N/A
  recovery: implement the minimal local JSAPI signing slice and rerun the identical focused command
```

```yaml
change: implement-wechatpay-apiv3-foundation
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: a56aeec6b041bf1a31988a888956ad851e128469
candidate_sha: not-yet-created
phase: writer
command_or_action: issue tracker discovery
exit_result: BLOCKED_LOCAL_GOVERNANCE
sanitized_summary: docs/agents/issue-tracker.md is absent; no external Issue was created or inferred
artifact_or_environment: repository governance files at exact base
unverified_boundary: external tracker state and linkage
external_asset:
  owner: repository governance owner
  missing: issue-tracker workflow document and configured tracker
  recovery: establish the repository tracker workflow in a separate authorized change
```

```yaml
change: implement-wechatpay-apiv3-foundation
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: a56aeec6b041bf1a31988a888956ad851e128469
candidate_sha: not-yet-created
phase: green
command_or_action: focused runtime transport, refund SUCCESS timestamp, and refund identifier tests
exit_result: exit-0
sanitized_summary: runtime origin/timeout/redirect/fresh-HTTP1 policy passed; verified refund SUCCESS carried success_time; invalid refund selector combinations made zero provider calls
artifact_or_environment: public NewClient plus loopback signed provider fixtures
unverified_boundary: no real transport, transaction, refund, or external provider result
external_asset:
  owner: customer merchant administrator and development/UAT owners
  missing: formal WeChat Pay runtime assets and authorized transaction
  recovery: run separately authorized external gates after assets exist
```

```yaml
change: implement-wechatpay-apiv3-foundation
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: a56aeec6b041bf1a31988a888956ad851e128469
candidate_sha: not-yet-created
phase: refactor
command_or_action: rerun each of seven exact focused tracer tests, then focused package test and go doc export audit
exit_result: exit-0
sanitized_summary: all seven original tracer commands passed after splitting client, errors, signatures, strict JSON, JSAPI, callback, and other operations; no signer/verifier/decrypter/clock/nonce/transport abstraction is exported
artifact_or_environment: final local wechatpay package using only the Go standard library
unverified_boundary: no real WeChat, UI, persistence, idempotency, transaction, push, deploy, or integration evidence
external_asset:
  owner: N/A
  missing: N/A
  recovery: rerun the listed focused commands from the exact candidate SHA
```

```yaml
change: implement-wechatpay-apiv3-foundation
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: a56aeec6b041bf1a31988a888956ad851e128469
candidate_sha: not-yet-created
phase: writer
command_or_action: focused test/race/format; full services/api test/race/vet/build; repository smoke
exit_result: PASS
sanitized_summary: focused package and race passed; all services/api packages passed test and race; vet/build were clean; smoke reported PASS
artifact_or_environment: final product-code tree before candidate commit
unverified_boundary: exact-SHA review and clean detached verification remain pending; local fixtures do not prove any real WeChat asset or funds result
external_asset:
  owner: customer merchant administrator and development/UAT owners
  missing: all formal WeChat Pay assets listed in ticket.md
  recovery: provision and authorize assets, then run a separately authorized controlled real-WeChat gate
```

## Writer score before immutable commit

- C=9：frozen TX-00 contract、W3/UI0、typed operations、信任/错误边界与外部资产可追溯；真实微信仍未验证。
- T=10：真实多轮 Red→Green；签名、时间窗、篡改、严格 DTO、解密、typed operations、失败分类、timeout 与 race 均可执行。
- V=8：writer gates 已完整；exact candidate SHA、双审与 detached verifier 尚待提交后执行。
- R=9：模糊/临时失败不产生业务终态，错误 retryability 明确，客户端不自动重放，恢复命令完整。
- 总分：36；产品/代码硬阻断为 0。外部 tracker linkage 保持 `BLOCKED_LOCAL_GOVERNANCE`，真实微信资产保持 `BLOCKED_EXTERNAL`，二者均未被本地 fixture 冒充 PASS。
