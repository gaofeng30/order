# web-admin-scope-conformance Specification

## Purpose
定义 PC 网页商户后台加载的页面模块、导航分组、内存态、数据契约与文案必须与生效 spec `mvp-product-baseline` 的一期范围保持一致，不得存在已排除能力的实现、入口或表述。
## Requirements
### Requirement: The PC admin loads no module for an excluded capability

`apps/web-admin/index.html` MUST NOT 通过 `<script>` 加载任何实现生效 spec 排除能力的页面模块。会员等级、会员名单、名单导入与优惠券的页面文件 MUST NOT 存在于仓库中。

#### Scenario: Script tags are audited

- **WHEN** 检查 `index.html` 的脚本加载列表
- **THEN** 其中不存在 `pages/levels.js`、`pages/members.js`、`pages/member-import.js`、`pages/coupons.js`
- **AND** 这些文件在仓库中不存在

#### Scenario: Excluded module is reintroduced

- **WHEN** 任一 change 为排除能力新增页面模块或脚本标签
- **THEN** scope 一致性检查失败
- **AND** 该能力必须先取得新的正式范围授权

### Requirement: The PC admin navigation carries no excluded group

侧边导航 MUST NOT 声明「会员与营销」分组或其下属路由。导航元数据 MUST NOT 携带表示二期能力的标记，顶栏副标题 MUST NOT 输出「二期能力」相关文案。

内存态 `window.__store` MUST NOT 初始化 `levels`、`members`、`coupons`、`couponUsed` 任一键。

#### Scenario: Navigation is audited

- **WHEN** 检查路由与外壳模块
- **THEN** 不存在「会员与营销」分组、其下属路由或二期能力标记
- **AND** 内存态初始化不含被排除能力的任一键

### Requirement: The PC admin data layer exposes no excluded capability

种子模块 MUST NOT 导出 `LEVELS`、`MEMBERS`、`COUPONS`、`MY_COUPON_USED`。契约层 MUST NOT 导出会员等级、会员名单、名单导入、优惠券或用户卡包的任何方法。

删除菜品 MUST NOT 读写优惠券数据，也 MUST NOT 在返回值中携带被停用券的数量。

菜品、订单、分类与营业设置的既有契约 MUST 在删除后保持可用。

#### Scenario: Seed and contract exports are loaded and inspected

- **WHEN** 在无 DOM 的运行环境中加载种子与契约模块并枚举导出
- **THEN** 不存在被排除能力的任何种子或方法
- **AND** 菜品、订单、分类与营业设置的契约方法仍然可调用

#### Scenario: A product is deleted

- **WHEN** 调用删除菜品契约
- **THEN** 该菜品从菜品表移除
- **AND** 返回值不含被停用券的数量，过程不读取任何优惠券数据

### Requirement: The PC admin carries no prose describing an excluded capability

`apps/web-admin` 下的脚本、样式与页面文件 MUST NOT 在注释、文案或样式分节标题中出现会员等级、会员名单、优惠券或「二期能力」等已排除能力的表述。

#### Scenario: Prose residue is audited

- **WHEN** 全目录扫描脚本、样式与页面文件的文本内容
- **THEN** 不出现被排除能力的名称或二期能力表述
- **AND** 残留出现时检查指名具体文件与命中词

### Requirement: The PC admin catalog carries no retired field

PC 后台种子与契约层 MUST NOT 保存或接受标签、过敏原、月售与数量库存。菜品管理表格 MUST NOT 提供库存列或销量列，菜品编辑表单 MUST NOT 提供库存输入项。

菜品的售罄与上下架控件 MUST 保留并保持可用。

#### Scenario: Seed and contract are loaded and inspected

- **WHEN** 在无 DOM 的运行环境中加载种子与契约模块
- **THEN** 商品不含 `tags`、`allergens`、`sold`、`stock` 任一字段，契约不接受数量入参
- **AND** 销售状态与保存商品的契约仍然可调用

#### Scenario: Product table and editor are audited

- **WHEN** 检查菜品管理页面
- **THEN** 表格不含库存列与销量列，编辑表单不含库存输入
- **AND** 售罄与上下架控件仍然存在

### Requirement: The PC admin dashboard todo list matches the first-phase scope

工作台待办 MUST 只统计待制作数。库存告急与待取超时 MUST NOT 出现在待办中。

商品销量排行 MUST 保留。

#### Scenario: Dashboard todos are audited

- **WHEN** 检查工作台待办
- **THEN** 不存在库存告急与待取超时条目
- **AND** 销量排行仍然渲染

### Requirement: The PC admin drives orders through the six-state machine only

订单契约的可推进转换 MUST 只有 `制作中 → 待取餐` 与 `待取餐 → 已完成`，MUST NOT 包含 `已预约`。订单泳道 MUST 为 `已预约` / `制作中` / `待取餐` / `已完成` / `已退款` / `全部`，MUST NOT 包含任何已废止状态。

