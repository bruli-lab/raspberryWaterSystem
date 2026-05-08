package listener

import (
	"context"

	"github.com/bruli/raspberryWaterSystem/internal/cqs"
	"github.com/bruli/raspberryWaterSystem/internal/domain/zone"
	"github.com/bruli/raspberryWaterSystem/internal/infra/disk"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type WriteExecutionLogEventOnExecuteFertilizerZone struct {
	eventsRepo *disk.EventsRepository
	tracer     trace.Tracer
}

func (w WriteExecutionLogEventOnExecuteFertilizerZone) Listen(ctx context.Context, ev cqs.Event) error {
	ctx, span := w.tracer.Start(ctx, "WriteExecutionLogEventOnExecuteFertilizerZone.Listen")
	defer span.End()
	event, _ := ev.(zone.FertilizerZoneExecuted)
	evnt, err := disk.NewFromExecutionLog(ctx, &disk.Log{
		Seconds:    int(event.ZoneSeconds),
		ZoneName:   event.ZoneName,
		ExecutedAt: event.EventAt(),
	})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	if err = w.eventsRepo.Save(ctx, evnt); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	span.SetStatus(codes.Ok, "event saved")
	return nil
}

func NewWriteExecutionLogEventOnExecuteFertilizerZone(eventsRepo *disk.EventsRepository, tracer trace.Tracer) *WriteExecutionLogEventOnExecuteFertilizerZone {
	return &WriteExecutionLogEventOnExecuteFertilizerZone{eventsRepo: eventsRepo, tracer: tracer}
}
