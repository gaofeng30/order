## Why

`pages/profile/profile.wxml` 在微信开发者工具里**根本编译不过**：

```
[ WXML file compile error] ./pages/profile/profile.wxml
expect end-tag `scroll-view`., near `view`
> 50 |     </view>
```

第 46 行有一个多余的 `</view>`。它是删块留下的残骸：`8ea91e6` 删掉「我的优惠券」行、`8b6118c` 删掉「设置」行之后，`<view class="card group">` 空了，而它的闭合标签留在了原地。

**这已经是同一类问题的第二次。** 上一次是 `confirm.wxss` 的 `unexpected token ';'`，同样是删块留下的孤立片段。当时的结论是「按完整规则整块删除，不做逐行正则」，那是操作纪律；纪律挡不住第三次，门禁才能。

`tests/lint_wx.py` 存在的理由正是「抓 Node harness 看不见的编译期错误」，它已经在查 WXSS 花括号配平和孤立 `wx:elif`，也已经维护着一个 WXML 层级栈用于 `wx:elif` 的同级判定 —— **但那个栈从不校验闭合标签的名字**，弹出即算数。于是标签交叉、多余闭合、未闭合三种情况全部漏过。所以整轮改造里 `WX_LINT=PASS` 一路绿灯，而这个页面从头到尾打不开。

## What Changes

- **修复 `pages/profile/profile.wxml`**：删掉第 46 行多余的 `</view>`，使「切换身份入口」与底部占位回到 `.pad body` 内。
- **扩展 `tests/lint_wx.py`**：复用现有的 WXML 层级栈，补上标签名匹配，报出三类结构错误 —— 孤立闭合标签、闭合与开启不匹配、文件结束仍有未闭合标签。
- 复用现有 `TAG` 正则（它已正确处理 `wx:if="{{a > 0}}"` 这类属性值内的 `>`），不新写解析器。

## Capabilities

### Modified Capabilities

- `miniprogram-scope-conformance`：新增一条 requirement —— WXML 的标签结构完整性由静态门禁保证，而非依赖开发者工具在人工打开某个页面时才报错。

## Impact

- Owner：branch `worktree-wxml`，worktree `/Users/marcusz/Projects/Test Tool/order/.claude/worktrees/wxml`。
- Owned paths：`apps/wechat-miniprogram/pages/profile/profile.wxml`、`apps/wechat-miniprogram/tests/lint_wx.py`、`openspec/changes/enforce-wxml-tag-balance/**`、`openspec/specs/miniprogram-scope-conformance/spec.md`。
- Non-goals：
  - **不改 profile 页的内容与范围**。该页的「外卖配送 · 暂未开放」一行与 §5.9「不含：我的优惠券、会员中心、外卖配送」冲突，是一个独立的范围问题，另开 change 处理；本 change 只让页面编译得过。
  - 不做 WXML 属性、表达式或组件引用的校验；只做标签配对。
  - 不改 WXSS 检查、不改 `wx:elif` 检查。
- Gate：`gate_type=W2`；`ui_level_target=UI1`；`ui_level_actual=UI1`。
- 最小成功标准：`check_wxml_balance.py` 十项全过；base_sha 树上四项红；小程序既有测试不回归；归档门禁失败集合与 base 逐行一致。
