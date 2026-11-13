package app_test

import (
	"context"
	"errors"
	"testing"

	"github.com/bruli-lab/go-core/cqs"
	"github.com/bruli/raspberryWaterSystem/internal/fixtures"

	"github.com/bruli/raspberryWaterSystem/internal/domain/program"

	"github.com/bruli/raspberryWaterSystem/internal/app"
	"github.com/stretchr/testify/require"
)

func TestFindProgramsInTimeHandle(t *testing.T) {
	errTest := errors.New("")
	daily := fixtures.ProgramBuilder{}.Build()
	odd := fixtures.ProgramBuilder{}.Build()
	even := fixtures.ProgramBuilder{}.Build()
	weekly := fixtures.WeeklyBuilder{}.Build()
	temp := fixtures.TemperatureBuilder{}.Build()
	queryTest := app.FindProgramsInTimeQuery{}
	tests := []struct {
		name string
		expectedErr, dailyErr,
		oddErr, evenErr,
		weeklyErr, tempErr error
		expectedResult   any
		daily, odd, even *program.Program
		weekly           *program.Weekly
		temp             program.Temperature
		query            cqs.Query
	}{
		{
			name:        "with an invalid query, then it returns an invalid command error",
			query:       invalidQuery{},
			expectedErr: cqs.InvalidQueryError{},
		},
		{
			name:        "and daily repository returns an error, then it returns same error",
			dailyErr:    errTest,
			expectedErr: errTest,
			query:       queryTest,
		},
		{
			name:        "and odd repository returns an error, then it returns same error",
			oddErr:      errTest,
			expectedErr: errTest,
			daily:       &daily,
			query:       queryTest,
		},
		{
			name:        "and even repository returns an error, then it returns same error",
			evenErr:     errTest,
			expectedErr: errTest,
			daily:       &daily,
			odd:         &odd,
			query:       queryTest,
		},
		{
			name:        "and weekly repository returns an error, then it returns same error",
			weeklyErr:   errTest,
			expectedErr: errTest,
			daily:       &daily,
			odd:         &odd,
			even:        &even,
			query:       queryTest,
		},
		{
			name:        "and temperature repository returns an error, then it returns same error",
			tempErr:     errTest,
			expectedErr: errTest,
			daily:       &daily,
			odd:         &odd,
			even:        &even,
			weekly:      &weekly,
			query:       queryTest,
		},
		{
			name:   "then it returns a valid result",
			daily:  &daily,
			odd:    &odd,
			even:   &even,
			weekly: &weekly,
			temp:   temp,
			expectedResult: app.ProgramsInTime{
				Daily:       &daily,
				Odd:         &odd,
				Even:        &even,
				Weekly:      &weekly,
				Temperature: &temp,
			},
			query: queryTest,
		},
	}
	for _, tt := range tests {
		t.Run(`Given a FindProgramsInTime query handler,
		when Handle method is called `+tt.name, func(t *testing.T) {
			t.Parallel()
			daily := &ProgramRepositoryMock{
				FindByHourFunc: func(_ context.Context, _ *program.Hour) (*program.Program, error) {
					return tt.daily, tt.dailyErr
				},
			}
			odd := &ProgramRepositoryMock{
				FindByHourFunc: func(_ context.Context, _ *program.Hour) (*program.Program, error) {
					return tt.odd, tt.oddErr
				},
			}
			even := &ProgramRepositoryMock{
				FindByHourFunc: func(_ context.Context, _ *program.Hour) (*program.Program, error) {
					return tt.even, tt.evenErr
				},
			}
			weekly := &WeeklyProgramRepositoryMock{
				FindByDayAndHourFunc: func(_ context.Context, _ *program.WeekDay, _ *program.Hour) (*program.Weekly, error) {
					return tt.weekly, tt.weeklyErr
				},
			}
			temperature := &TemperatureProgramRepositoryMock{
				FindByTemperatureAndHourFunc: func(_ context.Context, _ float32, _ program.Hour) (program.Temperature, error) {
					return tt.temp, tt.tempErr
				},
			}
			handler := app.NewFindProgramsInTime(daily, odd, even, weekly, temperature, tracer())
			result, err := handler.Handle(context.Background(), tt.query)
			if err != nil {
				require.IsType(t, tt.expectedErr, err)
				return
			}
			require.Equal(t, tt.expectedErr, err)
			require.Equal(t, tt.expectedResult, result)
		})
	}
}
