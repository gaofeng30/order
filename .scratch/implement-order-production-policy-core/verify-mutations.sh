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
  local mutation_expression=$2
  local mutation_expected=$3
  local mutation_test=$4
  local mutation_copy="${mutation_root}/${mutation_name}"
  local mutation_policy="${mutation_copy}/services/api/internal/orderproduction/policy.go"

  mkdir -p "${mutation_copy}/services/api/internal/orderproduction"
  cp "${mutation_repo}/go.mod" "${mutation_repo}/go.sum" "${mutation_copy}/"
  cp "${mutation_repo}/services/api/internal/orderproduction/"*.go \
    "${mutation_copy}/services/api/internal/orderproduction/"

  perl -0pi -e "${mutation_expression}" "${mutation_policy}"
  if ! grep -Fq "${mutation_expected}" "${mutation_policy}"; then
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
    printf 'mutation survived: %s test=%s\n' "${mutation_name}" "${mutation_test}" >&2
    exit 81
  fi
  printf 'MUTATION_KILLED name=%s exit=%s test=%s\n' \
    "${mutation_name}" "${mutation_exit}" "${mutation_test}"
}

run_mutation \
  initial_less_than_to_less_equal \
  's/pickupAt\.Sub\(paymentSucceededAt\) < productionLeadTime/pickupAt.Sub(paymentSucceededAt) <= productionLeadTime/' \
  'pickupAt.Sub(paymentSucceededAt) <= productionLeadTime' \
  '^TestInitialStateExactlyThirtyMinutesBeforePickupStartsReserved$'

run_mutation \
  advance_greater_equal_to_greater \
  's/if observedAt\.Before\(threshold\) \{/if !observedAt.After(threshold) {/' \
  'if !observedAt.After(threshold) {' \
  '^TestAdvanceAtThresholdStartsPreparing$'

run_mutation \
  insufficient_time_starts_reserved \
  's/return StatePreparing, nil/return StateReserved, nil/' \
  'return StateReserved, nil' \
  '^TestInitialStateLessThanThirtyMinutesBeforePickup$'

run_mutation \
  successor_rolls_back \
  's/return Decision\{State: current\}, nil/return Decision{State: StatePreparing, Changed: true}, nil/' \
  'return Decision{State: StatePreparing, Changed: true}, nil' \
  '^TestAdvanceDoesNotMoveSuccessorStates$'

run_mutation \
  invalid_state_is_accepted \
  's/return Decision\{\}, &Error\{kind: ErrorInvalidState\}/return Decision{State: current}, nil/' \
  'return Decision{State: current}, nil' \
  '^TestAdvanceRejectsInvalidState$'
