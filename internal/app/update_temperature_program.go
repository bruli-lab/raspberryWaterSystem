package app

import (
	"context"

	"github.com/bruli-lab/go-core/cqs"
	"github.com/bruli-lab/go-core/event"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const UpdateTemperatureProgramCommandName = "updateTemperatureProgram"

type UpdateTemperatureProgramCommand struct {
	Temperature float32
	Programs    []ProgramInput
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
	if err := u.update(ctx, co); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	span.SetStatus(codes.Ok, "temperature program updated")
	return nil, nil
}

func (u UpdateTemperatureProgram) update(ctx context.Context, co UpdateTemperatureProgramCommand) error {
	programs, err := buildProgramsWithoutFertilizer(co.Programs)
	if err != nil {
		return UpdateTemperatureProgramError{msg: err.Error()}
	}
	temp, err := u.repo.FindByTemperature(ctx, co.Temperature)
	if err != nil {
		return err
	}
	if err = temp.Update(programs); err != nil {
		return UpdateTemperatureProgramError{msg: err.Error()}
	}
	return u.repo.Save(ctx, temp)
}

type UpdateTemperatureProgramError struct {
	msg string
}

func (u UpdateTemperatureProgramError) Error() string {
	return "failed to update temperature program: " + u.msg
}

func NewUpdateTemperatureProgram(repo TemperatureProgramRepository, tracer trace.Tracer) *UpdateTemperatureProgram {
	return &UpdateTemperatureProgram{repo: repo, tracer: tracer}
}
