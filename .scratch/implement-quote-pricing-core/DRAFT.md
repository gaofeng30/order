# DRAFT: implement-quote-pricing-core

## 固定点与状态

- change: `implement-quote-pricing-core`
- status: `THIRD_REPLACEMENT_READY_FOR_EXTERNAL_SHA_HANDOFF`
- `base_sha`: `8bcdf3d6b1ea41529adaa54f463cc118c69e0e25`
- source branch: `codex/order-delivery-integration`
- writer branch: `codex/implement-quote-pricing-core`
- writer worktree: `/Users/vivix/.codex/worktrees/9ede/order`
- `candidate_sha: external-post-commit`；最终完整 SHA 只由 immutable handoff/外部 receipt 绑定，避免把未来 commit SHA 写进自身。
- invalidated candidate: `7a5412546e9d1c59e1213ea668245e60db52e63e`；主控独立 Standards 审查发现 task evidence 不满足逐 task 统一模板，独立 Spec 审查又发现多重非法输入的 stable error priority 未冻结/未对抗测试。该 SHA 的全部 writer review 与 detached receipt 失效，禁止集成。
- invalidated replacement: `8650359395bd0b5117217dee967ec6b09d831a0b`；独立 Standards 审查发现 evidence checker 只比较全文件字段总数，允许单 record 缺字段/缺 phase，并未精确要求旧 receipt 作废语义。该 SHA 的 Spec 0 finding 和全部 writer/review receipt 均失效，禁止集成。
- replacement scope: P1 在 `.scratch/implement-quote-pricing-core/**` 追加逐 fenced record/字段/phase/exit 校验及缺字段、缺 phase、缺旧 receipt 作废语义的负向 failure shield；P2 保留已通过的 Spec、mutation harness 与外部包 public-seam 组合错误测试。除非 fresh Gate 暴露真实业务失败，不修改 `calculator.go`、`errors.go` 或其他业务实现。
- `gate_type`: `W3`（报价与金额语义）
- `ui_level_target`: `UI0`
- `ui_level_actual`: `UI0`
- owner: 本 change 独立 writer
- governance: `GOVERNANCE_PENDING`；`docs/agents/issue-tracker.md` 尚未初始化，本目录 `spec.md` 是主控明确提供的 scratch Spec source。本 change 不配置 tracker，也不把治理待办当业务 blocker。

## 目标、Module 与 seam

在 `services/api/internal/quotepricing` 建立 backend-only、纯 in-process 深 Module，把整数分、逐商品 half-up、逐行数量、合计、溢出和 fail-closed 语义收敛到唯一业务 Interface：

```go
Calculate(input Input) (Result, error)
```

`Input` 只携带调用方已冻结的 `RatePercent` 和价格/数量行；`Result` 返回折扣率快照、同序逐行金额与整单原价/减免/应付。Supporting types 与 typed error 只描述该 Interface 的输入、输出和稳定错误模式，不新增辅助业务函数、clock、repo、adapter、配置或策略选择 seam。

这是无依赖的 in-process Module，测试与未来调用方只穿过同一 Interface。删除它会迫使菜单/详情/结算调用方复制逐商品舍入、数量顺序、溢出和合计规则，因此该 seam 提供 Leverage 与 Locality。

## Owned / read-only / 依赖 / 非目标

唯一 owned paths：

- `.scratch/implement-quote-pricing-core/**`
- `services/api/internal/quotepricing/**`

其余全部只读，尤其是 `docs/product/**`、`docs/quality/**`、`CONTEXT.md`、`services/api/internal/paymentobservation/**`、`orderproduction/**`、`wechatpay/**`、`catalog/**`、`menu/**`、`identity/**`、`merchantidentity/**`、`storefront/**`、`httpapi/**`、`services/api/cmd/**`、migrations、`go.mod`、`go.sum` 与 `apps/**`。

