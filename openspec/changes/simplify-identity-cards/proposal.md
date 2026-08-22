## Why

身份选择页的商户端卡片在真机上塌了：卡片比用户端窄一圈、居中而非撑满，「商户端」三个字逐字竖排，描述文字每行两三个字。

原因是上一个 change 把 `.id-card` 直接套在了微信的 `<button>` 上。`<button>` 自带一整套默认样式，其中 `margin-left: auto; margin-right: auto` 在 flex 列容器里会**覆盖 `align-self: stretch`**，使元素收缩到内容宽度并居中；再叠加它默认的左右内边距与字号，卡片的横向空间被挤没，`.id-name` 与 `.id-desc` 只能逐字折行。

在同一个元素上既要承载布局、又要压住微信 `<button>` 的默认值，是一场必输的样式竞赛：默认值有十来条，压住这次不代表压住下次基础库更新。

同时项目方要求：两张卡片只保留「用户端」「商户端」，去掉描述文字。

## What Changes

- **布局与按钮分离**：`<button>` 退化为零样式的透明外壳，卡片本身回到 `<view class="id-card">`。按钮不再承担任何布局职责。
- **去掉两张卡片的描述文字**及其 `.id-desc` 样式（无残留死样式）。
- **门禁新增结构规则**：授权按钮 MUST NOT 同时承载卡片布局类 —— 锁住成因，而不只是锁住这次的表现。

## Capabilities

### Modified Capabilities

- `miniprogram-scope-conformance`：新增一条 requirement —— 身份选择页两张卡片只标身份，且微信授权控件不得承载布局。

## Impact

- Owner：branch `worktree-id-cards`，worktree `/Users/marcusz/Projects/Test Tool/order/.claude/worktrees/id-cards`。
- Owned paths：`apps/wechat-miniprogram/pages/launch/**`、`openspec/changes/simplify-identity-cards/**`、`openspec/specs/miniprogram-scope-conformance/spec.md`。
- Non-goals：
  - 不改授权行为本身（触发、拒绝、允许三条路径与提示文案上一个 change 已定）。
  - 不改用户端入口的跳转目标。
  - 不改页面其余部分（品牌区、主视觉、版本行）。
- Gate：`gate_type=W2`；`ui_level_target=UI1`；`ui_level_actual=UI1`。
- 最小成功标准：`check_identity_cards.js` 七项全过；base_sha 树上四项红；小程序既有测试不回归；`lint_wx.py` 通过；归档门禁失败集合与 base 逐行一致。
