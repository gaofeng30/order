# implement-merchant-identity-rbac-core

## 状态与唯一结果

- 状态：`IMPLEMENTING / REVIEWING_WIP`；旧 SHA `dbdb508194a7e5524c3f5abdb98c60e5fb0b9878`、`9cd2ffa330e40af05e91e5424b9ad252327b15f1` 与 `e89a7561382b486c4958cceb455eb698736d129c` 已失效，不得进入 verifier。
- tracker：`BLOCKED_LOCAL_GOVERNANCE`。仓库缺少 `docs/agents/issue-tracker.md`；本 change 不创建/更新 GitHub Issue，不伪造 `READY`。
- 唯一业务结果：实现 merchant identity 核心 v12/v13、`GET /api/v1/me/identity`、`POST /api/v1/me/merchant-login` 与 transaction-bound `AuthorizeInTx`；现有三条 Mini Program identity/phone API 保持不变。
- 最小成功标准：严格 HTTP、schema、实时 RBAC、首次绑定/审计原子性及故障恢复在 fresh MySQL 8.0.46 闭环；Writer 全 Gate 与 fixed-base 双轴 Review 通过后形成 Candidate，等待主控独立审计批准。

## Gate 与固定基线

- `gate_type`: `W3`。
- `ui_level_target`: `UI0`。
- `ui_level_actual`: `UI0`（纯后端；不宣称浏览器、小程序、真机、UAT、生产或部署运行）。
- owner: `implement-merchant-identity-rbac-core` independent Writer Session。
- worktree: `/Users/vivix/.codex/worktrees/aa63/order`。
- branch: `codex/implement-merchant-identity-rbac-core`。
- `base_sha`: `122913c6bcc8c22acb73e05385d54449f27c2465`。
- base branch: `codex/order-delivery-integration`；开工时已核验精确 SHA、staging worktree clean。
- dependency: fixed base 已含 independently verified storefront v11，migrations 当前精确止于 v11；transaction migrations 从 v14 起。
- canonical product authority: `docs/product/online-ordering-system-prd-0818.md` §4.2–§4.4、§5.9、§6.1、§6.5–§6.10、§12、§15.5、§15.6。
- read-only contract reference: exact `8567157693aeb6cf09ceec95dfaaee2a090f3780` 的 `.scratch/freeze-merchant-miniprogram-auth-rbac-contract/{spec.md,tickets.md}`，只取 T1 与共享授权契约，不复用其 WIP 状态或历史证据。
- `candidate_sha`: `not-yet-created`。当前 fixed review target 由主控在线程外绑定到每轮 clean HEAD；双审与主控复审 PASS 后，主控直接将当时 clean HEAD 指定为 Candidate，不再修改本工件，避免 commit 内容自引用自身 SHA。

## Owned / read-only / non-goals

唯一 owned paths：

- `.scratch/implement-merchant-identity-rbac-core/**`
- `services/api/internal/merchantidentity/**`
- `services/api/migrations/000012_create_merchant_accounts.sql`
- `services/api/migrations/000013_create_merchant_action_audits.sql`
- `services/api/migrations/embed_test.go`，仅追加 v12/v13
- `services/api/internal/catalog/migrations_test.go`，仅 exact list 追加 v12/v13
- `services/api/internal/httpapi/router.go`、`router_test.go`，仅两条 T1 route/wiring
- `services/api/cmd/order-api/main.go`、`main_test.go`，仅 merchantidentity dependency wiring

只读：`services/api/internal/identity/**`、`services/api/internal/wechat/**`、`storefront/**`、`wechatpay/**`、`apps/**`、package files、其他 migrations、PRD/quality/old scratch。若现有 identity/wechat 编译 seam 必须变化，停止并回报精确文件与理由，不自行扩权。

shared ownership：当前包含 fixed base 的分支仅 staging 与本 Writer；staging 相对 base 对 router/main 无 delta。冻结 DAG 为 `T1 -> T4`，T1 完成并转交前 T4 不得并行写 shared files。

