#!/usr/bin/env bash
set -euo pipefail

mutation_repo=$(git rev-parse --show-toplevel)
mutation_root=$(mktemp -d /private/tmp/staff-identity-mutations.XXXXXX)
mutation_package=services/api/internal/staffidentity

cleanup_mutations() {
  case "${mutation_root}" in
    /private/tmp/staff-identity-mutations.*)
      rm -rf -- "${mutation_root}"
      ;;
    *)
      printf 'refusing unsafe mutation cleanup: %s\n' "${mutation_root}" >&2
      exit 99
      ;;
  esac
}
trap cleanup_mutations EXIT

byte_count() {
  local count_file=$1
  local count_needle=$2
  MUTATION_NEEDLE="${count_needle}" perl -0777 -e '
    use bytes;
    my $needle = $ENV{"MUTATION_NEEDLE"};
    local $/;
    my $data = <>;
    my ($count, $offset) = (0, 0);
    while ((my $index = index($data, $needle, $offset)) >= 0) {
      $count++;
      $offset = $index + length($needle);
    }
    print $count;
  ' "${count_file}"
}

replace_exact_bytes() {
  local replace_file=$1
  local replace_original=$2
  local replace_replacement=$3
  MUTATION_ORIGINAL="${replace_original}" MUTATION_REPLACEMENT="${replace_replacement}" \
    perl -0777 -pi -e '
      use bytes;
      my $original = $ENV{"MUTATION_ORIGINAL"};
      my $replacement = $ENV{"MUTATION_REPLACEMENT"};
      my $index = index($_, $original);
      die "exact mutation source missing\n" if $index < 0;
      substr($_, $index, length($original), $replacement);
    ' "${replace_file}"
}

test_function_count() {
  local test_name=$1
  { grep -Fh -- "func ${test_name}(" "${mutation_repo}/${mutation_package}/"*_test.go || true; } | wc -l | tr -d ' '
}

check_binding() {
  local mutation_name=$1
  local mutation_file=$2
  local mutation_original=$3
  local mutation_replacement=$4
  local mutation_test=$5
  local source_file="${mutation_repo}/${mutation_file}"
  local original_count test_count

  if [[ ! -f "${source_file}" ]]; then
    printf 'mutation source file missing: %s file=%s\n' "${mutation_name}" "${mutation_file}" >&2
    exit 78
  fi
  original_count=$(byte_count "${source_file}" "${mutation_original}")
  if [[ ${original_count} -ne 1 ]]; then
    printf 'mutation source count=%s, want 1: %s\n' "${original_count}" "${mutation_name}" >&2
    exit 79
  fi
  test_count=$(test_function_count "${mutation_test}")
  if [[ ${test_count} -ne 1 ]]; then
    printf 'mutation test count=%s, want 1: %s test=%s\n' "${test_count}" "${mutation_name}" "${mutation_test}" >&2
    exit 78
  fi
  if [[ "${mutation_original}" == "${mutation_replacement}" ]]; then
    printf 'mutation replacement is identical: %s\n' "${mutation_name}" >&2
    exit 80
  fi
}

run_mutation() {
  local mutation_name=$1
  local mutation_file=$2
  local mutation_original=$3
  local mutation_replacement=$4
  local mutation_test=$5
  local mutation_copy="${mutation_root}/${mutation_name}"
  local mutation_target="${mutation_copy}/${mutation_file}"
  local replacement_count_before remaining_count replacement_count_after changed_files
  local test_exit

  check_binding "${mutation_name}" "${mutation_file}" "${mutation_original}" "${mutation_replacement}" "${mutation_test}"

  mkdir -p "${mutation_copy}/${mutation_package}" "${mutation_root}/go-cache"
  cp "${mutation_repo}/go.mod" "${mutation_repo}/go.sum" "${mutation_copy}/"
  cp "${mutation_repo}/${mutation_package}/"*.go "${mutation_copy}/${mutation_package}/"

  replacement_count_before=$(byte_count "${mutation_target}" "${mutation_replacement}")
  replace_exact_bytes "${mutation_target}" "${mutation_original}" "${mutation_replacement}"
  remaining_count=$(byte_count "${mutation_target}" "${mutation_original}")
  replacement_count_after=$(byte_count "${mutation_target}" "${mutation_replacement}")
  changed_files=$({ diff -qr "${mutation_repo}/${mutation_package}" "${mutation_copy}/${mutation_package}" || true; } | wc -l | tr -d ' ')
  if [[ ${remaining_count} -ne 0 || ${replacement_count_after} -ne $((replacement_count_before + 1)) || ${changed_files} -ne 1 ]]; then
    printf 'mutation uniqueness failed: %s old=%s new_before=%s new_after=%s files=%s\n' \
      "${mutation_name}" "${remaining_count}" "${replacement_count_before}" "${replacement_count_after}" "${changed_files}" >&2
    exit 80
  fi

  if [[ ${MUTATION_SELF_TEST:-} == go-exit2 ]]; then
    printf 'simulated go infrastructure failure\n' >"${mutation_copy}/test.log"
    test_exit=2
  else
    set +e
    (
      cd "${mutation_copy}"
      GOCACHE="${mutation_root}/go-cache" GOPROXY=off GOTOOLCHAIN=go1.26.5 \
        go test "./${mutation_package}" -run "^${mutation_test}$" -count=1
    ) >"${mutation_copy}/test.log" 2>&1
    test_exit=$?
    set -e
  fi

  if [[ ${test_exit} -ne 1 ]] || ! grep -Fq -- "--- FAIL: ${mutation_test}" "${mutation_copy}/test.log"; then
    printf 'mutation did not reach named assertion: %s exit=%s marker=%s\n' \
      "${mutation_name}" "${test_exit}" "${mutation_test}" >&2
    exit 82
  fi
  printf 'MUTATION_KILLED name=%s exit=1 marker=%s source_files=1\n' \
    "${mutation_name}" "${mutation_test}"
}

