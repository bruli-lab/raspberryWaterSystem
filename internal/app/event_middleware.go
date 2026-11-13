package app

import (
	"context"
	"fmt"

	"github.com/bruli-lab/go-core/cqs"
	"github.com/bruli-lab/go-core/event"
	"go.opentelemetry.io/otel/trace"
)

func NewEventMiddleware(evCh chan<- TracedEvent, tracer trace.Tracer) cqs.CommandHandlerMiddleware {
	return func(h cqs.CommandHandler) cqs.CommandHandler {
		return cqs.CommandHandlerFunc(func(ctx context.Context, cmd cqs.Command) ([]event.Event, error) {
			ctx, span := tracer.Start(ctx, fmt.Sprintf("command handler for %s", cmd.Name()))
			defer span.End()
			events, err := h.Handle(ctx, cmd)
			for _, ev := range events {
				evCh <- TracedEvent{
					SpanContext: trace.SpanContextFromContext(ctx),
					Event:       ev,
				}
			}
			return nil, err
		})
	}
}
