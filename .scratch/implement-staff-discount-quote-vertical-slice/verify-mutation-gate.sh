#!/usr/bin/env bash
set -euo pipefail

quote_mutation_repo=$(git rev-parse --show-toplevel)
quote_mutation_tests='^(TestProviderCreatesStaffQuoteFromOneLockedServerSnapshot|TestProviderCreatesStaffQuoteFromMatchingExtraPhoneAndName|TestProviderRejectsMissingOrClosedServiceDate|TestProviderRejectsDuplicateOrUnavailableFlavors|TestProviderReplaysCompleteStoredQuoteAndConflictsWithoutRepricing|TestCreateRejectsClientOwnedMoneyWithoutCallingApplication|TestProviderFailsClosedWhenStoredSnapshotDigestDoesNotMatch|TestLaterDiscountFactAffectsOnlyQuotesCreatedAfterIt|TestFinalizeForPrepayInTxUsesStrictTenMinuteBoundary|TestFinalizeForPrepayInTxRevalidatesFactsWithoutDiscountOrVersionOnlyStaleness|TestLoadSnapshotInTxNeverReadsCurrentFacts|TestCreateContactContractRejectsMissingAndClientPhone|TestContactSnapshotIsCoveredByImmutableDigest|TestCreateRejectsSubCentPaymentAmountWithoutWritingQuote|TestFinalizeRejectsPersistedZeroPaymentSnapshot|TestFinalizeUsesEarlierPickupAsEffectiveDeadline|TestLoadSnapshotFailsClosedWhenPickupPredatesCreation|TestFrozenSnapshotDigestIncludesExpiryAndCoverObjectKey|TestQuoteCreateReceiptUniqueRaceRollsBackThenReplaysInNewTransaction|TestAuthenticatedUserCreatesAndReadsOwnImmutableQuote)$'

cd "${quote_mutation_repo}"
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
  go test ./services/api/internal/quote -run "${quote_mutation_tests}" -count=1
bash "${quote_mutation_repo}/.scratch/implement-staff-discount-quote-vertical-slice/mutation.sh"
