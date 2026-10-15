package cloudbasepg

import (
	"context"
	"time"
)

// MaintainAuthentication renews the ordinary login before expiry, including
// while the API is idle. No business work, leases or retries live in this loop:
// PostgreSQL retains and schedules those across process loss. Every successful
// login publishes its credential before being made available to RPC callers.
// Startup takes the same path, so restart restores authentication automatically.
func (c *Client) MaintainAuthentication(ctx context.Context) {
	for ctx.Err() == nil {
		_, err := c.accessToken(ctx)
		delay := 30 * time.Second
		if err == nil {
			c.mu.Lock()
			delay = c.cached.usableUntil.Sub(c.now())
			c.mu.Unlock()
			if delay <= 0 {
				delay = time.Second
			}
		}
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
	}
}
