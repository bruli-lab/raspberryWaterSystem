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

const CreateWeeklyProgramCommandName = "createWeeklyProgram"

type CreateWeeklyProgramCommand struct {
	Weekly *program.Weekly
}

func (c CreateWeeklyProgramCommand) Name() string {
	return CreateWeeklyProgramCommandName
}

type CreateWeeklyProgram struct {
	repo   WeeklyProgramRepository
	tracer trace.Tracer
}

func (c CreateWeeklyProgram) Handle(ctx context.Context, cmd cqs.Command) ([]event.Event, error) {
	ctx, span := c.tracer.Start(ctx, "CreateWeeklyProgramCmd")
	defer span.End()
	co, ok := cmd.(CreateWeeklyProgramCommand)
	if !ok {
		err := cqs.NewInvalidCommandError(CreateWeeklyProgramCommandName, cmd.Name())
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	day := co.Weekly.WeekDay()
	_, err := c.repo.FindByDay(ctx, &day)
	switch {
	case err == nil:
		err = CreateWeeklyProgramError{msg: fmt.Sprintf("a weekly program with day %s already exists", day.String())}
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	case errors.As(err, &errs.NotFoundError{}):
		if err = c.repo.Save(ctx, co.Weekly); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return nil, err
		}
		span.SetStatus(codes.Ok, "weekly program created")
		return nil, nil
	default:
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
}

func NewCreateWeeklyProgram(repo WeeklyProgramRepository, tracer trace.Tracer) *CreateWeeklyProgram {
	return &CreateWeeklyProgram{repo: repo, tracer: tracer}
}

type CreateWeeklyProgramError struct {
	msg string
}

func (c CreateWeeklyProgramError) Error() string {
	return "failed to create a weekly program: " + c.msg
}
