#!/usr/bin/env bash
set -euo pipefail

change_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "${change_dir}/../.." && pwd)"
mutation_root="$(mktemp -d)"

cleanup_mutations() {
  rm -rf "${mutation_root}"
}
trap cleanup_mutations EXIT

run_mutation() {
  local mutation_name="$1"
  local test_name="$2"
  local mutation_program="$3"
  local case_root="${mutation_root}/${mutation_name}"

  mkdir -p "${case_root}/services/api/internal"
  cp "${repo_root}/go.mod" "${repo_root}/go.sum" "${case_root}/"
  cp -R "${repo_root}/services/api/internal/paymentobservation" "${case_root}/services/api/internal/"
  cp -R "${repo_root}/services/api/internal/wechatpay" "${case_root}/services/api/internal/"
  perl -0pi -e "${mutation_program}" "${case_root}/services/api/internal/paymentobservation/normalize.go"

  if (
    cd "${case_root}"
    GOWORK=off GOPROXY=off GOTOOLCHAIN=go1.26.5 \
      go test ./services/api/internal/paymentobservation -run "${test_name}" -count=1 >/dev/null 2>&1
  ); then
    printf 'MUTATION_SURVIVED name=%s\n' "${mutation_name}" >&2
    return 1
  fi
  printf 'MUTATION_KILLED name=%s\n' "${mutation_name}"
}

run_mutation \
  canonical-domain \
  '^TestNormalizeCanonicalizesSuccessfulCallbackAndQueryIdentically$' \
  's/order\.payment-observation\.v1/order.payment-observation.v0/'

run_mutation \
  local-time \
  '^TestNormalizeCanonicalizesSuccessfulCallbackAndQueryIdentically$' \
  's/actualSuccessTime = successTime\.Format\(time\.RFC3339Nano\)/actualSuccessTime = time.Now\(\).UTC\(\).Format\(time.RFC3339Nano\)/'

run_mutation \
  mismatch-precedence \
  '^TestNormalizeUsesFrozenMismatchPrecedence$' \
  's/case expected\.AppID != transaction\.AppID:\n\t\treturn MismatchAppID\n\tcase expected\.MerchantID != transaction\.MerchantID:\n\t\treturn MismatchMerchantID/case expected.MerchantID != transaction.MerchantID:\n\t\treturn MismatchMerchantID\n\tcase expected.AppID != transaction.AppID:\n\t\treturn MismatchAppID/'

run_mutation \
  mismatch-accepted \
  '^TestNormalizePersistsOnlyTheFirstSafeMismatch$' \
  's/validation = ValidationRejectedMismatch/validation = ValidationAccepted/'

run_mutation \
  rejected-retains-provider-facts \
  '^TestNormalizePersistsOnlyTheFirstSafeMismatch$' \
  's/validation == ValidationRejectedMismatch \|\| state != StatePaid/state != StatePaid/'

run_mutation \
  malformed-expectation-accepted \
  '^TestNormalizeRejectsMalformedExpectation$' \
  's/if malformedExpectation\(expected\) \{/if false \&\& malformedExpectation(expected) {/'

run_mutation \
  transaction-id-collision \
  '^TestNormalizeDedupeSeparatesEveryCriticalFact$' \
  's/\n\t\ttransaction\.TransactionID,\n/\n\t\t"",\n/'

printf 'PAYMENT_OBSERVATION_MUTATION_GATE=PASS killed=7\n'
