package compensation

import (
	"context"
	"errors"
)

var errUnavailable = errors.New("compensation execution unavailable")

// Store owns atomic capability consumption and durable completion/retry state.
// Claim must reject replay and expired tokens across processes, not in memory.
type Target struct {
	Kind string
	ID   uint64
}

type Store interface {
	Claim(context.Context, string) (target Target, accepted bool, err error)
	Finish(context.Context, string, bool) error
}

func NewExecutor(store Store, runners map[string]func(context.Context, uint64) error) func(context.Context, string) error {
	// Freeze the allowed dispatch map when composing the application.
	allowed := make(map[string]func(context.Context, uint64) error, len(runners))
	for _, kind := range []string{"payment", "refund", "subscription"} {
		if run := runners[kind]; run != nil {
			allowed[kind] = run
		}
	}
	return func(ctx context.Context, token string) error {
		if store == nil {
			return errUnavailable
		}
		target, accepted, err := store.Claim(ctx, token)
		if err != nil {
			return errUnavailable
		}
		if !accepted {
			return ErrUnauthorized
		}
		run := allowed[target.Kind]
		if run == nil || target.ID == 0 {
			return errUnavailable
		}
		runErr := run(ctx, target.ID)
		// A canceled request cannot confirm persistence. The durable database wake
		// remains responsible for recovery if this completion cannot commit.
		if err := store.Finish(ctx, token, runErr == nil); err != nil {
			return errUnavailable
		}
		if runErr != nil {
			return errUnavailable
		}
		return nil
	}
}
