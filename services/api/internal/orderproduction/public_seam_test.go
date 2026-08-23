package orderproduction_test

import (
	"time"

	"github.com/gaofeng30/order/services/api/internal/orderproduction"
)

var (
	_ func(time.Time, time.Time) (orderproduction.State, error)                           = orderproduction.InitialState
	_ func(orderproduction.State, time.Time, time.Time) (orderproduction.Decision, error) = orderproduction.Advance
	_ func(orderproduction.TransitionInput) (orderproduction.Decision, error)             = orderproduction.Transition
)
