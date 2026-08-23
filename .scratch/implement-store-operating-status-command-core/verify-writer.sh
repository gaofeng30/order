#!/usr/bin/env bash
set -euo pipefail

writer_repo=$(git rev-parse --show-toplevel)
writer_base=8cae09d5bc3e659d8851e7588835e579101058ac
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
    printf 'writer source has unstaged changes\n' >&2
    exit 91
  fi
  if [[ -n $(git ls-files --others --exclude-standard) ]]; then
    printf 'writer source has untracked files\n' >&2
    exit 92
  fi
  while IFS= read -r writer_path; do
    [[ -z "${writer_path}" ]] && continue
    case "${writer_path}" in
      .scratch/implement-store-operating-status-command-core/*|services/api/internal/storestatus/*) ;;
      *)
        printf 'out-of-scope path: %s\n' "${writer_path}" >&2
        exit 93
        ;;
    esac
  done < <(writer_changed_paths)
  git diff --check "${writer_base}"...HEAD
  git diff --cached --check
  git diff --check
}

verify_writer_hygiene() {
  test -z "$(gofmt -l services/api/internal/storestatus)"
  bash -n .scratch/implement-store-operating-status-command-core/*.sh

  writer_mutation_files=$(rg -l -i \
    '(UPDATE[[:space:]]+storefront_settings|INSERT[[:space:]]+INTO[[:space:]]+storefront_settings|DELETE[[:space:]]+FROM[[:space:]]+storefront_settings|REPLACE[[:space:]]+INTO[[:space:]]+storefront_settings)' \
    services/api --glob '*.go' --glob '!**/*_test.go' || true)
  if [[ "${writer_mutation_files}" != "services/api/internal/storestatus/core.go" ]]; then
    printf 'storefront_settings production mutation ownership failed\n' >&2
    exit 94
  fi
  writer_update_count=$(rg -o -i 'UPDATE[[:space:]]+storefront_settings' services/api/internal/storestatus/core.go | wc -l | tr -d ' ')
  if [[ "${writer_update_count}" != 1 ]]; then
    printf 'storefront_settings update statement count is not one\n' >&2
    exit 95
  fi

  if rg -i \
    'PASSWORD=["\x27][^$]|-----BEGIN[[:space:]][A-Z ]+PRIVATE KEY|Authorization:[[:space:]]*[^<]|Cookie:[[:space:]]*[^<]|api[_-]?key[[:space:]]*[:=]' \
    -g '!verify-writer.sh' \
    .scratch/implement-store-operating-status-command-core services/api/internal/storestatus >/dev/null; then
    printf 'sensitive literal scan failed\n' >&2
    exit 96
  fi
}

verify_writer_source_state
verify_writer_hygiene
writer_tree_before=$(git write-tree)
writer_status_before=$(git status --porcelain)

GOPROXY=off GOTOOLCHAIN=go1.26.5 go test ./services/api/internal/storestatus -count=1
GOPROXY=off GOTOOLCHAIN=go1.26.5 go test -race ./services/api/internal/storestatus -count=20
bash .scratch/implement-store-operating-status-command-core/verify-mutation-gate.sh
bash .scratch/implement-store-operating-status-command-core/verify-mysql.sh \
  bash .scratch/implement-store-operating-status-command-core/verify-w3.sh
GOPROXY=off GOTOOLCHAIN=go1.26.5 go test ./services/api/... -count=1
GOPROXY=off GOTOOLCHAIN=go1.26.5 go test -race ./services/api/... -count=1
GOPROXY=off GOTOOLCHAIN=go1.26.5 go vet ./services/api/...
bash .scratch/implement-store-operating-status-command-core/verify-build.sh
GOPROXY=off GOTOOLCHAIN=go1.26.5 bash services/api/scripts/smoke.sh

verify_writer_source_state
verify_writer_hygiene
test "$(git write-tree)" = "${writer_tree_before}"
test "$(git status --porcelain)" = "${writer_status_before}"
if [[ -n $(docker ps --filter 'name=order-store-status-w3-' --format '{{.Names}}') ]]; then
  printf 'temporary store status MySQL container remains\n' >&2
  exit 97
fi
if find /private/tmp -maxdepth 1 -name 'order-store-status-w3-password.*' -print -quit | grep -q .; then
  printf 'temporary store status credential file remains\n' >&2
  exit 98
fi
if find /private/tmp -maxdepth 1 -name 'order-store-status-mutations.*' -print -quit | grep -q .; then
  printf 'temporary store status mutation directory remains\n' >&2
  exit 99
fi

printf 'WRITER_GATE=PASS base_sha=%s head_sha=%s tree=%s\n' \
  "${writer_base}" "$(git rev-parse HEAD)" "${writer_tree_before}"
