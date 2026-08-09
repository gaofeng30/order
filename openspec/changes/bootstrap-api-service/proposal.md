## Why

当前仓库只有两个基于内存 mock 的前端原型，没有可运行的服务端工程；订单、支付、商品等后续 change 因而缺少共同且可测试的承载边界。先建立一个不含业务逻辑的 Go API 进程基线，后续垂直能力即可在互斥 owned paths 中独立实现，而不重复搭建启动、健康检查和进程生命周期。

## What Changes

- 建立仓库级 Go module 和 `services/api/` 模块化单体目录，提供唯一可执行入口 `order-api`。
- 提供严格的环境配置加载、HTTP Server 超时、结构化访问日志、请求 ID、panic recovery 和优雅退出。
- 新增只表达进程状态的 `GET /health/live` 与 `GET /health/ready`；当前 readiness 不探测尚未引入的数据库。
- 建立该服务的单元测试、race、vet、build 和进程级 smoke 验收入口。
- 更新根 README，说明 API 服务的边界和本地启动方式。
- 不实现商品、用户、鉴权、订单、支付、退款、核销、统计、数据库、对象存储、定时任务或前端接入。

## Capabilities

### New Capabilities

- `api-service-bootstrap`: 定义 Go API 服务能被确定性配置、启动、探活、观测和优雅停止的最小运行契约。

### Modified Capabilities

无。

## Impact

- 状态：`DRAFT`；本 change 只完成规划，不授权实现。
- owner：后续绑定到 `codex/bootstrap-api-service` worktree 的唯一 backend writer；绑定前不得进入 `IMPLEMENTING`。
- owned paths：`go.mod`、`go.sum`、`services/api/**`、根 `README.md`、`openspec/changes/bootstrap-api-service/**`。
- shared contracts：新增运维端点 `/health/live`、`/health/ready`；不新增业务 API，不修改两个前端 adapter。
- 依赖：本地 `main@021e5c87ed31d02406031bb9c53cdf755fb2b071`；该 SHA 已完成独立验证并作为本规划 worktree 的精确基线，`origin/main` 暂未推送。
- 非目标：数据库及 migration、CloudBase/CVM 部署清单、Docker、CI/CD、微信登录与支付 SDK、业务实体与公共业务响应 envelope。
- 最小成功标准：严格 OpenSpec 校验通过；`go test`、`go test -race`、`go vet`、`go build` 通过；真实进程对两个健康端点返回约定结果、未知路径不误报成功、收到终止信号后在超时内退出；diff 不越过 owned paths。
