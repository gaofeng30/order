# DRAFT: implement-order-production-policy-core

## 固定点与状态

- change: `implement-order-production-policy-core`
- status: `REPLACEMENT_CANDIDATE_READY_FOR_EXACT_REVIEW`
- `base_sha`: `5e937f3599a16f4813d6021f4cd2dd637c3156a2`
- source branch: `codex/order-delivery-integration`
- writer branch: `codex/implement-order-production-policy-core`
- writer worktree: `/Users/vivix/.codex/worktrees/fd9e/order`
- `candidate_sha`: `39c0ea43067f30e5befd48290e9cb45711415452`
- `candidate_status`: `INVALIDATED_BY_STANDARDS_GOVERNANCE_AUDIT`
- `replacement_final_sha`: 由包含本治理修复的 commit 形成，并在 immutable external handoff 中绑定完整 SHA；不得把未来 commit SHA 写入其自身内容制造无限 amend。
- `gate_type`: `W3`（订单状态与并发确定性，按最高风险分类）
- `ui_level_target`: `UI0`
- `ui_level_actual`: `UI0`
- owner: 本 change 独立 writer
- tracker: `BLOCKED_LOCAL_GOVERNANCE`；仓库缺少 `docs/agents/issue-tracker.md`，不创建或伪造外部 issue，本目录 `spec.md` 是本 change 的唯一实施 Spec。

## 目标、Module 与 seam

目标是在 `services/api/internal/orderproduction` 建立 backend-only 深 Module，把支付成功建单初态与自动排产判断收敛到一个纯策略 seam。调用方和测试只通过以下 Interface 观察行为：

```go
InitialState(paymentSucceededAt, pickupAt time.Time) (State, error)
Advance(current State, observedAt, pickupAt time.Time) (Decision, error)
```

`State` 是本包六态 typed enum；`Decision` 只含 `State` 与 `Changed`。最小 typed `Error`/`ErrorKind` 只暴露 `INVALID_TIME`、`INVALID_STATE`，错误文本不包含输入。该 Module 无 clock、时区解析、I/O、全局状态、repo 或 Adapter；调用方显式传入绝对 `time.Time`。

删除该 Module 会迫使 TX-03/TX-04 的建单路径与 TX-05 scheduler 分别复制 30 分钟边界、非法状态和不回退规则，因此该 seam 提供 Leverage 与 Locality，不是 pass-through。

## 来源与约束等级

- canonical PRD：`docs/product/online-ordering-system-prd-0818.md` §7.1/§7.2/§7.4，是六态、支付成功后才建单、不足 30 分钟初态、定时推进与不回退的批准产品事实。
- transaction WIP：`/Users/vivix/.codex/worktrees/851f/order/.scratch/freeze-transaction-payment-contract/spec.md`，仅作 `WIP / NOT_APPROVED` 工程参考；不得据此声明整个交易纵切已批准或完成。
- 本 delegated 指令冻结了两条 PRD 字面边界的组合：恰好 30 分钟时 `InitialState` 返回 `RESERVED`，scheduler 在同一时刻调用 `Advance` 可推进 `PREPARING`。不得擅自合并为一步 `PREPARING`。

## Owned / read-only / 非目标

唯一 owned paths：

- `.scratch/implement-order-production-policy-core/**`
- `services/api/internal/orderproduction/**`

其余全部只读，尤其是 `services/api/internal/paymentobservation/**`、`services/api/internal/wechatpay/**`、`services/api/internal/migrate/**`、`services/api/migrations/**`、order/payment/quote/merchantidentity、`services/api/internal/httpapi/router.go`、`services/api/cmd/**`、apps、`go.mod`、`go.sum`、canonical PRD 与 `CONTEXT.md`。

非目标：DB、migration、取餐号、幂等/audit、scheduler worker、payment Apply、notification、refund、HTTP/客户端、composition、push、PR、integration、deployment 或外部系统写入。

## 依赖、未来调用方与时机

- 当前实现依赖仅为 Go 标准库与固定 code base，无运行时依赖。
- 未来 TX-03/TX-04：在 accepted Payment Observation 已持久化、冻结的预支付记录已加锁且服务端已确认支付成功后，传 `SuccessTime` 与 `pickupAt` 调用 `InitialState`；本 Module 不验支付、不建单。
- 未来 TX-05：scheduler 锁住 Order 后传当前状态、绝对 `observedAt` 与 `pickupAt` 调用 `Advance`；数据库原子写、取号、幂等与审计仍由未来纵切负责。

## 冲突审计

- 初始 worktree 为 detached exact base 且 clean；目标 branch 在创建前不存在。
- exact base 上两个 owned 目录均不存在；仓库内无现存 `orderproduction` Module 或六态生产策略实现。
- 邻接 payment observation、MySQL fixture 与 delivery-map branches 未在本 base 之后占用 owned paths；本 change 不消费其未集成实现。
- 任一并行 change 开始写入上述 owned paths 时立即停止并重新划分 ownership，不覆盖对方改动。

## Red -> Green -> Refactor

