> 状态：`CANDIDATE`。用户于 2026-08-09 批准方案，当前任务已绑定为 `codex/bootstrap-api-service` worktree 的唯一 backend writer；writer Gate 已通过，等待精确 SHA 独立验证。每勾选一项，必须在该项下记录决定性 Red/Green 命令、结果或设计理由。

## 1. Preflight and Red Evidence

- [x] 1.1 将唯一 writer 绑定到 `codex/bootstrap-api-service` worktree，确认 `main` 与基线均为已验证的 `021e5c87ed31d02406031bb9c53cdf755fb2b071`，确认 Go 1.26.5 toolchain 可用，并重跑 `openspec validate bootstrap-api-service --strict`。
  - 证据：`git worktree list --porcelain`、`git rev-parse HEAD` 与 `git merge-base --is-ancestor 021e5c87... HEAD` 确认 writer 位于 `codex/bootstrap-api-service@8ec2c54a...`，本地 `main` 和基线均为 `021e5c87...`。
  - 证据：`GOPROXY=https://goproxy.cn,direct GOTOOLCHAIN=go1.26.5 go version` 返回 `go version go1.26.5 darwin/arm64`；`openspec validate bootstrap-api-service --strict` 返回 valid。
- [x] 1.2 先新增配置测试，覆盖默认值、合法覆盖、非法/非正 shutdown timeout 和监听失败；运行 focused test 并记录因实现缺失产生的 Red。
  - Red：`GOTOOLCHAIN=go1.26.5 go test ./services/api/internal/config` 因四处 `undefined: Load` 编译失败。监听失败由职责所属的 `app.TestRunReturnsListenError` 覆盖，避免 config 承担网络行为。
- [x] 1.3 先新增 router 与 middleware 测试，覆盖 health 200/JSON、错误方法 405、未知路径 404、服务端 request ID、脱敏访问日志和 panic recovery；运行 focused test 并记录 Red。
  - Red：`GOPROXY=https://goproxy.cn,direct GOTOOLCHAIN=go1.26.5 go test -mod=mod ./services/api/internal/httpapi` 因 `undefined: NewRouter/newRouter` 编译失败。
- [x] 1.4 先新增 app lifecycle 测试和真实进程 smoke 验收，覆盖正常 SIGTERM、在途请求完成、shutdown 超时和非零失败退出；运行并记录 Red。
  - Red：`GOTOOLCHAIN=go1.26.5 go test ./services/api/internal/app` 因 config 尚无非测试实现而失败；`bash services/api/scripts/smoke.sh` 因 `services/api/cmd/order-api` 不存在而失败。测试已覆盖监听失败、正常停止、在途完成与 shutdown 超时；脚本覆盖真实 SIGTERM 和无效配置非零退出。

## 2. Minimal Green Implementation

- [x] 2.1 创建根 `go.mod`/`go.sum`，固定 Go 1.26 language version、`toolchain go1.26.5` 与 Gin v1.12.0，并实现 `internal/config` 使 1.2 的测试转 Green。
  - Green：`GOTOOLCHAIN=go1.26.5 go test ./services/api/internal/config` PASS；`go.mod` 为 `go 1.26.0`、`toolchain go1.26.5`，唯一直接依赖为 `github.com/gin-gonic/gin v1.12.0`。
- [x] 2.2 实现 `internal/httpapi` router 与两个 health handler，使 200/405/404 契约测试转 Green，不新增 `/api/v1` 或业务 envelope。
  - Green：`GOTOOLCHAIN=go1.26.5 go test ./services/api/internal/httpapi` PASS，覆盖两个健康端点、405 与 404。
- [x] 2.3 实现服务端 request ID、JSON `slog` 访问日志和 recovery middleware，使 1.3 剩余测试转 Green，确认日志不含 query、body 或 header 值。
  - Green：同一 HTTP focused test PASS，断言 request ID 不信任客户端值，访问日志含 method/path/status/duration 且不含 query、body、Authorization、Cookie 或任意 header 值；panic 返回 500 且日志不含 panic value。
