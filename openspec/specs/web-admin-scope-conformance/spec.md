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

### Requirement: Order money is stored in cents and settles exactly

订单金额 MUST 以整数分存储，MUST NOT 出现浮点或以元为单位的中间态。涉及金额的字段为 `subtotal`（原价小计）、`discountCut`（折扣减免）、`total`（实付），以及 `items` 行内的原价单价与折后单价。

每笔订单 MUST 同时满足三条恒等式：`subtotal - discountCut === total`；`items` 逐行「原价单价 × 数量」之和 `=== subtotal`；`items` 逐行「折后单价 × 数量」之和 `=== total`。任一不成立即为数据错误。

折扣 MUST 逐商品先舍入到分再乘数量：折后单价 `=== round(原价单价 × discountRate / 100)`。整单先乘后舍入 MUST NOT 使用 —— 两种口径在多行订单上会差到分。

契约层 MUST 导出唯一的分转元展示函数，页面 MUST 经由它渲染金额，MUST NOT 自行做除以 100 的算术。

#### Scenario: Every order is audited for settlement

- **WHEN** 遍历订单数据
- **THEN** 三条恒等式全部成立，且三个金额字段均为整数
- **AND** 员工折扣单的逐行折后单价等于原价单价乘折扣率后舍入到分

#### Scenario: A page renders an amount

- **WHEN** 检查任何渲染订单金额的页面
- **THEN** 金额经由契约层的分转元函数产出
- **AND** 页面源码中不存在手写的除以 100

### Requirement: Every order carries the payment and refund facts

订单记录 MUST 携带对账所需的支付事实：`paidAt`（支付成功时间，精确到秒）、`txnId`（微信交易号，全局唯一）、`discountRate` 与 `isStaff`（身份与折扣率快照）。

取餐信息 MUST 由 `pickupDate`（营业日期）、`pickupTime`（时间点 `hh:mm`）、`mealPeriod`（`lunch` 或 `dinner`）、`pickupPoint` 四项承载，MUST NOT 保留「距取餐还有几分钟」这类派生量 —— 它随时间变化，存下来必然过期。

处于 `退款中` 或 `已退款` 的订单 MUST 携带退款记录，含退款单号、退款金额、退款状态、操作人、退款时间与退款原因；退款状态 MUST 与订单状态一致。其余状态的订单 MUST NOT 携带退款记录。

退款金额 MUST 恒等于订单实付。一期只支持原路全额退款（§7.7），部分退款 MUST 被拒绝且 MUST NOT 创建退款记录。

口味与备注 MUST 绑定在 `items` 行内，整单级 MUST 只有 `orderNote`，MUST NOT 存在整单级口味字段。

#### Scenario: A refunded order is audited

- **WHEN** 检查状态为 `退款中` 或 `已退款` 的订单
- **THEN** 它带有完整的退款记录且退款状态与订单状态一致
- **AND** 退款金额等于订单实付

#### Scenario: A non-refunded order is audited

- **WHEN** 检查其余四个状态的订单
- **THEN** 它不带退款记录

#### Scenario: The order detail is opened

- **WHEN** 主账号在订单管理页选中一笔订单
- **THEN** 详情展示支付时间、取餐日期与时间点、取餐点与微信交易号
- **AND** 员工折扣单额外展示折扣率与减免金额，且原价小计减减免等于实付

### Requirement: Each of the six order states has its own lane

订单管理页 MUST 为六个状态各设一条泳道，外加一条「全部」。`退款中` MUST 自成一格：它是唯一需要人工盯到到账的状态，混在「全部」里等同于没有入口。

每条泳道的计数 MUST 等于该状态的订单数，「全部」的计数 MUST 等于订单总数。

#### Scenario: The lanes are audited

- **WHEN** 检查订单管理页的泳道集合与计数
- **THEN** 六个状态各有一条泳道，另有一条「全部」
- **AND** 每条泳道的计数与该状态的订单数一致

### Requirement: The PC admin reconciles payments against the wechat bill

PC 后台 MUST 提供财务与对账页，并 MUST 从侧边导航可达。页面 MUST 支持按日期区间筛选，起始日期晚于结束日期 MUST 被拒绝且不改变当前区间。

