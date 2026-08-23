#!/usr/bin/env bash
set -euo pipefail

writer_repo=$(git rev-parse --show-toplevel)
writer_base=f3c4efa4cd665652d93d5da76f92d18c4bdc59ac
cd "${writer_repo}"

writer_changed_paths() {
  {
    git diff --name-only "${writer_base}"...HEAD
    git diff --cached --name-only
    git diff --name-only
    git ls-files --others --exclude-standard
  } | LC_ALL=C sort -u
}

verify_writer_source_state() {
  git merge-base --is-ancestor "${writer_base}" HEAD
  if ! git diff --quiet; then
    printf 'post-freeze writer source has unstaged changes\n' >&2
    exit 91
  fi
  if [[ -n $(git ls-files --others --exclude-standard) ]]; then
    printf 'post-freeze writer source has untracked files\n' >&2
    exit 92
  fi

  while IFS= read -r writer_owned_file; do
    [[ -z "${writer_owned_file}" ]] && continue
    case "${writer_owned_file}" in
      .scratch/implement-pickup-options-read-api/*|services/api/internal/menu/handler.go|services/api/internal/menu/pickup_options.go|services/api/internal/menu/pickup_options_test.go|services/api/internal/menu/mysql_integration_test.go|services/api/internal/httpapi/pickup_options_test.go) ;;
      *)
        printf 'out-of-scope file: %s\n' "${writer_owned_file}" >&2
        exit 97
        ;;
    esac
  done < <(writer_changed_paths)

  git diff --check "${writer_base}"...HEAD
  git diff --cached --check
  git diff --check
}

verify_writer_source_state
writer_tree_before=$(git ls-files --stage | LC_ALL=C sort | shasum -a 256 | awk '{print $1}')

.scratch/implement-pickup-options-read-api/replay-rgr.sh
GOPROXY=off GOTOOLCHAIN=go1.26.5 go test ./services/api/internal/menu ./services/api/internal/httpapi -count=1
GOPROXY=off GOTOOLCHAIN=go1.26.5 go test -race ./services/api/internal/menu ./services/api/internal/httpapi -count=1
GOPROXY=off GOTOOLCHAIN=go1.26.5 go test ./services/api/internal/menu ./services/api/internal/httpapi -count=20
GOPROXY=off GOTOOLCHAIN=go1.26.5 .scratch/implement-pickup-options-read-api/verify-mutation-gate.sh
.scratch/implement-pickup-options-read-api/verify-mysql.sh
GOPROXY=off GOTOOLCHAIN=go1.26.5 go test ./services/api/... -count=1
GOPROXY=off GOTOOLCHAIN=go1.26.5 go test -race ./services/api/... -count=1
GOPROXY=off GOTOOLCHAIN=go1.26.5 go vet ./services/api/...
GOPROXY=off GOTOOLCHAIN=go1.26.5 go build ./services/api/...
GOPROXY=off GOTOOLCHAIN=go1.26.5 bash services/api/scripts/smoke.sh
.scratch/implement-pickup-options-read-api/verify-ui1.sh

if [[ -e tools/miniprogram-ui/node_modules || -L tools/miniprogram-ui/node_modules ]]; then
  printf 'temporary UI1 runtime asset remains\n' >&2
  exit 94
fi
if [[ -n $(docker ps --filter 'name=order-pickup-options-' --format '{{.Names}}') ]]; then
  printf 'temporary pickup-options MySQL container remains\n' >&2
  exit 95
fi
if find /private/tmp -maxdepth 1 -name 'order-pickup-options-password.*' -print -quit | grep -q .; then
  printf 'temporary pickup-options credential file remains\n' >&2
  exit 96
fi

verify_writer_source_state
writer_tree_after=$(git ls-files --stage | LC_ALL=C sort | shasum -a 256 | awk '{print $1}')
if [[ "${writer_tree_before}" != "${writer_tree_after}" ]]; then
  printf 'post-freeze staged source tree changed during Writer Gate\n' >&2
  exit 99
fi
test -z "$(gofmt -l services/api/internal/menu/handler.go services/api/internal/menu/pickup_options.go services/api/internal/menu/pickup_options_test.go services/api/internal/menu/mysql_integration_test.go services/api/internal/httpapi/pickup_options_test.go)"
bash -n .scratch/implement-pickup-options-read-api/*.sh
if rg -n 'PASSWORD=["'\''][^$]|-----BEGIN|Authorization:[[:space:]]*[^<]|Cookie:[[:space:]]*[^<]' \
  -g '!verify-writer.sh' \
  .scratch/implement-pickup-options-read-api \
  services/api/internal/menu/pickup_options.go \
  services/api/internal/menu/pickup_options_test.go \
  services/api/internal/menu/mysql_integration_test.go \
  services/api/internal/httpapi/pickup_options_test.go; then
  printf 'sensitive literal scan failed\n' >&2
  exit 98
fi

printf 'WRITER_GATE=PASS base_sha=%s head_sha=%s source_tree_sha256=%s\n' \
  "${writer_base}" "$(git rev-parse HEAD)" "${writer_tree_after}"
