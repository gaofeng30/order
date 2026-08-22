#!/usr/bin/env bash
set -euo pipefail

storefront_repo_root=$(git rev-parse --show-toplevel)
storefront_container="order-storefront-w3-$$"
storefront_password_file=$(mktemp /private/tmp/order-storefront-w3-password.XXXXXX)

cleanup_storefront_mysql() {
  docker stop -t 1 "${storefront_container}" >/dev/null 2>&1 || true
  unlink "${storefront_password_file}" >/dev/null 2>&1 || true
}
trap cleanup_storefront_mysql EXIT

chmod 600 "${storefront_password_file}"
openssl rand -hex -out "${storefront_password_file}" 24
IFS= read -r storefront_password < "${storefront_password_file}"

docker run -d --rm \
  --name "${storefront_container}" \
  -p 127.0.0.1::3306 \
  -e MYSQL_ROOT_PASSWORD="${storefront_password}" \
  mysql:8.0.46-oraclelinux9 >/dev/null

storefront_ready=NO
for _ in {1..60}; do
  if docker exec "${storefront_container}" mysqladmin --user=root --password="${storefront_password}" ping --silent >/dev/null 2>&1; then
    storefront_ready=YES
    break
  fi
  sleep 1
done
if [[ "${storefront_ready}" != YES ]]; then
  printf 'temporary storefront MySQL did not become ready\n' >&2
  exit 1
fi

IFS= read -r storefront_binding < <(docker port "${storefront_container}" 3306/tcp)
storefront_port=${storefront_binding##*:}

cd "${storefront_repo_root}"
ORDER_TEST_MYSQL_HOST=127.0.0.1 \
ORDER_TEST_MYSQL_PORT="${storefront_port}" \
ORDER_TEST_MYSQL_USER=root \
ORDER_TEST_MYSQL_PASSWORD="${storefront_password}" \
ORDER_TEST_MYSQL_TLS_MODE=disabled \
ORDER_TEST_MYSQL_INSTANCE=order-mysql-w3 \
ORDER_TEST_MYSQL_ISOLATED=YES \
GOPROXY=off GOTOOLCHAIN=go1.26.5 \
go test -race ./services/api/internal/storefront -run '^TestStorefrontMySQL8Integration$' -count=1 -timeout=3m
