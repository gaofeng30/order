#!/usr/bin/env bash
set -euo pipefail

static_repo=$(git rev-parse --show-toplevel)
static_base=19ca1e46e106f293070f0cdf820951e31107cba6
static_package=services/api/internal/staffidentity
static_mode=${1:-prestage}

cd "${static_repo}"

case "${static_mode}" in
  prestage|post-freeze)
    ;;
  *)
    printf 'usage: %s [prestage|post-freeze]\n' "$0" >&2
    exit 64
    ;;
esac

staged_files=$(git diff --cached --name-only "${static_base}" --)
unstaged_files=$(git diff --name-only --)
untracked_files=$(git ls-files --others --exclude-standard)
changed_files=$({
  printf '%s\n' "${staged_files}"
  printf '%s\n' "${unstaged_files}"
  printf '%s\n' "${untracked_files}"
} | sed '/^$/d' | sort -u)
if [[ -z "${changed_files}" ]]; then
  printf 'owned-path Gate found no change\n' >&2
  exit 70
fi

if [[ "${static_mode}" == post-freeze ]]; then
  if [[ -z "${staged_files}" ]]; then
    printf 'post-freeze Gate requires a non-empty staged change set\n' >&2
    exit 80
  fi
  if [[ -n "${unstaged_files}" ]]; then
    printf 'post-freeze Gate found unstaged tracked paths:\n%s\n' "${unstaged_files}" >&2
    exit 81
  fi
  if [[ -n "${untracked_files}" ]]; then
    printf 'post-freeze Gate found untracked paths:\n%s\n' "${untracked_files}" >&2
    exit 82
  fi
fi

while IFS= read -r changed_file; do
  [[ -n "${changed_file}" ]] || continue
  case "${changed_file}" in
    .scratch/implement-staff-identity-snapshot-core/*|services/api/internal/staffidentity/*|go.mod)
      ;;
    *)
      printf 'owned-path violation: %s\n' "${changed_file}" >&2
      exit 71
      ;;
  esac
  if [[ -f "${changed_file}" ]] && [[ $(tail -c 1 "${changed_file}" | wc -l | tr -d ' ') -ne 1 ]]; then
    printf 'missing final newline: %s\n' "${changed_file}" >&2
    exit 72
  fi
done <<<"${changed_files}"

git diff --quiet "${static_base}" -- go.sum
mod_numstat=$(git diff --numstat "${static_base}" -- go.mod)
if [[ "${mod_numstat}" != $'1\t1\tgo.mod' ]]; then
  printf 'go.mod diff is not exactly one add and one delete: %s\n' "${mod_numstat}" >&2
  exit 73
fi
if [[ $(git diff "${static_base}" -- go.mod | grep -Fxc -- $'+\tgolang.org/x/text v0.34.0') -ne 1 ]] || \
  [[ $(git diff "${static_base}" -- go.mod | grep -Fxc -- $'-\tgolang.org/x/text v0.34.0 // indirect') -ne 1 ]]; then
  printf 'go.mod x/text direct-promotion diff mismatch\n' >&2
  exit 74
fi

test -z "$(gofmt -l "${static_package}")"
git diff --cached --check "${static_base}" --
git diff --check --
if rg -n '[[:blank:]]+$' ${changed_files} >/dev/null; then
  printf 'trailing whitespace found in owned diff\n' >&2
  exit 75
fi
if rg -n -i '(authorization[[:space:]]*:|cookie[[:space:]]*:|begin [a-z ]*private key|api[_-]?v?3?[_-]?key[[:space:]]*[:=]|password[[:space:]]*[:=])' "${static_package}" >/dev/null; then
  printf 'sensitive token pattern found in staffidentity package\n' >&2
  exit 76
fi
if rg -n '\b(UserID|RecordID|Source|DiscountRate)\b' "${static_package}" >/dev/null; then
  printf 'forbidden pass-through field found in staffidentity package\n' >&2
  exit 77
fi
if rg -n '\b(os|time|io|net/http|database/sql|math/rand)\b' "${static_package}"/*.go >/dev/null; then
  printf 'forbidden I/O, environment, time, or random dependency found\n' >&2
  exit 78
fi

imports=$(GOPROXY=off GOTOOLCHAIN=go1.26.5 go list -f '{{join .Imports "\n"}}' ./services/api/internal/staffidentity | sort)
expected_imports=$'strings\nunicode/utf8\ngolang.org/x/text/unicode/norm\ngolang.org/x/text/width'
expected_imports=$(printf '%s\n' "${expected_imports}" | sort)
if [[ "${imports}" != "${expected_imports}" ]]; then
  printf 'production import set mismatch\n' >&2
  exit 79
fi

printf 'STATIC_GATE=PASS mode=%s owned_files=%s staged=%s unstaged=%s untracked=%s go_mod=direct-x-text-v0.34.0 go_sum=unchanged\n' \
  "${static_mode}" \
  "$(printf '%s\n' "${changed_files}" | sed '/^$/d' | wc -l | tr -d ' ')" \
  "$(printf '%s\n' "${staged_files}" | sed '/^$/d' | wc -l | tr -d ' ')" \
  "$(printf '%s\n' "${unstaged_files}" | sed '/^$/d' | wc -l | tr -d ' ')" \
  "$(printf '%s\n' "${untracked_files}" | sed '/^$/d' | wc -l | tr -d ' ')"
