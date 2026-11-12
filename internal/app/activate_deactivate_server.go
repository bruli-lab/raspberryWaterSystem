package app

import (
	"context"

	"github.com/bruli-lab/go-core/cqs"
	"github.com/bruli-lab/go-core/event"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const ActivateDeactivateServerCmdName = "activateDeactivateServer"

type ActivateDeactivateServerCmd struct {
	Active bool
}

func (a ActivateDeactivateServerCmd) Name() string {
	return ActivateDeactivateServerCmdName
}

type ActivateDeactivateServer struct {
	stRepo StatusRepository
	tracer trace.Tracer
}

func (a ActivateDeactivateServer) Handle(ctx context.Context, cmd cqs.Command) ([]event.Event, error) {
	ctx, span := a.tracer.Start(ctx, "ActivateDeactivateServerCmd")
	defer span.End()
	co, ok := cmd.(ActivateDeactivateServerCmd)
	if !ok {
		err := cqs.NewInvalidCommandError(ActivateDeactivateServerCmdName, cmd.Name())
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	st, err := a.stRepo.Find(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	switch {
	case co.Active:
		st.Activate()
	default:
		st.Deactivate()
	}
	span.SetStatus(codes.Ok, "server activated/deactivated")
	return nil, a.stRepo.Update(ctx, st)
}

func NewActivateDeactivateServer(stRepo StatusRepository, tracer trace.Tracer) *ActivateDeactivateServer {
	return &ActivateDeactivateServer{stRepo: stRepo, tracer: tracer}
}
