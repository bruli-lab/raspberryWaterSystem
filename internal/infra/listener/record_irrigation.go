package listener

import (
	"context"
	"errors"
	"time"

	"github.com/bruli-lab/go-core/cqs"
	"github.com/bruli/raspberryWaterSystem/internal/app"
	"github.com/bruli/raspberryWaterSystem/internal/domain/program"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type irrigationRecord struct {
	zoneName   string
	seconds    program.Seconds
	executedAt time.Time
	message    string
}

func recordIrrigation(ctx context.Context, ch cqs.CommandHandler, span trace.Span, record irrigationRecord) error {
	_, logErr := ch.Handle(ctx, app.SaveExecutionLogCmd{
		ZoneName:   record.zoneName,
		Seconds:    record.seconds,
		ExecutedAt: record.executedAt,
	})
	_, notifyErr := ch.Handle(ctx, app.PublishMessageCmd{Message: record.message})
	if err := errors.Join(logErr, notifyErr); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	span.SetStatus(codes.Ok, "pins executed")
	return nil
}
