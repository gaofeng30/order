## Context

生效 spec `mvp-product-baseline` 已把会员等级与优惠券列入一期排除范围，并规定定价机制为全局单一折扣率。小程序端仍完整实现着被删除的那一套。

前端删除工作按 UI1 证据可得性拆成两个 change：小程序端有 Node 测试 harness（`apps/wechat-miniprogram/tests/page-harness.js`，当前 13 项通过），可取得真实 UI1 证据；`apps/web-admin` 仓库内没有任何 runner，只能取得静态解析级证据。本 change 只做小程序端。

## Goals / Non-Goals

**Goals**

- 让小程序端不存在任何会员券的页面、路由、组件、全局态、种子数据与接口契约。
- 让结算页在没有任何优惠机制的前提下走通加购到下单全链路。
- 把「小程序端页面与算价链必须符合一期范围」变成一条可执行的 UI1 断言，供后续两个删除类 change 复用。

**Non-Goals**

- 不动 `apps/web-admin/**`。
- 不实现全局折扣率。
- 不做其他删除类 change 的内容（标签、过敏原、月售、库存位、品牌选择页、商户端工作台）。

## Decisions

### 等级折扣与优惠券一起删，不留占位

生效 spec 的机制是**全局单一折扣率**，由 PC 后台一个数值驱动；当前实现是 `Level` 表驱动的等级折扣，一人一档、每档一个折扣率。两者是不同的数据模型，不是同一机制的改名。

保留等级折扣当作「未来员工折扣的占位」会同时违反两条约束：生效 spec 的 `First-phase scope is closed and singular` 明确排除会员等级且不得预留；`AGENTS.md` 禁止「过渡双轨」与「假设未来需求的方案」。

代价是删除后到全局折扣率实现之间，结算页只显示商品小计、不显示任何优惠。这是用户已确认接受的中间状态，且它比留一个语义错误的占位更诚实：读代码的人看到的是「还没有优惠机制」，而不是「有一套等级折扣」。

### 为什么新建 capability 而不是挂在 `mvp-product-baseline` 下

`mvp-product-baseline` 是产品行为基线，描述的是业务规则，不描述某个客户端的文件结构。「小程序端不得存在排除能力的页面与全局态」是实现侧的一致性约束，属于另一个层面。

新建 `miniprogram-scope-conformance` 让后续两个删除类 change 可以直接 MODIFY 它追加断言，而不必各自发明一套检查。

### UI1 断言写成「缺失断言」而不是「快照对比」

删除类改动最容易出的错是删了一半：页面目录删了但 `app.json` 路由还在，或者接口契约删了但 `globalData` 还留着空数组。快照对比只能说明「和上次不一样」，说明不了「哪一处没删干净」。

因此断言逐项指名：路由列表、页面目录、`globalData` 键、`api.js` 导出、`data.js` 导出、模块文件、WXML 片段。每一条失败都直接指向具体残留位置。

### 结算页保留下单链路，不降级为占位

删掉优惠明细卡后，结算页仍然必须能完成「读购物车快照 → 展示明细 → 生成订单 → 跳结果页」。UI1 用例覆盖这条完整链路，而不只是断言优惠卡消失。否则删除动作可能在删优惠的同时打断下单，而断言不会发现。

## Risks / Trade-offs

| 风险 | 处置 |
| --- | --- |
| 删除后原型演示能力下降（无优惠展示） | 用户已确认接受。全局折扣率由后续新增类 change 补齐 |
| `utils/api.js` 与 `utils/data.js` 同时被本 change 和 `strip-retired-catalog-fields` 需要 | 两者 owned paths 冲突。本 change 先执行并取得这两个文件的所有权，`strip-retired-catalog-fields` 声明对本 change 的依赖，不并行 |
| 既有 UI1 用例 `all-scope promo` 会被删除动作打破 | 该用例在 Green 阶段改写为无券路径，改写内容与删除范围一一对应，不是为了让测试通过而放宽断言 |
| PC 后台仍保留会员券页面，两端短暂不一致 | 两端 mock 数据层本就独立，不存在跨端数据不一致。`remove-member-coupon-admin-pages` 随后处理，其 UI1 阻塞不影响本 change |

## Invalidation conditions

- 客户对会员等级或优惠券作出新的正式确认，改变一期范围；
- 生效 spec 的 `First-phase scope is closed and singular` 或 `Employee pricing uses one global discount rate applied per product` 被修改；
- `apps/wechat-miniprogram/tests/page-harness.js` 被其他 change 修改，导致本 change 的 UI1 用例不可执行。