all_mutations() {
  local mutation_handler=$1

  "${mutation_handler}" primary_requires_name "${mutation_package}/resolve.go" \
    'if entry.Phone == input.PrimaryPhone && entry.Enabled {' \
    'if entry.Phone == input.PrimaryPhone && entry.Enabled && input.Extra != nil && normalizeName(entry.Name) == normalizeName(input.Extra.Name) {' \
    TestResolveEnabledPrimaryIsStaffIgnoringName

  "${mutation_handler}" extra_phone_only_authorizes "${mutation_package}/resolve.go" \
    'if entry.Phone == input.Extra.Phone && normalizeName(entry.Name) == extraName && entry.Enabled {' \
    'if entry.Phone == input.Extra.Phone && (normalizeName(entry.Name) == extraName || normalizeName(entry.Name) != extraName) && entry.Enabled {' \
    TestResolveExtraNameMismatchIsVisitor

  "${mutation_handler}" primary_enabled_ignored "${mutation_package}/resolve.go" \
    'if entry.Phone == input.PrimaryPhone && entry.Enabled {' \
    'if entry.Phone == input.PrimaryPhone {' \
    TestResolveDisabledPrimaryIsVisitor

  "${mutation_handler}" disabled_extra_authorizes "${mutation_package}/resolve.go" \
    'if entry.Phone == input.Extra.Phone && normalizeName(entry.Name) == extraName && entry.Enabled {' \
    'if entry.Phone == input.Extra.Phone && normalizeName(entry.Name) == extraName {' \
    TestResolveDisabledExtraIsVisitor

  "${mutation_handler}" width_fold_omitted "${mutation_package}/resolve.go" \
    'folded := width.Fold.String(name)' \
    'folded := name; _ = width.Fold' \
    TestResolveFoldsNameWidth

  "${mutation_handler}" ascii_space_deletion_omitted "${mutation_package}/resolve.go" \
    'return strings.ReplaceAll(composed, " ", "")' \
    'return strings.ReplaceAll(composed, "\x00", "")' \
    TestResolveDeletesPostFoldASCIISpaces

  "${mutation_handler}" nfkc_over_normalizes "${mutation_package}/resolve.go" \
    'composed := norm.NFC.String(folded)' \
    'composed := norm.NFKC.String(folded)' \
    TestResolveDoesNotApplyNFKC

  "${mutation_handler}" name_mismatch_authorizes "${mutation_package}/resolve.go" \
    'normalizeName(entry.Name) == extraName' \
    'normalizeName(entry.Name) != extraName' \
    TestResolveExtraNameMismatchIsVisitor

  "${mutation_handler}" duplicate_allowed "${mutation_package}/resolve.go" \
    'if _, duplicate := seenPhones[entry.Phone]; duplicate {' \
    'if _, duplicate := seenPhones[entry.Phone]; duplicate && false {' \
    TestResolveRejectsDuplicateWhitelistPhone

  "${mutation_handler}" visitor_drops_version "${mutation_package}/resolve.go" \
    'return Snapshot{Kind: KindVisitor, WhitelistVersion: input.WhitelistVersion}, nil' \
    'return Snapshot{Kind: KindVisitor}, nil' \
    TestResolveNoMatchIsVersionedVisitor

  "${mutation_handler}" malformed_evidence_becomes_visitor "${mutation_package}/resolve.go" \
    $'if normalizeName(entry.Name) == "" {\n\t\t\treturn Snapshot{}, newError(ErrorInvalidWhitelistSnapshot)\n\t\t}' \
    $'if normalizeName(entry.Name) == "" {\n\t\t\treturn Snapshot{Kind: KindVisitor, WhitelistVersion: input.WhitelistVersion}, nil\n\t\t}' \
    TestResolveRejectsWhitelistEntryWithEmptyNormalizedName

  "${mutation_handler}" error_returns_partial_snapshot "${mutation_package}/resolve.go" \
    'return Snapshot{}, newError(ErrorInvalidPrimaryPhone)' \
    'return Snapshot{Kind: KindVisitor, WhitelistVersion: input.WhitelistVersion}, newError(ErrorInvalidPrimaryPhone)' \
    TestResolveRejectsInvalidPrimaryPhone

  "${mutation_handler}" primary_validation_loses_priority "${mutation_package}/resolve.go" \
    'if !isCanonicalPhone(input.PrimaryPhone) {' \
    'if !isCanonicalPhone(input.PrimaryPhone) && input.Extra == nil {' \
    TestResolveRejectsInvalidPrimaryBeforeLowerPriorityEvidence

  "${mutation_handler}" extra_validation_loses_priority "${mutation_package}/resolve.go" \
    'if input.Extra != nil && !isCanonicalPhone(input.Extra.Phone) {' \
    'if input.Extra != nil && !isCanonicalPhone(input.Extra.Phone) && input.WhitelistVersion != 0 {' \
    TestResolveRejectsInvalidExtraBeforeWhitelistEvidence

  "${mutation_handler}" zero_version_allowed "${mutation_package}/resolve.go" \
    'if input.WhitelistVersion == 0 {' \
    'if input.WhitelistVersion == ^uint64(0) {' \
    TestResolveRejectsZeroWhitelistVersion

  "${mutation_handler}" unrelated_entry_ignored "${mutation_package}/resolve.go" \
    'if entry.Phone != input.PrimaryPhone && (input.Extra == nil || entry.Phone != input.Extra.Phone) {' \
    'if false && entry.Phone != input.PrimaryPhone && (input.Extra == nil || entry.Phone != input.Extra.Phone) {' \
    TestResolveRejectsUnrelatedWhitelistEntry

  "${mutation_handler}" input_is_mutated "${mutation_package}/resolve.go" \
    'seenPhones := make(map[string]struct{}, len(input.CandidateEntries))' \
    'input.CandidateEntries[0].Name = "mutated"; seenPhones := map[string]struct{}{}' \
    TestResolveDoesNotMutateInputAndIsDeterministic
}

