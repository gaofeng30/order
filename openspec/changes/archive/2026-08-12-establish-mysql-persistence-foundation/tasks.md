> 状态：`CANDIDATE`。`gate_type=W3`、`ui_level_target=UI0`、`ui_level_actual=UI0`、`base_sha=5ba5340cf9098724c0eb2284fdc5b14cb97be5dc`、`approved_sha=c17ba4a5bfe7556b779fac093925df609358fe05`、`previous_candidate_sha=836c8a5c829ddd3ac394d682612f037dccb0383a（INVALIDATED）`、`replacement_candidate_sha=SELF（由本地 replacement commit 生成并在 handoff 绑定精确 SHA）`。本地 MySQL 环境为 `ESTABLISHED`、真实 W3 为 `PASS`；这不是 `BLOCKED_EXTERNAL`。writer 已完成最小修复与全部 writer Gate，等待新 exact-SHA independent verification。
>
> `approval_date=2026-08-13`；`approver=主 Agent`。批准依据是单能力 W3/UI0、canonical health 完整 MODIFIED、MySQL/真实 W3/production secret 边界唯一冻结、无行为未决、owned/依赖/非目标/42 tasks 完整且 strict PASS；这是主 Agent 在用户授予的自主裁决范围内作出的规划裁决，不表述为用户亲自确认。
>
> 每项完成后按 `docs/quality/change-quality-gates.md` 记录：change/gate/ui/base/candidate/phase、实际命令或操作、exit result、首个脱敏结果、artifact/environment、未验证边界及本地环境状态。candidate 自评 `C=10、T=10、V=8、R=8`、总分 36、硬阻断零；V=8 仅表示 exact-SHA verifier 资产完整，不表示 independent PASS。

## 1. Approval, Ownership and Local MySQL Prerequisite

- [x] 1.1 核验本 change 的 APPROVED 记录；重新完整读取 proposal、两份 spec、design、tasks、根 `AGENTS.md`、`docs/quality/change-quality-gates.md`、canonical `api-service-bootstrap` 与归档 bootstrap/production/product baselines，运行 `openspec validate establish-mysql-persistence-foundation --strict`，确认没有 Open Question 后才能进入 `IMPLEMENTING`。
  - Evidence: `change=establish-mysql-persistence-foundation; gate=W3/UI0; base=5ba5340cf9098724c0eb2284fdc5b14cb97be5dc; candidate=NOT_CREATED; phase=writer; action=openspec instructions apply + 完整读取上下文 + openspec validate --strict; exit=PASS; summary=APPROVED exact SHA c17ba4a...、4 artifacts 完整且 Open Questions=无; artifact=own change/canonical/三份 archived baselines; unverified=尚未实现或运行 W3; external=N/A`。
- [x] 1.2 在任何 Red 前确认 branch/worktree 仍是唯一 writer、HEAD 包含 exact base、index/worktree clean，且当前 diff 只含规划 commit；重新审计 main/router/config/README/smoke 的实际形态和全部 owned/protected paths，不吸收或回退其他 worker 改动。
  - Evidence: `change=establish-mysql-persistence-foundation; gate=W3/UI0; base=5ba5340cf9098724c0eb2284fdc5b14cb97be5dc; candidate=NOT_CREATED; phase=writer; action=git status/rev-parse/log + read current main/router/config/README/smoke/owned/protected; exit=PASS; summary=branch 唯一 writer、HEAD=c17ba4a...、main=exact base、开场 clean且无他人 diff; artifact=current worktree; unverified=尚未进入 Red; external=N/A`。
