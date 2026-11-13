package listener

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
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

type ExecutePinsOnExecuteFertilizerZone struct {
	ch    cqs.CommandHandler
	wait  Wait
	trace trace.Tracer
	log   *slog.Logger
}

func (e ExecutePinsOnExecuteFertilizerZone) Listen(ctx context.Context, ev event.Event) error {
	ctx, span := e.trace.Start(ctx, "executePinsOnExecuteFertilizerZone.Listen")
	defer span.End()
	executed, ok := ev.(zone.FertilizerZoneExecuted)
	if !ok {
		err := errors.New("wrong event type")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	if err := e.execution(ctx, &executed); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	sec := program.HydrateSeconds(int(executed.ZoneSeconds))
	return recordIrrigation(ctx, e.ch, span, irrigationRecord{
		zoneName:   executed.ZoneName,
		seconds:    sec,
		executedAt: executed.EventAt(),
		message:    fmt.Sprintf("%s fertilizer zone executed during %vs", executed.ZoneName, sec.Int()),
	})
}

func (e ExecutePinsOnExecuteFertilizerZone) airZone(ctx context.Context, executed *zone.FertilizerZoneExecuted) error {
	e.log.DebugContext(ctx, "execute air zone", slog.Uint64("seconds", uint64(executed.AirZoneSeconds)))
	defer e.log.DebugContext(ctx, "air zone stopped")
	return e.executePin(ctx, executed.AirZoneSeconds, []string{executed.AirZoneRelayPin})
}

func (e ExecutePinsOnExecuteFertilizerZone) executePin(ctx context.Context, seconds uint, pins []string) error {
	_, err := e.ch.Handle(ctx, app.ExecutePinsCmd{
		Seconds: seconds,
		Pins:    pins,
	})
	return err
}

func (e ExecutePinsOnExecuteFertilizerZone) fertilizerValve(ctx context.Context, executed *zone.FertilizerZoneExecuted) error {
	e.log.DebugContext(ctx, "execute fertilizer valve", slog.Uint64("seconds", uint64(executed.FertilizerValveSeconds)))
	defer e.log.DebugContext(ctx, "fertilizer valve stopped")
	return e.executePin(ctx, executed.FertilizerValveSeconds, []string{executed.FertilizerValveRelayPin})
}

func (e ExecutePinsOnExecuteFertilizerZone) fertilizerPump(ctx context.Context, executed *zone.FertilizerZoneExecuted) error {
	seconds := executed.FertilizerPumpSeconds
	e.log.DebugContext(ctx, "execute fertilizer pump", slog.Uint64("seconds", uint64(seconds)))
	defer e.log.DebugContext(ctx, "fertilizer pump stopped")
	return e.executePin(ctx, seconds, []string{executed.FertilizerPumpRelayPin})
}

func (e ExecutePinsOnExecuteFertilizerZone) zone(ctx context.Context, executed *zone.FertilizerZoneExecuted) error {
	e.log.DebugContext(ctx, "execute zone", slog.Uint64("seconds", uint64(executed.ZoneSeconds)))
	defer e.log.DebugContext(ctx, "zone stopped")
	seconds := executed.ZoneSeconds + executed.StabilizationZoneSeconds + executed.CleanValveSeconds
	return e.executePin(ctx, seconds, executed.ZoneRelayPins)
}

func (e ExecutePinsOnExecuteFertilizerZone) execution(ctx context.Context, executed *zone.FertilizerZoneExecuted) error {
	if err := e.airZone(ctx, executed); err != nil {
		return err
	}

	g, ctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		return e.zone(ctx, executed)
	})

	if err := e.wait(ctx, time.Duration(executed.StabilizationZoneSeconds)*time.Second); err != nil {
		return err
	}
	g.Go(func() error {
		if err := e.fertilizerValve(ctx, executed); err != nil {
			return err
		}
		return e.clean(ctx, executed)
	})

	g.Go(func() error {
		return e.fertilizerPump(ctx, executed)
	})

	return g.Wait()
}

func (e ExecutePinsOnExecuteFertilizerZone) clean(ctx context.Context, executed *zone.FertilizerZoneExecuted) error {
	e.log.DebugContext(ctx, "execute clean valve", slog.Uint64("seconds", uint64(executed.CleanValveSeconds)))
	defer e.log.DebugContext(ctx, "clean valve stopped")
	return e.executePin(ctx, executed.CleanValveSeconds, []string{executed.CleanValveRelayPin})
}

func NewExecutePinsOnExecuteFertilizerZone(ch cqs.CommandHandler, wait Wait, tracer trace.Tracer, log *slog.Logger) *ExecutePinsOnExecuteFertilizerZone {
	return &ExecutePinsOnExecuteFertilizerZone{ch: ch, wait: wait, trace: tracer, log: log}
}
