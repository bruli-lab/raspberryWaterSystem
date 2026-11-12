package listener

import (
	"context"
	"fmt"

	"github.com/bruli-lab/go-core/cqs"
	"github.com/bruli-lab/go-core/event"
	"github.com/bruli/raspberryWaterSystem/internal/app"
	"github.com/bruli/raspberryWaterSystem/internal/domain/program"
	"github.com/bruli/raspberryWaterSystem/internal/domain/zone"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type ExecutePinsOnExecuteZone struct {
	ch     cqs.CommandHandler
	tracer trace.Tracer
}

func (e ExecutePinsOnExecuteZone) Listen(ctx context.Context, ev event.Event) error {
	ctx, span := e.tracer.Start(ctx, "ExecutePinsOnExecuteZone.Listen")
	defer span.End()
	executed, _ := ev.(zone.Executed)
	if _, err := e.ch.Handle(ctx, app.ExecutePinsCmd{
		Seconds: executed.Seconds + executed.StabilizationSeconds,
		Pins:    executed.RelayPins,
	}); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	sec := program.HydrateSeconds(int(executed.Seconds))
	if _, err := e.ch.Handle(ctx, app.SaveExecutionLogCmd{
		ZoneName:   executed.ZoneName,
		Seconds:    sec,
		ExecutedAt: executed.EventAt(),
	}); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	message := fmt.Sprintf("%s zone executed during %vs", executed.ZoneName, sec.Int())
	if _, err := e.ch.Handle(ctx, app.PublishMessageCmd{Message: message}); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	span.SetStatus(codes.Ok, "pins executed")
	return nil
}

func NewExecutePinsOnExecuteZone(ch cqs.CommandHandler, tracer trace.Tracer) ExecutePinsOnExecuteZone {
	return ExecutePinsOnExecuteZone{ch: ch, tracer: tracer}
}
