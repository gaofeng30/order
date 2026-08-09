## Context

当前 `apps/wechat-miniprogram/` 与 `apps/web-admin/` 都通过内存 mock 运行，仓库没有 Go module、服务端入口、数据库或部署配置。已知规模为同时在线 100–200 人、5–10 分钟集中 100–300 单、日 500–1000 单；瓶颈是订单与支付链路的正确性，不是服务拆分或水平扩展能力。

本 change 只建立后续业务模块共同依赖的 HTTP 进程基线。历史架构调研推荐 Go/Gin 模块化单体，腾讯云当前官方资料也将 Gin 列为 CloudBase HTTP 云函数支持的 Go 框架；但生产部署、网络和数据库仍由独立 change 决定，不能混入本 change。

## Goals / Non-Goals

**Goals:**

- 建立一个本地可运行、可测试、可观测并能优雅退出的 `order-api`。
- 固定最小目录、配置、健康检查和进程生命周期契约，让后续业务 change 不重复搭脚手架。
- 在不引入业务实体的前提下，为数据库、鉴权、商品、订单和支付等后续 change 提供稳定依赖。
- 保持单进程模块化单体，匹配当前规模与两人协作成本。

**Non-Goals:**

- 不实现任何业务 API、业务响应 envelope、状态机、数据模型或 migration。
- 不接入微信、MySQL、Redis、消息队列、对象存储或支付 SDK。
- 不提供 Docker、CloudBase、CVM、CI/CD 或生产配置。
- 不修改小程序与 Web adapter，不承诺端到端业务可用。

## Decisions

### D1. 使用仓库级单 Go module 和单 API 进程

根 `go.mod` 使用 module path `github.com/gaofeng30/order`、Go 1.26 language version 和 `toolchain go1.26.5`；唯一可执行入口为 `services/api/cmd/order-api/main.go`。内部结构固定为：

```text
services/api/
├── cmd/order-api/main.go
├── internal/app/             # 依赖装配与进程生命周期
├── internal/config/          # 环境变量解析与校验
├── internal/httpapi/         # router、health 与 HTTP middleware
└── scripts/smoke.sh          # 真实进程 smoke
```

选择根 module 而不是 `go.work` + 多 module，因为当前只有一个 Go 服务，额外 module 会增加依赖版本和本地命令复杂度。Go 1.26.5 是规划时官方最新的 1.26 安全修订，本机当前 1.26.1 不作为可接受候选工具链。选择模块化单体而不是微服务，因为当前容量单实例即可承载，拆服务只会增加分布式事务、发布与排障成本。

### D2. HTTP 层固定使用 Gin v1.12.0

Gin 只承担路由、JSON 输出和 middleware 链，业务逻辑不得进入 handler 或 middleware。本 change 在 `go.mod` 锁定 `github.com/gin-gonic/gin v1.12.0`，不引入 ORM、依赖注入框架或配置框架。

相较纯 `net/http`，Gin 为后续参数绑定、路由分组和统一 middleware 提供一致约定；相较 go-zero、Kratos 或微服务框架，它没有当前项目用不到的 RPC、服务治理和代码生成面。生产平台适配不能反向污染 handler。

### D3. 配置只有两个显式环境变量

- `ORDER_API_HTTP_ADDR`：监听地址，未设置时为 `:8080`。
- `ORDER_API_SHUTDOWN_TIMEOUT`：优雅退出上限，未设置时为 `10s`，必须能被 `time.ParseDuration` 解析且大于 0。

HTTP Server 固定设置非零的 read-header、read、write、idle timeout；这些安全边界不是本期业务配置，不开放更多环境变量。配置在监听端口前一次性加载并校验，错误直接阻止启动，日志只记录非敏感配置。

### D4. 健康端点只表达当前可证明的状态

`GET /health/live` 和 `GET /health/ready` 均返回 HTTP 200、`application/json` 和 `{ "status": "ok" }`。当前服务没有外部依赖，因此 ready 仅表示配置加载完成且 router 已构建；后续数据库 change 必须修改 readiness 契约并使旧验证失效。

非 GET 方法对已知健康路径返回 405，未知路径返回 404，避免探针或错误路由被误判为成功。本 change 不定义 `/api/v1` 或业务错误 envelope。

### D5. 最小可观测性使用 Go 标准库

使用 `log/slog` 输出 JSON 日志。每个请求获得一个服务端生成的 request ID，并写入 `X-Request-ID` 响应头；访问日志包含 request ID、method、path、status、duration，不记录 query、body、authorization、cookie 或手机号等数据。panic recovery 返回 500、记录 request ID 和错误类型，但不向客户端暴露堆栈。

不直接信任客户端传入的 request ID，避免日志关联键被伪造。未来网关透传规则由部署或鉴权 change 单独定义。

### D6. 进程生命周期由 context 驱动

`main` 只负责捕获 `SIGINT`/`SIGTERM`、构造根 context 并把退出码交给可测试的 app 层。监听失败必须返回非零；收到终止信号后停止接收新请求，并在 shutdown timeout 内等待在途请求，超时或 shutdown 错误也返回非零。

通过接口或函数参数注入 listener/handler/logger，使生命周期测试不依赖固定端口。`scripts/smoke.sh` 构建临时二进制、选择空闲本地端口、检查健康端点与 404、发送 `SIGTERM` 并断言进程正常退出。

### D7. 并行 change 只共享一个显式装配点

本 change 完成前独占 `go.mod`、`go.sum`、`services/api/**` 和根 README。集成本 change 后，后续业务 change 可以分别拥有 `services/api/internal/<module>/**`；需要修改 `internal/app/`、router 根装配、`go.mod` 或公共 HTTP 契约的 change 必须声明共享路径并串行集成，不用隐式 `init()` 注册来规避冲突。

任何对 Go/Gin 版本、目录边界、配置名、健康响应、middleware 顺序、进程生命周期、spec、tasks 或候选 SHA 的修改，都会使本 change 的旧验证失效。

## Risks / Trade-offs

- [健康 ready 暂未覆盖数据库] → 响应只声明“进程 ready”，数据库 change 在引入连接时必须同步升级 readiness spec 与测试。
- [Gin 增加第三方依赖面] → 只锁定一个直接框架依赖，使用 `go mod tidy`、`go vet`、race 和测试约束；不引入框架插件集合。
- [根 module 未来包含 jobs] → 当前保持单 module；只有出现真实、可独立发布且依赖隔离的需要时才另起架构 change。
- [并行业务模块争用装配点] → 模块代码可并行，公共装配路径按 owned paths 由一个 writer 串行接入，不引入全局注册副作用。
- [本地 smoke 在慢机器上波动] → 脚本轮询健康端点并设硬超时，不使用固定 sleep 作为成功判定。

## Migration Plan

1. 在实现 worktree 先提交配置、health、middleware 和生命周期的失败测试，记录 Red。
2. 建立 root Go module 与目录骨架，按最小实现逐项转 Green。
3. 重构包边界并重跑同一测试、race、vet、build 与进程 smoke。
4. 更新 README，严格校验 OpenSpec 与 owned-path diff，提交候选 SHA。
5. 在独立 detached worktree 对精确 SHA 重跑全部验收；通过后才允许集成本地 `main`。

回滚时整体撤销该 change，即删除 root Go module、`services/api/` 和 README 对应说明；两个现有前端原型不受影响。

## Open Questions

无。生产使用 CloudBase HTTP 云函数还是其他运行形态、MySQL 连接方式及外部网络配置均被明确拆到后续 change，不阻塞本地 API 基线。