契约层 MUST NOT 导出回退方法，页面 MUST NOT 提供回退动作。

状态语义映射 MUST 只覆盖六态与非订单语义。订单种子 MUST 只使用六态，且 MUST 至少包含一条 `已预约` 订单以填充该泳道。

不可推进订单的行内说明文案 MUST NOT 与状态名拼接出重复的「已」字。

核销 MUST 拒绝已退款订单，并对非 `待取餐` 订单提示尚未备好。

#### Scenario: Contract and lanes are inspected

- **WHEN** 在无 DOM 的运行环境中加载订单契约
- **THEN** 可推进转换只有两条且不含 `已预约`，泳道为六态口径
- **AND** 不存在回退方法

#### Scenario: A reserved order is advanced by the merchant

- **WHEN** 商户尝试推进一条 `已预约` 订单
- **THEN** 请求被拒绝且状态保持 `已预约`

#### Scenario: A terminal order is advanced

- **WHEN** 商户对已处于 `已完成` 的订单再次执行推进
- **THEN** 请求被拒绝且状态不变

### Requirement: The PC admin configures cutoffs per meal period

营业设置 MUST NOT 保留门店级的单一截单时刻或营业起止时间。设置 MUST 按餐段配置截单时刻、取餐起止时间，并 MUST 提供一个全局的取餐时间粒度。

保存 MUST 校验：粒度大于 0、餐段列表非空、每个餐段的截单与取餐起止均已填写、取餐结束时间不早于开始时间。任一校验不通过 MUST 拒绝保存。

#### Scenario: Settings are loaded and inspected

- **WHEN** 在无 DOM 的运行环境中加载营业设置
- **THEN** 不存在门店级单一截单时刻或营业起止时间
- **AND** 每个餐段都带截单时刻与取餐起止，且存在取餐时间粒度

#### Scenario: An invalid period is saved

- **WHEN** 提交取餐结束早于开始的餐段、非正的粒度或空餐段列表
- **THEN** 保存被拒绝并给出具体原因
- **AND** 既有设置不被修改

### Requirement: Every product declares a meal period

商品记录 MUST 保存餐段可售字段，取值 MUST 为 `all`（全天）、`lunch`（午餐）、`dinner`（晚餐）三者之一。

保存商品的契约 MUST 把餐段可售视为必填并校验取值：缺失或非三选一 MUST 拒绝保存。批量调价等复用保存契约的路径 MUST 透传该字段，MUST NOT 在更新时丢失。

餐段的取值集合与展示标签 MUST 由契约层单一提供，页面 MUST 从契约渲染，MUST NOT 各自硬编码。

#### Scenario: Seeded products are inspected

- **WHEN** 在无 DOM 的运行环境中加载商品种子
- **THEN** 每条商品都带合法的餐段可售取值

#### Scenario: A product is saved without a meal period

- **WHEN** 保存商品时未提供餐段可售，或提供了三选一以外的值
- **THEN** 保存被拒绝并提示选择餐段可售
- **AND** 既有商品不被修改

#### Scenario: The product screen is audited

- **WHEN** 检查菜品管理页
- **THEN** 表格含餐段列且从契约的标签映射渲染
- **AND** 编辑表单含必填的餐段可售下拉，选项从契约的取值集合渲染，提交时带上该字段

### Requirement: The PC admin maintains the staff discount whitelist

PC 后台 MUST 提供员工折扣白名单页，并 MUST 从侧边导航可达。

白名单记录 MUST 只有两个可填字段：手机号（唯一识别键）与姓名。记录 MUST 另带四类非填写字段：状态（新增时默认启用，由列表行内开关切换）、加入时间（自动）、微信绑定标记与累计消费、累计单量（只读）。记录 MUST NOT 保存单位、部门、工号或备注。

保存 MUST 校验：手机号非空且为合法手机号格式；姓名非空；手机号在名单内唯一。任一不通过 MUST 拒绝保存且不修改既有记录。

编辑既有记录 MUST 只覆盖手机号与姓名，MUST NOT 重置加入时间、微信绑定标记、累计消费或累计单量。切换状态 MUST 同样不影响这些字段。

停用 MUST 保留记录，删除 MUST 需二次确认并在确认文案中说明停用与删除的区别。

#### Scenario: A whitelist entry is created

- **WHEN** 主账号提交合法的手机号与姓名
- **THEN** 记录被创建且状态为启用、微信绑定为否、累计消费与单量为零
- **AND** 加入时间由系统写入

#### Scenario: An invalid or duplicate entry is submitted

- **WHEN** 提交缺失手机号、格式错误的手机号、缺失姓名，或名单内已存在的手机号
- **THEN** 保存被拒绝并给出具体原因
- **AND** 既有记录不被修改

