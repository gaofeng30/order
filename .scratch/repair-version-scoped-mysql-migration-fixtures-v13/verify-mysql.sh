#!/usr/bin/env bash
set -euo pipefail

version_scope_repo=$(git rev-parse --show-toplevel)
version_scope_profile=${1:-w3}
version_scope_container="order-version-scope-${version_scope_profile}-$$"
version_scope_password_file=$(mktemp /private/tmp/order-version-scope-password.XXXXXX)

cleanup_version_scope_mysql() {
  docker stop -t 1 "${version_scope_container}" >/dev/null 2>&1 || true
  unlink "${version_scope_password_file}" >/dev/null 2>&1 || true
}
trap cleanup_version_scope_mysql EXIT

chmod 600 "${version_scope_password_file}"
openssl rand -hex -out "${version_scope_password_file}" 24
IFS= read -r version_scope_password < "${version_scope_password_file}"

docker run -d --rm \
  --name "${version_scope_container}" \
  -p 127.0.0.1::3306 \
  -e MYSQL_ROOT_PASSWORD="${version_scope_password}" \
  mysql:8.0.46-oraclelinux9 >/dev/null

version_scope_ready=NO
for _ in {1..90}; do
  if docker exec "${version_scope_container}" mysqladmin --host=127.0.0.1 --protocol=tcp --user=root --password="${version_scope_password}" ping --silent >/dev/null 2>&1; then
    version_scope_ready=YES
    break
  fi
  sleep 1
done
if [[ "${version_scope_ready}" != YES ]]; then
  printf 'temporary MySQL did not become ready\n' >&2
  exit 90
fi

IFS= read -r version_scope_binding < <(docker port "${version_scope_container}" 3306/tcp)
if [[ "${version_scope_binding}" != 127.0.0.1:* ]]; then
  printf 'temporary MySQL is not loopback-only: %s\n' "${version_scope_binding}" >&2
  exit 91
fi
version_scope_port=${version_scope_binding##*:}
version_scope_version=$(docker exec "${version_scope_container}" mysql --batch --skip-column-names --user=root --password="${version_scope_password}" -e 'SELECT VERSION()' 2>/dev/null)
if [[ "${version_scope_version}" != 8.0.46 ]]; then
  printf 'temporary MySQL version=%s, want 8.0.46\n' "${version_scope_version}" >&2
  exit 92
fi
printf 'MYSQL_ENV image=mysql:8.0.46-oraclelinux9 version=%s binding=%s profile=%s\n' "${version_scope_version}" "${version_scope_binding}" "${version_scope_profile}"

version_scope_env=(
  ORDER_TEST_MYSQL_HOST=127.0.0.1
  ORDER_TEST_MYSQL_PORT="${version_scope_port}"
  ORDER_TEST_MYSQL_USER=root
  ORDER_TEST_MYSQL_PASSWORD="${version_scope_password}"
  ORDER_TEST_MYSQL_TLS_MODE=disabled
  ORDER_TEST_MYSQL_INSTANCE=order-mysql-w3
  ORDER_TEST_MYSQL_ISOLATED=YES
  GOPROXY=off
  GOTOOLCHAIN=go1.26.5
)
version_scope_historical_regex='^(TestCatalogSchemaIntegration|TestCatalogRepositoryAndHTTPIntegration|TestCatalogSingleStatementSnapshotIntegration|TestMiniprogramSessionMySQLIntegration|TestMiniprogramPhoneMySQLIntegration|TestMiniprogramPhoneStatusMySQLIntegration|TestMenuMySQLIntegration|TestStorefrontMySQL8Integration|TestHistoricalMigrationPrefixRequiresExactVersion)$'

cd "${version_scope_repo}"
case "${version_scope_profile}" in
  catalog-focused)
    env "${version_scope_env[@]}" go test -race ./services/api/internal/catalog -run '^(TestCatalogSchemaIntegration|TestHistoricalMigrationPrefixRequiresExactVersion)$' -count=1 -timeout=5m
    ;;
  storefront-focused)
    env "${version_scope_env[@]}" go test -race ./services/api/internal/storefront -run '^(TestStorefrontMySQL8Integration|TestHistoricalMigrationPrefixRequiresExactVersion)$' -count=1 -timeout=5m
    ;;
  migrate-focused)
    env "${version_scope_env[@]}" go test -race ./services/api/internal/migrate -run '^TestMySQL8Integration$' -count=1 -timeout=8m
    ;;
  w3)
    env "${version_scope_env[@]}" go test ./services/api/internal/catalog ./services/api/internal/identity ./services/api/internal/menu ./services/api/internal/storefront -run "${version_scope_historical_regex}" -count=1 -timeout=5m
    env "${version_scope_env[@]}" go test -race ./services/api/internal/catalog ./services/api/internal/identity ./services/api/internal/menu ./services/api/internal/storefront -run "${version_scope_historical_regex}" -count=1 -timeout=8m
    env "${version_scope_env[@]}" go test -race ./services/api/internal/migrate -run '^TestMySQL8Integration$' -count=1 -timeout=8m
    ;;
  adjacent)
    env "${version_scope_env[@]}" go test -race ./services/api/internal/migrate ./services/api/internal/merchantidentity ./services/api/internal/storefront ./services/api/internal/wechatpay -count=1 -timeout=10m
    ;;
  full)
    env "${version_scope_env[@]}" go test ./services/api/... -count=1 -timeout=15m
    env "${version_scope_env[@]}" go test -race ./services/api/... -count=1 -timeout=20m
    ;;
  *)
    printf 'unknown profile: %s\n' "${version_scope_profile}" >&2
    exit 64
    ;;
esac
