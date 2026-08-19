#!/usr/bin/env bash
set -euo pipefail

temporary_directory="$(mktemp -d)"
binary_path="${temporary_directory}/order-api"
log_path="${temporary_directory}/order-api.log"
server_pid=""

cleanup() {
  if [[ -n "${server_pid}" ]] && kill -0 "${server_pid}" 2>/dev/null; then
    kill -TERM "${server_pid}" 2>/dev/null || true
    wait "${server_pid}" 2>/dev/null || true
  fi
  rm -f "${binary_path}" "${log_path}" "${temporary_directory}/ready.json" \
    "${temporary_directory}/session.json" "${temporary_directory}/phone.json" \
    "${temporary_directory}/primary-phone.json" "${temporary_directory}/primary-phone.headers" \
    "${temporary_directory}/route.json" \
    "${temporary_directory}/catalog.json" "${temporary_directory}/menu.json"
  rmdir "${temporary_directory}" 2>/dev/null || true
}
trap cleanup EXIT

go build -o "${binary_path}" ./services/api/cmd/order-api

ORDER_API_HTTP_ADDR="127.0.0.1:0" \
ORDER_API_SHUTDOWN_TIMEOUT="2s" \
ORDER_ENV="development" \
ORDER_DB_HOST="127.0.0.1" \
ORDER_DB_PORT="1" \
ORDER_DB_NAME="order_smoke" \
ORDER_DB_USER="order_smoke" \
ORDER_DB_PASSWORD="smoke-canary-secret" \
ORDER_DB_TLS_MODE="disabled" \
ORDER_WECHAT_MINIPROGRAM_APP_ID="wx-smoke-app-id-canary" \
ORDER_WECHAT_MINIPROGRAM_APP_SECRET="smoke-miniprogram-secret-canary" \
  "${binary_path}" >"${log_path}" 2>&1 &
server_pid=$!

