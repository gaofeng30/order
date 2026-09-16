## Context

当前 standard receipt verifier 只接受 candidate checkpoint 中唯一的 `## Module base` 或 `## Current Runtime` 区段，并要求 `state=CANDIDATE`。目标 governance candidate 的 immutable checkpoint 使用更早的顶层表格格式，实际保留 `lifecycle=IMPLEMENTING`、`candidate_sha=none`；其归档 commit 又正确地保持 candidate checkpoint/tasks byte-identical。修历史会破坏 exact-SHA 证据，直接改受保护 v1 judge/runner 会使 bootstrap 与第五 binding 的 blob Gate 失败。

同时，标准 `verify_receipt.py --list` 会枚举所有名为 `lifecycle-receipt.md` 的文件。若把不兼容 receipt 放到标准文件名，现有全量 Gate 将永久失败。因此兼容证据必须使用独立文件名并由新入口显式选择。

## Goals / Non-Goals

**Goals:**

- 只为 change `enforce-miniprogram-user-regression-gate`、candidate `f5719e9…`、archive `b53cb52…` 和固定 checkpoint/tasks hashes 接受一次 pre-convention 兼容解释。
- 复用受保护 v1 的 schema、form、Git ancestry/archive/ownership/task/receipt-history 和 controlled profile 验证；不复制整套 judge。
- 保持旧 v1 `--list`/`--chain`、五项 bindings、Mini Program mechanical profile 和历史 receipt 语义不变。
- 在输出中同时报告原始 checkpoint `IMPLEMENTING`/`none`、兼容规则 ID 和 `actor_independence=NOT_PROVEN_BY_MECHANICAL_REPLAY`。

**Non-Goals:**

- 不为其他旧 checkpoint、其他 change/SHA/路径提供兼容；不把 stale checkpoint 一般化为 candidate。
- 不修改任何既有控制工具、历史证据或产品行为；不证明 Agent 身份、UI2/UI3、真实微信、远端 required check。
- candidate 不创建 governance receipt；集成、归档、receipt-only commit 和 exact-R 验证仍是分离阶段。

## Decisions

### D1. 新增窄适配器，不修改受保护 v1 judge

新增 `verify_legacy_miniprogram_receipt.py`。入口只接受固定 change，先验证当前仓库中 v1 judge、schema、runner、bindings 和 Mini Program wrapper 的 Git blob 等于在 exact-B 冻结的值，再从 `legacy-lifecycle-receipt.md` 解析相同 v1 schema。

适配器随后调用受保护 v1 verifier，但只在调用作用域内替换两个输入边界：receipt 枚举增加唯一 legacy 文件；checkpoint parser 只在输入 SHA256 等于 `535a53…` 且顶层表格关键字段逐项精确匹配时返回兼容 candidate-stage view。调用结束即恢复原函数。最终 payload 追加 recorded state、recorded candidate 和兼容规则 ID，避免把归一化 view 叙述成历史原文。

选择这个方案是因为复制完整 verifier 会产生第二套长期 judge，直接修改 v1 会破坏已绑定字节，改历史 checkpoint 则毁掉 exact-SHA 事实。

### D2. 使用独立 legacy receipt 文件名

目标 receipt 固定为 `openspec/changes/archive/2026-08-21-enforce-miniprogram-user-regression-gate/legacy-lifecycle-receipt.md`。适配器拒绝同 change 的标准 receipt、重复 legacy receipt、symlink、错误目录或任何后续 edit。

这样旧 `verify_receipt.py --list/--chain` 继续验证其原有集合；新的显式入口才验证这一个兼容对象。该命名不是未来扩展机制，所有后续 change 仍必须使用标准 `lifecycle-receipt.md` 和 candidate checkpoint 格式。

### D3. 兼容解释依赖全部精确事实，而非 checkpoint 状态单项

允许兼容的前提同时包括固定 base/candidate/archive SHA、checkpoint/tasks SHA256、archive sole parent、完整 archive diff、owned paths、receipt one-add history、binding definition hash、受保护 blobs、exact profile `MECHANICAL_PASS` 和 clean worktree。recorded attestation 仍是不可信审计字段，不能补偿任何失败。

原始 checkpoint 的 `IMPLEMENTING`/`none` 被明确输出；PASS 的含义是“该已知 pre-convention candidate 的归档与后置机械证据一致”，不是“历史 checkpoint 曾记录 CANDIDATE”。

### D4. 生命周期顺序保持双独立验证

本 repair candidate `C` 先由两个独立 Agent 在不同 clean detached worktree 验证。双 PASS 后按已授权边界纯快进 local main，并只做确定性 archive `A`。之后 integrator 单独添加 legacy receipt 得到 `R`；另一独立 Agent 在 exact `R` 运行生产兼容入口并派生 PASS。任何 `C/A/R` bytes、rebase 或路径变化都使旧验证失效。

## Risks / Trade-offs

- [私有函数适配可能随 v1 变化失配] → 入口先固定 v1 judge/schema/runner/bindings/wrapper Git blobs；任一变化直接 fail closed，必须用新 OpenSpec 重新设计。
- [compatibility view 被误读为历史 CANDIDATE] → payload 强制输出 recorded `IMPLEMENTING`/`none` 和 exact-only 规则 ID；spec 禁止该 view 用于其他 change 或 actor claim。
- [legacy 文件不进入旧 `--list`] → exact-R Gate 必须同时运行旧 `--list`/`--chain` 和新显式 verifier；两者缺一即 NO-GO。
- [adapter/archive 后被修改] → archive Gate 比较 C/A tool/test bytes；R 必须是唯一 legacy receipt add，exact-R verifier比较 A/R adapter blobs并检查 ending clean。

## Migration Plan

1. 在独立 writer 从 exact-B 建立 OpenSpec，先记录现有 standard verifier 的确定性 Red。
2. 以测试先行实现 exact-only adapter，完成 focused/full/forward Gates并产生 `C`。
3. 两个独立 Agent 对 exact `C` PASS 后，本地纯快进并归档为 `A`，重验 archive diff 和 canonical delta。
4. 从 `A` 追加唯一 legacy receipt 为 `R`；另一独立 Agent 对 exact `R` 运行旧 v1 Gate与新兼容 Gate。
5. exact-R PASS 后关闭 governance receipt TODO；回滚只需在尚未开始下一模块时回退 `R`、`A` 和 `C` 的本地纯快进，绝不改历史 candidate/archive。

## Open Questions

None. 该兼容入口、对象、路径、hash、验证顺序和非目标均固定。
