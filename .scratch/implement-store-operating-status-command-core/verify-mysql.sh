#!/usr/bin/env bash
set -euo pipefail

store_status_repo=$(git rev-parse --show-toplevel)
store_status_container="order-store-status-w3-$$"
store_status_password_file=$(mktemp /private/tmp/order-store-status-w3-password.XXXXXX)

cleanup_store_status_mysql() {
  docker stop -t 1 "${store_status_container}" >/dev/null 2>&1 || true
  unlink "${store_status_password_file}" >/dev/null 2>&1 || true
}
trap cleanup_store_status_mysql EXIT

chmod 600 "${store_status_password_file}"
openssl rand -hex -out "${store_status_password_file}" 24
IFS= read -r store_status_password < "${store_status_password_file}"

docker run -d --rm \
  --name "${store_status_container}" \
  -p 127.0.0.1::3306 \
  -e MYSQL_ROOT_PASSWORD="${store_status_password}" \
  mysql:8.0.46-oraclelinux9 >/dev/null

store_status_ready=NO
for _ in {1..60}; do
  if docker exec "${store_status_container}" mysqladmin --host=127.0.0.1 --protocol=tcp --user=root --password="${store_status_password}" ping --silent >/dev/null 2>&1; then
    store_status_ready=YES
    break
  fi
  sleep 1
done
if [[ "${store_status_ready}" != YES ]]; then
  printf 'temporary store-status MySQL did not become ready\n' >&2
  exit 1
fi

IFS= read -r store_status_binding < <(docker port "${store_status_container}" 3306/tcp)
store_status_port=${store_status_binding##*:}

cd "${store_status_repo}"
if (( $# == 0 )); then
  store_status_command=(go test -race ./services/api/internal/storestatus -count=1 -timeout=5m)
else
  store_status_command=("$@")
fi
env \
  ORDER_TEST_MYSQL_HOST=127.0.0.1 \
  ORDER_TEST_MYSQL_PORT="${store_status_port}" \
  ORDER_TEST_MYSQL_USER=root \
  ORDER_TEST_MYSQL_PASSWORD="${store_status_password}" \
  ORDER_TEST_MYSQL_TLS_MODE=disabled \
  ORDER_TEST_MYSQL_INSTANCE=order-mysql-w3 \
  ORDER_TEST_MYSQL_ISOLATED=YES \
  GOPROXY=off \
  GOTOOLCHAIN=go1.26.5 \
  "${store_status_command[@]}"
