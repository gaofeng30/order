> 状态：`APPROVED`。`gate_type=W3`、`ui_level_target=UI0`、`ui_level_actual=NOT_RUN`、`base_sha=5ba5340cf9098724c0eb2284fdc5b14cb97be5dc`、`candidate_sha=NOT_CREATED`。当前本地 MySQL 环境为 `NOT_ESTABLISHED`，真实 W3 为 `NOT_RUN`；这不是 `BLOCKED_EXTERNAL`。本轮只记录 approval，所有 tasks 必须保持未勾选，不执行 apply。
>
> `approval_date=2026-08-13`；`approver=主 Agent`。批准依据是单能力 W3/UI0、canonical health 完整 MODIFIED、MySQL/真实 W3/production secret 边界唯一冻结、无行为未决、owned/依赖/非目标/42 tasks 完整且 strict PASS；这是主 Agent 在用户授予的自主裁决范围内作出的规划裁决，不表述为用户亲自确认。
>
> 每项完成后按 `docs/quality/change-quality-gates.md` 记录：change/gate/ui/base/candidate/phase、实际命令或操作、exit result、首个脱敏结果、artifact/environment、未验证边界及本地环境状态。APPROVED 尚未实现，不计算 C/T/V/R 实现分数；candidate 目标 `C=10、T=10、V=8、R=8` 只在真实证据齐全后评定。

## 1. Approval, Ownership and Local MySQL Prerequisite

- [ ] 1.1 核验本 change 的 APPROVED 记录；重新完整读取 proposal、两份 spec、design、tasks、根 `AGENTS.md`、`docs/quality/change-quality-gates.md`、canonical `api-service-bootstrap` 与归档 bootstrap/production/product baselines，运行 `openspec validate establish-mysql-persistence-foundation --strict`，确认没有 Open Question 后才能进入 `IMPLEMENTING`。
- [ ] 1.2 在任何 Red 前确认 branch/worktree 仍是唯一 writer、HEAD 包含 exact base、index/worktree clean，且当前 diff 只含规划 commit；重新审计 main/router/config/README/smoke 的实际形态和全部 owned/protected paths，不吸收或回退其他 worker 改动。
- [ ] 1.3 只读运行 `brew info --json=v2 colima`，要求 stable=`0.10.3` 且 arm64 bottle SHA-256=`a9dfd1fa0a4aee62fef75974f39f174e4da774f7ba495c43dd0bcc23633381b8`；若 `colima` 不存在则按锁定 bottle 安装，存在则版本必须精确一致。检查 `order-mysql-w3` profile 不存在；若已存在，停止并先确认归属，不得删除或复用未知 profile。
- [ ] 1.4 创建唯一 `order-mysql-w3` profile，固定 Docker runtime、`aarch64`/VZ、2 CPU、4 GiB memory、20 GiB disk、Kubernetes off、workspace mount none、无外部 network address；记录 Colima/Docker engine 版本和 profile 配置的脱敏证据，环境状态从 `NOT_ESTABLISHED` 变为 `ESTABLISHED`。
- [ ] 1.5 在 0600 `mktemp -d` 中生成一次性 MySQL 凭据；仅在 profile 内 pull/run `docker.io/library/mysql@sha256:0e7040b532c0f2ac8cb822695d33025522acd5252175cb104a5929aa66b40222`，容器名 `order-mysql-w3`、只映射 `127.0.0.1` 随机端口、不挂宿主目录/复用 volume。核对 repo digest、`linux/arm64`、container health 与 `SELECT VERSION()`=`8.0.x`；不得输出 secret/DSN。任一步失败记 FAIL，由 writer 从首错继续，未成功不得进入 Red。

## 2. Red: Configuration, Pool and HTTP Contract

