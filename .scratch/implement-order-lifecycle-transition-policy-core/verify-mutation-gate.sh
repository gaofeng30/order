#!/usr/bin/env bash
set -euo pipefail

transition_gate_repo=$(git rev-parse --show-toplevel)
transition_gate_script="${transition_gate_repo}/.scratch/implement-order-lifecycle-transition-policy-core/verify-mutations.sh"

set +e
(
  cd "${transition_gate_repo}"
  bash -c 'go() { return 2; }; export -f go; bash "$1"' _ "${transition_gate_script}"
) >/dev/null 2>&1
transition_gate_failure_exit=$?
set -e
if [[ ${transition_gate_failure_exit} -ne 82 ]]; then
  printf 'mutation infrastructure shield exit=%s, want 82\n' \
    "${transition_gate_failure_exit}" >&2
  exit 83
fi
printf 'MUTATION_INFRASTRUCTURE_SHIELD=PASS expected_exit=82\n'

GOPROXY=off GOTOOLCHAIN=go1.26.5 bash "${transition_gate_script}"
