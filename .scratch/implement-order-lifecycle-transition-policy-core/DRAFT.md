# DRAFT: implement-order-lifecycle-transition-policy-core

## 固定点与治理状态

- change: `implement-order-lifecycle-transition-policy-core`
- status: `REPLACEMENT_CANDIDATE_BY_EXTERNAL_POST_COMMIT_RECEIPT`
- `base_sha`: `8bcdf3d6b1ea41529adaa54f463cc118c69e0e25`
- owner: 当前 delegated 独立 writer Session
- writer branch: `codex/implement-order-lifecycle-transition-policy-core`
- writer worktree: `/Users/vivix/.codex/worktrees/a1f2/order`
- `candidate_sha`: `external-post-commit`；完整 SHA 只能在中文 commit 形成后由外部 receipt 绑定，不能写进产生该 SHA 的 commit。
- `gate_type`: `W3`（改变订单、退款、核销的状态转换结果）
- `ui_level_target`: `UI0`
- `ui_level_actual`: `UI0`
- tracker: `GOVERNANCE_PENDING`；仓库没有 `docs/agents/issue-tracker.md`，本 change 由已确认 delegated 指令与本目录 `spec.md` 冻结，不配置 tracker、不伪造 issue。

初始证据：worktree 以 detached exact base 启动且 clean；目标 branch 原先不存在，已在当前 worktree 创建并切换，未创建第二个 writer worktree。

首个 candidate `2cfc0e4954ea80c07680ae56b1d055ff6c627731` 已因 Standards review 的两项 finding 作废：tasks 缺少逐 task 完整 Gate receipt，且按值输入的运行时“不修改”断言属于无法因实现变化失败的 tautological test。旧 Gate/review 不继承；replacement 以编译期 `time.Time` 值字段约束替代该断言，并从头重跑全部 Gate/review。

## 唯一目标与深 Module

扩展 exact base 已集成的纯 `orderproduction` Module，复用现有 `State`、`Decision`、`InitialState` 与 `Advance`，新增唯一公共生命周期转换 Interface：

```go
func Transition(input TransitionInput) (Decision, error)
```

`TransitionInput` 只含 `Current State`、稳定 typed `Trigger`、`ObservedAt time.Time`、`PickupAt time.Time`。时间仅供 `USER_CANCEL` 使用；其他 trigger 的结果不得依赖两个时间值。Module 继续保持纯计算、无 I/O、无 repo、无 clock、无 Adapter、无共享可变状态。

该 Interface 把合法状态边、严格 30 分钟取消边界、typed fail-closed 错误集中在一个 seam。删除它会迫使未来备好、核销、取消、退款与 verified refund caller 重复状态矩阵，因此提供 Leverage 与 Locality；不会建立第二套状态 package 或第二个转换 Interface。

## 来源与硬约束

- 产品事实源只有 `docs/product/online-ordering-system-prd-0818.md` §6.6、§7.1、§7.5–§7.7、§12/§14。
- exact base 上 `orderproduction.InitialState` 与 `orderproduction.Advance` 的 Interface、语义与测试是既有公共契约，不修改。
- confirmed delegated 指令冻结 trigger 集、合法矩阵、严格 `>30m`、typed fail-closed、失败零 Decision、终态与未来 caller 责任。
- scratch Spec 只冻结本明确 change；不把 `openspec/` 或未配置 tracker 当工具入口。

## Owned / read-only / 非目标

唯一 owned paths：

- `.scratch/implement-order-lifecycle-transition-policy-core/**`
- `services/api/internal/orderproduction/**`

其余全部只读，尤其：

- `.scratch/implement-quote-pricing-core/**`
- `services/api/internal/quotepricing/**`
- `docs/product/**`
- `services/api/internal/paymentobservation/**`
- `services/api/internal/wechatpay/**`
- `services/api/internal/merchantidentity/**`
- `services/api/internal/httpapi/**`
- `services/api/cmd/**`
- `services/api/migrations/**`
- `go.mod`、`go.sum`、`apps/**`

非目标：v14、router/main、DB/HTTP/client、角色与订单归属、权限、持久化幂等、CAS/锁/事务/audit、微信退款状态解析、退款调用、核销 token/二维码、通知、恢复、integration、push、PR、deploy、生产/微信访问。

## Interface 与失败语义

```go
type Trigger string

const (
    TriggerMerchantMarkReady       Trigger = "MERCHANT_MARK_READY"
    TriggerRedeemSucceeded         Trigger = "REDEEM_SUCCEEDED"
    TriggerUserCancel              Trigger = "USER_CANCEL"
    TriggerOwnerRefundRequested    Trigger = "OWNER_REFUND_REQUESTED"
    TriggerVerifiedRefundSucceeded Trigger = "VERIFIED_REFUND_SUCCEEDED"
)

type TransitionInput struct {
    Current    State
    Trigger    Trigger
    ObservedAt time.Time
    PickupAt   time.Time
}

func Transition(input TransitionInput) (Decision, error)
```

- 成功转换只返回目标 `State` 与 `Changed:true`。
- 所有失败返回精确 `Decision{}` 与 typed `*Error`。
- 错误优先级固定为：先验证 `Current`，再验证 `Trigger`，再判定该 state/trigger 边是否合法；仅合法候选 `RESERVED + USER_CANCEL` 校验时间。
- `INVALID_STATE`：空、未知、deprecated state。
- `INVALID_TRIGGER`：空或未知 trigger。
- `INVALID_TIME`：合法候选用户取消的任一时间为零。
- `TRANSITION_NOT_ALLOWED`：已知 state/trigger 不在合法矩阵，或用户取消距取餐 `<=30m`。
- error text 只含稳定 kind，不回显输入。
- 非 `USER_CANCEL` trigger 完全忽略 `ObservedAt/PickupAt`；零值和任意非零值不得改变结果。
- 输入按值传递；实现不得修改输入。相同输入重复/并发调用确定且 race clean。

