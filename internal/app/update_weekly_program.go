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
	WeekDay  program.WeekDay
	Programs []program.Program
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
	weekly, err := u.repo.FindByDay(ctx, &co.WeekDay)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	weekly.Update(co.Programs)

	if err = u.repo.Save(ctx, weekly); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	span.SetStatus(codes.Ok, "weekly program updated")
	return nil, nil
}

func NewUpdateWeeklyProgram(repo WeeklyProgramRepository, tracer trace.Tracer) *UpdateWeeklyProgram {
	return &UpdateWeeklyProgram{repo: repo, tracer: tracer}
}
