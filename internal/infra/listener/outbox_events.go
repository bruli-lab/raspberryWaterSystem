package listener

import (
	"context"
	"time"

	"github.com/bruli/raspberryWaterSystem/internal/infra/outbox"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

//go:generate go tool moq -out outbox_events_mock_test.go -pkg listener_test . OutboxEvents

type OutboxEvents interface {
	Save(ctx context.Context, ev *outbox.Event) error
}

func saveExecutionLogEvent(ctx context.Context, events OutboxEvents, span trace.Span, executionLog outbox.ExecutionLog, now time.Time) error {
	ev, err := outbox.NewFromExecutionLog(ctx, executionLog, now)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	if err = events.Save(ctx, ev); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	span.SetStatus(codes.Ok, "event saved")
	return nil
}
