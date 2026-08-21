## Red

- [x] 1. 写 `checks/check_daily_soldout.js`，十六项（十二项新增 + 四项接管）。
- [x] 2. base_sha `1bc66c2` 上运行 → `DAILY_SOLDOUT_GATE=FAIL (11/16)`。
  - 失败信息均为能力缺失：`mini program p003.status = soldout`、`has no sell-out record set`、`has no isSoldOut derivation`、`the sell-out toggle changed the shelf state` 等。
- [x] 3. 记录基线：小程序 87 pass / 0 fail；归档门禁 31 项，14 项 FAIL。

## Green

- [x] 4. 两端种子：`p003` 的 `status` 由 `'soldout'` 改回 `'on'`；新增 `PRODUCT_SOLD_OUT_DATES`，含今日一条（`p003`）与昨日一条（`p006`）。
  - 昨日那条不可省：没有它，「次日自然清零」与「这个商品从来没被标过」在数据上不可区分。
- [x] 5. 两端派生：`isSoldOut(productId, serviceDate)` 与 `isSellable(product, serviceDate)`。可售 = 上架且该取餐日期无记录，两个维度分别判断。
- [x] 6. 两端写入：小程序 `api.setSoldOut(id, serviceDate, bool)`；PC `Api.setSoldOut(id, bool, serviceDate?)` 与 `Api.setShelf(id, 'on'|'off')`。`setShelf` 拒绝第三个取值。
- [x] 7. 两端菜品页：售罄开关只写当前营业日的记录，标签由两个维度现算；PC 批量按钮分流为「批量置售罄写记录」与「批量上下架改 status」。
- [x] 8. PC §7.3 补建校验：`blockingReason` 改按该笔的 `pickupDate` 判断售罄，不再读商品的全局状态。

## Refactor

- [x] 9. 全仓复核：无任何 `status === 'soldout'` 读取点；商品对象上无售罄字段；售罄记录只有 `productId` 与 `serviceDate` 两个键。

## 命名冲突（发现并修正）

- [x] 10. 首版把列表行的售罄布尔命名为 `sold`，与 §0.2 已废止的**月售数量** `sold` 撞名，被 `catalog-fields-ui1.test.js` 的废止字段清单拦下。改名 `soldOut`。
  - 同一文件另有 `/\bsold\b/` 断言命中了 `api.js` 注释里的 REST 路径 `sold-out`。该断言防的是月售字段，`sold-out` 是不同的词，属误报，已收紧为按字段形态匹配（`sold:` / `.sold` / `月售`），而不是放宽拦截范围。

## 本地验证

- [x] 11. `DAILY_SOLDOUT_GATE=PASS`（16/16）。
- [x] 12. `node --test tests/*.test.js` → 94 pass / 0 fail（既有 87 项不回归 + 新增 7 项）。
  - `merchant-scope-ui1.test.js` 的「售罄开关端到端」原断言 `status` 会变化，已按新事实改为断记录集变化且 `status` 不变、次日不受影响。
- [x] 13. `python3 tests/lint_wx.py .` → `WX_LINT=PASS`。
- [x] 14. 归档门禁与 base 逐行 diff：**仅 `strip-retired-catalog-fields/check_catalog_fields.js` 由 PASS 转 FAIL**，即 proposal 记录的接管，其余 30 项一致。
- [x] 15. UI1：`tests/daily-soldout-ui1.test.js` 七项，驱动真实 `Page` 对象。
  - 覆盖：商品只带上下架、今日售罄不及次日、昨日售罄自动清零、下架商品不因无售罄记录而可售、开关只写当日记录且切回时删除记录、行标签由两维度现算、记录集只存存在性。

## 独立验证

- [x] 16. 候选 SHA `9070e9c`（本轮首次一次通过，未误纳构建产物）。
- [x] 17. 在干净 detached worktree 对 `9070e9c` 只读验证。
  - `DIRTY=0`；`DAILY_SOLDOUT_GATE=PASS`；`node --test tests/*.test.js` → 94 pass / 0 fail；`WX_LINT=PASS`；归档门禁与 base 逐行 diff 仅 `check_catalog_fields.js` 一行由 PASS 转 FAIL，即已记录的接管。
