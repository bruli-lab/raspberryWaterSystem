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

const (
	UpdateDailyProgramCommandName = "updateDailyProgram"
	UpdateOddProgramCommandName   = "updateOddProgram"
	UpdateEvenProgramCommandName  = "updateEvenProgram"
)

type UpdateDailyProgramCommand struct {
	Program *program.Program
}

func (u UpdateDailyProgramCommand) Name() string {
	return UpdateDailyProgramCommandName
}

type UpdateOddProgramCommand struct {
	Program *program.Program
}

func (u UpdateOddProgramCommand) Name() string {
	return UpdateOddProgramCommandName
}

type UpdateEvenProgramCommand struct {
	Program *program.Program
}

func (u UpdateEvenProgramCommand) Name() string {
	return UpdateEvenProgramCommandName
}

type UpdateDailyProgram struct {
	UpdateProgram
	tracer trace.Tracer
}

func (u UpdateDailyProgram) Handle(ctx context.Context, cmd cqs.Command) ([]event.Event, error) {
	ctx, span := u.tracer.Start(ctx, "UpdateDailyProgramCmd")
	defer span.End()
	co, ok := cmd.(UpdateDailyProgramCommand)
	if !ok {
		err := cqs.NewInvalidCommandError(UpdateDailyProgramCommandName, cmd.Name())
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	if err := u.Update(ctx, co.Program); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	span.SetStatus(codes.Ok, "program updated")
	return nil, nil
}

func NewUpdateDailyProgram(programRepo ProgramRepository, zonesRepo ZoneRepository, tracer trace.Tracer) *UpdateDailyProgram {
	return &UpdateDailyProgram{
		programRepo: programRepo, zonesRepo: zonesRepo,
		tracer: tracer,
	}
}

type UpdateOddProgram struct {
	UpdateProgram
	tracer trace.Tracer
}

func (u UpdateOddProgram) Handle(ctx context.Context, cmd cqs.Command) ([]event.Event, error) {
	ctx, span := u.tracer.Start(ctx, "UpdateOddProgramCmd")
	defer span.End()
	co, ok := cmd.(UpdateOddProgramCommand)
	if !ok {
		err := cqs.NewInvalidCommandError(UpdateOddProgramCommandName, cmd.Name())
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	if err := u.Update(ctx, co.Program); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	span.SetStatus(codes.Ok, "odd program updated")
	return nil, nil
}

func NewUpdateOddProgram(programRepo ProgramRepository, zonesRepo ZoneRepository, tracer trace.Tracer) *UpdateOddProgram {
	return &UpdateOddProgram{
		programRepo: programRepo, zonesRepo: zonesRepo,
		tracer: tracer,
	}
}

type UpdateEvenProgram struct {
	UpdateProgram
	tracer trace.Tracer
}

func (u UpdateEvenProgram) Handle(ctx context.Context, cmd cqs.Command) ([]event.Event, error) {
	ctx, span := u.tracer.Start(ctx, "UpdateEvenProgramCmd")
	defer span.End()
	co, ok := cmd.(UpdateEvenProgramCommand)
	if !ok {
		err := cqs.NewInvalidCommandError(UpdateEvenProgramCommandName, cmd.Name())
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	if err := u.Update(ctx, co.Program); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	span.SetStatus(codes.Ok, "even program updated")
	return nil, nil
}

func NewUpdateEvenProgram(programRepo ProgramRepository, zonesRepo ZoneRepository, tracer trace.Tracer) *UpdateEvenProgram {
	return &UpdateEvenProgram{
		programRepo: programRepo, zonesRepo: zonesRepo,
		tracer: tracer,
	}
}

type UpdateProgramError struct {
	msg string
}

func (u UpdateProgramError) Error() string {
	return "failed to update program: " + u.msg
}

type UpdateProgram struct {
	programRepo ProgramRepository
	zonesRepo   ZoneRepository
}

func (u UpdateProgram) Update(ctx context.Context, prog *program.Program) error {
	hour := prog.Hour()
	if _, err := u.programRepo.FindByHour(ctx, &hour); err != nil {
		return err
	}
	if err := u.checkZone(ctx, prog.Executions()); err != nil {
		return err
	}
	return u.programRepo.Save(ctx, prog)
}

func (u UpdateProgram) checkZone(ctx context.Context, executions []program.Execution) error {
	for _, e := range executions {
		for _, z := range e.Zones() {
			if _, err := u.zonesRepo.FindByID(ctx, z); err != nil {
				switch {
				case errors.As(err, &errs.NotFoundError{}):
					return UpdateProgramError{msg: fmt.Sprintf("a zone with id %s, not found", z)}
				default:
					return err
				}
			}
		}
	}
	return nil
}
