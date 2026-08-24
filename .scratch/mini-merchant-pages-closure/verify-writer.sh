#!/usr/bin/env bash
set -euo pipefail

closure_repo=$(git rev-parse --show-toplevel)
closure_base=00cf26f0815ed58979b228806f1f99519b6c40bf
closure_container=order-mysql-w3
closure_port=33093
closure_deps=/Users/vivix/.codex/worktrees/a61d/order/tools/miniprogram-ui
cd "${closure_repo}"

closure_paths() {
  {
    git diff --name-only "${closure_base}"...HEAD
    git diff --cached --name-only
    git diff --name-only
    git ls-files --others --exclude-standard
  } | LC_ALL=C sort -u
}

verify_source() {
  git merge-base --is-ancestor "${closure_base}" HEAD
  git diff --check "${closure_base}"...HEAD
  git diff --cached --check
  git diff --check
  while IFS= read -r closure_path; do
    [[ -z "${closure_path}" ]] && continue
    case "${closure_path}" in
      .scratch/mini-merchant-pages-closure/*|\
      apps/wechat-miniprogram/pages/admin-orders/*|\
      apps/wechat-miniprogram/pages/admin-order-detail/*|\
      apps/wechat-miniprogram/pages/admin-products/*|\
      apps/wechat-miniprogram/tests/merchant-pages-closure-*|\
      apps/wechat-miniprogram/tests/overnight-merchant-ui0.test.js|\
      services/api/internal/fulfillment/*) ;;
      *) printf 'out-of-scope path: %s\n' "${closure_path}" >&2; exit 97 ;;
    esac
  done < <(closure_paths)
}

verify_source
test -z "$(gofmt -l services/api/internal/fulfillment/mysql.go services/api/internal/fulfillment/mysql_integration_test.go)"
node --check apps/wechat-miniprogram/tests/run-merchant-pages-closure-ui1.mjs
node --check apps/wechat-miniprogram/tests/merchant-pages-closure-ui1.spec.cjs
node --test apps/wechat-miniprogram/tests/merchant-pages-closure-ui0.test.js apps/wechat-miniprogram/tests/overnight-merchant-ui0.test.js
python3 apps/wechat-miniprogram/tests/lint_wx.py apps/wechat-miniprogram
GOPROXY=off GOTOOLCHAIN=go1.26.5 go test -race ./services/api/internal/fulfillment -count=1
GOPROXY=off GOTOOLCHAIN=go1.26.5 go vet ./services/api/internal/fulfillment

closure_schema="order_merchant_closure_$(date +%s)_${RANDOM}"
[[ "${closure_schema}" =~ ^order_merchant_closure_[0-9]+_[0-9]+$ ]]
closure_password=$(/opt/homebrew/bin/docker inspect --format '{{range .Config.Env}}{{println .}}{{end}}' "${closure_container}" | sed -n 's/^MYSQL_ROOT_PASSWORD=//p' | head -1)
[[ -n "${closure_password}" ]]
closure_tmp=$(mktemp -d /private/tmp/order-merchant-closure.XXXXXX)
closure_api_pid=''
cleanup_closure() {
  if [[ -n "${closure_api_pid}" ]]; then
    kill "${closure_api_pid}" 2>/dev/null || true
    wait "${closure_api_pid}" 2>/dev/null || true
  fi
  /opt/homebrew/bin/docker exec "${closure_container}" sh -c "exec mysql -uroot -p\"\$MYSQL_ROOT_PASSWORD\" --execute 'DROP DATABASE IF EXISTS ${closure_schema}'" >/dev/null 2>&1 || true
  find "${closure_tmp}" -type f -delete 2>/dev/null || true
  rmdir "${closure_tmp}" 2>/dev/null || true
}
trap cleanup_closure EXIT INT TERM

/opt/homebrew/bin/docker exec "${closure_container}" sh -c "exec mysql -uroot -p\"\$MYSQL_ROOT_PASSWORD\" --execute 'CREATE DATABASE ${closure_schema} CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci'"
GOCACHE=/Users/vivix/Library/Caches/go-build GOMODCACHE=/Users/vivix/go/pkg/mod go build -o "${closure_tmp}/order-migrate" ./services/api/cmd/order-migrate
GOCACHE=/Users/vivix/Library/Caches/go-build GOMODCACHE=/Users/vivix/go/pkg/mod go build -o "${closure_tmp}/order-bootstrap" ./services/api/cmd/order-bootstrap
GOCACHE=/Users/vivix/Library/Caches/go-build GOMODCACHE=/Users/vivix/go/pkg/mod go build -o "${closure_tmp}/order-api" ./services/api/cmd/order-api
export ORDER_ENV=development ORDER_DB_HOST=127.0.0.1 ORDER_DB_PORT="${closure_port}" ORDER_DB_NAME="${closure_schema}" ORDER_DB_USER=root ORDER_DB_PASSWORD="${closure_password}" ORDER_DB_TLS_MODE=disabled
export ORDER_WECHAT_MINIPROGRAM_APP_ID=order-local-app ORDER_WECHAT_MINIPROGRAM_APP_SECRET=order-local-secret
"${closure_tmp}/order-migrate"
ORDER_BOOTSTRAP_OWNER_PHONE=+8613800000000 ORDER_BOOTSTRAP_OWNER_NAME=甲方管理员 ORDER_BOOTSTRAP_STORE_NAME=甲方门店 ORDER_BOOTSTRAP_STORE_ADDRESS=甲方地址 ORDER_BOOTSTRAP_PICKUP_POINT=甲方取餐点 "${closure_tmp}/order-bootstrap"
closure_api_port=$(ruby -rsocket -e 'socket=TCPServer.new("127.0.0.1",0); puts socket.addr[1]; socket.close')
ORDER_API_HTTP_ADDR="127.0.0.1:${closure_api_port}" ORDER_LOCAL_PAYMENT_AUTO_PAY=true "${closure_tmp}/order-api" >"${closure_tmp}/api.log" 2>&1 &
closure_api_pid=$!
for _ in {1..80}; do
  curl -fsS "http://127.0.0.1:${closure_api_port}/health/ready" >/dev/null 2>&1 && break
  sleep 0.1
done
curl -fsS "http://127.0.0.1:${closure_api_port}/health/ready" >/dev/null

closure_candidate=$(git rev-parse HEAD)
PLAYWRIGHT_BROWSERS_PATH=0 MINIPROGRAM_UI_DEPS="${closure_deps}" \
ORDER_MERCHANT_CLOSURE_API_ORIGIN="http://127.0.0.1:${closure_api_port}" \
ORDER_MERCHANT_CLOSURE_MYSQL_CONTAINER="${closure_container}" \
ORDER_MERCHANT_CLOSURE_MYSQL_DATABASE="${closure_schema}" \
ORDER_MERCHANT_CLOSURE_MYSQL_USER=root \
ORDER_MERCHANT_CLOSURE_MYSQL_PASSWORD="${closure_password}" \
ORDER_MERCHANT_CLOSURE_RECEIPT_PATH="/private/tmp/order-merchant-closure-${closure_candidate}/receipt.json" \
node apps/wechat-miniprogram/tests/run-merchant-pages-closure-ui1.mjs

verify_source
printf 'MERCHANT_CLOSURE_WRITER_GATE=PASS base=%s candidate=%s tree=%s receipt=%s\n' \
  "${closure_base}" "${closure_candidate}" "$(git rev-parse "${closure_candidate}^{tree}")" \
  "/private/tmp/order-merchant-closure-${closure_candidate}/receipt.json"
