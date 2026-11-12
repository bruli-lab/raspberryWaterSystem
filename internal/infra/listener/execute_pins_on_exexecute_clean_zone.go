package listener

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"runtime/debug"
	"time"

	"github.com/bruli/raspberryWaterSystem/internal/app"
	"github.com/bruli/raspberryWaterSystem/internal/cqs"
	"github.com/bruli/raspberryWaterSystem/internal/domain/program"
	"github.com/bruli/raspberryWaterSystem/internal/domain/zone"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/sync/errgroup"
)

type ExecutePinsOnExecuteCleanZone struct {
	ch    cqs.CommandHandler
	trace trace.Tracer
	log   *slog.Logger
}

func (e ExecutePinsOnExecuteCleanZone) Listen(ctx context.Context, ev cqs.Event) error {
	ctx, span := e.trace.Start(ctx, "executePinsOnExecuteCleanZone.Listen")
	defer span.End()
	event, ok := ev.(zone.CleanZoneExecuted)
	if !ok {
		err := errors.New("wrong event type")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	go e.runCleanCycle(ctx, &event)
	span.SetStatus(codes.Ok, "clean cycle started")
	return nil
}

func (e ExecutePinsOnExecuteCleanZone) runCleanCycle(ctx context.Context, event *zone.CleanZoneExecuted) {
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
	if err := e.cleanCycle(ctx, span, event); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		e.log.ErrorContext(ctx, "failed running clean cycle", slog.String("zone", event.ZoneName), slog.String("error", err.Error()))
		return
	}
	span.SetStatus(codes.Ok, "clean executed")
}

func (e ExecutePinsOnExecuteCleanZone) cleanCycle(ctx context.Context, span trace.Span, event *zone.CleanZoneExecuted) error {
	if err := e.cleanExecution(ctx, event); err != nil {
		return err
	}
	sec := program.HydrateSeconds(int(event.ZoneSeconds))
	e.notify(ctx, span, fmt.Sprintf("%s clean zone executed during %vs", event.ZoneName, sec.Int()))
	e.log.DebugContext(ctx, "wait for clarify process", slog.Duration("duration", event.WaitDuration))
	timer := time.NewTimer(event.WaitDuration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
	}
	if err := e.clarifyExecution(ctx, event); err != nil {
		return err
	}
	e.notify(ctx, span, fmt.Sprintf("clarify zone %s finished!!", event.ZoneName))
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

func (e ExecutePinsOnExecuteCleanZone) fertilizerValvule(ctx context.Context, seconds uint, pin string) error {
	e.log.DebugContext(ctx, "execute fertilizer valvule", slog.Uint64("seconds", uint64(seconds)))
	defer e.log.DebugContext(ctx, "fertilizer valvule stopped")
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

func (e ExecutePinsOnExecuteCleanZone) cleanExecution(ctx context.Context, event *zone.CleanZoneExecuted) error {
	g, ctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		seconds := event.ZoneSeconds + event.StabilizationZoneSeconds
		return e.zone(ctx, seconds, event.ZoneRelayPins)
	})

	timer := time.NewTimer(time.Duration(event.StabilizationZoneSeconds) * time.Second)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
	}
	g.Go(func() error {
		return e.fertilizerValvule(ctx, event.FertilizerValvuleSeconds, event.FertilizerValvuleRelayPin)
	})

	g.Go(func() error {
		return e.fertilizerPump(ctx, event.FertilizerPumpSeconds, event.FertilizerPumpRelayPin)
	})

	return g.Wait()
}

func (e ExecutePinsOnExecuteCleanZone) clarifyExecution(ctx context.Context, event *zone.CleanZoneExecuted) error {
	e.log.DebugContext(ctx, "execution clarify process", slog.Duration("duration", event.ClarifyDuration))

	g, ctx := errgroup.WithContext(ctx)
	timer := time.NewTimer(time.Duration(event.StabilizationZoneSeconds) * time.Second)
	defer timer.Stop()

	seconds := uint(event.ClarifyDuration.Seconds())
	g.Go(func() error {
		return e.zone(ctx, seconds+event.StabilizationZoneSeconds, event.ZoneRelayPins)
	})
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
	}
	g.Go(func() error {
		return e.fertilizerPump(ctx, seconds, event.FertilizerPumpRelayPin)
	})
	g.Go(func() error {
		return e.fertilizerValvule(ctx, seconds, event.FertilizerValvuleRelayPin)
	})

	return g.Wait()
}

func NewExecutePinsOnExecuteCleanZone(ch cqs.CommandHandler, tracer trace.Tracer, log *slog.Logger) *ExecutePinsOnExecuteCleanZone {
	return &ExecutePinsOnExecuteCleanZone{ch: ch, trace: tracer, log: log}
}