- [x] 1.3 只读运行 `brew info --json=v2 colima`，要求 stable=`0.10.3` 且 arm64 bottle SHA-256=`a9dfd1fa0a4aee62fef75974f39f174e4da774f7ba495c43dd0bcc23633381b8`；若 `colima` 不存在则按锁定 bottle 安装，存在则版本必须精确一致。检查 `order-mysql-w3` profile 不存在；若已存在，停止并先确认归属，不得删除或复用未知 profile。
  - Evidence: `change=establish-mysql-persistence-foundation; gate=W3/UI0; base=5ba5340cf9098724c0eb2284fdc5b14cb97be5dc; candidate=NOT_CREATED; phase=writer; action=brew info + brew install colima docker + version/path/profile absence checks; exit=PASS; summary=arm64、32 GiB free、Colima 0.10.3 bottle SHA a9df...、Docker CLI 29.7.2、Lima 2.2.0，安装于 /opt/homebrew 且同名 profile 不存在; artifact=local development prerequisite; unverified=MySQL image/profile 尚未建立; external=writer-managed local runtime`。
- [x] 1.4 创建唯一 `order-mysql-w3` profile，固定 Docker runtime、`aarch64`/VZ、2 CPU、4 GiB memory、10 GiB disk、Kubernetes off、workspace mount none、无外部 network address；记录 Colima/Docker engine 版本和 profile 配置的脱敏证据，环境状态从 `NOT_ESTABLISHED` 变为 `ESTABLISHED`。
  - Evidence: `change=establish-mysql-persistence-foundation; gate=W3/UI0; base=5ba5340cf9098724c0eb2284fdc5b14cb97be5dc; candidate=NOT_CREATED; phase=writer; action=colima start/status/list + Lima guest mount核验 + docker engine info; exit=PASS; summary=首次 native downloader redirect EOF，经 --downloader curl 单次恢复；VZ/aarch64、2 CPU、4 GiB、10 GiB、Docker engine 29.5.2、K8s/autoActivate/networkAddress=false，guest 无 /Users 宿主挂载; artifact=order-mysql-w3 profile; unverified=真实 W3 尚未运行; external=writer-managed local runtime`。
- [x] 1.5 在 0600 `mktemp -d` 中生成一次性 MySQL 凭据；实际 pull/run 前从 Docker Official Registry 枚举精确 8.0.x tags并选择最新 patch。2026-08-13 冻结 `mysql:8.0.46-oraclelinux9`、manifest list `sha256:7dcddc01f13bab2f15cde676d44d01f61fc9f99fe7785e86196dfc07d358ae2b`、`linux/arm64/v8` platform `sha256:213bbfaf699693a40a20a12bb4342d2589a15a3dc7153db698eaed252a92458e`；仅在 profile 内按 platform digest pull/run，容器名 `order-mysql-w3`、只映射 `127.0.0.1` 随机端口、不挂宿主目录/复用 volume。核对 repo digest、architecture、container health 与 `SELECT VERSION()`=`8.0.46`；不得输出 secret/DSN。任一步失败记 FAIL，由 writer 从首错继续，未成功不得进入 Red。
  - Evidence: `change=establish-mysql-persistence-foundation; gate=W3/UI0; base=5ba5340cf9098724c0eb2284fdc5b14cb97be5dc; candidate=NOT_CREATED; phase=writer; action=Official Registry tags/manifest GET+HEAD、digest pull、exact-label container inspect、SELECT VERSION/global session; exit=PASS; summary=最新官方 8.0.46，manifest list 7dcddc...8ae2b，linux/arm64/v8 213bbf...2458e；唯一容器 healthy、loopback random port、tmpfs noexec/nosuid 1 GiB、无 bind/volume、version 8.0.46、UTC/utf8mb4 server，凭据文件 0600未输出; artifact=order-mysql-w3 container; unverified=Go connector与W3矩阵尚未运行; external=writer-managed real MySQL 8.0`。

## 2. Red: Configuration, Pool and HTTP Contract

- [x] 2.1 先扩展 `internal/config` tests，覆盖 `ORDER_ENV` 枚举、dev/test 六个结构化字段、port/TLS 校验、全部模式拒绝 `ORDER_DB_DSN`、production 拒绝 env password 并固定 `production_secret_source_unavailable`；用 canary secret 断言 error/log 不回显。运行 focused test，记录因目标实现缺失产生的 Red。
  - Evidence: `go test ./services/api/internal/config -run 'TestLoad' -count=1` exit 1；首错为 `Config` 缺少 `Environment/Database` 及 `Reason`，canary 未输出。
