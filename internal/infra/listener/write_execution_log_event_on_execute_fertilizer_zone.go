package listener

import (
	"context"
	"time"

	"github.com/bruli-lab/go-core/event"
	"github.com/bruli/raspberryWaterSystem/internal/domain/zone"
	"github.com/bruli/raspberryWaterSystem/internal/infra/outbox"
	"go.opentelemetry.io/otel/trace"
)

type WriteExecutionLogEventOnExecuteFertilizerZone struct {
	events OutboxEvents
	now    func() time.Time
	tracer trace.Tracer
}

func (w WriteExecutionLogEventOnExecuteFertilizerZone) Listen(ctx context.Context, ev event.Event) error {
	ctx, span := w.tracer.Start(ctx, "WriteExecutionLogEventOnExecuteFertilizerZone.Listen")
	defer span.End()
	executed, _ := ev.(zone.FertilizerZoneExecuted)
	return saveExecutionLogEvent(ctx, w.events, span, outbox.ExecutionLog{
		Seconds:    int(executed.ZoneSeconds),
		ZoneName:   executed.ZoneName,
		ExecutedAt: executed.EventAt(),
	}, w.now())
}

func NewWriteExecutionLogEventOnExecuteFertilizerZone(events OutboxEvents, now func() time.Time, tracer trace.Tracer) *WriteExecutionLogEventOnExecuteFertilizerZone {
	return &WriteExecutionLogEventOnExecuteFertilizerZone{events: events, now: now, tracer: tracer}
}