收款 MUST 按**支付日期**归集，MUST NOT 按营业日期归集：微信商户平台的交易账单以交易时间为准，而预约单可以今天支付、次日取餐，两者按不同日期落账则必然对不上。每笔收款 MUST 同时展示支付时间与营业日期，避免管理员自行推断。

退款 MUST 按**退款到账日期**归集，MUST NOT 跟随原订单的支付日期：跨日退款在微信账单里出现在到账当天。页面 MUST 说明「今天的退款里可能含前几天的订单」。

收款明细 MUST 逐笔展示订单号、支付时间、微信交易号与实付金额（§6.11）。

#### Scenario: A reservation paid the day before is grouped

- **WHEN** 一笔订单支付于某日、取餐营业日期为次日，按支付当日筛选
- **THEN** 该笔出现在支付当日的收款明细中
- **AND** 按营业日期当日筛选时，该笔不出现在收款明细中

#### Scenario: A refund settles on a later day

- **WHEN** 一笔退款的到账日期晚于原订单的支付日期
- **THEN** 该退款出现在到账当日的退款记录中
- **AND** 不出现在原订单支付当日的退款记录中

#### Scenario: An invalid date range is submitted

- **WHEN** 起始日期晚于结束日期
- **THEN** 变更被拒绝并给出原因
- **AND** 页面保持原有区间与数据

### Requirement: The reconciliation summary nets by amount

对账汇总 MUST 给出实收合计、退款合计、净额三项，且 MUST 满足 `净额 === 实收合计 − 退款合计`。三项 MUST 以整数分计算后再格式化，MUST NOT 以元为单位求和。

退款 MUST 按**金额**扣减。一期只有全额退款（§7.7），因此每笔退款金额恒等于原订单实付；但汇总 MUST NOT 因此改为按订单笔数扣减 —— 跨区间的退款（原订单支付于区间外）在区间内只有退款没有收款，按笔数扣无从扣起。

汇总 MUST 标出未到账的退款笔数（状态为 `退款中` 者），它们是净额已扣但钱尚未退出的部分。

金额为负时符号 MUST 在货币符号之前，MUST NOT 出现 `¥-12.00` 这种排版。

#### Scenario: A refund settled outside its payment range is netted

- **WHEN** 区间内含一笔退款，其原订单支付于区间之外
- **THEN** 净额扣减该退款金额，且实收合计不含该订单
- **AND** 汇总的三项满足净额等式

#### Scenario: The net goes negative

- **WHEN** 某日只有退款而无收款
- **THEN** 净额展示为负值且负号在货币符号之前

### Requirement: The refund record is traceable and complete

每条退款记录 MUST 携带退款单号、退款金额、退款状态与操作人（§6.11），并 MUST 携带可回溯到原订单的信息：订单号与微信交易号。缺少回溯信息的退款记录在对账时无法定位原单，MUST NOT 出现。

退款金额 MUST 恒等于原订单实付（§7.7 一期只支持原路全额退款）。契约层 MUST NOT 导出「是否部分退款」这类标记，页面 MUST NOT 渲染它 —— 一个恒为假的分支只会让人以为部分退款是可能的。

操作人 MUST 取自当前登录的商户账号，MUST NOT 硬编。

#### Scenario: A refund record is audited

- **WHEN** 检查任一退款记录
- **THEN** 它带有退款单号、金额、状态与操作人，且金额等于原订单实付
- **AND** 它带有原订单号与微信交易号

### Requirement: The payment detail export survives Excel

导出 MUST 产出可直接用 Excel 打开的明细文件，且 MUST 防住两处会让文件变得不可用的陷阱：

- 文件 MUST 带 UTF-8 BOM。缺少 BOM 时 Excel 会把中文识别为乱码。
- 微信交易号与退款单号是长数字串，MUST 以文本形式导出。直接写入数字 Excel 会转成科学计数法，导致交易号无法与账单比对。

导出 MUST 含表头，且 MUST 每笔收款一行。金额 MUST 以元导出，MUST NOT 直接导出分。

#### Scenario: The export is opened in Excel

- **WHEN** 导出某日期区间的收款明细
- **THEN** 文件以 UTF-8 BOM 开头，含表头且行数等于收款笔数加一
- **AND** 微信交易号以原样长数字串出现，未被转为科学计数法

