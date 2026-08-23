#!/usr/bin/env bash
set -euo pipefail

rgr_repo=$(git rev-parse --show-toplevel)
rgr_root=$(mktemp -d /private/tmp/order-pickup-options-rgr.XXXXXX)

cleanup_rgr() {
  rm -rf "${rgr_root}"
}
trap cleanup_rgr EXIT

literal_count() {
  local rgr_source_file=$1
  local rgr_source_text=$2
  if [[ -z "${rgr_source_text}" ]]; then
    printf 'RGR literal needle is empty\n' >&2
    return 78
  fi
  RGR_SOURCE_TEXT="${rgr_source_text}" perl -0ne '
    use bytes;
    my $needle = $ENV{"RGR_SOURCE_TEXT"};
    die "empty RGR literal needle\n" if length($needle) == 0;
    my $count = 0;
    my $offset = 0;
    while (1) {
      my $position = index($_, $needle, $offset);
      last if $position < 0;
      $count++;
      $offset = $position + length($needle);
    }
    print $count;
  ' "${rgr_source_file}"
}

replace_unique_literal() {
  local rgr_name=$1
  local rgr_source_file=$2
  local rgr_original=$3
  local rgr_replacement=$4
  local rgr_original_count

  if [[ -z "${rgr_original}" ]]; then
    printf 'RGR literal needle is empty: %s\n' "${rgr_name}" >&2
    return 78
  fi
  if ! rgr_original_count=$(literal_count "${rgr_source_file}" "${rgr_original}"); then
    return 78
  fi
  if [[ ${rgr_original_count} -ne 1 ]]; then
    printf 'RGR replay source is not unique: %s count=%s\n' "${rgr_name}" "${rgr_original_count}" >&2
    return 79
  fi

  local rgr_before_hash
  rgr_before_hash=$(shasum -a 256 "${rgr_source_file}" | awk '{print $1}')
  RGR_SOURCE_TEXT="${rgr_original}" RGR_REPLACEMENT_TEXT="${rgr_replacement}" perl -0pi -e '
    use bytes;
    my $original = $ENV{"RGR_SOURCE_TEXT"};
    my $replacement = $ENV{"RGR_REPLACEMENT_TEXT"};
    die "empty RGR literal needle\n" if length($original) == 0;
    my $position = index($_, $original);
    die "missing RGR literal needle\n" if $position < 0;
    die "non-unique RGR literal needle\n" if index($_, $original, $position + length($original)) >= 0;
    substr($_, $position, length($original), $replacement);
  ' "${rgr_source_file}"

  local rgr_after_hash
  local rgr_after_original_count
  rgr_after_hash=$(shasum -a 256 "${rgr_source_file}" | awk '{print $1}')
  rgr_after_original_count=$(literal_count "${rgr_source_file}" "${rgr_original}")
  if [[ "${rgr_before_hash}" == "${rgr_after_hash}" || ${rgr_after_original_count} -ne 0 ]]; then
    printf 'RGR replay replacement failed: %s\n' "${rgr_name}" >&2
    return 80
  fi
  if [[ -n "${rgr_replacement}" ]]; then
    local rgr_after_replacement_count
    rgr_after_replacement_count=$(literal_count "${rgr_source_file}" "${rgr_replacement}")
    if [[ ${rgr_after_replacement_count} -ne 1 ]]; then
      printf 'RGR replay replacement was not unique: %s count=%s\n' "${rgr_name}" "${rgr_after_replacement_count}" >&2
      return 80
    fi
  fi
}

source_tree_hash() {
  local rgr_copy=$1
  (
    cd "${rgr_copy}"
    find services/api -type f -name '*.go' -print | LC_ALL=C sort | while IFS= read -r rgr_go_file; do
      shasum -a 256 "${rgr_go_file}"
    done | shasum -a 256 | awk '{print $1}'
  )
}

replay_red() {
  local rgr_name=$1
  local rgr_relative_file=$2
  local rgr_original=$3
  local rgr_replacement=$4
  local rgr_package=$5
  local rgr_test=$6
  local rgr_marker=$7
  local rgr_copy="${rgr_root}/${rgr_name}"
  local rgr_file="${rgr_copy}/${rgr_relative_file}"

  mkdir -p "${rgr_copy}/services"
  cp "${rgr_repo}/go.mod" "${rgr_repo}/go.sum" "${rgr_copy}/"
  cp -R "${rgr_repo}/services/api" "${rgr_copy}/services/"

  replace_unique_literal "${rgr_name}" "${rgr_file}" "${rgr_original}" "${rgr_replacement}"

  local rgr_source_tree_sha256
  rgr_source_tree_sha256=$(source_tree_hash "${rgr_copy}")
  set +e
  (
    cd "${rgr_copy}"
    GOPROXY=off GOTOOLCHAIN=go1.26.5 go test "${rgr_package}" -run "${rgr_test}" -count=1
  ) >"${rgr_copy}/test.log" 2>&1
  local rgr_exit=$?
  set -e
  if [[ ${rgr_exit} -ne 1 ]] || ! grep -Fq -- "${rgr_marker}" "${rgr_copy}/test.log"; then
    printf 'RGR replay missed expected Red: %s exit=%s marker=%s\n' "${rgr_name}" "${rgr_exit}" "${rgr_marker}" >&2
    exit 81
  fi
  printf 'RGR_REPLAY name=%s source_tree_sha256=%s exit=1 marker=%s\n' \
    "${rgr_name}" "${rgr_source_tree_sha256}" "${rgr_marker}"
  rm -rf "${rgr_copy}"
}

