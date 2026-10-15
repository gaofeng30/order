> 2026-10-10 当前执行约束：用户已明确停止全部测试与回归并删除测试用例；旧测试及 Gate 记录仅作为历史证据，未运行不能记为通过。本轮直接完成 UAT SQL 安装、服务构建部署及小程序预览交付；不等待旧测试门禁。实际数据库已安装 113 个函数并登记 38 条平台执行记录，既有业务数据与账本保持不变。

# Tasks

Status: APPROVED / IMPLEMENTING；base/ownership/dependencies 见 proposal。gate_type W3；UI0 target/actual；候选未创建，完整云 UAT 未验证。

- [x] 1.1 从精确 base 新建干净 worktree/branch，读取规则、已授权范围与固定四用途契约。
- [x] 1.2 通过现有公开 loader 取得旧代码行为 Red；无真实外部调用。
- [x] 1.3 OpenSpec strict；最小平台 source 与 UAT config 路由 Green。
- [x] 2.1 缺值/用途/取消/非法材料/生产隔离/并发与修复恢复安全回归。
- [x] 2.2 Refactor 阶段重跑同一 tracer 与受影响回归；记录必要或无需额外重构的理由。
- [x] 3.1 受影响 ordinary/race、TLS/COS、vet/build、隔离 smoke、gofmt/diff/owned/secret 检查。
- [x] 3.2 Writer 两轴自审、C/T/V/R 与提交前 owned-only 核对。
- [ ] 3.3 本地候选提交；完整 SHA 与提交后 clean 由仓库外 writer-receipt.json.log 记录，不为自引用 SHA 修改已提交候选。
- [ ] 4.1 不同 verifier 的 clean detached exact-SHA 只读验证。
- [ ] 4.2 root 完整合成四用途 CloudRun exact-SHA probe、版本与清理复读；NOT_RUN。

## Evidence

工件目录 `.scratch/use-uat-cloudrun-platform-secrets/`（文件/目录以 `.log` 结尾，仓库现有规则忽略，不提交）。

- change: use-uat-cloudrun-platform-secrets
  gate_type: W3
  ui_level_target: UI0
  ui_level_actual: UI0
  base_sha: bec51e6fe1017bcd04d471ea74f47d0dc1391168
  candidate_sha: NOT_CREATED
  phase: red
  command_or_action: GOPROXY=off GOTOOLCHAIN=go1.26.5 go test ./services/api/internal/config -run '^TestUATRuntimeLoadsFourPlatformPurposesWithoutCloudIdentity$' -count=1
  exit_result: 1
  sanitized_summary: 旧公开 Load 返回 ORDER_DB_PASSWORD production_secret_environment_forbidden，目标业务变量行为缺失；不是新符号编译失败。
  artifact_or_environment: isolated writer；red-public-loader.log；合成 RSA/JSON；全 HTTP 拦截。
  unverified_boundary: 没有 CloudRun、PG/TLS、客户材料、真实资金或全量 UAT。
  external_asset: N/A for this local Red

### 最终相同 harness 的 Red / Green / Refactor

- 为构造真实 native 微信支付 client，初次 Green 发现测试泛型 RoundTripper 与既有 client 克隆 `*http.Transport` 不兼容；只把测试拦截器改为真实 Transport 的阻断 Dial，没有改产品 client。该 harness panic 不作为产品行为 Red。
- 最终同一 tracer 测试文件覆盖到精确 base 的隔离 `git archive` 后重跑同命令：exit 1，仍为 `ORDER_DB_PASSWORD: production_secret_environment_forbidden`（`red-current-harness.log`）。归档未含任何本次业务实现、未连接云，完成后删除。
- Writer 同一命令 Green 与 Refactor 均 exit 0（`green-public-loader.log`、`refactor-public-loader.log`）。四用途材料经当前公开 loader 形成真实 RSA2048 支付 client 与版本化 AES cipher，CloudBase identity 留空，所有 HTTP Dial 阻断且计数 0。
- 配置 source 仅一个静态 allowlist，无共享状态；无需额外重构。没有抽取旁路、修改 parser、业务、SQL 或生产配置。Refactor 重跑同 tracer 与完整受影响 ordinary/race，未扩展相邻代码。

