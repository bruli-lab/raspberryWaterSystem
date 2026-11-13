package app

import (
	"context"

	"github.com/bruli-lab/go-core/cqs"
	"github.com/bruli-lab/go-core/event"
	"github.com/bruli/raspberryWaterSystem/internal/domain/program"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const (
	RemoveDailyProgramCommandName = "removeDailyProgram"
	RemoveOddProgramCommandName   = "removeOddProgram"
	RemoveEvenProgramCommandName  = "removeEvenProgram"
)

type RemoveDailyProgramCommand struct {
	Hour *program.Hour
}

func (c RemoveDailyProgramCommand) Name() string {
	return RemoveDailyProgramCommandName
}

type RemoveOddProgramCommand struct {
	Hour *program.Hour
}

func (c RemoveOddProgramCommand) Name() string {
	return RemoveOddProgramCommandName
}

type RemoveEvenProgramCommand struct {
	Hour *program.Hour
}

func (c RemoveEvenProgramCommand) Name() string {
	return RemoveEvenProgramCommandName
}

type RemoveDailyProgram struct {
	RemoveProgram
	tracer trace.Tracer
}

func (r RemoveDailyProgram) Handle(ctx context.Context, cmd cqs.Command) ([]event.Event, error) {
	ctx, span := r.tracer.Start(ctx, "RemoveDailyProgramCmd")
	defer span.End()
	co, ok := cmd.(RemoveDailyProgramCommand)
	if !ok {
		err := cqs.NewInvalidCommandError(RemoveDailyProgramCommandName, cmd.Name())
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	if err := r.Remove(ctx, co.Hour); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	span.SetStatus(codes.Ok, "program removed")
	return nil, nil
}

func NewRemoveDailyProgram(pr ProgramRepository, tracer trace.Tracer) *RemoveDailyProgram {
	return &RemoveDailyProgram{pr: pr, tracer: tracer}
}

type RemoveOddProgram struct {
	RemoveProgram
	tracer trace.Tracer
}

func (r RemoveOddProgram) Handle(ctx context.Context, cmd cqs.Command) ([]event.Event, error) {
	ctx, span := r.tracer.Start(ctx, "RemoveOddProgramCmd")
	defer span.End()
	co, ok := cmd.(RemoveOddProgramCommand)
	if !ok {
		err := cqs.NewInvalidCommandError(RemoveOddProgramCommandName, cmd.Name())
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	if err := r.Remove(ctx, co.Hour); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	span.SetStatus(codes.Ok, "odd program removed")
	return nil, nil
}

func NewRemoveOddProgram(pr ProgramRepository, tracer trace.Tracer) *RemoveOddProgram {
	return &RemoveOddProgram{
		pr: pr, tracer: tracer,
	}
}

type RemoveEvenProgram struct {
	RemoveProgram
	tracer trace.Tracer
}

func (r RemoveEvenProgram) Handle(ctx context.Context, cmd cqs.Command) ([]event.Event, error) {
	ctx, span := r.tracer.Start(ctx, "RemoveEvenProgramCmd")
	defer span.End()
	co, ok := cmd.(RemoveEvenProgramCommand)
	if !ok {
		err := cqs.NewInvalidCommandError(RemoveEvenProgramCommandName, cmd.Name())
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	if err := r.Remove(ctx, co.Hour); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	span.SetStatus(codes.Ok, "even program removed")
	return nil, nil
}

func NewRemoveEvenProgram(pr ProgramRepository, tracer trace.Tracer) *RemoveEvenProgram {
	return &RemoveEvenProgram{pr: pr, tracer: tracer}
}

type RemoveProgram struct {
	pr ProgramRepository
}

func (r RemoveProgram) Remove(ctx context.Context, hour *program.Hour) error {
	if _, err := r.pr.FindByHour(ctx, hour); err != nil {
		return err
	}
	return r.pr.Remove(ctx, hour)
}
