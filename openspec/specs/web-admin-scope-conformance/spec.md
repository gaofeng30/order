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
