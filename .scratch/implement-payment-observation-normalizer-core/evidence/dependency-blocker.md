# Dependency blocker: real MySQL suites lag exact-base migration v13

**Status:** `HISTORICAL_FAIL / RESOLVED_UPSTREAM_IN_5e937f3599a16f4813d6021f4cd2dd637c3156a2`

本文件保留旧 exact base 的真实 failure，不改写成 PASS。主控已把下述 foundation change 集成到
新 staging SHA；当前 writer 仍需独立重跑，新 candidate 仍需 fresh detached 再跑。

## 决定性运行事实

- exact base / current HEAD lineage：`b7f484f54decfa38bc36bbed1ada041828d87483`
- 本 change 对 `services/api/migrations/**` 和下列 protected tests 的 diff：空。
- fresh loopback-only `mysql:8.0.46-oraclelinux9` 已建立并通过 TCP readiness；失败后容器与
  0600 临时凭据均清理。
- 新模块与 `wechatpay` 在同次 `-race` 命令通过；`merchantidentity` 当前 v1-v13 suite 通过。
- 完整矩阵退出 1；首个失败为 `TestMySQL8Integration: embedded migration count = 13, want 10`。

## 全部已确认的受影响 protected 断言

| 文件 | 固定旧版本断言 |
| --- | --- |
| `services/api/internal/migrate/mysql_integration_test.go` | `wantNames` 只列 v1-v10，随后要求 migration count 等于 10 |
| `services/api/internal/catalog/mysql_integration_test.go` | count=10；v1→v10 applied=9；repeat from/to=10 |
| `services/api/internal/identity/mysql_integration_test.go` | session count=10、v1→v10、repeat v10；phone 全集 count=10、v9→v10 applied=1；status 全集 to=10/applied=10 |
| `services/api/internal/menu/mysql_integration_test.go` | count=10；v3→v10 applied=7；repeat v10 |
| `services/api/internal/storefront/mysql_integration_test.go` | count=11；apply/repeat v11；测试名和错误文案也固定 v1-v11 |

仓库既有 `services/api/scripts/mysql-integration.sh`、`catalog-integration.sh`、
`menu-integration.sh`、`miniprogram-session-integration.sh` 与
`miniprogram-phone-integration.sh` 只是调用上述同一失败 tests，不提供 v1-v13 等价替代。
`TestMerchantIdentityMySQL8Integration` 虽能真实 apply/repeat v1-v13，但不覆盖 catalog、identity、
menu、storefront 的相同行为集合，因此不能替代完整 W3 matrix。

## 最小独立 foundation change

建议 change：`align-real-mysql-integration-suites-v13`。

Exact owned paths：

- `.scratch/align-real-mysql-integration-suites-v13/**`
- `services/api/internal/migrate/mysql_integration_test.go`
- `services/api/internal/catalog/mysql_integration_test.go`
- `services/api/internal/identity/mysql_integration_test.go`
- `services/api/internal/menu/mysql_integration_test.go`
- `services/api/internal/storefront/mysql_integration_test.go`

其余全部只读，尤其 product implementation、`services/api/migrations/**`、`go.mod/go.sum`、
本 `paymentobservation/**`。

Red：在 exact base 的 fresh MySQL 8.0.46 上运行完整上述 5 packages（可并列
`merchantidentity`），首个决定性错误必须仍是 `count=13, want=10`，并登记所有旧
ToVersion/AppliedCount/repeat 断言。

Green：只把 suite 的“全集 latest”断言对齐 v13；保留 identity 的 v9→v10 专项迁移切片语义，
但全集最终运行到 v13。不得改 migration/product code，不得删除/skip 任一 real-MySQL 行为。

验收：fresh loopback-only MySQL 8.0.46，运行：

```bash
GOPROXY=off GOTOOLCHAIN=go1.26.5 go test -race \
  ./services/api/internal/migrate \
  ./services/api/internal/catalog \
  ./services/api/internal/identity \
  ./services/api/internal/menu \
  ./services/api/internal/storefront \
  ./services/api/internal/merchantidentity \
  -count=1 -timeout=10m
```

再运行 `go test ./services/api/...`、`go test -race ./services/api/...`、`go vet`、controlled
build/smoke、owned/protected diff、fresh detached exact-SHA verifier。该 foundation change 集成到
主控固定 base 后，本 change 必须更新 base，形成新候选并从头重跑全部 Gate；当前旧证据不得复用为
Candidate PASS。