- [x] 2.2 先新增 `internal/database` tests，冻结 `mysql.Config` 的 TCP/UTC/parseTime/utf8mb4/collation/TLS/dial-read-write timeout/unsafe flags、NopLogger 和 pool 10/10/3m/1m；断言 constructor 惰性不 ping、结构化错误不泄密。运行 focused test，记录 Red。
  - Evidence: focused test exit 1；首错为锁定的 `github.com/go-sql-driver/mysql` 与目标 package 尚不存在，未发生网络访问。
- [x] 2.3 先修改 router tests，覆盖 live 不调用 DB、ready current=200，以及 unreachable/incompatible/uninitialized/dirty/behind/too-new/checksum=503 的精确 JSON；保留 405/404、middleware 与无敏感正文断言。运行 `go test ./services/api/internal/httpapi -run 'TestHealth' -count=1`，记录 Red。
  - Evidence: focused test exit 1；首错为旧 `NewRouter` 只有 logger 参数且 `ReadinessResult` 不存在。
- [x] 2.4 先把无数据库 `smoke.sh` 的目标改为结构化 development 配置、live 200、ready 503/database_unreachable、404、SIGTERM clean exit 和 invalid config exit 1；运行并记录旧实现把 ready 返回 200 或不识别 DB 配置的 Red。
  - Evidence: `bash services/api/scripts/smoke.sh` exit 1；旧 API 的 ready 仍为进程级 200，未满足目标 503 reason。

## 3. Red: Migration Files, Runner and CLI

- [x] 3.1 先新增 `migrate.Load` tests，覆盖六位连续编号、缺号/重复、UTF-8/LF/newline、单终止分号、单 statement、禁止 down/seed/DELIMITER/SOURCE/LOAD DATA，以及原始 bytes SHA-256；使用 `fstest.MapFS` 记录因 loader 缺失产生的 Red。
  - Evidence: migrate focused test exit 1；首错为 `Load/Migration` 未定义。
- [x] 3.2 先新增 history/state tests，覆盖 table absent、version 1 精确表形、clean prefix、current、dirty、too-new、缺失历史、name/checksum drift、MySQL 非 8.0/session 非 UTC或 utf8mb4；fake 只验证内部分类，不记 W3 PASS。
  - Evidence: migrate focused test exit 1；首错包含 `historyRow/classifyHistory/compatibleSession` 未定义；fake 证据未计 W3。
- [x] 3.3 先新增 runner orchestration tests，断言 `GET_LOCK`、全部 schema/history/SQL 与 `RELEASE_LOCK` 只走同一专用 connection；覆盖 lock=0/NULL/error、release/close、首次、重复、statement failure 留 dirty 且无 clean/后续、再次运行先拒绝 dirty。运行 focused tests 记录 Red。
  - Evidence: migrate focused test exit 1；`runLocked` 尚不存在；测试已冻结首次、重复、lock 三失败、dirty 停止及 history-table-create crash 重跑顺序。
- [x] 3.4 先新增 `order-migrate` tests，覆盖零参数 exit 0、所有运行错误 exit 1、任意参数 exit 2 且不连接，以及成功/失败 JSON 字段白名单；用 DSN/password/SQL/server-error canary 断言 stdout/stderr 不泄漏，记录 Red。
  - Evidence: CLI focused test exit 1；目标 migrate package/execute 尚不存在。
  - Verifier Red: exact candidate `836c8a5c829ddd3ac394d682612f037dccb0383a` 已失效；`MIGRATION_SUCCESS_EVENT_CONTRACT_MISMATCH #1`。先把 success test 改为 stdout 必须为空、stderr 恰有一条 JSON 且 `event=migration_complete`，再在未改实现上运行 `GOPROXY=off GOTOOLCHAIN=go1.26.5 go test ./services/api/cmd/order-migrate -run '^TestExecuteSuccessAndCurrentExitZero$' -count=1`，exit 1；首错为 stdout 实际含 `event=migration_completed` 的单条成功摘要。
  - Writer Green/Refactor: 仅把成功 logger 改为 stderr 且 `event=migration_complete`；同一 focused test 与 CLI 全包 test 均 exit 0，stdout empty、stderr 单条 JSON、失败日志与 0/1/2 exit 语义不变；生产/测试断言中的旧串为 0。