运行依赖仅为 Go 标准库和固定 base。无前置 change 依赖；未来调用方必须先冻结有效折扣率快照：访客传 `100`，员工传已冻结全局 rate。调用时机是未来菜单/详情展示或服务端报价需要一致成交单价时；本 Module 不识别员工、不读取全局配置、不决定配置生效策略。

非目标：Quote/Prepay/Order 创建、身份/白名单、价格版本、P5 保存即生效/定时/营业日生效、TTL、DB、router、migration、客户端、I/O、全局状态、随机、时间、push、PR、integration、deployment 或外部系统写入。

## Interface、数学与错误

- `RatePercent` 是 `int64` 整数百分比数学值，合法范围 `0..100`。`0%` 仅表示 calculator 可表示，不批准 PC 保存 policy。
- `Lines` 必须非空并保持调用顺序。每行只有 `UnitPriceCents int64`（允许 `0`，禁止负数）和 `Quantity int64`（必须 `>0`）；不携带或回传商品 id/name。
- 每行先计算 `roundedUnit = halfUp(UnitPriceCents * RatePercent / 100)`，其中非负余数 `>=50` 向上到下一整数分；再计算 `originalLine = UnitPriceCents * Quantity`、`payableLine = roundedUnit * Quantity`。
- 整单 `OriginalSubtotalCents`、`PayableCents` 分别按调用顺序累加逐行结果；`DiscountCents = OriginalSubtotalCents - PayableCents`。禁止先聚合整单原价再折扣。
- 每个乘法和加法都必须在 `int64` 内可表示；折扣乘法、行数量乘法、跨行合计任一溢出都返回零 `Result` 与稳定 typed/redacted `OVERFLOW`。
- 非法 rate、空行集、负价格、非正 quantity 分别返回零 `Result` 与稳定 typed/redacted `INVALID_RATE`、`EMPTY_LINES`、`INVALID_PRICE`、`INVALID_QUANTITY`。错误文本不含 input values，错误不得 panic 或 wrap 输入。
- 多重非法输入的 stable priority 固定为 `INVALID_RATE → EMPTY_LINES → 按输入行顺序逐行 price → quantity → arithmetic`；首个错误立即返回精确零 `Result`，不得继续到低优先级错误或后续行。
- 相同输入重复/并发确定，race clean，不修改 caller slice，无 I/O/global/random/time。

决定性 worked examples：

1. `101分 × 85% = 86分`，quantity `2`，逐行/整单原价 `202`、应付 `172`、减免 `30`。
2. 两行各 `1分 × 50%`，逐商品均 half-up 为 `1分`，总应付 `2分`；若错误地对 `2分` 小计一次折扣会得到 `1分`。
3. `5分 × 50% = 3分` 后再乘 quantity `3`，原价 `15`、应付 `9`，证明数量在舍入后相乘。
4. `100%` 原价不变；`0%` 和 `0元` 商品合法。
5. 空购物车、非法 rate/price/quantity、折扣乘法/行乘法/cross-line sum overflow 全部 fail closed。
6. 输出同序、输入不变、重复和并发确定。

## Red -> Green -> Refactor

1. DRAFT/spec/tasks 先冻结 Interface、公式、错误、ownership、Gate 与外部边界。
2. Public-seam 编译 Red：外部测试只引用已冻结 exported Interface，因实现缺失真实编译失败；再只补齐可编译 surface。
3. 严格纵向 tracer：每次只加入一个 public-seam 行为测试，取得命名 FAIL/失败断言后写使其通过的最小实现；依次覆盖 worked examples、边界、错误、溢出、顺序/不变性、重复/并发确定性。
4. 全部行为 Green 后才 Refactor；重跑相同 focused/race/determinism。
5. mutation harness 在临时副本注入十一个可逆 mutant，覆盖 half-up、逐商品而非小计、数量在舍入后、rate 边界、rate/empty priority、空购物车、price/quantity priority、折扣乘法/行乘法/cross-line sum overflow、错误时零 Result。每个 source pattern 必须恰好一次；只有 target test exit `1` 且出现指定 `--- FAIL: Test...` 才算 killed。infrastructure 非 `1` 或缺少 marker 必须使 harness fail；原 worktree 不改动。

