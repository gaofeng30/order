#!/usr/bin/env bash
set -euo pipefail

m04_repo=$(git rev-parse --show-toplevel)
m04_base=7a3fd996a4702c5514dc69aeff6c49f7145bb7ae
m04_mode=${1:-full}
m04_container=order-mysql-w3
m04_port=33093
m04_deps=${MINIPROGRAM_UI_DEPS:-/Users/vivix/.codex/worktrees/a61d/order/tools/miniprogram-ui}
cd "${m04_repo}"

m04_paths() {
  {
    git diff --name-only "${m04_base}"...HEAD
    git diff --cached --name-only
    git diff --name-only
    git ls-files --others --exclude-standard
  } | LC_ALL=C sort -u
}

verify_source() {
  test "$(git merge-base "${m04_base}" HEAD)" = "${m04_base}"
  git diff --check "${m04_base}"...HEAD
  git diff --cached --check
  git diff --check
  while IFS= read -r m04_path; do
    [[ -z "${m04_path}" ]] && continue
    case "${m04_path}" in
      .scratch/page-m04-cross-date-redeem/*|\
      .scratch/mini-merchant-pages-closure/miniprogram-gates.json|\
      apps/wechat-miniprogram/tests/merchant-pages-closure-m04-cross-date-contract.test.js|\
      apps/wechat-miniprogram/tests/merchant-pages-closure-ui1.spec.cjs|\
      apps/wechat-miniprogram/tests/run-merchant-pages-closure-ui1.mjs) ;;
      *) printf 'out-of-scope path: %s\n' "${m04_path}" >&2; exit 97 ;;
    esac
  done < <(m04_paths)
}

verify_static() {
  verify_source
  node --check apps/wechat-miniprogram/tests/run-merchant-pages-closure-ui1.mjs
  node --check apps/wechat-miniprogram/tests/merchant-pages-closure-ui1.spec.cjs
  node --test apps/wechat-miniprogram/tests/merchant-pages-closure-m04-cross-date-contract.test.js
  python3 apps/wechat-miniprogram/tests/lint_wx.py apps/wechat-miniprogram
}

verify_static
if [[ "${m04_mode}" == static ]]; then
  printf 'PAGE_M04_STATIC_GATE=PASS base=%s\n' "${m04_base}"
  exit 0
fi
[[ "${m04_mode}" == full ]] || { printf 'usage: %s [static|full]\n' "$0" >&2; exit 64; }

m04_schema="order_m04_cross_date_$(date +%s)_${RANDOM}"
[[ "${m04_schema}" =~ ^order_m04_cross_date_[0-9]+_[0-9]+$ ]]
m04_password=$(/opt/homebrew/bin/docker inspect --format '{{range .Config.Env}}{{println .}}{{end}}' "${m04_container}" | sed -n 's/^MYSQL_ROOT_PASSWORD=//p' | head -1)
[[ -n "${m04_password}" ]]
m04_tmp=$(mktemp -d /private/tmp/order-m04-cross-date.XXXXXX)
m04_api_pid=''
cleanup_m04() {
  if [[ -n "${m04_api_pid}" ]]; then
    kill "${m04_api_pid}" 2>/dev/null || true
    wait "${m04_api_pid}" 2>/dev/null || true
  fi
  /opt/homebrew/bin/docker exec "${m04_container}" sh -c "exec mysql -uroot -p\"\$MYSQL_ROOT_PASSWORD\" --execute 'DROP DATABASE IF EXISTS ${m04_schema}'" >/dev/null 2>&1 || true
  find "${m04_tmp}" -type f -delete 2>/dev/null || true
  rmdir "${m04_tmp}" 2>/dev/null || true
}
trap cleanup_m04 EXIT INT TERM

/opt/homebrew/bin/docker exec "${m04_container}" sh -c "exec mysql -uroot -p\"\$MYSQL_ROOT_PASSWORD\" --execute 'CREATE DATABASE ${m04_schema} CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci'"
GOCACHE=/Users/vivix/Library/Caches/go-build GOMODCACHE=/Users/vivix/go/pkg/mod go build -o "${m04_tmp}/order-migrate" ./services/api/cmd/order-migrate
GOCACHE=/Users/vivix/Library/Caches/go-build GOMODCACHE=/Users/vivix/go/pkg/mod go build -o "${m04_tmp}/order-bootstrap" ./services/api/cmd/order-bootstrap
GOCACHE=/Users/vivix/Library/Caches/go-build GOMODCACHE=/Users/vivix/go/pkg/mod go build -o "${m04_tmp}/order-api" ./services/api/cmd/order-api
export ORDER_ENV=development ORDER_DB_HOST=127.0.0.1 ORDER_DB_PORT="${m04_port}" ORDER_DB_NAME="${m04_schema}" ORDER_DB_USER=root ORDER_DB_PASSWORD="${m04_password}" ORDER_DB_TLS_MODE=disabled
export ORDER_WECHAT_MINIPROGRAM_APP_ID=order-local-app ORDER_WECHAT_MINIPROGRAM_APP_SECRET=order-local-secret
"${m04_tmp}/order-migrate"
ORDER_BOOTSTRAP_OWNER_PHONE=+8613800000000 ORDER_BOOTSTRAP_OWNER_NAME=甲方管理员 ORDER_BOOTSTRAP_STORE_NAME=甲方门店 ORDER_BOOTSTRAP_STORE_ADDRESS=甲方地址 ORDER_BOOTSTRAP_PICKUP_POINT=甲方取餐点 "${m04_tmp}/order-bootstrap"
m04_api_port=$(ruby -rsocket -e 'socket=TCPServer.new("127.0.0.1",0); puts socket.addr[1]; socket.close')
ORDER_API_HTTP_ADDR="127.0.0.1:${m04_api_port}" ORDER_LOCAL_PAYMENT_AUTO_PAY=true "${m04_tmp}/order-api" >"${m04_tmp}/api.log" 2>&1 &
m04_api_pid=$!
for _ in {1..80}; do
  curl -fsS "http://127.0.0.1:${m04_api_port}/health/ready" >/dev/null 2>&1 && break
  sleep 0.1
done
curl -fsS "http://127.0.0.1:${m04_api_port}/health/ready" >/dev/null

m04_candidate=$(git rev-parse HEAD)
PLAYWRIGHT_BROWSERS_PATH=0 MINIPROGRAM_UI_DEPS="${m04_deps}" \
ORDER_MERCHANT_CLOSURE_API_ORIGIN="http://127.0.0.1:${m04_api_port}" \
ORDER_MERCHANT_CLOSURE_MYSQL_CONTAINER="${m04_container}" \
ORDER_MERCHANT_CLOSURE_MYSQL_DATABASE="${m04_schema}" \
ORDER_MERCHANT_CLOSURE_MYSQL_USER=root \
ORDER_MERCHANT_CLOSURE_MYSQL_PASSWORD="${m04_password}" \
ORDER_MERCHANT_CLOSURE_RECEIPT_PATH="/private/tmp/order-m04-cross-date-${m04_candidate}/receipt.json" \
node apps/wechat-miniprogram/tests/run-merchant-pages-closure-ui1.mjs

verify_static
printf 'PAGE_M04_WRITER_GATE=PASS base=%s candidate=%s tree=%s receipt=%s\n' \
  "${m04_base}" "${m04_candidate}" "$(git rev-parse "${m04_candidate}^{tree}")" \
  "/private/tmp/order-m04-cross-date-${m04_candidate}/receipt.json"
