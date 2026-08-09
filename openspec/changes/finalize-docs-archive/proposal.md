## Why

仓库中的历史合同、商品资料、设计产物和上线指南已经被重新分类，但旧路径仍被文档入口引用，且移动结果尚未形成可验证的变更边界。需要先收口这批归档，确保后续仓库重构建立在清晰、可追溯的文档基线上。

## What Changes

- 保留已经完成的历史资料分类移动，不修改归档文件内容。
- 将合同草稿归档到 `docs/archive/contracts/`，正式合同继续保留在 `docs/合同相关/`。
- 删除已确认无用途的空文件 `docs/design/posters/Untitled`。
- 更新根 README 和文档索引中的旧路径，使所有导航链接指向现存文件或目录。
- 验证每个被移动文件与 `main` 原路径内容一致，并确认除指定空文件外没有资料丢失。

## Capabilities

### New Capabilities

- `documentation-archive`: 定义历史资料归档、入口导航和移动完整性的可验收要求。

### Modified Capabilities

无。

## Impact

- 影响范围：`docs/`、根 `README.md` 和本 change 的 OpenSpec 文件。
- 不影响：小程序、Web 管理端、公共接口、业务数据和部署配置。
- 路径调整会改变历史资料的仓库位置，但不改变文件内容。
