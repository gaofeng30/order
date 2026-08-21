## ADDED Requirements

### Requirement: The PC admin maintains the merchant account roster

PC 后台 MUST 提供商户账号名单页，并 MUST 从侧边导航可达。该名单决定谁能登录商户端与 PC 后台。

账号记录 MUST 只有三个可填字段：手机号（唯一识别键）、姓名、角色。角色 MUST 只有主账号与子账号两种取值，且取值与显示名 MUST 由契约层导出，页面 MUST NOT 自行硬编码。记录 MUST 另带三类非填写字段：状态（新增时默认启用，由列表行内开关切换）、微信绑定标记（新增时为未绑定，由商户本人在小程序端绑定后写入）与创建时间。

页面 MUST 逐行展示该账号的可用范围：主账号为 PC 后台加小程序商户端全部权限；子账号 MUST 只限小程序的订单、核销、菜品三个页面。

保存 MUST 校验：手机号非空且为合法手机号格式；姓名非空；角色为两种取值之一；手机号在本名单内唯一。任一不通过 MUST 拒绝保存且不修改既有记录。

编辑既有记录 MUST 只覆盖手机号、姓名与角色，MUST NOT 重置微信绑定标记、状态或创建时间。

停用 MUST 保留记录，删除 MUST 需二次确认并在确认文案中说明停用与删除的区别、以及删除会解绑已绑定的微信。

#### Scenario: An account is created

- **WHEN** 提交合法的手机号、姓名与角色
- **THEN** 账号被创建且状态为启用、微信绑定为否
- **AND** 创建时间由系统写入

#### Scenario: An invalid or duplicate account is submitted

- **WHEN** 提交缺失手机号、格式错误的手机号、缺失姓名、非法角色，或本名单内已存在的手机号
- **THEN** 保存被拒绝并给出具体原因
- **AND** 既有记录不被修改

#### Scenario: An existing account is edited

- **WHEN** 编辑既有账号的姓名或角色
- **THEN** 微信绑定标记、状态与创建时间保持不变

### Requirement: The last enabled owner can never be removed, disabled or demoted

契约层 MUST 保证任何时刻至少有一个启用的主账号：当某主账号是唯一启用的主账号时，删除它、停用它、以及把它降级为子账号这三条路径 MUST 全部被拒绝，且拒绝 MUST 发生在契约层而非仅由页面隐藏按钮实现。

拒绝提示 MUST 指出被拒账号、被拒动作与解除办法（先添加并启用另一个主账号）。

名单中只剩一个启用的主账号时，页面 MUST 展示提示，说明该账号不可停用、删除或降级。

存在第二个启用的主账号后，上述三条路径 MUST 全部恢复可用。

#### Scenario: The sole owner is attacked from every direction

- **WHEN** 名单内只有一个启用的主账号，且尝试删除它、停用它或将其角色改为子账号
- **THEN** 三次操作全部被拒绝并给出可解除的原因
- **AND** 该账号的角色与启用状态保持不变

#### Scenario: A second owner unlocks the first

- **WHEN** 添加并启用第二个主账号后再停用第一个
- **THEN** 操作成功

### Requirement: The merchant roster is separate from the discount whitelist

商户账号名单与员工折扣白名单 MUST 是两份互不影响的数据：前者决定登录权限，后者决定顾客结算是否打折。两份名单 MUST NOT 共用同一条记录，同一手机号出现在两份名单中 MUST 各自独立生效。

商户账号名单 MUST NOT 提供批量导入：账号数量少且授予的是登录权限，逐条录入是有意的约束。页面 MUST NOT 提供文件上传入口，导航 MUST NOT 提供对应的导入页。

页面 MUST 用文案说明这两份名单的分工，避免管理员误把折扣名单当作授权名单。

#### Scenario: The two rosters are audited

- **WHEN** 检查两份名单的存储与记录标识
- **THEN** 二者为不同的集合且标识不重叠

#### Scenario: The roster page is audited for import

- **WHEN** 检查商户账号名单页与侧边导航
- **THEN** 页面不装配任何预览-提交导入流程，也不提供文件选择控件
- **AND** 导航中不存在商户账号导入页
