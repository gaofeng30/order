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
