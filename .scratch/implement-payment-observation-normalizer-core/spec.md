# Payment Observation Normalizer Core Spec

**Status:** `WRITER_RUNTIME_PASS / CANDIDATE_PENDING`

本 Spec 只冻结本 change 的 Go interface 与纯领域行为，不批准或替代交易契约 worktree 的
`WIP / NOT_APPROVED` 产品与持久化设计。

## Module、seam 与 interface

`paymentobservation` 是 in-process 深模块；唯一 seam 位于：

```go
func Normalize(expected Expectation, input Input) (Observation, error)
```

调用方：未来已完成 provider 验签/解密/严格解码的 callback handler 与主动查单 worker。
业务场景：在任何数据库写入之前，把同一可信 provider transaction 归一成唯一、最小、可持久化
领域 Observation。调用时机：typed `wechatpay.Transaction` 已形成、本地 frozen prepayment
Expectation 已读取之后；本模块不校验签名、不访问 DB、不投影订单。

## 输入契约

`Expectation` 固定包含 `AppID`、`MerchantID`、`OutTradeNo`、正整数 `TotalAmount` 和固定
`Currency="CNY"`。字符串不能为空或含 NUL。

`Input` 固定包含：

- `Source`: `CALLBACK` 或 `ACTIVE_QUERY`；Source 只用于校验可消费的 source/state 组合，
  不进入 Observation 或 dedupe。
- `Transaction`: 只读 `wechatpay.Transaction`。

支持的 provider 状态只取当前冻结交易链已消费的三态：

| provider `TradeState` | domain `State` | source |
| --- | --- | --- |
| `SUCCESS` | `PAID` | `CALLBACK`、`ACTIVE_QUERY` |
| `NOTPAY` | `NOT_PAID` | 仅 `ACTIVE_QUERY` |
| `CLOSED` | `CLOSED` | 仅 `ACTIVE_QUERY` |

`SUCCESS` 必须带非空 `TransactionID`、非零 `SuccessTime`、正 `Amount.Total` 与非空 currency。
`NOTPAY/CLOSED` 的 transaction/success/amount/currency 可以全空；若 provider typed 值携带金额，
amount 与 currency 必须成对存在并参与匹配/canonicalization。所有字符串字段拒绝 NUL。

## 输出契约

`Observation` 只含：

- `DedupeKey`: 64 位小写十六进制 SHA-256；
- `Validation`: `ACCEPTED` 或 `REJECTED_MISMATCH`；
- `Mismatch`: `NONE/APP_ID/MERCHANT_ID/OUT_TRADE_NO/TOTAL_AMOUNT/CURRENCY`；
- `State`: `PAID/NOT_PAID/CLOSED`；
- `OutTradeNo`: 本地 Expectation 的 merchant order number；
- accepted `PAID` 所需的最小 provider 事实：`TransactionID`、UTC `SuccessTime`、
  `TotalAmount`、`Currency`。

accepted 非支付态和 rejected observation 不保留 transaction ID、时间、金额或 provider raw 值。
任何结果都不保留 Source、Payer/OpenID、TradeStateDescription、BankType、Attach、手机号、姓名、
证书、签名、raw body、notification ID、callback headers 或本地收件时间。

## 验证与错误

处理顺序固定：

1. 验证 Expectation 结构；
2. 验证 Source 与 Transaction 结构；
3. 映射 provider 状态并验证 source/state 组合；
4. 按以下顺序裁决业务 mismatch：`AppID -> MerchantID -> OutTradeNo -> TotalAmount -> Currency`；
5. 构造 canonical bytes、dedupe 与最小 Observation。

业务 mismatch 返回 `Observation{Validation: REJECTED_MISMATCH}` 且 `error=nil`，保证未来 ingress
可以先耐久保存再停止 Apply。若多项不匹配，只保留上述优先级最高的一个安全枚举，但 dedupe
仍覆盖全部 expected/provider 关键事实。

malformed/unsupported 不产生 Observation，并返回 `*Error` 的稳定 Kind：

- `MALFORMED_EXPECTATION`
- `MALFORMED_INPUT`
- `UNSUPPORTED_SOURCE`
- `UNSUPPORTED_TRADE_STATE`
- `UNSUPPORTED_SOURCE_STATE`

错误文本只含 kind，不含输入值。

## Canonical bytes 与 dedupe

canonical v1 为以下固定顺序字段的 UTF-8 字节，以单个 NUL 分隔、无尾随 NUL；所有输入字符串
含 NUL 时拒绝。时间统一为 UTC `RFC3339Nano`，正整数使用十进制无前导零，缺失 provider
可选字段使用空串：

```text
order.payment-observation.v1
validation
mismatch
state
expected_appid
actual_appid
expected_mchid
actual_mchid
expected_out_trade_no
actual_out_trade_no
expected_total_amount
actual_total_amount_or_empty
expected_currency
actual_currency_or_empty
transaction_id_or_empty
success_time_utc_or_empty
```

`DedupeKey = lowercase_hex(SHA256(canonical_v1_bytes))`。Source、map、JSON raw body、header、
notification ID、本地时间及被明确排除的 provider/个人字段不参与。因 Source 被排除，同一
SUCCESS transaction 从 callback 和 active-query 得到 byte-for-byte 相同内容和 key；任一关键
expected/provider 事实、状态或 validation 改变都改变 canonical bytes。

## 不变量

- 相同 Expectation + Input 重复调用完全相等，无时钟、随机数或全局状态。
- accepted 必须先通过所有业务匹配；unsupported/malformed 绝不伪造 accepted。
- mismatch observation 可持久化但永远不是支付成功 Apply 授权。
- `PAID` 不等于 Order；本模块不建单、不分配取餐号、不决定正常/延迟 Apply mode。
- 不引入 P5 折扣、PAYMENT-TTL、退款、outbox 或任何未来扩展 hook。
