package worker_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/bruli/raspberryWaterSystem/internal/fixtures"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"

	"github.com/bruli/raspberryWaterSystem/internal/domain/program"

	"github.com/bruli/raspberryWaterSystem/internal/app"
	"github.com/bruli/raspberryWaterSystem/internal/cqs"
	"github.com/stretchr/testify/require"

	"github.com/bruli/raspberryWaterSystem/internal/infra/worker"
)

func TestExecutionInTime(t *testing.T) {
	now, err := time.Parse("2006-01-02T15:04", "2022-08-05T21:00")
	require.NoError(t, err)
	nowEven, err := time.Parse("2006-01-02T15:04", "2022-08-06T21:00")
	require.NoError(t, err)
	hour, err := program.ParseHour(now.Format(program.HourLayout))
	require.NoError(t, err)
	errTest := errors.New("")
	status := fixtures.StatusBuilder{Active: true}.Build()
	fertilizerProgram := fixtures.FertilizerProgramBuilder{Hour: &hour}.Build()
	prog := fixtures.ProgramBuilder{Hour: &hour}.Build()
	day := program.WeekDay(now.Weekday())
	weekly := fixtures.WeeklyBuilder{WeekDay: &day, Programs: []program.Program{
		prog,
	}}.Build()
	temperature := fixtures.TemperatureBuilder{Temperature: new(status.Weather().Temperature().Float32()), Programs: []program.Program{
		prog,
	}}.Build()
	progams := app.ProgramsInTime{
		Daily:       &fertilizerProgram,
		Odd:         &fertilizerProgram,
		Even:        &fertilizerProgram,
		Weekly:      &weekly,
		Temperature: &temperature,
	}
	tests := []struct {
		name string
		expectedErr, findStatusErr,
		findProgramsErr, dailyErr,
		oddEvenErr, weeklyErr, tempErr error
		statusResult, programsResult any
		now                          time.Time
	}{
		{
			name:          "and find status returns an error, then it returns a read current status error",
			now:           now,
			findStatusErr: errTest,
			expectedErr:   worker.ReadCurrentStatusError{},
		},
		{
			name:            "and find programs returns error, then it returns a find programs error",
			now:             now,
			statusResult:    status,
			expectedErr:     worker.FindProgramsError{},
			findProgramsErr: errTest,
		},
		{
			name:           "and execute daily returns error, then it returns an execute daily error",
			now:            now,
			statusResult:   status,
			programsResult: progams,
			expectedErr:    worker.ExecuteDailyError{},
			dailyErr:       errTest,
		},
		{
			name:           "and execute odd returns error, then it returns an execute odd event error",
			now:            now,
			statusResult:   status,
			programsResult: progams,
			expectedErr:    worker.ExecuteOddEvenError{},
			oddEvenErr:     errTest,
		},
		{
			name:           "and execute even returns error, then it returns an execute odd event error",
			now:            nowEven,
			statusResult:   status,
			programsResult: progams,
			expectedErr:    worker.ExecuteOddEvenError{},
			oddEvenErr:     errTest,
		},
		{
			name:           "and execute weekly returns error, then it returns an execute weekly error",
			now:            now,
			statusResult:   status,
			programsResult: progams,
			expectedErr:    worker.ExecuteWeeklyError{},
			weeklyErr:      errTest,
		},
		{
			name:           "and execute temperature returns error, then it returns an execute temperature error",
			now:            now,
			statusResult:   status,
			programsResult: progams,
			expectedErr:    worker.ExecuteTemperatureError{},
			tempErr:        errTest,
		},
		{
			name:           "then it returns nil",
			now:            now,
			statusResult:   status,
			programsResult: progams,
		},
	}
	for _, tt := range tests {
		t.Run(`Given a ExecutionInTime function,
		when is called `+tt.name, func(t *testing.T) {
			qh := &QueryHandlerMock{
				HandleFunc: func(_ context.Context, query cqs.Query) (any, error) {
					_, findStatus := query.(app.FindStatusQuery)
					if findStatus {
						return tt.statusResult, tt.findStatusErr
					}
					_, findPrograms := query.(app.FindProgramsInTimeQuery)
					if findPrograms {
						return tt.programsResult, tt.findProgramsErr
					}
					return nil, nil
				},
			}
			ch := &CommandHandlerMock{}
			ch.HandleFunc = func(_ context.Context, _ cqs.Command) ([]cqs.Event, error) {
				switch len(ch.HandleCalls()) {
				case 1:
					return nil, tt.dailyErr
				case 2:
					return nil, tt.oddEvenErr
				case 3:
					return nil, tt.weeklyErr
				case 4:
					return nil, tt.tempErr
				default:
					return nil, nil
				}
			}
			err = worker.ExecutionInTime(context.Background(), qh, ch, tt.now, tracer())
			if err != nil {
				require.ErrorAs(t, err, &tt.expectedErr)
				return
			}
			require.Equal(t, tt.expectedErr, err)
		})
	}
}

func tracer() trace.Tracer {
	return noop.NewTracerProvider().Tracer("test")
}