## 完整 current x trigger 矩阵

`R=RESERVED`、`P=PREPARING`、`Q=READY_FOR_PICKUP`、`C=COMPLETED`、`F=REFUNDING`、`D=REFUNDED`。

| current | MERCHANT_MARK_READY | REDEEM_SUCCEEDED | USER_CANCEL | OWNER_REFUND_REQUESTED | VERIFIED_REFUND_SUCCEEDED |
| --- | --- | --- | --- | --- | --- |
| R | reject | reject | `F` iff `PickupAt-ObservedAt >30m` | `F` | reject |
| P | `Q` | reject | reject | `F` | reject |
| Q | reject | `C` | reject | `F` | reject |
| C | reject | reject | reject | `F` | reject |
| F | reject | reject | reject | reject | `D` |
| D | reject | reject | reject | reject | reject |

不存在同状态重复成功；未来持久化幂等层必须返回第一次结果。`REFUNDED` 是终态。不得引入待支付、已取消、异常等状态。

## Red -> Green -> Refactor

确认的测试 seam 只有公共 `orderproduction.Transition` Interface；测试 package 使用 `orderproduction_test`，不触及私有实现。

1. public-seam 编译 Red：外部测试引用冻结的 `Trigger`、`TransitionInput` 与 `Transition`，因 Interface 尚不存在退出 1。
2. 逐 tracer：备好、核销、用户取消 `>30m`、四个 owner refund 前态、verified refund；每个先以目标行为缺失获得 Red，再做最小 Green。
3. 用户取消边界：`>30m` 成功，`=30m` 与 `<30m` fail closed；制作中取消拒绝。
4. 完整六态 x 五 trigger 表验证 8 条合法边与 22 条非法边；invalid/deprecated state、unknown trigger、零时间、失败精确零 Decision。
5. 非取消 trigger 的时间独立性、输入不变、重复/并发确定性。
6. Refactor 后重跑同一 focused、race `-count=20`、determinism 与 mutation Gate。

## Mutation Gate

临时副本内至少 8 个可逆 mutant，每个 source pattern 必须恰好一次；只有目标测试 exit 1 且出现指定 `--- FAIL: Test...` marker 才算 killed。infrastructure failure shield 必须证明模拟 tool failure 不会被误报为 killed。覆盖：

1. 错误前态也可备好；
2. 跳态核销；
3. `=30m` 仍可用户取消；
4. 制作中用户取消；
5. 漏掉 `COMPLETED` owner refund；
6. 允许 `REFUNDING` 重复发起 owner refund；
7. 非 `REFUNDING` 完成退款；
8. 接受 invalid/deprecated state。

## Writer / Review / Verifier / Integration 命令

- focused：`GOPROXY=off GOTOOLCHAIN=go1.26.5 go test ./services/api/internal/orderproduction -count=1`
- race/determinism：`GOPROXY=off GOTOOLCHAIN=go1.26.5 go test -race ./services/api/internal/orderproduction -count=20`
- mutation：`bash .scratch/implement-order-lifecycle-transition-policy-core/verify-mutation-gate.sh`
- fresh loopback-only MySQL 8.0.46 全邻接回归：`bash .scratch/repair-version-scoped-mysql-migration-fixtures-v13/verify-mysql.sh full`；该证据只证明 `services/api` 邻接回归，不证明本纯 Module 的 DB 事务、锁、CAS 或幂等。
- static/build/smoke：`GOPROXY=off GOTOOLCHAIN=go1.26.5 go vet ./services/api/...`；`GOPROXY=off GOTOOLCHAIN=go1.26.5 go build ./services/api/...`；`GOPROXY=off GOTOOLCHAIN=go1.26.5 bash services/api/scripts/smoke.sh`。
- format/diff/scope/security：gofmt、`git diff --check`、owned/protected path audit、sensitive-pattern audit、shell syntax、clean status。
- review fixed point：`git diff 8bcdf3d6b1ea41529adaa54f463cc118c69e0e25...HEAD` 与 `git log 8bcdf3d6b1ea41529adaa54f463cc118c69e0e25..HEAD --oneline`；按 `$code-review` 并行 Standards/Spec 双轴，任一 finding 由 writer 修复后产生新 SHA，从头 Gate/review。
- verifier：双审 0 finding 后，在 fresh clean detached exact candidate SHA 从头串行重跑全部声明 Gate，并确认 detached HEAD、exact SHA、clean、无受保护路径改动。
- integration：只允许未来集成人在依赖满足、candidate independent PASS 且取得单独授权后执行；本 writer 不集成。

## 外部资产

- 本 change 无 UI、微信、生产或真实资金资产要求；`ui_level_actual=UI0`。
- writer/verifier Gate 需要本机 Go 1.26.5、Docker daemon 与可创建 fresh loopback-only MySQL 8.0.46 容器；writer 已实际建立 fresh `8.0.46`、`127.0.0.1` 临时环境并完成全回归。
- asset owner: writer/verifier；missing: writer 无，verifier 必须独立重建；recovery: Docker daemon 可用且 `mysql:8.0.46-oraclelinux9` 可拉取/启动后从头重跑。

## 未来集成边界（只冻结，不实现）

唯一 DB state writer 在锁单并完成角色、资源归属、权限、幂等与 provider 事实验证后调用 `Transition`。备好、核销、取消、退款、verified refund Apply 仍需独立 change 证明 CAS/锁/幂等/audit 同事务、退款与核销竞争、失败恢复、通知与二维码/token；本 Module 不接收角色、HTTP/body、raw provider status，也不宣称证明这些能力。