### Requirement: The page states what it does not reconcile

页面 MUST 明写「自动拉取并比对微信账单一期未实现」，并说明本页数字只汇总本系统内的订单、不代表已与微信核平。缺少该声明时管理员会把汇总数字当作系统核过的结果。

该缺口 MUST 同时记录在 PRD 的前端与后端缺口清单中。

#### Scenario: The page is audited for its own limits

- **WHEN** 检查财务与对账页
- **THEN** 页面写明自动核对未实现，并给出人工核对的口径
- **AND** PRD 的缺口清单中有对应条目

### Requirement: The owner initiates a full refund from the order

订单管理页 MUST 提供发起退款入口。退款 MUST 只能从 `已预约`、`制作中`、`待取餐`、`已完成` 四个状态发起（§7.1 旁路），从 `退款中` 或 `已退款` 发起 MUST 被拒绝且 MUST NOT 改动既有退款记录。

契约方法 MUST NOT 接受调用方指定的退款金额：一期只支持原路全额退款（§7.7），表达不出来的请求不需要校验，也不会被后续改动悄悄放开。

退款原因 MUST 必填，空白或纯空格 MUST 被拒绝。原因与操作人 MUST 一并记入退款记录，供财务对账追责。

发起退款 MUST 只把订单推进到 `退款中`。只有微信确认退款成功才是 `已退款`（§7.7），因此 PC 后台 MUST NOT 自行将订单置为 `已退款`。

重复对同一订单发起退款 MUST 被拒绝，且 MUST NOT 产生第二条退款记录、MUST NOT 改变已生成的退款单号（§7.1 幂等）。

退款 MUST 有二次确认，确认层 MUST 展示将退金额、操作人与不可撤销的后果。订单管理页 MUST NOT 提供任何撤销或回退已完成转换的入口（§7.1）。

#### Scenario: A refund is initiated from a live order

- **WHEN** 主账号对处于四个可退状态之一的订单填写原因并确认退款
- **THEN** 订单进入 `退款中`，退款记录含全额金额、原因、当前登录账号与时间戳
- **AND** 订单不会被直接置为 `已退款`

#### Scenario: A refund is attempted twice

- **WHEN** 对同一订单再次发起退款
- **THEN** 请求被拒绝并说明当前状态
- **AND** 该订单仍然只有一条退款记录，退款单号不变

#### Scenario: A refund is submitted without a reason

- **WHEN** 退款原因为空或只有空格
- **THEN** 请求被拒绝并说明原因必填

### Requirement: Uncollected is a query, not a seventh state

「未取餐」MUST 作为查询口径提供：营业日已结束仍处于 `待取餐` 的订单（§6.7）。它 MUST NOT 进入状态集合，MUST NOT 成为第七条泳道。

该筛选 MUST 只在 `待取餐` 泳道下可用，切换到其他泳道时 MUST 自动解除 —— 它是挂在 `待取餐` 上的条件，带到别处会把列表筛成空的。

营业日 MUST 由契约层单一下发，页面 MUST NOT 各自硬编日期。

#### Scenario: The uncollected filter is applied

- **WHEN** 在 `待取餐` 泳道启用「未取餐」
- **THEN** 列表只剩营业日期早于当前营业日的 `待取餐` 订单
- **AND** 当日的 `待取餐` 订单被排除

#### Scenario: The state set is audited

- **WHEN** 检查泳道集合与状态集合
- **THEN** 二者均不包含「未取餐」

#### Scenario: The lane changes while the filter is on

- **WHEN** 启用「未取餐」后切换到其他泳道，或退款后自动跳转到 `退款中`
- **THEN** 筛选自动解除，目标泳道展示其完整列表

### Requirement: Pending payments are records, not orders

PC 后台 MUST 提供「支付待处理」页，并 MUST 从侧边导航可达。页面承载 §7.3 对账兜底产生的条目：已发起支付、微信查得已支付、但自动补建订单失败的预支付记录。

