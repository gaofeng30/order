#!/usr/bin/env bash
set -euo pipefail

mutation_repo=$(git rev-parse --show-toplevel)
mutation_root=$(mktemp -d /private/tmp/order-pickup-options-mutations.XXXXXX)

cleanup_mutations() {
  rm -rf "${mutation_root}"
}
trap cleanup_mutations EXIT

literal_count() {
  local source_path=$1
  local source_text=$2
  MUTATION_SOURCE_TEXT="${source_text}" perl -0ne '
    BEGIN { $needle = $ENV{"MUTATION_SOURCE_TEXT"}; $count = 0 }
    $count++ while /\Q$needle\E/g;
    END { print $count }
  ' "${source_path}"
}

run_mutation() {
  local mutation_name=$1
  local mutation_relative_file=$2
  local mutation_original=$3
  local mutation_replacement=$4
  local mutation_test=$5
  local mutation_failure_marker=$6
  local mutation_copy="${mutation_root}/${mutation_name}"
  local mutation_file="${mutation_copy}/${mutation_relative_file}"

  mkdir -p "${mutation_copy}/services/api/internal"
  cp "${mutation_repo}/go.mod" "${mutation_repo}/go.sum" "${mutation_copy}/"
  cp -R "${mutation_repo}/services/api/internal/menu" "${mutation_copy}/services/api/internal/"
  cp -R "${mutation_repo}/services/api/internal/database" "${mutation_copy}/services/api/internal/"
  cp -R "${mutation_repo}/services/api/internal/migrate" "${mutation_copy}/services/api/internal/"
  mkdir -p "${mutation_copy}/services/api/migrations"
  cp "${mutation_repo}/services/api/migrations/"* "${mutation_copy}/services/api/migrations/"

  local mutation_original_count
  mutation_original_count=$(literal_count "${mutation_file}" "${mutation_original}")
  if [[ ${mutation_original_count} -ne 1 ]]; then
    printf 'mutation source count=%s, want 1: %s\n' "${mutation_original_count}" "${mutation_name}" >&2
    exit 79
  fi

  MUTATION_SOURCE_TEXT="${mutation_original}" MUTATION_REPLACEMENT_TEXT="${mutation_replacement}" perl -0pi -e '
    BEGIN { $original = $ENV{"MUTATION_SOURCE_TEXT"}; $replacement = $ENV{"MUTATION_REPLACEMENT_TEXT"} }
    s/\Q$original\E/$replacement/;
  ' "${mutation_file}"
  if [[ $(literal_count "${mutation_file}" "${mutation_original}") -ne 0 || $(literal_count "${mutation_file}" "${mutation_replacement}") -ne 1 ]]; then
    printf 'mutation was not applied exactly once: %s\n' "${mutation_name}" >&2
    exit 80
  fi

  set +e
  (
    cd "${mutation_copy}"
    GOPROXY=off GOTOOLCHAIN=go1.26.5 go test ./services/api/internal/menu -run "${mutation_test}" -count=1
  ) >"${mutation_copy}/test.log" 2>&1
  local mutation_exit=$?
  set -e
  if [[ ${mutation_exit} -eq 0 ]]; then
    printf 'mutation survived: %s test=%s\n' "${mutation_name}" "${mutation_test}" >&2
    exit 81
  fi
  if [[ ${mutation_exit} -ne 1 ]] || ! grep -Fq -- "${mutation_failure_marker}" "${mutation_copy}/test.log"; then
    printf 'mutation test missed named FAIL marker: %s exit=%s marker=%s\n' "${mutation_name}" "${mutation_exit}" "${mutation_failure_marker}" >&2
    exit 82
  fi
  printf 'MUTATION_KILLED name=%s exit=1 marker=%s\n' "${mutation_name}" "${mutation_failure_marker}"
}

run_mutation \
  omit_tomorrow \
  services/api/internal/menu/pickup_options.go \
  'for dayOffset := 0; dayOffset < 2; dayOffset++ {' \
  'for dayOffset := 0; dayOffset < 1; dayOffset++ {' \
  '^TestPickupOptionsUsesShanghaiTodayAndTomorrow$' \
  '--- FAIL: TestPickupOptionsUsesShanghaiTodayAndTomorrow'

run_mutation \
  omit_closed_endpoint \
  services/api/internal/menu/pickup_options.go \
  'minute <= period.pickupEndMinute' \
  'minute < period.pickupEndMinute' \
  '^TestPickupOptionsIncludesEveryMealAndClosedRangeEndpoint$' \
  '--- FAIL: TestPickupOptionsIncludesEveryMealAndClosedRangeEndpoint'

run_mutation \
  ignore_configured_interval \
  services/api/internal/menu/pickup_options.go \
  'minute += period.intervalMinutes' \
  'minute += 30' \
  '^TestPickupOptionsHonorsConfiguredInterval$' \
  '--- FAIL: TestPickupOptionsHonorsConfiguredInterval'

run_mutation \
  exact_cutoff_remains_orderable \
  services/api/internal/menu/pickup_options.go \
  'mealOrderable := now.Before(cutoffAt)' \
  'mealOrderable := !now.After(cutoffAt)' \
  '^TestPickupOptionsUsesStrictCutoffAndDateOrderabilityOR$/^exact$' \
  '--- FAIL: TestPickupOptionsUsesStrictCutoffAndDateOrderabilityOR'

run_mutation \
  use_utc_calendar_date \
  services/api/internal/menu/pickup_options.go \
  'now := handler.now().In(shanghaiLocation)' \
  'now := handler.now()' \
  '^TestPickupOptionsUsesShanghaiTodayAndTomorrow$' \
  '--- FAIL: TestPickupOptionsUsesShanghaiTodayAndTomorrow'

run_mutation \
  delete_cutoff_meals \
  services/api/internal/menu/pickup_options.go \
  'dateOrderable = dateOrderable || mealOrderable' \
  $'if mealOrderable {\n\t\t\t\tdateOrderable = true\n\t\t\t} else {\n\t\t\t\tcontinue\n\t\t\t}' \
  '^TestPickupOptionsIncludesEveryMealAndClosedRangeEndpoint$' \
  '--- FAIL: TestPickupOptionsIncludesEveryMealAndClosedRangeEndpoint'

run_mutation \
  date_orderable_not_or \
  services/api/internal/menu/pickup_options.go \
  'dateOrderable = dateOrderable || mealOrderable' \
  'dateOrderable = false' \
  '^TestPickupOptionsDateOrderableUsesAnyMeal$' \
  '--- FAIL: TestPickupOptionsDateOrderableUsesAnyMeal'

run_mutation \
  overlap_returns_partial_success \
  services/api/internal/menu/selection.go \
  $'if periods[0].pickupStartMinute <= periods[1].pickupEndMinute && periods[1].pickupStartMinute <= periods[0].pickupEndMinute {\n\t\treturn nil, ErrMenuUnavailable\n\t}' \
  $'if periods[0].pickupStartMinute <= periods[1].pickupEndMinute && periods[1].pickupStartMinute <= periods[0].pickupEndMinute {\n\t\treturn periods[:1], nil\n\t}' \
  '^TestPickupOptionsFailsClosedForEveryInvalidCompleteConfiguration$/^overlap$' \
  '--- FAIL: TestPickupOptionsFailsClosedForEveryInvalidCompleteConfiguration'