server_address=""
for _ in {1..100}; do
  if ! kill -0 "${server_pid}" 2>/dev/null; then
    wait "${server_pid}"
    echo "order-api exited before becoming ready" >&2
    exit 1
  fi
  server_address="$(sed -n 's/.*"msg":"http server started".*"addr":"\([^"]*\)".*/\1/p' "${log_path}" | tail -1)"
  if [[ -n "${server_address}" ]]; then
    break
  fi
  sleep 0.1
done

if [[ -z "${server_address}" ]]; then
  echo "order-api did not report its listener" >&2
  exit 1
fi

live_response="$(curl --fail --silent --show-error "http://${server_address}/health/live")"
[[ "${live_response}" == '{"status":"ok"}' ]]

ready_file="${temporary_directory}/ready.json"
ready_status="$(curl --silent --show-error --output "${ready_file}" --write-out '%{http_code}' "http://${server_address}/health/ready")"
[[ "${ready_status}" == "503" ]]
[[ "$(<"${ready_file}")" == '{"status":"not_ready","reason":"database_unreachable"}' ]]

session_file="${temporary_directory}/session.json"
session_status="$(curl --silent --show-error --output "${session_file}" --write-out '%{http_code}' \
  -H 'Content-Type: application/json' --data '{}' "http://${server_address}/api/v1/auth/miniprogram/session")"
[[ "${session_status}" == "400" ]]
[[ "$(<"${session_file}")" == '{"error":{"code":"INVALID_REQUEST","message":"invalid request"}}' ]]

phone_file="${temporary_directory}/phone.json"
phone_status="$(curl --silent --show-error --output "${phone_file}" --write-out '%{http_code}' \
  -H 'Content-Type: application/json' --data '{"code":"smoke-phone-code-canary"}' \
  "http://${server_address}/api/v1/me/bind-phone")"
[[ "${phone_status}" == "401" ]]
[[ "$(<"${phone_file}")" == '{"error":{"code":"UNAUTHENTICATED","message":"authentication required"}}' ]]

primary_phone_file="${temporary_directory}/primary-phone.json"
primary_phone_headers="${temporary_directory}/primary-phone.headers"
primary_phone_status="$(curl --silent --show-error --dump-header "${primary_phone_headers}" --output "${primary_phone_file}" --write-out '%{http_code}' \
  "http://${server_address}/api/v1/me/primary-phone")"
if [[ "${primary_phone_status}" != "401" ]]; then
  echo "primary-phone status route returned ${primary_phone_status}, want 401" >&2
  exit 1
fi
[[ "$(<"${primary_phone_file}")" == '{"error":{"code":"UNAUTHENTICATED","message":"authentication required"}}' ]]
grep -Eiq '^Cache-Control: no-store\r?$' "${primary_phone_headers}"

route_file="${temporary_directory}/route.json"
route_status="$(curl --silent --show-error --output "${route_file}" --write-out '%{http_code}' \
  "http://${server_address}/api/v1/auth/miniprogram/session")"
[[ "${route_status}" == "405" ]]
[[ ! -s "${route_file}" ]]
route_status="$(curl --silent --show-error --output "${route_file}" --write-out '%{http_code}' \
  -H 'Content-Type: application/json' --data '{}' "http://${server_address}/auth/miniprogram/session")"
[[ "${route_status}" == "404" ]]
[[ ! -s "${route_file}" ]]
route_status="$(curl --silent --show-error --output "${route_file}" --write-out '%{http_code}' \
  -H 'Content-Type: application/json' --data '{"code":"smoke-phone-code-canary"}' \
  "http://${server_address}/me/bind-phone")"
[[ "${route_status}" == "404" ]]
[[ ! -s "${route_file}" ]]
route_status="$(curl --silent --show-error --output "${route_file}" --write-out '%{http_code}' \
  -H 'Content-Type: application/json' --data '{}' \
  "http://${server_address}/api/v1/me/profile")"
[[ "${route_status}" == "404" ]]
[[ ! -s "${route_file}" ]]

catalog_file="${temporary_directory}/catalog.json"
catalog_status="$(curl --silent --show-error --output "${catalog_file}" --write-out '%{http_code}' \
  "http://${server_address}/api/v1/catalog")"
[[ "${catalog_status}" == "503" ]]
[[ "$(<"${catalog_file}")" == '{"error":{"code":"CATALOG_UNAVAILABLE","message":"catalog temporarily unavailable"}}' ]]

menu_file="${temporary_directory}/menu.json"
service_date="$(TZ=Asia/Shanghai date +%F)"
menu_status="$(curl --silent --show-error --output "${menu_file}" --write-out '%{http_code}' \
  "http://${server_address}/api/v1/menu?date=${service_date}&time=12:00")"
[[ "${menu_status}" == "503" ]]
[[ "$(<"${menu_file}")" == '{"error":{"code":"MENU_UNAVAILABLE","message":"menu temporarily unavailable"}}' ]]

status="$(curl --silent --output /dev/null --write-out '%{http_code}' "http://${server_address}/missing")"
[[ "${status}" == "404" ]]

kill -TERM "${server_pid}"
wait "${server_pid}"
server_pid=""

if ORDER_ENV="development" \
  ORDER_DB_HOST="127.0.0.1" \
  ORDER_DB_PORT="invalid" \
  ORDER_DB_NAME="order_smoke" \
  ORDER_DB_USER="order_smoke" \
  ORDER_DB_PASSWORD="smoke-canary-secret" \
  ORDER_DB_TLS_MODE="disabled" \
  ORDER_WECHAT_MINIPROGRAM_APP_ID="wx-smoke-app-id-canary" \
  ORDER_WECHAT_MINIPROGRAM_APP_SECRET="smoke-miniprogram-secret-canary" \
  "${binary_path}" >>"${log_path}" 2>&1; then
  echo "order-api accepted invalid database configuration" >&2
  exit 1
fi

grep -q 'configuration error' "${log_path}"
if grep -Eq 'smoke-canary-secret|wx-smoke-app-id-canary|smoke-miniprogram-secret-canary|smoke-phone-code-canary' "${log_path}"; then
  echo "order-api leaked the smoke canary" >&2
  exit 1
fi
echo "smoke: PASS"
