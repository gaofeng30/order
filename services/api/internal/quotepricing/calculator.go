package quotepricing

import "math"

const percentageDivisor int64 = 100

// Input is a frozen discount rate and ordered price/quantity lines.
type Input struct {
	RatePercent int64
	Lines       []Line
}

// Line is one unit price and quantity supplied by the caller.
type Line struct {
	UnitPriceCents int64
	Quantity       int64
}

// LineResult is the calculated monetary result for one input line.
type LineResult struct {
	OriginalUnitPriceCents   int64
	DiscountedUnitPriceCents int64
	Quantity                 int64
	OriginalSubtotalCents    int64
	PayableSubtotalCents     int64
}

// Result is the ordered line and total result of a calculation.
type Result struct {
	RatePercent           int64
	Lines                 []LineResult
	OriginalSubtotalCents int64
	DiscountCents         int64
	PayableCents          int64
}

// Calculate calculates a quote from a frozen discount rate and ordered lines.
func Calculate(input Input) (Result, error) {
	if input.RatePercent < 0 || input.RatePercent > 100 {
		return Result{}, newError(ErrorInvalidRate)
	}
	if len(input.Lines) == 0 {
		return Result{}, newError(ErrorEmptyLines)
	}
	result := Result{
		RatePercent: input.RatePercent,
		Lines:       make([]LineResult, 0, len(input.Lines)),
	}
	for _, line := range input.Lines {
		if line.UnitPriceCents < 0 {
			return Result{}, newError(ErrorInvalidPrice)
		}
		if line.Quantity <= 0 {
			return Result{}, newError(ErrorInvalidQuantity)
		}
		discountedUnitPriceCents, ok := discountedPrice(line.UnitPriceCents, input.RatePercent)
		if !ok {
			return Result{}, newError(ErrorOverflow)
		}
		originalSubtotalCents, ok := checkedMultiply(line.UnitPriceCents, line.Quantity)
		if !ok {
			return Result{}, newError(ErrorOverflow)
		}
		payableSubtotalCents, ok := checkedMultiply(discountedUnitPriceCents, line.Quantity)
		if !ok {
			return Result{}, newError(ErrorOverflow)
		}
		result.Lines = append(result.Lines, LineResult{
			OriginalUnitPriceCents:   line.UnitPriceCents,
			DiscountedUnitPriceCents: discountedUnitPriceCents,
			Quantity:                 line.Quantity,
			OriginalSubtotalCents:    originalSubtotalCents,
			PayableSubtotalCents:     payableSubtotalCents,
		})
		nextOriginalTotal, ok := checkedAdd(result.OriginalSubtotalCents, originalSubtotalCents)
		if !ok {
			return Result{}, newError(ErrorOverflow)
		}
		result.OriginalSubtotalCents = nextOriginalTotal
		nextPayableTotal, ok := checkedAdd(result.PayableCents, payableSubtotalCents)
		if !ok {
			return Result{}, newError(ErrorOverflow)
		}
		result.PayableCents = nextPayableTotal
	}
	result.DiscountCents = result.OriginalSubtotalCents - result.PayableCents
	return result, nil
}

func discountedPrice(unitPriceCents, ratePercent int64) (int64, bool) {
	product, ok := checkedMultiply(unitPriceCents, ratePercent)
	if !ok {
		return 0, false
	}
	rounded := product / percentageDivisor
	if product%percentageDivisor >= percentageDivisor/2 {
		rounded, ok = checkedAdd(rounded, 1)
		if !ok {
			return 0, false
		}
	}
	return rounded, true
}

func checkedMultiply(left, right int64) (int64, bool) {
	if left != 0 && right > math.MaxInt64/left {
		return 0, false
	}
	return left * right, true
}

func checkedAdd(left, right int64) (int64, bool) {
	if right > math.MaxInt64-left {
		return 0, false
	}
	return left + right, true
}