- [x] 2.4 实现 `internal/app`、`cmd/order-api` 与 `scripts/smoke.sh`，设置固定非零 HTTP timeout，使 1.4 的生命周期与真实进程验收转 Green。
  - Green：`GOTOOLCHAIN=go1.26.5 go test ./services/api/internal/app` PASS；全包 focused regression PASS；`bash services/api/scripts/smoke.sh` 返回 `smoke: PASS`。
- [x] 2.5 更新根 README，只增加 `services/api/` 边界、本地启动、健康检查和验证命令，不宣称业务 API、数据库或部署已完成。
  - Green：README 已增加 API 目录、启动、探活与验证命令，并明确两个前端仍使用 mock、当前无业务 API/数据库/部署接入。

## 3. Refactor Without Contract Drift

- [x] 3.1 运行 `gofmt` 并整理 `cmd`、`app`、`config`、`httpapi` 职责；删除重复实现，保持 handler 无业务逻辑且不引入额外框架。
  - Refactor：`cmd` 仅处理配置/信号/退出码，`app` 仅处理 listener/server 生命周期，`config` 仅解析两个环境变量，`httpapi` 分离 health/router/middleware；HTTP status 改用标准库常量，`gofmt` check PASS。
- [x] 3.2 重跑 1.2–1.4 的同一 focused tests 与 smoke，记录 Refactor 后 Green；如任何公开行为变化，先回到 OpenSpec 并重新批准。
  - Refactor Green：依次重跑 config、httpapi、app focused tests 全部 PASS；`bash services/api/scripts/smoke.sh` 返回 `smoke: PASS`，无公开行为变化。

## 4. Writer Verification and Candidate

- [x] 4.1 执行 `test -z "$(gofmt -l services/api)"`、`go test ./services/api/...`、`go test -race ./services/api/...`、`go vet ./services/api/...`、`go build ./services/api/...`。
  - Writer Gate：五条命令均在 Go 1.26.5 下 PASS；race 覆盖 app/config/httpapi，cmd 无测试文件且 build PASS。
- [x] 4.2 执行 `bash services/api/scripts/smoke.sh`，验证两个健康端点、404、真实 `SIGTERM` 与 clean exit。
  - Writer Gate：脚本构建临时二进制，以 `127.0.0.1:0` 启动，验证 live/ready JSON、未知路径 404、真实 `SIGTERM` 的状态 0，并验证无效配置非零退出；结果 `smoke: PASS`。
- [x] 4.3 执行 `openspec validate bootstrap-api-service --strict`、`git diff --check`，并确认相对基线的所有路径仅属于 `go.mod`、`go.sum`、`services/api/**`、根 README 和本 change OpenSpec。
  - Writer Gate：strict validation 与 `git diff --check` PASS；基线 diff 加 untracked 文件共 18 个路径，全部位于 owned paths。`go list -m` 确认唯一直接第三方依赖为 `github.com/gin-gonic/gin v1.12.0`，源码不存在 `/api/v1`、entity 或 repository。
- [ ] 4.4 检查 tasks 中每项 Red/Green/Refactor 证据后提交 implementation candidate，记录完整 SHA；writer 不得把自测声明为独立验证。

## 5. Exact-SHA Independent Verification

- [ ] 5.1 verifier 在另一个干净 detached worktree 对 implementation candidate 完整 SHA 重跑 4.1–4.3 与 specs 全部场景，只读报告 PASS/FAIL；失败由 writer 修复并产生新 SHA。
- [ ] 5.2 writer 只把 5.1 的候选 SHA、命令和结果写回本文件并勾选已完成任务，提交 final metadata candidate；另一 verifier 必须对这个最终完整 SHA 再跑全部验收，且不再修改任何 artifact，PASS 后 change 才进入 `INDEPENDENT_VERIFIED`。