#### Scenario: An existing entry is edited or toggled

- **WHEN** 编辑既有记录的姓名，或切换其启用状态
- **THEN** 加入时间、微信绑定标记、累计消费与累计单量保持不变

### Requirement: The global discount rate is maintained with the whitelist

全局折扣率 MUST 在员工折扣白名单页维护，取值 MUST 为 1 到 100 的整数百分比，表示员工实付比例，100 表示无折扣。

保存 MUST 校验整数与取值范围，非整数或越界 MUST 拒绝。页面 MUST 说明该折扣对所有命中名单的用户与所有商品统一生效、逐商品先舍入到分再乘数量、且修改只影响新报价不回算历史订单。

#### Scenario: The rate is saved

- **WHEN** 主账号提交 1 到 100 之间的整数
- **THEN** 折扣率被保存并给出确认反馈

#### Scenario: An invalid rate is submitted

- **WHEN** 提交 0、大于 100 的值或小数
- **THEN** 保存被拒绝并给出具体原因

### Requirement: Bulk import follows a preview-then-commit flow

PC 后台 MUST 提供菜品与员工白名单两个批量导入页，且 MUST 从侧边导航可达。

导入 MUST 只接受 `.xlsx`，非该扩展名 MUST 拒绝。文件 MUST 按表头名匹配列，MUST NOT 依赖列顺序；缺少必填列 MUST 中止整份文件；未知列 MUST 忽略并在预览中列出被忽略的列名，MUST NOT 因此判定整份文件异常；超出单次行数上限 MUST 中止并提示分批导入。

流程 MUST 为预览与提交两次独立动作。预览 MUST 返回新增数、更新数、异常列表与一次性令牌，异常项 MUST 带 1 起算的表内行号与具体原因；**预览 MUST NOT 写入任何数据**。存在异常行时 MUST 允许跳过异常行继续提交。

提交 MUST 幂等：同一令牌重复提交 MUST 只生效一次并标记为重复。

页面 MUST 只调用契约方法，MUST NOT 自行解析文件。

#### Scenario: A non-xlsx file is chosen

- **WHEN** 选择的文件扩展名不是 `.xlsx`
- **THEN** 拒绝并提示只接受 `.xlsx`

#### Scenario: The header is inspected

- **WHEN** 表头缺少必填列
- **THEN** 整份文件中止并指出缺少哪些列
- **AND** 表头含未知列时忽略该列并在预览中列出其列名

#### Scenario: A file is previewed then committed

- **WHEN** 预览返回计数与令牌
- **THEN** 预览阶段数据未被写入
- **AND** 用令牌提交后数据生效，再次用同一令牌提交时标记为重复且不重复写入

### Requirement: Product import only adds and never overwrites

菜品导入模板 MUST 为菜品名称、售价、分类、餐段可售四个必填列加选填的描述，**MUST NOT 包含图片列**。导入页 MUST 明确提示图片不在模板中、导入的菜品先无图上架。

导入 MUST 只新增：名称已存在的行 MUST 标记为异常并跳过，MUST NOT 覆盖既有商品的任何字段。同一文件内重名 MUST 判定为异常。

售价 MUST 为大于 0 的数值，餐段可售 MUST 为全天 / 午餐 / 晚餐之一，否则该行 MUST 为异常。

分类不存在时 MUST 自动新建，排序追加末尾且默认对用户端可见；同名新分类在一次导入中 MUST 只新建一次；预览 MUST 单独列出本次将新建的分类。导入的商品 MUST 默认可售且 MUST NOT 伪造图片。

#### Scenario: A row names an existing product

- **WHEN** 导入文件中某行的菜品名称已存在于菜品表
- **THEN** 该行标记为异常并跳过
- **AND** 既有商品的售价与描述保持不变

#### Scenario: A file introduces a new category twice

- **WHEN** 同一文件的多行使用同一个尚不存在的分类
- **THEN** 预览列出该分类为将新建
- **AND** 提交后该分类只被创建一次

### Requirement: Staff import overwrites by phone without resetting system fields

员工导入模板 MUST 只有姓名与手机号两列。手机号 MUST 为唯一识别键：已存在则覆盖更新，不存在则新增。

覆盖更新 MUST 只写入姓名与手机号，MUST 保留状态、加入时间、微信绑定关系、累计消费与累计单量；**导入 MUST NOT 把已停用的记录重新启用**。

同一文件内手机号重复 MUST 判定为异常，MUST NOT 静默取最后一条。姓名或手机号缺失、手机号格式错误 MUST 判定为异常。

#### Scenario: A disabled record is re-imported

- **WHEN** 导入文件包含一条已停用员工的手机号
- **THEN** 该记录的姓名被更新
- **AND** 其状态仍为停用，加入时间与累计统计保持不变

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
