package quotepricing_test

import (
	"errors"
	"fmt"
	"math"
	"reflect"
	"strconv"
	"sync"
	"testing"

	"github.com/gaofeng30/order/services/api/internal/quotepricing"
)

func requirePricingError(t *testing.T, got quotepricing.Result, err error, want quotepricing.ErrorKind) {
	t.Helper()
	if !reflect.DeepEqual(got, quotepricing.Result{}) {
		t.Fatalf("Calculate() result = %+v, want zero Result", got)
	}
	var pricingError *quotepricing.Error
	if !errors.As(err, &pricingError) {
		t.Fatalf("Calculate() error = %v, want *quotepricing.Error", err)
	}
	if pricingError.Kind() != want {
		t.Fatalf("Calculate() error kind = %q, want %q", pricingError.Kind(), want)
	}
	if err.Error() != "quotepricing: "+string(want) {
		t.Fatalf("Calculate() error text = %q, want stable redacted text", err)
	}
}

func TestCalculateRoundsUnitHalfUpBeforeQuantity(t *testing.T) {
	got, err := quotepricing.Calculate(quotepricing.Input{
		RatePercent: 85,
		Lines: []quotepricing.Line{
			{UnitPriceCents: 101, Quantity: 2},
		},
	})
	if err != nil {
		t.Fatalf("Calculate() error = %v", err)
	}
	want := quotepricing.Result{
		RatePercent: 85,
		Lines: []quotepricing.LineResult{
			{
				OriginalUnitPriceCents:   101,
				DiscountedUnitPriceCents: 86,
				Quantity:                 2,
				OriginalSubtotalCents:    202,
				PayableSubtotalCents:     172,
			},
		},
		OriginalSubtotalCents: 202,
		DiscountCents:         30,
		PayableCents:          172,
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Calculate() = %+v, want %+v", got, want)
	}
}

func TestCalculateRoundsEachUnitBeforeSumming(t *testing.T) {
	got, err := quotepricing.Calculate(quotepricing.Input{
		RatePercent: 50,
		Lines: []quotepricing.Line{
			{UnitPriceCents: 1, Quantity: 1},
			{UnitPriceCents: 1, Quantity: 1},
		},
	})
	if err != nil {
		t.Fatalf("Calculate() error = %v", err)
	}
	want := quotepricing.Result{
		RatePercent: 50,
		Lines: []quotepricing.LineResult{
			{
				OriginalUnitPriceCents:   1,
				DiscountedUnitPriceCents: 1,
				Quantity:                 1,
				OriginalSubtotalCents:    1,
				PayableSubtotalCents:     1,
			},
			{
				OriginalUnitPriceCents:   1,
				DiscountedUnitPriceCents: 1,
				Quantity:                 1,
				OriginalSubtotalCents:    1,
				PayableSubtotalCents:     1,
			},
		},
		OriginalSubtotalCents: 2,
		DiscountCents:         0,
		PayableCents:          2,
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Calculate() = %+v, want %+v", got, want)
	}
}

func TestCalculateMultipliesQuantityAfterUnitRounding(t *testing.T) {
	got, err := quotepricing.Calculate(quotepricing.Input{
		RatePercent: 50,
		Lines: []quotepricing.Line{
			{UnitPriceCents: 5, Quantity: 3},
		},
	})
	if err != nil {
		t.Fatalf("Calculate() error = %v", err)
	}
	want := quotepricing.Result{
		RatePercent: 50,
		Lines: []quotepricing.LineResult{
			{
				OriginalUnitPriceCents:   5,
				DiscountedUnitPriceCents: 3,
				Quantity:                 3,
				OriginalSubtotalCents:    15,
				PayableSubtotalCents:     9,
			},
		},
		OriginalSubtotalCents: 15,
		DiscountCents:         6,
		PayableCents:          9,
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Calculate() = %+v, want %+v", got, want)
	}
}

func TestCalculateRejectsRateOutsideMathematicalRange(t *testing.T) {
	for _, ratePercent := range []int64{-1, 101} {
		t.Run(strconv.FormatInt(ratePercent, 10), func(t *testing.T) {
			got, err := quotepricing.Calculate(quotepricing.Input{
				RatePercent: ratePercent,
				Lines: []quotepricing.Line{
					{UnitPriceCents: 10, Quantity: 1},
				},
			})
			requirePricingError(t, got, err, quotepricing.ErrorInvalidRate)
		})
	}
}

