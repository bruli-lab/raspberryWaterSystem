package worker

import (
	"context"
	"errors"
	"fmt"

	"github.com/bruli/raspberryWaterSystem/internal/infra/outbox"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

type PendingEvents interface {
	FindAll(ctx context.Context) ([]outbox.Event, error)
	Remove(ctx context.Context, ev *outbox.Event) error
}

type EventPublisher interface {
	IsConnected() bool
	PublishEvent(ctx context.Context, id, subject string, payload []byte) error
}

func PublishOutboxEvents(ctx context.Context, pending PendingEvents, publisher EventPublisher, tracer trace.Tracer) (int, error) {
	ctx, span := tracer.Start(ctx, "PublishOutboxEvents worker")
	defer span.End()
	if !publisher.IsConnected() {
		err := PublisherNotConnectedError{}
		span.SetStatus(codes.Error, err.Error())
		return 0, err
	}
	events, err := pending.FindAll(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return 0, err
	}
	published := 0
	var failures []error
	for i := range events {
		if err = publishOutboxEvent(ctx, pending, publisher, &events[i], tracer); err != nil {
			failures = append(failures, PublishOutboxEventError{id: events[i].ID, err: err})
			if !publisher.IsConnected() {
				failures = append(failures, PublisherNotConnectedError{})
				break
			}
			continue
		}
		published++
	}
	span.SetAttributes(attribute.Int("events.published", published))
	if len(failures) > 0 {
		err = errors.Join(failures...)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return published, err
	}
	span.SetStatus(codes.Ok, "outbox events published")
	return published, nil
}

func publishOutboxEvent(ctx context.Context, pending PendingEvents, publisher EventPublisher, ev *outbox.Event, tracer trace.Tracer) error {
	parentCtx := propagation.TraceContext{}.Extract(ctx, propagation.MapCarrier{"traceparent": ev.Trace["traceparent"]})
	ctx, span := tracer.Start(parentCtx, ev.EventType)
	defer span.End()
	if err := publisher.PublishEvent(ctx, ev.ID, ev.EventType, ev.Payload); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	if err := pending.Remove(ctx, ev); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	span.SetStatus(codes.Ok, "event published and removed")
	return nil
}

type PublisherNotConnectedError struct{}

func (PublisherNotConnectedError) Error() string {
	return "event publisher is not connected, events kept in the outbox"
}

type PublishOutboxEventError struct {
	id  string
	err error
}

func (p PublishOutboxEventError) Error() string {
	return fmt.Sprintf("failed publishing outbox event %s: %s", p.id, p.err)
}

func (p PublishOutboxEventError) Unwrap() error {
	return p.err
}
