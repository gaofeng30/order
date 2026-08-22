#!/usr/bin/env bash
set -euo pipefail

repository_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
api_unit="${repository_root}/deploy/systemd/order-api.service"
migrate_unit="${repository_root}/deploy/systemd/order-migrate.service"
environment_file="${repository_root}/deploy/systemd/order-production.env.example"
preflight="${repository_root}/deploy/systemd/order-preflight"
healthcheck="${repository_root}/deploy/systemd/order-healthcheck"
startup_check="${repository_root}/deploy/checks/verify-production-startup.sh"
runbook="${repository_root}/deploy/runbooks/cvm-recovery.md"

for required_file in "${api_unit}" "${migrate_unit}" "${environment_file}" "${preflight}" "${healthcheck}" "${startup_check}" "${runbook}"; do
  if [[ ! -f "${required_file}" ]]; then
    echo "missing runtime artifact: ${required_file#"${repository_root}/"}" >&2
    exit 1
  fi
done

require_literal() {
  local file="$1"
  local literal="$2"
  if ! grep -Fq -- "${literal}" "${file}"; then
    echo "missing required contract in ${file#"${repository_root}/"}: ${literal}" >&2
    exit 1
  fi
}

for required_key in ORDER_ENV ORDER_API_HTTP_ADDR ORDER_API_SHUTDOWN_TIMEOUT ORDER_DB_HOST ORDER_DB_PORT ORDER_DB_NAME ORDER_DB_USER ORDER_DB_TLS_MODE ORDER_WECHAT_MINIPROGRAM_APP_ID ORDER_TENCENT_REGION; do
  if ! grep -Eq "^${required_key}=" "${environment_file}"; then
    echo "missing EnvironmentFile key: ${required_key}" >&2
    exit 1
  fi
done

if grep -Eq '^(ORDER_DB_PASSWORD|ORDER_DB_DSN|ORDER_WECHAT_MINIPROGRAM_APP_SECRET|TENCENTCLOUD_SECRET_ID|TENCENTCLOUD_SECRET_KEY|TENCENTCLOUD_TOKEN|TENCENTCLOUD_CREDENTIALS_FILE)=' "${environment_file}"; then
  echo "EnvironmentFile contains a forbidden secret or persistent credential field" >&2
  exit 1
fi

require_literal "${api_unit}" "User=order"
require_literal "${api_unit}" "EnvironmentFile=/etc/order/order-production.env"
require_literal "${api_unit}" "ExecStartPre=+/opt/order/current/order-preflight"
require_literal "${api_unit}" "ExecStart=/opt/order/current/order-api"
require_literal "${api_unit}" "ExecStartPost=/opt/order/current/order-healthcheck live"
require_literal "${migrate_unit}" "ExecStart=/opt/order/current/order-migrate"
require_literal "${runbook}" "order-migrate.service"
require_literal "${runbook}" "order-healthcheck ready"
require_literal "${runbook}" "CAM Role"

require_literal "${repository_root}/services/api/internal/config/config.go" 'order-production-db-password'
require_literal "${repository_root}/services/api/internal/config/config.go" 'order-production-wechat-miniprogram-app-secret'
require_literal "${repository_root}/services/api/internal/config/tencent_ssm.go" 'common.DefaultCvmRoleProvider()'
require_literal "${repository_root}/services/api/internal/config/tencent_ssm.go" 'ssm.tencentcloudapi.com'

if rg -n '/order/\{env\}/' "${repository_root}/docs/product/online-ordering-system-technical.md" "${repository_root}/docs/微信小程序开发和运维指南/腾讯云操作指南.md" >/dev/null; then
  echo "retired slash-style SSM contract remains" >&2
  exit 1
fi
if rg -n 'DefaultProviderChain|DefaultEnvProvider|DefaultProfileProvider|NewCredential\(|NewTokenCredential\(' "${repository_root}/services/api/internal/config" -g '*.go' -g '!**/*_test.go' >/dev/null; then
  echo "production config contains a forbidden credential provider" >&2
  exit 1
fi
if find "${repository_root}/deploy" -iname '*docker*' -print -quit | grep -q .; then
  echo "Docker artifact found in systemd/CVM change" >&2
  exit 1
fi

bash -n "${preflight}" "${healthcheck}" "${startup_check}"
echo "RUNTIME_FOUNDATION_GATE=PASS"
