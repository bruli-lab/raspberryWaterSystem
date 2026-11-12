package app

import (
	"context"
	"fmt"

	"github.com/bruli/raspberryWaterSystem/internal/cqs"
	"github.com/bruli/raspberryWaterSystem/internal/infra/tracing"
	"go.opentelemetry.io/otel/trace"
)

func NewEventMiddleware(evCh chan<- tracing.Event, tracer trace.Tracer) cqs.CommandHandlerMiddleware {
	return func(h cqs.CommandHandler) cqs.CommandHandler {
		return cqs.CommandHandlerFunc(func(ctx context.Context, cmd cqs.Command) ([]cqs.Event, error) {
			ctx, span := tracer.Start(ctx, fmt.Sprintf("command handler for %s", cmd.Name()))
			defer span.End()
			events, err := h.Handle(ctx, cmd)
			for _, ev := range events {
				evCh <- tracing.Event{
					SpanContext: trace.SpanContextFromContext(ctx),
					Event:       ev,
				}
			}
			return nil, err
		})
	}
}
