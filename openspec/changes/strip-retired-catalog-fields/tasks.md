## 0. 门禁声明

```yaml
change: strip-retired-catalog-fields
gate_type: W2
ui_level_target: UI1
ui_level_actual: UI1
owner: worktree-strip-retired-catalog-fields
worktree: .claude/worktrees/strip-retired-catalog-fields
owned_paths:
  - apps/wechat-miniprogram/**
  - apps/web-admin/**
  - openspec/changes/strip-retired-catalog-fields/**
base_sha: 8071853cda0a2bd0dd3af97cfec19e37b6aee4ff
candidate_sha: external-post-commit
external_assets: none
dependencies:
  - remove-member-coupon-capability（已归档于 base_sha）
  - remove-member-coupon-admin-pages（已归档于 base_sha）
  - remove-retired-entry-screens（已归档于 base_sha）
```

门禁命令：

```
cd apps/wechat-miniprogram && npm test
node openspec/changes/strip-retired-catalog-fields/checks/check_catalog_fields.js <tree-root>
```

工具边界：仓库未安装 `openspec` CLI，strict 校验记 `BLOCKED_EXTERNAL`。小程序 UI1 来自 Node harness，PC 后台 UI1 来自本地静态服务器加浏览器实际运行，均不构成微信开发者工具或真机证据。

## 1. Boundary and approval

- [x] 1.1 实测收窄范围。
  - Evidence: 规划阶段估计本 change 需覆盖用户端菜单与详情，实测发现两页早已切到 API 目录，且既有 UI1 的 `UNSUPPORTED_PRODUCT_FIELDS = ['stock','sold','status','tags','image',...]` 已把这些字段挡在用户端商品对象之外。真实范围收窄为商户端页面与两端 mock 种子。
- [x] 1.2 与前三个 change 的串行关系。
  - Evidence: 三者均已集成并归档于 `base_sha`，共享 `apps/**`。`remove-retired-entry-screens` 删除了 `admin-dashboard` 整页，本 change 原本对该页「库存告急 + 月售排行」的改动因此自动消失，重叠面由四处降为两处（`utils/data.js`、`tests/*`），且两处已无并发写者。
- [x] 1.3 确定删 `stock` 但保留 `status`。
  - Evidence: 评审 §3「商品只有可售和售罄两种销售状态」。`status` 是一期唯一的可售性机制，必须保留；被删的是数量维度。断言因此分两侧写，见 `design.md`。

## 2. Red

- [x] 2.1 小程序端：新增 `tests/catalog-fields-ui1.test.js`。
  - Red: `node --test tests/catalog-fields-ui1.test.js` → `tests 5 / pass 1 / fail 4`。四项分别因种子仍带 `tags/allergens/sold/stock`、契约仍做库存校验、商户列表仍渲染库存与月售、编辑页仍有库存输入而失败。第 5 条「售罄链路端到端可用」为回归保护，此时即应通过。
- [x] 2.2 PC 端：新增 `checks/check_catalog_fields.js`（数据层 Node 运行态 + 页面层静态）。
  - Red: `node checks/check_catalog_fields.js <base-tree>` → `exit=1`，`CATALOG_FIELDS_GATE=FAIL`，四项失败并逐一指名：`p001.tags still seeded`、`api.js still handles a quantity`、`products page still renders a quantity`、`dashboard still shows a low-stock todo`。

## 3. Green

- [x] 3.1 两端种子移除四类字段，保留 `status`。
  - Evidence: `apps/wechat-miniprogram/utils/data.js` 与 `apps/web-admin/data/seed.js` 的 `MENU` 逐条去掉 `sold` / `stock` / `tags` / `allergens`。`RANK` 保留——它是 PC 销量排行的独立数据源，与商品 `sold` 字段不是一回事。
- [x] 3.2 两端契约层移除库存校验与数量入参。
  - Evidence: 两端 `saveProduct` 去掉 `Number(p.stock)` 校验、`patch` 去掉 `stock`、新建默认值由 `{ sold: 0, status, tags: [], allergens: ['无'], specs: [] }` 收敛为 `{ status, specs: [] }`；body 注释同步。小程序 `utils/api.js` 94 → 91 行，PC `data/api.js` 272 → 269 行。
- [x] 3.3 小程序商户端页面清理。
  - Evidence: `admin-products.js` 去掉 `low` 判定与 `stock/sold` 行字段；`.wxml` 去掉库存位与月售位；`.wxss` 去掉对应样式；`admin-product-edit.js` 去掉 `stock` 表单字段与数字校验，`.wxml` 去掉整个库存输入行。
