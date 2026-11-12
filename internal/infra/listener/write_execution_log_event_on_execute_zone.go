package listener

import (
	"context"

	"github.com/bruli-lab/go-core/event"
	"github.com/bruli/raspberryWaterSystem/internal/domain/zone"
	"github.com/bruli/raspberryWaterSystem/internal/infra/disk"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type WriteExecutionLogEventOnExecuteZone struct {
	eventsRepo *disk.EventsRepository
	tracer     trace.Tracer
}

func (w WriteExecutionLogEventOnExecuteZone) Listen(ctx context.Context, ev event.Event) error {
	ctx, span := w.tracer.Start(ctx, "WriteExecutionLogEventOnExecuteZone.Listen")
	defer span.End()
	executed, _ := ev.(zone.Executed)
	evnt, err := disk.NewFromExecutionLog(ctx, &disk.Log{
		Seconds:    int(executed.Seconds),
		ZoneName:   executed.ZoneName,
		ExecutedAt: executed.EventAt(),
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

func NewWriteExecutionLogEventOnExecuteZone(eventsRepo *disk.EventsRepository, tracer trace.Tracer) *WriteExecutionLogEventOnExecuteZone {
	return &WriteExecutionLogEventOnExecuteZone{eventsRepo: eventsRepo, tracer: tracer}
}
