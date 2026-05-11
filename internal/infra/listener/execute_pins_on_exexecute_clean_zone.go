package listener

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
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
	if err := e.cleanExecution(ctx, &event); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	sec, _ := program.ParseSeconds(int(event.ZoneSeconds))
	message := fmt.Sprintf("%s clean zone executed during %vs", event.ZoneName, sec.Int())
	if _, err := e.ch.Handle(ctx, app.PublishMessageCmd{Message: message}); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	e.log.DebugContext(ctx, "wait for clarify process", slog.Duration("duration", event.WaitDuration))
	timer := time.NewTimer(event.WaitDuration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
	}
	if err := e.clarifyExecution(ctx, &event); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	message = fmt.Sprintf("clarify zone %s finished!!", event.ZoneName)
	if _, err := e.ch.Handle(ctx, app.PublishMessageCmd{Message: message}); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	span.SetStatus(codes.Ok, "clean executed")

	return nil
}

func (e ExecutePinsOnExecuteCleanZone) executePin(ctx context.Context, seconds uint, pins []string) error {
	_, err := e.ch.Handle(ctx, app.ExecutePinsCmd{
		Seconds: seconds,
		Pins:    pins,
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
