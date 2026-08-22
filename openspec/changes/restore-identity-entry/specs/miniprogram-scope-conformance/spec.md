## MODIFIED Requirements

### Requirement: The identity screen is not the entry and is not offered to everyone

`app.json` 的入口页 MUST 是身份选择页（项目方 2026-08-22 决策，回写入 §4.4）。普通用户 MUST 能从该页的用户端入口直接进入浏览，全程 MUST NOT 被索取任何身份。

在服务端给出商户绑定判定之前，身份选择页 MUST 对所有人可见；判定就位后 MUST 按 §4.4 只对已绑定商户展示。

身份选择页 MUST 继续存在于 §3.5 的用户端 9 屏之中，并 MUST 保留经个人中心「切换身份」进入的路径（§5.9）。

身份选择页的返回控件 MUST 指向可达目标，MUST NOT 调用已随 §0.2 删除的能力。

#### Scenario: The entry route is audited

- **WHEN** 检查 `app.json` 的页面顺序
- **THEN** 首个页面是身份选择页
- **AND** 用户端首页仍在页面清单中

#### Scenario: An ordinary user opens the mini program

- **WHEN** 用户打开小程序并点击用户端入口
- **THEN** 直接进入首页
- **AND** 全程没有任何身份或手机号授权弹窗

#### Scenario: A merchant returns to the identity screen

- **WHEN** 已进入身份选择页的使用者点击返回
- **THEN** 返回到上一页或首页
- **AND** 不触发任何已删除能力

## ADDED Requirements

### Requirement: The merchant entry triggers a real WeChat authorisation and claims no verification

身份选择页的商户端入口 MUST 通过微信的 `open-type="getPhoneNumber"` 控件触发授权，MUST NOT 使用自绘弹层代替 —— 一个只切换页面状态就放行的「授权」既不收集身份，又让使用者以为该阶段已完成校验。

用户拒绝授权 MUST 留在身份选择页并说明商户端为何需要验证，MUST NOT 渲染为错误或阻断路径：拒绝是合法选择。

授权成功后，在服务端比对商户账号名单的能力就位之前，界面 MUST 明确声明**身份校验尚未发生**。MUST NOT 出现「验证通过」「身份已确认」一类表述 —— 前端拿到的是加密数据，明文手机号只能由服务端换取（§4.4）。

前端 MUST NOT 以任何形式充当商户端四屏的访问控制。§4.4 末条要求鉴权由服务端执行，客户端隐藏入口不能代替。

#### Scenario: A merchant taps the merchant entry

- **WHEN** 使用者点击商户端入口
- **THEN** 弹出的是微信自身的手机号授权面板
- **AND** 页面上不存在自绘的授权弹层

#### Scenario: The user declines

- **WHEN** 使用者在授权面板上选择拒绝
- **THEN** 停留在身份选择页并得到说明
- **AND** 不出现错误提示或阻断

#### Scenario: The user allows

- **WHEN** 使用者授权成功
- **THEN** 进入商户端
- **AND** 界面明确声明身份校验待服务端接入
