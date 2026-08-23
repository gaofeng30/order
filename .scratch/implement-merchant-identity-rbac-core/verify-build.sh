#!/usr/bin/env bash
set -euo pipefail

merchant_build_root=$(git rev-parse --show-toplevel)
merchant_build_output=$(mktemp -d /private/tmp/order-merchant-build.XXXXXX)

cleanup_merchant_build() {
  if [[ -d "${merchant_build_output}" ]] && [[ "$(realpath "${merchant_build_output}")" == /private/tmp/order-merchant-build.* ]]; then
    /usr/bin/trash "${merchant_build_output}"
  fi
}
trap cleanup_merchant_build EXIT

cd "${merchant_build_root}"
merchant_build_status_before=$(git status --porcelain)
GOPROXY=off GOTOOLCHAIN=go1.26.5 \
  go build -o "${merchant_build_output}/" ./services/api/...

test -x "${merchant_build_output}/order-api"
test -x "${merchant_build_output}/order-migrate"
test "$(git status --porcelain)" = "${merchant_build_status_before}"
