## ADDED Requirements

### Requirement: The home screen shows the store notice and business status from configuration

首页 MUST 展示门店公告，内容 MUST 取自营业设置的公告字段，MUST NOT 在页面中写死。

首页 MUST 展示当前营业状态，取值为 `营业中` / `休息中` / `已截单`，MUST 跟随商户端的切换。营业状态 MUST NOT 由截单时刻派生 —— §6.9 允许主账号手动覆盖营业时间规则，派生值只是默认值而非事实。

首页 MUST NOT 保留任何「未开放」「即将上线」一类的占位角标或其渲染分支（§0.2）。

#### Scenario: The merchant edits the notice

- **WHEN** 营业设置里的公告内容变化
- **THEN** 首页展示新的公告
- **AND** 首页脚本与模板中不存在写死的公告文本

#### Scenario: The merchant pauses service

- **WHEN** 商户在小程序商户端把门店切为 `休息中`
- **THEN** 用户端首页展示 `休息中`
- **AND** 该展示不依赖截单时刻的推导

### Requirement: The home screen keeps a standing banner for orders in flight

存在 `已预约` / `制作中` / `待取餐` 订单时，首页顶部 MUST 常驻一条提示，展示进行中单数与**最近一单的取餐时刻**；无此类订单时 MUST NOT 渲染该提示。

「最近一单」MUST 按取餐时刻排序确定，MUST NOT 按下单时间 —— 用户关心的是下一顿何时能取。

任一订单进入 `待取餐` 时，提示 MUST 改为「已备好，可取餐」并高亮，MUST NOT 在该文案中继续强调单数而稀释行动号召。

点击提示 MUST 直达对应订单的取餐码页。

该提示是 §5.10 订阅消息被拒时的兜底：餐饮类目只能一次性订阅，拒绝授权的用户 MUST 仍能从首页得知餐已备好。

#### Scenario: A user has orders in flight

- **WHEN** 用户存在 `已预约` 或 `制作中` 订单且无 `待取餐`
- **THEN** 首页顶部展示进行中单数与最近一单的取餐时刻
- **AND** 点击进入该订单

#### Scenario: A meal is ready

- **WHEN** 用户的任一订单进入 `待取餐`
- **THEN** 提示文案变为「已备好，可取餐」并高亮
- **AND** 点击进入该订单的取餐码页

#### Scenario: A user has no order in flight

- **WHEN** 用户没有 `已预约` / `制作中` / `待取餐` 订单
- **THEN** 首页不渲染该提示
