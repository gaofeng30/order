#!/usr/bin/env bash
set -euo pipefail

quote_mysql_base=1657aa9451f612e4605fabd084ccab07542ac81a
quote_mysql_repo=$(git rev-parse --show-toplevel)
quote_mysql_container="order-quote-w3-$$"
quote_mysql_password_file=$(mktemp /private/tmp/order-quote-w3-password.XXXXXX)

cleanup_quote_mysql() {
  docker stop -t 1 "${quote_mysql_container}" >/dev/null 2>&1 || true
  unlink "${quote_mysql_password_file}" >/dev/null 2>&1 || true
}
trap cleanup_quote_mysql EXIT

cd "${quote_mysql_repo}"
git cat-file -e "${quote_mysql_base}^{commit}"
chmod 600 "${quote_mysql_password_file}"
openssl rand -hex -out "${quote_mysql_password_file}" 24
IFS= read -r quote_mysql_password < "${quote_mysql_password_file}"

docker run -d --rm \
  --name "${quote_mysql_container}" \
  -p 127.0.0.1::3306 \
  -e MYSQL_ROOT_PASSWORD="${quote_mysql_password}" \
  mysql:8.0.46-oraclelinux9 >/dev/null

quote_mysql_ready=NO
for _ in {1..60}; do
  if docker exec "${quote_mysql_container}" mysqladmin --host=127.0.0.1 --protocol=tcp --user=root --password="${quote_mysql_password}" ping --silent >/dev/null 2>&1; then
    quote_mysql_ready=YES
    break
  fi
  sleep 1
done
if [[ "${quote_mysql_ready}" != YES ]]; then
  printf 'temporary quote MySQL did not become ready\n' >&2
  exit 1
fi

IFS= read -r quote_mysql_binding < <(docker port "${quote_mysql_container}" 3306/tcp)
quote_mysql_port=${quote_mysql_binding##*:}
if (( $# == 0 )); then
  quote_mysql_command=(bash .scratch/implement-staff-discount-quote-vertical-slice/verify-w3.sh)
else
  quote_mysql_command=("$@")
fi

env \
  ORDER_TEST_MYSQL_HOST=127.0.0.1 \
  ORDER_TEST_MYSQL_PORT="${quote_mysql_port}" \
  ORDER_TEST_MYSQL_USER=root \
  ORDER_TEST_MYSQL_PASSWORD="${quote_mysql_password}" \
  ORDER_TEST_MYSQL_TLS_MODE=disabled \
  ORDER_TEST_MYSQL_INSTANCE=order-mysql-w3 \
  ORDER_TEST_MYSQL_ISOLATED=YES \
  GOWORK=off \
  GOPROXY=off \
  GOTOOLCHAIN=go1.26.5 \
  "${quote_mysql_command[@]}"
