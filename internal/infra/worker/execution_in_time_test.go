package worker_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/bruli-lab/go-core/cqs"
	"github.com/bruli-lab/go-core/event"
	"github.com/bruli/raspberryWaterSystem/internal/app"
	"github.com/bruli/raspberryWaterSystem/internal/domain/program"
	"github.com/bruli/raspberryWaterSystem/internal/fixtures"
	"github.com/bruli/raspberryWaterSystem/internal/infra/worker"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

func programsInTime(t *testing.T, hour program.Hour, day time.Weekday, temperature float32) app.ProgramsInTime {
	t.Helper()
	programFor := func(zoneID string, fertilizer bool) program.Program {
		return fixtures.ProgramBuilder{
			Hour:       &hour,
			Fertilizer: fertilizer,
			Executions: []program.Execution{fixtures.ExecutionBuilder{Zones: []string{zoneID}}.Build()},
		}.Build()
	}
	daily := programFor("daily", true)
	odd := programFor("odd", false)
	even := programFor("even", false)
	weekDay := program.WeekDay(day)
	weekly := fixtures.WeeklyBuilder{WeekDay: &weekDay, Programs: []program.Program{programFor("weekly", false)}}.Build()
	temp := fixtures.TemperatureBuilder{Temperature: &temperature, Programs: []program.Program{programFor("temperature", false)}}.Build()
	return app.ProgramsInTime{Daily: &daily, Odd: &odd, Even: &even, Weekly: &weekly, Temperature: &temp}
}

func TestExecutionInTime(t *testing.T) {
	oddDay := time.Date(2022, time.August, 5, 21, 0, 0, 0, time.UTC)
	evenDay := time.Date(2022, time.August, 6, 21, 0, 0, 0, time.UTC)
	hour, err := program.ParseHour(oddDay.Format(program.HourLayout))
	require.NoError(t, err)
	st := fixtures.StatusBuilder{Active: true}.Build()
	currentTemperature := st.Weather().Temperature().Float32()
	errTest := errors.New("test error")
	tests := []struct {
		name                    string
		now                     time.Time
		findStatusErr           error
		findProgramsErr         error
		programs                func(t *testing.T, now time.Time) app.ProgramsInTime
		failingZone             string
		expectedErr             any
		expectedZones           []string
		expectedFertilizerZones []string
	}{
		{
			name:          "and find status returns an error, then it returns a read current status error and executes nothing",
			now:           oddDay,
			findStatusErr: errTest,
			expectedErr:   &worker.ReadCurrentStatusError{},
		},
		{
			name:            "and find programs returns an error, then it returns a find programs error and executes nothing",
			now:             oddDay,
			findProgramsErr: errTest,
			expectedErr:     &worker.FindProgramsError{},
		},
		{
			name:          "and the daily program fails, then it returns an execute daily error",
			now:           oddDay,
			failingZone:   "daily",
			expectedErr:   &worker.ExecuteDailyError{},
			expectedZones: []string{"daily"},
		},
		{
			name:          "on an odd day and the odd program fails, then it returns an execute odd even error",
			now:           oddDay,
			failingZone:   "odd",
			expectedErr:   &worker.ExecuteOddEvenError{},
			expectedZones: []string{"daily", "odd"},
		},
		{
			name:          "on an even day and the even program fails, then it returns an execute odd even error",
			now:           evenDay,
			failingZone:   "even",
			expectedErr:   &worker.ExecuteOddEvenError{},
			expectedZones: []string{"daily", "even"},
		},
		{
			name:          "and the weekly program fails, then it returns an execute weekly error",
			now:           oddDay,
			failingZone:   "weekly",
			expectedErr:   &worker.ExecuteWeeklyError{},
			expectedZones: []string{"daily", "odd", "weekly"},
		},
		{
			name:          "and the temperature program fails, then it returns an execute temperature error",
			now:           oddDay,
			failingZone:   "temperature",
			expectedErr:   &worker.ExecuteTemperatureError{},
			expectedZones: []string{"daily", "odd", "weekly", "temperature"},
		},
		{
			name:                    "on an odd day and every program works, then it executes daily, odd, weekly and temperature programs with the daily fertilizer",
			now:                     oddDay,
			expectedZones:           []string{"daily", "odd", "weekly", "temperature"},
			expectedFertilizerZones: []string{"daily"},
		},
		{
			name:                    "on an even day and every program works, then it executes the even program instead of the odd one",
			now:                     evenDay,
			expectedZones:           []string{"daily", "even", "weekly", "temperature"},
			expectedFertilizerZones: []string{"daily"},
		},
		{
			name: "and the weekly program is for another day and the temperature program needs more heat, then it executes only daily and odd programs",
			now:  oddDay,
			programs: func(t *testing.T, now time.Time) app.ProgramsInTime {
				t.Helper()
				return programsInTime(t, hour, now.Weekday()+1, currentTemperature+1)
			},
			expectedZones:           []string{"daily", "odd"},
			expectedFertilizerZones: []string{"daily"},
		},
	}
	for _, tt := range tests {
		t.Run(`Given a ExecutionInTime function, when is called `+tt.name, func(t *testing.T) {
			t.Parallel()
			programs := programsInTime(t, hour, tt.now.Weekday(), currentTemperature)
			if tt.programs != nil {
				programs = tt.programs(t, tt.now)
			}
			qh := &cqs.QueryHandlerMock{
				HandleFunc: func(_ context.Context, query cqs.Query) (any, error) {
					switch query.(type) {
					case app.FindStatusQuery:
						return st, tt.findStatusErr
					case app.FindProgramsInTimeQuery:
						return programs, tt.findProgramsErr
					default:
						return nil, nil
					}
				},
			}
			var (
				m               sync.Mutex
				zones           []string
				fertilizerZones []string
			)
			ch := &cqs.CommandHandlerMock{
				HandleFunc: func(_ context.Context, cmd cqs.Command) ([]event.Event, error) {
					execute, ok := cmd.(app.ExecuteZoneWithStatusCmd)
					if !ok {
						return nil, nil
					}
					m.Lock()
					defer m.Unlock()
					zones = append(zones, execute.ZoneID)
					if execute.WithFertilizer {
						fertilizerZones = append(fertilizerZones, execute.ZoneID)
					}
					if execute.ZoneID == tt.failingZone {
						return nil, errTest
					}
					return nil, nil
				},
			}

			err := worker.ExecutionInTime(t.Context(), qh, ch, tt.now, tracer())

			if tt.expectedErr != nil {
				require.ErrorAs(t, err, tt.expectedErr)
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, tt.expectedZones, zones)
			if tt.expectedErr == nil {
				require.Equal(t, tt.expectedFertilizerZones, fertilizerZones)
			}
		})
	}
}

func tracer() trace.Tracer {
	return noop.NewTracerProvider().Tracer("test")
}
