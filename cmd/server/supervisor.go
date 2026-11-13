package main

import (
	"context"
	"log/slog"
	"time"

	"github.com/bruli/raspberryWaterSystem/internal/app"
	"github.com/bruli/raspberryWaterSystem/internal/infra/worker"
)

const workerRestartDelay = 5 * time.Second

type relaysExecutor interface {
	app.PinExecutor
	TurnOffAll(ctx context.Context) error
	Stop(ctx context.Context) error
}

func superviseWorker(ctx context.Context, log *slog.Logger, name string, run func(context.Context)) {
	worker.Supervise(ctx, log, name, workerRestartDelay, run)
}
