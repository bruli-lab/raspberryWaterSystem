package app

import (
	"context"

	"github.com/bruli-lab/go-core/cqs"
	"github.com/bruli/raspberryWaterSystem/internal/domain/program"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const FindAllProgramsQueryName = "findAllPrograms"

type FindAllProgramsQuery struct{}

func (f FindAllProgramsQuery) Name() string {
	return FindAllProgramsQueryName
}

type FindAllPrograms struct {
	daily, odd, even ProgramRepository
	weekly           WeeklyProgramRepository
	temperature      TemperatureProgramRepository
	tracer           trace.Tracer
}

func (f *FindAllPrograms) Handle(ctx context.Context, _ cqs.Query) (any, error) {
	ctx, span := f.tracer.Start(ctx, "FindAllPrograms")
	defer span.End()
	dailies, err := f.daily.FindAll(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	odd, err := f.odd.FindAll(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	even, err := f.even.FindAll(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	weekly, err := f.weekly.FindAll(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	temp, err := f.temperature.FindAll(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	span.SetStatus(codes.Ok, "programs found")
	return AllPrograms{
		Daily:       dailies,
		Odd:         odd,
		Even:        even,
		Weekly:      weekly,
		Temperature: temp,
	}, nil
}

func NewFindAllPrograms(
	daily ProgramRepository,
	odd ProgramRepository,
	even ProgramRepository,
	weekly WeeklyProgramRepository,
	temperature TemperatureProgramRepository,
	tracer trace.Tracer,
) *FindAllPrograms {
	return &FindAllPrograms{daily: daily, odd: odd, even: even, weekly: weekly, temperature: temperature, tracer: tracer}
}

type AllPrograms struct {
	Daily, Odd, Even []program.Program
	Weekly           []program.Weekly
	Temperature      []program.Temperature
}
