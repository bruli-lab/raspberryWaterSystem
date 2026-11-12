package app

import (
	"context"
	"errors"
	"time"

	"github.com/bruli-lab/go-core/cqs"
	"github.com/bruli/raspberryWaterSystem/internal/domain/program"
	errs "github.com/bruli/raspberryWaterSystem/internal/errors"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const FindProgramsInTimeQueryName = "findProgramsInTime"

type FindProgramsInTimeQuery struct {
	On          time.Time
	Temperature float32
}

func (f FindProgramsInTimeQuery) Name() string {
	return FindProgramsInTimeQueryName
}

type FindProgramsInTime struct {
	Daily, Odd, Even ProgramRepository
	Weekly           WeeklyProgramRepository
	Temperature      TemperatureProgramRepository
	tracer           trace.Tracer
}

func (f *FindProgramsInTime) Handle(ctx context.Context, query cqs.Query) (any, error) {
	ctx, span := f.tracer.Start(ctx, "FindProgramsInTime")
	defer span.End()
	q, ok := query.(FindProgramsInTimeQuery)
	if !ok {
		err := cqs.NewInvalidQueryError(FindProgramsInTimeQueryName, query.Name())
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	hour, _ := program.ParseHour(q.On.Format(program.HourLayout))
	daily, err := f.findDaily(ctx, &hour)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	odd, err := f.findOdd(ctx, &hour)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	even, err := f.findEven(ctx, &hour)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	weekly, err := f.findWeekly(ctx, hour, q.On.Weekday())
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	temp, err := f.findTemperature(ctx, hour, q.Temperature)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	span.SetStatus(codes.Ok, "programs found")
	return ProgramsInTime{
		Daily:       daily,
		Odd:         odd,
		Even:        even,
		Weekly:      weekly,
		Temperature: temp,
	}, nil
}

func (f *FindProgramsInTime) findDaily(ctx context.Context, hour *program.Hour) (*program.FertilizerProgram, error) {
	daily, err := f.Daily.FindByHour(ctx, hour)
	return f.findProgram(err, daily)
}

func (f *FindProgramsInTime) findProgram(err error, prgInTime *program.FertilizerProgram) (*program.FertilizerProgram, error) {
	if err != nil {
		if !errors.As(err, &errs.NotFoundError{}) {
			return nil, err
		}
	}
	return prgInTime, nil
}

func (f *FindProgramsInTime) findOdd(ctx context.Context, hour *program.Hour) (*program.FertilizerProgram, error) {
	odd, err := f.Odd.FindByHour(ctx, hour)
	return f.findProgram(err, odd)
}

func (f *FindProgramsInTime) findEven(ctx context.Context, hour *program.Hour) (*program.FertilizerProgram, error) {
	even, err := f.Even.FindByHour(ctx, hour)
	return f.findProgram(err, even)
}

func (f *FindProgramsInTime) findWeekly(ctx context.Context, hour program.Hour, day time.Weekday) (*program.Weekly, error) {
	weekDay := program.WeekDay(day)
	weekly, err := f.Weekly.FindByDayAndHour(ctx, &weekDay, &hour)
	if err != nil {
		if !errors.As(err, &errs.NotFoundError{}) {
			return nil, err
		}
	}
	return weekly, nil
}

func (f *FindProgramsInTime) findTemperature(ctx context.Context, hour program.Hour, temperature float32) (*program.Temperature, error) {
	temp, err := f.Temperature.FindByTemperatureAndHour(ctx, temperature, hour)
	if err != nil {
		if !errors.As(err, &errs.NotFoundError{}) {
			return nil, err
		}
	}
	return &temp, nil
}

func NewFindProgramsInTime(
	daily, odd, even ProgramRepository,
	weekly WeeklyProgramRepository,
	temperature TemperatureProgramRepository,
	tracer trace.Tracer,
) *FindProgramsInTime {
	return &FindProgramsInTime{Daily: daily, Odd: odd, Even: even, Weekly: weekly, Temperature: temperature, tracer: tracer}
}

type ProgramsInTime struct {
	Daily, Odd, Even *program.FertilizerProgram
	Weekly           *program.Weekly
	Temperature      *program.Temperature
}
