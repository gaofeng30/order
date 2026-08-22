#!/usr/bin/env bash
set -euo pipefail

repository_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
temporary_directory="$(mktemp -d)"
binary_path="${temporary_directory}/order-api"
log_path="${temporary_directory}/order-api.log"
credential_canary="production-persistent-credential-must-not-leak"

cleanup() {
  rm -f "${binary_path}" "${log_path}"
  rmdir "${temporary_directory}" 2>/dev/null || true
}
trap cleanup EXIT

cd "${repository_root}"
go test ./services/api/internal/config -count=1
go build -o "${binary_path}" ./services/api/cmd/order-api

if env -i \
  PATH="${PATH}" \
  ORDER_ENV="production" \
  TENCENTCLOUD_SECRET_ID="${credential_canary}" \
  "${binary_path}" >"${log_path}" 2>&1; then
  echo "order-api accepted a persistent production credential" >&2
  exit 1
fi

grep -Fq '"reason":"production_secret_environment_forbidden"' "${log_path}"
if grep -Eq "${credential_canary}|TENCENTCLOUD_SECRET_ID|order-production-db-password|order-production-wechat-miniprogram-app-secret" "${log_path}"; then
  echo "order-api leaked production credential details" >&2
  exit 1
fi

echo "PRODUCTION_STARTUP_GATE=PASS"
