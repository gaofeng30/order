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
  rm -f "${binary_path}" "${log_path}"
  rmdir "${temporary_directory}" 2>/dev/null || true
}
trap cleanup EXIT

go build -o "${binary_path}" ./services/api/cmd/order-api

ORDER_API_HTTP_ADDR="127.0.0.1:0" \
ORDER_API_SHUTDOWN_TIMEOUT="2s" \
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

for path in health/live health/ready; do
  response="$(curl --fail --silent --show-error "http://${server_address}/${path}")"
  [[ "${response}" == '{"status":"ok"}' ]]
done

status="$(curl --silent --output /dev/null --write-out '%{http_code}' "http://${server_address}/missing")"
[[ "${status}" == "404" ]]

kill -TERM "${server_pid}"
wait "${server_pid}"
server_pid=""

if ORDER_API_SHUTDOWN_TIMEOUT="invalid" "${binary_path}" >>"${log_path}" 2>&1; then
  echo "order-api accepted invalid shutdown timeout" >&2
  exit 1
fi

grep -q 'configuration error' "${log_path}"
echo "smoke: PASS"
