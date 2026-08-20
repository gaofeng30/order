## Context

仓库在 `2f2db4a31f66f992997880a02b438c9690bbb845` 同时存在三种互相冲突的产品表达：旧 canonical PRD、现行 canonical `mvp-product-baseline`，以及已经吸收 2026-08-19 客户澄清的 0818 PRD。前两者仍要求数量库存/软预占、九态、接单、四角色和逐商品员工价；0818 PRD §1–§14 则冻结为无数量库存、仅预约、支付成功才建单、六态、无接单、主/子账号和全局折扣率。

本 change 只做 W0 文档事实源收敛。0818 PRD、review 和现有 canonical spec 均为只读证据；canonical spec 只会在本 change 集成后由 OpenSpec archive 流程应用 delta，但 archive 不属于本 change 的授权或任务。

## Goals / Non-Goals

**Goals:**

- 让 0818 PRD §1–§14 成为唯一有效产品 PRD，并保留 review 作为客户裁决证据。
- 用完整 delta 冻结 §3 一期范围和 §12 的 I1–I16，明确删除旧冲突契约。
- 让旧 PRD 只承担废止入口与跳转，不再提供可选择的第二套业务正文。
- 把 P1–P5 绑定到各自下游模块，不让未确认选择泄漏进实现。

**Non-Goals:**

- 不修改 0818 PRD/review、现有 canonical specs、业务代码、API、schema、运行配置、其他 changes 或治理文件。
- 不解决 P1–P5，不评估或修复当前代码与新基线之间的差距。
- 不推送、部署、集成或 archive；candidate 提交只包含 owned paths，后续生命周期动作仍需独立 Gate 与授权。

## Decisions

### Use one authoritative PRD and one evidence record

选定 0818 PRD §1–§14 作为唯一有效产品正文；review 只证明客户确认来源。旧 PRD 将被替换为薄指针，不保留旧正文。没有采用“双文档按优先级并存”，因为这仍允许后续 writer 误引被裁决失效的低顺位规则。

旧 PRD 薄指针保留 `§13.2 外部 Gate` 的重定向锚点，明确 0818 PRD 对旧 §13.2 的引用由 canonical `mvp-product-baseline` 中的十二 Gate requirement 承接；不复制第二份 Gate 台账。

### Replace obsolete product requirements instead of weakening them

delta 对仍成立但内容变化的 requirement 提供完整 `MODIFIED` 块；对名称和语义均已失效的库存、软预占、九态、固定时段、四角色、逐商品员工价 requirements 使用 `REMOVED`，并以新的可测试 requirements 承接。没有在旧 requirement 下增加例外，因为那会形成新旧行为双轨。

I1–I16 显式映射为：预约/日期餐段/离散时间点、无库存、支付确认建单与对账、六态与自动排产、取消退款、身份、全局折扣、两角色、取餐号与消息、服务端事实源。内容检查必须证明 16 个编号均有规范落点，并证明即时单、数量库存、软预占、九态、接单、四角色、会员券和逐商品员工价只以“已废止/不得存在”语义出现。

### Keep unresolved product choices out of this W0 contract

P1 只阻塞营业状态切换归属；P2 只阻塞跨营业日未取订单处置时限；P3 只阻塞 PC 扫码登录的会话/设备信任；P4 只阻塞附加手机号数量模型；P5 只阻塞全局折扣率生效时机。任何下游 change 命中对应边界时必须先取得客户确认。本 W0 只收敛已确认范围与不变量，因此 dependencies 与 external assets 均为 none。

### Verify the documentation change with one real W0 Red/Green check

实现开始前，对旧 PRD 执行同一个标准库内容检查：要求文件为短薄指针、包含 0818 目标与废止声明、保留外部 Gate 重定向，且不存在旧有效正文。当前 985 行旧 PRD必须使检查失败；最小替换后同一检查必须通过。Refactor 重跑同一检查，并追加 strict、源文件 byte guard、`git diff --check` 和 owned-path audit。

## Risks / Trade-offs

- [Risk] 0818 PRD §13.2 仍以文字引用旧 PRD 的十二 Gate。→ 旧 PRD 保留同名重定向锚点，canonical delta 完整承接十二 Gate，不复制旧业务正文。
- [Risk] 大量旧规则被删除后，下游代码仍与新基线不一致。→ 本 W0 只建立事实源；每个受影响业务模块另建独立 W1–W3 change，不把代码改动塞进本 change。
- [Risk] P1–P5 的暂定文字被误当成确认结论。→ delta 和 checkpoint 都逐项标记定向产品决策阻塞，验收禁止把暂定口径写成 MUST。
- [Risk] archive 后 canonical spec 变化导致旧候选证明失效。→ candidate 只绑定 exact SHA；proposal/design/spec/tasks、base、验收命令或 candidate 任一变化均重新执行 writer Gate 与独立验证。

## Migration Plan

1. 经明确批准后进入 IMPLEMENTING，在修改旧 PRD 前运行 focused check 得到真实 Red。
2. 只把旧 PRD替换为废止薄指针与外部 Gate 重定向，得到 Green；不编辑只读共享契约。
3. Refactor 后运行 W0 writer Gate，提交仅 owned paths 的 exact candidate。
4. 由不同 verifier 在 clean detached worktree 对 exact candidate 只读重跑全部声明检查。
5. 只有 exact candidate 获独立 PASS 且获得单独集成授权后，才可集成本地 main；main 推进或内容变化即形成新 candidate 并重验。

回退仅恢复 `docs/product/online-ordering-system-prd.md` 与本 change 路径，不触碰业务运行数据。禁止 push、deploy、integration 和 archive。

## Open Questions

本 W0 无阻塞性未决问题。P1–P5 保持为下游定向阻塞，不在本 change 中回答。
