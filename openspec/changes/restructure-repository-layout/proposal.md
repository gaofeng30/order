## Why

仓库根目录直接并列原型代码、文档和工具，无法清楚表达“可运行应用”边界，也会让后续新增 Go API、契约和部署目录时继续扁平扩张。需要先把两个现有前端归入 `apps/`，为后续模块化开发建立稳定路径。

## What Changes

- **BREAKING** 将 `miniprogram/` 移动到 `apps/wechat-miniprogram/`。
- **BREAKING** 将 `web-admin/` 移动到 `apps/web-admin/`。
- 更新微信开发者工具配置、Web 相对资源路径、图片生成工具和当前文档中的有效路径引用。
- 保持小程序原生目录直接作为 `miniprogramRoot`，不额外嵌套 `src/`。
- 保持两个原型的业务行为、页面、mock 数据和运行方式不变。
- 不创建后端、契约、CI 或空目录占位，这些由后续独立 changes 负责。

## Capabilities

### New Capabilities

- `repository-app-layout`: 定义现有可运行前端在 `apps/` 下的稳定位置及路径迁移验收要求。

### Modified Capabilities

无。

## Impact

- owner：本 change 的 writer。
- owned paths：`miniprogram/**`、`web-admin/**`、`apps/**`、`project.config.json`、根 `README.md`、`tools/generate-dish-images/generate_dish_images.py`、`docs/product/online-ordering-system-prd.md`、本 change OpenSpec。
- 依赖：`finalize-docs-archive@983d3f455b121623f174698c9aecf330eddce94c`，因为两个 changes 都修改根 `README.md`；本分支从该已验证 SHA 建立。
- 不影响：产品范围、接口签名、订单数据、支付规则、部署环境和归档资料内容。