## Writer / Review / Verifier / Integration Gate

- focused：`GOPROXY=off GOTOOLCHAIN=go1.26.5 go test ./services/api/internal/quotepricing -count=1`
- race/determinism：`GOPROXY=off GOTOOLCHAIN=go1.26.5 go test -race ./services/api/internal/quotepricing -count=20`
- mutation + infrastructure shield：`bash .scratch/implement-quote-pricing-core/verify-mutation-gate.sh`
- evidence structure + failure shield：`bash .scratch/implement-quote-pricing-core/verify-evidence-gate.sh`
- W3 邻接：`.scratch/repair-version-scoped-mysql-migration-fixtures-v13/verify-mysql.sh full`，必须 fresh `mysql:8.0.46-oraclelinux9` 且 loopback-only；它只证明全 `services/api` 邻接回归，不证明本纯 Module 的金额语义。
- static/build/smoke：`GOPROXY=off GOTOOLCHAIN=go1.26.5 go vet ./services/api/...`；`GOPROXY=off GOTOOLCHAIN=go1.26.5 go build ./services/api/...`；`GOPROXY=off GOTOOLCHAIN=go1.26.5 bash services/api/scripts/smoke.sh`。
- formatting/scope/sensitive：`test -z "$(gofmt -l services/api/internal/quotepricing)"`；`git diff --check 8bcdf3d6b1ea41529adaa54f463cc118c69e0e25...HEAD`；base diff 只允许两个 owned paths；敏感扫描只报告文件/规则摘要。
- review fixed point：`git diff 8bcdf3d6b1ea41529adaa54f463cc118c69e0e25...HEAD` 与 `git log 8bcdf3d6b1ea41529adaa54f463cc118c69e0e25..HEAD --oneline`；`$code-review` Standards/Spec 两轴并行绑定 exact candidate。
- verifier：在 fresh clean detached worktree checkout exact candidate，从头重跑 focused/race/mutation（含 shield）、fresh MySQL full、vet/build/smoke、format/diff/owned/sensitive/clean；只读且不得修改业务文件。
- integration：本次不做。未来须独立授权；rebase/merge 或任一实现/spec/tasks/Gate/SHA 变化均使 review/verifier receipt 失效。

## 外部资产与最小成功标准

| 资产 | owner | 当前状态 | 恢复条件 |
| --- | --- | --- | --- |
| Docker 与 pinned MySQL 8.0.46 image | writer/verifier | writer 与 verifier 均须 fresh 创建 | loopback-only 启动并由脚本清理 |
| 真实报价调用方、配置、身份、订单/支付 DB | 后续纵切 owner | `N/A_FOR_THIS_PURE_MODULE` / 未验证 | 未来获授权纵切按各自 Gate 验收 |
| UI | N/A | backend-only `UI0` | 本 change 不建立或宣称 UI1/UI2/UI3 |

最小成功：全部 public-seam 行为、stable multi-invalid priority 与 fail-closed 语义通过，11 个目标 mutant 被行为断言杀死，focused/race/determinism、fresh MySQL 全 API、vet/build/smoke 与静态范围 Gate 通过；只提交 owned paths，中文完整 replacement commit；双轴零 finding；fresh detached exact-SHA 全 Gate PASS；writer/verifier clean；post-commit review/verifier checklist 在实际外部 receipt 前保持 pending。

Writer C/T/V/R：`C=10, T=10, V=8, R=8, total=36`，每项均不低于 `8`，硬阻断为零。`V=8` 仅表示 exact candidate 可形成；不得冒充尚未发生的 independent verifier PASS。
