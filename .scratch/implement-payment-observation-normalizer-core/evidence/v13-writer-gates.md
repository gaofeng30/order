# v13 writer runtime Gate receipt

**Base:** `5e937f3599a16f4813d6021f4cd2dd637c3156a2`

**Scope:** writer worktree。本文件只记录可在冻结前完成的 runtime/static 证据；exact candidate
diff、clean、review 与 detached verifier 在 commit 后外部绑定 SHA，不回填自引用。首次 review
真实 finding 修复后，表内所有 runtime Gate 已从头重跑；见 `review-fix.md`。

| Gate | Result | 决定性结果 |
| --- | --- | --- |
| fresh exact-base RED | PASS | 仅复制 seam test 后 `no non-test Go files`，exit 1；见 `v13-red.md` |
| focused | PASS | `go test ./services/api/internal/paymentobservation -count=1` |
| focused race | PASS | 同 package `go test -race` |
| mutation/resilience | PASS | canonical domain、local time、precedence、accepted/rejected、malformed、provider facts、collision 共 7/7 killed |
| fresh MySQL W3 | PASS | `mysql:8.0.46-oraclelinux9`；migrate/catalog/identity/menu/merchantidentity/storefront/wechatpay/paymentobservation 全部 `-race` PASS |
| full API tests | PASS | `go test ./services/api/... -count=1` |
| full API race | PASS | `go test -race ./services/api/... -count=1` |
| vet | PASS | `go vet ./services/api/...` |
| build | PASS | `go build ./services/api/...` |
| controlled smoke | PASS | `services/api/scripts/smoke.sh` 输出 `smoke: PASS` |
| format/shell/python syntax | PASS | Go 无 `gofmt -l` 输出；两个 shell 脚本 `bash -n`；change checker 可编译且未保留 pycache |
| minimal data surface | PASS | production 精确禁止字段扫描无命中；reflection test 固定 Observation 仅 9 个领域字段 |
| owned/protected pre-commit | PASS | dirty 仅两个 owned roots；`go.mod/go.sum`、migration/router/main/order/ingress/apps 无修改 |
| review-finding resilience | PASS | malformed transaction 优先于 callback/state 错误的组合测试先 RED 后 Green |

MySQL 证据是邻接真实数据库回归；`paymentobservation` 本身是无 I/O 纯逻辑模块，不把邻接回归
表述为本模块持久化证明。正式微信验签/回调、真实查单、资金、数据库持久化、订单 Apply 和 UI
均不在本 change 验证范围。
