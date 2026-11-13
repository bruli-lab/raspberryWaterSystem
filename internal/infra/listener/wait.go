package listener

import (
	"context"
	"time"
)

type Wait func(ctx context.Context, d time.Duration) error

func WaitFor(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
