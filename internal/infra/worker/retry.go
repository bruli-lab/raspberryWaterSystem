package worker

import (
	"context"
	"log/slog"
	"time"
)

func RetryUntilDone(ctx context.Context, log *slog.Logger, name string, retryDelay time.Duration, run func(context.Context) error) {
	for {
		err := run(ctx)
		if err == nil || ctx.Err() != nil {
			return
		}
		log.ErrorContext(ctx, "[WORKER] failed, retrying",
			slog.String("worker", name),
			slog.String("error", err.Error()),
			slog.Duration("retry_in", retryDelay))
		timer := time.NewTimer(retryDelay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
	}
}
