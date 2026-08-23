# implement-payment-observation-normalizer-core

**Status:** `WRITER_RUNTIME_PASS / CANDIDATE_PENDING`

**Approval source:** 主控对本 backend-only slice 的明确委派；不代表交易契约 WIP 已获产品批准。

## 固定点

- `base_sha`: `5e937f3599a16f4813d6021f4cd2dd637c3156a2`
- `branch`: `codex/implement-payment-observation-normalizer-core-v13`
- `worktree`: `/Users/vivix/.codex/worktrees/142a/order`
- `gate_type`: `W3`
- `ui_level_target`: `UI0`
- `ui_level_actual`: `UI0`
- owner: 本 writer session，单 writer

## 目标与最小成功标准

在 `services/api/internal/paymentobservation` 建立一个 in-process 深模块。调用方把已由
`wechatpay` 信任边界验签/解密/解码后的 typed `wechatpay.Transaction` 与本地冻结的
`Expectation` 交给唯一 interface：

```go
Normalize(expected Expectation, input Input) (Observation, error)
```

最小成功标准：callback 与 active-query 对同一 provider 事实产出同一 canonical
Observation；严格验证本地冻结事实；业务冲突产出可持久化 `REJECTED_MISMATCH`；结构错误或
未支持状态只返回稳定 typed error；SHA-256 dedupe 对重复稳定、对关键事实变化敏感；结果不含
个人信息、原始载荷、回调通知 ID、签名或本地时间。

## 所有权与只读边界

Owned paths：

- `.scratch/implement-payment-observation-normalizer-core/**`
- `services/api/internal/paymentobservation/**`

Read-only：

- `services/api/internal/wechatpay/**`
- `services/api/internal/httpapi/**`
- `services/api/cmd/**`
- `services/api/migrations/**`
- `services/api/internal/order*/**`
- `services/api/internal/payment*/**`，但本 change 新建的 `paymentobservation/**` 除外
- `services/api/internal/quote*/**`
- `services/api/internal/merchantidentity/**`
- `apps/**`
- `go.mod`、`go.sum`

禁止修改 router、migration、composition、公共 transaction contract、客户端或外部系统。

## 依赖、来源与非目标

- 直接代码依赖：exact base 已存在的只读 `services/api/internal/wechatpay.Transaction`。
- 工程输入：`/Users/vivix/.codex/worktrees/851f/order/.scratch/freeze-transaction-payment-contract/**`
  仅只读，保持 `WIP / NOT_APPROVED`；旧 review/PASS 不作为本 change 验收证据。
- canonical 产品事实：`docs/product/online-ordering-system-prd-0818.md` 的支付、订单、幂等和
  敏感数据条款。
- 非目标：数据库表、migration、callback/query HTTP、订单投影、预支付、退款、outbox、
  router/main、P5 折扣、PAYMENT-TTL、真实微信联调。

## Red -> Green -> Refactor

- Red：先由公共 seam 测试调用不存在的 `Normalize`，记录首个决定性编译错误；随后每个行为
  slice 都先得到失败再最小实现。
- Green：同一 focused suite 通过，覆盖 accepted、rejected mismatch、malformed、unsupported、
  callback/query canonical equality、重复稳定和 collision。
- Refactor：只在全部行为 Green 后收窄命名/重复实现，重跑完全相同 focused 与 race suite。

## Gate 命令

Writer / Verifier 对同一 exact candidate 执行：

```bash
GOPROXY=off GOTOOLCHAIN=go1.26.5 go test ./services/api/internal/paymentobservation -count=1
GOPROXY=off GOTOOLCHAIN=go1.26.5 go test -race ./services/api/internal/paymentobservation -count=1
bash .scratch/implement-payment-observation-normalizer-core/mutation-test.sh
bash .scratch/implement-payment-observation-normalizer-core/verify-mysql.sh
GOPROXY=off GOTOOLCHAIN=go1.26.5 go test ./services/api/... -count=1
GOPROXY=off GOTOOLCHAIN=go1.26.5 go test -race ./services/api/... -count=1
GOPROXY=off GOTOOLCHAIN=go1.26.5 go vet ./services/api/...
GOPROXY=off GOTOOLCHAIN=go1.26.5 go build ./services/api/...
GOPROXY=off GOTOOLCHAIN=go1.26.5 bash services/api/scripts/smoke.sh
test -z "$(gofmt -l services/api/internal/paymentobservation)"
git diff --check 5e937f3599a16f4813d6021f4cd2dd637c3156a2...HEAD
python3 .scratch/implement-payment-observation-normalizer-core/check_change.py
```

Integration 仅在依赖已进入目标 integration base、candidate exact SHA 独立复验 PASS 且另获集成
授权后执行；本 change 不 push、不建 PR、不集成、不部署。

## 外部资产

| 资产 | owner | 当前状态 | 恢复条件 / 边界 |
| --- | --- | --- | --- |
| 临时隔离 MySQL 8.0 | Writer/Verifier | `WRITER_PASS / VERIFIER_PENDING` | Writer 已在 fresh MySQL 8.0.46 重跑完整矩阵；detached exact-candidate Verifier 仍须另启 fresh container，不复用本次 PASS |
| 正式 AppID/mchid、证书、APIv3 key、HTTPS callback、查单权限、真实资金 | 客户商户管理员 + 开发方 | `BLOCKED_EXTERNAL` | 未来 ingress/UAT change 获单独授权；本 change 不读取、不需要、不验证 |
| UI runner / 真机 | N/A | `N/A` | backend-only 无 UI；UI0 不证明 UI1/UI2/UI3 |

## Dependency 恢复

旧 base `b7f484f...7483` 的完整 W3 failure 保留在 `evidence/dependency-blocker.md`，不得改写。
主控已把独立 foundation change 集成为新 exact base `5e937f...56a2`；本 writer 只在当前 base
重新取得 RED 并从头通过全部 Gate 后形成 Candidate。
