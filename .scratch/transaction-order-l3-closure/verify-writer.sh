#!/usr/bin/env bash
set -euo pipefail

tx_repo=$(git rev-parse --show-toplevel)
tx_base=fd33f58fb2e812ede1057b9ae3822e047cbfe497
tx_mode=${1:-full}
tx_deps=${MINIPROGRAM_UI_DEPS:-/Users/vivix/.codex/worktrees/a61d/order/tools/miniprogram-ui}
cd "${tx_repo}"

tx_paths() {
  {
    git diff --name-only "${tx_base}"...HEAD
    git diff --cached --name-only
    git diff --name-only
    git ls-files --others --exclude-standard
  } | LC_ALL=C sort -u
}

verify_source() {
  test "$(git merge-base "${tx_base}" HEAD)" = "${tx_base}"
  git diff --check "${tx_base}"...HEAD
  git diff --cached --check
  git diff --check
  while IFS= read -r tx_path; do
    [[ -z "${tx_path}" ]] && continue
    case "${tx_path}" in
      .scratch/transaction-order-l3-closure/*|\
      services/api/cmd/order-api/acceptance_transaction_order_l3_e2e_test.go|\
      services/api/internal/refund/handler.go|\
      services/api/internal/refund/handler_test.go|\
      tools/miniprogram-ui/karma.transaction-order-l3.conf.cjs|\
      tools/miniprogram-ui/run-ui1-transaction-order-l3.mjs|\
      tools/miniprogram-ui/test/transaction-order-l3-closure-contract.test.mjs|\
      tools/miniprogram-ui/test/browser/transaction-order-l3-runtime-endpoint-config.cjs|\
      tools/miniprogram-ui/test/browser/ui1-transaction-order-l3.spec.cjs|\
      apps/wechat-miniprogram/pages/confirm/confirm.js|\
      apps/wechat-miniprogram/tests/overnight-checkout-ui0.test.js|\
      apps/wechat-miniprogram/pages/home/home.js|\
      apps/wechat-miniprogram/pages/home/home.wxml|\
      apps/wechat-miniprogram/pages/order-detail/order-detail.js|\
      apps/wechat-miniprogram/pages/order-detail/order-detail.wxml) ;;
      *) printf 'out-of-scope path: %s\n' "${tx_path}" >&2; exit 97 ;;
    esac
  done < <(tx_paths)
}

verify_static() {
  verify_source
  node --check tools/miniprogram-ui/run-ui1-transaction-order-l3.mjs
  node --check tools/miniprogram-ui/karma.transaction-order-l3.conf.cjs
  node --check tools/miniprogram-ui/test/browser/transaction-order-l3-runtime-endpoint-config.cjs
  node --check tools/miniprogram-ui/test/browser/ui1-transaction-order-l3.spec.cjs
  node --test tools/miniprogram-ui/test/transaction-order-l3-closure-contract.test.mjs
  node --test apps/wechat-miniprogram/tests/overnight-checkout-ui0.test.js
  GOTOOLCHAIN=go1.26.5 GOPROXY=off go test ./services/api/internal/refund -run '^TestUserCancelMapsFrozenCancellationBoundaryToConflict$' -count=1
  GOTOOLCHAIN=go1.26.5 GOPROXY=off go test ./services/api/cmd/order-api -run '^TestTransactionOrderL3Server$' -count=1
  python3 apps/wechat-miniprogram/tests/lint_wx.py apps/wechat-miniprogram
}

verify_static
if [[ "${tx_mode}" == static ]]; then
  printf 'TRANSACTION_ORDER_L3_STATIC_GATE=PASS base=%s\n' "${tx_base}"
  exit 0
fi
[[ "${tx_mode}" == full ]] || { printf 'usage: %s [static|full]\n' "$0" >&2; exit 64; }

: "${ORDER_TEST_MYSQL_HOST:?ORDER_TEST_MYSQL_HOST is required}"
: "${ORDER_TEST_MYSQL_PORT:?ORDER_TEST_MYSQL_PORT is required}"
: "${ORDER_TEST_MYSQL_USER:?ORDER_TEST_MYSQL_USER is required}"
: "${ORDER_TEST_MYSQL_PASSWORD:?ORDER_TEST_MYSQL_PASSWORD is required}"
: "${ORDER_TEST_MYSQL_TLS_MODE:?ORDER_TEST_MYSQL_TLS_MODE is required}"
[[ "${ORDER_TEST_MYSQL_INSTANCE:-}" == order-mysql-w3 ]]
[[ "${ORDER_TEST_MYSQL_ISOLATED:-}" == YES ]]
[[ -f "${tx_deps}/package.json" ]]
[[ "$(/opt/homebrew/bin/docker inspect --format '{{.State.Running}}' order-mysql-w3)" == true ]]

tx_candidate=$(git rev-parse HEAD)
PLAYWRIGHT_BROWSERS_PATH=0 MINIPROGRAM_UI_DEPS="${tx_deps}" \
GOCACHE=/Users/vivix/Library/Caches/go-build GOMODCACHE=/Users/vivix/go/pkg/mod \
GOTOOLCHAIN=go1.26.5 GOPROXY=off \
node tools/miniprogram-ui/run-ui1-transaction-order-l3.mjs

verify_static
printf 'TRANSACTION_ORDER_L3_WRITER_GATE=PASS base=%s candidate=%s tree=%s receipt=%s\n' \
  "${tx_base}" "${tx_candidate}" "$(git rev-parse "${tx_candidate}^{tree}")" \
  "/private/tmp/order-transaction-l3-${tx_candidate:0:12}/receipt.json"