非目标：T2 Mini Program 页面/guard/UI3；T3 订单/核销/售罄/营业状态 consumer；T4 account CRUD/最后主账号/PC 页面与 PC auth；解绑、logout、session TTL、多门店、密码、邀请码、未来角色抽象；integration、push、PR、deploy、微信/腾讯云/生产访问或写入。

## 已确认 TDD seams

1. HTTP seam：root router 上只观察两个新 versioned endpoint 的 strict Bearer/body/content-type、exact DTO/error、`Cache-Control: no-store` 和 PII fail-closed。
2. Migration seam：embedded exact v1-v13 chain 与 fresh MySQL 8.0.46 首次/重复 application、DDL/constraint/FK/unique/delete retention。
3. Merchant identity transaction seam：真实 MySQL 上通过 public service/repository API 观察首次 login、幂等、审计、rollback/deadlock/commit-unknown 与同-code concurrency。
4. Authorization seam：caller 创建 transaction，调用 `AuthorizeInTx(ctx, tx, userID, action, target)` 后在同 tx 读写；观察 shared lock、实时 enabled/role/binding/version 与单一提交顺序。
5. PII seam：响应、结构化日志、耐久审计和任务证据扫描不得含手机号、姓名、openid、code/token/provider body 或认证材料；fixtures 仅用合成值，普通证据只留枚举结果。

## HTTP 冻结

- 只新增 `GET /api/v1/me/identity` 与 `POST /api/v1/me/merchant-login`；成功和错误均 `Cache-Control: no-store`，只接受一个严格 `Authorization: Bearer <opaque>`。
- identity body 必须空；成功 exact `user.primary_phone_bound` 与 `merchant:null` 或 exact `{role,auth_version}`，不返回内部 ID、openid 或手机号。
- merchant-login 只接受 `application/json`，body `<=1KiB`，exact `{code}`；code 非空、不 trim、`<=256` bytes；未知/重复/尾随/错误类型均拒绝。
- 幂等已有启用 binding 不调用 provider；首次未绑定 provider 最多一次且不重试。
- stable errors：400 `INVALID_REQUEST`；401 `UNAUTHENTICATED`；403 `MERCHANT_ACCOUNT_NOT_AVAILABLE`；403 `FORBIDDEN`；409 `PHONE_IN_USE`；409 `PRIMARY_PHONE_MISMATCH`；422 `PHONE_CODE_REJECTED`；503 `MERCHANT_IDENTITY_UNAVAILABLE`。
- 不泄漏名单差异、手机号、姓名、openid、code/token/provider body；基础格式或无 session 不写耐久审计，数据库/审计不可用不伪造成功。

## Schema / transaction / PII 冻结

- v12/v13 每文件恰好一个 forward-only `CREATE TABLE`。
- `merchant_accounts`：内部 id；规范 phone 非空字节唯一；name 非空；role 仅 `OWNER|SUBACCOUNT`；enabled default true；positive record/auth version；`bound_user_id` nullable、全表唯一且 FK `miniprogram_users`；`bound_at` 成对；UTC microsecond timestamps；created_by/updated_by 可追溯内部主账号。与员工折扣名单完全分离。
- `merchant_action_audits`：账号 FK nullable、`ON UPDATE RESTRICT ON DELETE SET NULL`；account/role/auth snapshot 全空或全非空；硬删后 snapshot/action/result/reason/target/internal actor/time 保留且不 cascade；不复制 phone/name/openid/code/token。
- actor 仅 `merchant_owner|merchant_subaccount`；持久角色仅 `OWNER|SUBACCOUNT`。action 仅六个冻结值；主/子允许前五个，只有 OWNER 允许 `merchant.account.manage`。
- `AuthorizeInTx` 只用 caller tx，以 `bound_user_id SELECT ... FOR SHARE` 读取当前账号，实时校验 enabled/role/action/binding；只返回内部 account ID、actor、record_version、auth_version；不 begin/commit/rollback。
- 账号启停/角色/删除与业务授权争用同一当前行，形成单一提交顺序。fixture 必须证明停用立即拒绝、重启恢复 binding、角色下一请求生效、删除失效。
- 首次 merchant-login 的主手机号绑定、账号 binding+versions、成功审计单事务单 commit；未命中/停用/他绑不写主手机号/binding。
- 同 code 并发只允许在受限重读证明同一 user/account/version 已完成绑定时把 provider reject 收敛为同一成功，否则 422；不 retry provider。

