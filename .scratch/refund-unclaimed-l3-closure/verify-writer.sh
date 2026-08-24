#!/usr/bin/env bash
set -euo pipefail

refund_repo=$(git rev-parse --show-toplevel)
refund_base=c5ab765b82f6869e6fb2176b2adb812a81b925d3
refund_mode=${1:-full}
refund_deps=${MINIPROGRAM_UI_DEPS:-/Users/vivix/.codex/worktrees/a61d/order/tools/miniprogram-ui}
cd "${refund_repo}"

refund_paths() {
  {
    git diff --name-only "${refund_base}"...HEAD
    git diff --cached --name-only
    git diff --name-only
    git ls-files --others --exclude-standard
  } | LC_ALL=C sort -u
}

verify_source() {
  test "$(git merge-base "${refund_base}" HEAD)" = "${refund_base}"
  git diff --check "${refund_base}"...HEAD
  git diff --cached --check
  git diff --check
  while IFS= read -r refund_path; do
    [[ -z "${refund_path}" ]] && continue
    case "${refund_path}" in
      .scratch/refund-unclaimed-l3-closure/*|\
      services/api/cmd/order-api/acceptance_refund_unclaimed_l3_e2e_test.go|\
      tools/miniprogram-ui/karma.refund-unclaimed-l3.conf.cjs|\
      tools/miniprogram-ui/run-ui1-refund-unclaimed-l3.mjs|\
      tools/miniprogram-ui/test/refund-unclaimed-l3-closure-contract.test.mjs|\
      tools/miniprogram-ui/test/browser/refund-unclaimed-l3-runtime-endpoint-config.cjs|\
      tools/miniprogram-ui/test/browser/ui1-refund-unclaimed-l3.spec.cjs) ;;
      *) printf 'out-of-scope path: %s\n' "${refund_path}" >&2; exit 97 ;;
    esac
  done < <(refund_paths)
}

verify_static() {
  verify_source
  test -z "$(gofmt -l services/api/cmd/order-api/acceptance_refund_unclaimed_l3_e2e_test.go)"
  node --check tools/miniprogram-ui/run-ui1-refund-unclaimed-l3.mjs
  node --check tools/miniprogram-ui/karma.refund-unclaimed-l3.conf.cjs
  node --check tools/miniprogram-ui/test/browser/refund-unclaimed-l3-runtime-endpoint-config.cjs
  node --check tools/miniprogram-ui/test/browser/ui1-refund-unclaimed-l3.spec.cjs
  node --test tools/miniprogram-ui/test/refund-unclaimed-l3-closure-contract.test.mjs
  GOTOOLCHAIN=go1.26.5 GOPROXY=off go test ./services/api/cmd/order-api -run '^TestRefundUnclaimedL3Server$' -count=1
  python3 apps/wechat-miniprogram/tests/lint_wx.py apps/wechat-miniprogram
}

verify_static
if [[ "${refund_mode}" == static ]]; then
  printf 'REFUND_UNCLAIMED_L3_STATIC_GATE=PASS base=%s\n' "${refund_base}"
  exit 0
fi
[[ "${refund_mode}" == full ]] || { printf 'usage: %s [static|full]\n' "$0" >&2; exit 64; }

: "${ORDER_TEST_MYSQL_HOST:?ORDER_TEST_MYSQL_HOST is required}"
: "${ORDER_TEST_MYSQL_PORT:?ORDER_TEST_MYSQL_PORT is required}"
: "${ORDER_TEST_MYSQL_USER:?ORDER_TEST_MYSQL_USER is required}"
: "${ORDER_TEST_MYSQL_PASSWORD:?ORDER_TEST_MYSQL_PASSWORD is required}"
: "${ORDER_TEST_MYSQL_TLS_MODE:?ORDER_TEST_MYSQL_TLS_MODE is required}"
[[ "${ORDER_TEST_MYSQL_INSTANCE:-}" == order-mysql-w3 ]]
[[ "${ORDER_TEST_MYSQL_ISOLATED:-}" == YES ]]
[[ -f "${refund_deps}/package.json" ]]
[[ "$(/opt/homebrew/bin/docker inspect --format '{{.State.Running}}' order-mysql-w3)" == true ]]

refund_candidate=$(git rev-parse HEAD)
PLAYWRIGHT_BROWSERS_PATH=0 MINIPROGRAM_UI_DEPS="${refund_deps}" \
GOCACHE=/Users/vivix/Library/Caches/go-build GOMODCACHE=/Users/vivix/go/pkg/mod \
GOTOOLCHAIN=go1.26.5 GOPROXY=off \
node tools/miniprogram-ui/run-ui1-refund-unclaimed-l3.mjs

verify_static
printf 'REFUND_UNCLAIMED_L3_WRITER_GATE=PASS base=%s candidate=%s tree=%s receipt=%s\n' \
  "${refund_base}" "${refund_candidate}" "$(git rev-parse "${refund_candidate}^{tree}")" \
  "/private/tmp/order-refund-unclaimed-l3-${refund_candidate:0:12}/receipt.json"
