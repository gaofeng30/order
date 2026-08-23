#!/usr/bin/env bash
set -euo pipefail

mutation_gate_repo=$(git rev-parse --show-toplevel)
mutation_gate_script="${mutation_gate_repo}/.scratch/implement-staff-identity-snapshot-core/verify-mutations.sh"
mutation_gate_mode=${1:-run}

run_rejection_self_test() {
  local self_test_name=$1
  local expected_exit=$2
  local self_test_output self_test_exit

  set +e
  self_test_output=$(
    cd "${mutation_gate_repo}"
    MUTATION_SELF_TEST="${self_test_name}" bash "${mutation_gate_script}" 2>&1
  )
  self_test_exit=$?
  set -e
  if [[ ${self_test_exit} -ne ${expected_exit} ]]; then
    printf 'mutation self-test failed: %s exit=%s want=%s output=%s\n' \
      "${self_test_name}" "${self_test_exit}" "${expected_exit}" "${self_test_output}" >&2
    exit 83
  fi
  printf 'MUTATION_SELF_TEST=PASS name=%s rejected_exit=%s\n' "${self_test_name}" "${self_test_exit}"
}

(
  cd "${mutation_gate_repo}"
  bash "${mutation_gate_script}" check-bindings
)
run_rejection_self_test missing-source 79
run_rejection_self_test duplicate-source 79
run_rejection_self_test go-exit2 82

case "${mutation_gate_mode}" in
  self-test)
    printf 'MUTATION_INFRASTRUCTURE_SHIELD=PASS self_tests=3 real_mutations=NOT_RUN\n'
    ;;
  run)
    (
      cd "${mutation_gate_repo}"
      bash "${mutation_gate_script}" run
    )
    ;;
  *)
    printf 'usage: %s [self-test|run]\n' "$0" >&2
    exit 64
    ;;
esac