run_self_test() {
  case "${MUTATION_SELF_TEST}" in
    missing-source)
      run_mutation self_missing_source "${mutation_package}/resolve.go" \
        '__MISSING_MUTATION_SOURCE__' 'replacement' TestResolveNoMatchIsVersionedVisitor
      ;;
    duplicate-source)
      run_mutation self_duplicate_source "${mutation_package}/resolve.go" \
        'return Snapshot{}, newError(ErrorInvalidWhitelistSnapshot)' \
        'return Snapshot{Kind: KindVisitor}, newError(ErrorInvalidWhitelistSnapshot)' \
        TestResolveRejectsZeroWhitelistVersion
      ;;
    go-exit2)
      run_mutation self_go_exit2 "${mutation_package}/resolve.go" \
        'return strings.ReplaceAll(composed, " ", "")' \
        'return strings.ReplaceAll(composed, "\x00", "")' \
        TestResolveDeletesPostFoldASCIISpaces
      ;;
    *)
      printf 'unknown MUTATION_SELF_TEST=%s\n' "${MUTATION_SELF_TEST}" >&2
      exit 64
      ;;
  esac
}

if [[ -n ${MUTATION_SELF_TEST:-} ]]; then
  run_self_test
  exit 0
fi

run_one_mutation() {
  local mutation_name=$1
  if [[ "${mutation_name}" != "${run_one_name}" ]]; then
    return
  fi
  run_one_found=1
  run_mutation "$@"
}

case "${1:-run}" in
  check-bindings)
    all_mutations check_binding
    printf 'MUTATION_BINDINGS=PASS mutants=17 source_count=1 test_count=1\n'
    ;;
  run)
    all_mutations run_mutation
    ;;
  run-one)
    if [[ $# -ne 2 || -z ${2} ]]; then
      printf 'usage: %s run-one <mutation-name>\n' "$0" >&2
      exit 64
    fi
    run_one_name=$2
    run_one_found=0
    all_mutations run_one_mutation
    if [[ ${run_one_found} -ne 1 ]]; then
      printf 'unknown mutation name: %s\n' "${run_one_name}" >&2
      exit 64
    fi
    ;;
  *)
    printf 'usage: %s [check-bindings|run|run-one <mutation-name>]\n' "$0" >&2
    exit 64
    ;;
esac