- [x] 3.5 先创建 `000001_create_schema_migrations.sql` 预期结构 tests 与 embed existence test；目标 SQL 只能创建精确 InnoDB/utf8mb4_0900_ai_ci history table，不含业务表。运行 test 并记录文件/包尚不存在的 Red，随后才允许写 production SQL。
  - Evidence: migrations focused test exit 1；`FS` 未定义且 SQL 不存在；测试明确要求 `CREATE TABLE IF NOT EXISTS` 的崩溃窗口可恢复语义。

## 4. Red: Real MySQL 8.0 W3 Matrix

- [x] 4.1 先实现 integration test 的安全 harness 断言：只接受 `ORDER_TEST_MYSQL_INSTANCE=order-mysql-w3`、`ISOLATED=YES`、真实 `8.0.x`；只创建本次 `order_test_<128-bit-random-hex>` schema，所有退出路径精确 drop，名称/归属不匹配则 FAIL 且不删除。运行脚本，记录目标实现缺失的真实 Red；环境已建立，不能把 Red 写成 `BLOCKED_EXTERNAL`。
  - Evidence: 在 healthy `order-mysql-w3` 上运行脚本 exit 1；首错为锁定 MySQL driver/目标实现缺失，不是环境阻塞；harness 已冻结完整 env、8.0、128-bit schema 与精确 cleanup。
- [x] 4.2 在真实 MySQL 8.0 先写首次/重复/并发/锁同连接场景：empty schema ready=uninitialized；首次产生唯一 clean v1；重复零写入；两 runner 只有一次 schema 副作用；probe 中 `IS_USED_LOCK` 等于 `CONNECTION_ID()`。运行并记录 Red。
  - Evidence: 同一次真实脚本 Red；测试场景已先于实现落盘，并含建表后未记 v1 的重跑恢复。
- [x] 4.3 在真实 MySQL 8.0 先写 failure/dirty/behind/too-new 场景：非法 v2 不产生 clean v2、dirty 保留且阻断 v3/重跑；clean strict prefix=behind 并可补齐；current+1=too-new 且零写入。运行并记录 Red。
  - Evidence: 同一次真实脚本 Red；failure/dirty/behind/too-new 的写入与零写入断言已落盘。
- [x] 4.4 在真实进程先写 application boundary 场景：空 schema 启动 `order-api` 后 live 200、ready uninitialized 且 table 仍不存在；使用已关闭 loopback listener 得到 unreachable 503；外部运行真实 `order-migrate` 后不重启 API，下一 ready=200。运行并记录 Red。
  - Evidence: 同一次真实脚本 Red；test 已冻结真实 binary build/start、空 schema 无自动迁移、外部 CLI 后无需重启恢复 ready，以及 closed-listener unreachable。

## 5. Green: Minimal Persistence Foundation

- [x] 5.1 只在 `go.mod`/`go.sum` 加入 `github.com/go-sql-driver/mysql v1.10.0`，实现 `config` 的单一环境/结构化字段模型与 production fail-fast；使 2.1 Green，不增加 alias、raw DSN、SSM/provider、pool 配置或业务字段。
  - Evidence: config focused tests PASS；direct driver=`v1.10.0`，production 固定拒绝 env secret/DSN 并返回 SSM source unavailable。
