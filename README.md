# 在线点单微信小程序

这是一个面向政府/企业食堂场景的在线点单微信小程序原型。当前版本用于客户预览和上线前方案评审，重点覆盖用户点单、自提取餐、商户商品管理、订单管理、核销和基础经营看板。

## 当前定位

- **产品阶段**：P0 可预览交互原型。
- **运行形态**：原生微信小程序、静态 Web 管理端，以及独立的 Go API 进程基线。
- **数据来源**：两个前端仍使用本地 mock 数据；API 已具备 MySQL 8.0 连接、显式迁移与 DB/schema readiness 基础，但仍没有业务表、业务接口或支付接入。
- **评审目标**：确认页面效果、业务流程、功能范围和后续正式开发边界。

## 目录结构

```text
.
├── apps/
│   ├── wechat-miniprogram/   # 原生微信小程序源码根目录
│   │   ├── app.js            # 小程序入口逻辑
│   │   ├── app.json          # 页面、窗口、tabBar 配置
│   │   ├── app.wxss          # 全局样式
│   │   ├── sitemap.json      # 小程序索引配置
│   │   ├── pages/            # 用户端与商户端页面
│   │   ├── components/       # 复用组件
│   │   ├── assets/           # 静态资源
│   │   ├── mock/             # 演示数据
│   │   └── utils/            # 本地状态与工具函数
│   └── web-admin/            # 商户端 PC 网页版
│       ├── index.html        # 唯一入口，浏览器直接打开
│       ├── app.css           # 设计 token + 布局与控件
│       ├── app.js            # hash 路由、左侧导航、顶栏账号下拉
│       ├── data/             # 演示数据与接口契约层
│       ├── ui/               # 表格 / 抽屉 / 弹层 / Toast / 图标
│       └── pages/            # 11 条路由，覆盖商户端全部功能
├── services/
│   └── api/                   # Go API 进程基线（配置、健康检查、日志与优雅退出）
│       ├── cmd/order-api/     # API 进程入口（不会自动迁移）
│       ├── cmd/order-migrate/ # 唯一 forward-only 迁移入口
│       ├── internal/          # app、config、database、migrate 与 httpapi
│       ├── migrations/        # compile-time embedded SQL migrations
│       └── scripts/           # 无 DB smoke 与隔离 MySQL 8.0 W3 验收
├── docs/
│   ├── README.md             # 文档索引与建议阅读顺序
│   ├── product/              # PRD、需求、技术方案、客户沟通材料
│   ├── 合同相关/             # 当前正式合同
│   ├── 微信小程序开发和运维指南/ # 注册、支付、域名、备案与云资源指南
│   ├── archive/contracts/    # 历史合同草稿
│   └── 商品列表和展示（旧版已归档）/ # 旧版商品、价目与视觉资料
├── project.config.json       # 微信开发者工具项目配置
├── go.mod                    # 仓库级 Go module
├── LICENSE
└── README.md
```

## 快速预览

1. 打开微信开发者工具。
2. 选择“导入项目”。
3. 项目目录选择本仓库根目录。
4. AppID 可使用 `project.config.json` 中的配置，或在开发者工具中选择测试号。
5. 编译后从启动页选择“用户端点单”或“商户端管理”。

`project.config.json` 已配置 `miniprogramRoot: "apps/wechat-miniprogram/"`，因此导入根目录即可识别小程序源码。

### 商户端 PC 网页版

双击 `apps/web-admin/index.html` 用浏览器打开即可，无需构建、无需服务器、无需安装依赖。建议窗口宽度 ≥ 1280px。

两个入口互相独立、可同时打开：想看手机形态开微信开发者工具，想看电脑形态开浏览器。

> **当前阶段两端数据不互通。** PC 端自带一套独立演示数据，在小程序里下的单电脑上看不到，反之亦然。
> 这是 P0 原型阶段的取舍：真实打通需要后端 API，属正式开发范围。
> 两端的接口契约层（`apps/web-admin/data/api.js` 与 `apps/wechat-miniprogram/utils/api.js`）方法名、入参、返回结构完全一致，
> 后端就位后各自把内部实现换成 HTTP 请求即可，页面代码不动。

### API 服务基线

需要 Go 1.26.5 和隔离的 MySQL 8.0。development/test 只接受结构化连接配置，不接受原始 DSN：

```bash
export ORDER_ENV=development
export ORDER_DB_HOST=127.0.0.1
export ORDER_DB_PORT=3306
export ORDER_DB_NAME=order_development
export ORDER_DB_USER=order_development
export ORDER_DB_PASSWORD='<local-only password>'
export ORDER_DB_TLS_MODE=disabled

GOTOOLCHAIN=go1.26.5 go run ./services/api/cmd/order-migrate
GOTOOLCHAIN=go1.26.5 go run ./services/api/cmd/order-api
```

