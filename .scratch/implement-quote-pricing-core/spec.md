# Spec: quote pricing core

## Interface

`services/api/internal/quotepricing` 是 backend-only、纯 in-process 算价 Module。唯一业务函数 Interface：

```go
func Calculate(input Input) (Result, error)
```

Supporting types：

```go
type Input struct {
    RatePercent int64
    Lines       []Line
}

type Line struct {
    UnitPriceCents int64
    Quantity       int64
}

type LineResult struct {
    OriginalUnitPriceCents   int64
    DiscountedUnitPriceCents int64
    Quantity                 int64
    OriginalSubtotalCents    int64
    PayableSubtotalCents     int64
}

type Result struct {
    RatePercent          int64
    Lines                []LineResult
    OriginalSubtotalCents int64
    DiscountCents         int64
    PayableCents          int64
}
```

`Error` / `ErrorKind` 提供稳定、脱敏的 `EMPTY_LINES`、`INVALID_RATE`、`INVALID_PRICE`、`INVALID_QUANTITY`、`OVERFLOW`。除 `Calculate` 外不得导出辅助业务函数；Module 不暴露或创建 clock、repo、adapter、配置选择 Interface。

## 输入契约

- 调用方必须传已经冻结的有效折扣率快照；Module 不识别员工、不读取设置、不决定设置何时生效。
- `RatePercent` 是整数百分比数学值，合法范围 `0..100`。访客未来传 `100`，员工未来传已冻结全局 rate。`0` 可计算不等于 PC policy 允许保存。
- `Lines` 必须非空并保持调用顺序。每行只含 `UnitPriceCents`（整数分，`>=0`）和 `Quantity`（`>0`），不携带/回传商品 id、name 或其他 pass-through 数据。
- Module 不修改 `input.Lines` 或其 backing array。

## 算价与结果

对每行按以下顺序：

1. `percentageProduct = UnitPriceCents * RatePercent`，乘法必须在 `int64` 可表示。
2. `DiscountedUnitPriceCents = percentageProduct / 100`；若非负余数 `>=50`，再安全加 `1`，即正数 half-up 到整数分。
3. `OriginalSubtotalCents = UnitPriceCents * Quantity`，安全乘法。
4. `PayableSubtotalCents = DiscountedUnitPriceCents * Quantity`，安全乘法；数量不得进入第 1/2 步。
5. 整单原价与应付分别按行安全累加，`DiscountCents = OriginalSubtotalCents - PayableCents`。

逐行输出复述输入原价、数量和冻结的折后单价，顺序与输入一致。不得先汇总原价小计后折扣。

决定性结果：

| 输入 | 逐行/整单结果 |
| --- | --- |
| `101 × 85%, qty=2` | unit `86`; original `202`; payable `172`; cut `30` |
| 两行 `1 × 50%, qty=1` | 两行 unit/payable 均 `1`; total payable `2` |
| `5 × 50%, qty=3` | unit `3`; original `15`; payable `9`; cut `6` |
| `100%` | 原价不变 |
| `0%` | 折后与应付为 `0` |
| `0元` | 原价/折后/应付为 `0`，合法 |

## 失败语义

- 稳定错误优先级固定为：先校验 `RatePercent`，再校验空 `Lines`，随后严格按输入行顺序逐行执行 `UnitPriceCents`、`Quantity`、该行 arithmetic。任一阶段发现首个错误立即返回精确零值 `Result{}`，不得继续到较低优先级错误或后续行。
- 因此 `invalid rate + empty lines` 必须返回 `INVALID_RATE`；同一行同时 `negative price + non-positive quantity` 必须返回 `INVALID_PRICE`。
- 空 `Lines`：`EMPTY_LINES`。
- `RatePercent <0` 或 `>100`：`INVALID_RATE`。
- 任一负价格：`INVALID_PRICE`。
- 任一 `Quantity <=0`：`INVALID_QUANTITY`。
- 折扣乘法、原价/应付行乘法或跨行原价/应付加法任一溢出：`OVERFLOW`。
- arithmetic 包含该行折扣乘法/half-up 加法、原价行乘法、应付行乘法及按当前行顺序发生的整单累加；首个 arithmetic overflow 返回 `OVERFLOW`。
- 任一错误必须返回精确零值 `Result{}`；不得返回 partial lines/totals，不得 panic，不得 wrap/打印 input values。

## 确定性与边界

相同输入重复和并发调用必须得到相同 Result/错误 kind，race clean；实现不得访问 I/O、global、random、time。该 Module 不创建 Quote/Prepay/Order，不承诺数据库、router、身份、白名单、价格版本、配置生效、TTL、支付或客户端行为。

## Evidence binding

- `base_sha`: `8bcdf3d6b1ea41529adaa54f463cc118c69e0e25`
- `candidate_sha: external-post-commit`
- review/verifier 只接受 immutable handoff 绑定的 exact SHA；任何实现、本 Spec、tasks、Gate 或 SHA 变化都使旧 receipt 失效。
