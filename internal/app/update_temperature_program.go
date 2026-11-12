package app

import (
	"context"

	"github.com/bruli-lab/go-core/cqs"
	"github.com/bruli-lab/go-core/event"
	"github.com/bruli/raspberryWaterSystem/internal/domain/program"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const UpdateTemperatureProgramCommandName = "updateTemperatureProgram"

type UpdateTemperatureProgramCommand struct {
	Temperature float32
	Programs    []program.Program
}

func (u UpdateTemperatureProgramCommand) Name() string {
	return UpdateTemperatureProgramCommandName
}

type UpdateTemperatureProgram struct {
	repo   TemperatureProgramRepository
	tracer trace.Tracer
}

func (u UpdateTemperatureProgram) Handle(ctx context.Context, cmd cqs.Command) ([]event.Event, error) {
	ctx, span := u.tracer.Start(ctx, "UpdateTemperatureProgramCmd")
	defer span.End()
	co, ok := cmd.(UpdateTemperatureProgramCommand)
	if !ok {
		err := cqs.NewInvalidCommandError(UpdateTemperatureProgramCommandName, cmd.Name())
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	pr, err := u.repo.FindByTemperature(ctx, co.Temperature)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	pr.Update(co.Programs)

	if err = u.repo.Save(ctx, pr); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	span.SetStatus(codes.Ok, "temperature program updated")
	return nil, nil
}

func NewUpdateTemperatureProgram(repo TemperatureProgramRepository, tracer trace.Tracer) *UpdateTemperatureProgram {
	return &UpdateTemperatureProgram{repo: repo, tracer: tracer}
}
