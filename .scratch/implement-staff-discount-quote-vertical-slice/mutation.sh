#!/usr/bin/env bash
set -euo pipefail

quote_mutation_base=1657aa9451f612e4605fabd084ccab07542ac81a
quote_repo_root=$(git rev-parse --show-toplevel)
quote_mutation_root=$(mktemp -d /private/tmp/order-quote-mutations.XXXXXX)

cleanup_quote_mutations() {
  if [[ -d "${quote_mutation_root}" ]] && [[ "$(realpath "${quote_mutation_root}")" == /private/tmp/order-quote-mutations.* ]]; then
    find "${quote_mutation_root}" -depth -delete
  fi
}
trap cleanup_quote_mutations EXIT

git cat-file -e "${quote_mutation_base}^{commit}"
mkdir -p "${quote_mutation_root}/order"
cp "${quote_repo_root}/go.mod" "${quote_repo_root}/go.sum" "${quote_mutation_root}/order/"
rsync -a "${quote_repo_root}/services/" "${quote_mutation_root}/order/services/"

run_mutant() {
  local name="$1"
  local file="$2"
  local expression="$3"
  local test_pattern="$4"
  local expected_test="$5"
  local original="${quote_repo_root}/${file}"
  local mutated="${quote_mutation_root}/order/${file}"
  local log="${quote_mutation_root}/${expected_test}.log"

  cp "${original}" "${mutated}"
  perl -0pi -e "${expression}" "${mutated}"
  if cmp -s "${original}" "${mutated}"; then
    printf 'mutation did not apply: %s\n' "${name}" >&2
    exit 1
  fi
  if (
    cd "${quote_mutation_root}/order"
    env \
      -u ORDER_TEST_MYSQL_HOST \
      -u ORDER_TEST_MYSQL_PORT \
      -u ORDER_TEST_MYSQL_USER \
      -u ORDER_TEST_MYSQL_PASSWORD \
      -u ORDER_TEST_MYSQL_TLS_MODE \
      -u ORDER_TEST_MYSQL_INSTANCE \
      -u ORDER_TEST_MYSQL_ISOLATED \
		GOWORK=off \
		GOCACHE=/private/tmp/order-tx02-final-gocache \
      GOPROXY=off \
      GOTOOLCHAIN=go1.26.5 \
      go test ./services/api/internal/quote -run "${test_pattern}" -count=1 >"${log}" 2>&1
  ); then
    printf 'mutation escaped: %s\n' "${name}" >&2
    exit 1
  fi
  if ! rg -F -- "--- FAIL: ${expected_test}" "${log}" >/dev/null; then
    printf 'mutation produced infrastructure or compile failure instead of the expected behavioral failure: %s\n' "${name}" >&2
    sed -n '1,120p' "${log}" >&2
    exit 1
  fi
  printf 'killed: %s by %s\n' "${name}" "${expected_test}"
}

run_mutant \
  "staff rate forced to visitor rate" \
  "services/api/internal/quote/provider.go" \
  's/ratePercent = settings\.RatePercent/ratePercent = 100/' \
  '^TestProviderCreatesStaffQuoteFromOneLockedServerSnapshot$' \
  'TestProviderCreatesStaffQuoteFromOneLockedServerSnapshot'

run_mutant \
  "idempotency conflict bypassed" \
  "services/api/internal/quote/provider.go" \
  's/existing\.requestDigest != requestDigest/existing.requestDigest != existing.requestDigest/' \
  '^TestProviderReplaysCompleteStoredQuoteAndConflictsWithoutRepricing$' \
  'TestProviderReplaysCompleteStoredQuoteAndConflictsWithoutRepricing'

run_mutant \
  "client money accepted" \
  "services/api/internal/quote/handler.go" \
  's/decoder\.DisallowUnknownFields\(\)/\/\/ mutation removed unknown-field rejection/' \
  '^TestCreateRejectsClientOwnedMoneyWithoutCallingApplication$' \
  'TestCreateRejectsClientOwnedMoneyWithoutCallingApplication'

run_mutant \
  "snapshot digest ignored" \
  "services/api/internal/quote/provider.go" \
  's/hashQuoteSnapshot\(header\.quote\) != header\.quote\.SnapshotDigest/false/' \
  '^TestProviderFailsClosedWhenStoredSnapshotDigestDoesNotMatch$' \
  'TestProviderFailsClosedWhenStoredSnapshotDigestDoesNotMatch'

