## ADDED Requirements

### Requirement: Browsing never asks for identity

启动小程序 MUST 直接进入可浏览状态，MUST NOT 弹出任何手机号或身份授权（§14）。

从启动到提交订单之前的任何路径 —— 首页、菜单、菜品详情、购物车 —— MUST NOT 索取手机号、昵称或头像。身份校验 MUST 只发生在首次提交订单前（§5.6）。

小程序 MUST NOT 保留不调用微信接口的假授权控件。一个只切换页面状态就放行的「授权」弹窗既不收集身份，又让使用者以为该阶段需要授权，两头都是错的。

#### Scenario: A first-time user opens the mini program

- **WHEN** 用户首次打开小程序
- **THEN** 直接进入用户端首页
- **AND** 全程没有任何身份或手机号授权弹窗

#### Scenario: The browsing path is audited

- **WHEN** 检查启动页与浏览路径上的全部页面
- **THEN** 不存在授权弹窗的状态、处理器或模板
- **AND** 不存在只切换状态即放行的假授权控件

### Requirement: The identity screen is not the entry and is not offered to everyone

`app.json` 的入口页 MUST 是用户端首页，MUST NOT 是身份选择页。§4.4 规定未绑定商户手机号的用户直接进用户端首页；在绑定判定由服务端给出之前，MUST 走该默认分支。

身份选择页 MUST 继续存在（§3.5 用户端 9 屏之一），仅经个人中心的「切换身份」进入（§5.9）。

浏览路径上的页面 MUST NOT 提供直达身份选择页的入口 —— 普通用户不该看到该页。

身份选择页的返回控件 MUST 指向可达目标，MUST NOT 调用已随 §0.2 删除的能力。

#### Scenario: The entry route is audited

- **WHEN** 检查 `app.json` 的页面顺序
- **THEN** 首个页面是用户端首页
- **AND** 身份选择页仍在页面清单中

#### Scenario: A user browses the home screen

- **WHEN** 用户停留在首页
- **THEN** 页面上没有通往身份选择页的控件

#### Scenario: A merchant returns to the identity screen

- **WHEN** 已进入身份选择页的使用者点击返回
- **THEN** 返回到上一页或首页
- **AND** 不触发任何已删除能力
