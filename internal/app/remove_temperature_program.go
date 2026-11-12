package app

import (
	"context"

	"github.com/bruli-lab/go-core/cqs"
	"github.com/bruli-lab/go-core/event"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const RemoveTemperatureProgramCommandName = "removeTemperatureProgram"

type RemoveTemperatureProgramCommand struct {
	Temperature float32
}

func (r RemoveTemperatureProgramCommand) Name() string {
	return RemoveTemperatureProgramCommandName
}

type RemoveTemperatureProgram struct {
	repo   TemperatureProgramRepository
	tracer trace.Tracer
}

func (r RemoveTemperatureProgram) Handle(ctx context.Context, cmd cqs.Command) ([]event.Event, error) {
	ctx, span := r.tracer.Start(ctx, "RemoveTemperatureProgramCmd")
	defer span.End()
	co, ok := cmd.(RemoveTemperatureProgramCommand)
	if !ok {
		err := cqs.NewInvalidCommandError(RemoveTemperatureProgramCommandName, cmd.Name())
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	if _, err := r.repo.FindByTemperature(ctx, co.Temperature); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	if err := r.repo.Remove(ctx, co.Temperature); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	span.SetStatus(codes.Ok, "program removed")
	return nil, nil
}

func NewRemoveTemperatureProgram(repo TemperatureProgramRepository, tracer trace.Tracer) *RemoveTemperatureProgram {
	return &RemoveTemperatureProgram{repo: repo, tracer: tracer}
}