- [x] 5.2 实现 `internal/database.ConnectionConfig` 与 `Open`，用 `mysql.Config`/`NewConnector`/`sql.OpenDB` 构造固定参数惰性 pool和脱敏分类；使 2.2 Green，不引入 ORM 或网络 startup check。
  - Evidence: database focused tests PASS；首轮真实连接暴露 `Params[charset]` 的 MySQL 1193，最小改为 v1.10 官方 `mysql.Charset` option 后 session/server utf8mb4 真机断言 PASS。
- [x] 5.3 实现 `services/api/migrations/embed.go`、唯一基础 SQL 与 `migrate.Load`；使 3.1/3.5 Green，确认只存在 `schema_migrations`，文件 bytes/name/version/checksum 不可漂移。
  - Evidence: loader/embed focused tests PASS；migration 集合只有 `000001_create_schema_migrations.sql`，原始 bytes SHA-256 且 `CREATE TABLE IF NOT EXISTS`。
- [x] 5.4 实现 `migrate.Check` 的 MySQL 8.0/session/table shape/history state，只读映射七个 ready reason；使 3.2 Green，不创建/修改 schema。
  - Evidence: state/health focused tests与真实 unreachable/incompatible/uninitialized/dirty/behind/too-new/checksum matrix PASS。
- [x] 5.5 实现 `migrate.Run` 的专用 connection、30 秒 `GET_LOCK`、首次表创建/核形、clean history 校验、version 2+ dirty→statement→clean 与零写入重复路径；使 3.3 Green，不提供 down/force/repair/seed。
  - Evidence: runner focused tests PASS；真实并发 probe 的 `IS_USED_LOCK=CONNECTION_ID`，建表后未记 v1 的重跑恢复、dirty 阻断及 pool InUse=0 PASS。
- [x] 5.6 实现零参数 `cmd/order-migrate` 的 0/1/2 退出与 JSON 摘要字段白名单；使 3.4 Green，底层 MySQL error/SQL/连接字段不进入日志。
  - Evidence: CLI focused tests PASS；成功/失败字段白名单、usage 不连接与 canary 不泄漏均 PASS。
- [x] 5.7 只修改 `main.go`、health/router 三文件装配惰性 pool 与窄 `ReadinessFunc`；保持 `internal/app/**`、middleware、live、405/404 和 graceful shutdown 不变，使 2.3 Green，确认启动/health 无 migration。
  - Evidence: HTTP/app tests与真实 binary boundary PASS；空 schema 启动后 table 仍不存在，外部 CLI 后无需重启 ready=200。
- [x] 5.8 完成 `mysql-integration.sh` 与安全 harness，使 4.1–4.4 在锁摘要真实 MySQL 8.0 全部 Green；完成每场景随机 schema cleanup，任何 cleanup failure 都保持 FAIL。
  - Evidence: 锁摘要 MySQL `8.0.46` real `go test -race` W3 PASS；首次/重复/崩溃恢复/并发锁/失败dirty/behind/too-new/checksum/unreachable/incompatible/真实进程全部 PASS，残留随机 schema=0。
- [x] 5.9 更新 `smoke.sh` 与根 README：准确写明结构化 dev/test 配置、production SSM 未实现因此不可启动、显式 `order-migrate`、live/ready 200/503、无业务表/API以及真实 W3 命令；使 2.4 Green，不写云 HA/备份/RPO 已证明。
  - Evidence: no-DB smoke PASS；README 明确 production 不可启动、显式 migrate、health 契约和无业务表/API，不声称云 HA/备份/RPO。

## 6. Refactor and Recovery Review

- [x] 6.1 `gofmt` 并审查职责：config 只解析、database 只 connector/pool、migrate 只文件/history/lock、cmd 只装配、HTTP 只映射；删除重复/死代码，确认未修改或间接注册 `internal/app/**`、middleware、业务模块或通用 provider。
  - Evidence: `gofmt`/diff/职责审查 PASS；protected app/middleware 零 diff，无 ORM/provider/业务注册。
