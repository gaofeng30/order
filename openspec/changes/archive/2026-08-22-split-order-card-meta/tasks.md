## Red

- [x] 1. 写 `checks/check_card_meta.js`，七项。
- [x] 2. base_sha `7865793` 上运行 → `CARD_META_GATE=FAIL (5/7)`。
  - 其中一项直接打出拼接现场：`o1 time text still carries the pickup place: 预约 今天 17:00 · 党政办公中心后院老食堂北门`。
  - 另一项指出根因不止于拼接：`.oc-time centres its icon; a wrapped line would leave it aligned to neither row` —— 图标垂直居中于整块，一旦文字折行就与两行都不对齐，正是截图里的现象。
- [x] 3. 记录基线：小程序 102 pass / 0 fail；归档门禁 37 项，14 项 FAIL。

## Green

- [x] 4. `orders.js` 拆为 `timeText`（预约时间）与 `placeText`（取餐地点）两个字段。
  - 不做「太长就截断」或「窄屏才换行」：前者让地址显示不全（用户到不了现场），后者让排版正确与否取决于客户填了多长的地址。两行是无条件的。
- [x] 5. 模板渲染为两行，各带自己的图标（日历 / 定位）。原 `typeIcon` 字段随之删除，不留没人用的字段。
- [x] 6. `.oc-time` / `.oc-place` 的 `align-items` 由 `center` 改为 `flex-start`，图标加 4rpx 上边距对齐文字。
  - 这条不只修当前这一处：取餐点是客户可配值，将来换更长的地址仍会折行，`flex-start` 让那时的排版依然成立。

## 门禁自身的问题（发现并修正）

- [x] 7. 「信息行对齐首行」一项原用 `wxss.indexOf('.oc-time ')` 定位规则，抓到的却是 `.oc-time icon` 这条后代选择器，于是在正确实现上误报。改为按「选择器 { 声明 }」切分并只匹配独立选择器。
  - 同一个毛病本轮反复出现：**用字符串查找去理解结构化文本**。WXSS 有真正的规则边界，就该按边界解析。

## 本地验证

- [x] 8. `CARD_META_GATE=PASS`（7/7）。
- [x] 9. `node --test tests/*.test.js` → 102 pass / 0 fail。
  - `order-model-ui1.test.js` 有一条断言钉住了旧的拼接格式（`/^预约 … · /`），已按新事实改为分别断言两个字段。
- [x] 10. `python3 tests/lint_wx.py .` → `WX_LINT=PASS`。
- [x] 11. 归档门禁 37 项与 base 逐行一致，**无接管**。
- [x] 12. UI1：驱动真实 `Page` 确认输出为 `timeText: '预约 今天 17:00 取餐'` 与 `placeText: '党政办公中心后院老食堂北门'` 两个独立字段。

## 独立验证

- [x] 13. 候选 SHA `9c597b7`。
- [x] 14. 在干净 detached worktree 对 `9c597b7` 只读验证：`DIRTY=0`；`CARD_META_GATE=PASS`；102 pass / 0 fail；`WX_LINT=PASS`；归档门禁 37 项与 base 逐行一致。
