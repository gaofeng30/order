## Red

- [x] 1. 写 `checks/check_merchant_search.js`，覆盖三条 requirement 的十二项断言。
  - 断言全部指向能力事实：搜索框断「slice 内存在 `<input>` 且 `bindinput="onKw"` 且 `value="{{kw}}"`」而非断「搜索」二字；营业日限定断「`utils/util.js` 导出 `findByCode` 且两个页面文件内不出现 `.code ===` / `.code.toUpperCase()`」而非断注释文字。
- [x] 2. 在 base_sha `4664f04` 树上运行门禁。
  - `node checks/check_merchant_search.js <repo-root>` → `MERCHANT_SEARCH_GATE=FAIL (11/12)`。
  - 唯一绿项为「all javascript parses」，即非回归守卫。十一项红的失败信息均为能力缺失（`the search area still has no input element`、`the shared pickup-code resolver does not exist`、`no pickup code repeats across two business days — the business-day rule is unfalsifiable` 等），无一为「找不到某字符串」。
- [x] 3. 记录不回归基线。
  - `node --test tests/*.test.js` → 65 pass / 0 fail。
  - 归档门禁全量 29 项 → 14 项 FAIL，落盘为 base 对照集。

## Green

- [x] 4. `utils/data.js`：导出 `BUSINESS_DAY = '2026-08-21'`（与 PC `data/api.js` 同值）；`ADMIN_ORDERS` 九条补 `pickupDate`；补 `a9`（2026-08-20、取餐号 `0118`，与当日 `a5` 重号）、`a10`（2026-08-20、取餐号 `0203`，仅存在于旧营业日）、`a11`（`退款中`）。
  - `a9` / `a10` 缺一不可：前者证明「同号不同日」，后者证明「跨日提示」。只有前者时跨日提示无样本，只有后者时营业日限定不可证伪。
- [x] 5. 两份种子的 `code` 统一 4 位数字，正则一次改 12 处，去掉 `A` / `B` 前缀（§7.8：即时单已删除，不再有前缀区分）。
- [x] 6. `utils/util.js`：新增 `findByCode` / `codeHint` / `searchOrders`。`searchOrders` 的 4 位分支复用 `findByCode`，营业日规则因此只有一份实现。
- [x] 7. `admin-orders.wxml`：静态 `<text class="search-ph">` 换成受控 `<input>`；新增 `code-hint` 提示行；`admin-orders.wxss` 补 `.search-in` 与 `.code-hint`。
- [x] 8. `admin-orders.js`：`kw` / `hint` 局部状态、`onKw`、`build()` 搜索分支、`switchLane` 清空搜索态。
- [x] 9. `admin-verify.js`：删除自有的 `findOrder`，改调 `findByCode`；未命中走 `codeHint`；`sims` → `['0118','0112','0090']`；placeholder → `如 0118`。
  - 附带补一条 `退款中` 的拒绝分支。该状态是本 change 引入种子的，不补则会落到「订单尚未备好」这个错误文案上。

## Refactor

- [x] 10. 复核边界。
  - 取餐号解析只有 `findByCode` 一处（门禁第 7 项按结构断言）。
  - `kw` 未进 `globalData`，未成为订单模型字段（门禁第 3 项 + UI1 第 8 项）。
  - `LANES` / `NEXT` / `ACT` 未改动，`order-lifecycle-ui1.test.js:142` 的五档断言原样通过。

## 本地验证

- [x] 11. `node checks/check_merchant_search.js <worktree>` → `MERCHANT_SEARCH_GATE=PASS`（12/12）。
- [x] 12. `node --test tests/*.test.js` → 73 pass / 0 fail（既有 65 项零回归 + 新增 8 项）。
- [x] 13. `python3 tests/lint_wx.py .` → `WX_LINT=PASS`。
- [x] 14. 归档门禁全量 29 项与 base 逐行 `diff` 一致（同为 14 项 FAIL，均为既有失败）。
- [x] 15. UI1：`tests/merchant-search-ui1.test.js` 八项，在页面 harness 中驱动真实 `Page` 对象与真实 handler。
  - 该文件单独拷进 base_sha 的干净 detached worktree 运行 → 8 fail / 0 pass；在候选树 → 8 pass / 0 fail。
  - 覆盖：取餐号跨泳道命中、订单号与手机尾号命中、点泳道清空搜索并还原泳道、跨营业日号给出含日期与替代方式的提示、重号只解析当前营业日、手工核销拒绝跨日号并报出其营业日、`退款中` 单经搜索与「全部」泳道可达且泳道集合未变、搜索不泄漏进订单模型。

## 独立验证

- [ ] 16. 提交产生候选 SHA。
- [ ] 17. 在干净 detached worktree 对该精确 SHA 只读重跑 11–15。
