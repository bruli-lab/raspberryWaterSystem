package listener

import (
	"context"
	"time"

	"github.com/bruli-lab/go-core/event"
	"github.com/bruli/raspberryWaterSystem/internal/domain/zone"
	"github.com/bruli/raspberryWaterSystem/internal/infra/outbox"
	"go.opentelemetry.io/otel/trace"
)

type WriteExecutionLogEventOnExecuteZone struct {
	events OutboxEvents
	now    func() time.Time
	tracer trace.Tracer
}

func (w WriteExecutionLogEventOnExecuteZone) Listen(ctx context.Context, ev event.Event) error {
	ctx, span := w.tracer.Start(ctx, "WriteExecutionLogEventOnExecuteZone.Listen")
	defer span.End()
	executed, _ := ev.(zone.Executed)
	return saveExecutionLogEvent(ctx, w.events, span, outbox.ExecutionLog{
		Seconds:    int(executed.Seconds),
		ZoneName:   executed.ZoneName,
		ExecutedAt: executed.EventAt(),
	}, w.now())
}

func NewWriteExecutionLogEventOnExecuteZone(events OutboxEvents, now func() time.Time, tracer trace.Tracer) *WriteExecutionLogEventOnExecuteZone {
	return &WriteExecutionLogEventOnExecuteZone{events: events, now: now, tracer: tracer}
}
