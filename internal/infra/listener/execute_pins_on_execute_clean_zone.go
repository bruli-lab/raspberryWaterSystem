package listener

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"runtime/debug"
	"time"

	"github.com/bruli-lab/go-core/cqs"
	"github.com/bruli-lab/go-core/event"
	"github.com/bruli/raspberryWaterSystem/internal/app"
	"github.com/bruli/raspberryWaterSystem/internal/domain/program"
	"github.com/bruli/raspberryWaterSystem/internal/domain/zone"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/sync/errgroup"
)

type ExecutePinsOnExecuteCleanZone struct {
	ch    cqs.CommandHandler
	wait  Wait
	trace trace.Tracer
	log   *slog.Logger
}

func (e ExecutePinsOnExecuteCleanZone) Listen(ctx context.Context, ev event.Event) error {
	ctx, span := e.trace.Start(ctx, "executePinsOnExecuteCleanZone.Listen")
	defer span.End()
	executed, ok := ev.(zone.CleanZoneExecuted)
	if !ok {
		err := errors.New("wrong event type")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	go e.runCleanCycle(ctx, &executed)
	span.SetStatus(codes.Ok, "clean cycle started")
	return nil
}

func (e ExecutePinsOnExecuteCleanZone) runCleanCycle(ctx context.Context, executed *zone.CleanZoneExecuted) {
	ctx, span := e.trace.Start(ctx, "executePinsOnExecuteCleanZone.cleanCycle")
	defer span.End()
	defer e.finishCleaning(ctx, span)
	defer func() {
		if r := recover(); r != nil {
			err := fmt.Errorf("clean cycle panicked: %v", r)
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			e.log.ErrorContext(ctx, "clean cycle panicked", slog.Any("panic", r), slog.String("stack", string(debug.Stack())))
		}
	}()
	if err := e.cleanCycle(ctx, span, executed); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		e.log.ErrorContext(ctx, "failed running clean cycle", slog.String("zone", executed.ZoneName), slog.String("error", err.Error()))
		return
	}
	span.SetStatus(codes.Ok, "clean executed")
}

func (e ExecutePinsOnExecuteCleanZone) cleanCycle(ctx context.Context, span trace.Span, executed *zone.CleanZoneExecuted) error {
	if err := e.cleanExecution(ctx, executed); err != nil {
		return err
	}
	sec := program.HydrateSeconds(int(executed.ZoneSeconds))
	e.notify(ctx, span, fmt.Sprintf("%s clean zone executed during %vs", executed.ZoneName, sec.Int()))
	e.log.DebugContext(ctx, "wait for clarify process", slog.Duration("duration", executed.WaitDuration))
	if err := e.wait(ctx, executed.WaitDuration); err != nil {
		return err
	}
	if err := e.clarifyExecution(ctx, executed); err != nil {
		return err
	}
	e.notify(ctx, span, fmt.Sprintf("clarify zone %s finished!!", executed.ZoneName))
	return nil
}

func (e ExecutePinsOnExecuteCleanZone) finishCleaning(ctx context.Context, span trace.Span) {
	if _, err := e.ch.Handle(context.WithoutCancel(ctx), app.ChangeCleanStatusCmd{Clean: false}); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		e.log.ErrorContext(ctx, "failed finishing clean status", slog.String("error", err.Error()))
	}
}

func (e ExecutePinsOnExecuteCleanZone) notify(ctx context.Context, span trace.Span, message string) {
	if _, err := e.ch.Handle(ctx, app.PublishMessageCmd{Message: message}); err != nil {
		span.RecordError(err)
		e.log.ErrorContext(ctx, "failed publishing clean zone notification", slog.String("error", err.Error()))
	}
}

func (e ExecutePinsOnExecuteCleanZone) executePin(ctx context.Context, seconds uint, pins []string) error {
	_, err := e.ch.Handle(ctx, app.ExecutePinsCmd{
		Seconds:    seconds,
		Pins:       pins,
		CleanCycle: true,
	})
	return err
}

func (e ExecutePinsOnExecuteCleanZone) fertilizerValve(ctx context.Context, seconds uint, pin string) error {
	e.log.DebugContext(ctx, "execute fertilizer valve", slog.Uint64("seconds", uint64(seconds)))
	defer e.log.DebugContext(ctx, "fertilizer valve stopped")
	return e.executePin(ctx, seconds, []string{pin})
}

func (e ExecutePinsOnExecuteCleanZone) fertilizerPump(ctx context.Context, seconds uint, pin string) error {
	e.log.DebugContext(ctx, "execute fertilizer pump", slog.Uint64("seconds", uint64(seconds)))
	defer e.log.DebugContext(ctx, "fertilizer pump stopped")
	return e.executePin(ctx, seconds, []string{pin})
}

func (e ExecutePinsOnExecuteCleanZone) zone(ctx context.Context, seconds uint, pins []string) error {
	e.log.DebugContext(ctx, "execute zone", slog.Uint64("seconds", uint64(seconds)))
	defer e.log.DebugContext(ctx, "zone stopped")
	return e.executePin(ctx, seconds, pins)
}

func (e ExecutePinsOnExecuteCleanZone) cleanExecution(ctx context.Context, executed *zone.CleanZoneExecuted) error {
	g, ctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		seconds := executed.ZoneSeconds + executed.StabilizationZoneSeconds
		return e.zone(ctx, seconds, executed.ZoneRelayPins)
	})

	if err := e.wait(ctx, time.Duration(executed.StabilizationZoneSeconds)*time.Second); err != nil {
		return err
	}
	g.Go(func() error {
		return e.fertilizerValve(ctx, executed.FertilizerValveSeconds, executed.FertilizerValveRelayPin)
	})

	g.Go(func() error {
		return e.fertilizerPump(ctx, executed.FertilizerPumpSeconds, executed.FertilizerPumpRelayPin)
	})

	return g.Wait()
}

func (e ExecutePinsOnExecuteCleanZone) clarifyExecution(ctx context.Context, executed *zone.CleanZoneExecuted) error {
	e.log.DebugContext(ctx, "execution clarify process", slog.Duration("duration", executed.ClarifyDuration))

	g, ctx := errgroup.WithContext(ctx)
	seconds := uint(executed.ClarifyDuration.Seconds())
	g.Go(func() error {
		return e.zone(ctx, seconds+executed.StabilizationZoneSeconds, executed.ZoneRelayPins)
	})
	if err := e.wait(ctx, time.Duration(executed.StabilizationZoneSeconds)*time.Second); err != nil {
		return err
	}
	g.Go(func() error {
		return e.fertilizerPump(ctx, seconds, executed.FertilizerPumpRelayPin)
	})
	g.Go(func() error {
		return e.fertilizerValve(ctx, seconds, executed.FertilizerValveRelayPin)
	})

	return g.Wait()
}

func NewExecutePinsOnExecuteCleanZone(ch cqs.CommandHandler, wait Wait, tracer trace.Tracer, log *slog.Logger) *ExecutePinsOnExecuteCleanZone {
	return &ExecutePinsOnExecuteCleanZone{ch: ch, wait: wait, trace: tracer, log: log}
}
