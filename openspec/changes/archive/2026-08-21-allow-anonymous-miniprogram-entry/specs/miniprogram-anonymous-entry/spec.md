## ADDED Requirements

### Requirement: User entry opens anonymous browsing directly

启动页的“用户端”入口 MUST 直接进入现有用户端首页。该操作 MUST NOT 先展示昵称、头像、手机号或其他资料授权弹层，MUST NOT 调用交互式资料/手机号授权能力，且 MUST NOT 以用户同意授权作为浏览首页的前置条件。

本 requirement 不禁止其他 change 在后台执行不打扰用户的静默会话；手机号授权仍只属于后续首次提交订单前的独立能力，不得在本 change 中提前实现、模拟或占位。

#### Scenario: Anonymous user enters the home page

- **WHEN** 未提供手机号、昵称或头像的用户在启动页点击“用户端”
- **THEN** 小程序直接进入现有用户端首页
- **AND** 过程中不出现资料或手机号授权弹层，不要求用户点击允许或拒绝

#### Scenario: User entry is repeated after a cold compile

- **WHEN** 微信开发者工具重新编译 exact candidate 并再次从启动页点击“用户端”
- **THEN** 用户仍直接进入首页，且启动页没有保存上一次运行的授权状态
- **AND** 页面与日志不展示手机号、login code、token 或其他敏感身份值

### Requirement: Launch page retains only its valid entry responsibilities

启动页 MUST 继续展示品牌内容、用户端入口与既有商户端入口，但 MUST 删除只为错误授权弹层服务的状态、事件、图标、模板与样式。删除弹层 MUST NOT 改变现有页面路由表、用户端首页内容、商户端页面或公共导航实现。

#### Scenario: Launch page adjacent content remains available

- **WHEN** exact candidate 在微信开发者工具普通编译后展示启动页
- **THEN** 品牌内容、用户端入口与商户端入口均正常显示，且调试器没有 candidate 编译或运行错误
- **AND** 启动页中不存在“微信授权登录”“申请获取并使用你的”“昵称、头像与手机号”或对应遮罩/允许/拒绝控件

#### Scenario: Removed authorization code cannot be reactivated

- **WHEN** focused test 检查最终启动页 JS、WXML 与 WXSS
- **THEN** 启动页不存在 `auth`、`openAuth`、`closeAuth`、`allowAuth`、授权弹层模板或其专用样式
- **AND** “用户端”入口只通过既有导航路径进入 `home`，不新增兼容分支、存储或替代提示