### 受影响检查

最终 ordinary/race 实际命令：`python3 .scratch/use-uat-cloudrun-platform-secrets/run-local-check.py <ordinary|race> test [-race] -json -count=1 ./services/api/internal/config ./services/api/cmd/order-api ./services/api/cmd/order-migrate ./services/api/cmd/order-bootstrap ./services/api/internal/database ./services/api/internal/objectstore`；runner 完成后重命名为 `run-local-check.py.log`，内容未变。

- 两次各 exit 0：284 PASS、4 SKIP、0 FAIL；六包全部 PASS。SKIP 为四个 opt-in UI1/L3 server helper，不是本切片 UI 或 cloud 验收：`TestPCClosureUI1Server`、`TestPCRemainingUI1Server`、`TestRefundUnclaimedL3Server`、`TestTransactionOrderL3Server`。
- 仓库 `pg-local-gate.py` 使用 ephemeral loopback PostgreSQL16.13、Go1.26.5、合成资料、SCRAM 临时密码。两次 source_unchanged=true、cloud_connected=false、cleanup=stopped、password_removed/data_removed=true；保留 `ordinary-evidence.log/`、`race-evidence.log/` 与摘要。它是本地配置/consumer/TLS/COS 回归，不能称为 CloudBase PG17 接入或 full UAT。
- 首次将 TMPDIR 放长 `.scratch` 路径导致 Unix socket 超过 103-byte 上限（Go tests 未启动）；既有 runner 改回 `/tmp` 临时运行，完成后才将已清理的证据移入本 change `.scratch`。失败记录保留于 `tmp.log/`，cleanup=confirmed_not_running、password/data removed，未改 canonical runner 或数据库契约。
- 相同六包 `GOPROXY=off GOTOOLCHAIN=go1.26.5 go vet ...`、`go build ...` exit 0；`vet.log`/`build.log`。
- 最小隔离环境执行 `bash services/api/scripts/smoke.sh` exit 0（`smoke.log`）：development localhost、无云/客户凭据；live200、database-unreachable ready503、错误路由/认证拒绝及脱敏退出检查。
- `test -z "$(gofmt -l services/api)"`、`git diff --check`、OpenSpec strict 均 exit 0；只按 owned allowlist暂存，`.scratch` 全部忽略且无真实材料。不存在硬编码私钥/证书/APIv3/key、生产写入、push 或部署。

### Writer 自审与可恢复性

Standards：固定四个 logical purpose 唯一映射，无 env 拼接/selector/fallback；source 取消/缺值错误不带值；UAT 仅放行旧两业务变量，生产 CVM/SSM 与 COS 身份路径不变；TLS/DSN/永久腾讯凭据条件保持。

Correctness：公开 tracer、四缺值/空值修复恢复、非法 parser、other-purpose/production 隔离、TLS disabled 拒绝、16并发只读/旧 parsed snapshot 不变/key拷贝、原 TLS CA/host/no-fallback 与 COS tuple 回归均纳入本次 ordinary/race。启动/显式 load 失败无替代源；修复有效同环境值后重启/new load 恢复。轮换须新 revision，撤销材料不可简单退回旧 revision。

提交前评分 C9/T10/V8/R9 =36；各项 >=8、Writer 硬阻断 0。V8 仅表示提交后的 exact-SHA clean-detached 验证包可交付，未获得 independent PASS。source/验收/候选变动会使验证失效；缺失外部资产继续在 full-UAT 任务保留，不能越级报告。

真实 PG/TLS、真实微信/支付/退款/通知终态、真机与 95 cases/25 pages 属于 full-UAT 外部任务，本候选不声称通过。UI/SQL/生产部署均不改，相应实现 Gate N/A；不可沿用旧候选 1459 PASS 证明新 SHA。集成、archive、push、部署均未执行。