run_mutant \
  "cover object key omitted from immutable digest" \
  "services/api/internal/quote/digest.go" \
  's/value\.writeBool\(item\.ImageObjectKey != ""\)/value.writeBool(false)/; s/value\.writeString\(item\.ImageObjectKey\)/value.writeString("")/' \
  '^TestFrozenSnapshotDigestIncludesExpiryAndCoverObjectKey$' \
  'TestFrozenSnapshotDigestIncludesExpiryAndCoverObjectKey'

run_mutant \
  "effective expiry omitted from immutable digest" \
  "services/api/internal/quote/digest.go" \
  's/value\.writeInt64\(input\.ExpiresAt\.UTC\(\)\.UnixMicro\(\)\)/value.writeInt64(0)/' \
  '^TestFrozenSnapshotDigestIncludesExpiryAndCoverObjectKey$' \
  'TestFrozenSnapshotDigestIncludesExpiryAndCoverObjectKey'

run_mutant \
  "Quote receipt UNIQUE race returned as public conflict" \
  "services/api/internal/quote/receipts.go" \
  's/return ErrOperationReceiptExists/return ErrIdempotencyConflict/' \
  '^TestQuoteCreateReceiptUniqueRaceRollsBackThenReplaysInNewTransaction$' \
  'TestQuoteCreateReceiptUniqueRaceRollsBackThenReplaysInNewTransaction'

run_mutant \
  "Finalize ignores current cover drift" \
  "services/api/internal/quote/prepay.go" \
  's/hashProductSource\(record, snapshot\.Pickup\.Date\) != item\.ProductSourceVersion/false/' \
  '^TestFinalizeForPrepayInTxRevalidatesFactsWithoutDiscountOrVersionOnlyStaleness$' \
  'TestFinalizeForPrepayInTxRevalidatesFactsWithoutDiscountOrVersionOnlyStaleness'

run_mutant \
  "empty flavors collapse to null" \
  "services/api/internal/quote/provider.go" \
  's/append\(make\(\[\]string, 0, len\(item\.Flavors\)\), item\.Flavors\.\.\.\)/append([]string(nil), item.Flavors...)/' \
  '^TestLaterDiscountFactAffectsOnlyQuotesCreatedAfterIt$' \
  'TestLaterDiscountFactAffectsOnlyQuotesCreatedAfterIt'

run_mutant \
  "exact ten-minute deadline remains valid" \
  "services/api/internal/quote/prepay.go" \
  's/!observedAt\.Before\(snapshot\.ExpiresAt\)/observedAt.After(snapshot.ExpiresAt)/' \
  '^TestFinalizeForPrepayInTxUsesStrictTenMinuteBoundary$' \
  'TestFinalizeForPrepayInTxUsesStrictTenMinuteBoundary'

run_mutant \
  "discount version drift incorrectly makes quote stale" \
  "services/api/internal/quote/prepay.go" \
  's/identity\.Snapshot\.Kind != snapshot\.Identity\.Kind/settings.DiscountVersion != snapshot.Discount.Version || identity.Snapshot.Kind != snapshot.Identity.Kind/' \
  '^TestFinalizeForPrepayInTxRevalidatesFactsWithoutDiscountOrVersionOnlyStaleness$' \
  'TestFinalizeForPrepayInTxRevalidatesFactsWithoutDiscountOrVersionOnlyStaleness'

run_mutant \
  "frozen snapshot load rereads mutable settings" \
  "services/api/internal/quote/prepay.go" \
  's/header, found, err := readQuoteHeader\(ctx, transaction, "WHERE id=\?", quoteID\)/_, _ = readSourceSettings(ctx, transaction)\n\theader, found, err := readQuoteHeader(ctx, transaction, "WHERE id=?", quoteID)/' \
  '^TestLoadSnapshotInTxNeverReadsCurrentFacts$' \
  'TestLoadSnapshotInTxNeverReadsCurrentFacts'

run_mutant \
  "client-owned contact phone accepted" \
  "services/api/internal/quote/handler.go" \
  's/(ContactName string[[:space:]]+`json:"contact_name"`)/$1\n\tContactPhone string `json:"contact_phone"`/' \
  '^TestCreateContactContractRejectsMissingAndClientPhone$' \
  'TestCreateContactContractRejectsMissingAndClientPhone'

run_mutant \
  "contact name omitted from immutable digest" \
  "services/api/internal/quote/digest.go" \
  's/value\.writeString\(input\.Contact\.Name\)/value.writeString("")/' \
  '^TestContactSnapshotIsCoveredByImmutableDigest$' \
  'TestContactSnapshotIsCoveredByImmutableDigest'

