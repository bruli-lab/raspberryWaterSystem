package disk

import (
	"context"

	"github.com/bruli/raspberryWaterSystem/internal/errors"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/bruli/raspberryWaterSystem/internal/domain/program"
)

type weeklyMap = map[string]programMap

type WeeklyRepository struct {
	path   string
	tracer trace.Tracer
}

func (w WeeklyRepository) FindByDay(ctx context.Context, day *program.WeekDay) (*program.Weekly, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
		_, span := w.tracer.Start(ctx, "WeeklyRepository.FindByDay")
		defer span.End()
		defer lockFile(w.path)()
		weekly := make(weeklyMap)
		if err := readYamlFile(w.path, &weekly); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return nil, err
		}
		byDay, ok := weekly[day.String()]
		if !ok {
			err := errors.NotFoundError{}
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return nil, err
		}
		weeklyByDay, skipped := buildProgramWeeklyByDay(day, byDay)
		recordSkippedEntries(span, skipped)
		span.SetStatus(codes.Ok, "weekly found")
		return weeklyByDay, nil
	}
}

func buildProgramWeeklyByDay(day *program.WeekDay, prg programMap) (*program.Weekly, []error) {
	programs, skipped := buildPrograms(prg)
	weekly := program.HydrateWeekly(*day, programs)
	return &weekly, skipped
}

func (w WeeklyRepository) Remove(ctx context.Context, day *program.WeekDay) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		_, span := w.tracer.Start(ctx, "WeeklyRepository.Remove")
		defer span.End()
		defer lockFile(w.path)()
		weekly := make(weeklyMap)
		if err := readYamlFile(w.path, &weekly); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return err
		}
		delete(weekly, day.String())
		if err := writeYamlFile(w.path, &weekly); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return err
		}
		span.SetStatus(codes.Ok, "weekly removed")
		return nil
	}
}

func (w WeeklyRepository) FindByDayAndHour(ctx context.Context, day *program.WeekDay, hour *program.Hour) (*program.Weekly, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
		_, span := w.tracer.Start(ctx, "WeeklyRepository.FindByDayAndHour")
		defer span.End()
		defer lockFile(w.path)()
		weekly := make(weeklyMap)
		if err := readYamlFile(w.path, &weekly); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return nil, err
		}
		byDay, ok := weekly[day.String()]
		if !ok {
			err := errors.NewNotFoundError(day.String())
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return nil, err
		}
		byHour, ok := byDay[hour.String()]
		if !ok {
			err := errors.NewNotFoundError(hour.String())
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return nil, err
		}
		span.SetStatus(codes.Ok, "weekly and hour found")
		return buildProgramWeeklyByHour(day, hour, byHour), nil
	}
}

func buildProgramWeeklyByHour(day *program.WeekDay, hour *program.Hour, prgms []executions) *program.Weekly {
	programs := []program.Program{program.HydrateProgramWithoutFertilizer(*hour, buildProgramExecutions(prgms))}
	weekly := program.HydrateWeekly(*day, programs)
	return &weekly
}

func (w WeeklyRepository) Save(ctx context.Context, prg *program.Weekly) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		_, span := w.tracer.Start(ctx, "WeeklyRepository.Save")
		defer span.End()
		defer lockFile(w.path)()
		weekly := make(weeklyMap)
		if err := readYamlFile(w.path, &weekly); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return err
		}
		weekly[prg.WeekDay().String()] = buildProgramMap(prg.Programs())
		if err := writeYamlFile(w.path, weekly); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return err
		}
		span.SetStatus(codes.Ok, "weekly saved")
		return nil
	}
}

func (w WeeklyRepository) FindAll(ctx context.Context) ([]program.Weekly, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
		_, span := w.tracer.Start(ctx, "WeeklyRepository.FindAll")
		defer span.End()
		defer lockFile(w.path)()
		weekly := make(weeklyMap)
		if err := readYamlFile(w.path, &weekly); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return nil, err
		}
		weeklies, skipped := buildWeeklyPrograms(weekly)
		recordSkippedEntries(span, skipped)
		span.SetStatus(codes.Ok, "weeklys found")
		return weeklies, nil
	}
}

func buildWeeklyPrograms(weekly weeklyMap) ([]program.Weekly, []error) {
	prgms := make([]program.Weekly, 0, len(weekly))
	var skipped []error
	for dayStr, w := range weekly {
		day, err := program.ParseWeekDay(dayStr)
		if err != nil {
			skipped = append(skipped, InvalidStoredEntryError{key: dayStr, cause: err})
			continue
		}
		programs, skippedPrograms := buildPrograms(w)
		skipped = append(skipped, skippedPrograms...)
		prgms = append(prgms, program.HydrateWeekly(day, programs))
	}
	return prgms, skipped
}

func NewWeeklyRepository(path string, tracer trace.Tracer) WeeklyRepository {
	return WeeklyRepository{path: path, tracer: tracer}
}
