#!/usr/bin/env bash
set -euo pipefail

transition_mutation_repo=$(git rev-parse --show-toplevel)
transition_mutation_root=$(mktemp -d /private/tmp/order-lifecycle-transition-mutations.XXXXXX)

cleanup_transition_mutations() {
  rm -rf "${transition_mutation_root}"
}
trap cleanup_transition_mutations EXIT

run_transition_mutation() {
  local mutation_name=$1
  local mutation_original=$2
  local mutation_expression=$3
  local mutation_expected=$4
  local mutation_test=$5
  local mutation_failure_marker=$6
  local mutation_copy="${transition_mutation_root}/${mutation_name}"
  local mutation_policy="${mutation_copy}/services/api/internal/orderproduction/transition.go"

  mkdir -p "${mutation_copy}/services/api/internal/orderproduction"
  cp "${transition_mutation_repo}/go.mod" "${transition_mutation_repo}/go.sum" "${mutation_copy}/"
  cp "${transition_mutation_repo}/services/api/internal/orderproduction/"*.go \
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
    printf 'mutation test did not reach expected behavior assertion: %s exit=%s\n' \
      "${mutation_name}" "${mutation_exit}" >&2
    exit 82
  fi
  printf 'MUTATION_KILLED name=%s exit=1 marker=%s\n' \
    "${mutation_name}" "${mutation_failure_marker}"
}

run_transition_mutation \
  wrong_predecessor_can_mark_ready \
  'if input.Current == StatePreparing {' \
  's/if input\.Current == StatePreparing \{/if input.Current == StateReserved {/' \
  'if input.Current == StateReserved {' \
  '^TestTransitionMatchesCompleteStateTriggerMatrix$' \
  '--- FAIL: TestTransitionMatchesCompleteStateTriggerMatrix/RESERVED/MERCHANT_MARK_READY'

run_transition_mutation \
  redeem_can_skip_ready_state \
  'if input.Current == StateReadyForPickup {' \
  's/if input\.Current == StateReadyForPickup \{/if input.Current == StatePreparing {/' \
  'if input.Current == StatePreparing {' \
  '^TestTransitionMatchesCompleteStateTriggerMatrix$' \
  '--- FAIL: TestTransitionMatchesCompleteStateTriggerMatrix/PREPARING/REDEEM_SUCCEEDED'

run_transition_mutation \
  cancel_allowed_at_exactly_thirty_minutes \
  'if input.PickupAt.Sub(input.ObservedAt) <= productionLeadTime {' \
  's/input\.PickupAt\.Sub\(input\.ObservedAt\) <= productionLeadTime/input.PickupAt.Sub(input.ObservedAt) < productionLeadTime/' \
  'if input.PickupAt.Sub(input.ObservedAt) < productionLeadTime {' \
  '^TestTransitionRejectsUserCancelAtOrInsideThirtyMinutes$' \
  '--- FAIL: TestTransitionRejectsUserCancelAtOrInsideThirtyMinutes/exactly_thirty_minutes'

run_transition_mutation \
  preparing_user_can_cancel \
  'if input.Current != StateReserved {' \
  's/if input\.Current != StateReserved \{/if input.Current != StatePreparing {/' \
  'if input.Current != StatePreparing {' \
  '^TestTransitionMatchesCompleteStateTriggerMatrix$' \
  '--- FAIL: TestTransitionMatchesCompleteStateTriggerMatrix/PREPARING/USER_CANCEL'

run_transition_mutation \
  completed_owner_refund_omitted \
  'case StateReserved, StatePreparing, StateReadyForPickup, StateCompleted:' \
  's/case StateReserved, StatePreparing, StateReadyForPickup, StateCompleted:/case StateReserved, StatePreparing, StateReadyForPickup:/' \
  'case StateReserved, StatePreparing, StateReadyForPickup:' \
  '^TestTransitionOwnerCanRequestRefundFromEligibleStates$' \
  '--- FAIL: TestTransitionOwnerCanRequestRefundFromEligibleStates/COMPLETED'

run_transition_mutation \
  refunding_owner_refund_repeated \
  'case StateReserved, StatePreparing, StateReadyForPickup, StateCompleted:' \
  's/case StateReserved, StatePreparing, StateReadyForPickup, StateCompleted:/case StateReserved, StatePreparing, StateReadyForPickup, StateCompleted, StateRefunding:/' \
  'case StateReserved, StatePreparing, StateReadyForPickup, StateCompleted, StateRefunding:' \
  '^TestTransitionMatchesCompleteStateTriggerMatrix$' \
  '--- FAIL: TestTransitionMatchesCompleteStateTriggerMatrix/REFUNDING/OWNER_REFUND_REQUESTED'

run_transition_mutation \
  verified_refund_completes_non_refunding_order \
  'if input.Current == StateRefunding {' \
  's/if input\.Current == StateRefunding \{/if input.Current == StateCompleted {/' \
  'if input.Current == StateCompleted {' \
  '^TestTransitionMatchesCompleteStateTriggerMatrix$' \
  '--- FAIL: TestTransitionMatchesCompleteStateTriggerMatrix/COMPLETED/VERIFIED_REFUND_SUCCEEDED'

run_transition_mutation \
  invalid_state_is_accepted \
  'return false' \
  's/return false/return true/' \
  'return true' \
  '^TestTransitionRejectsInvalidAndDeprecatedStates$' \
  '--- FAIL: TestTransitionRejectsInvalidAndDeprecatedStates'
