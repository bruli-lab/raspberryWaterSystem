package app

import (
	"context"

	"github.com/bruli-lab/go-core/cqs"
	"github.com/bruli-lab/go-core/event"
	"github.com/bruli/raspberryWaterSystem/internal/domain/program"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const UpdateWeeklyProgramCommandName = "updateWeeklyProgram"

type UpdateWeeklyProgramCommand struct {
	WeekDay  string
	Programs []ProgramInput
}

func (u UpdateWeeklyProgramCommand) Name() string {
	return UpdateWeeklyProgramCommandName
}

type UpdateWeeklyProgram struct {
	repo   WeeklyProgramRepository
	tracer trace.Tracer
}

func (u UpdateWeeklyProgram) Handle(ctx context.Context, cmd cqs.Command) ([]event.Event, error) {
	ctx, span := u.tracer.Start(ctx, "UpdateWeeklyProgramCmd")
	defer span.End()
	co, ok := cmd.(UpdateWeeklyProgramCommand)
	if !ok {
		err := cqs.NewInvalidCommandError(UpdateWeeklyProgramCommandName, cmd.Name())
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	if err := u.update(ctx, co); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	span.SetStatus(codes.Ok, "weekly program updated")
	return nil, nil
}

func (u UpdateWeeklyProgram) update(ctx context.Context, co UpdateWeeklyProgramCommand) error {
	day, err := program.ParseWeekDay(co.WeekDay)
	if err != nil {
		return UpdateWeeklyProgramError{msg: err.Error()}
	}
	programs, err := buildProgramsWithoutFertilizer(co.Programs)
	if err != nil {
		return UpdateWeeklyProgramError{msg: err.Error()}
	}
	weekly, err := u.repo.FindByDay(ctx, &day)
	if err != nil {
		return err
	}
	if err = weekly.Update(programs); err != nil {
		return UpdateWeeklyProgramError{msg: err.Error()}
	}
	return u.repo.Save(ctx, weekly)
}

type UpdateWeeklyProgramError struct {
	msg string
}

func (u UpdateWeeklyProgramError) Error() string {
	return "failed to update weekly program: " + u.msg
}

func NewUpdateWeeklyProgram(repo WeeklyProgramRepository, tracer trace.Tracer) *UpdateWeeklyProgram {
	return &UpdateWeeklyProgram{repo: repo, tracer: tracer}
}
