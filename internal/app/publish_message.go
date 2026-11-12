package app

import (
	"context"

	"github.com/bruli-lab/go-core/cqs"
	"github.com/bruli-lab/go-core/event"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const PublishMessageCmdName = "publishMessage"

type PublishMessageCmd struct {
	Message string
}

func (p PublishMessageCmd) Name() string {
	return PublishMessageCmdName
}

type PublishMessage struct {
	mp     MessagePublisher
	tracer trace.Tracer
}

func (p PublishMessage) Handle(ctx context.Context, cmd cqs.Command) ([]event.Event, error) {
	ctx, span := p.tracer.Start(ctx, "PublishMessageCmd")
	defer span.End()
	co, ok := cmd.(PublishMessageCmd)
	if !ok {
		err := cqs.NewInvalidCommandError(PublishMessageCmdName, cmd.Name())
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	if err := p.mp.Publish(ctx, co.Message); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	span.SetStatus(codes.Ok, "message published")
	return nil, nil
}

func NewPublishMessage(elp MessagePublisher, tracer trace.Tracer) PublishMessage {
	return PublishMessage{mp: elp, tracer: tracer}
}
