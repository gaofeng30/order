#!/usr/bin/env bash
set -euo pipefail

mutation_repo=$(git rev-parse --show-toplevel)
mutation_base=8cae09d5bc3e659d8851e7588835e579101058ac
mutation_root=$(mktemp -d /private/tmp/order-store-status-mutations.XXXXXX)
mutation_copy="${mutation_root}/repo"
mutation_source="${mutation_copy}/services/api/internal/storestatus/core.go"
mutation_output="${mutation_root}/test-output.txt"

cleanup_store_status_mutations() {
  rm -rf "${mutation_root}"
}
trap cleanup_store_status_mutations EXIT

mkdir -p "${mutation_copy}"
git -C "${mutation_repo}" archive "${mutation_base}" | tar -xf - -C "${mutation_copy}"
mkdir -p "${mutation_copy}/services/api/internal/storestatus"
cp -R "${mutation_repo}/services/api/internal/storestatus/." "${mutation_copy}/services/api/internal/storestatus/"

mutation_original=$(mktemp "${mutation_root}/core-original.XXXXXX")
cp "${mutation_source}" "${mutation_original}"

replace_exactly_once() {
  local source_file=$1
  local expected=$2
  local replacement=$3
  local matches
  matches=$(MUTATION_EXPECTED="${expected}" perl -0ne '$count += () = /\Q$ENV{MUTATION_EXPECTED}\E/g; END { print $count + 0 }' "${source_file}")
  if [[ "${matches}" != 1 ]]; then
    return 64
  fi
  MUTATION_EXPECTED="${expected}" MUTATION_REPLACEMENT="${replacement}" \
    perl -0pi -e 's/\Q$ENV{MUTATION_EXPECTED}\E/$ENV{MUTATION_REPLACEMENT}/' "${source_file}"
}

if replace_exactly_once "${mutation_source}" "this source fragment is intentionally absent" "shield unexpectedly accepted"; then
  printf 'mutation infrastructure shield failed: absent source was accepted\n' >&2
  exit 1
fi
printf 'PASS infrastructure-failure-shield\n'

run_mutant() {
  local name=$1
  local expected=$2
  local replacement=$3
  local test_pattern=$4
  local failure_marker=$5

  cp "${mutation_original}" "${mutation_source}"
  if ! replace_exactly_once "${mutation_source}" "${expected}" "${replacement}"; then
    printf 'mutation %s infrastructure failure: source match was not exactly one\n' "${name}" >&2
    exit 1
  fi
  gofmt -w "${mutation_source}"
  if (
    cd "${mutation_copy}"
    go test ./services/api/internal/storestatus -run "${test_pattern}" -count=1 -timeout=2m
  ) >"${mutation_output}" 2>&1; then
    printf 'mutation survived: %s\n' "${name}" >&2
    sed -n '1,160p' "${mutation_output}" >&2
    exit 1
  fi
  if ! grep -F -- "${failure_marker}" "${mutation_output}" >/dev/null; then
    printf 'mutation %s failed outside its named behavior assertion\n' "${name}" >&2
    sed -n '1,160p' "${mutation_output}" >&2
    exit 1
  fi
  printf 'KILLED %s\n' "${name}"
}

run_mutant \
  illegal-enum \
  $'func validStatus(status storefront.BusinessStatus) bool {\n\tswitch status {\n\tcase storefront.BusinessOpen, storefront.BusinessClosed, storefront.BusinessCutoff:\n\t\treturn true\n\tdefault:\n\t\treturn false\n\t}\n}' \
  $'func validStatus(status storefront.BusinessStatus) bool {\n\tswitch status {\n\tcase storefront.BusinessOpen, storefront.BusinessClosed, storefront.BusinessCutoff:\n\t\treturn true\n\tdefault:\n\t\treturn true\n\t}\n}' \
  '^TestApplyRejectsInvalidStatusBeforeDependencies$' \
  'TestApplyRejectsInvalidStatusBeforeDependencies'

