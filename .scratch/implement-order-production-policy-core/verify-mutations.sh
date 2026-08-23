#!/usr/bin/env bash
set -euo pipefail

mutation_repo=$(git rev-parse --show-toplevel)
mutation_root=$(mktemp -d /private/tmp/order-production-mutations.XXXXXX)

cleanup_mutations() {
  rm -rf "${mutation_root}"
}
trap cleanup_mutations EXIT

run_mutation() {
  local mutation_name=$1
  local mutation_original=$2
  local mutation_expression=$3
  local mutation_expected=$4
  local mutation_test=$5
  local mutation_failure_marker=$6
  local mutation_copy="${mutation_root}/${mutation_name}"
  local mutation_policy="${mutation_copy}/services/api/internal/orderproduction/policy.go"

  mkdir -p "${mutation_copy}/services/api/internal/orderproduction"
  cp "${mutation_repo}/go.mod" "${mutation_repo}/go.sum" "${mutation_copy}/"
  cp "${mutation_repo}/services/api/internal/orderproduction/"*.go \
    "${mutation_copy}/services/api/internal/orderproduction/"

  local mutation_original_count
  mutation_original_count=$(grep -Foc "${mutation_original}" "${mutation_policy}" || true)
  if [[ ${mutation_original_count} -ne 1 ]]; then
    printf 'mutation source count=%s, want 1: %s\n' \
      "${mutation_original_count}" "${mutation_name}" >&2
    exit 79
  fi
  perl -0pi -e "${mutation_expression}" "${mutation_policy}"
  if grep -Fq "${mutation_original}" "${mutation_policy}" || \
    ! grep -Fq "${mutation_expected}" "${mutation_policy}"; then
    printf 'mutation was not applied: %s\n' "${mutation_name}" >&2
    exit 80
  fi

  set +e
  (
    cd "${mutation_copy}"
    GOPROXY=off GOTOOLCHAIN=go1.26.5 go test ./services/api/internal/orderproduction \
      -run "${mutation_test}" -count=1
  ) >"${mutation_copy}/test.log" 2>&1
  local mutation_exit=$?
  set -e
  if [[ ${mutation_exit} -eq 0 ]]; then
    printf 'mutation survived: %s test=%s\n' \
      "${mutation_name}" "${mutation_test}" >&2
    exit 81
  fi
  if [[ ${mutation_exit} -ne 1 ]] || \
    ! grep -Fq -- "${mutation_failure_marker}" "${mutation_copy}/test.log"; then
    printf 'mutation test did not reach the expected behavior assertion: %s exit=%s\n' \
      "${mutation_name}" "${mutation_exit}" >&2
    exit 82
  fi
  printf 'MUTATION_KILLED name=%s exit=1 marker=%s\n' \
    "${mutation_name}" "${mutation_failure_marker}"
}

run_mutation \
  initial_less_than_to_less_equal \
  'pickupAt.Sub(paymentSucceededAt) < productionLeadTime' \
  's/pickupAt\.Sub\(paymentSucceededAt\) < productionLeadTime/pickupAt.Sub(paymentSucceededAt) <= productionLeadTime/' \
  'pickupAt.Sub(paymentSucceededAt) <= productionLeadTime' \
  '^TestInitialStateExactlyThirtyMinutesBeforePickupStartsReserved$' \
  '--- FAIL: TestInitialStateExactlyThirtyMinutesBeforePickupStartsReserved'

run_mutation \
  advance_greater_equal_to_greater \
  'if observedAt.Before(threshold) {' \
  's/if observedAt\.Before\(threshold\) \{/if !observedAt.After(threshold) {/' \
  'if !observedAt.After(threshold) {' \
  '^TestAdvanceAtThresholdStartsPreparing$' \
  '--- FAIL: TestAdvanceAtThresholdStartsPreparing'

run_mutation \
  insufficient_time_starts_reserved \
  'return StatePreparing, nil' \
  's/return StatePreparing, nil/return StateReserved, nil/' \
  'return StateReserved, nil' \
  '^TestInitialStateLessThanThirtyMinutesBeforePickup$' \
  '--- FAIL: TestInitialStateLessThanThirtyMinutesBeforePickup'

run_mutation \
  successor_rolls_back \
  'return Decision{State: current}, nil' \
  's/return Decision\{State: current\}, nil/return Decision{State: StatePreparing, Changed: true}, nil/' \
  'return Decision{State: StatePreparing, Changed: true}, nil' \
  '^TestAdvanceDoesNotMoveSuccessorStates$' \
  '--- FAIL: TestAdvanceDoesNotMoveSuccessorStates'

run_mutation \
  invalid_state_is_accepted \
  'return Decision{}, &Error{kind: ErrorInvalidState}' \
  's/return Decision\{\}, &Error\{kind: ErrorInvalidState\}/return Decision{State: current}, nil/' \
  'return Decision{State: current}, nil' \
  '^TestAdvanceRejectsInvalidState$' \
  '--- FAIL: TestAdvanceRejectsInvalidState'
