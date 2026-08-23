# Spec: implement-order-lifecycle-transition-policy-core

## 1. 范围与事实源

本 Spec 仅适用于 exact base `8bcdf3d6b1ea41529adaa54f463cc118c69e0e25` 上的 `services/api/internal/orderproduction`。产品事实源只有 canonical PRD `docs/product/online-ordering-system-prd-0818.md` §6.6、§7.1、§7.5–§7.7、§12/§14；delegated 指令进一步冻结本 change 的 Interface、矩阵和验证方式。

现有 `State`、`Decision`、`InitialState`、`Advance` MUST 复用且语义/测试 MUST NOT 改变。MUST NOT 新建第二套状态 package。

## 2. 公共 Interface

Module MUST 新增且只新增一个生命周期转换函数：

```go
func Transition(input TransitionInput) (Decision, error)
```

`TransitionInput` MUST 且只能包含：

```go
type TransitionInput struct {
    Current    State
    Trigger    Trigger
    ObservedAt time.Time
    PickupAt   time.Time
}
```

`Trigger` MUST 是稳定 typed string enum，且只允许以下常量：

- `MERCHANT_MARK_READY`
- `REDEEM_SUCCEEDED`
- `USER_CANCEL`
- `OWNER_REFUND_REQUESTED`
- `VERIFIED_REFUND_SUCCEEDED`

Module MUST 保持无 I/O、无 DB/repo、无 clock、无 Adapter、无全局可变状态。角色、订单归属、权限、幂等与微信退款真实状态 MUST 由未来 caller 在调用前验证；Input MUST NOT 接收角色、HTTP/body 或 raw provider status。

## 3. 合法转换

以下是全部合法转换，成功 MUST 返回 `{State: <target>, Changed: true}` 与 `nil`：

1. `PREPARING + MERCHANT_MARK_READY -> READY_FOR_PICKUP`
2. `READY_FOR_PICKUP + REDEEM_SUCCEEDED -> COMPLETED`
3. `RESERVED + USER_CANCEL -> REFUNDING`，仅当 `ObservedAt`/`PickupAt` 均非零且 `PickupAt-ObservedAt` 严格 `>30m`
4. `RESERVED + OWNER_REFUND_REQUESTED -> REFUNDING`
5. `PREPARING + OWNER_REFUND_REQUESTED -> REFUNDING`
6. `READY_FOR_PICKUP + OWNER_REFUND_REQUESTED -> REFUNDING`
7. `COMPLETED + OWNER_REFUND_REQUESTED -> REFUNDING`
8. `REFUNDING + VERIFIED_REFUND_SUCCEEDED -> REFUNDED`

完整矩阵：

| current | MARK_READY | REDEEM | USER_CANCEL | OWNER_REFUND | VERIFIED_REFUND |
| --- | --- | --- | --- | --- | --- |
| `RESERVED` | reject | reject | `REFUNDING` iff `>30m` | `REFUNDING` | reject |
| `PREPARING` | `READY_FOR_PICKUP` | reject | reject | `REFUNDING` | reject |
| `READY_FOR_PICKUP` | reject | `COMPLETED` | reject | `REFUNDING` | reject |
| `COMPLETED` | reject | reject | reject | `REFUNDING` | reject |
| `REFUNDING` | reject | reject | reject | reject | `REFUNDED` |
| `REFUNDED` | reject | reject | reject | reject | reject |

`REFUNDED` MUST 是终态。MUST NOT 把同状态重复请求当成功；未来持久化幂等层负责返回第一次结果。MUST NOT 引入待支付、已取消、异常或其他状态。

## 4. 时间与错误边界

- `ObservedAt` 与 `PickupAt` 仅用于合法候选 `RESERVED + USER_CANCEL`。
- 非 `USER_CANCEL` trigger MUST 完全忽略两时间；零值或任意非零值 MUST NOT 改变其合法/非法结果。
- 用户取消恰好 `30m`、小于 `30m`、时间反向均 MUST 拒绝；只有严格 `>30m` 成功。
- 合法候选用户取消任一零时间 MUST 返回 `INVALID_TIME`。
- 空、未知、deprecated state MUST 返回 `INVALID_STATE`。
- 空或未知 trigger MUST 返回 `INVALID_TRIGGER`。
- 其他已知 current/trigger 组合 MUST 返回 `TRANSITION_NOT_ALLOWED`。
- 错误验证顺序 MUST 是 current、trigger、矩阵边、用户取消时间；因此无效 state 不得被时间或 trigger 路径误接纳。
- 所有错误 MUST 是 typed `*Error`，error text MUST 只暴露稳定 kind，不回显输入。
- 所有错误 MUST 返回精确零 `Decision{}`；不得返回状态或 `Changed:true`。

## 5. 纯度与确定性

- `TransitionInput` 按值传入，Module MUST NOT 修改调用方输入。
- 相同输入的重复调用 MUST 返回相同 Decision/error kind。
- 并发调用 MUST 无共享可变状态并通过 race。
- 现有 `InitialState`/`Advance` 的返回、边界、错误与并发行为 MUST 保持不变。

## 6. 验收与 mutation

所有行为测试 MUST 只通过公共 `Transition` Interface。验收 MUST 覆盖：

- 五类合法 trigger 边（owner refund 覆盖四个前态）；
- 用户取消 `>30m`、`=30m`、`<30m`、零时间与制作中拒绝；
- 六态 x 五 trigger 完整矩阵的 8 条合法边与 22 条非法边；
- invalid/deprecated state、unknown trigger、失败零 Decision；
- 非取消 trigger 时间独立、输入不变、重复/并发确定性；
- public seam 编译约束与既有回归。

Mutation harness MUST 在临时副本注入至少 8 个可逆 mutant，source pattern 每项恰好一次；只有目标测试 exit 1 且含指定命名 FAIL marker 才算 killed。harness MUST 先以模拟 tool failure 证明 infrastructure failure shield。八项 mutation 必须分别覆盖错误前态备好、跳态核销、`=30m` 可取消、制作中可取消、漏掉 COMPLETED owner refund、允许 REFUNDING 重复退款、非 REFUNDING 完成退款、接受 invalid/deprecated state。

## 7. W3/UI0 与证据边界

- gate type: `W3`；target/actual: `UI0`。
- focused、race `-count=20`、determinism、mutation、fresh loopback-only MySQL 8.0.46 全 `services/api` test/race、vet/build/smoke、gofmt/diff/owned/protected/sensitive/clean MUST PASS。
- fresh MySQL 全回归只是邻接证据，MUST NOT 宣称证明本纯 Module 的 DB 事务、锁、CAS、幂等或真实退款/核销。
- phase 与 `exit_result` MUST 使用协议枚举；历史/其他 SHA 结果不得继承。
- 中文 commit 形成 exact candidate；candidate SHA 用 external post-commit receipt 绑定。
- fixed base 双轴 Standards/Spec review MUST 0 finding；任何修复产生新 SHA 并使旧 Gate/review 失效。
- 双审通过后 MUST 在 fresh clean detached exact-SHA 从头重跑全部 Gate。
- 本 change MUST NOT integration、push、PR、deploy 或访问微信/生产。

## 8. 未来调用方（不实现）

唯一 DB state writer MUST 在锁单并完成鉴权、资源归属、幂等与 provider 事实验证后调用 `Transition`。未来 Apply change MUST 证明 CAS/锁/幂等/audit 同事务、退款与核销竞争、失败恢复、通知与二维码/token；本 Spec 不把这些能力算作当前验收结果。
