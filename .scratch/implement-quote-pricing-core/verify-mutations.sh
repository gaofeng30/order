#!/usr/bin/env bash
set -euo pipefail

mutation_repo=$(git rev-parse --show-toplevel)
mutation_root=$(mktemp -d /private/tmp/quote-pricing-mutations.XXXXXX)

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
  local mutation_calculator="${mutation_copy}/services/api/internal/quotepricing/calculator.go"

  mkdir -p "${mutation_copy}/services/api/internal/quotepricing"
  cp "${mutation_repo}/go.mod" "${mutation_repo}/go.sum" "${mutation_copy}/"
  cp "${mutation_repo}/services/api/internal/quotepricing/"*.go \
    "${mutation_copy}/services/api/internal/quotepricing/"

  local mutation_original_count
  mutation_original_count=$(grep -Foc "${mutation_original}" "${mutation_calculator}" || true)
  if [[ ${mutation_original_count} -ne 1 ]]; then
    printf 'mutation source count=%s, want 1: %s\n' \
      "${mutation_original_count}" "${mutation_name}" >&2
    exit 79
  fi
  perl -0pi -e "${mutation_expression}" "${mutation_calculator}"
  if grep -Fq "${mutation_original}" "${mutation_calculator}" || \
    ! grep -Fq "${mutation_expected}" "${mutation_calculator}"; then
    printf 'mutation was not applied: %s\n' "${mutation_name}" >&2
    exit 80
  fi

  set +e
  (
    cd "${mutation_copy}"
    GOPROXY=off GOTOOLCHAIN=go1.26.5 go test ./services/api/internal/quotepricing \
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

run_mutation \
  half_up_threshold \
  'if product%percentageDivisor >= percentageDivisor/2 {' \
  's#if product%percentageDivisor >= percentageDivisor/2 \{#if product%percentageDivisor > percentageDivisor/2 {#' \
  'if product%percentageDivisor > percentageDivisor/2 {' \
  '^TestCalculateRoundsEachUnitBeforeSumming$' \
  '--- FAIL: TestCalculateRoundsEachUnitBeforeSumming'

run_mutation \
  aggregate_subtotal_discount \
  'nextPayableTotal, ok := checkedAdd(result.PayableCents, payableSubtotalCents)' \
  's#nextPayableTotal, ok := checkedAdd\(result\.PayableCents, payableSubtotalCents\)#nextPayableTotal, ok := discountedPrice(nextOriginalTotal, input.RatePercent)#' \
  'nextPayableTotal, ok := discountedPrice(nextOriginalTotal, input.RatePercent)' \
  '^TestCalculateRoundsEachUnitBeforeSumming$' \
  '--- FAIL: TestCalculateRoundsEachUnitBeforeSumming'

run_mutation \
  quantity_before_unit_rounding \
  'payableSubtotalCents, ok := checkedMultiply(discountedUnitPriceCents, line.Quantity)' \
  's#payableSubtotalCents, ok := checkedMultiply\(discountedUnitPriceCents, line\.Quantity\)#payableSubtotalCents, ok := discountedPrice(originalSubtotalCents, input.RatePercent)#' \
  'payableSubtotalCents, ok := discountedPrice(originalSubtotalCents, input.RatePercent)' \
  '^TestCalculateMultipliesQuantityAfterUnitRounding$' \
  '--- FAIL: TestCalculateMultipliesQuantityAfterUnitRounding'

run_mutation \
  reject_one_hundred_percent \
  'if input.RatePercent < 0 || input.RatePercent > 100 {' \
  's#if input\.RatePercent < 0 \|\| input\.RatePercent > 100 \{#if input.RatePercent < 0 || input.RatePercent >= 100 {#' \
  'if input.RatePercent < 0 || input.RatePercent >= 100 {' \
  '^TestCalculateAcceptsZeroAndOneHundredPercent$' \
  '--- FAIL: TestCalculateAcceptsZeroAndOneHundredPercent'

run_mutation \
  empty_lines_precede_invalid_rate \
  'if input.RatePercent < 0 || input.RatePercent > 100 {' \
  's#if input\.RatePercent < 0 \|\| input\.RatePercent > 100 \{#if len(input.Lines) == 0 { return Result{}, newError(ErrorEmptyLines) }; if input.RatePercent > 100 || input.RatePercent < 0 {#' \
  'if len(input.Lines) == 0 { return Result{}, newError(ErrorEmptyLines) }; if input.RatePercent > 100 || input.RatePercent < 0 {' \
  '^TestCalculateRejectsInvalidRateBeforeEmptyLines$' \
  '--- FAIL: TestCalculateRejectsInvalidRateBeforeEmptyLines'

run_mutation \
  accept_empty_lines \
  'if len(input.Lines) == 0 {' \
  's#if len\(input\.Lines\) == 0 \{#if len(input.Lines) < 0 {#' \
  'if len(input.Lines) < 0 {' \
  '^TestCalculateRejectsEmptyLines$' \
  '--- FAIL: TestCalculateRejectsEmptyLines'

run_mutation \
  quantity_precedes_negative_price \
  'if line.UnitPriceCents < 0 {' \
  's#if line\.UnitPriceCents < 0 \{#if line.Quantity <= 0 { return Result{}, newError(ErrorInvalidQuantity) }; if 0 > line.UnitPriceCents {#' \
  'if line.Quantity <= 0 { return Result{}, newError(ErrorInvalidQuantity) }; if 0 > line.UnitPriceCents {' \
  '^TestCalculateRejectsNegativePriceBeforeNonPositiveQuantity$' \
  '--- FAIL: TestCalculateRejectsNegativePriceBeforeNonPositiveQuantity'

run_mutation \
  unchecked_original_line_multiplication \
  'originalSubtotalCents, ok := checkedMultiply(line.UnitPriceCents, line.Quantity)' \
  's#originalSubtotalCents, ok := checkedMultiply\(line\.UnitPriceCents, line\.Quantity\)#originalSubtotalCents, ok := line.UnitPriceCents * line.Quantity, true#' \
  'originalSubtotalCents, ok := line.UnitPriceCents * line.Quantity, true' \
  '^TestCalculateRejectsOriginalLineMultiplicationOverflow$' \
  '--- FAIL: TestCalculateRejectsOriginalLineMultiplicationOverflow'

run_mutation \
  unchecked_discount_multiplication \
  'product, ok := checkedMultiply(unitPriceCents, ratePercent)' \
  's#product, ok := checkedMultiply\(unitPriceCents, ratePercent\)#product, ok := unitPriceCents * ratePercent, true#' \
  'product, ok := unitPriceCents * ratePercent, true' \
  '^TestCalculateRejectsDiscountMultiplicationOverflow$' \
  '--- FAIL: TestCalculateRejectsDiscountMultiplicationOverflow'

run_mutation \
  unchecked_cross_line_addition \
  'nextOriginalTotal, ok := checkedAdd(result.OriginalSubtotalCents, originalSubtotalCents)' \
  's#nextOriginalTotal, ok := checkedAdd\(result\.OriginalSubtotalCents, originalSubtotalCents\)#nextOriginalTotal, ok := result.OriginalSubtotalCents + originalSubtotalCents, true#' \
  'nextOriginalTotal, ok := result.OriginalSubtotalCents + originalSubtotalCents, true' \
  '^TestCalculateRejectsCrossLineSumOverflow$' \
  '--- FAIL: TestCalculateRejectsCrossLineSumOverflow'

run_mutation \
  return_partial_result_on_error \
  'return Result{}, newError(ErrorInvalidQuantity)' \
  's#return Result\{\}, newError\(ErrorInvalidQuantity\)#return result, newError(ErrorInvalidQuantity)#' \
  'return result, newError(ErrorInvalidQuantity)' \
  '^TestCalculateDiscardsPartialResultOnLaterInvalidLine$' \
  '--- FAIL: TestCalculateDiscardsPartialResultOnLaterInvalidLine'
