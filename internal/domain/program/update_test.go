package program_test

import (
	"testing"

	"github.com/bruli/raspberryWaterSystem/internal/domain/program"
	"github.com/bruli/raspberryWaterSystem/internal/fixtures"
	"github.com/stretchr/testify/require"
)

func TestProgramUpdate(t *testing.T) {
	executions := []program.Execution{fixtures.ExecutionBuilder{Zones: []string{"2"}}.Build()}
	tests := []struct {
		name        string
		executions  []program.Execution
		expectedErr error
	}{
		{
			name:        "with empty executions, then it returns an empty programs error and keeps the program unchanged",
			expectedErr: program.ErrEmptyPrograms,
		},
		{
			name:       "with valid values, then it changes fertilizer, executions and status keeping the hour",
			executions: executions,
		},
	}
	for _, tt := range tests {
		t.Run(`Given a Program, when Update method is called `+tt.name, func(t *testing.T) {
			t.Parallel()
			prg := fixtures.ProgramBuilder{}.Build()
			original := fixtures.ProgramBuilder{}.Build()

			err := prg.Update(true, tt.executions, program.DisabledStatus)

			if tt.expectedErr != nil {
				require.ErrorIs(t, err, tt.expectedErr)
				require.Equal(t, original, prg)
				return
			}
			require.NoError(t, err)
			require.Equal(t, original.Hour(), prg.Hour())
			require.True(t, prg.IsFertilizer())
			require.Equal(t, tt.executions, prg.Executions())
			require.Equal(t, program.DisabledStatus, prg.Status())
		})
	}
}

func TestWeeklyUpdate(t *testing.T) {
	programs := []program.Program{fixtures.ProgramBuilder{Fertilizer: true}.Build()}
	tests := []struct {
		name        string
		programs    []program.Program
		expectedErr error
	}{
		{
			name:        "with empty programs, then it returns an empty programs error and keeps the weekly unchanged",
			expectedErr: program.ErrEmptyPrograms,
		},
		{
			name:     "with programs, then it replaces the programs",
			programs: programs,
		},
	}
	for _, tt := range tests {
		t.Run(`Given a Weekly, when Update method is called `+tt.name, func(t *testing.T) {
			t.Parallel()
			weekly := fixtures.WeeklyBuilder{}.Build()
			original := fixtures.WeeklyBuilder{}.Build()

			err := weekly.Update(tt.programs)

			if tt.expectedErr != nil {
				require.ErrorIs(t, err, tt.expectedErr)
				require.Equal(t, original, weekly)
				return
			}
			require.NoError(t, err)
			require.Equal(t, original.WeekDay(), weekly.WeekDay())
			require.Equal(t, tt.programs, weekly.Programs())
		})
	}
}

func TestTemperatureUpdate(t *testing.T) {
	programs := []program.Program{fixtures.ProgramBuilder{Fertilizer: true}.Build()}
	tests := []struct {
		name        string
		programs    []program.Program
		expectedErr error
	}{
		{
			name:        "with empty programs, then it returns an empty programs error and keeps the temperature unchanged",
			expectedErr: program.ErrEmptyPrograms,
		},
		{
			name:     "with programs, then it replaces the programs",
			programs: programs,
		},
	}
	for _, tt := range tests {
		t.Run(`Given a Temperature, when Update method is called `+tt.name, func(t *testing.T) {
			t.Parallel()
			temp := fixtures.TemperatureBuilder{}.Build()
			original := fixtures.TemperatureBuilder{}.Build()

			err := temp.Update(tt.programs)

			if tt.expectedErr != nil {
				require.ErrorIs(t, err, tt.expectedErr)
				require.Equal(t, original, temp)
				return
			}
			require.NoError(t, err)
			require.Equal(t, original.Temperature(), temp.Temperature())
			require.Equal(t, tt.programs, temp.Programs())
		})
	}
}
