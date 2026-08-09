## Context

当前仓库有两个可运行入口：原生微信小程序 `miniprogram/` 和零构建 PC 管理端 `web-admin/`。它们都位于根目录，Web 还通过 `../miniprogram/assets/` 读取小程序图片。后续目标结构会继续增加 `services/`、`contracts/`、`deploy/` 等顶层边界，因此先迁移现有应用，避免把目录移动与框架升级、后端初始化混成一个 change。

本 change 基于已经独立验证的 `finalize-docs-archive@983d3f4`，因为根 README 是共享写点；治理 change 在另一 worktree 独立推进，不与本 change 争用文件。

## Goals / Non-Goals

**Goals:**

- 将两个现有应用稳定放入 `apps/`，名称明确且不依赖团队黑话。
- 让微信开发者工具、Web 资源、工具脚本和当前文档在迁移后仍指向真实路径。
- 证明除路径适配文件外，其余应用文件内容没有变化。
- 保持原型现有语法、页面集合和直接预览方式可用。

**Non-Goals:**

- 不引入 Vue、TypeScript、Vite、npm workspace 或前端构建链。
- 不建立 Go API、数据库、HTTP 契约、测试框架、CI 或部署目录。
- 不重构小程序页面、Web 全局变量或 mock 业务逻辑。
- 不改写 `docs/商品列表和展示（旧版已归档）/` 内的历史路径。

## Decisions

### D1. 应用统一放在 `apps/`

原生小程序使用 `apps/wechat-miniprogram/`，PC 管理端使用 `apps/web-admin/`。`wechat-miniprogram` 比 `miniapp` 或 `miniprogram` 更明确表达微信平台与产品形态；`web-admin` 保留现有语义。

### D2. 小程序根目录不增加 `src/`

`project.config.json` 的 `miniprogramRoot` 直接指向 `apps/wechat-miniprogram/`。原生微信工程已经以 `app.js`、`app.json`、`pages/` 为根，再套一层 `src/` 只会增加配置和路径复杂度，不产生模块边界价值。

### D3. Web 保持零构建相对资源关系

两个应用迁移后是兄弟目录，因此 Web 对共享图片的运行时相对路径从 `../miniprogram/` 改为 `../wechat-miniprogram/`。代码注释和仓库文档使用完整仓库路径 `apps/wechat-miniprogram/`，避免与运行时 URL 混淆。

### D4. 只修改活跃路径引用

更新根 README、`apps/wechat-miniprogram/README.md`、当前 PRD 和活跃工具脚本。历史归档中的旧路径属于当时证据，保持原样。微信官方 URL 中的 `/miniprogram/` 和 `compileType: "miniprogram"` 是平台值，不进行替换。

### D5. 迁移完整性按“路径集合 + 修改白名单”验证

先比较移动前后的文件相对路径集合和文件总数；对没有路径引用的文件逐个比较 Git blob hash。明确需要适配路径的文件进入小型白名单，并通过语法、JSON、页面完整性、资源存在性和旧路径扫描验证。

## Risks / Trade-offs

- [遗漏硬编码路径导致预览资源失败] → 扫描活跃代码和工具中的旧路径，并验证每个 Web 本地资源引用目标存在。
- [机械移动夹带业务修改] → 对非白名单文件做 blob hash 比较，白名单只允许路径和说明文字变化。
- [与归档 change 的 README 冲突] → 明确 stacked dependency，从已验证归档 SHA 开分支。
- [后续框架选择影响目录] → 本 change 只建立稳定应用边界；框架升级单独 change，不在这里预留双轨。

## Migration Plan

1. 建立失败基线：目标目录不存在、`miniprogramRoot` 与活跃引用仍指向旧路径。
2. 机械移动两个完整目录。
3. 更新配置、运行时资源路径、活跃工具和文档引用。
4. 运行结构完整性、路径、JS、JSON、页面和 OpenSpec 验证。
5. 提交候选 SHA，在独立 worktree 复验。

回滚时整体撤销本 change，即恢复两个根目录和原配置。

## Open Questions

无。