func TestCalculateAcceptsZeroAndOneHundredPercent(t *testing.T) {
	tests := []struct {
		name                string
		ratePercent         int64
		discountedUnitPrice int64
		payable             int64
		discount            int64
	}{
		{name: "zero", ratePercent: 0, discountedUnitPrice: 0, payable: 0, discount: 20},
		{name: "one hundred", ratePercent: 100, discountedUnitPrice: 10, payable: 20, discount: 0},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := quotepricing.Calculate(quotepricing.Input{
				RatePercent: test.ratePercent,
				Lines: []quotepricing.Line{
					{UnitPriceCents: 10, Quantity: 2},
				},
			})
			if err != nil {
				t.Fatalf("Calculate() error = %v", err)
			}
			want := quotepricing.Result{
				RatePercent: test.ratePercent,
				Lines: []quotepricing.LineResult{
					{
						OriginalUnitPriceCents:   10,
						DiscountedUnitPriceCents: test.discountedUnitPrice,
						Quantity:                 2,
						OriginalSubtotalCents:    20,
						PayableSubtotalCents:     test.payable,
					},
				},
				OriginalSubtotalCents: 20,
				DiscountCents:         test.discount,
				PayableCents:          test.payable,
			}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("Calculate() = %+v, want %+v", got, want)
			}
		})
	}
}

func TestCalculateRejectsNegativeUnitPrice(t *testing.T) {
	got, err := quotepricing.Calculate(quotepricing.Input{
		RatePercent: 50,
		Lines: []quotepricing.Line{
			{UnitPriceCents: -1, Quantity: 1},
		},
	})
	requirePricingError(t, got, err, quotepricing.ErrorInvalidPrice)
}

