## ADDED Requirements

### Requirement: Historical materials are classified by purpose

仓库 MUST 将正式合同、合同草稿、旧版商品与展示资料、微信小程序开发和运维指南分别保存在已确认的目标目录中。

#### Scenario: Contract materials are separated by status

- **WHEN** 维护者查找当前正式合同或历史合同草稿
- **THEN** 正式合同存在于 `docs/合同相关/`
- **AND** 合同草稿存在于 `docs/archive/contracts/`

#### Scenario: Historical product and display materials are archived

- **WHEN** 维护者查找旧版商品、价目、供应商、Logo 或海报资料
- **THEN** 这些资料存在于 `docs/商品列表和展示（旧版已归档）/`

#### Scenario: Operations guides remain discoverable

- **WHEN** 维护者查找微信小程序注册、支付、域名、备案或腾讯云指南
- **THEN** 这些指南存在于 `docs/微信小程序开发和运维指南/`

### Requirement: Archived file content is preserved

除明确删除项外，仓库 MUST 保证每个旧资料文件在新目录中存在内容完全相同的对应文件，且移动前后的文件总数一致。

#### Scenario: Moved files pass content integrity verification

- **WHEN** 比较 `main` 旧资料目录与本 change 新资料目录的 Git blob hash 多重集
- **THEN** 排除 `docs/design/posters/Untitled` 后两个集合完全一致

### Requirement: Deletion is precisely scoped

仓库 MUST 只删除已确认无用途的空文件 `docs/design/posters/Untitled`，不得因文件名相似而删除其他历史材料。

#### Scenario: Confirmed empty file is absent

- **WHEN** 检查归档后的仓库树
- **THEN** `docs/design/posters/Untitled` 不存在
- **AND** 其他被移动的历史材料仍通过完整性验证

### Requirement: Current documentation navigation resolves

根 `README.md` 和 `docs/README.md` MUST 不再引用本 change 中被迁移的旧路径，且它们列出的本地链接 MUST 指向现存文件或目录。

#### Scenario: Reader follows repository documentation links

- **WHEN** 读者从根 README 或文档索引打开合同、指南或历史资料链接
- **THEN** 每个链接都能解析到仓库中的现存目标
