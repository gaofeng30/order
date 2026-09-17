## ADDED Requirements

### Requirement: Registered WXML templates have matched element structure

小程序结构检查 MUST 对每个已登记 WXML 文件维护非自闭合元素栈。每个关闭标签 MUST 与最近的未关闭标签同名；多余关闭标签、错序关闭标签或文件结束时仍未关闭的标签 MUST 使检查非零失败，并报告文件和首个决定性行号。现有同级 `wx:if/wx:elif/wx:else` 规则 MUST 保持有效。

#### Scenario: Mismatched nested tags fail before delivery
- **WHEN** 受控 WXML 夹具先打开 `scroll-view` 再以 `</view>` 错序关闭
- **THEN** 结构检查以非零退出并定位首个不匹配行
- **AND** 不得返回 `WX_LINT=PASS`

#### Scenario: Valid templates preserve conditional checks
- **WHEN** 仓库全部 WXML 标签正确配对且条件分支有合法同级前驱
- **THEN** 结构检查返回 `WX_LINT=PASS`
- **AND** 原有 `wx:elif/wx:else` 负例语义不被削弱

### Requirement: Personal profile compiles without behavior drift

个人中心 WXML MUST 在微信开发者工具中成功编译并呈现原有身份卡、订单入口、客服入口、切换身份入口和用户 tabbar。修复 MUST NOT 改变文案、事件绑定、样式、页面脚本或业务状态。

#### Scenario: Exact candidate renders personal profile
- **WHEN** 微信开发者工具 Stable 2.02.2608040 使用已分配测试号编译 exact candidate 并进入个人中心
- **THEN** 编译器不再报告标签闭合错误，个人中心关键入口可见
- **AND** 页面不存在本 change 引入的控制台编译错误

#### Scenario: Adjacent launch path remains usable
- **WHEN** 同一 exact candidate 从启动页进入用户端路径
- **THEN** 启动页与用户端导航仍可呈现
- **AND** 本 change 不修改任何非 owned 小程序产品文件
