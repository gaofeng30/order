# v13 exact-base RED

**Base:** `5e937f3599a16f4813d6021f4cd2dd637c3156a2`

**Isolation:** fresh detached disposable worktree；只复制
`services/api/internal/paymentobservation/normalize_test.go`，不复制 production implementation。

```text
$ GOPROXY=off GOTOOLCHAIN=go1.26.5 go test ./services/api/internal/paymentobservation -run '^TestNormalizeCanonicalizesSuccessfulCallbackAndQueryIdentically$' -count=1
github.com/gaofeng30/order/services/api/internal/paymentobservation: no non-test Go files
FAIL github.com/gaofeng30/order/services/api/internal/paymentobservation [build failed]
RED_EXIT=1
```

首个决定性错误证明新 exact base 没有公共 `Normalize` production seam；disposable worktree 已移除。
旧 `b7f484f...7483` RED 仅保留为历史证据，不用于本次候选。