- [ ] 2.1 先扩展 `internal/config` tests，覆盖 `ORDER_ENV` 枚举、dev/test 六个结构化字段、port/TLS 校验、全部模式拒绝 `ORDER_DB_DSN`、production 拒绝 env password 并固定 `production_secret_source_unavailable`；用 canary secret 断言 error/log 不回显。运行 focused test，记录因目标实现缺失产生的 Red。
- [ ] 2.2 先新增 `internal/database` tests，冻结 `mysql.Config` 的 TCP/UTC/parseTime/utf8mb4/collation/TLS/dial-read-write timeout/unsafe flags、NopLogger 和 pool 10/10/3m/1m；断言 constructor 惰性不 ping、结构化错误不泄密。运行 focused test，记录 Red。
- [ ] 2.3 先修改 router tests，覆盖 live 不调用 DB、ready current=200，以及 unreachable/incompatible/uninitialized/dirty/behind/too-new/checksum=503 的精确 JSON；保留 405/404、middleware 与无敏感正文断言。运行 `go test ./services/api/internal/httpapi -run 'TestHealth' -count=1`，记录 Red。
- [ ] 2.4 先把无数据库 `smoke.sh` 的目标改为结构化 development 配置、live 200、ready 503/database_unreachable、404、SIGTERM clean exit 和 invalid config exit 1；运行并记录旧实现把 ready 返回 200 或不识别 DB 配置的 Red。

## 3. Red: Migration Files, Runner and CLI

- [ ] 3.1 先新增 `migrate.Load` tests，覆盖六位连续编号、缺号/重复、UTF-8/LF/newline、单终止分号、单 statement、禁止 down/seed/DELIMITER/SOURCE/LOAD DATA，以及原始 bytes SHA-256；使用 `fstest.MapFS` 记录因 loader 缺失产生的 Red。
- [ ] 3.2 先新增 history/state tests，覆盖 table absent、version 1 精确表形、clean prefix、current、dirty、too-new、缺失历史、name/checksum drift、MySQL 非 8.0/session 非 UTC或 utf8mb4；fake 只验证内部分类，不记 W3 PASS。
- [ ] 3.3 先新增 runner orchestration tests，断言 `GET_LOCK`、全部 schema/history/SQL 与 `RELEASE_LOCK` 只走同一专用 connection；覆盖 lock=0/NULL/error、release/close、首次、重复、statement failure 留 dirty 且无 clean/后续、再次运行先拒绝 dirty。运行 focused tests 记录 Red。
- [ ] 3.4 先新增 `order-migrate` tests，覆盖零参数 exit 0、所有运行错误 exit 1、任意参数 exit 2 且不连接，以及成功/失败 JSON 字段白名单；用 DSN/password/SQL/server-error canary 断言 stdout/stderr 不泄漏，记录 Red。
- [ ] 3.5 先创建 `000001_create_schema_migrations.sql` 预期结构 tests 与 embed existence test；目标 SQL 只能创建精确 InnoDB/utf8mb4_0900_ai_ci history table，不含业务表。运行 test 并记录文件/包尚不存在的 Red，随后才允许写 production SQL。

## 4. Red: Real MySQL 8.0 W3 Matrix

- [ ] 4.1 先实现 integration test 的安全 harness 断言：只接受 `ORDER_TEST_MYSQL_INSTANCE=order-mysql-w3`、`ISOLATED=YES`、真实 `8.0.x`；只创建本次 `order_test_<128-bit-random-hex>` schema，所有退出路径精确 drop，名称/归属不匹配则 FAIL 且不删除。运行脚本，记录目标实现缺失的真实 Red；环境已建立，不能把 Red 写成 `BLOCKED_EXTERNAL`。
- [ ] 4.2 在真实 MySQL 8.0 先写首次/重复/并发/锁同连接场景：empty schema ready=uninitialized；首次产生唯一 clean v1；重复零写入；两 runner 只有一次 schema 副作用；probe 中 `IS_USED_LOCK` 等于 `CONNECTION_ID()`。运行并记录 Red。
- [ ] 4.3 在真实 MySQL 8.0 先写 failure/dirty/behind/too-new 场景：非法 v2 不产生 clean v2、dirty 保留且阻断 v3/重跑；clean strict prefix=behind 并可补齐；current+1=too-new 且零写入。运行并记录 Red。
- [ ] 4.4 在真实进程先写 application boundary 场景：空 schema 启动 `order-api` 后 live 200、ready uninitialized 且 table 仍不存在；使用已关闭 loopback listener 得到 unreachable 503；外部运行真实 `order-migrate` 后不重启 API，下一 ready=200。运行并记录 Red。

