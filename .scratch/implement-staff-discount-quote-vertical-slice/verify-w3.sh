#!/usr/bin/env bash
set -euo pipefail

quote_w3_base=1657aa9451f612e4605fabd084ccab07542ac81a
quote_w3_repo=$(git rev-parse --show-toplevel)
quote_w3_required=(
  ORDER_TEST_MYSQL_HOST
  ORDER_TEST_MYSQL_PORT
  ORDER_TEST_MYSQL_USER
  ORDER_TEST_MYSQL_PASSWORD
  ORDER_TEST_MYSQL_TLS_MODE
  ORDER_TEST_MYSQL_INSTANCE
  ORDER_TEST_MYSQL_ISOLATED
)

cd "${quote_w3_repo}"
git cat-file -e "${quote_w3_base}^{commit}"
for quote_w3_key in "${quote_w3_required[@]}"; do
  if [[ -z "${!quote_w3_key:-}" ]]; then
    printf 'missing isolated MySQL environment: %s\n' "${quote_w3_key}" >&2
    exit 1
  fi
done
if [[ "${ORDER_TEST_MYSQL_INSTANCE}" != order-mysql-w3 || "${ORDER_TEST_MYSQL_ISOLATED}" != YES ]]; then
  printf 'refusing non-isolated quote MySQL gate\n' >&2
  exit 1
fi

GOWORK=off GOCACHE=/private/tmp/order-tx02-final-gocache GOPROXY=off GOTOOLCHAIN=go1.26.5 \
  go test -race ./services/api/internal/quote -run '^TestQuoteMySQL8Integration$' -count=1 -timeout=10m
