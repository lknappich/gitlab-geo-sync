package reconciler

import (
	"context"
	"time"

	"github.com/rs/zerolog/log"
)

// Retry calls fn up to maxAttempts times with exponential backoff, stopping
// on success or context cancellation. The first retry waits initialDelay;
// each subsequent retry doubles the delay (capped at maxDelay). A nil error
// from fn stops the loop. The last non-nil error is returned.
func Retry(ctx context.Context, name string, maxAttempts int, initialDelay, maxDelay time.Duration, fn func() error) error {
	var lastErr error
	delay := initialDelay
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		if err := fn(); err != nil {
			lastErr = err
			if attempt == maxAttempts {
				break
			}
			log.Warn().Err(err).Str("component", name).Int("attempt", attempt).
				Int("max_attempts", maxAttempts).
				Dur("retry_in", delay).
				Msg("transient failure, retrying")
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}
			delay *= 2
			if delay > maxDelay {
				delay = maxDelay
			}
			continue
		}
		return nil
	}
	return lastErr
}