rgr_infrastructure_self_check() {
  local rgr_self_check_root="${rgr_root}/infrastructure-self-check"
  local rgr_self_check_file="${rgr_self_check_root}/handler.go"
  local rgr_route_literal=$'\tengine.GET("/api/v1/menu/pickup-options", handler.getPickupOptions)\n'
  mkdir -p "${rgr_self_check_root}"
  cp "${rgr_repo}/services/api/internal/menu/handler.go" "${rgr_self_check_file}"

  local rgr_initial_hash
  local rgr_query_count
  rgr_initial_hash=$(shasum -a 256 "${rgr_self_check_file}" | awk '{print $1}')
  rgr_query_count=$(literal_count "${rgr_self_check_file}" 'Query()')

  set +e
  replace_unique_literal self_check_missing "${rgr_self_check_file}" '__RGR_SOURCE_DOES_NOT_EXIST__' '' >/dev/null 2>&1
  local rgr_missing_exit=$?
  replace_unique_literal self_check_empty "${rgr_self_check_file}" '' 'replacement' >/dev/null 2>&1
  local rgr_empty_exit=$?
  set -e
  if [[ ${rgr_missing_exit} -ne 79 || ${rgr_empty_exit} -ne 78 ]]; then
    printf 'RGR infrastructure source shield failed: missing=%s empty=%s\n' "${rgr_missing_exit}" "${rgr_empty_exit}" >&2
    exit 82
  fi
  if [[ "${rgr_initial_hash}" != "$(shasum -a 256 "${rgr_self_check_file}" | awk '{print $1}')" ]]; then
    printf 'RGR infrastructure failed source mutated the fixture\n' >&2
    exit 82
  fi

  local rgr_before_size
  rgr_before_size=$(wc -c < "${rgr_self_check_file}" | tr -d ' ')
  replace_unique_literal self_check_unique "${rgr_self_check_file}" "${rgr_route_literal}" ''
  local rgr_after_size
  rgr_after_size=$(wc -c < "${rgr_self_check_file}" | tr -d ' ')
  if [[ $(literal_count "${rgr_self_check_file}" "${rgr_route_literal}") -ne 0 ||
        $(literal_count "${rgr_self_check_file}" 'Query()') -ne ${rgr_query_count} ||
        $((rgr_before_size - rgr_after_size)) -ne ${#rgr_route_literal} ]]; then
    printf 'RGR infrastructure unique replacement changed non-target bytes\n' >&2
    exit 82
  fi
  printf 'RGR_INFRASTRUCTURE_SELF_CHECK=PASS missing_exit=79 empty_exit=78 query_count=%s\n' "${rgr_query_count}"
  rm -rf "${rgr_self_check_root}"
}

rgr_infrastructure_self_check

replay_red \
  route_missing \
  services/api/internal/menu/handler.go \
  $'\tengine.GET("/api/v1/menu/pickup-options", handler.getPickupOptions)\n' \
  '' \
  ./services/api/internal/httpapi \
  '^TestPickupOptionsRouteFailsClosedAnonymously$' \
  '--- FAIL: TestPickupOptionsRouteFailsClosedAnonymously'

replay_red \
  shanghai_dates_missing \
  services/api/internal/menu/pickup_options.go \
  'now := handler.now().In(shanghaiLocation)' \
  'now := handler.now()' \
  ./services/api/internal/menu \
  '^TestPickupOptionsUsesShanghaiTodayAndTomorrow$' \
  '--- FAIL: TestPickupOptionsUsesShanghaiTodayAndTomorrow'

replay_red \
  configured_pickup_point_missing \
  services/api/internal/menu/pickup_options.go \
  'pickupTimes = append(pickupTimes, fmt.Sprintf("%02d:%02d", minute/60, minute%60))' \
  $'if minute != period.pickupStartMinute+period.intervalMinutes {\n\t\t\t\tpickupTime := fmt.Sprintf("%02d:%02d", minute/60, minute%60)\n\t\t\t\tpickupTimes = append(pickupTimes, pickupTime)\n\t\t\t}' \
  ./services/api/internal/menu \
  '^TestPickupOptionsEnumeratesEveryConfiguredPickupTime$' \
  '--- FAIL: TestPickupOptionsEnumeratesEveryConfiguredPickupTime'

replay_red \
  closed_endpoint_missing \
  services/api/internal/menu/pickup_options.go \
  'minute <= period.pickupEndMinute' \
  'minute < period.pickupEndMinute' \
  ./services/api/internal/menu \
  '^TestPickupOptionsIncludesEveryMealAndClosedRangeEndpoint$' \
  '--- FAIL: TestPickupOptionsIncludesEveryMealAndClosedRangeEndpoint'

replay_red \
  configured_interval_missing \
  services/api/internal/menu/pickup_options.go \
  'minute += period.intervalMinutes' \
  'minute += 30' \
  ./services/api/internal/menu \
  '^TestPickupOptionsHonorsConfiguredInterval$' \
  '--- FAIL: TestPickupOptionsHonorsConfiguredInterval'

replay_red \
  strict_cutoff_missing \
  services/api/internal/menu/pickup_options.go \
  'mealOrderable := now.Before(cutoffAt)' \
  'mealOrderable := !now.After(cutoffAt)' \
  ./services/api/internal/menu \
  '^TestPickupOptionsUsesStrictCutoffAndDateOrderabilityOR$/^exact$' \
  '--- FAIL: TestPickupOptionsUsesStrictCutoffAndDateOrderabilityOR'