条目 MUST NOT 是订单：MUST NOT 携带六个订单状态中的任何一个，MUST NOT 携带取餐号。取餐号在订单生成时分配（§7.8），而这些条目正是没能生成订单的那一批；带上状态或取餐号等于把 PRD 已删除的「异常订单」概念重新引入（§7.3 明确不引入 `异常` 状态）。

条目 MUST NOT 出现在订单管理的任何泳道中，MUST NOT 计入泳道计数，「支付待处理」MUST NOT 成为一条泳道。

每条 MUST 携带完整的支付事实：预支付单号、微信交易号、支付时间、支付金额、顾客联系方式、意向取餐信息与商品明细。钱已经收到，缺任一项都会导致既对不上账也退不了款。

未建单原因 MUST 为 §7.3 列举的三种之一：商品已下架、取餐时间已过、数据校验不通过。

页面 MUST 说明条目的来源是后端定时任务、且只有自动补建也失败的才会出现在此，避免主账号把该列表误认为实时的全部支付异常。

#### Scenario: A pending entry is audited

- **WHEN** 检查任一待处理条目
- **THEN** 它带有预支付单号、微信交易号、支付时间、金额、顾客、意向取餐与商品明细
- **AND** 它既没有订单状态也没有取餐号

#### Scenario: The order lanes are audited

- **WHEN** 检查订单列表与泳道计数
- **THEN** 待处理条目不出现在任何泳道中，泳道计数不含它们
- **AND** 泳道集合中没有「支付待处理」

### Requirement: Manual rebuild is refused while the cause still holds

人工建单 MUST 在提交前重新校验阻塞原因，原因仍成立时 MUST 拒绝，并 MUST 在拒绝信息中指出原因与解除办法。原因未解除就建单，等于为一道做不出来的菜分配取餐号，顾客会第二次白跑。

页面 MUST 在打开确认层时就展示当前的阻塞判定，原因仍在时 MUST NOT 提供确认按钮 —— 让主账号在点之前就知道能不能成。

建单成功时 MUST 沿用原支付事实：微信交易号、支付时间与实付金额 MUST 与预支付记录一致。订单 MUST 按 §7.8 分配 4 位取餐号且在同一取餐日期内唯一，MUST 按 §7.4 判定进入 `已预约` 或 `制作中`。

建单成功后条目 MUST 离开待处理列表。重复对同一条目建单 MUST 被拒绝，MUST NOT 生成第二笔订单，也 MUST NOT 重新占用取餐号（§7.8）。

#### Scenario: A rebuild is attempted while the product is still off the menu

- **WHEN** 对因商品已下架而未建单的条目点击人工建单
- **THEN** 确认层展示该原因与解除办法，且不提供确认按钮
- **AND** 没有产生任何订单

#### Scenario: The cause is cleared and the rebuild is retried

- **WHEN** 主账号重新上架该商品后再次建单
- **THEN** 订单生成，交易号、支付时间与实付金额与原记录一致
- **AND** 取餐号为 4 位数字且在该取餐日期内唯一
- **AND** 该条目从待处理列表中消失

#### Scenario: A rebuild is repeated

- **WHEN** 对已建单的条目再次建单
- **THEN** 请求被拒绝
- **AND** 该交易号下仍然只有一笔订单，取餐号不变

### Requirement: Voiding a pending payment refunds in full and reaches the ledger

退款作废 MUST 按原路全额退回（§7.7），契约方法 MUST NOT 接受调用方指定的金额。作废原因 MUST 必填，原因与操作人 MUST 一并记录。

作废 MUST 只把退款推进到 `退款中`，MUST NOT 自行置为 `已退款`（§7.7 只有微信确认才算完成）。

作废 MUST NOT 生成订单。作废后条目 MUST 离开待处理列表。

被作废的条目 MUST 进入财务与对账页的退款台账：顾客确实付过款，这笔退款同样出现在微信账单上。台账条目 MUST 可回溯（以预支付单号与微信交易号定位），并 MUST 与普通退单可区分。

#### Scenario: A pending payment is voided

- **WHEN** 主账号填写原因并确认作废
- **THEN** 退款金额等于已收金额，状态为 `退款中`，操作人为当前登录账号
- **AND** 没有生成任何订单，该条目从待处理列表中消失

#### Scenario: The refund ledger is checked

