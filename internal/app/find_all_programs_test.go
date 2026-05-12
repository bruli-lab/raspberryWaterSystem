package app_test

import (
	"context"
	"errors"
	"testing"

	"github.com/bruli-lab/go-core/cqs"
	"github.com/bruli/raspberryWaterSystem/internal/domain/program"
	"github.com/bruli/raspberryWaterSystem/internal/fixtures"

	"github.com/stretchr/testify/require"

	"github.com/bruli/raspberryWaterSystem/internal/app"
)

func TestFindAllProgramsHandle(t *testing.T) {
	errTest := errors.New("")
	dailies := []program.FertilizerProgram{
		fixtures.FertilizerProgramBuilder{}.Build(),
	}
	odds := []program.FertilizerProgram{
		fixtures.FertilizerProgramBuilder{}.Build(),
	}
	evens := []program.FertilizerProgram{
		fixtures.FertilizerProgramBuilder{}.Build(),
	}
	weeklies := []program.Weekly{
		fixtures.WeeklyBuilder{}.Build(),
	}
	temps := []program.Temperature{
		fixtures.TemperatureBuilder{}.Build(),
	}
	tests := []struct {
		name  string
		query cqs.Query
		expectedErr, dailyErr,
		oddErr, evenErr,
		weeklyErr, tempErr error
		expectedResult     any
		dailies, odd, even []program.FertilizerProgram
		weeklies           []program.Weekly
		temps              []program.Temperature
	}{
		{
			name:        "and daily repository returns an error, then it returns same error",
			dailyErr:    errTest,
			expectedErr: errTest,
		},
		{
			name:        "and odd repository returns an error, then it returns same error",
			oddErr:      errTest,
			expectedErr: errTest,
			dailies: []program.FertilizerProgram{
				fixtures.FertilizerProgramBuilder{}.Build(),
			},
		},
		{
			name:        "and even repository returns an error, then it returns same error",
			evenErr:     errTest,
			expectedErr: errTest,
			dailies:     dailies,
			odd:         odds,
		},
		{
			name:        "and weekly repository returns an error, then it returns same error",
			weeklyErr:   errTest,
			expectedErr: errTest,
			dailies:     dailies,
			odd:         odds,
			even:        evens,
		},
		{
			name:        "and temperature repository returns an error, then it returns same error",
			tempErr:     errTest,
			expectedErr: errTest,
			dailies:     dailies,
			odd:         odds,
			even:        evens,
			weeklies:    weeklies,
		},
		{
			name:     "then it returns a valid result",
			dailies:  dailies,
			odd:      odds,
			even:     evens,
			weeklies: weeklies,
			temps:    temps,
			expectedResult: app.AllPrograms{
				Daily:       dailies,
				Odd:         odds,
				Even:        evens,
				Weekly:      weeklies,
				Temperature: temps,
			},
		},
	}
	for _, tt := range tests {
		t.Run(`Given a FindAllPrograms query handler,
		when Handle method is called `+tt.name, func(t *testing.T) {
			t.Parallel()
			temperature := &TemperatureProgramRepositoryMock{
				FindAllFunc: func(_ context.Context) ([]program.Temperature, error) {
					return tt.temps, tt.tempErr
				},
			}
			daily := &ProgramRepositoryMock{
				FindAllFunc: func(_ context.Context) ([]program.FertilizerProgram, error) {
					return tt.dailies, tt.dailyErr
				},
			}
			odd := &ProgramRepositoryMock{
				FindAllFunc: func(_ context.Context) ([]program.FertilizerProgram, error) {
					return tt.odd, tt.oddErr
				},
			}
			even := &ProgramRepositoryMock{
				FindAllFunc: func(_ context.Context) ([]program.FertilizerProgram, error) {
					return tt.even, tt.evenErr
				},
			}
			weekly := &WeeklyProgramRepositoryMock{
				FindAllFunc: func(_ context.Context) ([]program.Weekly, error) {
					return tt.weeklies, tt.weeklyErr
				},
			}
			handler := app.NewFindAllPrograms(daily, odd, even, weekly, temperature, tracer())
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
