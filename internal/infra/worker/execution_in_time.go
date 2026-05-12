package worker

import (
	"context"
	"time"

	"github.com/bruli-lab/go-core/cqs"
	"github.com/bruli/raspberryWaterSystem/internal/app"
	"github.com/bruli/raspberryWaterSystem/internal/domain/program"
	"github.com/bruli/raspberryWaterSystem/internal/domain/status"
	"github.com/bruli/raspberryWaterSystem/internal/domain/weather"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

func ExecutionInTime(ctx context.Context, qh cqs.QueryHandler, ch cqs.CommandHandler, now time.Time, tracer trace.Tracer) error {
	ctx, span := tracer.Start(ctx, "ExecutionInTime worker")
	defer span.End()
	st, err := readingCurrentStatus(ctx, qh)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	temp := st.Weather().Temperature()
	prgms, err := findingPrograms(ctx, qh, now, temp)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	if err = executeDaily(ctx, prgms, ch, now); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	if err = executeOddEven(ctx, now, prgms, ch); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	if err = executeWeekly(ctx, prgms, ch, now); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	if err = executeTemperature(ctx, prgms, ch, now, temp); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	span.SetStatus(codes.Ok, "execution in time")
	return nil
}

func findingPrograms(ctx context.Context, qh cqs.QueryHandler, now time.Time, temp weather.Temperature) (app.ProgramsInTime, error) {
	resultPrmgs, err := qh.Handle(ctx, app.FindProgramsInTimeQuery{
		On:          now,
		Temperature: temp.Float32(),
	})
	if err != nil {
		return app.ProgramsInTime{}, FindProgramsError{err: err}
	}
	prgms, _ := resultPrmgs.(app.ProgramsInTime)
	return prgms, nil
}

func readingCurrentStatus(ctx context.Context, qh cqs.QueryHandler) (status.Status, error) {
	resultSt, err := qh.Handle(ctx, app.FindStatusQuery{})
	if err != nil {
		return status.Status{}, ReadCurrentStatusError{err: err}
	}
	st, _ := resultSt.(status.Status)
	return st, nil
}

func executeTemperature(ctx context.Context, prgms app.ProgramsInTime, ch cqs.CommandHandler, now time.Time, currentTemp weather.Temperature) error {
	if prgms.Temperature != nil {
		if currentTemp.Float32() >= prgms.Temperature.Temperature() {
			for _, pr := range prgms.Temperature.Programs() {
				if err := executeProgram(ctx, ch, pr, now); err != nil {
					return ExecuteTemperatureError{err: err}
				}
			}
		}
	}
	return nil
}

func executeWeekly(ctx context.Context, prgms app.ProgramsInTime, ch cqs.CommandHandler, now time.Time) error {
	if prgms.Weekly != nil {
		if now.Weekday().String() == prgms.Weekly.WeekDay().String() {
			for _, pr := range prgms.Weekly.Programs() {
				if err := executeProgram(ctx, ch, pr, now); err != nil {
					return ExecuteWeeklyError{err: err}
				}
			}
		}
	}
	return nil
}

func executeOddEven(ctx context.Context, now time.Time, prgms app.ProgramsInTime, ch cqs.CommandHandler) error {
	var oddEvenPrgms *program.FertilizerProgram

	switch {
	case isEven(now) && prgms.Even != nil:
		oddEvenPrgms = prgms.Even
	case !isEven(now) && prgms.Odd != nil:
		oddEvenPrgms = prgms.Odd
	}
	if oddEvenPrgms != nil {
		if err := executeFertilizerProgram(ctx, ch, *oddEvenPrgms, now); err != nil {
			return ExecuteOddEvenError{err: err}
		}
	}
	return nil
}

func executeDaily(ctx context.Context, prgms app.ProgramsInTime, ch cqs.CommandHandler, now time.Time) error {
	if prgms.Daily != nil {
		if err := executeFertilizerProgram(ctx, ch, *prgms.Daily, now); err != nil {
			return ExecuteDailyError{err: err}
		}
	}
	return nil
}

func isEven(now time.Time) bool {
	day := now.Day()
	rest := day % 2
	return rest == 0
}

func executeFertilizerProgram(ctx context.Context, ch cqs.CommandHandler, prg program.FertilizerProgram, now time.Time) error {
	nowHour := now.Format(program.HourLayout)
	if nowHour == prg.Hour().String() {
		for _, exec := range prg.Executions() {
			for _, zo := range exec.Zones() {
				if _, err := ch.Handle(ctx, app.ExecuteZoneWithStatusCmd{
					Seconds:        uint(exec.Seconds().Int()),
					ZoneID:         zo,
					WithFertilizer: prg.IsFertilizer(),
				}); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func executeProgram(ctx context.Context, ch cqs.CommandHandler, prg program.Program, now time.Time) error {
	nowHour := now.Format(program.HourLayout)
	if nowHour == prg.Hour().String() {
		for _, exec := range prg.Executions() {
			for _, zo := range exec.Zones() {
				if _, err := ch.Handle(ctx, app.ExecuteZoneWithStatusCmd{
					Seconds:        uint(exec.Seconds().Int()),
					ZoneID:         zo,
					WithFertilizer: prg.IsFertilizer(),
				}); err != nil {
					return err
				}
			}
		}
	}
	return nil
}
