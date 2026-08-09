## ADDED Requirements

### Requirement: Runnable frontends have stable application boundaries

仓库 MUST 将原生微信小程序放在 `apps/wechat-miniprogram/`，将 PC 管理端放在 `apps/web-admin/`，并移除对应的旧根目录。

#### Scenario: Developer locates runnable applications

- **WHEN** 开发者查看仓库 `apps/` 目录
- **THEN** 微信小程序入口存在于 `apps/wechat-miniprogram/app.js`
- **AND** PC 管理端入口存在于 `apps/web-admin/index.html`
- **AND** 根目录不再存在 `miniprogram/` 或 `web-admin/`

### Requirement: WeChat developer tooling resolves the new native root

`project.config.json` MUST 将 `miniprogramRoot` 设置为 `apps/wechat-miniprogram/`，且该目录直接包含原生小程序入口和全部已注册页面。

#### Scenario: Repository root is imported into WeChat DevTools

- **WHEN** 微信开发者工具按根 `project.config.json` 导入仓库
- **THEN** `miniprogramRoot` 指向现存目录
- **AND** `app.json` 注册的每个页面都存在对应 `.js`、`.json`、`.wxml` 和 `.wxss` 文件

### Requirement: Web Admin keeps its direct-preview resources resolvable

PC 管理端 MUST 在不引入构建工具或服务器的前提下，继续通过相对路径读取所需脚本、样式和小程序共享图片。

#### Scenario: Developer opens the Web Admin entry

- **WHEN** 开发者直接打开 `apps/web-admin/index.html`
- **THEN** HTML 引用的本地样式与脚本目标全部存在
- **AND** Web 代码中的共享图片路径解析到 `apps/wechat-miniprogram/assets/`

### Requirement: Repository path migration preserves prototype behavior

目录迁移 MUST 保留原有文件集合和业务实现；只有配置、运行时资源路径、活跃工具和当前说明文档可以发生路径适配。

#### Scenario: Unrelated application files are compared before and after migration

- **WHEN** 对比迁移前后的应用文件相对路径和非适配文件 Git blob hash
- **THEN** 文件数量与路径集合一致
- **AND** 非适配文件内容完全一致

#### Scenario: Active source and documentation are scanned

- **WHEN** 扫描 README、当前产品文档、活跃工具和两个应用源码
- **THEN** 不存在仍指向根 `miniprogram/`、根 `web-admin/` 或运行时 `../miniprogram/` 的有效引用
- **AND** 平台固定值、官方 URL 与历史归档内容不作为错误替换
