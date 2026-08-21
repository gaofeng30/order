## ADDED Requirements

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
