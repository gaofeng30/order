#!/usr/bin/env bash
set -euo pipefail

required=(
  ORDER_TEST_MYSQL_HOST
  ORDER_TEST_MYSQL_PORT
  ORDER_TEST_MYSQL_USER
  ORDER_TEST_MYSQL_PASSWORD
  ORDER_TEST_MYSQL_TLS_MODE
  ORDER_TEST_MYSQL_INSTANCE
  ORDER_TEST_MYSQL_ISOLATED
)

for variable in "${required[@]}"; do
  if [[ -z "${!variable:-}" ]]; then
    printf 'menu integration environment is incomplete: %s\n' "${variable}" >&2
    exit 1
  fi
done

if [[ "${ORDER_TEST_MYSQL_INSTANCE}" != "order-mysql-w3" || "${ORDER_TEST_MYSQL_ISOLATED}" != "YES" ]]; then
  printf 'menu integration environment ownership mismatch\n' >&2
  exit 1
fi

GOTOOLCHAIN="${GOTOOLCHAIN:-go1.26.5}" GOPROXY="${GOPROXY:-off}" \
  go test -race ./services/api/internal/menu ./services/api/internal/catalog ./services/api/internal/httpapi \
  -run '^(TestMenuMySQLIntegration|TestCatalog.*Integration|TestFoundationAndCatalogIntegration)$' -count=1 -timeout=3m
