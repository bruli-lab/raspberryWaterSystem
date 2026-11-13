package app

import (
	"context"
	"errors"
	"fmt"

	"github.com/bruli-lab/go-core/cqs"
	"github.com/bruli-lab/go-core/event"
	"github.com/bruli/raspberryWaterSystem/internal/domain/program"
	errs "github.com/bruli/raspberryWaterSystem/internal/errors"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const CreateTemperatureProgramCommandName = "createTemperatureProgram"

type CreateTemperatureProgramCommand struct {
	Temperature *program.Temperature
}

func (c CreateTemperatureProgramCommand) Name() string {
	return CreateTemperatureProgramCommandName
}

type CreateTemperatureProgram struct {
	repo   TemperatureProgramRepository
	tracer trace.Tracer
}

func (c CreateTemperatureProgram) Handle(ctx context.Context, cmd cqs.Command) ([]event.Event, error) {
	ctx, span := c.tracer.Start(ctx, "CreateTemperatureProgramCmd")
	defer span.End()
	co, ok := cmd.(CreateTemperatureProgramCommand)
	if !ok {
		err := cqs.NewInvalidCommandError(CreateTemperatureProgramCommandName, cmd.Name())
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	_, err := c.repo.FindByTemperature(ctx, co.Temperature.Temperature())
	switch {
	case err == nil:
		err = CreateTemperatureProgramError{msg: fmt.Sprintf("a temperature program with temperature %v already exists", co.Temperature)}
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	case errors.As(err, &errs.NotFoundError{}):
		if err = c.repo.Save(ctx, co.Temperature); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return nil, err
		}
		span.SetStatus(codes.Ok, "temperature program created")
		return nil, nil
	default:
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
}

func NewCreateTemperatureProgram(repo TemperatureProgramRepository, tracer trace.Tracer) *CreateTemperatureProgram {
	return &CreateTemperatureProgram{repo: repo, tracer: tracer}
}

type CreateTemperatureProgramError struct {
	msg string
}

func (c CreateTemperatureProgramError) Error() string {
	return "failed to create a temperature program: " + c.msg
}
