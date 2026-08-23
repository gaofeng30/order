#!/usr/bin/env bash
set -euo pipefail

pickup_mysql_repo=$(git rev-parse --show-toplevel)
pickup_mysql_container="order-pickup-options-$$"
pickup_mysql_password_file=$(mktemp /private/tmp/order-pickup-options-password.XXXXXX)

cleanup_pickup_mysql() {
  docker stop -t 1 "${pickup_mysql_container}" >/dev/null 2>&1 || true
  unlink "${pickup_mysql_password_file}" >/dev/null 2>&1 || true
}
trap cleanup_pickup_mysql EXIT

chmod 600 "${pickup_mysql_password_file}"
openssl rand -hex -out "${pickup_mysql_password_file}" 24
IFS= read -r pickup_mysql_password < "${pickup_mysql_password_file}"

docker run -d --rm \
  --name "${pickup_mysql_container}" \
  -p 127.0.0.1::3306 \
  -e MYSQL_ROOT_PASSWORD="${pickup_mysql_password}" \
  mysql:8.0.46-oraclelinux9 >/dev/null

pickup_mysql_ready=NO
for _ in {1..90}; do
  if docker exec "${pickup_mysql_container}" mysqladmin --host=127.0.0.1 --protocol=tcp --user=root --password="${pickup_mysql_password}" ping --silent >/dev/null 2>&1; then
    pickup_mysql_ready=YES
    break
  fi
  sleep 1
done
if [[ "${pickup_mysql_ready}" != YES ]]; then
  printf 'temporary pickup-options MySQL did not become TCP-ready\n' >&2
  exit 90
fi

IFS= read -r pickup_mysql_binding < <(docker port "${pickup_mysql_container}" 3306/tcp)
if [[ "${pickup_mysql_binding}" != 127.0.0.1:* ]]; then
  printf 'temporary pickup-options MySQL is not loopback-only: %s\n' "${pickup_mysql_binding}" >&2
  exit 91
fi
pickup_mysql_port=${pickup_mysql_binding##*:}
pickup_mysql_version=$(docker exec "${pickup_mysql_container}" mysql --batch --skip-column-names --user=root --password="${pickup_mysql_password}" -e 'SELECT VERSION()' 2>/dev/null)
if [[ "${pickup_mysql_version}" != 8.0.46 ]]; then
  printf 'temporary pickup-options MySQL version=%s, want 8.0.46\n' "${pickup_mysql_version}" >&2
  exit 92
fi
printf 'MYSQL_ENV image=mysql:8.0.46-oraclelinux9 version=%s binding=%s\n' "${pickup_mysql_version}" "${pickup_mysql_binding}"

cd "${pickup_mysql_repo}"
env \
  ORDER_TEST_MYSQL_HOST=127.0.0.1 \
  ORDER_TEST_MYSQL_PORT="${pickup_mysql_port}" \
  ORDER_TEST_MYSQL_USER=root \
  ORDER_TEST_MYSQL_PASSWORD="${pickup_mysql_password}" \
  ORDER_TEST_MYSQL_TLS_MODE=disabled \
  ORDER_TEST_MYSQL_INSTANCE=order-mysql-w3 \
  ORDER_TEST_MYSQL_ISOLATED=YES \
  GOPROXY=off \
  GOTOOLCHAIN=go1.26.5 \
  bash services/api/scripts/menu-integration.sh
