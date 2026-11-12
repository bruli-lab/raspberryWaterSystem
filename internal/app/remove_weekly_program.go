package app

import (
	"context"

	"github.com/bruli-lab/go-core/cqs"
	"github.com/bruli-lab/go-core/event"
	"github.com/bruli/raspberryWaterSystem/internal/domain/program"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const RemoveWeeklyProgramCommandName = "removeWeeklyProgram"

type RemoveWeeklyProgramCommand struct {
	Day *program.WeekDay
}

func (r RemoveWeeklyProgramCommand) Name() string {
	return RemoveWeeklyProgramCommandName
}

type RemoveWeeklyProgram struct {
	repo   WeeklyProgramRepository
	tracer trace.Tracer
}

func (r RemoveWeeklyProgram) Handle(ctx context.Context, cmd cqs.Command) ([]event.Event, error) {
	ctx, span := r.tracer.Start(ctx, "RemoveWeeklyProgramCmd")
	defer span.End()
	co, ok := cmd.(RemoveWeeklyProgramCommand)
	if !ok {
		err := cqs.NewInvalidCommandError(RemoveWeeklyProgramCommandName, cmd.Name())
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	if _, err := r.repo.FindByDay(ctx, co.Day); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	if err := r.repo.Remove(ctx, co.Day); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	span.SetStatus(codes.Ok, "weekly program removed")
	return nil, nil
}

func NewRemoveWeeklyProgram(repo WeeklyProgramRepository, tracer trace.Tracer) *RemoveWeeklyProgram {
	return &RemoveWeeklyProgram{repo: repo, tracer: tracer}
}
