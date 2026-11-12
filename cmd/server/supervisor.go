package main

import (
	"context"
	"log/slog"
	"runtime/debug"
	"time"

	"github.com/bruli/raspberryWaterSystem/internal/app"
)

const workerRestartDelay = 5 * time.Second

type relaysExecutor interface {
	app.PinExecutor
	TurnOffAll(ctx context.Context) error
	Stop(ctx context.Context) error
}

func superviseWorker(ctx context.Context, log *slog.Logger, name string, worker func(context.Context)) {
	superviseWorkerWithDelay(ctx, log, name, workerRestartDelay, worker)
}

func superviseWorkerWithDelay(ctx context.Context, log *slog.Logger, name string, restartDelay time.Duration, worker func(context.Context)) {
	for {
		if finished := runRecovering(ctx, log, name, worker); finished {
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

func runRecovering(ctx context.Context, log *slog.Logger, name string, worker func(context.Context)) (finished bool) {
	defer func() {
		if r := recover(); r != nil {
			log.ErrorContext(ctx, "[WORKER] panic recovered, restarting worker",
				slog.String("worker", name),
				slog.Any("panic", r),
				slog.String("stack", string(debug.Stack())))
			finished = false
		}
	}()
	worker(ctx)
	return true
}
