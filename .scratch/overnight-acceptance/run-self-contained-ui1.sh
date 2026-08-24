#!/usr/bin/env bash
set -euo pipefail

mode=${1:-}
case "${mode}" in
  be22-be26|pc-remaining) ;;
  *) printf 'usage: %s be22-be26|pc-remaining\n' "$0" >&2; exit 64 ;;
esac

repo_root=$(git rev-parse --show-toplevel)
candidate_sha=$(git rev-parse HEAD)
evidence_root="/private/tmp/order-final-ledger-${candidate_sha:0:12}-${mode}"
generated_paths=(
  ".scratch/overnight-pc-catalog-image"
  ".scratch/overnight-pc-import"
  ".scratch/overnight-pc"
  ".scratch/pc05-pc07-pc12-closure"
)

cleanup_generated() {
  local status=$?
  local target
  for target in "${generated_paths[@]}"; do
    if [[ -n "$(git ls-files -- "${target}")" ]]; then
      printf 'refusing to remove tracked evidence path %s\n' "${target}" >&2
      status=1
      continue
    fi
    find "${target}" -type f -delete 2>/dev/null || true
    find "${target}" -depth -type d -empty -delete 2>/dev/null || true
  done
  return "${status}"
}
trap cleanup_generated EXIT INT TERM

cd "${repo_root}"
[[ -z "$(git status --porcelain=v1 --untracked-files=all)" ]]
[[ "${candidate_sha}" =~ ^[0-9a-f]{40}$ ]]
mkdir -p "${evidence_root}"

case "${mode}" in
  be22-be26)
    ORDER_BE22_BE26_RECEIPT_PATH="${evidence_root}/receipt.json" \
      node tools/miniprogram-ui/run-ui1-composed-be22-be26.mjs
    ;;
  pc-remaining)
    node apps/web-admin/tests/composed-ui1-pc05-pc07-pc12-closure-runner.mjs
    ;;
esac

cleanup_generated
trap - EXIT INT TERM
[[ -z "$(git status --porcelain=v1 --untracked-files=all)" ]]
printf 'FINAL_SELF_CONTAINED_UI1=PASS mode=%s candidate=%s cleanup=0\n' "${mode}" "${candidate_sha}"