## 5. Green: Minimal Persistence Foundation

- [ ] 5.1 只在 `go.mod`/`go.sum` 加入 `github.com/go-sql-driver/mysql v1.10.0`，实现 `config` 的单一环境/结构化字段模型与 production fail-fast；使 2.1 Green，不增加 alias、raw DSN、SSM/provider、pool 配置或业务字段。
- [ ] 5.2 实现 `internal/database.ConnectionConfig` 与 `Open`，用 `mysql.Config`/`NewConnector`/`sql.OpenDB` 构造固定参数惰性 pool和脱敏分类；使 2.2 Green，不引入 ORM 或网络 startup check。
- [ ] 5.3 实现 `services/api/migrations/embed.go`、唯一基础 SQL 与 `migrate.Load`；使 3.1/3.5 Green，确认只存在 `schema_migrations`，文件 bytes/name/version/checksum 不可漂移。
- [ ] 5.4 实现 `migrate.Check` 的 MySQL 8.0/session/table shape/history state，只读映射七个 ready reason；使 3.2 Green，不创建/修改 schema。
- [ ] 5.5 实现 `migrate.Run` 的专用 connection、30 秒 `GET_LOCK`、首次表创建/核形、clean history 校验、version 2+ dirty→statement→clean 与零写入重复路径；使 3.3 Green，不提供 down/force/repair/seed。
- [ ] 5.6 实现零参数 `cmd/order-migrate` 的 0/1/2 退出与 JSON 摘要字段白名单；使 3.4 Green，底层 MySQL error/SQL/连接字段不进入日志。
- [ ] 5.7 只修改 `main.go`、health/router 三文件装配惰性 pool 与窄 `ReadinessFunc`；保持 `internal/app/**`、middleware、live、405/404 和 graceful shutdown 不变，使 2.3 Green，确认启动/health 无 migration。
- [ ] 5.8 完成 `mysql-integration.sh` 与安全 harness，使 4.1–4.4 在锁摘要真实 MySQL 8.0 全部 Green；完成每场景随机 schema cleanup，任何 cleanup failure 都保持 FAIL。
- [ ] 5.9 更新 `smoke.sh` 与根 README：准确写明结构化 dev/test 配置、production SSM 未实现因此不可启动、显式 `order-migrate`、live/ready 200/503、无业务表/API以及真实 W3 命令；使 2.4 Green，不写云 HA/备份/RPO 已证明。

## 6. Refactor and Recovery Review

- [ ] 6.1 `gofmt` 并审查职责：config 只解析、database 只 connector/pool、migrate 只文件/history/lock、cmd 只装配、HTTP 只映射；删除重复/死代码，确认未修改或间接注册 `internal/app/**`、middleware、业务模块或通用 provider。
- [ ] 6.2 重跑 2.1–3.5 的同一 focused tests 与无 DB smoke，记录 Refactor 后 Green；任何参数、reason、SQL bytes、公共 health body 或 owned path 变化先同步四类 artifacts并重新批准。
- [ ] 6.3 删除 writer 容器但保留专属 profile，从同一锁定 digest 建全新空容器，重跑 4.1–4.4 全部真实 MySQL 8.0 场景；确认首次/重复/并发/锁/失败/dirty/behind/unreachable/too-new/no-auto-migrate/ready 全部 PASS 且随机 schema 清理完成。
- [ ] 6.4 人工审查 forward fix/rollback：旧 binary 只在兼容当前 schema 时回退；dirty 自动阻断；人工修复必须隔离复现、恢复点、review 和单独写授权；CLI 无 down/force/repair。确认本 change 未把云备份/HA/RPO/RTO 当证据。

## 7. Writer Verification and Candidate

