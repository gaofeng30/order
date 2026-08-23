#!/usr/bin/env bash
set -euo pipefail

mysql_container="order-payment-observation-w3-$$"
mysql_password_file="$(mktemp /tmp/order-payment-observation-w3-password.XXXXXX)"

cleanup_mysql() {
  docker stop -t 1 "${mysql_container}" >/dev/null 2>&1 || true
  unlink "${mysql_password_file}" >/dev/null 2>&1 || true
}
trap cleanup_mysql EXIT

chmod 600 "${mysql_password_file}"
openssl rand -hex -out "${mysql_password_file}" 24
IFS= read -r mysql_password < "${mysql_password_file}"

docker run -d --rm \
  --name "${mysql_container}" \
  -p 127.0.0.1::3306 \
  -e MYSQL_ROOT_PASSWORD="${mysql_password}" \
  mysql:8.0.46-oraclelinux9 >/dev/null

mysql_ready="NO"
for _ in $(seq 1 90); do
  if docker exec "${mysql_container}" \
    mysqladmin --host=127.0.0.1 --protocol=tcp --user=root --password="${mysql_password}" \
    ping --silent >/dev/null 2>&1; then
    mysql_ready="YES"
    break
  fi
  sleep 1
done
if [[ "${mysql_ready}" != "YES" ]]; then
  printf 'W3_MYSQL_GATE=FAIL reason=tcp-readiness-timeout\n' >&2
  exit 1
fi

IFS= read -r mysql_binding < <(docker port "${mysql_container}" 3306/tcp)
mysql_port="${mysql_binding##*:}"

ORDER_TEST_MYSQL_HOST=127.0.0.1 \
ORDER_TEST_MYSQL_PORT="${mysql_port}" \
ORDER_TEST_MYSQL_USER=root \
ORDER_TEST_MYSQL_PASSWORD="${mysql_password}" \
ORDER_TEST_MYSQL_TLS_MODE=disabled \
ORDER_TEST_MYSQL_INSTANCE=order-mysql-w3 \
ORDER_TEST_MYSQL_ISOLATED=YES \
GOPROXY=off \
GOTOOLCHAIN=go1.26.5 \
go test -race \
  ./services/api/internal/migrate \
  ./services/api/internal/catalog \
  ./services/api/internal/identity \
  ./services/api/internal/menu \
  ./services/api/internal/merchantidentity \
  ./services/api/internal/storefront \
  ./services/api/internal/wechatpay \
  ./services/api/internal/paymentobservation \
  -count=1 -timeout=10m

printf 'W3_MYSQL_ADJACENT_GATE=PASS engine=mysql-8.0.46 suites=full-existing-matrix payment_core=pure-no-db order_persistence_suite=not-present\n'
