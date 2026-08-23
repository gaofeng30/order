#!/usr/bin/env bash
set -euo pipefail

quote_build_repo=$(git rev-parse --show-toplevel)
quote_build_output=$(mktemp -d /private/tmp/order-quote-build.XXXXXX)

cleanup_quote_build() {
  if [[ -d "${quote_build_output}" ]] && [[ "$(realpath "${quote_build_output}")" == /private/tmp/order-quote-build.* ]]; then
    find "${quote_build_output}" -depth -delete
  fi
}
trap cleanup_quote_build EXIT

cd "${quote_build_repo}"
quote_build_state_before=$(git status --porcelain)
GOWORK=off GOCACHE=/private/tmp/order-tx02-final-gocache GOPROXY=off GOTOOLCHAIN=go1.26.5 \
  go build -o "${quote_build_output}/" ./services/api/...

test -x "${quote_build_output}/order-api"
test -x "${quote_build_output}/order-migrate"
test "$(git status --porcelain)" = "${quote_build_state_before}"
