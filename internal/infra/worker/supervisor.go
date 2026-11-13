package worker

import (
	"context"
	"log/slog"
	"runtime/debug"
	"time"
)

func Supervise(ctx context.Context, log *slog.Logger, name string, restartDelay time.Duration, run func(context.Context)) {
	for {
		if finished := runRecovering(ctx, log, name, run); finished {
			return
		}
		timer := time.NewTimer(restartDelay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
	}
}

func runRecovering(ctx context.Context, log *slog.Logger, name string, run func(context.Context)) (finished bool) {
	defer func() {
		if r := recover(); r != nil {
			log.ErrorContext(ctx, "[WORKER] panic recovered, restarting worker",
				slog.String("worker", name),
				slog.Any("panic", r),
				slog.String("stack", string(debug.Stack())))
			finished = false
		}
	}()
	run(ctx)
	return true
}
