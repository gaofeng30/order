## Context

`openspec/specs/mvp-product-baseline/spec.md` 于 2026-08-12 由 archived change `formalize-mvp-product-baseline` 建立，是一期产品行为的唯一生效 spec，也是后续业务 change 的验收事实源。

2026-08-19 客户评审推翻了其中大部分规则。评审结论已存为 `docs/product/online-ordering-system-prd-0818-review.md`（第 1 顺位证据），并于 `2f2db4a` 落入 `docs/product/online-ordering-system-prd-0818.md`。spec 至今未同步。

`services/api` 目前只有 catalog（菜品目录）能力，没有订单、支付、核销、鉴权或名单实现；两个前端仍为 mock/内存态。因此本次对齐**不涉及任何运行态数据或已实现行为的迁移**，只有规格文本。

## Goals / Non-Goals

**Goals**

- 使生效 spec 与当前第 1 顺位证据一致，解除后续前端与后端 change 的规格阻塞。
- 保留基线中未被客户评审触及、仍然正确的约束（幂等与审计、敏感数据边界、门店时区、整数分、订单快照、十二 Gate 链、可追踪性）。
- 让每条新规则可逐条追溯到评审记录条目或 0818 PRD 章节。

**Non-Goals**

- 不修改 `openspec/specs/mvp-product-baseline/spec.md` 本体。按仓库既有 OpenSpec 约定，change 只承载 delta，archive 时才应用到生效 spec。
- 不实现任何被重新定义的行为，不碰 `apps/**`、`services/**`、schema、部署配置。
- 不修改产品文档、合同、客户清单或其他 spec。
- 不处理 `feat/member-coupon` 分支的废弃。

## Decisions

### 为什么是一个 change，而不是按主题拆开

`AGENTS.md` 要求「可以分别验收或回滚的结果必须拆开」。本次 17 条 delta 看似可以按状态机、定价、权限、取餐模型拆成四个 change，但它们**无法独立回滚**：

- 六态依赖「支付成功才建单」（否则 `待支付` 无处安放），后者依赖删除软预占；
- 删除库存依赖新增按日售罄开关，否则商品可售性无定义；
- 取餐时间点依赖删除即时单，否则「餐段空档期」规则悬空。

任何部分回滚都会让 spec 自相矛盾（例如六态 + 保留软预占）。验收判据也只有一个：**spec 与评审记录一致且无残留已废止规则**。因此按「一个可独立理解、实现、验收和回滚的主能力」判定，这是一个 change。

### 为什么用 REMOVED + ADDED 而不是 MODIFIED

OpenSpec 的 MODIFIED 按 requirement 标题匹配。以下四条的标题本身就编码了被推翻的事实，继续沿用会留下错误标题：

| 原标题 | 问题 |
| --- | --- |
| `Orders use one **nine**-state production state machine` | 状态数变了 |
| `Employee price is an optional fixed **per-product** amount` | 定价形态变了 |
| `Every first-phase order uses one fixed pickup **slot**` | 时段变时间点 |
| `Merchant permissions use **four** server-enforced roles` | 角色数变了 |

因此这四条走 REMOVED（带 Reason 与 Migration）+ ADDED 新标题。`Inventory ...` 与 `Order submission uses a bounded atomic soft hold` 是纯删除，只有 REMOVED。

`Product sources`、`First-phase scope`、`Cancellation and refund`、`Employee identity` 四条标题仍然准确，走 MODIFIED 整条替换正文。

### 为什么十二 Gate 链不动

0818 PRD §13.2 注销了原第 13 项（会员券合同补充协议），并列出了需按新口径复核的既有 Gate（订阅模板 6 类收敛为 2 类、微信支付结果查询接口升为硬依赖）。

但 Gate 13 从未进入本 spec——它只存在于 0818 PRD 提案中。而订阅模板数量与支付查询接口属于 Gate 3（餐饮类目）、Gate 9（API 安全状态）、Gate 10（交易结算管理）的**实施细节**，spec 的十二 Gate 链只约束顺序、状态规则和敏感数据边界，不枚举这些细节。

因此该 requirement 保持不变。把 PRD 级的复核清单塞进 spec 会制造两份台账漂移，正是 0818 PRD §13.2 明确要避免的。

### 「未取餐」为什么写成查询口径而不是状态

评审第 21 条删除了待取超时状态。若为「营业日结束仍未核销」新增状态，等于把删掉的东西换个名字加回来，并且会让六态变七态、影响所有状态机断言。写成查询口径（一个筛选条件）既满足运营需要，又不触碰状态集合。该判定写在六态 requirement 内部，不单独成条。

### 对账兜底为什么独立成一条 requirement

它可以被并进 `Orders exist only after confirmed WeChat payment`，但两者的验收性质不同：前者是同步路径的正确性，后者是异步补偿的完整性，且后者需要独立的幂等与失败转人工场景。分开写让实现 change 可以先做同步路径、再做对账任务，而不必把两段验收捆在一起。

## Risks / Trade-offs

| 风险 | 处置 |
| --- | --- |
| 客户评审记录尚未取得书面签认（0818 PRD §16.4 C1） | spec 依据的是已记录的客户正式确认，符合 authority order。签认是商业动作，不阻塞 spec 对齐；若客户后续推翻，按新的第 1 顺位证据再开 change |
| `online-ordering-system-prd.md` 仍在仓库中且团队可能仍在依据它开发 | MODIFIED 的 authority order requirement 明写该文档中冲突条款失效，并新增 `Superseded baseline clause is cited` 场景使其可检查 |
| 删除超卖保护后现场缺货只能人工退款 | 这是客户明确接受的取舍（0818 PRD §16.1 R1/R2）。spec 中以 `MUST NOT 提供任何自动的产能或超卖保护` 显式表述，避免被后续 change 当作遗漏"补上" |
| 仓库无 `openspec` CLI，无法执行 `openspec validate --strict` | 以可执行的结构/内容检查替代：requirement 标题与基线精确匹配、每条 requirement 至少一个 scenario、delta 中无已废止概念的肯定性表述。工具缺口如实记录，不冒充 strict PASS |

## Invalidation conditions

出现以下任一情况时，本 change 的结论失效，必须重新规划：

- 客户对 2026-08-19 评审记录作出新的正式确认，改变其中任一条；
- `docs/product/online-ordering-system-prd-0818.md` 的 §1–§14 在本 change 集成前再次变更；
- `openspec/specs/mvp-product-baseline/spec.md` 被其他 change 修改，导致 MODIFIED/REMOVED 的标题不再匹配；
- 仓库引入 `openspec` CLI 且 `openspec validate --strict` 对本 delta 报错。
