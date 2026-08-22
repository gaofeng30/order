## Red

- [x] 1. 定位塌陷成因，不靠猜。`<button>` 在微信里自带 `margin-left/right: auto`；在 `.cards { display: flex; flex-direction: column }` 里，auto 外边距会**覆盖 `align-self: stretch`**，使元素收缩到内容宽度并居中。叠加它默认的左右内边距与字号，横向空间被挤没，`.id-name` 与 `.id-desc` 因此逐字折行。
- [x] 2. 写 `checks/check_identity_cards.js`，七项。
- [x] 3. base_sha `bbda134` 上运行 → `IDENTITY_CARDS_GATE=FAIL (4/7)`。
  - 首项直接指出成因：`the layout class sits on the <button>; wechat button defaults will fight it`。

## Green

- [x] 4. `<button>` 退化为透明外壳 `.id-plain`（`display:block; width:100%; margin:0; padding:0; background:transparent; text-align:left; font-size:inherit; color:inherit` + `::after { border: none }`），卡片布局回到内部 `<view class="id-card">`。
  - 两张卡因此共用同一个类、同一份规则 —— 不是「调到看起来一致」，而是没有第二份规则可以漂移。
- [x] 5. 两张卡片删除描述文字，`.id-desc` 规则一并删除。留一条没人引用的规则，下一个人会以为卡片还该有副标题。

## 上一个 change 的教训

- [x] 6. 上一版把 `.id-card` 直接套在 `<button>` 上，门禁与测试全绿而真机塌陷 —— 因为门禁查的是**结构**（有没有 `open-type`、回调通不通），布局正确与否没有任何自动化覆盖。
  - 本 change 的门禁因此断言「授权按钮上不得出现布局类」。它锁的是**成因**而不是这次的表现：宽度在静态检查里测不了，但「布局类是否落在 button 上」是可检的结构事实，且正是根子。
  - 仍需承认：视觉回归目前只能靠人看截图。

## 本地验证

- [x] 7. `IDENTITY_CARDS_GATE=PASS`（7/7）。
- [x] 8. `node --test tests/*.test.js` → 117 pass / 0 fail，无回归。
- [x] 9. `python3 tests/lint_wx.py .` → `WX_LINT=PASS`。
- [x] 10. 归档门禁 40 项与 base 逐行一致，**无接管**（上一个 change 的 `check_identity_entry.js` 仍全过：授权触发与三条回调路径未被本次改动影响）。

## 独立验证

- [ ] 11. 提交产生候选 SHA。
- [ ] 12. 在干净 detached worktree 对该精确 SHA 只读重跑 7–9。
