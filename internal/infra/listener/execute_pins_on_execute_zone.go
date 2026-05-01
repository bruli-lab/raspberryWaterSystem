package listener

import (
	"context"
	"fmt"

	"github.com/bruli/raspberryWaterSystem/internal/app"
	"github.com/bruli/raspberryWaterSystem/internal/cqs"
	"github.com/bruli/raspberryWaterSystem/internal/domain/program"
	"github.com/bruli/raspberryWaterSystem/internal/domain/zone"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type ExecutePinsOnExecuteZone struct {
	ch     cqs.CommandHandler
	tracer trace.Tracer
}

func (e ExecutePinsOnExecuteZone) Listen(ctx context.Context, ev cqs.Event) error {
	ctx, span := e.tracer.Start(ctx, "ExecutePinsOnExecuteZone.Listen")
	defer span.End()
	event, _ := ev.(zone.Executed)
	if _, err := e.ch.Handle(ctx, app.ExecutePinsCmd{
		Seconds: event.Seconds + event.StabilizationSeconds,
		Pins:    event.RelayPins,
	}); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	sec, _ := program.ParseSeconds(int(event.Seconds))
	if _, err := e.ch.Handle(ctx, app.SaveExecutionLogCmd{
		ZoneName:   event.ZoneName,
		Seconds:    sec,
		ExecutedAt: event.EventAt(),
	}); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	message := fmt.Sprintf("%s zone executed during %vs", event.ZoneName, sec.Int())
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