- [x] 3.4 PC 后台页面清理。
  - Evidence: `products.js` 319 → 311 行，去掉库存列、销量列、编辑表单库存输入、批量改价的 `stock` 透传，副标题改为「上下架、售罄与价格」；`dashboard.js` 129 → 125 行，待办去掉库存告急与待取超时。
- [x] 3.5 两端状态色阶清理。
  - Evidence: `utils/util.js` 与 `data/api.js` 的状态语义映射去掉 `待取超时: 'warn'` 与 `库存告急: 'warn'`。
- [x] 3.6 两端门禁至 Green。
  - Green: 小程序 `node --test tests/catalog-fields-ui1.test.js` → 5/5；PC `CATALOG_FIELDS_GATE=PASS`。

## 4. Refactor and writer gate

- [x] 4.1 修正一处自己写的正则误伤。
  - Refactor: 小程序用例的 `assert.doesNotMatch(wxml, /…|item\.sold|…/)` 会命中 `{{item.soldoutLabel}}`——那是必须保留的售罄控件，属假阳性。正则收紧为 `item\.sold\b`，并**新增** `assert.match(wxml, /toggleSoldout/)` 断言售罄控件仍在。修正的是判定粒度，不是放宽意图。
- [x] 4.2 一并删除「待取超时」待办，并显式记录越界。
  - Refactor: PC 工作台待办原有待制作、待取超时、库存告急三项。本 change 范围只覆盖库存告急（D2），待取超时属评审 §21（D6）。仍一并删除，因为 0818 PRD §6.12 规定待办只有待制作数一项，只删其一会让该单元既不符合旧规则也不符合新规则。这是本 change 唯一一处越出字面范围的改动，已在 `design.md` 与本条显式记录。
- [x] 4.3 UI1：小程序 Node harness。
  - Refactor: 新用例挂入 `npm test` 后 `tests 34 / pass 34 / fail 0`。既有 29 条无回归。全部 JS `node --check` 通过。
- [x] 4.4 UI1：PC 后台浏览器实际运行。
  - UI1 主场景：本地静态服务器提供候选树，浏览器打开 `web-admin/index.html`。菜品管理表格列为**菜品 / 分类 / 售价 / 状态 / 操作**——库存列与销量列已消失；售罄、上下架、编辑三个控件均在；副标题为「上下架、售罄与价格」。
  - UI1 表单：点开编辑抽屉，字段为**菜品图片 / 菜品名称 / 售价 / 分类 / 描述**，`#f-stock` 不存在。
  - UI1 工作台：待办只剩 `4单待制作`；销量排行正常渲染（招牌红烧牛腩 320、商务双拼饭 286…）；四张 KPI 卡完整。
  - UI1 运行态：页内枚举 `window.Seed.MENU` 对四类字段的泄漏为空数组；`window.Seed.RANK` 仍在。
  - UI1 链路完好性：点击「标记售罄」，`__store.menu[0].status` 由 `on` 变为 `soldout`，确认删字段未打断售罄链路。
  - 控制台：`runtimeErrors` 为空数组。
- [x] 4.5 owned-path 审计与 `git diff --check`。
  - Refactor: `changed=19 outside=0`，`OWNED=PASS`；`git diff --cached --stat HEAD -- services openspec/specs docs` 为空；`DIFF_CHECK=PASS`。全端对 `stock` / `allergens` / `月售` / `过敏原` / `库存告急` / `待取超时` 的扫描零命中。
- [x] 4.6 记录门禁证据与 candidate SHA。
  - Writer verdict: `{ gate_type: W2, ui_level_target: UI1, ui_level_actual: UI1, base_sha: 8071853cda0a2bd0dd3af97cfec19e37b6aee4ff, candidate_sha: external-post-commit（见 5.1）, hard_blockers: 0, unverified_boundary: 小程序 UI1 来自 Node harness、PC UI1 为人工浏览器操作，均未覆盖微信开发者工具与真机；openspec CLI 缺失 }`。

## 5. Independent verification

- [ ] 5.1 在干净 detached worktree 对精确 candidate SHA 只读验证。
- [ ] 5.2 记录 PASS/FAIL 与剩余外部边界。

## 6. 集成与后续

- 两份 delta MUST 与本 change 一并 archive 后应用到生效 spec。
- 删除类三件套至此完成。后续为新增类：按取餐日期的售罄开关、全局折扣率、六态状态机与即时单删除、取餐时间选择、手机号绑定与双要素、订阅消息、支付对账兜底、PC 后台扫码登录。
- 仍未处理：`feat/member-coupon` 分支废弃；`apps/web-admin` 的可提交 runner；PC 销量排行改接真实订单数据（PRD §6.12）。
