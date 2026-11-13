package program_test

import (
	"testing"
	"time"

	"github.com/bruli/raspberryWaterSystem/internal/domain/program"
	"github.com/bruli/raspberryWaterSystem/internal/fixtures"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	hour, err := program.ParseHour(program.HourLayout)
	require.NoError(t, err)
	executions := []program.Execution{fixtures.ExecutionBuilder{}.Build()}
	tests := []struct {
		name               string
		build              func([]program.Execution) (*program.Program, error)
		executions         []program.Execution
		expectedErr        error
		expectedFertilizer bool
	}{
		{
			name: "New function is called with empty executions, then it returns an empty programs error",
			build: func(execs []program.Execution) (*program.Program, error) {
				return program.New(hour, true, execs)
			},
			expectedErr: program.ErrEmptyPrograms,
		},
		{
			name: "New function is called with fertilizer, then it returns a program with fertilizer",
			build: func(execs []program.Execution) (*program.Program, error) {
				return program.New(hour, true, execs)
			},
			executions:         executions,
			expectedFertilizer: true,
		},
		{
			name: "NewWithoutFertilizer function is called with empty executions, then it returns an empty programs error",
			build: func(execs []program.Execution) (*program.Program, error) {
				return program.NewWithoutFertilizer(hour, execs)
			},
			expectedErr: program.ErrEmptyPrograms,
		},
		{
			name: "NewWithoutFertilizer function is called with all values, then it returns a program without fertilizer",
			build: func(execs []program.Execution) (*program.Program, error) {
				return program.NewWithoutFertilizer(hour, execs)
			},
			executions: executions,
		},
	}
	for _, tt := range tests {
		t.Run(`Given a Program struct, when `+tt.name, func(t *testing.T) {
			t.Parallel()
			prg, err := tt.build(tt.executions)
			if tt.expectedErr != nil {
				require.ErrorIs(t, err, tt.expectedErr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, hour, prg.Hour())
			require.Equal(t, tt.executions, prg.Executions())
			require.Equal(t, tt.expectedFertilizer, prg.IsFertilizer())
		})
	}
}

func TestNewWeekly(t *testing.T) {
	tests := []struct {
		name        string
		programs    []program.Program
		expectedErr error
	}{
		{
			name:        "with empty programs, then it returns an empty weekly programs error",
			expectedErr: program.ErrEmptyPrograms,
		},
		{
			name: "with programs, then it returns an valid weekly",
			programs: []program.Program{
				fixtures.ProgramBuilder{}.Build(),
			},
		},
	}
	for _, tt := range tests {
		t.Run(`Given a Weekly struct,
		when NewWeekly function is called, `+tt.name, func(t *testing.T) {
			t.Parallel()
			week, err := program.NewWeekly(program.WeekDay(time.Friday), tt.programs)
			if err != nil {
				require.IsType(t, tt.expectedErr, err)
				return
			}
			require.Equal(t, program.WeekDay(time.Friday), week.WeekDay())
			require.Equal(t, tt.programs, week.Programs())
		})
	}
}

func TestNewTemperature(t *testing.T) {
	programs := []program.Program{
		fixtures.ProgramBuilder{}.Build(),
	}
	tests := []struct {
		name        string
		temp        float32
		programs    []program.Program
		expectedErr error
	}{
		{
			name:        "with any programs, then it returns an empty programs error",
			expectedErr: program.ErrEmptyPrograms,
		},
		{
			name:        "with under zero temp, then it returns an invalid temperature error",
			temp:        float32(-5),
			programs:    programs,
			expectedErr: program.ErrInvalidTemperature,
		},
		{
			name:        "with over 50 temp, then it returns an invalid temperature error",
			temp:        float32(55),
			programs:    programs,
			expectedErr: program.ErrInvalidTemperature,
		},
		{
			name:     "with programs, then it returns a valid struct",
			programs: programs,
		},
	}
	for _, tt := range tests {
		t.Run(`Given a Temperature struct,
		when NewTemperature method is called `+tt.name, func(t *testing.T) {
			t.Parallel()
			temp, err := program.NewTemperature(tt.temp, tt.programs)
			if err != nil {
				require.ErrorIs(t, err, tt.expectedErr)
				return
			}
			require.Equal(t, tt.expectedErr, err)
			require.Equal(t, tt.temp, temp.Temperature())
			require.Equal(t, tt.programs, temp.Programs())
		})
	}
}
