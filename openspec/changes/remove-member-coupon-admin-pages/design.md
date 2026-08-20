## Context

PC 网页后台是零构建的浏览器全局脚本：`index.html` 按顺序加载 `data/*.js`、`ui/*.js`、`pages/*.js`、`app.js`，模块挂在 `window` 上。没有打包器、没有模块系统、没有测试运行器。

`remove-member-coupon-capability` 已完成小程序端的同类删除并取得真实 UI1。本 change 处理 PC 端，面对的核心差异是**证据可得性**。

## Goals / Non-Goals

**Goals**

- 让 PC 后台不存在任何会员券的页面、导航、内存态、种子、契约与文案。
- 在没有浏览器 runner 的前提下，取到当前可得的最强证据，并把不可得的部分如实标注。
- 把「PC 后台必须符合一期范围」变成一条可执行断言，供后续删除类 change 复用。

**Non-Goals**

- 不为 `apps/web-admin` 建设浏览器或 DOM runner。
- 不动小程序端、后端、生效 spec 或产品文档。

## Decisions

### UI1 记 BLOCKED_EXTERNAL，不降级冒充

质量门禁的决策表把 W2×UI0 标为硬阻断，同一文档又记载仓库当前没有锁定的浏览器 runner、缺少 UI1 资产即 `BLOCKED_EXTERNAL`。两条合起来的结论是：**这个 change 在当前仓库条件下拿不到完整的 W2 验收，而这是被文档预见并规定了处理方式的情况**，处理方式是如实记 `BLOCKED_EXTERNAL`，不是用静态检查冒充 UI1。

因此本 change 的 `ui_level_actual` 记 UI0，UI1 记 `BLOCKED_EXTERNAL` 并写明恢复条件。是否在此状态下集成，属于 lane 决策，不由本 change 单方面认定为已验收。

### 不顺手建 runner

给 `apps/web-admin` 加 DOM 垫片以取得渲染级证据，技术上可行，但那是建设测试基础设施，与「删除注定要删的代码」是两件事。`AGENTS.md` 明确禁止顺手处理相邻问题，也禁止一个 change 承载可独立验收的两件事。若要补 runner，应独立立项，届时本 change 与后续 PC 端 change 都能受益。

### 数据层用 Node 真实加载，不只做正则

`data/seed.js` 与 `data/api.js` 是无 DOM 依赖的纯 JS，只往 `window` 上挂对象。因此可以在 Node 的 `vm` 沙箱里配一个 `window` 垫片真实执行它们，然后断言导出内容与行为——包括调用 `deleteProduct` 验证它不再读写券数据、返回值不再带 `disabledCoupons`，以及菜品/订单/分类/营业设置契约在删除后仍可调用。

这比静态正则强得多：正则只能证明源码里没有某个词，运行态断言能证明**契约本身没有被删坏**。页面与导航层依赖 DOM，只能静态断言，该边界写在门禁脚本的文件头注释里。

### 文案残留做全目录扫描，不只查改动过的文件

小程序端那轮的经验：删除类改动最容易漏的不是代码而是注释、分节标题与样式类名。本门禁对 `apps/web-admin` 下所有 `.js` / `.html` / `.css` 做全目录文本扫描，命中即指名文件与命中词。

这条在实施中立刻生效——它抓到了 `app.css` 里一处「二期能力标签」样式分节，那是逐文件人工检查会漏掉的位置。

### 保留契约完好性断言

删除五组契约时最大的风险是连带打断菜品或订单。因此门禁不只断言「被排除的方法不存在」，还断言 `listProducts` / `deleteProduct` / `listOrders` / `listCategories` / `getSettings` 仍是可调用函数。缺了这一条，一次删过头的改动会静默通过。

## Risks / Trade-offs

| 风险 | 处置 |
| --- | --- |
| 无 UI1 证据，页面渲染是否被删坏无法验证 | 已记 `BLOCKED_EXTERNAL` 并写明恢复条件。缓解措施是数据层运行态断言 + 全部 JS 可解析 + 契约完好性断言，覆盖非渲染部分的绝大多数失败模式 |
| 导航分组删除后路由回退行为未验证 | 同上。`app.js` 的默认路由 `dashboard` 未被本 change 触碰 |
| 两端 mock 数据层独立，删除顺序不影响正确性 | 本 change 与小程序端 change 无 owned paths 交集，先后顺序无约束 |

## Invalidation conditions

- 客户对会员等级或优惠券作出新的正式确认，改变一期范围；
- 生效 spec 的一期范围或定价 requirement 被修改；
- `apps/web-admin` 引入浏览器 runner，此时 UI1 的 `BLOCKED_EXTERNAL` 应重新评估并补齐证据。
