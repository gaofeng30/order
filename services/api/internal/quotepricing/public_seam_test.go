package quotepricing_test

import "github.com/gaofeng30/order/services/api/internal/quotepricing"

var _ func(quotepricing.Input) (quotepricing.Result, error) = quotepricing.Calculate
