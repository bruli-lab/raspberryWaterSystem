package listener

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/bruli/raspberryWaterSystem/internal/app"
	"github.com/bruli/raspberryWaterSystem/internal/cqs"
	"github.com/bruli/raspberryWaterSystem/internal/domain/zone"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/sync/errgroup"
)

type ExecutePinsOnExecuteFertilizerZone struct {
	ch    cqs.CommandHandler
	trace trace.Tracer
	log   *slog.Logger
}

func (e ExecutePinsOnExecuteFertilizerZone) Listen(ctx context.Context, ev cqs.Event) error {
	ctx, span := e.trace.Start(ctx, "executePinsOnExecuteFertilizerZone.Listen")
	defer span.End()
	event, ok := ev.(zone.FertilizerZoneExecuted)
	if !ok {
		err := errors.New("wrong event type")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	if err := e.execution(ctx, &event); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	return nil
}

func (e ExecutePinsOnExecuteFertilizerZone) airZone(ctx context.Context, event *zone.FertilizerZoneExecuted) error {
	slog.InfoContext(ctx, "execute air zone", slog.Uint64("seconds", uint64(event.AirZoneSeconds)))
	defer slog.InfoContext(ctx, "air zone stopped")
	return e.executePin(ctx, event.AirZoneSeconds, []string{event.AirZoneRelayPin})
}

func (e ExecutePinsOnExecuteFertilizerZone) executePin(ctx context.Context, seconds uint, pins []string) error {
	_, err := e.ch.Handle(ctx, app.ExecutePinsCmd{
		Seconds: seconds,
		Pins:    pins,
	})
	return err
}

func (e ExecutePinsOnExecuteFertilizerZone) fertilizerValvule(ctx context.Context, event *zone.FertilizerZoneExecuted) error {
	e.log.Info("execute fertilizer valvule", slog.Uint64("seconds", uint64(event.FertilizerValvuleSeconds)))
	defer e.log.Info("fertilizer valvule stopped")
	return e.executePin(ctx, event.FertilizerValvuleSeconds, []string{event.FertilizerValvuleRelayPin})
}

func (e ExecutePinsOnExecuteFertilizerZone) fertilizerPump(ctx context.Context, event *zone.FertilizerZoneExecuted) error {
	seconds := event.CleanValvuleSeconds + event.FertilizerPumpSeconds
	e.log.Info("execute fertilizer pump", slog.Uint64("seconds", uint64(seconds)))
	defer e.log.Info("fertilizer pump stopped")
	return e.executePin(ctx, seconds, []string{event.FertilizerPumpRelayPin})
}

func (e ExecutePinsOnExecuteFertilizerZone) zone(ctx context.Context, event *zone.FertilizerZoneExecuted) error {
	e.log.Info("execute zone", slog.Uint64("seconds", uint64(event.ZoneSeconds)))
	defer e.log.Info("zone stopped")
	seconds := event.ZoneSeconds + event.StabilizationZoneSeconds + event.CleanValvuleSeconds
	return e.executePin(ctx, seconds, event.ZoneRelayPins)
}

func (e ExecutePinsOnExecuteFertilizerZone) execution(ctx context.Context, event *zone.FertilizerZoneExecuted) error {
	if err := e.airZone(ctx, event); err != nil {
		return err
	}

	g, ctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		return e.zone(ctx, event)
	})

	timer := time.NewTimer(time.Duration(event.StabilizationZoneSeconds) * time.Second)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
	}
	g.Go(func() error {
		if err := e.fertilizerValvule(ctx, event); err != nil {
			return err
		}
		return e.clean(ctx, event)
	})

	g.Go(func() error {
		return e.fertilizerPump(ctx, event)
	})

	return g.Wait()
}

func (e ExecutePinsOnExecuteFertilizerZone) clean(ctx context.Context, event *zone.FertilizerZoneExecuted) error {
	e.log.Info("execute clean valvule", slog.Uint64("seconds", uint64(event.CleanValvuleSeconds)))
	defer e.log.Info("clean valvule stopped")
	return e.executePin(ctx, event.CleanValvuleSeconds, []string{event.CleanValvuleRelayPin})
}

func NewExecutePinsOnExecuteFertilizerZone(ch cqs.CommandHandler, tracer trace.Tracer, log *slog.Logger) *ExecutePinsOnExecuteFertilizerZone {
	return &ExecutePinsOnExecuteFertilizerZone{ch: ch, trace: tracer, log: log}
}