run_mutant \
  "quote creation permits zero payment" \
  "services/api/internal/quote/provider.go" \
  's/if pricing\.PayableCents < 1 \{/if false {/' \
  '^TestCreateRejectsSubCentPaymentAmountWithoutWritingQuote$' \
  'TestCreateRejectsSubCentPaymentAmountWithoutWritingQuote'

run_mutant \
  "prepay finalization permits zero payment" \
  "services/api/internal/quote/prepay.go" \
  's/if snapshot\.PayableCents < 1 \{/if false {/' \
  '^TestFinalizeRejectsPersistedZeroPaymentSnapshot$' \
  'TestFinalizeRejectsPersistedZeroPaymentSnapshot'

run_mutant \
  "effective deadline ignores earlier pickup" \
  "services/api/internal/quote/provider.go" \
  's/if pickupUTC\.Before\(expiresAt\) \{/if false {/' \
  '^TestFinalizeUsesEarlierPickupAsEffectiveDeadline$' \
  'TestFinalizeUsesEarlierPickupAsEffectiveDeadline'

run_mutant \
  "pickup before creation is accepted" \
  "services/api/internal/quote/provider.go" \
  's/if !pickupUTC\.After\(createdUTC\) \{/if false {/' \
  '^TestLoadSnapshotFailsClosedWhenPickupPredatesCreation$' \
  'TestLoadSnapshotFailsClosedWhenPickupPredatesCreation'

run_mutant \
  "durable snapshot corruption becomes retryable unavailability" \
  "services/api/internal/quote/provider.go" \
  's/if errors\.Is\(err, ErrSnapshotInvalid\) \{/if false {/' \
  '^TestProviderFailsClosedWhenStoredSnapshotDigestDoesNotMatch$' \
  'TestProviderFailsClosedWhenStoredSnapshotDigestDoesNotMatch'

run_mutant \
  "extra phone and name omitted from identity resolution" \
  "services/api/internal/quote/provider.go" \
  's/Extra: user\.Extra/Extra: nil/' \
  '^TestProviderCreatesStaffQuoteFromMatchingExtraPhoneAndName$' \
  'TestProviderCreatesStaffQuoteFromMatchingExtraPhoneAndName'

run_mutant \
  "Quote create ignores closed or missing service date" \
  "services/api/internal/quote/provider.go" \
  's/if !serviceDateOpen \{/if serviceDateOpen \&\& false {/' \
  '^TestProviderRejectsMissingOrClosedServiceDate$' \
  'TestProviderRejectsMissingOrClosedServiceDate'

run_mutant \
  "Finalize ignores service date drift" \
  "services/api/internal/quote/prepay.go" \
  's/\|\| !serviceDateOpen \|\|/|| serviceDateOpen \&\& false ||/' \
  '^TestFinalizeForPrepayInTxRevalidatesFactsWithoutDiscountOrVersionOnlyStaleness$' \
  'TestFinalizeForPrepayInTxRevalidatesFactsWithoutDiscountOrVersionOnlyStaleness'

run_mutant \
  "Quote create accepts duplicate flavors" \
  "services/api/internal/quote/provider.go" \
  's/if _, duplicate := seenFlavors\[flavor\]; duplicate \{/if _, duplicate := seenFlavors[flavor]; duplicate \&\& false {/' \
  '^TestProviderRejectsDuplicateOrUnavailableFlavors$' \
  'TestProviderRejectsDuplicateOrUnavailableFlavors'

run_mutant \
  "Quote create accepts flavor outside storefront options" \
  "services/api/internal/quote/provider.go" \
  's/if !selectedFlavorsAvailable\(input\.Items, storefront\.FlavorOptions\) \{/if false {/' \
  '^TestProviderRejectsDuplicateOrUnavailableFlavors$' \
  'TestProviderRejectsDuplicateOrUnavailableFlavors'

run_mutant \
  "Finalize ignores storefront flavor drift" \
  "services/api/internal/quote/prepay.go" \
  's/\|\| !snapshotFlavorsAvailable\(snapshot\.Items, storefront\.FlavorOptions\)/|| false/' \
  '^TestFinalizeForPrepayInTxRevalidatesFactsWithoutDiscountOrVersionOnlyStaleness$' \
  'TestFinalizeForPrepayInTxRevalidatesFactsWithoutDiscountOrVersionOnlyStaleness'

run_mutant \
  "Quote pickup DTO regresses to meal" \
  "services/api/internal/quote/handler.go" \
  's/json:"meal_period"/json:"meal"/' \
  '^TestAuthenticatedUserCreatesAndReadsOwnImmutableQuote$' \
  'TestAuthenticatedUserCreatesAndReadsOwnImmutableQuote'
