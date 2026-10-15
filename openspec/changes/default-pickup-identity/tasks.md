# 默认取餐姓名与统一员工匹配

- 原始开发基线：`f3d45d37b4dd07baf3306d807e668e5654176738`；独立分支/worktree `feat/default-pickup-identity` / `.claude/worktrees/default-pickup-identity`。
- 用户确认：默认取餐姓名与附加手机号独立填写；下单自动填姓名；本人页面手机号完整显示；员工判定必须姓名与主/附加任一号码匹配同一启用记录。PC 对应说明同步。
- 最小成功标准：个人资料、菜单展示价、Quote 的原生 PG / SharedPG 路径一致；无姓名或不匹配按访客；本单临时联系人不改变身份资料；保留既有订单快照。
- Owned paths：小程序 profile/confirm/order-detail 及相应 store；PC staff/staff-import 说明；API merchantidentity/identity/staffidentity/staffdiscount/quote/orderquery/refund；SharedPG identity/quote/payment_prepare/payment_environment 及 schema；PG v45 迁移与 schema 契约；PRD、架构契约、PC/三端/后端清单及本工件目录。
- 依赖：原始开发顺序接续两个 PC 修正，但本功能不依赖其运行时代码；集成 main 仅移植本功能，共享文档冲突排除其他修正内容。既有 `extra_name/extra_name_key` 提升为默认姓名唯一存储，解除与附加手机号必须同时存在的约束，保留原数据；不新增重复姓名事实。
- [x] 核对差异：主手机号原本跳过姓名，附加手机号才核对；现有资料 API 仅返回脱敏号，没有独立姓名保存入口。
- [x] 完成 v45 约束文件、个人资料保存/返回、统一员工身份解析和 Quote 两条路径；补齐 SharedPG 支付准备/支付前环境复核。旧姓名原值保留，默认姓名新写入遵循订单 64 UTF-8 字节约束。
- [x] 小程序独立输入与保存、自动填充不覆盖手工输入、本人完整号码；PC 名单/表单/导入说明。商户订单投影不赋值完整号码字段，历史订单快照不改写。
- [x] 同步 PRD、接口契约、PC/三端/后端清单；历史 B3/B5 冲突条目标明已被本次确认覆盖。
- [x] `GOCACHE=/tmp/order-pc-preparation-go-cache go build -o /tmp/order-default-pickup-api ./cmd/order-api` 退出 0；模块元数据缓存提示不可写但产物生成成功。7 个修改的 JS 使用 `node --check` 通过，`git diff --check` 通过；SQL/迁移仅源文件与契约审查，未连接数据库执行。
- [ ] 真实 UAT：姓名与两个号码组合、停用、空姓名、改姓名、临时联系人、保存失败与多端读取、历史订单不变、窄屏布局。
- 按用户停测覆盖，未新增/运行测试、回归或 Gate；不将静态检查记为 UAT 通过。未获授权不推送、迁移在线数据库或部署。
- 必要整理：将旧附加手机号事务扩为两种互斥字段更新，共用锁和收据；无其他重构。内部 RPC 签名/现有权限不变，旧 phone+name 同时提交不再接受。部署必须配套应用 v45、更新 SharedPG 函数、API 与前端。

## main 集成（2026-10-15）

- 用户授权：先对齐远端 main，再将本 commit 推送至 main；本次明确允许 HTTPS，不修改持久化 remote。
- [x] HTTPS 获取远端 main `bfed590871a2b143b66b5d01f57e26312daafcfc`；独立分支/worktree `integrate/default-pickup-main` / `.claude/worktrees/default-pickup-main`。
- [x] 仅移植 `e09a8d2e5bfd86b17c89bc44a59ef66088adf646`。生产配置、图片存储及补偿任务沿用 main；不并入另外两个 PC 修正 commit。
- [x] 解决 PRD 与接口文档两处上下文冲突，只保留默认姓名与员工匹配变更；同步当前说明中的本人号码展示边界。
- [x] 集成后 `GOCACHE=/tmp/order-pc-preparation-go-cache go build -o /tmp/order-default-pickup-main-api ./cmd/order-api` 退出 0（模块元数据缓存不可写提示不影响产物）；7 个 JS 的 `node --check` 与 `git diff HEAD --check` 通过。按停测指示未运行测试、回归或 Gate，真实 UAT 仍未完成。
