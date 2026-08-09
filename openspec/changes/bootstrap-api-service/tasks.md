> 状态：`DRAFT`。本文件完成不等于授权实现；用户批准并绑定唯一 backend writer 后才能执行。每勾选一项，必须在该项下记录决定性 Red/Green 命令、结果或设计理由。

## 1. Preflight and Red Evidence

- [ ] 1.1 将唯一 writer 绑定到 `codex/bootstrap-api-service` worktree，确认 `main` 与基线均为已验证的 `021e5c87ed31d02406031bb9c53cdf755fb2b071`，确认 Go 1.26.5 toolchain 可用，并重跑 `openspec validate bootstrap-api-service --strict`。
- [ ] 1.2 先新增配置测试，覆盖默认值、合法覆盖、非法/非正 shutdown timeout 和监听失败；运行 focused test 并记录因实现缺失产生的 Red。
- [ ] 1.3 先新增 router 与 middleware 测试，覆盖 health 200/JSON、错误方法 405、未知路径 404、服务端 request ID、脱敏访问日志和 panic recovery；运行 focused test 并记录 Red。
- [ ] 1.4 先新增 app lifecycle 测试和真实进程 smoke 验收，覆盖正常 SIGTERM、在途请求完成、shutdown 超时和非零失败退出；运行并记录 Red。

## 2. Minimal Green Implementation

- [ ] 2.1 创建根 `go.mod`/`go.sum`，固定 Go 1.26 language version、`toolchain go1.26.5` 与 Gin v1.12.0，并实现 `internal/config` 使 1.2 的测试转 Green。
- [ ] 2.2 实现 `internal/httpapi` router 与两个 health handler，使 200/405/404 契约测试转 Green，不新增 `/api/v1` 或业务 envelope。
- [ ] 2.3 实现服务端 request ID、JSON `slog` 访问日志和 recovery middleware，使 1.3 剩余测试转 Green，确认日志不含 query、body 或 header 值。
- [ ] 2.4 实现 `internal/app`、`cmd/order-api` 与 `scripts/smoke.sh`，设置固定非零 HTTP timeout，使 1.4 的生命周期与真实进程验收转 Green。
- [ ] 2.5 更新根 README，只增加 `services/api/` 边界、本地启动、健康检查和验证命令，不宣称业务 API、数据库或部署已完成。

## 3. Refactor Without Contract Drift

- [ ] 3.1 运行 `gofmt` 并整理 `cmd`、`app`、`config`、`httpapi` 职责；删除重复实现，保持 handler 无业务逻辑且不引入额外框架。
- [ ] 3.2 重跑 1.2–1.4 的同一 focused tests 与 smoke，记录 Refactor 后 Green；如任何公开行为变化，先回到 OpenSpec 并重新批准。

## 4. Writer Verification and Candidate

- [ ] 4.1 执行 `test -z "$(gofmt -l services/api)"`、`go test ./services/api/...`、`go test -race ./services/api/...`、`go vet ./services/api/...`、`go build ./services/api/...`。
- [ ] 4.2 执行 `bash services/api/scripts/smoke.sh`，验证两个健康端点、404、真实 `SIGTERM` 与 clean exit。
- [ ] 4.3 执行 `openspec validate bootstrap-api-service --strict`、`git diff --check`，并确认相对基线的所有路径仅属于 `go.mod`、`go.sum`、`services/api/**`、根 README 和本 change OpenSpec。
- [ ] 4.4 检查 tasks 中每项 Red/Green/Refactor 证据后提交 implementation candidate，记录完整 SHA；writer 不得把自测声明为独立验证。

## 5. Exact-SHA Independent Verification

- [ ] 5.1 verifier 在另一个干净 detached worktree 对 implementation candidate 完整 SHA 重跑 4.1–4.3 与 specs 全部场景，只读报告 PASS/FAIL；失败由 writer 修复并产生新 SHA。
- [ ] 5.2 writer 只把 5.1 的候选 SHA、命令和结果写回本文件并勾选已完成任务，提交 final metadata candidate；另一 verifier 必须对这个最终完整 SHA 再跑全部验收，且不再修改任何 artifact，PASS 后 change 才进入 `INDEPENDENT_VERIFIED`。