- [ ] 7.1 运行 `test -z "$(gofmt -l services/api)"`、`GOPROXY=off GOTOOLCHAIN=go1.26.5 go test ./services/api/...` 与 focused contract tests；全部使用当前 diff 的真实结果。
- [ ] 7.2 运行 `GOPROXY=off GOTOOLCHAIN=go1.26.5 go test -race ./services/api/...`、`go vet ./services/api/...`、`go build ./services/api/...`，并分别构建 `./services/api/cmd/order-api` 与 `./services/api/cmd/order-migrate`。
- [ ] 7.3 运行 `GOPROXY=off GOTOOLCHAIN=go1.26.5 bash services/api/scripts/smoke.sh` 与 `GOPROXY=off GOTOOLCHAIN=go1.26.5 bash services/api/scripts/mysql-integration.sh`；前者只证明进程/live/ready-503，后者必须记录 Colima v0.10.3、锁定 image digest、真实 MySQL 8.0 全矩阵和 cleanup PASS。
- [ ] 7.4 运行 `openspec validate establish-mysql-persistence-foundation --strict`、`openspec status --change establish-mysql-persistence-foundation --json`、`git diff --check 5ba5340cf9098724c0eb2284fdc5b14cb97be5dc...HEAD`；确认四类 artifacts 完整、implementation tasks 有真实证据且不存在未决行为。
- [ ] 7.5 运行 owned-path allowlist 和 protected-path zero-diff 检查：只允许 proposal 中列出的精确 path/prefix；显式确认 `internal/app/**`、middleware、product docs、腾讯云指南、quality/loop skills、`apps/**`、`AGENTS.md` 与 archived artifacts 无 diff。检查 `go list -m -f '{{if not .Indirect}}{{.Path}} {{.Version}}{{end}}' all` 只新增锁定 MySQL driver direct dependency。
- [ ] 7.6 对全部 owned source/artifacts/README/scripts 执行敏感扫描，禁止 DSN、password 值、Authorization/Cookie、私钥、证书、手机号、AppID/账号标识、原始 SQL/error body；测试 canary 必须证明 CLI/API/log 不泄漏。检查 migration 集合只含 history table，无业务表/repository/API/ORM/seed/down。
- [ ] 7.7 汇总真实 evidence 后才评定 candidate `C=10、T=10、V=8、R=8` 且硬阻断为零；只暂存 owned paths并提交一个中文完整 CANDIDATE，记录 full SHA、base、digest、命令结果，确认 index/worktree clean。不得推送、创建/更新 PR、部署或写生产/外部系统。

## 8. Exact-SHA Independent Verification and Local Runtime Cleanup

- [ ] 8.1 verifier 在另一个 clean detached worktree 检出 7.7 的完整 SHA，确认 exact base/candidate、owned/protected diff和开场 clean；删除 writer MySQL 容器但不得删除/修改其他 profile，再从同一锁定 digest 在 `order-mysql-w3` 建全新空容器，核对 loopback、architecture、digest 与 MySQL 8.0。
- [ ] 8.2 verifier 只读重跑 7.1–7.6 以及 4.1–4.4 全部真实场景，检查结束 worktree clean、随机 schemas 全部清理；任何 FAIL 返回原 writer，新 SHA 从头重验，不沿用旧证据。
- [ ] 8.3 exact-SHA independent PASS 后，负责闭环的 agent 先枚举并确认唯一目标是 container/profile `order-mysql-w3`，再删除该 container、profile 和 data disk；确认 profile 不存在且默认/其他 profile 未变化。删除失败立即停止并报告，不重试更强、更宽或嵌套 shell 删除。
- [ ] 8.4 只有最终 SHA、未失效 independent PASS、runtime cleanup 和全部依赖满足后才能进入 `INDEPENDENT_VERIFIED`；main/rebase/merge、Go/SQL/spec/tasks/README/command/digest 任一变化都产生新 candidate并重跑。`serve-persistent-menu-catalog` 只能在本 change `INTEGRATED` 后开始占用共享路径。
