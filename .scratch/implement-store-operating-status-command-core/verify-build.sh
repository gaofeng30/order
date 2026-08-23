#!/usr/bin/env bash
set -euo pipefail

store_status_build_repo=$(git rev-parse --show-toplevel)
store_status_build_output=$(mktemp -d /private/tmp/order-store-status-build.XXXXXX)

cleanup_store_status_build() {
  if [[ -d "${store_status_build_output}" ]] && [[ "$(realpath "${store_status_build_output}")" == /private/tmp/order-store-status-build.* ]]; then
    /usr/bin/trash "${store_status_build_output}"
  fi
}
trap cleanup_store_status_build EXIT

cd "${store_status_build_repo}"
store_status_build_state_before=$(git status --porcelain)
GOPROXY=off GOTOOLCHAIN=go1.26.5 \
  go build -o "${store_status_build_output}/" ./services/api/...

test -x "${store_status_build_output}/order-api"
test -x "${store_status_build_output}/order-migrate"
test "$(git status --porcelain)" = "${store_status_build_state_before}"
