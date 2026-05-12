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
	trace "go.opentelemetry.io/otel/trace"
)

const (
	CreateDailyProgramCommandName = "createDailyProgram"
	CreateOddProgramCommandName   = "createOddProgram"
	CreateEvenProgramCommandName  = "createEvenProgram"
)

type CreateDailyProgramCommand struct {
	Program *program.FertilizerProgram
}

func (c CreateDailyProgramCommand) Name() string {
	return CreateDailyProgramCommandName
}

type CreateOddProgramCommand struct {
	Program *program.FertilizerProgram
}

func (c CreateOddProgramCommand) Name() string {
	return CreateOddProgramCommandName
}

type CreateEvenProgramCommand struct {
	Program *program.FertilizerProgram
}

func (c CreateEvenProgramCommand) Name() string {
	return CreateEvenProgramCommandName
}

type CreateDailyProgram struct {
	CreateProgram
	tracer trace.Tracer
}

func (c CreateDailyProgram) Handle(ctx context.Context, cmd cqs.Command) ([]event.Event, error) {
	ctx, span := c.tracer.Start(ctx, "CreateDailyProgramCmd")
	defer span.End()
	co, ok := cmd.(CreateDailyProgramCommand)
	if !ok {
		err := cqs.NewInvalidCommandError(CreateDailyProgramCommandName, cmd.Name())
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	if err := c.Create(ctx, co.Program); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	span.SetStatus(codes.Ok, "program created")
	return nil, nil
}

func NewCreateDailyProgram(programRepo ProgramRepository, zonesRepo ZoneRepository, tracer trace.Tracer) *CreateDailyProgram {
	return &CreateDailyProgram{
		programRepo: programRepo, zonesRepo: zonesRepo,
		tracer: tracer,
	}
}

type CreateOddProgram struct {
	CreateProgram
	tracer trace.Tracer
}

func (c CreateOddProgram) Handle(ctx context.Context, cmd cqs.Command) ([]event.Event, error) {
	ctx, span := c.tracer.Start(ctx, "CreateOddProgramCmd")
	co, ok := cmd.(CreateOddProgramCommand)
	if !ok {
		err := cqs.NewInvalidCommandError(CreateOddProgramCommandName, cmd.Name())
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	if err := c.Create(ctx, co.Program); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	span.SetStatus(codes.Ok, "odd program created")
	return nil, nil
}

func NewCreateOddProgram(programRepo ProgramRepository, zonesRepo ZoneRepository, tracer trace.Tracer) *CreateOddProgram {
	return &CreateOddProgram{
		programRepo: programRepo, zonesRepo: zonesRepo,
		tracer: tracer,
	}
}

type CreateEvenProgram struct {
	CreateProgram
	tracer trace.Tracer
}

func (c CreateEvenProgram) Handle(ctx context.Context, cmd cqs.Command) ([]event.Event, error) {
	ctx, span := c.tracer.Start(ctx, "CreateEvenProgramCmd")
	defer span.End()
	co, ok := cmd.(CreateEvenProgramCommand)
	if !ok {
		err := cqs.NewInvalidCommandError(CreateEvenProgramCommandName, cmd.Name())
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	if err := c.Create(ctx, co.Program); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	span.SetStatus(codes.Ok, "even program created")
	return nil, nil
}

func NewCreateEvenProgram(programRepo ProgramRepository, zonesRepo ZoneRepository, tracer trace.Tracer) *CreateEvenProgram {
	return &CreateEvenProgram{
		programRepo: programRepo, zonesRepo: zonesRepo,
		tracer: tracer,
	}
}

type CreateProgramError struct {
	msg string
}

func (c CreateProgramError) Error() string {
	return "failed to create program: " + c.msg
}

type CreateProgram struct {
	programRepo ProgramRepository
	zonesRepo   ZoneRepository
}

func (p CreateProgram) Create(ctx context.Context, prog *program.FertilizerProgram) error {
	hour := prog.Hour()
	_, err := p.programRepo.FindByHour(ctx, &hour)
	switch {
	case err == nil:
		return CreateProgramError{msg: fmt.Sprintf("a program with hour %s, already exists", hour.String())}
	case errors.As(err, &errs.NotFoundError{}):
		if err := p.checkZone(ctx, prog.Executions()); err != nil {
			return err
		}
		return p.programRepo.Save(ctx, prog)
	default:
		return err
	}
}

func (p CreateProgram) checkZone(ctx context.Context, executions []program.Execution) error {
	for _, e := range executions {
		for _, z := range e.Zones() {
			if _, err := p.zonesRepo.FindByID(ctx, z); err != nil {
				switch {
				case errors.As(err, &errs.NotFoundError{}):
					return CreateProgramError{msg: fmt.Sprintf("a zone with id %s, not found", z)}
				default:
					return err
				}
			}
		}
	}
	return nil
}
