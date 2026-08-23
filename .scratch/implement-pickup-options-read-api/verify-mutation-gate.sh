#!/usr/bin/env bash
set -euo pipefail

mutation_gate_repo=$(git rev-parse --show-toplevel)
mutation_gate_script="${mutation_gate_repo}/.scratch/implement-pickup-options-read-api/verify-mutations.sh"

set +e
(
  cd "${mutation_gate_repo}"
  bash -c 'go() { return 2; }; export -f go; bash "$1"' _ "${mutation_gate_script}"
) >/dev/null 2>&1
mutation_gate_failure_exit=$?
set -e
if [[ ${mutation_gate_failure_exit} -ne 82 ]]; then
  printf 'mutation infrastructure shield exit=%s, want 82\n' "${mutation_gate_failure_exit}" >&2
  exit 83
fi
printf 'MUTATION_INFRASTRUCTURE_SHIELD=PASS expected_exit=82\n'

GOPROXY=off GOTOOLCHAIN=go1.26.5 bash "${mutation_gate_script}"