## Red → Green → Refactor

- Red：先增加一个 vertical tracer，再运行对应 focused command；必须因固定基线缺少两路由、v12/v13、首次 binding/audit 或实时 authorizer 真实 FAIL，记录首个脱敏决定性错误。
- Green：每次只加入使当前 tracer 通过的最小实现；同一命令 PASS 后再进入下一 slice。不得先批量写完测试，也不得复用冻结 WIP 的 PASS。
- Refactor：全部 slices Green 后才收敛命名/重复；重跑同一 strict HTTP、migration、MySQL fault/concurrency、AuthorizeInTx、PII、race 与回归集合。

## Writer / reviewer / verifier / integration 命令

- focused：`GOPROXY=off GOTOOLCHAIN=go1.26.5 go test ./services/api/internal/merchantidentity ./services/api/internal/httpapi ./services/api/cmd/order-api ./services/api/migrations ./services/api/internal/catalog -count=1`。
- focused race：同一 package 集合加 `-race`。
- fresh MySQL：`.scratch/implement-merchant-identity-rbac-core/verify-mysql.sh` 启动 loopback-only `mysql:8.0.46-oraclelinux9`，等待 TCP ready，注入仓库七个 `ORDER_TEST_MYSQL_*` 变量，运行 merchantidentity W3 suite 后清理 container/credential；禁止 sqlmock/skip 冒充。
- full：`GOPROXY=off GOTOOLCHAIN=go1.26.5 go test ./services/api/... -count=1`；同路径 `go test -race`、`go vet`、`go build`。
- smoke：`GOPROXY=off GOTOOLCHAIN=go1.26.5 bash services/api/scripts/smoke.sh`。
- format/diff：`test -z "$(gofmt -l <owned-go-paths>)"`；`git diff --check 122913c6bcc8c22acb73e05385d54449f27c2465...HEAD`；exact owned-path allowlist；worktree/index clean。
- Review：中文功能提交后固定 `git diff 122913c6bcc8c22acb73e05385d54449f27c2465...HEAD` 与 `git log 122913c6bcc8c22acb73e05385d54449f27c2465..HEAD --oneline`，按 `$code-review` 并行 Standards/Spec；finding 修复产生 replacement SHA，并从头 Gate/Review。
- Verifier：主控独立审计批准后，fresh clean detached worktree 对 exact candidate SHA 只读重跑全部 focused/race/fresh-MySQL/full/vet/build/smoke/format/diff/owned/PII/clean Gate。
- Integration：本 Writer 不执行；不集成 staging/main，不 push/PR/deploy。
- OpenSpec：`N/A`；`openspec/**` 为只读历史且不在 owned paths，不伪造 validate PASS。

## 外部资产与恢复

| 资产 | owner | 当前状态 | 恢复条件 |
| --- | --- | --- | --- |
| fresh Docker MySQL 8.0.46 | Writer/Verifier | `WRITER_PASS`；最终 fresh run 已从 v1→v13 通过并清理 | 每个 replacement candidate 与获批 verifier 再启动全新 loopback container 重跑并清理 |
| Matt tracker | workflow owner / 用户 | `BLOCKED_LOCAL_GOVERNANCE` | 用户确认并配置 tracker 后另行关联，不追溯伪造 READY |
| 微信 AppID/手机号能力/UAT 账号/生产 CDB | 客户/平台/UAT owner | `BLOCKED_EXTERNAL` 且非 T1 Candidate Gate | 由 T2/T5 或部署 change 在单独授权、脱敏环境中验证 |

rollback：Candidate 未集成前删除本 branch/worktree即可；migration forward-only，无 down。集成后的部署/schema 恢复只由获授权 integration/runbook 处理，本 Writer 不操作外部数据库。
