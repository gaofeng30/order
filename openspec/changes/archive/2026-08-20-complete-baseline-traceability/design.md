## Context

`realign-mvp-product-baseline` 的 delta 划分依据是「客户评审是否触及该 requirement」。对 12 条基线 requirement 逐条判定时，`The baseline is traceable and has no behavioral TODO` 被判为未触及——它讲的是 PRD 追踪矩阵的完整性，不是任何具体业务规则。

该判定在 requirement 正文层面成立，在 scenario 层面不成立：其 `Traceability matrix is checked` 场景把「九态」「四角色」写成了追踪矩阵必须覆盖的具体维度。这两个维度已被同一批 delta 删除。

## Goals / Non-Goals

**Goals**

- 消除生效 spec 内部关于状态数与角色数的自相矛盾。
- 把「未被 delta 覆盖的 requirement 仍有残留」这条漏检路径变成可执行门禁。

**Non-Goals**

- 不重做 `realign-mvp-product-baseline`。该 change 已按其声明范围完成独立验证并集成；范围本身不完整属于新发现的缺陷，按新 change 处理，不改写已集成历史。
- 不扩大修正范围。除这一条 requirement 外，生效 spec 的其余 11 条已由残留门禁确认无残留。

## Decisions

### 为什么不回退重做上一个 change

`AGENTS.md` 规定「实现、规格、任务或 SHA 在验证后发生任何变化，旧验证立即失效」。这条约束的对象是**尚未集成的候选**。`realign-mvp-product-baseline` 已经完成独立验证并集成到本地 `main`，其验证结论对它当时声明的范围仍然有效——缺陷不在于它做错了什么，而在于范围划少了一条。

改写已集成历史会让那次独立验证的 SHA 失去指向，且无法解释为何一个已验证的候选被静默替换。开新 change 修正是可追溯的做法：缺陷、成因、门禁漏洞与修复各自留痕。

### 为什么门禁改成检查「未覆盖的生效 requirement」

上一轮的门禁 `check_realign.py` 只看 delta 文件：它验证 delta 内部一致（标题匹配、scenario 齐全、无残留表述），但对「delta 没写的部分」完全无感。真正要防的是**遗漏**，而遗漏按定义不在 delta 里。

新门禁反过来做：以生效 spec 为输入，减去所有待集成 delta 覆盖的标题，对剩余部分执行残留检查。这样任何一条「本该改却没改」的 requirement 都会被点名。两个门禁互补，都保留。

### 为什么两个 delta 必须同批 archive

`realign-mvp-product-baseline` 的 delta 若单独应用，生效 spec 会有一段时间处于自相矛盾状态（六态 + 要求覆盖九态）。两者同批应用可以避免这个中间态。这一点写进本 change 的依赖声明，作为集成前置条件。

## Risks / Trade-offs

| 风险 | 处置 |
| --- | --- |
| 同类遗漏可能还存在于其他 spec | 新门禁只覆盖 `mvp-product-baseline`。其余 spec（`api-service-bootstrap` 等）未被客户评审触及，暂不扩展；若后续有 change 修改它们，同一门禁模式可复用 |
| 残留词表与否定词表是启发式的 | 词表按当前已删除概念枚举，不追求通用。词表遗漏会导致漏检，误报则由否定词表消解；两者都在脚本内显式可读，便于后续 change 增补 |
| 门禁依赖「待集成 delta」的目录扫描 | archive 后 delta 移出 `openspec/changes/`，`covered` 集合变空，门禁转为对整份生效 spec 做残留检查——这正是 archive 后应有的更严口径，不需要改脚本 |

## Invalidation conditions

- `realign-mvp-product-baseline` 在集成后被回退或修改；
- 生效 spec 被其他 change 修改，导致 MODIFIED 标题不再匹配；
- 客户对 2026-08-19 评审记录作出新的正式确认，改变状态数或角色数。