func TestCalculateAcceptsZeroPrice(t *testing.T) {
	got, err := quotepricing.Calculate(quotepricing.Input{
		RatePercent: 85,
		Lines: []quotepricing.Line{
			{UnitPriceCents: 0, Quantity: 3},
		},
	})
	if err != nil {
		t.Fatalf("Calculate() error = %v", err)
	}
	want := quotepricing.Result{
		RatePercent: 85,
		Lines: []quotepricing.LineResult{
			{Quantity: 3},
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Calculate() = %+v, want %+v", got, want)
	}
}

func TestCalculateRejectsNonPositiveQuantity(t *testing.T) {
	for _, quantity := range []int64{0, -1} {
		t.Run(strconv.FormatInt(quantity, 10), func(t *testing.T) {
			got, err := quotepricing.Calculate(quotepricing.Input{
				RatePercent: 50,
				Lines: []quotepricing.Line{
					{UnitPriceCents: 10, Quantity: quantity},
				},
			})
			requirePricingError(t, got, err, quotepricing.ErrorInvalidQuantity)
		})
	}
}

func TestCalculateRejectsEmptyLines(t *testing.T) {
	got, err := quotepricing.Calculate(quotepricing.Input{RatePercent: 50})
	requirePricingError(t, got, err, quotepricing.ErrorEmptyLines)
}

func TestCalculateRejectsInvalidRateBeforeEmptyLines(t *testing.T) {
	got, err := quotepricing.Calculate(quotepricing.Input{RatePercent: -1})
	requirePricingError(t, got, err, quotepricing.ErrorInvalidRate)
}

func TestCalculateRejectsNegativePriceBeforeNonPositiveQuantity(t *testing.T) {
	got, err := quotepricing.Calculate(quotepricing.Input{
		RatePercent: 50,
		Lines: []quotepricing.Line{
			{UnitPriceCents: -1, Quantity: 0},
		},
	})
	requirePricingError(t, got, err, quotepricing.ErrorInvalidPrice)
}

func TestCalculateRejectsOriginalLineMultiplicationOverflow(t *testing.T) {
	got, err := quotepricing.Calculate(quotepricing.Input{
		RatePercent: 0,
		Lines: []quotepricing.Line{
			{UnitPriceCents: math.MaxInt64, Quantity: 2},
		},
	})
	requirePricingError(t, got, err, quotepricing.ErrorOverflow)
}

func TestCalculateRejectsDiscountMultiplicationOverflow(t *testing.T) {
	got, err := quotepricing.Calculate(quotepricing.Input{
		RatePercent: 2,
		Lines: []quotepricing.Line{
			{UnitPriceCents: math.MaxInt64, Quantity: 1},
		},
	})
	requirePricingError(t, got, err, quotepricing.ErrorOverflow)
}

func TestCalculateRejectsCrossLineSumOverflow(t *testing.T) {
	got, err := quotepricing.Calculate(quotepricing.Input{
		RatePercent: 0,
		Lines: []quotepricing.Line{
			{UnitPriceCents: math.MaxInt64, Quantity: 1},
			{UnitPriceCents: 1, Quantity: 1},
		},
	})
	requirePricingError(t, got, err, quotepricing.ErrorOverflow)
}

func TestCalculateDiscardsPartialResultOnLaterInvalidLine(t *testing.T) {
	got, err := quotepricing.Calculate(quotepricing.Input{
		RatePercent: 50,
		Lines: []quotepricing.Line{
			{UnitPriceCents: 10, Quantity: 1},
			{UnitPriceCents: 20, Quantity: 0},
		},
	})
	requirePricingError(t, got, err, quotepricing.ErrorInvalidQuantity)
}

func TestCalculatePreservesLineOrderAndCallerInput(t *testing.T) {
	lines := []quotepricing.Line{
		{UnitPriceCents: 11, Quantity: 2},
		{UnitPriceCents: 5, Quantity: 3},
		{UnitPriceCents: 0, Quantity: 4},
	}
	before := append([]quotepricing.Line(nil), lines...)
	got, err := quotepricing.Calculate(quotepricing.Input{RatePercent: 50, Lines: lines})
	if err != nil {
		t.Fatalf("Calculate() error = %v", err)
	}
	want := quotepricing.Result{
		RatePercent: 50,
		Lines: []quotepricing.LineResult{
			{OriginalUnitPriceCents: 11, DiscountedUnitPriceCents: 6, Quantity: 2, OriginalSubtotalCents: 22, PayableSubtotalCents: 12},
			{OriginalUnitPriceCents: 5, DiscountedUnitPriceCents: 3, Quantity: 3, OriginalSubtotalCents: 15, PayableSubtotalCents: 9},
			{OriginalUnitPriceCents: 0, DiscountedUnitPriceCents: 0, Quantity: 4, OriginalSubtotalCents: 0, PayableSubtotalCents: 0},
		},
		OriginalSubtotalCents: 37,
		DiscountCents:         16,
		PayableCents:          21,
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Calculate() = %+v, want %+v", got, want)
	}
	if !reflect.DeepEqual(lines, before) {
		t.Fatalf("caller lines changed: got %+v, want %+v", lines, before)
	}
}

func TestCalculateIsRepeatedlyDeterministic(t *testing.T) {
	input := quotepricing.Input{
		RatePercent: 85,
		Lines: []quotepricing.Line{
			{UnitPriceCents: 101, Quantity: 2},
			{UnitPriceCents: 1, Quantity: 1},
		},
	}
	want, err := quotepricing.Calculate(input)
	if err != nil {
		t.Fatalf("baseline Calculate() error = %v", err)
	}
	for iteration := 0; iteration < 100; iteration++ {
		got, gotErr := quotepricing.Calculate(input)
		if gotErr != nil {
			t.Fatalf("Calculate() iteration %d error = %v", iteration, gotErr)
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("Calculate() iteration %d = %+v, want %+v", iteration, got, want)
		}
	}
}

func TestCalculateFailureIsRepeatedlyDeterministic(t *testing.T) {
	input := quotepricing.Input{
		RatePercent: 50,
		Lines: []quotepricing.Line{
			{UnitPriceCents: 10, Quantity: 0},
		},
	}
	for iteration := 0; iteration < 100; iteration++ {
		got, err := quotepricing.Calculate(input)
		requirePricingError(t, got, err, quotepricing.ErrorInvalidQuantity)
	}
}

func TestCalculateIsConcurrentlyDeterministic(t *testing.T) {
	input := quotepricing.Input{
		RatePercent: 50,
		Lines: []quotepricing.Line{
			{UnitPriceCents: 11, Quantity: 2},
			{UnitPriceCents: 5, Quantity: 3},
		},
	}
	want, err := quotepricing.Calculate(input)
	if err != nil {
		t.Fatalf("baseline Calculate() error = %v", err)
	}
	const workers = 32
	failures := make(chan error, workers)
	var wait sync.WaitGroup
	for worker := 0; worker < workers; worker++ {
		wait.Add(1)
		go func(worker int) {
			defer wait.Done()
			for iteration := 0; iteration < 100; iteration++ {
				got, gotErr := quotepricing.Calculate(input)
				if gotErr != nil {
					failures <- fmt.Errorf("worker %d iteration %d error: %w", worker, iteration, gotErr)
					return
				}
				if !reflect.DeepEqual(got, want) {
					failures <- fmt.Errorf("worker %d iteration %d produced a different result", worker, iteration)
					return
				}
			}
		}(worker)
	}
	wait.Wait()
	close(failures)
	for failure := range failures {
		t.Error(failure)
	}
}