- [x] 6.2 重跑 2.1–3.5 的同一 focused tests 与无 DB smoke，记录 Refactor 后 Green；任何参数、reason、SQL bytes、公共 health body 或 owned path 变化先同步四类 artifacts并重新批准。
  - Evidence: config/database/httpapi/migrate/migrations/CLI focused tests及 `smoke.sh` 同矩阵全部 PASS；driver charset 事实已同步 design/spec，公共行为未变。
- [x] 6.3 删除 writer 容器但保留专属 profile，从同一锁定 digest 建全新空容器，重跑 4.1–4.4 全部真实 MySQL 8.0 场景；确认首次/重复/并发/锁/失败/dirty/behind/unreachable/too-new/no-auto-migrate/ready 全部 PASS 且随机 schema 清理完成。
  - Evidence: 只读确认 exact name+双 label+tmpfs/no bind 后 stop/remove 唯一 writer container；同一 arm64 digest重建 fresh healthy container，真实全矩阵 PASS，MySQL=8.0.46、残留 schema=0，profile 保留。
- [x] 6.4 人工审查 forward fix/rollback：旧 binary 只在兼容当前 schema 时回退；dirty 自动阻断；人工修复必须隔离复现、恢复点、review 和单独写授权；CLI 无 down/force/repair。确认本 change 未把云备份/HA/RPO/RTO 当证据。
  - Evidence: 人工审查 PASS；CLI 仅零参数 forward，dirty 自动阻断，无 down/force/repair；云 HA/备份/RPO/RTO 均未计入本 change 证据。

## 7. Writer Verification and Candidate

- [x] 7.1 运行 `test -z "$(gofmt -l services/api)"`、`GOPROXY=off GOTOOLCHAIN=go1.26.5 go test ./services/api/...` 与 focused contract tests；全部使用当前 diff 的真实结果。
  - Evidence: `gofmt -l` empty；全 API tests 与 config/database/httpapi/migrate/migrations/CLI focused contract tests均 exit 0。
- [x] 7.2 运行 `GOPROXY=off GOTOOLCHAIN=go1.26.5 go test -race ./services/api/...`、`go vet ./services/api/...`、`go build ./services/api/...`，并分别构建 `./services/api/cmd/order-api` 与 `./services/api/cmd/order-migrate`。
  - Evidence: race/vet/all build/两个 exact command build 全部 exit 0；GOPROXY=off、GOTOOLCHAIN=go1.26.5。
- [x] 7.3 运行 `GOPROXY=off GOTOOLCHAIN=go1.26.5 bash services/api/scripts/smoke.sh` 与 `GOPROXY=off GOTOOLCHAIN=go1.26.5 bash services/api/scripts/mysql-integration.sh`；前者只证明进程/live/ready-503，后者必须记录 Colima v0.10.3、锁定 image digest、真实 MySQL 8.0 全矩阵和 cleanup PASS。
  - Evidence: no-DB smoke PASS；Colima 0.10.3 + fresh MySQL 8.0.46 container、manifest `7dcddc...8ae2b`/arm64 platform `213bbf...2458e` 上 real race W3 全矩阵 PASS，随机 schema 残留=0。
- [x] 7.4 运行 `openspec validate establish-mysql-persistence-foundation --strict`、`openspec status --change establish-mysql-persistence-foundation --json`、`git diff --check 5ba5340cf9098724c0eb2284fdc5b14cb97be5dc...HEAD`；确认四类 artifacts 完整、implementation tasks 有真实证据且不存在未决行为。
  - Evidence: strict PASS；status JSON 的 proposal/design/specs/tasks 均 done；pre-commit full diff check PASS，candidate commit 后同 base range check在 handoff重新绑定；Open Questions=无。
