#!/usr/bin/env bash
set -euo pipefail

merchant_repo_root=$(git rev-parse --show-toplevel)
merchant_container="order-merchant-identity-w3-$$"
merchant_password_file=$(mktemp /private/tmp/order-merchant-identity-w3-password.XXXXXX)

cleanup_merchant_mysql() {
  docker stop -t 1 "${merchant_container}" >/dev/null 2>&1 || true
  unlink "${merchant_password_file}" >/dev/null 2>&1 || true
}
trap cleanup_merchant_mysql EXIT

chmod 600 "${merchant_password_file}"
openssl rand -hex -out "${merchant_password_file}" 24
IFS= read -r merchant_password < "${merchant_password_file}"

docker run -d --rm \
  --name "${merchant_container}" \
  -p 127.0.0.1::3306 \
  -e MYSQL_ROOT_PASSWORD="${merchant_password}" \
  mysql:8.0.46-oraclelinux9 >/dev/null

merchant_ready=NO
for _ in {1..60}; do
  if docker exec "${merchant_container}" mysqladmin --host=127.0.0.1 --protocol=tcp --user=root --password="${merchant_password}" ping --silent >/dev/null 2>&1; then
    merchant_ready=YES
    break
  fi
  sleep 1
done
if [[ "${merchant_ready}" != YES ]]; then
  printf 'temporary merchant identity MySQL did not become ready\n' >&2
  exit 1
fi

IFS= read -r merchant_binding < <(docker port "${merchant_container}" 3306/tcp)
merchant_port=${merchant_binding##*:}

cd "${merchant_repo_root}"
ORDER_TEST_MYSQL_HOST=127.0.0.1 \
ORDER_TEST_MYSQL_PORT="${merchant_port}" \
ORDER_TEST_MYSQL_USER=root \
ORDER_TEST_MYSQL_PASSWORD="${merchant_password}" \
ORDER_TEST_MYSQL_TLS_MODE=disabled \
ORDER_TEST_MYSQL_INSTANCE=order-mysql-w3 \
ORDER_TEST_MYSQL_ISOLATED=YES \
GOPROXY=off GOTOOLCHAIN=go1.26.5 \
go test -race ./services/api/internal/merchantidentity -run '^TestMerchantIdentityMySQL8Integration$' -count=1 -timeout=5m