必须先显式执行零参数 `order-migrate`，API 启动和健康检查绝不会自动迁移。默认监听 `:8080`：

```bash
curl http://127.0.0.1:8080/health/live
curl -i http://127.0.0.1:8080/health/ready
```

`/health/live` 只反映进程存活并返回 200；`/health/ready` 仅在真实 MySQL 8.0 可达且 embedded migration history 完全 current 时返回 200，否则返回 503 与稳定 reason。可通过 `ORDER_API_HTTP_ADDR` 修改监听地址，通过 `ORDER_API_SHUTDOWN_TIMEOUT` 修改优雅退出上限。

production 模式拒绝 `ORDER_DB_PASSWORD` 和 `ORDER_DB_DSN`；运行时 SSM secret loader 尚未实现，因此当前 production 模式会 fail fast，不能启动。不得用 development/test 环境变量绕过该边界。

完整本地验证命令：

```bash
go test ./services/api/...
go test -race ./services/api/...
go vet ./services/api/...
go build ./services/api/...
bash services/api/scripts/smoke.sh
# 由专属隔离 MySQL 8.0 环境注入 ORDER_TEST_MYSQL_* 后运行：
bash services/api/scripts/mysql-integration.sh
```

当前 migration 集合只创建 `schema_migrations`；没有商品、用户、订单、库存、支付等业务表，也没有 repository、ORM、seed、down/force/repair 命令或业务 API。两个前端当前不会调用该进程。

## 评审走查路径

用户端主链路：

1. 启动页进入“用户端点单”。
2. 首页查看营业状态、取餐地点、公告、今日推荐和外卖预留入口。
3. 进入点单页，切换分类并加购菜品；售罄和下架菜品会显示不可购买状态。
4. 打开购物车弹层调整数量，进入确认订单。
5. 填写联系人、手机号和备注，模拟确认支付。
6. 查看支付成功页、取餐号、二维码样式凭证和订单详情。

商户端主链路：

1. 启动页进入“商户端管理”。
2. 在数据看板查看今日营收、订单数、访问量、点击量和销量排行。
3. 进入订单管理，查看用户刚提交的新订单。
4. 进入核销页，使用默认取餐号完成手动核销。
5. 返回订单详情或数据看板，确认订单状态更新。
6. 继续查看商品管理、分类管理和营业设置的主要交互。

商户端 PC 网页版主链路（浏览器打开 `apps/web-admin/index.html`）：

1. 工作台查看四项 KPI、今日待办、实时订单与销量排行；直接在实时订单上点“备好”推进状态。
2. 订单管理切换泳道，选中订单在右侧面板看完整单据，就地推进到下一状态。
3. 扫码核销输入 `A118` 回车（USB 扫码枪扫完自动回车，与手动输入等价），核对后确认核销。
4. 菜品管理勾选多个菜品，用“批量改价”按百分比统一调价；打开编辑抽屉改价、换图。
5. 分类管理拖动手柄调整用户端分类顺序。
6. 会员名单页点“批量导入”，把 CSV 拖进去看新增 / 更新 / 异常分列预览后确认导入。
7. 右上角账号下拉切换营业状态，顶栏状态胶囊同步。

> 会员等级、会员名单、批量导入、优惠券四页为二期能力，不在一期合同范围，界面上已标注。

## 上线前规范检查

当前原型已按发布审阅习惯拆分为“配置、源码、文档、设计规则”四类目录。正式上线前仍需完成以下事项：

- 接入真实微信登录、微信支付、支付回调和退款流程。
- 接入后端 API、数据库、对象存储和订单核销服务。
- 将 mock 数据替换为服务端数据，并补充异常、空状态和权限校验。
- 确认正式 AppID、服务器域名、业务域名、隐私协议和小程序类目。
- 在微信开发者工具中完成体验版上传、真机预览、性能检查和提交审核。
- 根据 `docs/product/online-ordering-system-customer-discussion.md` 回填客户确认后的业务规则。

## 文档索引

- [文档总览](./docs/README.md)
- [需求文档](./docs/product/online-ordering-system-requirements.md)
- [技术文档](./docs/product/online-ordering-system-technical.md)
- [客户沟通与待讨论事项](./docs/product/online-ordering-system-customer-discussion.md)
- [PRD 草稿](./docs/product/online-ordering-system-prd.md)
- [微信小程序开发和运维指南](./docs/微信小程序开发和运维指南/)
- [历史商品与视觉资料](./docs/商品列表和展示（旧版已归档）/)

## 版本说明

当前版本适合客户演示、内部评审和正式开发前的范围确认，不应直接作为生产系统发布。正式开发阶段需要补齐业务后端、支付、安全、权限、监控和发布审核配置。