- **WHEN** 在财务与对账页查看退款记录
- **THEN** 被作废的条目在列，带预支付单号与微信交易号
- **AND** 它与普通退单可以区分

#### Scenario: A void is submitted without a reason

- **WHEN** 作废原因为空或只有空格
- **THEN** 请求被拒绝并说明原因必填

### Requirement: The reconciliation surfaces the received-but-unbuilt gap

对账汇总 MUST 报出区间内「已收款未建单」的笔数与金额。这些款项已经在微信账户中，却不挂在任何订单上 —— 微信账单会比实收合计正好多出这个数。

这两个数 MUST NOT 计入实收合计，净额等式 MUST 保持 `净额 === 实收合计 − 退款合计` 不变；它们 MUST 作为旁注呈现，并 MUST 指向「支付待处理」页。

字段命名 MUST 与「未到账退款笔数」区分开：同名不同义在对账页上会直接导致误读。

页面 MUST 说明该差额的成因与消解方式：条目处理完毕后差额归零。

#### Scenario: The summary is read with unresolved entries

- **WHEN** 区间内存在已收款未建单的条目
- **THEN** 汇总报出其笔数与金额，并指向支付待处理页
- **AND** 实收合计不含它们，净额等式仍然成立

#### Scenario: All entries are resolved

- **WHEN** 区间内的条目全部建单或作废完毕
- **THEN** 已收款未建单的笔数与金额归零
- **AND** 建单的金额进入实收合计，作废的金额进入退款合计

### Requirement: The PC admin has no standalone verify screen

PC 后台 MUST NOT 提供独立的扫码核销页（§15.5.3）。扫码在手机上进行（小程序商户端，评审 §23），PC 侧不需要一个模拟扫码枪的输入页。

PC 页面集合 MUST 与 PRD §3.5 的页面清单逐项一致，页数 MUST 相等。多一页少一页都意味着两边有一侧未同步。

删除 MUST 是能力上的删除：页面文件、导航路由、脚本挂载与该页专属样式 MUST 全部移除，MUST NOT 留下无入口的孤儿页或无选择器命中的死样式。

#### Scenario: The page set is audited

- **WHEN** 检查侧边导航路由、`pages/` 目录与 `index.html` 的脚本挂载
- **THEN** 三处均不存在独立核销页
- **AND** 路由集合与 PRD §3.5 的页面清单逐项一致且页数相等

### Requirement: Manual verify is located through order search

核销 MUST 仍是 `待取餐 → 已完成` 的唯一路径（§6.6），其入口 MUST 保留在订单详情。独立核销页删除后，定位待核销订单的能力 MUST 由订单管理页的搜索承担，MUST NOT 因删页而丢失。

订单管理 MUST 提供按取餐号、订单号、手机号的搜索（§6.6 末条、§6.7）。搜索 MUST 跨泳道：发起核销时并不知道该单当前处于哪个状态。

搜索 MUST NOT 成为一条泳道，MUST NOT 进入订单状态集合。

搜索与泳道 MUST NOT 同时决定列表内容：选择泳道 MUST 退出搜索态，否则使用者无法判断当前看到的是哪一种结果集。

#### Scenario: An order is located for verification

- **WHEN** 主账号在订单管理搜索取餐号、订单号或手机号
- **THEN** 匹配的订单出现在列表中，且结果可跨多个状态
- **AND** 选中后详情仍提供核销动作

#### Scenario: The lane is clicked while a search is active

- **WHEN** 处于搜索态时点击任一泳道
- **THEN** 搜索被清空，列表回到该泳道的完整结果

### Requirement: A pickup code only resolves within the current business day

按取餐号搜索 MUST 只匹配当前营业日的订单（§6.6）。取餐号按取餐日期从 `0001` 累计，跨营业日可能重复（§7.8）—— 不加限定时一个 4 位数字会同时命中多天的订单，凭它核销就会核错单。

订单号与手机号 MUST NOT 受该限制：订单号全局唯一，手机号不存在跨日歧义。4 位数字 MUST 同时按取餐号与手机号片段匹配，但只有取餐号那一半受营业日限制。

