## Why

§14 验收标准第一条：「用户能免手机号浏览；**启动时不弹手机号授权**；首次提交订单前完成微信手机号授权」。当前实现直接违反这一条，而且**弹反了**。

`app.json` 的 `pages[0]` 是 `pages/launch/launch`，所有人启动即落在身份选择页。该页两张卡片的绑定是：

```xml
<view class="id-card primary" bindtap="openAuth">            ← 用户端
<view class="id-card" data-to="admin-orders" bindtap="go">   ← 商户端
```

- **用户端**卡片弹出「申请获取并使用你的昵称、头像与手机号」，点「允许」才进首页；
- **商户端**卡片直接 `navigateTo` 到 `admin-orders`，**零验证**。

需要手机号的那一侧完全敞开，不需要的那一侧反而拦着。

§4.4 定的启动链路是：静默 `wx.login` 取 openid → 服务端查是否已绑定商户手机号 → **已绑定进身份选择页，未绑定直接进用户端首页**。也就是说普通用户根本不该看到身份选择页。当前把它设成入口页，等于对所有人执行了「已绑定」分支。

另有两处同源问题：

- `launch.js` 的 `toBrand()` 调用 `nav.toBrand()`，但品牌选择页早已随 §0.2 删除、`utils/util.js` 里已无该方法。点击该页左上角返回键会抛 `TypeError`。
- `pages/home/home.wxml` 的 `<navbar ... exit>` 给所有人提供了一键回身份选择页的入口，与「普通用户不该看到身份选择页」相冲突。

## What Changes

- **入口页改为 `pages/home/home`**。后端未就位时无法判定绑定状态，因此走 §4.4 的默认分支「未绑定 → 直接进用户端首页」。这是唯一有依据的默认值。
- **删除假授权弹窗**：`auth` 状态、`openAuth` / `closeAuth` / `allowAuth`、内联微信图标常量、弹窗模板与 `.auth-*` 样式。该弹窗不调用任何微信接口，`allowAuth()` 只是跳首页 —— 它没有真的在收手机号，但它让演示时的客户以为一期就是「一进来就要手机号」。
- **身份选择页保留但不再是入口**，经个人中心「切换身份」进入（§5.9 已定义该入口）。
- **修 `toBrand` 死处理器**：返回键改用 `nav.back()`。
- **首页导航栏去掉 `exit`**。

## Capabilities

### Modified Capabilities

- `miniprogram-scope-conformance`：新增两条 requirement —— 启动即可免手机号浏览、不得在任何浏览路径上索取身份；身份选择页不是入口页且不向未绑定商户的用户提供入口。

## Impact

- Owner：branch `worktree-browse-first`，worktree `/Users/marcusz/Projects/Test Tool/order/.claude/worktrees/browse-first`。
- Owned paths：`apps/wechat-miniprogram/{app.json,pages/launch/**,pages/home/home.wxml}`、`apps/wechat-miniprogram/tests/browse-first-ui1.test.js`、`openspec/changes/browse-before-identity/**`、`openspec/specs/miniprogram-scope-conformance/spec.md`。
- Non-goals：
  - **不实现真实身份识别链路**。静默 `wx.login`、`getPhoneNumber`、openid 绑定判定全部依赖服务端，§16.5 已把「身份识别链路」与「启动路由判定」列为待补齐。本 change 只移除违反 §14 的拦截，不伪造一个能通过的授权。
  - **不给商户端卡片加身份验证弹窗**。§4.4 的绑定入口在个人中心的「商户登录」，不在身份选择页；在此新增一条授权路径属于 §4.4 之外的新增设计，需先改 PRD。
  - **不实现商户端鉴权**。§4.4 末条明写「客户端菜单隐藏不能代替鉴权」，四个 `admin-*` 页的数据接口必须由服务端各自校验角色。本 change 不碰这件事，也不制造已经解决的错觉。
  - 不改个人中心的「切换身份」入口可见性（§5.9 要求仅对已绑定商户展示，同样依赖后端）。
- Gate：`gate_type=W2`；`ui_level_target=UI1`；`ui_level_actual=UI1`。
- 最小成功标准：`check_browse_first.js` 九项全过；base_sha 树上六项红；小程序既有测试不回归；`lint_wx.py` 通过；归档门禁失败集合与 base 逐行一致。
