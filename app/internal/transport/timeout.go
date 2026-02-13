package transport

import (
	"context"
	"time"
)

func EffectiveTimeout(ctx context.Context, fallback time.Duration) time.Duration {
	if ctx == nil {
		return fallback
	}
	if deadline, ok := ctx.Deadline(); ok {
		remaining := time.Until(deadline)
		if remaining <= 0 {
			return 1 * time.Millisecond
		}
		if remaining < fallback {
			return remaining
		}
	}
	return fallback
}
