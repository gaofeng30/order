# Spec: order production policy core

## Interface

`services/api/internal/orderproduction` 是 backend-only 纯策略 Module。唯一业务函数 Interface：

```go
func InitialState(paymentSucceededAt, pickupAt time.Time) (State, error)
func Advance(current State, observedAt, pickupAt time.Time) (Decision, error)
```

Supporting types：

- `State`：typed enum，只允许 `RESERVED`、`PREPARING`、`READY_FOR_PICKUP`、`COMPLETED`、`REFUNDING`、`REFUNDED`。
- `Decision`：只含 `State State`、`Changed bool`。
- `Error` / `ErrorKind`：只表达 `INVALID_TIME`、`INVALID_STATE`；错误文本不得包含任何输入时间或状态值。

调用方必须显式传绝对 `time.Time`。Module 不创建 clock，不解析时区，不访问 I/O、DB、repo，不持有全局状态。

## InitialState

- 仅由调用方在服务端已确认支付成功后调用；本 Module 不验证支付，不创建订单，不分配取餐号。
- `paymentSucceededAt` 与 `pickupAt` 必须非零，且 `pickupAt` 必须严格晚于 `paymentSucceededAt`；否则返回 `INVALID_TIME`，不返回可用状态。
- `pickupAt-paymentSucceededAt < 30m`：返回 `PREPARING`。
- `pickupAt-paymentSucceededAt >= 30m`：返回 `RESERVED`。
- 特别冻结：恰好 30 分钟返回 `RESERVED`。

## Advance

- `observedAt` 与 `pickupAt` 必须非零；否则返回 `INVALID_TIME`，不返回可用 Decision。
- `current` 必须是六态之一；空状态、未知状态和已废止状态均返回 `INVALID_STATE`，不返回可用 Decision。
- `current=RESERVED` 且 `observedAt < pickupAt-30m`：返回 `{State: RESERVED, Changed: false}`。
- `current=RESERVED` 且 `observedAt >= pickupAt-30m`：返回 `{State: PREPARING, Changed: true}`。恰好阈值与 scheduler 漏跑后都推进。
- `current` 为 `PREPARING`、`READY_FOR_PICKUP`、`COMPLETED`、`REFUNDING`、`REFUNDED`：返回原状态与 `Changed:false`，永不回退。

## 边界组合

PRD §7.4 同时规定「不足 30 分钟初态直接制作中」与「取餐前 30 分钟定时推进」。所以支付成功距取餐恰好 30 分钟时：

1. `InitialState` 先返回 `RESERVED`；
2. scheduler 若在同一绝对时刻观察，可立即用 `Advance` 得到 `PREPARING/Changed:true`。

这是刻意保留的两步行为，不得把 `InitialState` 改为一步 `PREPARING`。

## 状态表

| current / input | 时间条件 | 结果 | Changed | error |
| --- | --- | --- | --- | --- |
| Initial | `0 < pickup-success < 30m` | `PREPARING` | N/A | nil |
| Initial | `pickup-success >= 30m` | `RESERVED` | N/A | nil |
| Initial | 任一零时间或 `pickup<=success` | 不可用 | N/A | `INVALID_TIME` |
| `RESERVED` | `observed < pickup-30m` | `RESERVED` | false | nil |
| `RESERVED` | `observed >= pickup-30m` | `PREPARING` | true | nil |
| 其余五个合法状态 | 任意非零时间 | 原状态 | false | nil |
| 空/未知/废止状态 | 任意非零时间 | 不可用 | false | `INVALID_STATE` |
| 任意状态 | `observedAt` 或 `pickupAt` 为零 | 不可用 | false | `INVALID_TIME` |

## 确定性与失败语义

- 相同输入重复调用必须产生相同返回值/错误 kind；并发调用无共享可变状态且 race clean。
- 错误必须 fail closed：不得返回一个可被误当作合法推进的状态或 `Changed:true`。
- Module 只决定策略，不承诺数据库原子性、幂等键、审计、取餐号、scheduler 恢复或真实支付事实；这些属于未来纵切。
