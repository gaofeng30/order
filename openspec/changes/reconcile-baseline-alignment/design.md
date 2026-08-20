## Context

远端与本地并行推进，各自完成了同一件事：把生效 spec 对齐到 0818 PRD。合并后必须收敛到一份，否则后续 change 的 delta 会按不同的 requirement 标题匹配。

## Decisions

### 保留本地线，取代远端线

判据是**取代成本**，不是先后或优劣：

- 远端 delta 未归档，取代它只需归档 + 吸收两条独有 requirement。
- 本地线已生效，且后续五个已集成 change（会员券删除 ×2、入口屏删除、目录字段删除、六态状态机、取餐时间点）的 delta 全部按其 requirement 标题写成。回滚本地线要重写这五个 change 的全部 delta。

该裁决由用户于 2026-08-21 作出。

### 两条独有 requirement 逐字吸收，不改写

远端那两条覆盖了本地线没写全的内容：取餐号按日累计与跨日核销边界、二维码 token 的生成时机与失效规则、两类一次性订阅消息及拒绝订阅的兜底、统计只来自服务端确认数据与未取餐查询口径。

逐字吸收而非改写措辞，是为了让远端线的作者能在生效 spec 中认出自己的产物，也避免我在转述中引入偏差。

### 归档而非删除，并留取代说明

`adopt-0818-prd-baseline` 是另一条工作线的完整产物，包含 proposal / design / tasks / goal-checkpoint / 门禁脚本。直接删除会让那条线的工作凭空消失，且无法解释。

因此移入 archive 并加 `SUPERSEDED.md`，写明：发生了什么、为什么保留另一条、成果去了哪里、哪一部分未被取代（旧 PRD 改薄指针的动作由它完成并生效）。

### 门禁反向检查「不得有第二份」

上一轮的教训是：重复工作直到 merge 才暴露。因此门禁不只检查生效 spec 的内容，还扫描 `openspec/changes/` 下是否存在第二份未归档的 `specs/mvp-product-baseline/spec.md`——有就报名字。下次再出现并行对齐，在 change 阶段就会被点出来。

### 顺带补齐一处已记录的门禁词表缺口

`complete-baseline-traceability` 的 `design.md` 曾记录：残留门禁在 archive 后会对 `Matrix cites a retired dimension` 场景误判，因其 WHEN 子句必须枚举被禁术语才能禁止它们，而 THEN 子句「PRD 实施验收失败」不在否定词表内，当时判定「留给下一个触及本 spec 的 change」。本 change 正是那个 change，已把「失败」补入词表。

## Risks / Trade-offs

| 风险 | 处置 |
| --- | --- |
| 远端线作者可能不认同取代判定 | `SUPERSEDED.md` 完整记录了判据与成果去向，成果本身逐字保留在生效 spec 中；如需反向取代，回滚路径也已写明 |
| 归档后 `verify_baseline.py` 因路径深度失效 | 如实记录，不修改归档产物。其约束已由新门禁继承并扩展 |
| 生效 spec 增至 15 条，后续 change 需按新标题写 delta | 两条为 ADDED，未改动任何既有标题，已集成 change 的 delta 不受影响 |

## Invalidation conditions

- 用户或远端线维护方推翻取代判定；
- 生效 spec 被其他 change 修改，导致两条吸收项的标题不再存在；
- 0818 PRD 的基线地位再次变化。