1. DRAFT/spec/tasks 先冻结 Interface、状态表、边界、ownership、Gate 与外部资产。
2. Public-seam 编译 Red：外部测试只引用已冻结的 exported Interface，因实现缺失真实编译失败。
3. 逐 tracer 纵切：`InitialState` 的 `>30m`、`<30m`、`=30m`；`Advance` 的阈值前、恰好阈值、漏跑后；五个后继态不回退；非法/废止/空状态、零时间、`pickupAt<=paymentSucceededAt`；并发与重复确定性。每次先见目标行为 Red，再加最小 Green。
4. Refactor 仅在全部行为 Green 后进行；重跑相同 focused 与 race。
5. Writer mutation Gate 在临时副本依次注入五个可逆 mutant：Initial `<` 改 `<=`、Advance `>=` 改 `>`、不足 30 分钟初态错为 `RESERVED`、后继态回退、非法/废止态被接受。每个 source pattern 必须恰好命中一次；每个 focused test 必须以 exit 1 到达指定 `--- FAIL: Test...` 行为断言。build/toolchain/setup 等其他非零退出必须使 harness 自身失败；原工作树保持未变并重跑全绿。

## Candidate 作废与两阶段治理

- 业务候选 `39c0ea43067f30e5befd48290e9cb45711415452` 已由主控第三方 Standards/Governance 审计作废；旧双审和旧动态 Gate 不得继承到 replacement。
- 作废原因：任务状态/checklist 没有闭环；mutation harness 又把任意非零 `go test` 错报为 `MUTATION_KILLED`，不能证明目标行为断言杀死 mutant。
- 两阶段稳定验收：本工件记录完整旧业务候选 SHA 与作废事实；随后只在 owned paths 提交治理修复形成 replacement final SHA；该 final exact SHA 由 immutable external handoff 绑定，并从固定 base 对完整 diff 重跑双轴和 fresh detached 全 Gate。
- 本治理不改变 `orderproduction` 业务 Interface 或策略实现，不扩大 owned paths，不集成、不推送。

## Writer / Review / Verifier / Integration 命令

- focused：`GOPROXY=off GOTOOLCHAIN=go1.26.5 go test ./services/api/internal/orderproduction -count=1`
- focused race/determinism：`GOPROXY=off GOTOOLCHAIN=go1.26.5 go test -race ./services/api/internal/orderproduction -count=20`
- mutation：`bash .scratch/implement-order-production-policy-core/verify-mutation-gate.sh`；先证明 infrastructure failure 被拒绝，再运行五个真实 mutant。
- W3 邻接 schema/事务回归：`.scratch/repair-version-scoped-mysql-migration-fixtures-v13/verify-mysql.sh full`，必须是 fresh loopback-only `mysql:8.0.46-oraclelinux9`。它不证明本纯 Module 的策略正确；本 Module 的决定性证据是 focused/race/determinism/mutation。
- static/build/smoke：`GOPROXY=off GOTOOLCHAIN=go1.26.5 go vet ./services/api/...`；`GOPROXY=off GOTOOLCHAIN=go1.26.5 go build ./services/api/...`；`GOPROXY=off GOTOOLCHAIN=go1.26.5 bash services/api/scripts/smoke.sh`。
- formatting/diff/owned/PII：`test -z "$(gofmt -l services/api/internal/orderproduction)"`；`git diff --check 5e937f3599a16f4813d6021f4cd2dd637c3156a2...HEAD`；只允许两个 owned path；扫描 owned diff 中的 credential/header/token/个人数据 canary，只报告文件与规则摘要。
- review fixed point：`git diff 5e937f3599a16f4813d6021f4cd2dd637c3156a2...HEAD` 与 `git log 5e937f3599a16f4813d6021f4cd2dd637c3156a2..HEAD --oneline`；Standards/Spec 两轴并行绑定 exact candidate SHA。
- verifier：在全新 clean detached worktree checkout exact candidate，重跑 focused/race/mutation（含 infrastructure failure shield）、fresh MySQL full、vet/build/smoke、format/diff/owned/PII/clean；不得修改业务文件。
- integration：本次不做。未来 integration owner 在依赖满足且 exact SHA 独立 PASS 后另行获权处理；rebase/merge 后候选与验证失效。

## 外部资产与未验证边界

| 资产 | owner | 当前状态 | 恢复条件 |
| --- | --- | --- | --- |
| Docker 与 pinned MySQL 8.0.46 image | writer/verifier | writer 本地可用且已完成 fresh Gate；verifier 仍须独立重跑 | 启动 fresh loopback-only container 并由脚本清理 |
| 正式支付、证书、回调、真实订单 DB/scheduler | 客户商户管理员、开发/UAT owner | `N/A_FOR_THIS_PURE_MODULE` / 未验证 | 只能在未来已授权 TX-03/TX-04/TX-05 纵切与真实平台 Gate 中验收 |
| UI | N/A | backend-only `UI0` | 本 change 不建立或宣称 UI1/UI2/UI3 |

## 最小成功标准

- Interface 与冻结状态/时间边界全部由 public-seam 测试覆盖，非法输入 fail closed，重复/并发输出确定且 race clean。
- 五个指定 mutant 全部被 focused test 杀死；fresh MySQL 8.0.46 full、full race、vet/build/smoke 与静态范围检查通过。
- 只提交 owned paths，中文完整 commit；两个 review 轴零 finding；fresh detached exact-SHA 全 Gate PASS；writer 与 verifier worktree clean。