run_mutant \
  bypass-authorization \
  $'authorization, err := core.authorizer.AuthorizeInTx(\n\t\tctx,\n\t\ttransaction,\n\t\tcommand.UserID,\n\t\tmerchantidentity.ActionStoreStatusWrite,\n\t\tmerchantidentity.Target{Type: "storefront_settings", ID: 1},\n\t)\n\tif err != nil {\n\t\treturn Result{}, err\n\t}' \
  $'authorization := merchantidentity.Authorization{\n\t\tMerchantAccountID: 1,\n\t\tActor: merchantidentity.ActorMerchantOwner,\n\t\tRecordVersion: 1,\n\t\tAuthVersion: 1,\n\t}' \
  '^TestApplyRejectsDisabledAccountWithoutMutation$' \
  'TestApplyRejectsDisabledAccountWithoutMutation'

run_mutant \
  remove-folded-authorization-retry \
  $'\tif errors.Is(err, merchantidentity.ErrUnavailable) {\n\t\treturn true\n\t}\n' \
  '' \
  '^TestApplyRetriesAuthorizationLockTimeoutWithFreshRole$' \
  'TestApplyRetriesAuthorizationLockTimeoutWithFreshRole'

run_mutant \
  wrong-authorization-action \
  $'\t\tmerchantidentity.ActionStoreStatusWrite,\n\t\tmerchantidentity.Target{Type: "storefront_settings", ID: 1},' \
  $'\t\tmerchantidentity.ActionOrderRead,\n\t\tmerchantidentity.Target{Type: "storefront_settings", ID: 1},' \
  '^TestApplyUsesFrozenAuthorizationActionAndTarget$' \
  'TestApplyUsesFrozenAuthorizationActionAndTarget'

run_mutant \
  wrong-authorization-target \
  'merchantidentity.Target{Type: "storefront_settings", ID: 1},' \
  'merchantidentity.Target{Type: "storefront_settings", ID: 2},' \
  '^TestApplyUsesFrozenAuthorizationActionAndTarget$' \
  'TestApplyUsesFrozenAuthorizationActionAndTarget'

run_mutant \
  remove-for-update \
  'SELECT business_status FROM storefront_settings WHERE id=1 FOR UPDATE' \
  'SELECT business_status FROM storefront_settings WHERE id=1' \
  '^TestApplyConcurrentSameKeyNoOpRequiresSingletonLock$' \
  'TestApplyConcurrentSameKeyNoOpRequiresSingletonLock'

run_mutant \
  duplicate-key-rewrites \
  $'if found {\n\t\tif err := core.commit(transaction); err != nil {' \
  $'if false && found {\n\t\tif err := core.commit(transaction); err != nil {' \
  '^TestApplyReplayReturnsFirstResultWithoutRewrite$' \
  'TestApplyReplayReturnsFirstResultWithoutRewrite'

run_mutant \
  conflicting-key-allowed \
  'if targetType != "storefront_settings" || targetID != 1 || after != command.DesiredStatus {' \
  'if false && (targetType != "storefront_settings" || targetID != 1 || after != command.DesiredStatus) {' \
  '^TestApplyRejectsSameKeyWithDifferentDesiredStatus$' \
  'TestApplyRejectsSameKeyWithDifferentDesiredStatus'

run_mutant \
  changes-another-storefront-column \
  'UPDATE storefront_settings SET business_status=? WHERE id=1' \
  "UPDATE storefront_settings SET business_status=?,announcement='mutation' WHERE id=1" \
  '^TestApplyOwnerChangesOnlyBusinessStatus$' \
  'TestApplyOwnerChangesOnlyBusinessStatus'

run_mutant \
  corrupt-success-audit \
  ") VALUES (?,?,?,?,?,?,'SUCCEEDED',?,'storefront_settings',1,?,?,?,?,?)" \
  ") VALUES (?,?,?,?,?,?,'REJECTED',?,'storefront_settings',1,?,?,?,?,?)" \
  '^TestApplyWritesExactSuccessAudit$' \
  'TestApplyWritesExactSuccessAudit'

run_mutant \
  audit-failure-still-commits \
  $'); err != nil {\n\t\treturn Result{}, err\n\t}\n\tif err := core.commit(transaction); err != nil {' \
  $'); err != nil {\n\t\treturn Result{}, core.commit(transaction)\n\t}\n\tif err := core.commit(transaction); err != nil {' \
  '^TestApplyAuditInsertFailureRollsBackAndSameKeyRecovers$' \
  'TestApplyAuditInsertFailureRollsBackAndSameKeyRecovers'

cmp -s "${mutation_original}" "${mutation_repo}/services/api/internal/storestatus/core.go"
printf 'PASS writer-source-unchanged\n'