当某取餐号在当前营业日无匹配、却存在于其他营业日时，页面 MUST 报出该事实并指出可用的定位方式（订单号 / 手机号搜索，或「未取餐」筛选）。空结果 MUST NOT 被呈现为「不存在该订单」。

#### Scenario: A code from an earlier business day is entered

- **WHEN** 输入一个只存在于更早营业日的取餐号
- **THEN** 列表不返回该订单
- **AND** 页面提示该取餐号在哪个营业日存在，并给出可用的定位方式

#### Scenario: The same order is located by its order number

- **WHEN** 改用订单号搜索同一笔跨营业日订单
- **THEN** 该订单被找到

### Requirement: Verification refuses refunded orders and never double-counts

核销 MUST 拒绝处于 `退款中` 或 `已退款` 的订单（§6.6 已退款订单不得核销）。该校验 MUST 由契约层承担，MUST NOT 依附于任何单一页面 —— 否则删页即删校验。

重复核销 MUST NOT 改变订单状态、MUST NOT 重复计入营收或订单量（§6.6 幂等）。

#### Scenario: A refunded order is verified

- **WHEN** 对 `退款中` 或 `已退款` 的订单发起核销
- **THEN** 请求被拒绝且订单状态不变

#### Scenario: Verification is repeated

- **WHEN** 对已核销的订单再次核销
- **THEN** 请求被拒绝且订单保持 `已完成`

### Requirement: PC order lines carry the same product name snapshot

PC 后台的订单项 MUST 使用与小程序相同的 §15.6.2 元组 `[id, name, qty, price, discountedPrice, flavors?, note?]`。两端 MUST NOT 各自定义订单项形状。

PC 的订单摘要 MUST NOT 按 id 回查商品名称，因此 MUST NOT 保留「商品已删除」一类的回查兜底 —— 该兜底是回查方案的补丁，回查移除后它不再有触发条件，留着会误述当前行为。

§15.6.2 的三条结算恒等式在新元组下 MUST 依然成立：逐行 `qty × price` 之和等于 `subtotal`；逐行 `qty × discountedPrice` 之和等于 `total`；`subtotal - discountCut` 等于 `total`。

#### Scenario: PC order lines are audited

- **WHEN** 检查 PC 订单种子的每一行订单项
- **THEN** 形状与小程序一致，第二项为非空字符串名称
- **AND** 三条结算恒等式逐单成立

#### Scenario: An order summary is rendered

- **WHEN** PC 渲染订单摘要或订单详情
- **THEN** 名称取自订单自身
- **AND** 代码中不存在按 id 取名称的回查或其兜底分支

### Requirement: PC stores daily sell-out apart from shelf state

PC 后台 MUST 使用与小程序相同的模型：`status` 只表达上下架，当日售罄存为 `(productId, serviceDate)` 记录。两端 MUST NOT 各自定义商品销售状态的形状。

PC 的售罄开关（含批量）MUST 写入当前营业日的记录；上下架开关 MUST 只改 `status`。同一次操作 MUST NOT 同时改动两个维度。

#### Scenario: PC product states are audited

- **WHEN** 检查 PC 商品种子与菜品页的读写点
- **THEN** `status` 只出现 `'on'` 与 `'off'`
- **AND** 售罄开关不修改 `status`，上下架开关不写售罄记录

### Requirement: Rebuilding a paid order checks sell-out by pickup date

§7.3 的补建订单校验 MUST 按该笔支付的**取餐日期**判断商品是否售罄，MUST NOT 按当前时刻的全局商品状态判断。

#### Scenario: A pending payment is rebuilt

- **WHEN** 主账号补建一笔「有支付无订单」的条目
- **THEN** 售罄校验针对该条目的取餐日期
- **AND** 其他日期的售罄记录不参与该判断

### Requirement: PC uses the same pickup point constant

PC 后台的取餐点 MUST 取自与小程序同值的常量，MUST NOT 各自维护。营业设置页展示的取餐点 MUST 是该常量，MUST NOT 是另一个字段（如取餐窗口描述）。

PC 订单与支付待处理条目的取餐点快照 MUST 与该常量一致。

#### Scenario: The settings page renders the pickup point

- **WHEN** 主账号打开营业设置
- **THEN** 页面展示的取餐点即该常量
- **AND** 与小程序展示的是同一串文字
