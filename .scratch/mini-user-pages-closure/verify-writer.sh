#!/usr/bin/env bash
set -euo pipefail

repo_root=$(git rev-parse --show-toplevel)
expected_base="00cf26f0815ed58979b228806f1f99519b6c40bf"
candidate_sha=$(git rev-parse HEAD)
mysql_host=${ORDER_TEST_MYSQL_HOST:?ORDER_TEST_MYSQL_HOST is required}
mysql_port=${ORDER_TEST_MYSQL_PORT:?ORDER_TEST_MYSQL_PORT is required}
mysql_user=${ORDER_TEST_MYSQL_USER:?ORDER_TEST_MYSQL_USER is required}
mysql_password=${ORDER_TEST_MYSQL_PASSWORD:?ORDER_TEST_MYSQL_PASSWORD is required}
mysql_tls=${ORDER_TEST_MYSQL_TLS_MODE:?ORDER_TEST_MYSQL_TLS_MODE is required}
mysql_container=${ORDER_TEST_MYSQL_INSTANCE:?ORDER_TEST_MYSQL_INSTANCE is required}
api_port=${ORDER_USER_PAGES_API_PORT:?ORDER_USER_PAGES_API_PORT is required}
dependency_root=${MINIPROGRAM_UI_DEPS:?MINIPROGRAM_UI_DEPS is required}
database="order_user_pages_$(date +%s)_$$"
build_root=$(mktemp -d /private/tmp/order-mini-user-pages-build.XXXXXX)
evidence_root="/private/tmp/order-mini-user-pages-${candidate_sha:0:12}-$(date +%s)-$$"
api_binary="${build_root}/order-api"
migrate_binary="${build_root}/order-migrate"
bootstrap_binary="${build_root}/order-bootstrap"
api_log="${evidence_root}/order-api.log"
receipt_path="${evidence_root}/receipt.json"
api_pid=""
schema_created=NO

cleanup() {
  local status=$?
  if [[ -n "${api_pid}" ]] && kill -0 "${api_pid}" 2>/dev/null; then
    kill -TERM "${api_pid}" 2>/dev/null || true
    wait "${api_pid}" 2>/dev/null || true
  fi
  if [[ "${schema_created}" == YES ]]; then
    /opt/homebrew/bin/docker exec -e "MYSQL_PWD=${mysql_password}" "${mysql_container}" \
      mysql --batch --raw --skip-column-names -u "${mysql_user}" \
      --execute="DROP DATABASE IF EXISTS \`${database}\`" >/dev/null 2>&1 || status=1
  fi
  rm -f "${api_binary}" "${migrate_binary}" "${bootstrap_binary}"
  rmdir "${build_root}" 2>/dev/null || status=1
  return "${status}"
}
trap cleanup EXIT INT TERM

cd "${repo_root}"

[[ "${ORDER_TEST_MYSQL_ISOLATED:-}" == YES ]]
[[ "${PLAYWRIGHT_BROWSERS_PATH:-}" == 0 ]]
[[ "${mysql_port}" =~ ^[1-9][0-9]{0,4}$ ]]
[[ "${api_port}" =~ ^[1-9][0-9]{3,4}$ ]]
[[ "${database}" =~ ^order_user_pages_[0-9_]+$ ]]
[[ -f "${dependency_root}/package.json" ]]
[[ -z "$(git status --porcelain)" ]]
git merge-base --is-ancestor "${expected_base}" "${candidate_sha}"

unexpected_paths=$(git diff --name-only "${expected_base}...${candidate_sha}" | \
  rg -v '^(apps/wechat-miniprogram/pages/detail/detail\.(js|wxml|wxss)|apps/wechat-miniprogram/tests/mini-user-pages-closure-ui0\.test\.js|services/api/cmd/order-bootstrap/(application\.go|application_mysql_test\.go|main_test\.go)|tools/miniprogram-ui/(karma\.composed-user-pages-closure\.conf\.cjs|run-ui1-composed-user-pages-closure\.mjs|test/browser/(composed-user-pages-runtime-endpoint-config\.cjs|ui1-composed-user-pages-closure\.spec\.cjs))|\.scratch/mini-user-pages-closure/(miniprogram-gates\.json|verify-writer\.sh))$' || true)
if [[ -n "${unexpected_paths}" ]]; then
  printf 'owned path violation:\n%s\n' "${unexpected_paths}" >&2
  exit 1
fi

git diff --check "${expected_base}...${candidate_sha}"
node --check tools/miniprogram-ui/run-ui1-composed-user-pages-closure.mjs
node --check tools/miniprogram-ui/test/browser/ui1-composed-user-pages-closure.spec.cjs
node --check tools/miniprogram-ui/test/browser/composed-user-pages-runtime-endpoint-config.cjs
node --check tools/miniprogram-ui/karma.composed-user-pages-closure.conf.cjs
node --test apps/wechat-miniprogram/tests/mini-user-pages-closure-ui0.test.js
npm --prefix apps/wechat-miniprogram test
python3 apps/wechat-miniprogram/tests/lint_wx.py apps/wechat-miniprogram
go test ./services/api/cmd/order-bootstrap -count=1 -timeout=5m

go build -o "${api_binary}" ./services/api/cmd/order-api
go build -o "${migrate_binary}" ./services/api/cmd/order-migrate
go build -o "${bootstrap_binary}" ./services/api/cmd/order-bootstrap

if lsof -nP -iTCP:"${api_port}" -sTCP:LISTEN >/dev/null 2>&1; then
  printf 'private API port %s is already in use\n' "${api_port}" >&2
  exit 1