- [x] 7.5 运行 owned-path allowlist 和 protected-path zero-diff 检查：只允许 proposal 中列出的精确 path/prefix；显式确认 `internal/app/**`、middleware、product docs、腾讯云指南、quality/loop skills、`apps/**`、`AGENTS.md` 与 archived artifacts 无 diff。检查 `go list -m -f '{{if not .Indirect}}{{.Path}} {{.Version}}{{end}}' all` 只新增锁定 MySQL driver direct dependency。
  - Evidence: tracked+untracked owned allowlist PASS（31 files），全部命名 protected scopes zero diff；direct modules 仅 existing Gin 与新增 `github.com/go-sql-driver/mysql v1.10.0`。
- [x] 7.6 对全部 owned source/artifacts/README/scripts 执行敏感扫描，禁止 DSN、password 值、Authorization/Cookie、私钥、证书、手机号、AppID/账号标识、原始 SQL/error body；测试 canary 必须证明 CLI/API/log 不泄漏。检查 migration 集合只含 history table，无业务表/repository/API/ORM/seed/down。
  - Evidence: static sensitive/logging scan PASS，真实临时凭据未进入 repo；config/CLI/HTTP/smoke canary tests PASS；SQL 集合仅 `schema_migrations`，无业务表/ORM/seed/down/force/repair。
- [x] 7.7 汇总真实 evidence 后才评定 candidate `C=10、T=10、V=8、R=8` 且硬阻断为零；只暂存 owned paths并提交一个中文完整 CANDIDATE，记录 full SHA、base、digest、命令结果，确认 index/worktree clean。不得推送、创建/更新 PR、部署或写生产/外部系统。
  - Evidence: `change=establish-mysql-persistence-foundation; gate=W3/UI0; base=5ba5340cf9098724c0eb2284fdc5b14cb97be5dc; candidate=SELF; phase=writer; exit=PASS; C/T/V/R=10/10/8/8 total=36; hard_blockers=0; artifact=owned paths only; unverified=exact-SHA independent verification/integration/archive 仍待 8.1-8.4; external=production SSM/CAM 独立 change`。本地中文 candidate commit与 exact SHA 由提交本身及 handoff绑定；无 push/PR/deploy/external write。
  - Verifier recovery: 旧 SHA FAIL 后只修改 CLI implementation/test 与本 tasks 证据；focused、Go fmt/test/race/vet/build、no-DB smoke、fresh digest MySQL 8.0.46 W3、strict/status、owned/protected/sensitive、JS/42 JSON 全部 PASS，随机 schema 残留=0；`C/T/V/R=10/10/8/8`、hard blockers=0，8.1-8.4 仍未勾，新 exact SHA 由 replacement commit/handoff 绑定。

## 8. Exact-SHA Independent Verification and Local Runtime Cleanup

- [ ] 8.1 verifier 在另一个 clean detached worktree 检出 7.7 的完整 SHA，确认 exact base/candidate、owned/protected diff和开场 clean；删除 writer MySQL 容器但不得删除/修改其他 profile，再从同一锁定 digest 在 `order-mysql-w3` 建全新空容器，核对 loopback、architecture、digest 与 MySQL 8.0。
- [ ] 8.2 verifier 只读重跑 7.1–7.6 以及 4.1–4.4 全部真实场景，检查结束 worktree clean、随机 schemas 全部清理；任何 FAIL 返回原 writer，新 SHA 从头重验，不沿用旧证据。
- [ ] 8.3 exact-SHA independent PASS 后，负责闭环的 agent 先枚举并确认唯一目标是 container/profile `order-mysql-w3`，再删除该 container、profile 和 data disk；确认 profile 不存在且默认/其他 profile 未变化。删除失败立即停止并报告，不重试更强、更宽或嵌套 shell 删除。
- [ ] 8.4 只有最终 SHA、未失效 independent PASS、runtime cleanup 和全部依赖满足后才能进入 `INDEPENDENT_VERIFIED`；main/rebase/merge、Go/SQL/spec/tasks/README/command/digest 任一变化都产生新 candidate并重跑。`serve-persistent-menu-catalog` 只能在本 change `INTEGRATED` 后开始占用共享路径。