fi
[[ "$(/opt/homebrew/bin/docker inspect --format '{{.State.Running}}' "${mysql_container}")" == true ]]

mkdir -p "${evidence_root}"
/opt/homebrew/bin/docker exec -e "MYSQL_PWD=${mysql_password}" "${mysql_container}" \
  mysql --batch --raw --skip-column-names -u "${mysql_user}" \
  --execute="CREATE DATABASE \`${database}\` CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci"
schema_created=YES

runtime_env=(
  "ORDER_ENV=development"
  "ORDER_DB_HOST=${mysql_host}"
  "ORDER_DB_PORT=${mysql_port}"
  "ORDER_DB_NAME=${database}"
  "ORDER_DB_USER=${mysql_user}"
  "ORDER_DB_PASSWORD=${mysql_password}"
  "ORDER_DB_TLS_MODE=${mysql_tls}"
  "ORDER_WECHAT_MINIPROGRAM_APP_ID=wx-local-order"
  "ORDER_WECHAT_MINIPROGRAM_APP_SECRET=order-local-secret"
)

env "${runtime_env[@]}" "${migrate_binary}"
migration_state=$(/opt/homebrew/bin/docker exec -e "MYSQL_PWD=${mysql_password}" "${mysql_container}" \
  mysql --batch --raw --skip-column-names -u "${mysql_user}" --database="${database}" \
  --execute="SELECT IF(COUNT(*)=44 AND MAX(version)=44 AND SUM(dirty)=0,1,0) FROM schema_migrations")
[[ "${migration_state}" == 1 ]]

env "${runtime_env[@]}" \
  "ORDER_BOOTSTRAP_OWNER_PHONE=+8613800000000" \
  "ORDER_BOOTSTRAP_OWNER_NAME=本地验收主账号" \
  "ORDER_BOOTSTRAP_STORE_NAME=绥安食品" \
  "ORDER_BOOTSTRAP_STORE_ADDRESS=本地验收地址" \
  "ORDER_BOOTSTRAP_PICKUP_POINT=本地验收取餐点" \
  "${bootstrap_binary}"

env "${runtime_env[@]}" \
  "ORDER_API_HTTP_ADDR=127.0.0.1:${api_port}" \
  "ORDER_API_SHUTDOWN_TIMEOUT=2s" \
  "ORDER_LOCAL_PAYMENT_AUTO_PAY=true" \
  "${api_binary}" >"${api_log}" 2>&1 &
api_pid=$!

ready=NO
for _ in {1..100}; do
  if ! kill -0 "${api_pid}" 2>/dev/null; then
    wait "${api_pid}" || true
    printf 'order-api exited before readiness; see %s\n' "${api_log}" >&2
    exit 1
  fi
  if [[ "$(curl --silent --show-error --output /dev/null --write-out '%{http_code}' "http://127.0.0.1:${api_port}/health/ready" || true)" == 200 ]]; then
    ready=YES
    break
  fi
  sleep 0.1
done
[[ "${ready}" == YES ]]

ORDER_USER_PAGES_FRESH_DB=YES \
ORDER_COMPOSED_API_ORIGIN="http://127.0.0.1:${api_port}" \
ORDER_COMPOSED_MYSQL_CONTAINER="${mysql_container}" \
ORDER_COMPOSED_MYSQL_DATABASE="${database}" \
ORDER_COMPOSED_MYSQL_USER="${mysql_user}" \
ORDER_COMPOSED_MYSQL_PASSWORD="${mysql_password}" \
ORDER_USER_PAGES_EVIDENCE_DIR="${evidence_root}" \
ORDER_USER_PAGES_RECEIPT_PATH="${receipt_path}" \
MINIPROGRAM_UI_DEPS="${dependency_root}" \
PLAYWRIGHT_BROWSERS_PATH=0 \
  node tools/miniprogram-ui/run-ui1-composed-user-pages-closure.mjs

node -e 'const fs=require("node:fs"); const receipt=JSON.parse(fs.readFileSync(process.argv[1],"utf8")); if(receipt.status!=="PASS"||receipt.candidate_sha!==process.argv[2]||receipt.cases.length!==9) process.exit(1)' \
  "${receipt_path}" "${candidate_sha}"

kill -TERM "${api_pid}"
wait "${api_pid}"
api_pid=""
/opt/homebrew/bin/docker exec -e "MYSQL_PWD=${mysql_password}" "${mysql_container}" \
  mysql --batch --raw --skip-column-names -u "${mysql_user}" \
  --execute="DROP DATABASE \`${database}\`"
schema_created=NO
residual=$(/opt/homebrew/bin/docker exec -e "MYSQL_PWD=${mysql_password}" "${mysql_container}" \
  mysql --batch --raw --skip-column-names -u "${mysql_user}" \
  --execute="SELECT COUNT(*) FROM information_schema.schemata WHERE schema_name='${database}'")
[[ "${residual}" == 0 ]]
rm -f "${api_binary}" "${migrate_binary}" "${bootstrap_binary}"
rmdir "${build_root}"
trap - EXIT INT TERM
[[ -z "$(git status --porcelain)" ]]

printf 'MINI_USER_PAGES_WRITER_GATE %s\n' "$(node -e 'const fs=require("node:fs");const r=JSON.parse(fs.readFileSync(process.argv[1],"utf8"));process.stdout.write(JSON.stringify({status:r.status,candidate_sha:r.candidate_sha,cases:r.cases.length,browser:r.browser,evidence_level:r.evidence_level,child_results:r.child_results,mysql_evidence:r.mysql_evidence,receipt:process.argv[1],cleanup:{api_stopped:true,database_dropped:true}}))' "${receipt_path}")"
